// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/store"
)

// healthTracker watches FBR exchanges to detect outages. Rule 150R of the
// Sales Tax Rules, 2006 obliges the integrated person to report any
// operational failure or disruption of the e-invoicing system to the Board
// and the Commissioner within twenty-four hours; detected outages are logged
// as incidents so the operator can generate that report.
type healthTracker struct {
	mu    sync.Mutex
	state map[string]*connState
}

type connState struct {
	CompanyID    int64
	Env          domain.Environment
	LastSuccess  time.Time
	FirstFailure time.Time
	Failures     int
	LastError    string
	AuthFailure  bool
	// Recovered is set when FBR is usable again after failures; the worker
	// then resubmits the queued invoices at once.
	Recovered bool
	// PostFailing is set while posting invoices fails: only a successful
	// post (not a reference lookup or validation) clears it.
	PostFailing bool
	// Probe is set when FBR answered another call (e.g. a connection test)
	// while posting was failing: one queued invoice is then retried at once
	// to find out whether posting works again.
	Probe bool
}

// ConnectionStatus is reported in the dashboard.
type ConnectionStatus struct {
	Environment  domain.Environment `json:"environment"`
	Healthy      bool               `json:"healthy"`
	LastSuccess  string             `json:"lastSuccess"`
	FailingSince string             `json:"failingSince"`
	Failures     int                `json:"failures"`
	LastError    string             `json:"lastError"`
	AuthFailure  bool               `json:"authFailure"`
}

func newHealthTracker() *healthTracker { return &healthTracker{state: map[string]*connState{}} }

func key(companyID int64, env domain.Environment) string { return fmt.Sprintf("%d/%s", companyID, env) }

func (h *healthTracker) get(companyID int64, env domain.Environment) *connState {
	k := key(companyID, env)
	st, ok := h.state[k]
	if !ok {
		st = &connState{CompanyID: companyID, Env: env}
		h.state[k] = st
	}
	return st
}

func (h *healthTracker) observe(companyID int64, env domain.Environment, l fbr.CallLog) {
	if env == domain.EnvSimulator {
		return
	}
	// A call abandoned by this side (request cancelled) says nothing about FBR.
	if l.Err != nil && errors.Is(l.Err, context.Canceled) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.get(companyID, env)
	posting := l.Operation == "postinvoicedata"
	fail := func(auth bool) {
		if st.Failures == 0 {
			st.FirstFailure = time.Now()
		}
		st.Failures++
		st.AuthFailure = auth
		if posting {
			st.PostFailing = true
		}
		if l.Err != nil {
			st.LastError = l.Err.Error()
		}
	}
	switch l.ErrKind {
	case fbr.ErrNotSent, fbr.ErrUncertain, fbr.ErrUnavailable:
		fail(false)
	case fbr.ErrAuth:
		fail(true)
	default:
		st.LastSuccess = time.Now()
		// While posting fails, a successful lookup or validation does not
		// mean invoices can be reported again; a lookup triggers one probe.
		if st.PostFailing && !posting {
			if l.Operation != "validateinvoicedata" {
				st.Probe = true
			}
			return
		}
		if st.Failures > 0 {
			st.Recovered = true
		}
		st.Failures = 0
		st.FirstFailure = time.Time{}
		st.AuthFailure = false
		st.PostFailing = false
		st.LastError = ""
	}
}

// takeRecovered returns the connections that recovered (or should be
// probed) since the last call and clears their flags.
func (h *healthTracker) takeRecovered() []connState {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []connState
	for _, st := range h.state {
		if st.Recovered || st.Probe {
			out = append(out, *st)
			st.Recovered, st.Probe = false, false
		}
	}
	return out
}

func (h *healthTracker) snapshot() []connState {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]connState, 0, len(h.state))
	for _, st := range h.state {
		out = append(out, *st)
	}
	return out
}

// Connection returns the connectivity status for a company and environment.
func (s *Service) Connection(companyID int64, env domain.Environment) ConnectionStatus {
	s.health.mu.Lock()
	defer s.health.mu.Unlock()
	st, ok := s.health.state[key(companyID, env)]
	if !ok {
		st = &connState{CompanyID: companyID, Env: env} // nothing observed yet; not stored
	}
	cs := ConnectionStatus{Environment: env, Healthy: st.Failures == 0, Failures: st.Failures, LastError: st.LastError, AuthFailure: st.AuthFailure}
	if !st.LastSuccess.IsZero() {
		cs.LastSuccess = st.LastSuccess.UTC().Format(time.RFC3339)
	}
	if !st.FirstFailure.IsZero() {
		cs.FailingSince = st.FirstFailure.UTC().Format(time.RFC3339)
	}
	return cs
}

