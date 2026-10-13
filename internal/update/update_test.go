// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package update

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"einvoicing/internal/brand"
)

func testKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func testManifest(content []byte, url string) Manifest {
	sum := sha256.Sum256(content)
	return Manifest{Product: brand.ShortName, Version: "9.1.0", Released: "2026-10-20", Notes: "New: one-click updates <&>",
		Files: map[string]File{Platform(): {URL: url, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(content))}}}
}

func TestSignAndParse(t *testing.T) {
	pub, priv := testKey(t)
	m := testManifest([]byte("installer"), "https://updates.example.pk/v9.1.0/setup.exe")
	b, err := Sign(m, priv)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(b, pub)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "9.1.0" || got.Notes != m.Notes || got.Files[Platform()] != m.Files[Platform()] {
		t.Fatalf("round trip: %+v", got)
	}

	// Re-indenting the file (an editor, a web server) keeps it valid.
	var pretty, compact bytes.Buffer
	_ = json.Indent(&pretty, b, "", "\t")
	_ = json.Compact(&compact, b)
	for _, v := range [][]byte{pretty.Bytes(), compact.Bytes()} {
		if _, err := Parse(v, pub); err != nil {
			t.Fatalf("reformatted file rejected: %v", err)
		}
	}

	// Any change, another key or no key is refused.
	tampered := bytes.Replace(b, []byte(`9.1.0`), []byte(`9.9.0`), 1)
	if _, err := Parse(tampered, pub); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("tampered version: %v", err)
	}
	tampered = bytes.Replace(b, []byte(m.Files[Platform()].SHA256[:8]), []byte("00000000"), 1)
	if _, err := Parse(tampered, pub); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("tampered hash: %v", err)
	}
	other, _ := testKey(t)
	if _, err := Parse(b, other); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("other key: %v", err)
	}
	if _, err := Parse(b, nil); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("no key: %v", err)
	}
	if _, err := Parse([]byte(`<html>not found</html>`), pub); err == nil {
		t.Fatal("HTML page accepted")
	}

	// A licence-style signature over the bare manifest is not an update signature.
	mb, _ := json.Marshal(m)
	bare, _ := json.Marshal(signed{Manifest: mb, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, mb))})
	if _, err := Parse(bare, pub); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("signature without the update prefix: %v", err)
	}

	// Fields added by later versions of the signing tool stay verifiable.
	future := []byte(`{"product":"` + brand.ShortName + `","version":"9.2.0","notes":"x","channel":"stable","files":{"` + Platform() +
		`":{"url":"https://u.example.pk/a","sha256":"` + strings.Repeat("ab", 32) + `","size":10,"arch":"x"}}}`)
	ff, _ := json.Marshal(signed{Manifest: future, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, append([]byte(signedPrefix), future...)))})
	if got, err := Parse(ff, pub); err != nil || got.Version != "9.2.0" {
		t.Fatalf("manifest with new fields: %v", err)
	}
}

