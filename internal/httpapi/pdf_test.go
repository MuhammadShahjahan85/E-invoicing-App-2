// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPDFDownloads(t *testing.T) {
	e := newEnv(t)
	base := fmt.Sprintf("/companies/%d", e.compA)
	st, b := e.adminA.do("POST", base+"/invoices", sampleInvoice(map[string]any{"submit": true}))
	if st != 201 {
		t.Fatalf("invoice: %d %s", st, b)
	}
	var inv struct {
		ID         int64  `json:"id"`
		InternalNo string `json:"internalNo"`
		Status     string `json:"status"`
	}
	_ = json.Unmarshal(b, &inv)

	isPDF := func(name, path string) {
		t.Helper()
		resp, body := e.adminA.get(path)
		if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "application/pdf" || !bytes.HasPrefix(body, []byte("%PDF-")) {
			t.Fatalf("%s: %d %s %.80q", name, resp.StatusCode, resp.Header.Get("Content-Type"), body)
		}
		if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, `.pdf"`) {
			t.Fatalf("%s: content disposition %q", name, cd)
		}
	}
	isPDF("invoice", fmt.Sprintf("%s/invoices/%d/pdf", base, inv.ID))
	today := time.Now().Format("2006-01-02")
	for _, kind := range []string{"register", "tax-summary", "monthly", "annex-c", "customers"} {
		isPDF(kind, fmt.Sprintf("%s/reports/%s?format=pdf&from=%s&to=%s", base, kind, today[:8]+"01", today))
	}
	isPDF("report pack", base+"/compliance/pack?period="+today[:7])
	isPDF("report pack (next return)", base+"/compliance/pack")
	if st, _ := e.adminA.do("GET", base+"/compliance/pack?period=2026-13", nil); st != 422 {
		t.Fatalf("invalid period: got %d, want 422", st)
	}

	// The PDF of an accepted invoice counts as a printed copy.
	if inv.Status != "ACCEPTED" {
		t.Fatalf("invoice status %s, want ACCEPTED", inv.Status)
	}
	{
		_, b = e.adminA.do("GET", fmt.Sprintf("%s/invoices/%d", base, inv.ID), nil)
		var after struct {
			Invoice struct {
				PrintCount int `json:"printCount"`
			} `json:"invoice"`
		}
		_ = json.Unmarshal(b, &after)
		if after.Invoice.PrintCount != 1 {
			t.Fatalf("print count %d after a PDF download, want 1", after.Invoice.PrintCount)
		}
	}
	// Report packs are report data: operators cannot download them.
	op := e.login("operatorA", "Initial123")
	if st, _ := op.do("GET", base+"/compliance/pack", nil); st != 403 {
		t.Fatalf("operator report pack: got %d, want 403", st)
	}
}
