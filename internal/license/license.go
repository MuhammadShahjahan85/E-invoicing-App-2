// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

// Package license implements offline, signature-verified product licences.
//
// The vendor generates an Ed25519 key pair once (`licensegen keygen`), keeps
// the private key secret, and builds the product with the public key:
//
//	go build -ldflags "-X einvoicing/internal/license.PublicKeyB64=<base64>" ./cmd/einvoice
//
// A licence file (JSON) lists the licensee, the seller NTN/CNICs it covers,
// limits and validity, and carries the vendor's signature. Builds without a
// public key run in developer mode with no restrictions.
package license

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// PublicKeyB64 is set at build time by the vendor.
var PublicKeyB64 = ""

// TrialDays is the evaluation period (sandbox and simulator only).
const TrialDays = 30

// GraceDays is how long production keeps working after a licence expires.
const GraceDays = 15

// License describes what a customer bought.
type License struct {
	LicenseID    string   `json:"licenseId"`
	Licensee     string   `json:"licensee"`
	SellerNTNs   []string `json:"sellerNtns"` // "*" allows any seller
	MaxCompanies int      `json:"maxCompanies"`
	MaxUsers     int      `json:"maxUsers"`
	Edition      string   `json:"edition"`
	IssuedAt     string   `json:"issuedAt"`
	ExpiresAt    string   `json:"expiresAt"` // YYYY-MM-DD; empty = perpetual
	SupportUntil string   `json:"supportUntil"`
	Features     []string `json:"features"`
}

// Signed is the licence file format.
type Signed struct {
	License   License `json:"license"`
	Signature string  `json:"signature"`
}

// Canonical returns the bytes that are signed.
func Canonical(l License) []byte {
	b, _ := json.Marshal(l)
	return b
}

// Sign signs a licence with the vendor's private key.
func Sign(l License, priv ed25519.PrivateKey) Signed {
	sig := ed25519.Sign(priv, Canonical(l))
	return Signed{License: l, Signature: base64.StdEncoding.EncodeToString(sig)}
}

// Verify checks a signed licence.
func Verify(s Signed, pub ed25519.PublicKey) error {
	sig, err := base64.StdEncoding.DecodeString(s.Signature)
	if err != nil {
		return errors.New("licence signature is not valid base64")
	}
	if !ed25519.Verify(pub, Canonical(s.License), sig) {
		return errors.New("licence signature does not match (file altered or issued by another vendor)")
	}
	return nil
}

// SettingsStore persists the licence and trial start.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}

// Status is shown in the UI.
type Status struct {
	Mode         string   `json:"mode"` // developer | trial | trial_expired | licensed | grace | expired | invalid
	Licensee     string   `json:"licensee"`
	LicenseID    string   `json:"licenseId"`
	Edition      string   `json:"edition"`
	SellerNTNs   []string `json:"sellerNtns"`
	MaxCompanies int      `json:"maxCompanies"`
	MaxUsers     int      `json:"maxUsers"`
	ExpiresAt    string   `json:"expiresAt"`
	SupportUntil string   `json:"supportUntil"`
	DaysLeft     int      `json:"daysLeft"`
	Production   bool     `json:"production"`
	Message      string   `json:"message"`
}

// Manager evaluates the installed licence.
type Manager struct {
	pub   ed25519.PublicKey
	store SettingsStore
	now   func() time.Time
	mu    sync.Mutex
}

// NewManager builds a manager using the compiled-in public key.
func NewManager(st SettingsStore) (*Manager, error) {
	m := &Manager{store: st, now: time.Now}
	if PublicKeyB64 != "" {
		raw, err := base64.StdEncoding.DecodeString(PublicKeyB64)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, errors.New("invalid embedded licence public key")
		}
		m.pub = ed25519.PublicKey(raw)
	}
	return m, nil
}

// NewManagerWithKey is used by tests.
func NewManagerWithKey(st SettingsStore, pub ed25519.PublicKey, now func() time.Time) *Manager {
	return &Manager{store: st, pub: pub, now: now}
}

