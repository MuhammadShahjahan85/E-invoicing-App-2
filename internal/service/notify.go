// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/domain"
	"einvoicing/internal/mail"
	"einvoicing/internal/store"
)

// E-mail notifications tell the people responsible about problems that
// need action (FBR unreachable, rejected invoices, invoices not reported
// within 24 hours, unreported incidents, return deadlines) even when nobody
// has the system open, and can send a daily summary.

const (
	settingNotify         = "notify.email"
	settingNotifyPassword = "notify.email.password"
	settingNotifyState    = "notify.email.state"
	notifyEvery           = 5 * time.Minute
	notifyRepeatAfter     = 24 * time.Hour
)

// NotifySettings configures e-mail notifications for the installation.
type NotifySettings struct {
	Enabled     bool     `json:"enabled"`
	SMTPHost    string   `json:"smtpHost"`
	SMTPPort    int      `json:"smtpPort"`
	Security    string   `json:"security"`
	Username    string   `json:"username"`
	HasPassword bool     `json:"hasPassword"`
	From        string   `json:"from"`
	To          []string `json:"to"`
	// Immediate e-mails new errors and warnings as they arise (each at most
	// once a day while it lasts).
	Immediate bool `json:"immediate"`
	// DigestHour is the hour (Pakistan time) of the daily summary; -1 = off.
	DigestHour int `json:"digestHour"`
	// AppURL, when set, is linked from the e-mails (e.g. https://192.168.1.10:8443).
	AppURL    string `json:"appUrl"`
	LastSent  string `json:"lastSent"`
	LastError string `json:"lastError"`
}

// NotifyInput updates the settings. Password nil keeps the stored password;
// an empty string removes it.
type NotifyInput struct {
	NotifySettings
	Password *string `json:"password"`
}

// NotifySettings returns the current settings.
func (s *Service) NotifySettings(ctx context.Context) (NotifySettings, error) {
	ns := NotifySettings{SMTPPort: 587, Security: mail.SecuritySTARTTLS, Immediate: true, DigestHour: -1}
	v, err := s.Store.GetSetting(ctx, settingNotify)
	if err != nil {
		return ns, err
	}
	if v != "" {
		if err := json.Unmarshal([]byte(v), &ns); err != nil {
			return ns, err
		}
	}
	if ns.To == nil {
		ns.To = []string{}
	}
	pw, _ := s.Store.GetSetting(ctx, settingNotifyPassword)
	ns.HasPassword = pw != ""
	return ns, nil
}

// SaveNotifySettings validates and stores the settings.
func (s *Service) SaveNotifySettings(ctx context.Context, a Actor, in NotifyInput) (NotifySettings, error) {
	cur, err := s.NotifySettings(ctx)
	if err != nil {
		return cur, err
	}
	ns := in.NotifySettings
	ns.LastSent, ns.LastError, ns.HasPassword = cur.LastSent, cur.LastError, false
	ns.SMTPHost = strings.TrimSpace(ns.SMTPHost)
	ns.Username = strings.TrimSpace(ns.Username)
	ns.From = strings.TrimSpace(ns.From)
	ns.AppURL = strings.TrimRight(strings.TrimSpace(ns.AppURL), "/")
	if ns.SMTPPort == 0 {
		ns.SMTPPort = 587
	}
	if ns.Security == "" {
		ns.Security = mail.SecuritySTARTTLS
	}
	to, err := mail.ParseRecipients(strings.Join(ns.To, ","))
	if err != nil {
		return cur, Invalid("%s", err.Error())
	}
	ns.To = to
	if ns.DigestHour < -1 || ns.DigestHour > 23 {
		return cur, Invalid("the daily summary hour must be between 0 and 23")
	}
	if ns.AppURL != "" {
		u, err := url.Parse(ns.AppURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return cur, Invalid("the link to the system must start with https:// (for example https://192.168.1.10:8443)")
		}
	}
	if ns.Enabled {
		cfg := mail.Config{Host: ns.SMTPHost, Port: ns.SMTPPort, Security: ns.Security, From: ns.From}
		if err := cfg.Validate(); err != nil {
			return cur, Invalid("%s", err.Error())
		}
		if len(ns.To) == 0 {
			return cur, Invalid("enter at least one recipient")
		}
	}
	b, _ := json.Marshal(ns)
	if err := s.Store.SetSetting(ctx, settingNotify, string(b)); err != nil {
		return cur, err
	}
	if in.Password != nil {
		enc := ""
		if *in.Password != "" {
			if enc, err = s.Vault.Encrypt(*in.Password); err != nil {
				return cur, err
			}
		}
		if err := s.Store.SetSetting(ctx, settingNotifyPassword, enc); err != nil {
			return cur, err
		}
	}
	s.Audit(ctx, a, 0, "system.notifications", "settings", settingNotify, map[string]any{"enabled": ns.Enabled, "host": ns.SMTPHost,
		"to": ns.To, "immediate": ns.Immediate, "digestHour": ns.DigestHour, "passwordChanged": in.Password != nil})
	return s.NotifySettings(ctx)
}

