// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/store"

	"github.com/shopspring/decimal"
)

// ReturnDeadline is a due date of the monthly sales tax return of a tax
// period (a calendar month).
type ReturnDeadline struct {
	Period      string `json:"period"`      // YYYY-MM
	PeriodLabel string `json:"periodLabel"` // e.g. "September 2026"
	Kind        string `json:"kind"`        // "payment" or "filing"
	Due         string `json:"due"`         // YYYY-MM-DD
	DaysLeft    int    `json:"daysLeft"`    // negative once overdue
	// OriginalDue and Reference are set when FBR extended the filing date.
	OriginalDue string `json:"originalDue,omitempty"`
	Reference   string `json:"reference,omitempty"`
}

// applyExtension moves the filing deadline of a period to the date FBR
// extended it to. Payment dates are not changed: FBR's extension orders keep
// the condition that tax is deposited by the original due date.
func applyExtension(dls []ReturnDeadline, ext *store.ReturnExtension, today time.Time) []ReturnDeadline {
	if ext == nil {
		return dls
	}
	due, err := time.ParseInLocation("2006-01-02", ext.FilingDate, PKT)
	if err != nil {
		return dls
	}
	for i := range dls {
		if dls[i].Kind == "filing" && dls[i].Period == ext.Period {
			dls[i].OriginalDue, dls[i].Reference = dls[i].Due, ext.Reference
			dls[i].Due = due.Format("2006-01-02")
			dls[i].DaysLeft = int(math.Round(due.Sub(dayStart(today)).Hours() / 24))
		}
	}
	sort.SliceStable(dls, func(i, j int) bool { return dls[i].Due < dls[j].Due })
	return dls
}

// periodDeadlines returns a company's deadlines for a period, with any
// filing extension recorded for it.
func (s *Service) periodDeadlines(ctx context.Context, c *store.Company, p time.Time) []ReturnDeadline {
	pd, fd := returnDays(c)
	dls := PeriodDeadlines(p, pd, fd, s.Now())
	ext, err := s.Store.ReturnExtensionFor(ctx, c.ID, p.Format("2006-01"))
	if err != nil {
		return dls
	}
	return applyExtension(dls, ext, s.Now())
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

// CompanyDeadlines returns the upcoming return deadlines of a company: last
// month's until its last deadline (including any FBR extension) has passed,
// then this month's.
func (s *Service) CompanyDeadlines(ctx context.Context, c *store.Company) []ReturnDeadline {
	t := dayStart(s.Now())
	prev := time.Date(t.Year(), t.Month()-1, 1, 0, 0, 0, 0, PKT)
	dls := s.periodDeadlines(ctx, c, prev)
	if len(dls) > 0 && t.Format("2006-01-02") <= dls[len(dls)-1].Due {
		return dls
	}
	return s.periodDeadlines(ctx, c, time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, PKT))
}

// SetReturnExtension records the filing date FBR extended for a tax period.
func (s *Service) SetReturnExtension(ctx context.Context, a Actor, companyID int64, in store.ReturnExtension) (*store.ReturnExtension, error) {
	p, err := time.ParseInLocation("2006-01", strings.TrimSpace(in.Period), PKT)
	if err != nil {
		return nil, Invalid("tax period must be YYYY-MM")
	}
	due, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(in.FilingDate), PKT)
	if err != nil {
		return nil, Invalid("the extended filing date must be YYYY-MM-DD")
	}
	if !due.After(p.AddDate(0, 1, -1)) {
		return nil, Invalid("the extended filing date must be after the end of the tax period")
	}
	ref := strings.TrimSpace(in.Reference)
	if ref == "" {
		return nil, Invalid("enter the reference of FBR's extension order or circular (number and date)")
	}
	e := store.ReturnExtension{Period: p.Format("2006-01"), FilingDate: due.Format("2006-01-02"), Reference: truncate(ref, 300)}
	if err := s.Store.SetReturnExtension(ctx, companyID, e, a.UserID); err != nil {
		return nil, err
	}
	s.Audit(ctx, a, companyID, "return.extension", "company", fmt.Sprint(companyID), map[string]any{"period": e.Period, "filingDate": e.FilingDate, "reference": e.Reference})
	return s.Store.ReturnExtensionFor(ctx, companyID, e.Period)
}

// DeleteReturnExtension removes a recorded extension.
func (s *Service) DeleteReturnExtension(ctx context.Context, a Actor, companyID int64, period string) error {
	if err := s.Store.DeleteReturnExtension(ctx, companyID, period); err != nil {
		return err
	}
	s.Audit(ctx, a, companyID, "return.extension_removed", "company", fmt.Sprint(companyID), map[string]any{"period": period})
	return nil
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
		p, _ = time.ParseInLocation("2006-01", s.CompanyDeadlines(ctx, c)[0].Period, PKT)
	} else {
		var err error
		p, err = time.ParseInLocation("2006-01", period, PKT)
		if err != nil {
			return nil, Invalid("tax period must be YYYY-MM")
		}
	}
	from := p.Format("2006-01-02")
	to := p.AddDate(0, 1, -1).Format("2006-01-02")
	r := &PeriodReview{Period: p.Format("2006-01"), PeriodLabel: p.Format("January 2006"), From: from, To: to, Environment: env,
		Deadlines: s.periodDeadlines(ctx, c, p), Counts: map[string]int{}}
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
		{ID: "incidents", OK: unreportedIncidents == 0, Title: "Operational failures reported to FBR (rule 150XA)",
			Detail: plural(unreportedIncidents, "incident in this period is", "incidents in this period are") + " not marked as reported. " +
				"Report operational failures to FBR within 24 hours and record the reference.", Link: "/incidents"},
	}
	if env == domain.EnvSimulator {
		r.Checks = append(r.Checks, PeriodCheck{ID: "simulator", OK: false, Title: "These figures come from the training simulator",
			Detail: "Nothing in the simulator is reported to FBR. Switch the company to production to issue real invoices.", Link: "/settings/fbr"})
	}
	return r, nil
}

