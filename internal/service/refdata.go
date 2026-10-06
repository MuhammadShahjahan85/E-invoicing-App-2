package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/fbrmock"
	"einvoicing/internal/store"
)

// Reference data is cached per environment under keys "<env>:<kind>".
// Before the first sync the built-in seed lists are used, so the product
// works offline and in the training simulator.

func refKey(env domain.Environment, kind string) string { return string(env) + ":" + kind }

// EnsureSeedData loads the seed HS code list on first run.
func (s *Service) EnsureSeedData(ctx context.Context) error {
	n, err := s.Store.CountHSCodes(ctx)
	if err != nil || n > 0 {
		return err
	}
	var codes []store.HSCode
	for _, h := range fbrmock.SeedHSCodes() {
		codes = append(codes, store.HSCode{Code: h.Code, Description: h.Description})
	}
	return s.Store.ReplaceHSCodes(ctx, codes, "seed")
}

// cachedList returns the set of descriptions of a cached reference list.
func (s *Service) cachedList(ctx context.Context, companyID int64, env domain.Environment, kind string) map[string]bool {
	e, err := s.Store.GetRef(ctx, refKey(env, kind))
	if err != nil {
		return nil
	}
	out := map[string]bool{}
	switch kind {
	case "uom":
		var l []fbr.UOMRef
		_ = json.Unmarshal([]byte(e.Data), &l)
		for _, x := range l {
			out[x.Description] = true
		}
	case "transtypecode":
		var l []fbr.TransTypeRef
		_ = json.Unmarshal([]byte(e.Data), &l)
		for _, x := range l {
			out[x.Description] = true
		}
	case "provinces":
		var l []fbr.ProvinceRef
		_ = json.Unmarshal([]byte(e.Data), &l)
		for _, x := range l {
			out[strings.ToUpper(x.Name)] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// SyncReport summarises a reference data sync.
type SyncReport struct {
	Environment domain.Environment `json:"environment"`
	Counts      map[string]int     `json:"counts"`
	Errors      map[string]string  `json:"errors"`
	At          string             `json:"at"`
}

// SyncReference downloads FBR's reference lists for an environment.
func (s *Service) SyncReference(ctx context.Context, a Actor, companyID int64, env domain.Environment) (*SyncReport, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if env == "" {
		env = c.Environment
	}
	cl, err := s.Client(ctx, c, env, nil)
	if err != nil {
		return nil, err
	}
	rep := &SyncReport{Environment: env, Counts: map[string]int{}, Errors: map[string]string{}, At: s.Now().UTC().Format(time.RFC3339)}
	put := func(kind string, v any, n int, err error) {
		if err != nil {
			rep.Errors[kind] = err.Error()
			return
		}
		b, _ := json.Marshal(v)
		if err := s.Store.PutRef(ctx, refKey(env, kind), kind, string(b), "fbr"); err != nil {
			rep.Errors[kind] = err.Error()
			return
		}
		rep.Counts[kind] = n
	}
	prov, err := cl.Provinces(ctx)
	put("provinces", prov, len(prov), err)
	dt, err := cl.DocTypes(ctx)
	put("doctypecode", dt, len(dt), err)
	tt, err := cl.TransTypes(ctx)
	put("transtypecode", tt, len(tt), err)
	uoms, err := cl.UOMs(ctx)
	put("uom", uoms, len(uoms), err)
	sro, err := cl.SROItemCodes(ctx)
	put("sroitemcode", sro, len(sro), err)
	hs, err := cl.ItemDescCodes(ctx)
	if err != nil {
		rep.Errors["itemdesccode"] = err.Error()
	} else {
		codes := make([]store.HSCode, 0, len(hs))
		for _, h := range hs {
			codes = append(codes, store.HSCode{Code: h.Code, Description: h.Description})
		}
		source := "fbr"
		if env == domain.EnvSimulator {
			source = "seed"
		}
		if err := s.Store.ReplaceHSCodes(ctx, codes, source); err != nil {
			rep.Errors["itemdesccode"] = err.Error()
		} else {
			rep.Counts["itemdesccode"] = len(codes)
		}
	}
	// Fill the company's province code from the synced list.
	if c.ProvinceCode == 0 {
		for _, p := range prov {
			if strings.EqualFold(p.Name, c.Province) {
				c.ProvinceCode = p.Code
				_ = s.Store.UpdateCompany(ctx, c)
			}
		}
	}
	s.Audit(ctx, a, companyID, "refdata.sync", "company", fmt.Sprint(companyID), rep)
	return rep, nil
}

// Provinces returns the province list (synced or seed).
func (s *Service) Provinces(ctx context.Context, env domain.Environment) []fbr.ProvinceRef {
	if e, err := s.Store.GetRef(ctx, refKey(env, "provinces")); err == nil {
		var l []fbr.ProvinceRef
		if json.Unmarshal([]byte(e.Data), &l) == nil && len(l) > 0 {
			return l
		}
	}
	out := make([]fbr.ProvinceRef, 0, len(domain.Provinces))
	for _, p := range domain.Provinces {
		out = append(out, fbr.ProvinceRef{Code: p.Code, Name: p.Name})
	}
	return out
}

// SaleTypeInfo combines FBR's sale type list with the built-in behaviour.
type SaleTypeInfo struct {
	fbr.TransTypeRef
	Known bool            `json:"known"`
	Info  domain.SaleType `json:"info"`
}

// SaleTypes returns the sale type list (synced or seed).
func (s *Service) SaleTypes(ctx context.Context, env domain.Environment) []SaleTypeInfo {
	var l []fbr.TransTypeRef
	if e, err := s.Store.GetRef(ctx, refKey(env, "transtypecode")); err == nil {
		_ = json.Unmarshal([]byte(e.Data), &l)
	}
	if len(l) == 0 {
		for _, st := range domain.SaleTypes {
			l = append(l, fbr.TransTypeRef{Description: st.Name})
		}
	}
	out := make([]SaleTypeInfo, 0, len(l))
	for _, t := range l {
		st, ok := domain.LookupSaleType(t.Description)
		out = append(out, SaleTypeInfo{TransTypeRef: t, Known: ok, Info: st})
	}
	return out
}

// UOMs returns the unit of measure list (synced or seed).
func (s *Service) UOMs(ctx context.Context, env domain.Environment) []string {
	if e, err := s.Store.GetRef(ctx, refKey(env, "uom")); err == nil {
		var l []fbr.UOMRef
		if json.Unmarshal([]byte(e.Data), &l) == nil && len(l) > 0 {
			out := make([]string, 0, len(l))
			for _, u := range l {
				out = append(out, u.Description)
			}
			sort.Strings(out)
			return out
		}
	}
	return append([]string(nil), domain.UOMs...)
}

func (s *Service) provinceCode(ctx context.Context, c *store.Company, env domain.Environment) int {
	if c.ProvinceCode > 0 {
		return c.ProvinceCode
	}
	for _, p := range s.Provinces(ctx, env) {
		if strings.EqualFold(p.Name, c.Province) {
			return p.Code
		}
	}
	return 0
}

func (s *Service) transTypeID(ctx context.Context, env domain.Environment, saleType string) int {
	for _, st := range s.SaleTypes(ctx, env) {
		if st.Description == saleType && st.ID > 0 {
			return st.ID
		}
	}
	return 0
}

// cachedCall memoises a live reference lookup for a day.
func (s *Service) cachedCall(ctx context.Context, key, kind string, fetch func() (any, error), out any) error {
	if e, err := s.Store.GetRef(ctx, key); err == nil && time.Since(store.ParseTime(e.FetchedAt)) < 24*time.Hour {
		if json.Unmarshal([]byte(e.Data), out) == nil {
			return nil
		}
	}
	v, err := fetch()
	if err != nil {
		if e, err2 := s.Store.GetRef(ctx, key); err2 == nil { // stale cache beats nothing
			return json.Unmarshal([]byte(e.Data), out)
		}
		return err
	}
	b, _ := json.Marshal(v)
	_ = s.Store.PutRef(ctx, key, kind, string(b), "fbr")
	return json.Unmarshal(b, out)
}

// Rates returns the valid rate descriptions for a sale type on a date
// (SaleTypeToRate). Falls back to the sale type's default rate.
func (s *Service) Rates(ctx context.Context, companyID int64, env domain.Environment, saleType, date string) ([]fbr.RateRef, error) {
	c, err := s.Store.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	d := parseDate(date, s.Now())
	fallback := func() []fbr.RateRef {
		if st, ok := domain.LookupSaleType(saleType); ok {
			return []fbr.RateRef{{Description: st.DefaultRate}}
		}
		return nil
	}
	ttID := s.transTypeID(ctx, env, saleType)
	if ttID == 0 && env == domain.EnvSimulator {
		// Simulator ids are positional in the built-in catalogue.
		for i, st := range domain.SaleTypes {
			if st.Name == saleType {
				ttID = 1001 + i
			}
		}
	}
	if ttID == 0 {
		return fallback(), nil
	}
	prov := s.provinceCode(ctx, c, env)
	var out []fbr.RateRef
	key := fmt.Sprintf("%s:rates:%d:%d:%s", env, ttID, prov, d.Format("2006-01-02"))
	err = s.cachedCall(ctx, key, "rates", func() (any, error) {
		cl, err := s.Client(ctx, c, env, nil)
		if err != nil {
			return nil, err
		}
		return cl.SaleTypeToRate(ctx, d, ttID, prov)
	}, &out)
	if err != nil || len(out) == 0 {
		return fallback(), nil
	}
	return out, nil
}

// SROSchedules returns SRO/schedule references for a rate id.
func (s *Service) SROSchedules(ctx context.Context, companyID int64, env domain.Environment, rateID int, date string) ([]fbr.SROScheduleRef, error) {
	c, err := s.Store.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	d := parseDate(date, s.Now())
	prov := s.provinceCode(ctx, c, env)
	var out []fbr.SROScheduleRef
	key := fmt.Sprintf("%s:sroschedule:%d:%d:%s", env, rateID, prov, d.Format("2006-01-02"))
	err = s.cachedCall(ctx, key, "sroschedule", func() (any, error) {
		cl, err := s.Client(ctx, c, env, nil)
		if err != nil {
			return nil, err
		}
		return cl.SROSchedule(ctx, rateID, d, prov)
	}, &out)
	return out, err
}

// SROItems returns the serial numbers of an SRO.
func (s *Service) SROItems(ctx context.Context, companyID int64, env domain.Environment, sroID int, date string) ([]fbr.SROItemRef, error) {
	c, err := s.Store.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	d := parseDate(date, s.Now())
	var out []fbr.SROItemRef
	key := fmt.Sprintf("%s:sroitem:%d:%s", env, sroID, d.Format("2006-01-02"))
	err = s.cachedCall(ctx, key, "sroitem", func() (any, error) {
		cl, err := s.Client(ctx, c, env, nil)
		if err != nil {
			return nil, err
		}
		return cl.SROItems(ctx, d, sroID)
	}, &out)
	return out, err
}

// HSUOM returns the UoM FBR prescribes for an HS code.
func (s *Service) HSUOM(ctx context.Context, companyID int64, env domain.Environment, hsCode string) ([]fbr.UOMRef, error) {
	c, err := s.Store.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	var out []fbr.UOMRef
	key := fmt.Sprintf("%s:hsuom:%s", env, hsCode)
	err = s.cachedCall(ctx, key, "hsuom", func() (any, error) {
		cl, err := s.Client(ctx, c, env, nil)
		if err != nil {
			return nil, err
		}
		return cl.HSUOM(ctx, hsCode, 3)
	}, &out)
	return out, err
}

// BuyerStatus is the result of checking a buyer with FBR.
type BuyerStatus struct {
	RegNo            string `json:"regNo"`
	STATLActive      bool   `json:"statlActive"`
	STATLStatus      string `json:"statlStatus"`
	RegistrationType string `json:"registrationType"`
	Registered       bool   `json:"registered"`
	CheckedAt        string `json:"checkedAt"`
	Error            string `json:"error,omitempty"`
}

// CheckBuyer queries STATL and Get_Reg_Type for a registration number and,
// when customerID is given, stores the result on the customer.
func (s *Service) CheckBuyer(ctx context.Context, a Actor, companyID int64, env domain.Environment, regNo string, customerID int64) (*BuyerStatus, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if env == "" {
		env = c.Environment
	}
	var cust *store.Customer
	if customerID > 0 {
		cust, err = s.Store.GetCustomer(ctx, companyID, customerID)
		if err != nil {
			return nil, err
		}
		if regNo == "" {
			regNo = cust.NTNCNIC
		}
	}
	if regNo == "" {
		return nil, Invalid("an NTN or CNIC is required")
	}
	cl, err := s.Client(ctx, c, env, nil)
	if err != nil {
		return nil, err
	}
	bs := &BuyerStatus{RegNo: regNo, CheckedAt: s.Now().UTC().Format(time.RFC3339)}
	st, err := cl.STATL(ctx, regNo, s.Now().In(PKT))
	if err != nil {
		bs.Error = "STATL: " + err.Error()
	} else {
		bs.STATLActive, bs.STATLStatus = st.Active, st.Status
	}
	rt, err := cl.RegType(ctx, regNo)
	if err != nil {
		if bs.Error != "" {
			bs.Error += "; "
		}
		bs.Error += "Get_Reg_Type: " + err.Error()
	} else {
		bs.RegistrationType, bs.Registered = rt.RegistrationType, rt.Registered
	}
	if cust != nil && bs.Error == "" {
		cust.StatlStatus = bs.STATLStatus
		cust.FBRRegType = bs.RegistrationType
		cust.StatusCheckedAt = bs.CheckedAt
		if bs.Registered && bs.STATLActive {
			cust.RegistrationType = domain.Registered
		} else if rt.RegistrationType != "" {
			cust.RegistrationType = domain.Unregistered
		}
		if err := s.Store.SaveCustomer(ctx, cust); err != nil {
			return nil, err
		}
		s.Audit(ctx, a, companyID, "customer.fbr_check", "customer", fmt.Sprint(cust.ID), bs)
	}
	return bs, nil
}

func parseDate(s string, now time.Time) time.Time {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return now.In(PKT)
}
