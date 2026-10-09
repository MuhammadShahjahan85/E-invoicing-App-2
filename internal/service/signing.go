// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"einvoicing/internal/store"
)

// Rule 150R(4)(b) of the Sales Tax Rules, 2006 requires the electronic
// invoicing system to "create the digital signature and record the digital
// signature on the sales tax invoice". Each installation has an Ed25519 key;
// the private key is kept encrypted with the vault. An accepted invoice is
// signed over its FBR invoice number and its seal (which covers the invoice's
// content and chains it to the previous invoice).

const (
	settingSigningKey = "signing.key" // vault-encrypted private key seed (base64)
	settingSigningPub = "signing.pub" // public key (base64)
	signaturePrefix   = "veridian-di-signature-v1"
)

// signingKey returns the installation's signing key, creating it on first use.
func (s *Service) signingKey(ctx context.Context) (ed25519.PrivateKey, error) {
	s.signMu.Lock()
	defer s.signMu.Unlock()
	if s.signKey != nil {
		return s.signKey, nil
	}
	enc, err := s.Store.GetSetting(ctx, settingSigningKey)
	if err != nil {
		return nil, err
	}
	if enc != "" {
		seedB64, err := s.Vault.Decrypt(enc)
		if err != nil {
			return nil, fmt.Errorf("signing key: %w", err)
		}
		seed, err := base64.StdEncoding.DecodeString(seedB64)
		if err != nil || len(seed) != ed25519.SeedSize {
			return nil, errors.New("signing key is damaged")
		}
		s.signKey = ed25519.NewKeyFromSeed(seed)
		return s.signKey, nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	enc, err = s.Vault.Encrypt(base64.StdEncoding.EncodeToString(priv.Seed()))
	if err != nil {
		return nil, err
	}
	if err := s.Store.SetSetting(ctx, settingSigningKey, enc); err != nil {
		return nil, err
	}
	if err := s.Store.SetSetting(ctx, settingSigningPub, base64.StdEncoding.EncodeToString(pub)); err != nil {
		return nil, err
	}
	s.Audit(ctx, System, 0, "signing.key_created", "settings", settingSigningPub, map[string]any{"fingerprint": keyFingerprint(pub)})
	s.signKey = priv
	return priv, nil
}

func signatureMessage(inv *store.Invoice) []byte {
	return []byte(signaturePrefix + "|" + inv.FBRInvoiceNumber + "|" + inv.SealHash)
}

// signInvoice returns the digital signature of a sealed invoice.
func (s *Service) signInvoice(ctx context.Context, inv *store.Invoice) (string, error) {
	if inv.SealHash == "" || inv.FBRInvoiceNumber == "" {
		return "", errors.New("only invoices accepted by FBR are signed")
	}
	key, err := s.signingKey(ctx)
	if err != nil {
		return "", err
	}
	return signWith(key, inv), nil
}

func signWith(key ed25519.PrivateKey, inv *store.Invoice) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(key, signatureMessage(inv)))
}

// SigningPublicKey returns the installation's public key (creating the key
// pair if needed). It is derived from the private key; if the vault cannot
// open that (master.key lost), the stored public key is used so earlier
// signatures can still be checked.
func (s *Service) SigningPublicKey(ctx context.Context) (ed25519.PublicKey, error) {
	key, err := s.signingKey(ctx)
	if err == nil {
		return key.Public().(ed25519.PublicKey), nil
	}
	if pub, perr := s.storedPublicKey(ctx); perr == nil && pub != nil {
		return pub, nil
	}
	return nil, err
}

func (s *Service) storedPublicKey(ctx context.Context) (ed25519.PublicKey, error) {
	v, err := s.Store.GetSetting(ctx, settingSigningPub)
	if err != nil || v == "" {
		return nil, err
	}
	b, err := base64.StdEncoding.DecodeString(v)
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, errors.New("the stored public signing key is damaged")
	}
	return ed25519.PublicKey(b), nil
}

// signingKeyProblem describes a fault in the signing key pair: a private key
// the vault cannot open, or a stored public key that does not belong to it.
func (s *Service) signingKeyProblem(ctx context.Context) string {
	enc, err := s.Store.GetSetting(ctx, settingSigningKey)
	if err != nil || enc == "" {
		return ""
	}
	key, err := s.signingKey(ctx)
	if err != nil {
		return "the invoice signing key cannot be opened (" + err.Error() + "); restore master.key from your backup"
	}
	if pub, err := s.storedPublicKey(ctx); err != nil || (pub != nil && !pub.Equal(key.Public())) {
		return "the stored public signing key does not match this installation's private key"
	}
	return ""
}

// SignatureValid reports whether an invoice's signature matches its FBR
// number and seal under this installation's key.
func (s *Service) SignatureValid(ctx context.Context, inv *store.Invoice) bool {
	if inv.Signature == "" {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(inv.Signature)
	if err != nil {
		return false
	}
	pub, err := s.SigningPublicKey(ctx)
	if err != nil {
		return false
	}
	return ed25519.Verify(pub, signatureMessage(inv), sig)
}

// SigningKeyInfo describes the public key for auditors.
type SigningKeyInfo struct {
	Algorithm   string `json:"algorithm"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"` // PEM
}

// SigningKey returns the public key and its fingerprint.
func (s *Service) SigningKey(ctx context.Context) (*SigningKeyInfo, error) {
	pub, err := s.SigningPublicKey(ctx)
	if err != nil {
		return nil, err
	}
	// SubjectPublicKeyInfo for Ed25519 (RFC 8410): fixed 12-byte prefix.
	der := append([]byte{0x30, 0x2a, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70, 0x03, 0x21, 0x00}, pub...)
	return &SigningKeyInfo{
		Algorithm:   "Ed25519",
		Fingerprint: keyFingerprint(pub),
		PublicKey:   string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})),
	}, nil
}

func keyFingerprint(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	h := strings.ToUpper(hex.EncodeToString(sum[:8]))
	return h[0:4] + "-" + h[4:8] + "-" + h[8:12] + "-" + h[12:16]
}

// signPending signs invoices accepted before invoices were signed.
func (s *Service) signPending(ctx context.Context) {
	ids, err := s.Store.UnsignedSealedInvoices(ctx, 200)
	if err != nil || len(ids) == 0 {
		return
	}
	n := 0
	for _, id := range ids {
		inv, err := s.Store.GetInvoiceAnyCompany(ctx, id)
		if err != nil {
			continue
		}
		sig, err := s.signInvoice(ctx, inv)
		if err != nil {
			// The daily integrity check reports a key that cannot be opened;
			// warn here at most once an hour.
			if s.Now().Sub(s.lastSignWarn) >= time.Hour {
				s.lastSignWarn = s.Now()
				s.Log.Warn("signing invoice failed", "invoice", id, "err", err)
			}
			return
		}
		if err := s.Store.SetSignature(ctx, id, sig); err == nil {
			n++
		}
	}
	if n > 0 {
		s.Log.Info("digital signatures recorded on earlier invoices", "count", n)
	}
}