func TestValidate(t *testing.T) {
	_, priv := testKey(t)
	good := testManifest([]byte("x"), "https://u.example.pk/setup.exe")
	cases := map[string]func(m *Manifest){
		"plain http": func(m *Manifest) {
			f := m.Files[Platform()]
			f.URL = "http://u.example.pk/setup.exe"
			m.Files[Platform()] = f
		},
		"bad sha":       func(m *Manifest) { f := m.Files[Platform()]; f.SHA256 = "abc"; m.Files[Platform()] = f },
		"no size":       func(m *Manifest) { f := m.Files[Platform()]; f.Size = 0; m.Files[Platform()] = f },
		"other product": func(m *Manifest) { m.Product = "something-else" },
		"bad version":   func(m *Manifest) { m.Version = "1.2.0; rm -rf" },
		"no files":      func(m *Manifest) { m.Files = nil },
		"bad date":      func(m *Manifest) { m.Released = "20/10/2026" },
	}
	for name, mod := range cases {
		m := good
		m.Files = map[string]File{}
		for k, v := range good.Files {
			m.Files[k] = v
		}
		mod(&m)
		if _, err := Sign(m, priv); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"1.2.0", "1.1.9", 1},
		{"1.10.0", "1.9.0", 1},
		{"1.2.0", "1.2.0-rc1", 1},
		{"1.2.0-rc2", "1.2.0-rc1", 1},
		{"1.2.0-rc.10", "1.2.0-rc.9", 1},
		{"1.2.0-demo", "1.2.0-demo", 0},
		{"v1.2.0", "1.2.0", 0},
		{"1.2", "1.2.0", 0},
		{"1.0.0-dev", "1.0.0", -1},
		{"1.1.0", "1.2.0-demo", -1},
		{"2.0.0", "1.99.99", 1},
	} {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestFileName(t *testing.T) {
	for url, want := range map[string]string{
		"https://github.com/o/r/releases/download/v1.2.0/VeridianEInvoicingPakistan-Setup-1.2.0.exe": "VeridianEInvoicingPakistan-Setup-1.2.0.exe",
		"https://u.example.pk/a/..%2F..%2Fevil.exe":                                                  "evil.exe",
		"https://u.example.pk/":        "update",
		"https://u.example.pk/.hidden": "update.hidden",
	} {
		got := File{URL: url}.FileName()
		if got != want {
			t.Errorf("FileName(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestFetchAndDownload(t *testing.T) {
	pub, priv := testKey(t)
	content := bytes.Repeat([]byte("installer bytes "), 5000)
	var feed []byte
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/update.json":
			w.Write(feed)
		case "/setup.exe":
			w.Write(content)
		case "/short.exe":
			w.Write(content[:100])
		case "/redirect":
			http.Redirect(w, r, "http://example.pk/update.json", http.StatusFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	m := testManifest(content, srv.URL+"/setup.exe")
	var err error
	if feed, err = Sign(m, priv); err != nil {
		t.Fatal(err)
	}
	c := NewClient()
	c.HTTP.Transport = srv.Client().Transport
	ctx := context.Background()

	got, raw, err := c.FetchManifest(ctx, srv.URL+"/latest/update.json", pub)
	if err != nil || got.Version != "9.1.0" || !bytes.Equal(raw, feed) {
		t.Fatalf("fetch: %v", err)
	}
	if _, _, err := c.FetchManifest(ctx, srv.URL+"/missing.json", pub); !errors.Is(err, ErrNoRelease) {
		t.Fatalf("missing feed: %v", err)
	}
	if _, _, err := c.FetchManifest(ctx, strings.Replace(srv.URL, "https", "http", 1)+"/latest/update.json", pub); err == nil {
		t.Fatal("plain http address accepted")
	}
	if _, _, err := c.FetchManifest(ctx, srv.URL+"/redirect", pub); err == nil || !strings.Contains(err.Error(), "insecure") {
		t.Fatalf("redirect to http: %v", err)
	}

	dir := t.TempDir()
	dest := filepath.Join(dir, "setup.exe")
	var seen int64
	f := m.Files[Platform()]
	if err := c.Download(ctx, f, dest, func(n int64) { seen = n }); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFile(dest, f); err != nil {
		t.Fatal(err)
	}
	if seen <= 0 || seen > f.Size {
		t.Fatalf("progress: %d", seen)
	}

	// A file that does not match the signed size or SHA-256 is never kept.
	bad := f
	bad.URL = srv.URL + "/short.exe"
	if err := c.Download(ctx, bad, filepath.Join(dir, "short.exe"), nil); err == nil {
		t.Fatal("short file accepted")
	}
	bad = f
	bad.SHA256 = strings.Repeat("0", 64)
	if err := c.Download(ctx, bad, filepath.Join(dir, "altered.exe"), nil); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("altered file: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("files left behind: %v", entries)
	}

	// A file changed after download is detected.
	if err := os.WriteFile(dest, append(content[:len(content)-1], 'X'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFile(dest, f); err == nil {
		t.Fatal("changed file accepted")
	}
}