// Install verifies and stores a licence file.
func (m *Manager) Install(ctx context.Context, data []byte) (*Status, error) {
	if m.pub == nil {
		return nil, errors.New("this is a developer build without licence enforcement")
	}
	var s Signed
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("not a licence file: %w", err)
	}
	if err := Verify(s, m.pub); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.store.SetSetting(ctx, "license", string(data)); err != nil {
		return nil, err
	}
	st := m.status(ctx)
	return &st, nil
}

// Status returns the current licence status.
func (m *Manager) Status(ctx context.Context) Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status(ctx)
}

func (m *Manager) status(ctx context.Context) Status {
	if m.pub == nil {
		return Status{Mode: "developer", Production: true, Message: "Developer build — licence enforcement disabled."}
	}
	now := m.now()
	raw, _ := m.store.GetSetting(ctx, "license")
	if raw == "" {
		start, _ := m.store.GetSetting(ctx, "trial_started_at")
		if start == "" {
			start = now.UTC().Format(time.RFC3339)
			_ = m.store.SetSetting(ctx, "trial_started_at", start)
		}
		t, _ := time.Parse(time.RFC3339, start)
		left := TrialDays - int(now.Sub(t).Hours()/24)
		if left < 0 {
			return Status{Mode: "trial_expired", Message: "The evaluation period has ended. Install a licence to continue."}
		}
		return Status{Mode: "trial", DaysLeft: left, Message: fmt.Sprintf("Evaluation: %d days left. FBR sandbox and the training simulator are available; production reporting needs a licence.", left)}
	}
	var s Signed
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return Status{Mode: "invalid", Message: "The installed licence cannot be read."}
	}
	if err := Verify(s, m.pub); err != nil {
		return Status{Mode: "invalid", Message: err.Error()}
	}
	l := s.License
	st := Status{Mode: "licensed", Licensee: l.Licensee, LicenseID: l.LicenseID, Edition: l.Edition, SellerNTNs: l.SellerNTNs,
		MaxCompanies: l.MaxCompanies, MaxUsers: l.MaxUsers, ExpiresAt: l.ExpiresAt, SupportUntil: l.SupportUntil, Production: true}
	if l.ExpiresAt != "" {
		exp, err := time.Parse("2006-01-02", l.ExpiresAt)
		if err != nil {
			return Status{Mode: "invalid", Message: "licence expiry date is malformed"}
		}
		end := exp.Add(24 * time.Hour)
		st.DaysLeft = int(end.Sub(now).Hours() / 24)
		switch {
		case now.After(end.Add(GraceDays * 24 * time.Hour)):
			st.Mode, st.Production = "expired", false
			st.Message = "The licence expired on " + l.ExpiresAt + ". Production reporting is disabled; renew the licence."
		case now.After(end):
			st.Mode = "grace"
			st.Message = fmt.Sprintf("The licence expired on %s. Production reporting continues for a %d-day grace period; renew now.", l.ExpiresAt, GraceDays)
		default:
			st.Message = "Licensed to " + l.Licensee + " until " + l.ExpiresAt + "."
		}
	} else {
		st.Message = "Perpetual licence for " + l.Licensee + "."
	}
	return st
}

// CheckProduction implements service.LicenseChecker.
func (m *Manager) CheckProduction(sellerNTN string) error {
	st := m.Status(context.Background())
	if !st.Production {
		if st.Message != "" {
			return errors.New(st.Message)
		}
		return errors.New("no valid licence for production use")
	}
	if st.Mode == "developer" {
		return nil
	}
	for _, n := range st.SellerNTNs {
		if n == "*" || strings.TrimSpace(n) == sellerNTN {
			return nil
		}
	}
	return fmt.Errorf("the licence does not cover seller NTN/CNIC %s", sellerNTN)
}

// CheckLimits enforces the number of companies and users.
func (m *Manager) CheckLimits(companies, users int) error {
	st := m.Status(context.Background())
	if st.Mode != "licensed" && st.Mode != "grace" {
		return nil
	}
	if st.MaxCompanies > 0 && companies > st.MaxCompanies {
		return fmt.Errorf("the licence allows %d companies", st.MaxCompanies)
	}
	if st.MaxUsers > 0 && users > st.MaxUsers {
		return fmt.Errorf("the licence allows %d users", st.MaxUsers)
	}
	return nil
}