func (s *Service) mailConfig(ctx context.Context, ns NotifySettings) (mail.Config, error) {
	pw := ""
	if enc, _ := s.Store.GetSetting(ctx, settingNotifyPassword); enc != "" {
		var err error
		if pw, err = s.Vault.Decrypt(enc); err != nil {
			return mail.Config{}, fmt.Errorf("the stored e-mail password cannot be read: %w", err)
		}
	}
	return mail.Config{Host: ns.SMTPHost, Port: ns.SMTPPort, Security: ns.Security, Username: ns.Username, Password: pw,
		From: ns.From, Timeout: 30 * time.Second}, nil
}

// recordNotifyResult stores the outcome of the last e-mail for the settings page.
func (s *Service) recordNotifyResult(ctx context.Context, sendErr error) {
	ns, err := s.NotifySettings(ctx)
	if err != nil {
		return
	}
	if sendErr != nil {
		ns.LastError = s.Now().In(PKT).Format("02 Jan 2006 15:04") + ": " + sendErr.Error()
	} else {
		ns.LastSent, ns.LastError = s.Now().UTC().Format(time.RFC3339), ""
	}
	ns.HasPassword = false
	b, _ := json.Marshal(ns)
	_ = s.Store.SetSetting(ctx, settingNotify, string(b))
}

// SendTestEmail sends a test message to the configured recipients.
func (s *Service) SendTestEmail(ctx context.Context, a Actor) error {
	ns, err := s.NotifySettings(ctx)
	if err != nil {
		return err
	}
	if len(ns.To) == 0 {
		return Invalid("enter at least one recipient and save the settings first")
	}
	cfg, err := s.mailConfig(ctx, ns)
	if err != nil {
		return err
	}
	text := "This is a test e-mail from " + brand.ProductName + ".\n\nE-mail notifications are working. You will be told about problems that need action, such as FBR being unreachable, rejected invoices and invoices not reported within 24 hours." + footerText(ns.AppURL)
	body := `<p>This is a test e-mail from <b>` + html.EscapeString(brand.ProductName) + `</b>.</p><p>E-mail notifications are working. You will be told about problems that need action, such as FBR being unreachable, rejected invoices and invoices not reported within 24 hours.</p>`
	err = mail.Send(ctx, cfg, mail.Message{To: ns.To, Subject: brand.ProductName + " — test e-mail", Text: text, HTML: wrapHTML("Test e-mail", body, ns.AppURL)})
	s.recordNotifyResult(ctx, err)
	s.Audit(ctx, a, 0, "system.notifications.test", "settings", settingNotify, map[string]any{"to": ns.To, "ok": err == nil})
	if err != nil {
		return Invalid("the test e-mail could not be sent: %s", err.Error())
	}
	return nil
}

