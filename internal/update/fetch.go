// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"einvoicing/internal/brand"
)

// ErrNoRelease: the update address has no update file (nothing published yet).
var ErrNoRelease = errors.New("no update has been published yet")

// Client downloads update files and installers, over HTTPS only.
type Client struct {
	HTTP      *http.Client
	UserAgent string
}

// NewClient returns a client that uses the system's proxy settings.
func NewClient() *Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = time.Minute
	return &Client{
		HTTP:      &http.Client{Transport: tr, CheckRedirect: httpsOnly},
		UserAgent: brand.ShortName + "/" + brand.Version + " (" + Platform() + ")",
	}
}

func httpsOnly(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("too many redirects")
	}
	if req.URL.Scheme != "https" {
		return errors.New("the update server redirected to an insecure (http) address")
	}
	return nil
}

func (c *Client) get(ctx context.Context, addr string) (*http.Response, error) {
	u, err := url.Parse(addr)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, fmt.Errorf("the update address must start with https:// (got %q)", addr)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Cache-Control", "no-cache")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the update server: %w", err)
	}
	return resp, nil
}

// FetchManifest downloads and verifies the signed update file. It returns
// the manifest and the file as received.
func (c *Client) FetchManifest(ctx context.Context, addr string, pub ed25519.PublicKey) (*Manifest, []byte, error) {
	if pub == nil {
		return nil, nil, ErrNotConfigured
	}
	resp, err := c.get(ctx, addr)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, ErrNoRelease
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("the update server answered %s", resp.Status)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxManifestSize+1))
	if err != nil {
		return nil, nil, fmt.Errorf("could not read the update file: %w", err)
	}
	m, err := Parse(b, pub)
	if err != nil {
		return nil, nil, err
	}
	return m, b, nil
}

// Download saves an installer to dest after checking its size and SHA-256.
// progress, if set, receives the bytes received so far.
func (c *Client) Download(ctx context.Context, f File, dest string, progress func(done int64)) error {
	resp, err := c.get(ctx, f.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the update server answered %s for the installer", resp.Status)
	}
	if resp.ContentLength >= 0 && resp.ContentLength != f.Size {
		return fmt.Errorf("the installer on the server has the wrong size (%d bytes, expected %d)", resp.ContentLength, f.Size)
	}
	tmp := dest + ".part"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	h := sha256.New()
	pw := &progressWriter{fn: progress}
	n, err := io.Copy(io.MultiWriter(out, h, pw), io.LimitReader(resp.Body, f.Size+1))
	if err == nil {
		err = out.Sync()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	switch {
	case err != nil:
		err = fmt.Errorf("the download was interrupted: %w", err)
	case n != f.Size:
		err = fmt.Errorf("the installer has the wrong size (%d bytes, expected %d)", n, f.Size)
	case !f.SumMatches(h.Sum(nil)):
		err = errors.New("the installer is damaged or was altered (SHA-256 does not match the signed update file)")
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

type progressWriter struct {
	fn   func(int64)
	done int64
	last time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.done += int64(len(b))
	if p.fn != nil && time.Since(p.last) > 200*time.Millisecond {
		p.last = time.Now()
		p.fn(p.done)
	}
	return len(b), nil
}

// VerifyFile checks a downloaded installer against its signed size and SHA-256.
func VerifyFile(path string, f File) error {
	r, err := os.Open(path)
	if err != nil {
		return err
	}
	defer r.Close()
	return verifyReader(r, f)
}

func verifyReader(r io.Reader, f File) error {
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(r, f.Size+1))
	if err != nil {
		return err
	}
	if n != f.Size || !f.SumMatches(h.Sum(nil)) {
		return errors.New("the downloaded installer does not match the signed update file")
	}
	return nil
}