// OutageThreshold is how long FBR must be unreachable before an incident is
// opened; at least OutageMinFailures failed calls are also required, so one
// isolated failure does not create an incident.
var (
	OutageThreshold   = 10 * time.Minute
	OutageMinFailures = 3
)

// requeueRecovered makes the queued invoices of every connection that has just
// recovered due at once, so invoices issued during an outage reach FBR well
// within the 24 hours allowed after connectivity is restored.
func (s *Service) requeueRecovered(ctx context.Context) {
	for _, st := range s.health.takeRecovered() {
		if !st.Recovered {
			_, _ = s.Store.RequeueOldest(ctx, st.CompanyID, st.Env)
			continue
		}
		if n, err := s.Store.RequeueNow(ctx, st.CompanyID, st.Env); err == nil && n > 0 {
			s.Log.Info("FBR connection restored; resubmitting queued invoices", "company", st.CompanyID, "env", st.Env, "count", n)
			s.Audit(ctx, System, st.CompanyID, "worker.requeued", "invoice", "", fmt.Sprintf("%d invoices queued during the outage resubmitted after the connection was restored", n))
		}
	}
}

// checkIncidents opens or closes auto-detected incidents from the health
// state. Only production matters for Rule 150R: sandbox and simulator states
// neither open nor close the taxpayer's incidents.
func (s *Service) checkIncidents(ctx context.Context) {
	for _, st := range s.health.snapshot() {
		if st.Env != domain.EnvProduction {
			continue
		}
		kind := "fbr_unreachable"
		if st.AuthFailure {
			kind = "auth_failure"
		}
		open, _ := s.Store.OpenAutoIncident(ctx, st.CompanyID, kind)
		failing := st.Failures >= OutageMinFailures && !st.FirstFailure.IsZero() && time.Since(st.FirstFailure) >= OutageThreshold
		if st.AuthFailure && st.Failures > 0 {
			failing = true
		}
		switch {
		case failing && open == nil:
			desc := fmt.Sprintf("FBR Digital Invoicing (%s) unreachable since %s. Last error: %s. Invoices are queued and will be submitted automatically when the connection is restored.",
				st.Env.Label(), st.FirstFailure.In(PKT).Format("02-Jan-2006 15:04"), st.LastError)
			if st.AuthFailure {
				desc = fmt.Sprintf("FBR rejected the security token for %s (HTTP 401/403) since %s. Check the token on IRIS and the whitelisted IP address. Last error: %s",
					st.Env.Label(), st.FirstFailure.In(PKT).Format("02-Jan-2006 15:04"), st.LastError)
			}
			inc := &store.Incident{CompanyID: st.CompanyID, Kind: kind, Description: desc, StartedAt: st.FirstFailure.UTC().Format(time.RFC3339), AutoDetected: true}
			if err := s.Store.SaveIncident(ctx, inc); err == nil {
				s.Audit(ctx, System, st.CompanyID, "incident.opened", "incident", fmt.Sprint(inc.ID), desc)
			}
		case st.Failures == 0 && open == nil:
			// also close any open outage incident once healthy
			for _, k := range []string{"fbr_unreachable", "auth_failure"} {
				if inc, err := s.Store.OpenAutoIncident(ctx, st.CompanyID, k); err == nil && inc != nil {
					inc.EndedAt = time.Now().UTC().Format(time.RFC3339)
					if err := s.Store.SaveIncident(ctx, inc); err == nil {
						s.Audit(ctx, System, st.CompanyID, "incident.closed", "incident", fmt.Sprint(inc.ID), "connection restored")
					}
				}
			}
		case st.Failures == 0 && open != nil:
			open.EndedAt = time.Now().UTC().Format(time.RFC3339)
			if err := s.Store.SaveIncident(ctx, open); err == nil {
				s.Audit(ctx, System, st.CompanyID, "incident.closed", "incident", fmt.Sprint(open.ID), "connection restored")
			}
		}
	}
}
