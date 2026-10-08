// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"fmt"
	"strings"

	"einvoicing/internal/domain"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"
)

// ScenarioStatus is a scenario with this company's progress.
type ScenarioStatus struct {
	domain.Scenario
	Suggested bool               `json:"suggested"`
	Assigned  bool               `json:"assigned"`
	Passed    bool               `json:"passed"`
	LastRun   *store.ScenarioRun `json:"lastRun"`
}

// ScenarioOverview lists scenarios and readiness for production.
type ScenarioOverview struct {
	Scenarios          []ScenarioStatus `json:"scenarios"`
	AssignedCount      int              `json:"assignedCount"`
	PassedCount        int              `json:"passedCount"`
	ReadyForProduction bool             `json:"readyForProduction"`
	Suggested          []string         `json:"suggested"`
}

// Scenarios returns the scenario overview for a company.
func (s *Service) Scenarios(ctx context.Context, companyID int64) (*ScenarioOverview, error) {
	c, err := s.Store.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	suggested := domain.ApplicableScenariosMulti(c.BusinessActivities, c.Sectors)
	assigned := c.AssignedScenarios
	if len(assigned) == 0 {
		assigned = suggested
	}
	latest, passed, err := s.Store.LatestScenarioRuns(ctx, companyID)
	if err != nil {
		return nil, err
	}
	ov := &ScenarioOverview{Suggested: suggested}
	for _, sc := range domain.Scenarios {
		st := ScenarioStatus{Scenario: sc, Suggested: contains(suggested, sc.ID), Assigned: contains(assigned, sc.ID), Passed: passed[sc.ID], LastRun: latest[sc.ID]}
		if st.Assigned {
			ov.AssignedCount++
			if st.Passed {
				ov.PassedCount++
			}
		}
		ov.Scenarios = append(ov.Scenarios, st)
	}
	ov.ReadyForProduction = ov.AssignedCount > 0 && ov.PassedCount == ov.AssignedCount
	return ov, nil
}

// ScenarioInvoiceInput builds the invoice input for a scenario.
// mode "engine" computes taxes with the tax engine from the sample's
// quantities and values; mode "sample" reproduces FBR's published sample
// amounts verbatim.
func ScenarioInvoiceInput(sc domain.Scenario, c *store.Company, env domain.Environment, mode, date string) *InvoiceInput {
	it := sc.Item
	value := tax.F(it.ValueSalesExcludingST)
	item := ItemInput{
		HSCode: it.HSCode, Description: it.ProductDescription, UoM: it.UoM, Quantity: tax.F(it.Quantity),
		Value: &value, SaleType: it.SaleType, Rate: it.Rate, SROScheduleNo: it.SROScheduleNo, SROItemSerialNo: it.SROItemSerialNo,
	}
	if item.Description == "" {
		item.Description = sc.Title
	}
	if it.FixedNotifiedValueOrRetailPrice > 0 {
		rv := tax.F(it.FixedNotifiedValueOrRetailPrice)
		item.RetailValue = &rv
	}
	if mode == "sample" {
		st, ft, et, fed, wh, disc := tax.F(it.SalesTaxApplicable), tax.F(it.FurtherTax), tax.F(it.ExtraTax), tax.F(it.FEDPayable), tax.F(it.SalesTaxWithheldAtSource), tax.F(it.Discount)
		item.SalesTax, item.FurtherTax, item.ExtraTax, item.FED, item.STWithheld = &st, &ft, &et, &fed, &wh
		// The engine adds FED to the value of supply; FBR's sample value is
		// reproduced by passing it net of the sample's FED.
		if fed.IsPositive() {
			net := value.Sub(fed)
			item.Value = &net
		}
		item.DiscountAmount = disc
		item.FurtherTaxMode = "no"
	}
	province := c.Province
	return &InvoiceInput{
		Environment: env, DocType: string(domain.DocSaleInvoice), InvoiceDate: date, ScenarioID: sc.ID, Source: "scenario",
		Buyer: &BuyerInput{NTNCNIC: sc.BuyerNTNCNIC, Name: sc.BuyerName, Province: province, Address: c.Address,
			RegistrationType: string(sc.BuyerRegistrationType)},
		Notes: "FBR sandbox scenario " + sc.ID + ": " + sc.Title,
		Items: []ItemInput{item},
	}
}