// runNotifications is called by the worker. It does its work at most every
// few minutes and never runs twice at the same time.
func (s *Service) runNotifications(ctx context.Context) {
	if !s.notifyMu.TryLock() {
		return
	}
	defer s.notifyMu.Unlock()
	if time.Since(s.lastNotifyRun) < notifyEvery {
		return
	}
	s.lastNotifyRun = time.Now()
	s.notifyOnce(ctx)
}

type companyAlert struct {
	Company *store.Company
	Alert   Alert
	Key     string
}

// notifyOnce sends the e-mails that are due now.
func (s *Service) notifyOnce(ctx context.Context) {
	ns, err := s.NotifySettings(ctx)
	if err != nil || !ns.Enabled || len(ns.To) == 0 {
		return
	}
	cfg, err := s.mailConfig(ctx, ns)
	if err != nil {
		s.recordNotifyResult(ctx, err)
		return
	}
	state := map[string]string{}
	if v, _ := s.Store.GetSetting(ctx, settingNotifyState); v != "" {
		_ = json.Unmarshal([]byte(v), &state)
	}
	now := s.Now()
	companies, err := s.Store.ListCompanies(ctx)
	if err != nil {
		return
	}
	var active []*store.Company
	for _, c := range companies {
		if c.Active {
			active = append(active, c)
		}
	}

	if ns.Immediate {
		var fresh []companyAlert
		current := map[string]bool{}
		consider := func(c *store.Company, list []Alert) {
			for _, a := range list {
				if a.Severity == "info" {
					continue
				}
				id := int64(0)
				if c != nil {
					id = c.ID
				}
				key := fmt.Sprintf("a/%d/%s", id, a.ID)
				current[key] = true
				if t := store.ParseTime(state[key]); !t.IsZero() && now.Sub(t) < notifyRepeatAfter {
					continue
				}
				fresh = append(fresh, companyAlert{Company: c, Alert: a, Key: key})
			}
		}
		for _, c := range active {
			// Nothing in the training simulator is reported to FBR.
			if c.Environment == domain.EnvSimulator {
				continue
			}
			consider(c, s.companyAlerts(ctx, c, c.Environment))
		}
		consider(nil, s.systemAlerts(ctx, true))
		// Forget alerts that have cleared, so that a recurrence is e-mailed.
		for k := range state {
			if strings.HasPrefix(k, "a/") && !current[k] {
				delete(state, k)
			}
		}
		if len(fresh) > 0 {
			msg := alertMail(fresh, ns)
			err := mail.Send(ctx, cfg, msg)
			s.recordNotifyResult(ctx, err)
			if err != nil {
				s.Log.Warn("notification e-mail failed", "err", err)
			} else {
				for _, f := range fresh {
					state[f.Key] = now.UTC().Format(time.RFC3339)
				}
			}
		}
	}

	if ns.DigestHour >= 0 {
		pk := now.In(PKT)
		today := pk.Format("2006-01-02")
		if pk.Hour() >= ns.DigestHour && state["digest"] != today {
			err := mail.Send(ctx, cfg, s.digestMail(ctx, active, pk, ns))
			s.recordNotifyResult(ctx, err)
			if err != nil {
				s.Log.Warn("daily summary e-mail failed", "err", err)
			} else {
				state["digest"] = today
			}
		}
	}
	b, _ := json.Marshal(state)
	_ = s.Store.SetSetting(ctx, settingNotifyState, string(b))
}

