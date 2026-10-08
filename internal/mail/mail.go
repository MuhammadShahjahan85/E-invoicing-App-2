// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package mail sends notification e-mails through the client's own SMTP
// server (Office 365, Gmail, an office mail server, ...).
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// Security is how the connection to the SMTP server is protected.
const (
	SecuritySTARTTLS = "starttls" // plain connection upgraded with STARTTLS (port 587)
	SecurityTLS      = "tls"      // TLS from the start (port 465)
	SecurityNone     = "none"     // no encryption (internal relays only; no sign-in)
)

// Config describes the SMTP server.
type Config struct {
	Host     string
	Port     int
	Security string
	Username string
	Password string
	From     string
	Timeout  time.Duration
}

// Message is one e-mail with a plain-text and an HTML body.
type Message struct {
	To      []string
	Subject string
	Text    string
	HTML    string
}

// Validate checks the configuration without connecting.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" || strings.ContainsAny(c.Host, " \r\n/") {
		return errors.New("enter the SMTP server name, e.g. smtp.office365.com")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("enter the SMTP port, e.g. 587")
	}
	switch c.Security {
	case SecuritySTARTTLS, SecurityTLS, SecurityNone:
	default:
		return errors.New("choose the connection security: STARTTLS, TLS or none")
	}
	if _, err := mail.ParseAddress(c.From); err != nil {
		return fmt.Errorf("the sender address is not valid: %w", err)
	}
	return nil
}

// ParseRecipients splits a comma, semicolon or newline separated list.
func ParseRecipients(s string) ([]string, error) {
	var out []string
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '\r' }) {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		a, err := mail.ParseAddress(f)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid e-mail address", f)
		}
		out = append(out, a.Address)
	}
	return out, nil
}

// Send delivers a message.
func Send(ctx context.Context, c Config, m Message) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if len(m.To) == 0 {
		return errors.New("no recipients")
	}
	if strings.ContainsAny(m.Subject, "\r\n") {
		return errors.New("invalid subject")
	}
	from, _ := mail.ParseAddress(c.From)
	var to []string
	for _, r := range m.To {
		a, err := mail.ParseAddress(r)
		if err != nil {
			return fmt.Errorf("invalid recipient %q", r)
		}
		to = append(to, a.Address)
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	dialer := &net.Dialer{Timeout: timeout}
	tlsCfg := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	var conn net.Conn
	var err error
	if c.Security == SecurityTLS {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsCfg}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	cl, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SMTP greeting from %s: %w", addr, err)
	}
	defer cl.Close()
	host, _ := os.Hostname()
	if host == "" || strings.ContainsAny(host, " \r\n") {
		host = "localhost"
	}
	if err := cl.Hello(host); err != nil {
		return fmt.Errorf("SMTP HELO: %w", err)
	}
	if c.Security == SecuritySTARTTLS {
		if ok, _ := cl.Extension("STARTTLS"); !ok {
			return errors.New("the server does not offer STARTTLS; choose TLS (port 465) or check the port")
		}
		if err := cl.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("STARTTLS: %w", err)
		}
	}
	if c.Username != "" {
		ok, mechs := cl.Extension("AUTH")
		if !ok {
			return errors.New("the server does not accept sign-in (AUTH); clear the user name or use the submission port")
		}
		var auth smtp.Auth
		switch {
		case hasMech(mechs, "PLAIN"):
			auth = smtp.PlainAuth("", c.Username, c.Password, c.Host)
		case hasMech(mechs, "LOGIN"):
			auth = &loginAuth{user: c.Username, pass: c.Password, host: c.Host}
		default:
			return fmt.Errorf("no supported sign-in method (server offers %s)", mechs)
		}
		if err := cl.Auth(auth); err != nil {
			return fmt.Errorf("sign-in failed: %w", err)
		}
	}
	if err := cl.Mail(from.Address); err != nil {
		return fmt.Errorf("sender refused: %w", err)
	}
	for _, r := range to {
		if err := cl.Rcpt(r); err != nil {
			return fmt.Errorf("recipient %s refused: %w", r, err)
		}
	}
	w, err := cl.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write(build(from, to, m)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("message not accepted: %w", err)
	}
	return cl.Quit()
}

func hasMech(list, want string) bool {
	for _, m := range strings.Fields(strings.ToUpper(list)) {
		if m == want {
			return true
		}
	}
	return false
}

// loginAuth implements the LOGIN mechanism offered by Office 365 and
// Exchange. Like smtp.PlainAuth it refuses to send the password over an
// unencrypted connection to another host.
type loginAuth struct{ user, pass, host string }

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && !isLocal(server.Name) {
		return "", nil, errors.New("unencrypted connection")
	}
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(fromServer))) {
	case "username:", "user name", "username":
		return []byte(a.user), nil
	case "password:", "password":
		return []byte(a.pass), nil
	}
	return nil, fmt.Errorf("unexpected server challenge %q", fromServer)
}

func isLocal(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

// build assembles a multipart/alternative message.
func build(from *mail.Address, to []string, m Message) []byte {
	var b bytes.Buffer
	boundary := randomHex(12)
	h := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	h("From", from.String())
	h("To", strings.Join(to, ", "))
	h("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	h("Date", time.Now().Format(time.RFC1123Z))
	domain := "localhost"
	if i := strings.LastIndex(from.Address, "@"); i >= 0 {
		domain = from.Address[i+1:]
	}
	h("Message-ID", "<"+randomHex(16)+"@"+domain+">")
	h("MIME-Version", "1.0")
	h("Auto-Submitted", "auto-generated")
	h("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
	b.WriteString("\r\n")
	part := func(ctype, body string) {
		fmt.Fprintf(&b, "--%s\r\nContent-Type: %s; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", boundary, ctype)
		// The text-mode quoted-printable writer turns line ends into CRLF.
		qp := quotedprintable.NewWriter(&b)
		_, _ = qp.Write([]byte(strings.ReplaceAll(body, "\r\n", "\n")))
		_ = qp.Close()
		b.WriteString("\r\n")
	}
	part("text/plain", m.Text)
	if m.HTML != "" {
		part("text/html", m.HTML)
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes()
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