// RunScenario creates and submits a scenario invoice. env must be sandbox
// (the real certification run) or simulator (practice).
func (s *Service) RunScenario(ctx context.Context, a Actor, companyID int64, scenarioID, mode string, env domain.Environment) (*store.ScenarioRun, *store.Invoice, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, nil, err
	}
	sc, ok := domain.LookupScenario(scenarioID)
	if !ok {
		return nil, nil, Invalid("unknown scenario %q", scenarioID)
	}
	if env == "" {
		env = domain.EnvSandbox
		if !c.HasSandboxToken {
			env = domain.EnvSimulator
		}
	}
	if env == domain.EnvProduction {
		return nil, nil, Invalid("scenarios are run in the sandbox, never in production")
	}
	if mode != "sample" {
		mode = "engine"
	}
	in := ScenarioInvoiceInput(sc, c, env, mode, s.Today())
	if env == domain.EnvSimulator {
		in.ScenarioID = ""
	}
	run := &store.ScenarioRun{CompanyID: companyID, ScenarioID: sc.ID, Mode: mode + "@" + string(env), RunBy: a.UserID}
	inv, err := s.CreateInvoice(ctx, a, companyID, in)
	if err != nil {
		run.Status, run.Message = "error", err.Error()
		_ = s.Store.AddScenarioRun(ctx, run)
		return run, nil, nil
	}
	run.InvoiceID = &inv.ID
	inv, err = s.Submit(ctx, a, companyID, inv.ID, SubmitOptions{SkipFBRValidation: true})
	switch {
	case err != nil:
		run.Status, run.Message = "failed", err.Error()
		inv, _ = s.Store.GetInvoice(ctx, companyID, *run.InvoiceID)
	case inv.Status == domain.StatusAccepted:
		run.Status, run.FBRInvoiceNumber, run.Message = "passed", inv.FBRInvoiceNumber, "Accepted by FBR"
		if env == domain.EnvSimulator {
			run.Status = "practice"
			run.Message = "Accepted by the training simulator (run again in the FBR sandbox to certify)"
		}
	default:
		run.Status = "failed"
		var msgs []string
		for _, e := range inv.FBRErrors {
			msgs = append(msgs, e.String())
		}
		if len(msgs) == 0 {
			msgs = append(msgs, inv.LastError)
		}
		run.Message = strings.Join(msgs, "; ")
	}
	if err := s.Store.AddScenarioRun(ctx, run); err != nil {
		return nil, nil, err
	}
	s.Audit(ctx, a, companyID, "scenario.run", "scenario", sc.ID, map[string]any{"status": run.Status, "env": env, "mode": mode, "message": run.Message})
	return run, inv, nil
}

// SetAssignedScenarios records the scenarios IRIS assigned to the company.
func (s *Service) SetAssignedScenarios(ctx context.Context, a Actor, companyID int64, ids []string) error {
	c, err := s.Store.GetCompany(ctx, companyID)
	if err != nil {
		return err
	}
	var clean []string
	for _, id := range ids {
		if _, ok := domain.LookupScenario(id); !ok {
			return Invalid("unknown scenario %q", id)
		}
		clean = append(clean, strings.ToUpper(id))
	}
	c.AssignedScenarios = clean
	if err := s.Store.UpdateCompany(ctx, c); err != nil {
		return err
	}
	s.Audit(ctx, a, companyID, "scenario.assigned", "company", fmt.Sprint(companyID), clean)
	return nil
}