// alertMail lists new problems, most serious first.
func alertMail(list []companyAlert, ns NotifySettings) mail.Message {
	sort.SliceStable(list, func(i, j int) bool { return sevRank(list[i].Alert.Severity) < sevRank(list[j].Alert.Severity) })
	errors := 0
	for _, x := range list {
		if x.Alert.Severity == "error" {
			errors++
		}
	}
	subject := fmt.Sprintf("%d item(s) need attention", len(list))
	if errors > 0 {
		subject = fmt.Sprintf("Action needed: %s", list[0].Alert.Title)
		if len(list) > 1 {
			subject += fmt.Sprintf(" (+%d more)", len(list)-1)
		}
	}
	var t, h strings.Builder
	t.WriteString("The following need attention in " + brand.ProductName + ":\n\n")
	h.WriteString(`<p style="margin:0 0 14px">The following need attention:</p>`)
	for _, x := range list {
		who := "Installation"
		if x.Company != nil {
			who = x.Company.Name + " (NTN " + x.Company.NTNCNIC + ", " + x.Company.Environment.Label() + ")"
		}
		fmt.Fprintf(&t, "- [%s] %s — %s\n  %s\n", strings.ToUpper(x.Alert.Severity), who, x.Alert.Title, x.Alert.Detail)
		color := "#9a6b00"
		if x.Alert.Severity == "error" {
			color = "#b42318"
		}
		link := ""
		if ns.AppURL != "" && x.Alert.Link != "" {
			link = ` <a href="` + html.EscapeString(ns.AppURL+x.Alert.Link) + `" style="color:#127a60">Open</a>`
		}
		fmt.Fprintf(&h, `<div style="border-left:3px solid %s;padding:8px 12px;margin:0 0 10px;background:#f7faf9"><div style="font-size:12px;color:#4a5d57">%s</div><div style="font-weight:600;margin:2px 0">%s</div><div style="font-size:13px;color:#4a5d57">%s%s</div></div>`,
			color, html.EscapeString(who), html.EscapeString(x.Alert.Title), html.EscapeString(x.Alert.Detail), link)
	}
	t.WriteString("\nEach item is repeated at most once a day while it lasts." + footerText(ns.AppURL))
	return mail.Message{To: ns.To, Subject: brand.ProductName + " — " + subject, Text: t.String(), HTML: wrapHTML("Needs attention", h.String(), ns.AppURL)}
}

func sevRank(s string) int {
	switch s {
	case "error":
		return 0
	case "warning":
		return 1
	}
	return 2
}

// digestMail summarises yesterday, the month to date and open items.
func (s *Service) digestMail(ctx context.Context, companies []*store.Company, pk time.Time, ns NotifySettings) mail.Message {
	yesterday := pk.AddDate(0, 0, -1).Format("2006-01-02")
	monthStart := pk.Format("2006-01") + "-01"
	var t, h strings.Builder
	fmt.Fprintf(&t, "Daily summary for %s\n\n", pk.Format("Monday, 02 January 2006"))
	for _, c := range companies {
		env := c.Environment
		d, err := s.Store.GetDashboard(ctx, c.ID, env, yesterday, monthStart)
		if err != nil {
			continue
		}
		fmt.Fprintf(&t, "%s (NTN %s) — %s\n", c.Name, c.NTNCNIC, env.Label())
		fmt.Fprintf(&t, "  Yesterday: %d invoice(s) accepted, value Rs %s, sales tax Rs %s\n", d.TodayCount, d.TodayValue.StringFixed(2), d.TodaySalesTax.StringFixed(2))
		fmt.Fprintf(&t, "  This month: %d invoice(s), value Rs %s, sales tax Rs %s\n", d.MonthCount, d.MonthValue.StringFixed(2), d.MonthSalesTax.StringFixed(2))
		h.WriteString(`<h3 style="margin:18px 0 6px;font-size:15px">` + html.EscapeString(c.Name) + ` <span style="font-weight:400;color:#75877f;font-size:12px">NTN ` +
			html.EscapeString(c.NTNCNIC) + ` · ` + html.EscapeString(env.Label()) + `</span></h3>`)
		h.WriteString(`<table cellpadding="6" cellspacing="0" style="border-collapse:collapse;font-size:13px;width:100%">`)
		row := func(k, v string) {
			h.WriteString(`<tr><td style="border-bottom:1px solid #e1e8e5;color:#4a5d57">` + html.EscapeString(k) + `</td><td style="border-bottom:1px solid #e1e8e5;text-align:right">` + html.EscapeString(v) + `</td></tr>`)
		}
		row("Accepted yesterday", fmt.Sprintf("%d invoice(s) · Rs %s · tax Rs %s", d.TodayCount, d.TodayValue.StringFixed(2), d.TodaySalesTax.StringFixed(2)))
		row("This month", fmt.Sprintf("%d invoice(s) · Rs %s · tax Rs %s", d.MonthCount, d.MonthValue.StringFixed(2), d.MonthSalesTax.StringFixed(2)))
		if d.NeedsAttention > 0 {
			row("Rejected, queued or unreconciled", fmt.Sprint(d.NeedsAttention))
			fmt.Fprintf(&t, "  Needs attention: %d (rejected, queued or unreconciled)\n", d.NeedsAttention)
		}
		for _, dl := range s.CompanyDeadlines(ctx, c) {
			what := "Pay sales tax"
			if dl.Kind == "filing" {
				what = "File the return"
			}
			when := fmt.Sprintf("%s (%d days)", dl.Due, dl.DaysLeft)
			if dl.DaysLeft < 0 {
				when = dl.Due + " (passed)"
			}
			row(what+" for "+dl.PeriodLabel, when)
			fmt.Fprintf(&t, "  %s for %s: %s\n", what, dl.PeriodLabel, when)
		}
		h.WriteString(`</table>`)
		if env != domain.EnvSimulator {
			for _, a := range s.companyAlerts(ctx, c, env) {
				fmt.Fprintf(&t, "  * %s\n", a.Title)
				h.WriteString(`<div style="font-size:13px;margin:6px 0 0">• ` + html.EscapeString(a.Title) + `</div>`)
			}
		}
		t.WriteString("\n")
	}
	for _, a := range s.systemAlerts(ctx, true) {
		fmt.Fprintf(&t, "* %s\n", a.Title)
		h.WriteString(`<div style="font-size:13px;margin:10px 0 0">• ` + html.EscapeString(a.Title) + `</div>`)
	}
	t.WriteString(footerText(ns.AppURL))
	return mail.Message{To: ns.To, Subject: brand.ProductName + " — daily summary " + pk.Format("02 Jan 2006"), Text: t.String(),
		HTML: wrapHTML("Daily summary — "+pk.Format("Monday, 02 January 2006"), h.String(), ns.AppURL)}
}

