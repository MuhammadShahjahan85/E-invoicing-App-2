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
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"einvoicing/internal/license"
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
	fmt.Println("usage: licensegen keygen|issue|verify [flags]  (see source header for details)")
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", "vendor-private.key", "private key output file")
	_ = fs.Parse(args)
	if _, err := os.Stat(*out); err == nil {
		return fmt.Errorf("%s exists; refusing to overwrite a vendor key", *out)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		return err
	}
	fmt.Println("Private key written to", *out, "(keep it secret and backed up)")
	fmt.Println("Public key (embed in builds):")
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
