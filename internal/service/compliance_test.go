// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"strings"
	"testing"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/tax"
)

func pkt(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 10, 0, 0, 0, PKT) }

func TestReturnDueDateClampsToMonthEnd(t *testing.T) {
	cases := []struct {
		period time.Time
		day    int
		want   string
	}{
		{pkt(2026, time.September, 1), 18, "2026-10-18"},
		{pkt(2026, time.January, 1), 31, "2026-02-28"},  // January return, short February
		{pkt(2028, time.January, 1), 30, "2028-02-29"},  // leap year
		{pkt(2026, time.December, 1), 15, "2027-01-15"}, // across the year end
	}
	for _, c := range cases {
		if got := ReturnDueDate(c.period, c.day).Format("2006-01-02"); got != c.want {
			t.Errorf("ReturnDueDate(%s, %d) = %s, want %s", c.period.Format("2006-01"), c.day, got, c.want)
		}
	}
}

func TestUpcomingDeadlines(t *testing.T) {
	// Before the filing date: last month's return is being prepared.
	d := UpcomingDeadlines(pkt(2026, time.October, 7), 15, 18)
	if len(d) != 2 || d[0].Period != "2026-09" || d[0].Kind != "payment" || d[0].Due != "2026-10-15" || d[0].DaysLeft != 8 ||
		d[1].Kind != "filing" || d[1].Due != "2026-10-18" || d[1].DaysLeft != 11 || d[0].PeriodLabel != "September 2026" {
		t.Fatalf("7 Oct: %+v", d)
	}
	// On the filing date it is still due today.
	d = UpcomingDeadlines(pkt(2026, time.October, 18), 15, 18)
	if d[1].Period != "2026-09" || d[1].DaysLeft != 0 || d[0].DaysLeft != -3 {
		t.Fatalf("18 Oct: %+v", d)
	}
	// After it, the current month's return comes next.
	d = UpcomingDeadlines(pkt(2026, time.October, 19), 15, 18)
	if d[0].Period != "2026-10" || d[0].Due != "2026-11-15" || d[1].Due != "2026-11-18" {
		t.Fatalf("19 Oct: %+v", d)
	}
	// A company whose payment day falls after its filing day: earliest first.
	d = UpcomingDeadlines(pkt(2026, time.October, 7), 21, 18)
	if d[0].Kind != "filing" || d[1].Kind != "payment" || d[1].Due != "2026-10-21" {
		t.Fatalf("custom days: %+v", d)
	}
}

func TestDuplicateInvoiceAndPeriodReview(t *testing.T) {
	f := setup(t)
	cust, prod := f.customer(t), f.product(t)
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{CustomerID: &cust.ID,
		Items: []ItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("40"), DiscountAmount: tax.MustD("100")}}, Submit: true})
	if err != nil || inv.Status != domain.StatusAccepted {
		t.Fatalf("create: %v", err)
	}

	dup, err := f.svc.DuplicateInvoice(f.ctx, f.admin, f.cid, inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dup.ID == inv.ID || dup.Status != domain.StatusDraft || dup.FBRInvoiceNumber != "" || dup.InvoiceDate != f.svc.Today() ||
		dup.BuyerNTNCNIC != inv.BuyerNTNCNIC || len(dup.Items) != 1 || !dup.Totals.SalesTax.Equal(inv.Totals.SalesTax) ||
		dup.InternalNo == inv.InternalNo {
		t.Fatalf("duplicate %+v", dup)
	}
	// Debit notes are not duplicated.
	dn, err := f.svc.NewDebitNoteDraft(f.ctx, f.admin, f.cid, inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DuplicateInvoice(f.ctx, f.admin, f.cid, dn.ID); err == nil {
		t.Fatal("duplicating a debit note must be refused")
	}

	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	rev, err := f.svc.ReviewPeriod(f.ctx, c, "", time.Now().In(PKT).Format("2006-01"))
	if err != nil {
		t.Fatal(err)
	}
	if rev.Summary.SaleInvoices != 1 || !rev.Summary.SalesTax.Equal(inv.Totals.SalesTax) || len(rev.BySaleType) == 0 ||
		len(rev.TopItems) != 1 || rev.TopItems[0].HSCode != "3104.2000" || len(rev.TopBuyers) != 1 {
		t.Fatalf("review %+v", rev)
	}
	checks := map[string]PeriodCheck{}
	for _, ch := range rev.Checks {
		checks[ch.ID] = ch
	}
	// Two drafts are dated in the period (the duplicate and the debit note).
	if checks["drafts"].OK || !strings.HasPrefix(checks["drafts"].Detail, "2 drafts") || !checks["unreported"].OK || !checks["rejected"].OK {
		t.Fatalf("checks %+v", rev.Checks)
	}
	if _, ok := checks["simulator"]; !ok {
		t.Fatal("simulator figures must be flagged")
	}
	if _, err := f.svc.ReviewPeriod(f.ctx, c, "", "2026-13"); err == nil {
		t.Fatal("invalid period must be refused")
	}
}
