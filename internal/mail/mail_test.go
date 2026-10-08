// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package mail

import (
	"bufio"
	"context"
	"io"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"
	"testing"
	"time"
)

// fakeSMTP accepts one message on a local port and returns its DATA.
func fakeSMTP(t *testing.T, extensions ...string) (host string, port int, got chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	got = make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		w := func(s string) { _, _ = conn.Write([]byte(s + "\r\n")) }
		w("220 test ESMTP")
		var data strings.Builder
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					w("250 queued")
					got <- data.String()
					continue
				}
				data.WriteString(line)
				continue
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"):
				for _, e := range extensions {
					w("250-" + e)
				}
				w("250 8BITMIME")
			case cmd == "DATA":
				inData = true
				w("354 go ahead")
			case cmd == "QUIT":
				w("221 bye")
				return
			default:
				w("250 OK")
			}
		}
	}()
	a := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", a.Port, got
}

func TestSendPlainRelay(t *testing.T) {
	host, port, got := fakeSMTP(t)
	cfg := Config{Host: host, Port: port, Security: SecurityNone, From: "Veridian <alerts@example.pk>", Timeout: 5 * time.Second}
	err := Send(context.Background(), cfg, Message{To: []string{"accounts@example.pk"}, Subject: "3 invoices rejected — سیلز ٹیکس",
		Text: "Line one\nLine two", HTML: "<p>Line one</p>"})
	if err != nil {
		t.Fatal(err)
	}
	msg := <-got
	for _, want := range []string{"From: \"Veridian\" <alerts@example.pk>", "To: accounts@example.pk", "Subject: =?utf-8?q?", "multipart/alternative",
		"Content-Type: text/plain; charset=utf-8", "Content-Type: text/html; charset=utf-8", "Auto-Submitted: auto-generated"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message lacks %q:\n%s", want, msg)
		}
	}
	body, _ := readQP(msg)
	if !strings.Contains(body, "Line one\r\nLine two") {
		t.Errorf("text body not CRLF-separated: %q", body)
	}
}

func readQP(msg string) (string, error) {
	i := strings.Index(msg, "quoted-printable\r\n\r\n")
	if i < 0 {
		return "", nil
	}
	rest := msg[i+len("quoted-printable\r\n\r\n"):]
	if j := strings.Index(rest, "\r\n--"); j >= 0 {
		rest = rest[:j]
	}
	b, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(rest)))
	return string(b), err
}

func TestSTARTTLSRequiredWhenChosen(t *testing.T) {
	host, port, _ := fakeSMTP(t)
	cfg := Config{Host: host, Port: port, Security: SecuritySTARTTLS, From: "alerts@example.pk", Timeout: 5 * time.Second}
	err := Send(context.Background(), cfg, Message{To: []string{"a@example.pk"}, Subject: "x", Text: "x"})
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("expected a STARTTLS error, got %v", err)
	}
}

func TestNoPasswordOverPlainConnectionToAnotherHost(t *testing.T) {
	a := &loginAuth{user: "u", pass: "p", host: "mail.example.pk"}
	if _, _, err := a.Start(&smtp.ServerInfo{Name: "mail.example.pk"}); err == nil {
		t.Fatal("LOGIN must refuse an unencrypted connection to a remote host")
	}
	if mech, _, err := a.Start(&smtp.ServerInfo{Name: "mail.example.pk", TLS: true}); err != nil || mech != "LOGIN" {
		t.Fatalf("LOGIN over TLS: %s %v", mech, err)
	}
	if r, _ := a.Next([]byte("Password:"), true); string(r) != "p" {
		t.Fatalf("password challenge answered %q", r)
	}
}

func TestValidateAndRecipients(t *testing.T) {
	if err := (Config{Host: "smtp.office365.com", Port: 587, Security: SecuritySTARTTLS, From: "x"}).Validate(); err == nil {
		t.Error("bad sender accepted")
	}
	if err := (Config{Host: "", Port: 587, Security: SecuritySTARTTLS, From: "a@b.pk"}).Validate(); err == nil {
		t.Error("missing host accepted")
	}
	list, err := ParseRecipients("a@b.pk; Accounts <acc@b.pk>,\n c@d.pk")
	if err != nil || strings.Join(list, ",") != "a@b.pk,acc@b.pk,c@d.pk" {
		t.Fatalf("recipients %v %v", list, err)
	}
	if _, err := ParseRecipients("not an address"); err == nil {
		t.Error("invalid recipient accepted")
	}
}
