// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Command licensegen is the VENDOR tool for issuing product licences.
// Keep it, and the private key it creates, off customer machines.
//
//	licensegen keygen -out vendor-private.key
//	    Creates an Ed25519 key pair and prints the public key to embed in
//	    builds: go build -ldflags "-X einvoicing/internal/license.PublicKeyB64=<KEY>" ./cmd/einvoice
//
//	licensegen issue -key vendor-private.key -licensee "ABC (Pvt) Ltd" -ntn 1234567[,7654321] \
//	    [-expires 2027-06-30] [-support 2027-06-30] [-companies 1] [-users 5] [-edition standard] -out license.lic
//
//	licensegen verify -pub <base64> -in license.lic
//
// Software updates (see docs/LICENSING-AND-SALES.md, "Publishing updates"):
//
//	licensegen update-keygen -out update-private.key
//	    Creates the key pair that signs updates; builds embed the public key:
//	    -ldflags "-X einvoicing/internal/update.PublicKeyB64=<KEY>"
//
//	licensegen sign-update -key update-private.key -version 1.2.0 -notes-file notes.txt \
//	    -url-base https://github.com/OWNER/REPO/releases/download/v1.2.0/ \
//	    -windows VeridianEInvoicingPakistan-Setup-1.2.0.exe -linux einvoice-1.2.0-linux-amd64.tar.gz \
//	    [-released 2026-10-20] [-mandatory] [-free] -out update.json
//
//	licensegen verify-update -pub <base64> -in update.json [-dir folder-with-the-installers]
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/license"
	"einvoicing/internal/update"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen(os.Args[2:])
	case "issue":
		err = issue(os.Args[2:])
	case "verify":
		err = verify(os.Args[2:])
	case "update-keygen":
		err = updateKeygen(os.Args[2:])
	case "sign-update":
		err = signUpdate(os.Args[2:])
	case "verify-update":
		err = verifyUpdate(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("usage: licensegen keygen|issue|verify|update-keygen|sign-update|verify-update [flags]  (see source header for details)")
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", "vendor-private.key", "private key output file")
	_ = fs.Parse(args)
	return writeKeyPair(*out, "LICENSE_PUBKEY")
}

func updateKeygen(args []string) error {
	fs := flag.NewFlagSet("update-keygen", flag.ExitOnError)
	out := fs.String("out", "update-private.key", "private key output file")
	_ = fs.Parse(args)
	return writeKeyPair(*out, "UPDATE_PUBKEY")
}

func writeKeyPair(out, variable string) error {
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("%s exists; refusing to overwrite a vendor key", out)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		return err
	}
	fmt.Println("Private key written to", out, "(keep it secret and backed up)")
	fmt.Printf("Public key (embed in builds; GitHub variable %s):\n", variable)
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	return nil
}

func loadPriv(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key file")
	}
	return ed25519.PrivateKey(raw), nil
}

func issue(args []string) error {
	fs := flag.NewFlagSet("issue", flag.ExitOnError)
	key := fs.String("key", "vendor-private.key", "vendor private key")
	licensee := fs.String("licensee", "", "customer legal name")
	ntn := fs.String("ntn", "", "comma separated seller NTN/CNICs covered, or * for any")
	expires := fs.String("expires", "", "expiry date YYYY-MM-DD (empty = perpetual)")
	support := fs.String("support", "", "support/updates valid until YYYY-MM-DD")
	companies := fs.Int("companies", 1, "maximum companies (0 = unlimited)")
	users := fs.Int("users", 5, "maximum users (0 = unlimited)")
	edition := fs.String("edition", "standard", "edition name")
	id := fs.String("id", "", "licence id (default: generated)")
	out := fs.String("out", "license.lic", "output file")
	_ = fs.Parse(args)
	if *licensee == "" || *ntn == "" {
		return fmt.Errorf("-licensee and -ntn are required")
	}
	for _, d := range []string{*expires, *support} {
		if d != "" {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				return fmt.Errorf("dates must be YYYY-MM-DD")
			}
		}
	}
	priv, err := loadPriv(*key)
	if err != nil {
		return err
	}
	var ntns []string
	for _, n := range strings.Split(*ntn, ",") {
		if n = strings.TrimSpace(n); n != "" {
			ntns = append(ntns, n)
		}
	}
	lid := *id
	if lid == "" {
		b := make([]byte, 6)
		_, _ = rand.Read(b)
		lid = fmt.Sprintf("LIC-%s-%X", time.Now().Format("20060102"), b)
	}
	l := license.License{LicenseID: lid, Licensee: *licensee, SellerNTNs: ntns, MaxCompanies: *companies, MaxUsers: *users,
		Edition: *edition, IssuedAt: time.Now().Format("2006-01-02"), ExpiresAt: *expires, SupportUntil: *support}
	signed := license.Sign(l, priv)
	b, _ := json.MarshalIndent(signed, "", "  ")
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		return err
	}
	fmt.Println("Licence", lid, "written to", *out)
	return nil
}