// PeriodTie compares a tax period's books with what reached FBR and with
// Annexure-C of the sales tax return, which IRIS builds from the invoices
// FBR accepted. Each side of the triangle is one kind of difference:
// issued but not yet sent (books and FBR), sent but not accepted (FBR and
// Annexure-C) and unissued drafts dated in the period (books and
// Annexure-C). The period is the one whose return is due next.
type PeriodTie struct {
	Period      string          `json:"period"`
	PeriodLabel string          `json:"periodLabel"`
	FilingDue   string          `json:"filingDue"`
	DaysLeft    int             `json:"daysLeft"`
	Books       decimal.Decimal `json:"books"`     // issued documents, excluding sales tax
	AnnexC      decimal.Decimal `json:"annexC"`    // documents accepted by FBR, excluding sales tax
	AnnexCTax   decimal.Decimal `json:"annexCTax"` // sales tax on them
	Accepted    int             `json:"accepted"`
	Issued      int             `json:"issued"`
	Edges       []TieEdge       `json:"edges"`
	ChecksOpen  int             `json:"checksOpen"`
	ChecksTotal int             `json:"checksTotal"`
}

// TieEdge is one side of the period tie; it ties when Count is zero.
type TieEdge struct {
	ID     string          `json:"id"` // books-fbr | fbr-annexc | books-annexc
	Title  string          `json:"title"`
	Detail string          `json:"detail"`
	Count  int             `json:"count"`
	Value  decimal.Decimal `json:"value"`
	Link   string          `json:"link"`
}

// Tie works out the period tie for the tax period whose return is due next.
func (s *Service) Tie(ctx context.Context, c *store.Company, env domain.Environment) (*PeriodTie, error) {
	rev, err := s.ReviewPeriod(ctx, c, env, "")
	if err != nil {
		return nil, err
	}
	totals, err := s.Store.PeriodStatusTotals(ctx, c.ID, env, rev.From, rev.To)
	if err != nil {
		return nil, err
	}
	sum := func(statuses ...domain.InvoiceStatus) (int, decimal.Decimal) {
		n, v := 0, decimal.Zero
		for _, st := range statuses {
			t := totals[string(st)]
			n += t.Count
			v = v.Add(t.Value)
		}
		return n, v
	}
	t := &PeriodTie{Period: rev.Period, PeriodLabel: rev.PeriodLabel}
	for _, d := range rev.Deadlines {
		if d.Kind == "filing" {
			t.FilingDue, t.DaysLeft = d.Due, d.DaysLeft
		}
	}
	acc := totals[string(domain.StatusAccepted)]
	t.AnnexC, t.AnnexCTax, t.Accepted = acc.Value, acc.SalesTax, acc.Count
	t.Issued, t.Books = sum(domain.StatusAccepted, domain.StatusQueued, domain.StatusSubmitting, domain.StatusRejected, domain.StatusUncertain)
	link := func(statuses string) string {
		return "/invoices?status=" + statuses + "&from=" + rev.From + "&to=" + rev.To
	}
	n, v := sum(domain.StatusQueued, domain.StatusSubmitting)
	t.Edges = append(t.Edges, TieEdge{ID: "books-fbr", Title: "Books = FBR", Detail: "Every issued invoice sent to FBR",
		Count: n, Value: v, Link: link("QUEUED,SUBMITTING")})
	n, v = sum(domain.StatusRejected, domain.StatusUncertain)
	t.Edges = append(t.Edges, TieEdge{ID: "fbr-annexc", Title: "FBR = Annexure-C", Detail: "Every submission accepted by FBR",
		Count: n, Value: v, Link: link("REJECTED,UNCERTAIN")})
	n, v = sum(domain.StatusDraft, domain.StatusValidated)
	t.Edges = append(t.Edges, TieEdge{ID: "books-annexc", Title: "Books = Annexure-C", Detail: "No unissued drafts dated in the period",
		Count: n, Value: v, Link: link("DRAFT,VALIDATED")})
	for _, ch := range rev.Checks {
		if ch.ID == "simulator" {
			continue
		}
		t.ChecksTotal++
		if !ch.OK {
			t.ChecksOpen++
		}
	}
	return t, nil
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
			SROScheduleNo: it.SROScheduleNo, SROItemSerialNo: it.SROItemSerialNo,
			FEDType: it.FEDType, FEDRateText: it.FEDRateText, FEDUnitPrice: it.FEDUnitPrice, FEDSRO: it.FEDSRO, FEDSROSerial: it.FEDSROSerial})
	}
	inv, err := s.CreateInvoice(ctx, a, companyID, in)
	if err != nil {
		return nil, err
	}
	s.Audit(ctx, a, companyID, "invoice.duplicate", "invoice", fmt.Sprint(inv.ID), map[string]any{"from": orig.InternalNo, "fromId": orig.ID})
	return inv, nil
}
