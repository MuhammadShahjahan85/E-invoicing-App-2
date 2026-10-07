// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

package license

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"
)

type mem map[string]string

func (m mem) GetSetting(_ context.Context, k string) (string, error) { return m[k], nil }
func (m mem) SetSetting(_ context.Context, k, v string) error        { m[k] = v; return nil }

func TestLicenceLifecycle(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	st := mem{}
	m := NewManagerWithKey(st, pub, func() time.Time { return now })
	ctx := context.Background()

	if s := m.Status(ctx); s.Mode != "trial" || s.Production {
		t.Fatalf("trial status %+v", s)
	}
	if err := m.CheckProduction("0786909"); err == nil {
		t.Fatal("trial must not allow production")
	}
	signed := Sign(License{LicenseID: "L-1", Licensee: "ABC", SellerNTNs: []string{"0786909"}, ExpiresAt: "2027-06-30", MaxCompanies: 1}, priv)
	data, _ := json.Marshal(signed)
	if _, err := m.Install(ctx, data); err != nil {
		t.Fatal(err)
	}
	if err := m.CheckProduction("0786909"); err != nil {
		t.Fatal(err)
	}
	if err := m.CheckProduction("1234567"); err == nil {
		t.Fatal("other NTN must be refused")
	}
	if err := m.CheckLimits(2, 1); err == nil {
		t.Fatal("company limit not enforced")
	}
	// Tampering invalidates the signature.
	signed.License.SellerNTNs = []string{"*"}
	bad, _ := json.Marshal(signed)
	if _, err := m.Install(ctx, bad); err == nil {
		t.Fatal("tampered licence accepted")
	}
	// Expiry and grace.
	now = time.Date(2027, 7, 5, 0, 0, 0, 0, time.UTC)
	if s := m.Status(ctx); s.Mode != "grace" || !s.Production {
		t.Fatalf("grace status %+v", s)
	}
	now = time.Date(2027, 8, 1, 0, 0, 0, 0, time.UTC)
	if s := m.Status(ctx); s.Mode != "expired" || s.Production {
		t.Fatalf("expired status %+v", s)
	}
}

func TestDeveloperBuild(t *testing.T) {
	m, err := NewManager(mem{})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.CheckProduction("anything"); err != nil {
		t.Fatal(err)
	}
}
