// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"fmt"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/pdf"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"

	"github.com/shopspring/decimal"
)

// ReportPack renders the sales tax report pack of a tax period as a PDF:
// the period's figures, whether the books, FBR and Annexure-C tie, the
// period-close checklist, tax by sale type and rate, every document
// reported to FBR (to match with Annexure-C before filing, rule
// 150XD(2)), buyers, incidents and a sign-off block. An empty period means
// the period whose return is due next.
func (s *Service) ReportPack(ctx context.Context, c *store.Company, env domain.Environment, period string, m pdf.Meta) ([]byte, error) {
	if env == "" {
		env = c.Environment
	}
	rev, err := s.ReviewPeriod(ctx, c, env, period)
	if err != nil {
		return nil, err
	}
	tie, err := s.tieFor(ctx, c, env, rev)
	if err != nil {
		return nil, err
	}
	f := store.ReportFilter{CompanyID: c.ID, Environment: env, From: rev.From, To: rev.To}
	annex, err := s.Store.AnnexCReconciliation(ctx, f)
	if err != nil {
		return nil, err
	}
	buyers, err := s.Store.CustomerSummary(ctx, f)
	if err != nil {
		return nil, err
	}
	amt := func(v decimal.Decimal) string { return "Rs " + tax.FormatAmount(v) }

	m.Title = "Sales tax report pack · " + rev.PeriodLabel
	m.Subject = "Sales tax report pack for " + rev.PeriodLabel
	m.Environment = string(env)
	d := pdf.New(false, m)

	var pay, file ReturnDeadline
	for _, dl := range rev.Deadlines {
		if dl.Kind == "payment" {
			pay = dl
		} else {
			file = dl
		}
	}
	sub := "Prepared for the return of " + rev.PeriodLabel
	if file.Due != "" {
		sub += ", due " + longDate(file.Due)
	}
	sub += ". Every figure comes from the documents reported to FBR Digital Invoicing."
	d.Heading("Sales tax return · "+rev.PeriodLabel, "Sales tax report pack", sub)

	sm := rev.Summary
	net := sm.SalesTax.Sub(sm.DebitSalesTax)
	d.Tiles([]pdf.Tile{
		{Label: "Value of supplies", Value: amt(sm.ValueExclST), Sub: plural(sm.SaleInvoices, "sale invoice", "sale invoices") + " accepted", Hero: true},
		{Label: "Sales tax charged", Value: amt(sm.SalesTax), Sub: "Further tax " + amt(sm.FurtherTax)},
		{Label: "Debit notes", Value: amt(sm.DebitValue), Sub: plural(sm.DebitNotes, "note", "notes") + " · tax " + amt(sm.DebitSalesTax)},
		{Label: "Net sales tax", Value: amt(net), Sub: "Withheld by buyers " + amt(sm.STWithheld)},
	})

	// Books, FBR and Annexure-C.
	d.Section("Books, FBR and Annexure-C", "Each side ties when nothing is left to resolve. Annexure-C on IRIS is built from the documents FBR accepted.")
	var rows [3]pdf.TieRow
	for i, e := range tie.Edges {
		if i > 2 {
			break
		}
		detail := e.Detail
		if e.Count > 0 {
			detail = plural(e.Count, "document", "documents") + " to resolve"
		}
		rows[i] = pdf.TieRow{OK: e.Count == 0, Title: e.Title, Detail: detail, Value: tax.FormatAmount(e.Value)}
	}
	// The diagram's sides are books–FBR, books–Annexure-C and
	// FBR–Annexure-C; the list follows the same order.
	d.TiePanel([3]string{"BOOKS", "FBR", "ANNEX-C"}, [3]pdf.TieRow{rows[0], rows[2], rows[1]})

	kv := [][2]string{{"Tax period", rev.PeriodLabel + " (" + longDate(rev.From) + " to " + longDate(rev.To) + ")"}}
	if pay.Due != "" {
		kv = append(kv, [2]string{"Sales tax payment due", longDate(pay.Due)})
	}
	if file.Due != "" {
		v := longDate(file.Due)
		if file.OriginalDue != "" {
			v += " (extended by FBR from " + longDate(file.OriginalDue) + "; " + file.Reference + ")"
		}
		kv = append(kv, [2]string{"Return filing due", v})
	}
	kv = append(kv, [2]string{"Documents reported", fmt.Sprintf("%d accepted by FBR, %d issued in the period", tie.Accepted, tie.Issued)})
	kv = append(kv, [2]string{"Environment", envName(env)})
	d.KV(kv)

	d.Section("Period-close checklist", fmt.Sprintf("%d of %d checks need attention.", tie.ChecksOpen, tie.ChecksTotal))
	var checks []pdf.Check
	for _, ch := range rev.Checks {
		detail := ""
		if !ch.OK {
			detail = ch.Detail
		}
		checks = append(checks, pdf.Check{OK: ch.OK, Title: ch.Title, Detail: detail})
	}
	d.Checks(checks)

	// Tax by sale type and rate.
	d.Section("Sales tax by sale type and rate", "Documents accepted by FBR in the period.")
	st := TaxSummaryTable(rev.BySaleType)
	st.PDFCols = []int{0, 1, 2, 3, 5, 7, 8, 11}
	d.Table(st.Grid())

	// Buyers.
	d.Section("Buyers", "Value of supplies to each buyer in the period. Registered buyers claim input tax on these invoices through their Annexure-A.")
	bt := CustomerTable(buyers)
	bt.PDFCols = []int{0, 1, 2, 3, 4, 5, 8}
	d.Table(bt.Grid())

	// Every document reported to FBR.
	d.NewPage()
	d.Section("Documents reported to FBR (Annexure-C)", "Match these with Annexure-C of the return on IRIS before filing; investigate any document missing on either side (rule 150XD(2)).")
	at := AnnexCTable(annex)
	at.PDFCols = []int{0, 1, 3, 5, 6, 9, 10, 15, 16}
	d.Table(at.Grid())

	// Incidents.
	if len(rev.Incidents) > 0 {
		d.Section("Incidents in the period", "Operational failures must be reported to FBR and the Commissioner within 24 hours (rule 150XA(c)).")
		it := &Table{Title: "Incidents", Headers: []string{"Started", "Ended", "Kind", "Description", "Reported to FBR"}}
		for _, i := range rev.Incidents {
			rep := "Not reported"
			if i.ReportedAt != "" {
				rep = shortStamp(i.ReportedAt)
				if i.ReportReference != "" {
					rep += " · " + i.ReportReference
				}
			}
			ended := "Ongoing"
			if i.EndedAt != "" {
				ended = shortStamp(i.EndedAt)
			}
			it.Rows = append(it.Rows, []any{shortStamp(i.StartedAt), ended, incidentKind(i.Kind), i.Description, rep})
		}
		d.Table(it.Grid())
	}

	d.Section("Sign-off", "Review the pack, resolve the open checks and file the return on IRIS.")
	d.SignOff([]string{"Prepared by", "Reviewed by", "Approved by"})
	return d.Bytes()
}

// longDate formats YYYY-MM-DD as e.g. "Thu, 15 Oct 2026".
func longDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("Mon, 02 Jan 2006")
}

// shortStamp formats an RFC 3339 time in Pakistan time.
func shortStamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.In(PKT).Format("02 Jan 2006 15:04")
}

func envName(env domain.Environment) string {
	switch env {
	case domain.EnvProduction:
		return "FBR production"
	case domain.EnvSandbox:
		return "FBR sandbox (test submissions, not tax records)"
	default:
		return "Training simulator (nothing is reported to FBR)"
	}
}

func incidentKind(k string) string {
	switch k {
	case "fbr_unreachable":
		return "FBR unreachable"
	case "auth_failure":
		return "Token rejected"
	case "system_failure":
		return "System failure"
	case "power_failure":
		return "Power failure"
	case "tampering":
		return "Tampering"
	}
	return "Other"
}
