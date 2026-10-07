// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/store"
)

// ReturnDeadline is a due date of the monthly sales tax return of a tax
// period (a calendar month).
type ReturnDeadline struct {
	Period      string `json:"period"`      // YYYY-MM
	PeriodLabel string `json:"periodLabel"` // e.g. "September 2026"
	Kind        string `json:"kind"`        // "payment" or "filing"
	Due         string `json:"due"`         // YYYY-MM-DD
	DaysLeft    int    `json:"daysLeft"`    // negative once overdue
}

func dayStart(t time.Time) time.Time {
	t = t.In(PKT)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, PKT)
}

// ReturnDueDate is the given day of the month after the tax period, moved
// to the month's last day when the month is shorter.
func ReturnDueDate(period time.Time, day int) time.Time {
	first := time.Date(period.Year(), period.Month()+1, 1, 0, 0, 0, 0, PKT)
	last := first.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	if day < 1 {
		day = 1
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, PKT)
}

func returnDays(c *store.Company) (payment, filing int) {
	payment, filing = c.ReturnPaymentDay, c.ReturnFilingDay
	if payment <= 0 {
		payment = store.DefaultReturnPaymentDay
	}
	if filing <= 0 {
		filing = store.DefaultReturnFilingDay
	}
	return payment, filing
}

// PeriodDeadlines returns the payment and filing deadlines of a tax period,
// earliest first.
func PeriodDeadlines(period time.Time, paymentDay, filingDay int, today time.Time) []ReturnDeadline {
	p := time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, PKT)
	t := dayStart(today)
	mk := func(kind string, day int) ReturnDeadline {
		due := ReturnDueDate(p, day)
		return ReturnDeadline{Period: p.Format("2006-01"), PeriodLabel: p.Format("January 2006"), Kind: kind,
			Due: due.Format("2006-01-02"), DaysLeft: int(math.Round(due.Sub(t).Hours() / 24))}
	}
	out := []ReturnDeadline{mk("payment", paymentDay), mk("filing", filingDay)}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Due < out[j].Due })
	return out
}

// UpcomingDeadlines returns the deadlines of the return being prepared:
// last month's until its last deadline has passed, then this month's.
func UpcomingDeadlines(today time.Time, paymentDay, filingDay int) []ReturnDeadline {
	t := dayStart(today)
	prev := time.Date(t.Year(), t.Month()-1, 1, 0, 0, 0, 0, PKT)
	last := ReturnDueDate(prev, max(paymentDay, filingDay))
	if !t.After(last) {
		return PeriodDeadlines(prev, paymentDay, filingDay, t)
	}
	return PeriodDeadlines(time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, PKT), paymentDay, filingDay, t)
}

// CompanyDeadlines returns the upcoming return deadlines of a company.
func (s *Service) CompanyDeadlines(c *store.Company) []ReturnDeadline {
	p, f := returnDays(c)
	return UpcomingDeadlines(s.Now(), p, f)
}

// PeriodReview summarises one tax period for the monthly return: what was
// reported to FBR (and so appears in Annexure-C), what was not, and the
// incidents to account for.
type PeriodReview struct {
	Period      string                `json:"period"`
	PeriodLabel string                `json:"periodLabel"`
	From        string                `json:"from"`
	To          string                `json:"to"`
	Environment domain.Environment    `json:"environment"`
	Deadlines   []ReturnDeadline      `json:"deadlines"`
	Summary     *store.PeriodRow      `json:"summary"`
	BySaleType  []store.TaxSummaryRow `json:"bySaleType"`
	Statuses    []store.PeriodStatus  `json:"statuses"`
	Incidents   []*store.Incident     `json:"incidents"`
	Checks      []PeriodCheck         `json:"checks"`
	Counts      map[string]int        `json:"counts"`
	TopItems    []store.ItemRow       `json:"topItems"`
	TopBuyers   []store.CustomerRow   `json:"topBuyers"`
}

// PeriodCheck is one item of the period-close checklist.
type PeriodCheck struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Link   string `json:"link,omitempty"`
}

