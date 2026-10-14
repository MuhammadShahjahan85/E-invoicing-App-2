// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"strings"
	"testing"

	"einvoicing/internal/domain"
	"einvoicing/internal/hscodes"
	"einvoicing/internal/store"
)

func TestBuiltInHSCodesAndFBRList(t *testing.T) {
	f := setup(t)
	n, err := f.svc.Store.CountHSCodes(f.ctx)
	if err != nil || n != len(hscodes.All()) {
		t.Fatalf("stored %d codes, built in %d (%v)", n, len(hscodes.All()), err)
	}
	if src, _, _ := f.svc.Store.HSCodeSource(f.ctx); src != "tariff" {
		t.Fatalf("source %q before FBR's list is downloaded", src)
	}
	// Loading again (as after an update) changes nothing.
	if err := f.svc.EnsureSeedData(f.ctx); err != nil {
		t.Fatal(err)
	}

	// Unknown codes are warned about; known ones are not.
	if w := f.svc.hsCodeWarning(f.ctx, "9999.9999", false); !strings.Contains(w, "not in the Pakistan Customs Tariff") {
		t.Fatalf("unknown code: %q", w)
	}
	if w := f.svc.hsCodeWarning(f.ctx, "7214.9990", false); w != "" {
		t.Fatalf("known code warned before FBR's list exists: %q", w)
	}

	// FBR's list marks its codes, adds the ones the tariff lacks with FBR's
	// description, and keeps the tariff's descriptions.
	err = f.svc.Store.ReplaceHSCodes(f.ctx, []store.HSCode{
		{Code: "0101.2100", Description: "PURE-BRED BREEDING HORSES"},
		{Code: "9999.0001", Description: "FBR ONLY CODE"},
	}, "fbr")
	if err != nil {
		t.Fatal(err)
	}
	if src, n2, _ := f.svc.Store.HSCodeSource(f.ctx); src != "fbr" || n2 != n+1 {
		t.Fatalf("after FBR's list: %s %d", src, n2)
	}
	h, err := f.svc.Store.GetHSCode(f.ctx, "0101.2100")
	if err != nil || !h.InFBR || h.Description != "Pure-bred breeding animals" || h.Context == "" {
		t.Fatalf("0101.2100 after FBR's list: %+v %v", h, err)
	}
	if h, _ := f.svc.Store.GetHSCode(f.ctx, "9999.0001"); h == nil || !h.InFBR || h.Description != "FBR ONLY CODE" {
		t.Fatalf("FBR-only code: %+v", h)
	}
	if w := f.svc.hsCodeWarning(f.ctx, "7214.9990", true); !strings.Contains(w, "not on the HS code list downloaded from FBR") {
		t.Fatalf("code missing from FBR's list: %q", w)
	}
	// A newer tariff in an update keeps the FBR marks.
	if err := f.svc.Store.SetSetting(f.ctx, "hscodes.tariff", "2025-26"); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.EnsureSeedData(f.ctx); err != nil {
		t.Fatal(err)
	}
	if h, _ := f.svc.Store.GetHSCode(f.ctx, "0101.2100"); h == nil || !h.InFBR {
		t.Fatalf("FBR mark lost on reload: %+v", h)
	}
}

func TestReferenceDataRefreshesDaily(t *testing.T) {
	f := setup(t)
	// In the training simulator nothing is downloaded.
	f.svc.refreshReference(f.ctx)
	if last, _ := f.svc.Store.LastReferenceSync(f.ctx, domain.EnvSandbox); last != "" {
		t.Fatalf("downloaded for a simulator company: %s", last)
	}
	sandboxCompany(t, f)
	f.svc.refreshReference(f.ctx)
	last, _ := f.svc.Store.LastReferenceSync(f.ctx, domain.EnvSandbox)
	if last == "" {
		t.Fatal("FBR reference data not downloaded for a sandbox company")
	}
	if src, _, _ := f.svc.Store.HSCodeSource(f.ctx); src != "fbr" {
		t.Fatalf("HS codes not marked from FBR's list: %s", src)
	}
	// A fresh download is not repeated within a day.
	calls := func() int {
		_, n, _ := f.svc.Store.ListFBRCalls(f.ctx, f.cid, 0, 1, 0)
		return n
	}
	before := calls()
	f.svc.refreshReference(f.ctx)
	if calls() != before {
		t.Fatal("reference data downloaded again within a day")
	}
}