func verify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	pub := fs.String("pub", "", "base64 public key")
	in := fs.String("in", "license.lic", "licence file")
	_ = fs.Parse(args)
	raw, err := base64.StdEncoding.DecodeString(*pub)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key")
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	var s license.Signed
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if err := license.Verify(s, ed25519.PublicKey(raw)); err != nil {
		return err
	}
	fmt.Printf("Valid licence %s for %s (NTN %v, expires %q)\n", s.License.LicenseID, s.License.Licensee, s.License.SellerNTNs, s.License.ExpiresAt)
	return nil
}

// signUpdate writes the signed update file for a release.
func signUpdate(args []string) error {
	fs := flag.NewFlagSet("sign-update", flag.ExitOnError)
	key := fs.String("key", "update-private.key", "update private key")
	version := fs.String("version", "", "version of the release, e.g. 1.2.0")
	released := fs.String("released", time.Now().Format("2006-01-02"), "release date YYYY-MM-DD")
	notes := fs.String("notes", "", "what is new (or use -notes-file)")
	notesFile := fs.String("notes-file", "", "file with what is new")
	base := fs.String("url-base", "", "https address the installers are downloaded from (ending in /)")
	win := fs.String("windows", "", "Windows installer (Setup .exe)")
	lin := fs.String("linux", "", "Linux archive (.tar.gz)")
	mandatory := fs.Bool("mandatory", false, "install at the clients' update time even if they install updates by hand")
	free := fs.Bool("free", false, "also for licences whose support period has ended")
	out := fs.String("out", "update.json", "output file")
	_ = fs.Parse(args)
	if *version == "" || *base == "" || (*win == "" && *lin == "") {
		return fmt.Errorf("-version, -url-base and at least one of -windows and -linux are required")
	}
	if !strings.HasSuffix(*base, "/") {
		*base += "/"
	}
	text := *notes
	if *notesFile != "" {
		b, err := os.ReadFile(*notesFile)
		if err != nil {
			return err
		}
		text = strings.TrimSpace(string(b))
	}
	if text == "" {
		text = "Version " + *version
	}
	m := update.Manifest{Product: brand.ShortName, Version: strings.TrimPrefix(*version, "v"), Released: *released, Notes: text,
		Mandatory: *mandatory, Free: *free, Files: map[string]update.File{}}
	for platform, path := range map[string]string{"windows-amd64": *win, "linux-amd64": *lin} {
		if path == "" {
			continue
		}
		f, err := describe(path)
		if err != nil {
			return err
		}
		f.URL = *base + filepath.Base(path)
		m.Files[platform] = f
	}
	priv, err := loadPriv(*key)
	if err != nil {
		return err
	}
	b, err := update.Sign(m, priv)
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		return err
	}
	fmt.Printf("Update %s signed: %s (%d installer(s))\n", m.Version, *out, len(m.Files))
	return nil
}

func describe(path string) (update.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return update.File{}, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return update.File{}, err
	}
	return update.File{SHA256: hex.EncodeToString(h.Sum(nil)), Size: n}, nil
}

// verifyUpdate checks a signed update file and, optionally, the installers.
func verifyUpdate(args []string) error {
	fs := flag.NewFlagSet("verify-update", flag.ExitOnError)
	pub := fs.String("pub", "", "base64 update public key")
	in := fs.String("in", "update.json", "update file")
	dir := fs.String("dir", "", "folder with the installers, to check them too")
	_ = fs.Parse(args)
	key, err := update.ParsePublicKey(*pub)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	m, err := update.Parse(b, key)
	if err != nil {
		return err
	}
	fmt.Printf("Valid update %s (released %s, mandatory %v, free %v)\n%s\n", m.Version, m.Released, m.Mandatory, m.Free, m.Notes)
	for p, f := range m.Files {
		fmt.Printf("  %s: %s (%d bytes)\n", p, f.URL, f.Size)
		if *dir != "" {
			if err := update.VerifyFile(filepath.Join(*dir, f.FileName()), f); err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			fmt.Println("    file checked")
		}
	}
	return nil
}
