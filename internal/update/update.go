// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package update implements vendor-signed software updates.
//
// The vendor publishes, for each release, the installers and an update file
// (update.json) that lists the version, what is new, and the address, size
// and SHA-256 of each installer, signed with the vendor's update key
// (`licensegen update-keygen`, then `licensegen sign-update`). Builds carry
// the public key and the address of the update file:
//
//	go build -ldflags "-X einvoicing/internal/update.PublicKeyB64=<base64> \
//	    -X einvoicing/internal/update.FeedURL=https://…/update.json" ./cmd/einvoice
//
// An installation only ever installs a file whose size and SHA-256 match an
// update file signed with that key, and only a newer version. Builds without
// a key cannot install updates.
package update

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"einvoicing/internal/brand"
)

// PublicKeyB64 is the vendor's update public key, set at build time.
var PublicKeyB64 = ""

// FeedURL is the address of the vendor's signed update file, set at build time.
var FeedURL = ""

const (
	// signedPrefix separates update signatures from every other use of a key.
	signedPrefix = "veridian-einvoicing-pakistan/update/v1\n"
	// MaxManifestSize bounds the update file.
	MaxManifestSize = 256 << 10
	// MaxFileSize bounds an installer.
	MaxFileSize = 512 << 20
)

// File is one installer of a release.
type File struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Manifest describes a release.
type Manifest struct {
	Product  string `json:"product"`
	Version  string `json:"version"`
	Released string `json:"released"` // YYYY-MM-DD
	Notes    string `json:"notes"`
	// Mandatory updates are installed at the installation's update time even
	// when the administrator chose to install updates by hand.
	Mandatory bool `json:"mandatory,omitempty"`
	// Free updates may be installed after the licence's support period.
	Free bool `json:"free,omitempty"`
	// Files maps a platform ("windows-amd64", "linux-amd64") to its installer.
	Files map[string]File `json:"files"`
}

// signed is the update file format. The manifest is signed exactly as
// written (in compact form), so fields added later remain verifiable.
type signed struct {
	Manifest  json.RawMessage `json:"manifest"`
	Signature string          `json:"signature"`
}

var (
	// ErrNotConfigured: the build has no update key or address.
	ErrNotConfigured = errors.New("automatic updates are not set up in this build of the software; ask your vendor for the latest installer")
	// ErrBadSignature: the update file was not signed with the vendor's key.
	ErrBadSignature = errors.New("the update is not signed by " + brand.Vendor + " (signature check failed), so it was not used")
)

// PublicKey returns the compiled-in update key, or nil without one.
func PublicKey() (ed25519.PublicKey, error) {
	if PublicKeyB64 == "" {
		return nil, nil
	}
	return ParsePublicKey(PublicKeyB64)
}

// ParsePublicKey decodes a base64 Ed25519 public key.
func ParsePublicKey(b64 string) (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, errors.New("invalid update public key")
	}
	return ed25519.PublicKey(raw), nil
}

// Sign returns the signed update file for a manifest.
func Sign(m Manifest, priv ed25519.PrivateKey) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	mb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(priv, append([]byte(signedPrefix), mb...))
	out, err := json.MarshalIndent(signed{Manifest: mb, Signature: base64.StdEncoding.EncodeToString(sig)}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Parse verifies a signed update file and returns its manifest.
func Parse(b []byte, pub ed25519.PublicKey) (*Manifest, error) {
	if pub == nil {
		return nil, ErrNotConfigured
	}
	if len(b) > MaxManifestSize {
		return nil, errors.New("the update file is too large")
	}
	var s signed
	if err := json.Unmarshal(b, &s); err != nil || len(s.Manifest) == 0 {
		return nil, errors.New("the update address did not return an update file")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, s.Manifest); err != nil {
		return nil, errors.New("the update file is damaged")
	}
	sig, err := base64.StdEncoding.DecodeString(s.Signature)
	if err != nil || !ed25519.Verify(pub, append([]byte(signedPrefix), compact.Bytes()...), sig) {
		return nil, ErrBadSignature
	}
	var m Manifest
	if err := json.Unmarshal(compact.Bytes(), &m); err != nil {
		return nil, errors.New("the update file is damaged")
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

var (
	reSHA     = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	reVersion = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,3}(-[0-9A-Za-z.]+)?$`)
	reUnsafe  = regexp.MustCompile(`[^A-Za-z0-9._-]`)
)

// Validate checks a manifest's fields.
func (m *Manifest) Validate() error {
	if m.Product != brand.ShortName {
		return fmt.Errorf("the update is for another product (%q)", m.Product)
	}
	if !reVersion.MatchString(m.Version) {
		return fmt.Errorf("the update has an invalid version number %q", m.Version)
	}
	if m.Released != "" {
		if _, err := time.Parse("2006-01-02", m.Released); err != nil {
			return fmt.Errorf("the update has an invalid release date %q", m.Released)
		}
	}
	if len(m.Files) == 0 {
		return errors.New("the update lists no installers")
	}
	for p, f := range m.Files {
		u, err := url.Parse(f.URL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return fmt.Errorf("the %s installer address must start with https://", p)
		}
		if !reSHA.MatchString(f.SHA256) {
			return fmt.Errorf("the %s installer has an invalid SHA-256", p)
		}
		if f.Size <= 0 || f.Size > MaxFileSize {
			return fmt.Errorf("the %s installer has an invalid size", p)
		}
	}
	return nil
}

// FileName is a safe local name for a file, taken from its address.
func (f File) FileName() string {
	name := ""
	if u, err := url.Parse(f.URL); err == nil {
		name = path.Base(u.Path)
	}
	if name == "/" || name == "." {
		name = ""
	}
	name = reUnsafe.ReplaceAllString(name, "_")
	if name == "" || strings.HasPrefix(name, ".") {
		name = "update" + name
	}
	return name
}

// SumMatches reports whether a hex SHA-256 equals the file's.
func (f File) SumMatches(sum []byte) bool {
	want, err := hex.DecodeString(f.SHA256)
	return err == nil && bytes.Equal(want, sum)
}

// Platform names this build's platform, as used in Manifest.Files.
func Platform() string { return runtime.GOOS + "-" + runtime.GOARCH }

// Newer reports whether version a is newer than version b.
func Newer(a, b string) bool { return CompareVersions(a, b) > 0 }

// CompareVersions compares two version numbers like 1.2.0 and 1.10.0-rc1:
// numbers part by part, then a release above its pre-releases.
func CompareVersions(a, b string) int {
	an, ap := splitVersion(a)
	bn, bp := splitVersion(b)
	for i := range an {
		if an[i] != bn[i] {
			if an[i] < bn[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case ap == bp:
		return 0
	case ap == "":
		return 1
	case bp == "":
		return -1
	}
	return comparePre(ap, bp)
}

func splitVersion(v string) ([4]int, string) {
	var nums [4]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	pre := ""
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v, pre = v[:i], v[i+1:]
	}
	for i, p := range strings.SplitN(v, ".", 4) {
		n, _ := strconv.Atoi(p)
		nums[i] = n
	}
	return nums, pre
}

func comparePre(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		an, aerr := strconv.Atoi(as[i])
		bn, berr := strconv.Atoi(bs[i])
		switch {
		case aerr == nil && berr == nil:
			if an != bn {
				if an < bn {
					return -1
				}
				return 1
			}
		case aerr == nil:
			return -1
		case berr == nil:
			return 1
		default:
			if c := strings.Compare(as[i], bs[i]); c != 0 {
				return c
			}
		}
	}
	switch {
	case len(as) < len(bs):
		return -1
	case len(as) > len(bs):
		return 1
	}
	return 0
}
