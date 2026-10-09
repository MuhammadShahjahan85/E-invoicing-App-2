// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
)

func callLog(op string, kind fbr.ErrKind, err error) fbr.CallLog {
	return fbr.CallLog{Operation: op, ErrKind: kind, Err: err}
}

// Successful validations or lookups must not hide an outage of posting, and
// must not release the whole queue on every tick.
func TestHealthPostingOutageNotMaskedByOtherCalls(t *testing.T) {
	h := newHealthTracker()
	down := errors.New("503")
	for i := 0; i < 3; i++ {
		h.observe(1, domain.EnvProduction, callLog("validateinvoicedata", "", nil))
		h.observe(1, domain.EnvProduction, callLog("postinvoicedata", fbr.ErrUnavailable, down))
	}
	st := h.get(1, domain.EnvProduction)
	if st.Failures != 3 || !st.PostFailing || st.Recovered || st.Probe {
		t.Fatalf("validate successes masked the posting outage: %+v", *st)
	}
	// A connection test answered by FBR asks for one probe, not a full release.
	h.observe(1, domain.EnvProduction, callLog("provinces", "", nil))
	if st.Failures != 3 || !st.Probe || st.Recovered {
		t.Fatalf("lookup during a posting outage: %+v", *st)
	}
	// A successful post is a real recovery.
	h.observe(1, domain.EnvProduction, callLog("postinvoicedata", "", nil))
	if st.Failures != 0 || st.PostFailing || !st.Recovered {
		t.Fatalf("post success should recover: %+v", *st)
	}
	// Our own cancelled requests are not FBR failures.
	h.observe(1, domain.EnvProduction, callLog("postinvoicedata", fbr.ErrUncertain, fmt.Errorf("post: %w", context.Canceled)))
	if st.Failures != 0 {
		t.Fatalf("cancelled request counted as failure: %+v", *st)
	}
}

// Only production drives rule 150XA incidents: a healthy sandbox must not
// close the production outage incident, and an isolated failure opens none.
func TestOutageIncidentKeyedToProduction(t *testing.T) {
	f := setup(t)
	old := OutageThreshold
	OutageThreshold = 0
	defer func() { OutageThreshold = old }()
	down := errors.New("connection refused")

	// One isolated failure: no incident.
	f.svc.health.observe(f.cid, domain.EnvProduction, callLog("postinvoicedata", fbr.ErrNotSent, down))
	f.svc.checkIncidents(f.ctx)
	if n := incidentsOfKind(t, f, "fbr_unreachable"); n != 0 {
		t.Fatalf("single failure opened %d incidents", n)
	}

	for i := 0; i < 3; i++ {
		f.svc.health.observe(f.cid, domain.EnvProduction, callLog("postinvoicedata", fbr.ErrNotSent, down))
	}
	f.svc.health.observe(f.cid, domain.EnvSandbox, callLog("postinvoicedata", "", nil))
	for i := 0; i < 5; i++ {
		f.svc.checkIncidents(f.ctx)
	}
	list, _ := f.svc.Store.ListIncidents(f.ctx, f.cid, 0)
	open := 0
	for _, i := range list {
		if i.Kind == "fbr_unreachable" && i.EndedAt == "" {
			open++
		}
	}
	if len(list) != 1 || open != 1 {
		t.Fatalf("expected one open production incident, got %d incidents (%d open)", len(list), open)
	}
	// Recovery of production closes it.
	f.svc.health.observe(f.cid, domain.EnvProduction, callLog("postinvoicedata", "", nil))
	f.svc.checkIncidents(f.ctx)
	if got, _ := f.svc.Store.OpenAutoIncident(f.ctx, f.cid, "fbr_unreachable"); got != nil {
		t.Fatal("incident should close when production recovers")
	}
}