// ReviewPeriod builds the period review for a company's tax period
// ("YYYY-MM"; empty means the period of the return being prepared).
func (s *Service) ReviewPeriod(ctx context.Context, c *store.Company, env domain.Environment, period string) (*PeriodReview, error) {
	if env == "" {
		env = c.Environment
	}
	var p time.Time
	if period == "" {
		p, _ = time.ParseInLocation("2006-01", s.CompanyDeadlines(c)[0].Period, PKT)
	} else {
		var err error
		p, err = time.ParseInLocation("2006-01", period, PKT)
		if err != nil {
			return nil, Invalid("tax period must be YYYY-MM")
		}
	}
	from := p.Format("2006-01-02")
	to := p.AddDate(0, 1, -1).Format("2006-01-02")
	pd, fd := returnDays(c)
	r := &PeriodReview{Period: p.Format("2006-01"), PeriodLabel: p.Format("January 2006"), From: from, To: to, Environment: env,
		Deadlines: PeriodDeadlines(p, pd, fd, s.Now()), Counts: map[string]int{}}
	f := store.ReportFilter{CompanyID: c.ID, Environment: env, From: from, To: to}
	monthly, err := s.Store.MonthlySummary(ctx, f)
	if err != nil {
		return nil, err
	}
	r.Summary = &store.PeriodRow{Period: r.Period}
	if len(monthly) > 0 {
		r.Summary = &monthly[0]
	}
	if r.BySaleType, err = s.Store.TaxSummary(ctx, f); err != nil {
		return nil, err
	}
	if r.Statuses, err = s.Store.PeriodStatusCounts(ctx, c.ID, env, from, to); err != nil {
		return nil, err
	}
	if r.TopItems, err = s.Store.TopItems(ctx, f, 5); err != nil {
		return nil, err
	}
	buyers, err := s.Store.CustomerSummary(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(buyers) > 5 {
		buyers = buyers[:5]
	}
	r.TopBuyers = buyers
	// Incidents overlapping the period (timestamps are stored in UTC).
	pStart := p.UTC().Format(time.RFC3339)
	pEnd := p.AddDate(0, 1, 0).UTC().Format(time.RFC3339)
	if r.Incidents, err = s.Store.IncidentsBetween(ctx, c.ID, pStart, pEnd); err != nil {
		return nil, err
	}
	for _, st := range r.Statuses {
		r.Counts[st.Status] += st.Count
	}
	unreportedIncidents := 0
	for _, i := range r.Incidents {
		if i.ReportedAt == "" {
			unreportedIncidents++
		}
	}
	link := func(statuses string) string {
		return "/invoices?status=" + statuses + "&from=" + from + "&to=" + to
	}
	open := r.Counts[string(domain.StatusQueued)] + r.Counts[string(domain.StatusSubmitting)]
	r.Checks = []PeriodCheck{
		{ID: "unreported", OK: open == 0, Title: "Every issued invoice is reported to FBR",
			Detail: plural(open, "invoice is", "invoices are") + " issued but not yet reported. They are sent automatically as soon as FBR responds; " +
				"invoices issued during an outage must be uploaded within 24 hours of the connection being restored.",
			Link: link("QUEUED,SUBMITTING")},
		{ID: "rejected", OK: r.Counts[string(domain.StatusRejected)] == 0, Title: "No rejected invoices left uncorrected",
			Detail: plural(r.Counts[string(domain.StatusRejected)], "invoice was", "invoices were") + " rejected by FBR. Correct and resubmit them " +
				"so that they appear in Annexure-C.", Link: link("REJECTED")},
		{ID: "uncertain", OK: r.Counts[string(domain.StatusUncertain)] == 0, Title: "No submissions awaiting reconciliation",
			Detail: plural(r.Counts[string(domain.StatusUncertain)], "submission has", "submissions have") + " an unknown result. Check them on " +
				"IRIS and record the outcome.", Link: link("UNCERTAIN")},
		{ID: "drafts", OK: r.Counts[string(domain.StatusDraft)]+r.Counts[string(domain.StatusValidated)] == 0,
			Title: "No unissued drafts dated in this period",
			Detail: plural(r.Counts[string(domain.StatusDraft)]+r.Counts[string(domain.StatusValidated)], "draft is", "drafts are") +
				" dated in this period. An invoice must be issued and reported at the time of supply; issue or delete them.",
			Link: link("DRAFT,VALIDATED")},
		{ID: "incidents", OK: unreportedIncidents == 0, Title: "Operational failures reported to FBR (rule 150R)",
			Detail: plural(unreportedIncidents, "incident in this period is", "incidents in this period are") + " not marked as reported. " +
				"Report operational failures to FBR within 24 hours and record the reference.", Link: "/incidents"},
	}
	if env == domain.EnvSimulator {
		r.Checks = append(r.Checks, PeriodCheck{ID: "simulator", OK: false, Title: "These figures come from the training simulator",
			Detail: "Nothing in the simulator is reported to FBR. Switch the company to production to issue real invoices.", Link: "/settings/fbr"})
	}
	return r, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// DuplicateInvoice creates a draft sale invoice with the buyer and lines of
// an existing sale invoice, dated today, in the company's current
// environment. Amounts are recomputed by the tax engine.
func (s *Service) DuplicateInvoice(ctx context.Context, a Actor, companyID, id int64) (*store.Invoice, error) {
	orig, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if orig.DocType != domain.DocSaleInvoice {
		return nil, Invalid("only sale invoices can be duplicated; raise a debit note from the original sale invoice instead")
	}
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	in := &InvoiceInput{Environment: c.Environment, DocType: string(domain.DocSaleInvoice), InvoiceDate: s.Today(), CustomerID: orig.CustomerID,
		Buyer: &BuyerInput{NTNCNIC: orig.BuyerNTNCNIC, Name: orig.BuyerName, Province: orig.BuyerProvince, Address: orig.BuyerAddress,
			RegistrationType: string(orig.BuyerRegistrationType)},
		WithholdingMode: &orig.WithholdingMode, Notes: orig.Notes, Source: "ui"}
	if c.Environment == domain.EnvSandbox {
		in.ScenarioID = orig.ScenarioID
	}
	for _, it := range orig.Items {
		in.Items = append(in.Items, ItemInput{ProductID: it.ProductID, HSCode: it.HSCode, Description: it.Description, UoM: it.UoM,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, DiscountPercent: it.DiscountPercent, DiscountAmount: it.DiscountAmount,
			Value: it.ValueOverride, SaleType: it.SaleType, Rate: it.Rate, RetailPrice: it.RetailPrice, RetailValue: it.RetailValueOverride,
			FurtherTaxMode: it.FurtherTaxMode, ExtraTaxRate: it.ExtraTaxRate, FEDRate: it.FEDRate,
			SROScheduleNo: it.SROScheduleNo, SROItemSerialNo: it.SROItemSerialNo})
	}
	inv, err := s.CreateInvoice(ctx, a, companyID, in)
	if err != nil {
		return nil, err
	}
	s.Audit(ctx, a, companyID, "invoice.duplicate", "invoice", fmt.Sprint(inv.ID), map[string]any{"from": orig.InternalNo, "fromId": orig.ID})
	return inv, nil
}