func footerText(appURL string) string {
	s := "\n\n--\n" + brand.ProductName + " · " + brand.DevelopedBy + "\nChange these e-mails under Settings → System & licence → E-mail notifications."
	if appURL != "" {
		s += "\nOpen the system: " + appURL
	}
	return s
}

// wrapHTML lays an e-mail body out in the Veridian style (inline styles for
// e-mail clients).
func wrapHTML(title, body, appURL string) string {
	open := ""
	if appURL != "" {
		open = `<p style="margin:20px 0 0"><a href="` + html.EscapeString(appURL) + `" style="display:inline-block;background:#127a60;color:#ffffff;text-decoration:none;padding:9px 16px;border-radius:8px;font-weight:600">Open ` + html.EscapeString(brand.ProductName) + `</a></p>`
	}
	return `<!doctype html><html><body style="margin:0;background:#f2f5f4;font-family:Segoe UI,Arial,sans-serif;color:#0e1d19">` +
		`<div style="max-width:640px;margin:0 auto;padding:24px 16px">` +
		`<div style="background:#0e322b;border-radius:12px 12px 0 0;padding:16px 20px;color:#ffffff"><span style="font-size:18px;font-weight:700">Veridian</span>` +
		`<span style="display:block;font-size:10px;letter-spacing:2px;color:#e3b341;font-weight:600">E-INVOICING PAKISTAN</span></div>` +
		`<div style="background:#ffffff;border:1px solid #e1e8e5;border-top:0;border-radius:0 0 12px 12px;padding:20px">` +
		`<h2 style="margin:0 0 14px;font-size:18px">` + html.EscapeString(title) + `</h2>` + body + open + `</div>` +
		`<p style="font-size:11px;color:#75877f;text-align:center;margin:14px 0 0">` + html.EscapeString(brand.ProductName+" · "+brand.DevelopedBy) +
		`<br>Change these e-mails under Settings → System &amp; licence → E-mail notifications.</p></div></body></html>`
}
