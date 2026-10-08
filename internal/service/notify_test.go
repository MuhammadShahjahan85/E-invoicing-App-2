// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"bufio"
	"io"
	"mime/quotedprintable"
	"net"
	"strings"
	"sync"
	"testing"

	"einvoicing/internal/mail"
)

// mailbox is a fake SMTP server that keeps every message it receives.
type mailbox struct {
	mu   sync.Mutex
	msgs []string
	port int
}

func newMailbox(t *testing.T) *mailbox {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	mb := &mailbox{port: ln.Addr().(*net.TCPAddr).Port}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go mb.serve(conn)
		}
	}()
	return mb
}

func (mb *mailbox) serve(conn net.Conn) {
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
				mb.mu.Lock()
				mb.msgs = append(mb.msgs, decodeBodies(data.String()))
				mb.mu.Unlock()
				data.Reset()
				w("250 queued")
				continue
			}
			data.WriteString(line)
			continue
		}
		switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
		case strings.HasPrefix(cmd, "EHLO"):
			w("250 test")
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
}

// decodeBodies returns the headers plus the decoded text of every part.
func decodeBodies(msg string) string {
	out := msg
	for _, part := range strings.Split(msg, "quoted-printable\r\n\r\n")[1:] {
		if i := strings.Index(part, "\r\n--"); i >= 0 {
			part = part[:i]
		}
		b, _ := io.ReadAll(quotedprintable.NewReader(strings.NewReader(part)))
		out += "\n" + string(b)
	}
	return out
}

func (mb *mailbox) all() []string {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	return append([]string(nil), mb.msgs...)
}

func TestEmailNotifications(t *testing.T) {
	f := setup(t)
	mb := newMailbox(t)
	pw := ""
	in := NotifyInput{NotifySettings: NotifySettings{Enabled: true, SMTPHost: "127.0.0.1", SMTPPort: mb.port, Security: mail.SecurityNone,
		From: "Veridian <alerts@example.pk>", To: []string{"accounts@example.pk; owner@example.pk"}, Immediate: true, DigestHour: -1,
		AppURL: "https://192.168.1.10:8443/"}, Password: &pw}
	ns, err := f.svc.SaveNotifySettings(f.ctx, f.admin, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.To) != 2 || ns.AppURL != "https://192.168.1.10:8443" || ns.HasPassword {
		t.Fatalf("settings %+v", ns)
	}
	// Invalid settings are refused.
	bad := in
	bad.SMTPHost = ""
	if _, err := f.svc.SaveNotifySettings(f.ctx, f.admin, bad); err == nil {
		t.Fatal("missing SMTP server accepted")
	}

	if err := f.svc.SendTestEmail(f.ctx, f.admin); err != nil {
		t.Fatal(err)
	}
	if got := mb.all(); len(got) != 1 || !strings.Contains(got[0], "test e-mail") {
		t.Fatalf("test e-mail: %v", got)
	}

	// No backup exists yet: an immediate e-mail about it, once.
	f.svc.notifyOnce(f.ctx)
	got := mb.all()
	if len(got) != 2 || !strings.Contains(got[1], "No database backup yet") || !strings.Contains(got[1], "https://192.168.1.10:8443/settings/system") {
		t.Fatalf("alert e-mail: %d %v", len(got), got[len(got)-1])
	}
	f.svc.notifyOnce(f.ctx)
	if n := len(mb.all()); n != 2 {
		t.Fatalf("the same alert was e-mailed again within a day (%d e-mails)", n)
	}
	// Training-simulator companies do not raise e-mails about invoices.
	cust, prod := f.customer(t), f.product(t)
	if _, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{CustomerID: &cust.ID, Items: []ItemInput{{ProductID: &prod.ID}}}); err != nil {
		t.Fatal(err)
	}
	f.svc.notifyOnce(f.ctx)
	if n := len(mb.all()); n != 2 {
		t.Fatalf("simulator draft caused an e-mail (%d e-mails)", n)
	}

	// Daily summary at the chosen hour, once a day.
	in.DigestHour = f.svc.Now().In(PKT).Hour()
	in.Password = nil
	if _, err := f.svc.SaveNotifySettings(f.ctx, f.admin, in); err != nil {
		t.Fatal(err)
	}
	f.svc.notifyOnce(f.ctx)
	f.svc.notifyOnce(f.ctx)
	got = mb.all()
	if len(got) != 3 || !strings.Contains(got[2], "Daily summary for") || !strings.Contains(got[2], "Company 8") ||
		!strings.Contains(got[2], "Pay sales tax") {
		t.Fatalf("daily summary: %d %v", len(got), got[len(got)-1])
	}
	// The last result is shown on the settings page.
	ns, _ = f.svc.NotifySettings(f.ctx)
	if ns.LastSent == "" || ns.LastError != "" {
		t.Fatalf("last result %+v", ns)
	}
}

func TestNotifyPasswordIsEncrypted(t *testing.T) {
	f := setup(t)
	pw := "s3cret-app-password"
	if _, err := f.svc.SaveNotifySettings(f.ctx, f.admin, NotifyInput{NotifySettings: NotifySettings{SMTPHost: "smtp.office365.com", SMTPPort: 587,
		Security: mail.SecuritySTARTTLS, From: "alerts@example.pk", To: []string{"a@example.pk"}}, Password: &pw}); err != nil {
		t.Fatal(err)
	}
	stored, _ := f.svc.Store.GetSetting(f.ctx, settingNotifyPassword)
	if stored == "" || strings.Contains(stored, pw) {
		t.Fatalf("password stored as %q", stored)
	}
	plain, _ := f.svc.Store.GetSetting(f.ctx, settingNotify)
	if strings.Contains(plain, pw) {
		t.Fatal("password leaked into the settings JSON")
	}
	cfg, err := f.svc.mailConfig(f.ctx, NotifySettings{})
	if err != nil || cfg.Password != pw {
		t.Fatalf("decrypted password %q %v", cfg.Password, err)
	}
}
