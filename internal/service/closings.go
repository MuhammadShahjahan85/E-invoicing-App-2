// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/store"
)

// Rule 150R(4)(f) of the Sales Tax Rules, 2006: the electronic invoicing
// system "must perform closing on close of the day, week and month". The
// worker records a closing for every company when a day, an ISO week
// (Monday to Sunday) or a month ends; a closing is a hash-chained snapshot
// of the period's invoices that cannot be changed afterwards.

// Closing kinds.
const (
	ClosingDay   = "day"
	ClosingWeek  = "week"
	ClosingMonth = "month"
)

// Catch-up limits when the server was off for a while.
var closingCatchUp = map[string]int{ClosingDay: 62, ClosingWeek: 12, ClosingMonth: 12}

type closingPeriod struct{ key, start, end string }

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// periodOf returns the closing period of a kind that contains d.
func periodOf(kind string, d time.Time) closingPeriod {
	d = dateOnly(d)
	switch kind {
	case ClosingWeek:
		wd := int(d.Weekday())
		if wd == 0 {
			wd = 7
		}
		mon := d.AddDate(0, 0, 1-wd)
		y, w := d.ISOWeek()
		return closingPeriod{fmt.Sprintf("%d-W%02d", y, w), mon.Format("2006-01-02"), mon.AddDate(0, 0, 6).Format("2006-01-02")}
	case ClosingMonth:
		first := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
		return closingPeriod{first.Format("2006-01"), first.Format("2006-01-02"), first.AddDate(0, 1, -1).Format("2006-01-02")}
	default:
		s := d.Format("2006-01-02")
		return closingPeriod{s, s, s}
	}
}

// lastEnded returns the latest period of a kind that has ended before today:
// yesterday, last week or last month.
func lastEnded(kind string, today time.Time) closingPeriod {
	start, _ := time.Parse("2006-01-02", periodOf(kind, today).start)
	return periodOf(kind, start.AddDate(0, 0, -1))
}

// duePeriods returns the periods of a kind that have ended and are not yet
// closed, oldest first. Without an earlier closing only the latest ended
// period is due.
func duePeriods(kind string, today time.Time, last *store.Closing) []closingPeriod {
	latest := lastEnded(kind, today)
	if last == nil {
		return []closingPeriod{latest}
	}
	end, err := time.Parse("2006-01-02", last.PeriodEnd)
	if err != nil {
		return []closingPeriod{latest}
	}
	var out []closingPeriod
	for p := periodOf(kind, end.AddDate(0, 0, 1)); p.end <= latest.end && len(out) < closingCatchUp[kind]; {
		out = append(out, p)
		next, _ := time.Parse("2006-01-02", p.end)
		p = periodOf(kind, next.AddDate(0, 0, 1))
	}
	return out
}

// runClosings records due closings at most once an hour (worker).
func (s *Service) runClosings(ctx context.Context) {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if !s.lastClosingRun.IsZero() && s.Now().Sub(s.lastClosingRun) < time.Hour {
		return
	}
	s.lastClosingRun = s.Now()
	if n, err := s.CloseDuePeriods(ctx); err != nil {
		s.Log.Error("closings failed", "err", err)
	} else if n > 0 {
		s.Log.Info("closings recorded", "count", n)
	}
}

// CloseDuePeriods records every day, week and month closing that is due for
// the active companies (in each company's working environment).
func (s *Service) CloseDuePeriods(ctx context.Context) (int, error) {
	cs, err := s.Store.ListCompanies(ctx)
	if err != nil {
		return 0, err
	}
	today := s.Now().In(PKT)
	n := 0
	for _, c := range cs {
		if !c.Active {
			continue
		}
		for _, kind := range []string{ClosingDay, ClosingWeek, ClosingMonth} {
			last, err := s.Store.LastClosing(ctx, c.ID, c.Environment, kind)
			if err != nil {
				return n, err
			}
			for _, p := range duePeriods(kind, today, last) {
				sum, err := s.Store.ClosingSummaryFor(ctx, c.ID, c.Environment, p.start, p.end)
				if err != nil {
					return n, err
				}
				cl := &store.Closing{CompanyID: c.ID, Environment: c.Environment, Kind: kind, PeriodKey: p.key, PeriodStart: p.start, PeriodEnd: p.end, Summary: sum}
				if err := s.Store.InsertClosing(ctx, cl); err != nil {
					if errors.Is(err, store.ErrConflict) {
						continue
					}
					return n, err
				}
				n++
				s.Audit(ctx, System, c.ID, "closing."+kind, "closing", fmt.Sprint(cl.ID), map[string]any{"period": p.key, "env": c.Environment,
					"documents": sum.Documents, "reported": sum.Reported, "pending": sum.Pending + sum.Unreconciled, "hash": cl.Hash})
			}
		}
	}
	return n, nil
}

// ClosingList is the closings of a company with the state of their chain.
type ClosingList struct {
	Closings []*store.Closing `json:"closings"`
	Checked  int              `json:"checked"`
	ChainOK  bool             `json:"chainOk"`
	BrokenAt int64            `json:"brokenAt,omitempty"`
}

// Closings lists a company's closings and verifies their hash chain.
func (s *Service) Closings(ctx context.Context, c *store.Company, env domain.Environment, kind string, limit int) (*ClosingList, error) {
	list, err := s.Store.ListClosings(ctx, c.ID, env, kind, limit)
	if err != nil {
		return nil, err
	}
	broken, checked, err := s.Store.VerifyClosings(ctx, c.ID, env)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []*store.Closing{}
	}
	return &ClosingList{Closings: list, Checked: checked, ChainOK: broken == 0, BrokenAt: broken}, nil
}
