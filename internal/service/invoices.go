// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/security"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"
	"einvoicing/internal/validate"

	"github.com/shopspring/decimal"
)

// BuyerInput carries buyer particulars when no customer master is used.
type BuyerInput struct {
	NTNCNIC          string `json:"ntnCnic"`
	Name             string `json:"name"`
	Province         string `json:"province"`
	Address          string `json:"address"`
	RegistrationType string `json:"registrationType"`
}

// ItemInput is one line as entered by a user or sent by an ERP.
// Optional pointer amounts override the engine's computation.
type ItemInput struct {
	ProductID       *int64           `json:"productId"`
	ProductCode     string           `json:"productCode"`
	HSCode          string           `json:"hsCode"`
	Description     string           `json:"description"`
	UoM             string           `json:"uom"`
	Quantity        decimal.Decimal  `json:"quantity"`
	UnitPrice       decimal.Decimal  `json:"unitPrice"`
	DiscountPercent decimal.Decimal  `json:"discountPercent"`
	DiscountAmount  decimal.Decimal  `json:"discountAmount"`
	Value           *decimal.Decimal `json:"value"`
	SaleType        string           `json:"saleType"`
	Rate            string           `json:"rate"`
	RetailPrice     decimal.Decimal  `json:"retailPrice"`
	RetailValue     *decimal.Decimal `json:"retailValue"`
	FurtherTaxMode  string           `json:"furtherTaxMode"`
	FurtherTax      *decimal.Decimal `json:"furtherTax"`
	ExtraTaxRate    decimal.Decimal  `json:"extraTaxRate"`
	ExtraTax        *decimal.Decimal `json:"extraTax"`
	FEDRate         decimal.Decimal  `json:"fedRate"`
	FED             *decimal.Decimal `json:"fed"`
	STWithheld      *decimal.Decimal `json:"stWithheld"`
	SalesTax        *decimal.Decimal `json:"salesTax"`
	SROScheduleNo   string           `json:"sroScheduleNo"`
	SROItemSerialNo string           `json:"sroItemSerialNo"`
	// Federal excise duty particulars (rule 150R(13)(aa)-(ff)); blanks
	// are taken from the product.
	FEDType      string          `json:"fedType"`
	FEDRateText  string          `json:"fedRateText"`
	FEDUnitPrice decimal.Decimal `json:"fedUnitPrice"`
	FEDSRO       string          `json:"fedSro"`
	FEDSROSerial string          `json:"fedSroSerial"`
}

// InvoiceInput creates or updates an invoice.
type InvoiceInput struct {
	Environment     domain.Environment `json:"environment"`
	DocType         string             `json:"docType"`
	InvoiceDate     string             `json:"invoiceDate"`
	CustomerID      *int64             `json:"customerId"`
	Buyer           *BuyerInput        `json:"buyer"`
	WithholdingMode *string            `json:"withholdingMode"`
	InvoiceRefNo    string             `json:"invoiceRefNo"`
	RefInvoiceID    *int64             `json:"refInvoiceId"`
	ScenarioID      string             `json:"scenarioId"`
	ExternalRef     string             `json:"externalRef"`
	Notes           string             `json:"notes"`
	// AdvanceReceipt marks an invoice issued for an advance received before
	// the goods or services are supplied (section 23(1)); AdvanceRef names,
	// on the final invoice, the advance receipt invoices it adjusts.
	AdvanceReceipt bool        `json:"advanceReceipt"`
	AdvanceRef     string      `json:"advanceRef"`
	Items          []ItemInput `json:"items"`
	// Submit asks for immediate submission to FBR after saving.
	Submit bool `json:"submit"`
	// Source is "ui", "api", "import" or "scenario".
	Source string `json:"-"`
}

// Build turns input into a computed (unsaved) invoice for a company. It
// resolves masters, applies the tax engine and validates locally.
func (s *Service) Build(ctx context.Context, c *store.Company, in *InvoiceInput) (*store.Invoice, validate.Result, error) {
	env := in.Environment
	if env == "" {
		env = c.Environment
	}
	if !env.Valid() {
		return nil, validate.Result{}, Invalid("unknown environment %q", env)
	}
	docType := domain.DocType(strings.TrimSpace(in.DocType))
	if docType == "" {
		docType = domain.DocSaleInvoice
	}
	date := strings.TrimSpace(in.InvoiceDate)
	if date == "" {
		date = s.Today()
	}
	inv := &store.Invoice{
		CompanyID: c.ID, Environment: env, DocType: docType, InvoiceDate: date, Status: domain.StatusDraft,
		SellerNTNCNIC: c.NTNCNIC, SellerName: c.Name, SellerProvince: c.Province, SellerAddress: c.Address,
		InvoiceRefNo: strings.TrimSpace(in.InvoiceRefNo), RefInvoiceID: in.RefInvoiceID, ScenarioID: strings.ToUpper(strings.TrimSpace(in.ScenarioID)),
		ExternalRef: strings.TrimSpace(in.ExternalRef), Notes: in.Notes, Source: in.Source,
		AdvanceReceipt: in.AdvanceReceipt, AdvanceRef: truncate(cleanText(in.AdvanceRef), 300),
	}
	if inv.AdvanceReceipt && docType != domain.DocSaleInvoice {
		return nil, validate.Result{}, Invalid("only a sale invoice can be an advance receipt invoice")
	}
	if inv.AdvanceReceipt {
		inv.AdvanceRef = ""
	}
	if inv.Source == "" {
		inv.Source = "ui"
	}
	if env != domain.EnvSandbox {
		inv.ScenarioID = ""
	}

	// Buyer.
	if in.CustomerID != nil && *in.CustomerID > 0 {
		cu, err := s.Store.GetCustomer(ctx, c.ID, *in.CustomerID)
		if err != nil {
			return nil, validate.Result{}, Invalid("customer %d not found", *in.CustomerID)
		}
		inv.CustomerID = &cu.ID
		inv.BuyerNTNCNIC, inv.BuyerName, inv.BuyerProvince, inv.BuyerAddress = cu.NTNCNIC, cu.Name, cu.Province, cu.Address
		inv.BuyerRegistrationType = cu.RegistrationType
		inv.WithholdingMode = cu.WithholdingMode
	}
	if b := in.Buyer; b != nil {
		if b.NTNCNIC != "" || inv.CustomerID == nil {
			n, _ := validate.NormalizeRegNo(b.NTNCNIC)
			inv.BuyerNTNCNIC = n
		}
		if b.Name != "" {
			inv.BuyerName = cleanText(b.Name)
		}
		if b.Province != "" {
			inv.BuyerProvince = domain.NormalizeProvince(b.Province)
		}
		if b.Address != "" {
			inv.BuyerAddress = cleanText(b.Address)
		}
		if b.RegistrationType != "" {
			inv.BuyerRegistrationType = domain.NormalizeRegistrationType(b.RegistrationType)
		}
		if inv.CustomerID == nil && inv.BuyerNTNCNIC != "" {
			if cu, err := s.Store.FindCustomerByRegNo(ctx, c.ID, inv.BuyerNTNCNIC); err == nil {
				// A buyer known by NTN/CNIC takes the particulars recorded in
				// the customer master for anything the input leaves blank, in
				// particular the registration type (it decides further tax and
				// how FBR attributes the buyer's input tax).
				inv.CustomerID = &cu.ID
				if inv.WithholdingMode == "" {
					inv.WithholdingMode = cu.WithholdingMode
				}
				if inv.BuyerRegistrationType == "" {
					inv.BuyerRegistrationType = cu.RegistrationType
				}
				if inv.BuyerName == "" {
					inv.BuyerName = cu.Name
				}
				if inv.BuyerProvince == "" {
					inv.BuyerProvince = cu.Province
				}
				if inv.BuyerAddress == "" {
					inv.BuyerAddress = cu.Address
				}
			}
		}
	}
	if inv.BuyerRegistrationType == "" {
		inv.BuyerRegistrationType = domain.Unregistered
	}
	if in.WithholdingMode != nil {
		inv.WithholdingMode = *in.WithholdingMode
	}

	// Debit note reference.
	var original *store.Invoice
	if docType == domain.DocDebitNote {
		if inv.RefInvoiceID != nil {
			o, err := s.Store.GetInvoice(ctx, c.ID, *inv.RefInvoiceID)
			if err != nil {
				return nil, validate.Result{}, Invalid("referenced invoice not found")
			}
			original = o
			inv.InvoiceRefNo = o.FBRInvoiceNumber
		} else if inv.InvoiceRefNo != "" {
			if o, err := s.Store.GetInvoiceByFBRNumber(ctx, c.ID, inv.InvoiceRefNo); err == nil {
				original = o
				inv.RefInvoiceID = &o.ID
			}
		}
		if original != nil && original.Status != domain.StatusAccepted {
			return nil, validate.Result{}, Invalid("a debit note can only reference an invoice accepted by FBR")
		}
	}

	// Lines.
	if len(in.Items) == 0 {
		return nil, validate.Result{}, Invalid("at least one line item is required")
	}
	results := make([]tax.LineResult, 0, len(in.Items))
	saleTypes := make([]string, 0, len(in.Items))
	for i := range in.Items {
		it, res, err := s.buildItem(ctx, c, inv, &in.Items[i])
		if err != nil {
			return nil, validate.Result{}, fmt.Errorf("line %d: %w", i+1, err)
		}
		inv.Items = append(inv.Items, it)
		results = append(results, res)
		saleTypes = append(saleTypes, it.SaleType)
	}
	t := tax.SumLines(results, saleTypes)
	inv.Totals = store.Totals{Gross: t.Gross, Discount: t.Discount, ValueExclST: t.ValueExclST, RetailValue: t.RetailValue,
		SalesTax: t.SalesTax, FurtherTax: t.FurtherTax, ExtraTax: t.ExtraTax, FED: t.FED, STWithheld: t.STWithheld,
		TotalValue: t.TotalValue, AmountPayable: t.AmountPayable}

	res := s.ValidateLocal(ctx, c, inv, original)
	inv.Validation = res.Issues
	return inv, res, nil
}

func (s *Service) buildItem(ctx context.Context, c *store.Company, inv *store.Invoice, in *ItemInput) (*store.InvoiceItem, tax.LineResult, error) {
	it := &store.InvoiceItem{
		ProductID: in.ProductID, HSCode: strings.TrimSpace(in.HSCode), Description: cleanText(in.Description), UoM: strings.TrimSpace(in.UoM),
		Quantity: tax.R4(in.Quantity), UnitPrice: in.UnitPrice, DiscountPercent: in.DiscountPercent, DiscountAmount: in.DiscountAmount,
		ValueOverride: in.Value, SaleType: strings.TrimSpace(in.SaleType), Rate: strings.TrimSpace(in.Rate), RetailPrice: in.RetailPrice,
		RetailValueOverride: in.RetailValue, FurtherTaxMode: strings.TrimSpace(in.FurtherTaxMode), FurtherTaxOverride: in.FurtherTax,
		ExtraTaxRate: in.ExtraTaxRate, ExtraTaxOverride: in.ExtraTax, FEDRate: in.FEDRate, FEDOverride: in.FED,
		WithholdingOverride: in.STWithheld, SalesTaxOverride: in.SalesTax,
		SROScheduleNo: strings.TrimSpace(in.SROScheduleNo), SROItemSerialNo: strings.TrimSpace(in.SROItemSerialNo),
		FEDType: truncate(cleanText(in.FEDType), 60), FEDRateText: truncate(cleanText(in.FEDRateText), 60), FEDUnitPrice: in.FEDUnitPrice,
		FEDSRO: truncate(cleanText(in.FEDSRO), 120), FEDSROSerial: truncate(cleanText(in.FEDSROSerial), 60),
	}
	// Fill blanks from the product master.
	var p *store.Product
	if in.ProductID != nil && *in.ProductID > 0 {
		pp, err := s.Store.GetProduct(ctx, c.ID, *in.ProductID)
		if err != nil {
			return nil, tax.LineResult{}, Invalid("product %d not found", *in.ProductID)
		}
		p = pp
	} else if in.ProductCode != "" {
		pp, err := s.Store.FindProductByCode(ctx, c.ID, in.ProductCode)
		if err != nil {
			return nil, tax.LineResult{}, Invalid("product code %q not found", in.ProductCode)
		}
		p = pp
	}
	if p != nil {
		it.ProductID = &p.ID
		if it.HSCode == "" {
			it.HSCode = p.HSCode
		}
		if it.Description == "" {
			it.Description = p.Description
		}
		if it.UoM == "" {
			it.UoM = p.UoM
		}
		if it.SaleType == "" {
			it.SaleType = p.SaleType
		}
		if it.Rate == "" {
			it.Rate = p.Rate
		}
		if it.SROScheduleNo == "" {
			it.SROScheduleNo = p.SROScheduleNo
		}
		if it.SROItemSerialNo == "" {
			it.SROItemSerialNo = p.SROItemSerialNo
		}
		if it.UnitPrice.IsZero() && in.Value == nil {
			it.UnitPrice = p.UnitPrice
		}
		if it.RetailPrice.IsZero() {
			it.RetailPrice = p.RetailPrice
		}
		if it.FurtherTaxMode == "" {
			it.FurtherTaxMode = p.FurtherTaxMode
		}
		if it.ExtraTaxRate.IsZero() {
			it.ExtraTaxRate = p.ExtraTaxRate
		}
		if it.FEDRate.IsZero() {
			it.FEDRate = p.FEDRate
		}
		if it.FEDType == "" {
			it.FEDType = p.FEDType
		}
		if it.FEDRateText == "" {
			it.FEDRateText = p.FEDRateText
		}
		if it.FEDSRO == "" {
			it.FEDSRO = p.FEDSRO
		}
		if it.FEDSROSerial == "" {
			it.FEDSROSerial = p.FEDSROSerial
		}
	}
	it.SaleType = domain.CanonicalSaleTypeName(it.SaleType)
	if it.Rate == "" {
		if st, ok := domain.LookupSaleType(it.SaleType); ok {
			it.Rate = st.DefaultRate
		}
	}
	if it.FurtherTaxMode == "" {
		it.FurtherTaxMode = "auto"
	}

	li := tax.LineInput{
		Quantity: it.Quantity, UnitPrice: it.UnitPrice, Value: it.ValueOverride, DiscountAmount: it.DiscountAmount,
		DiscountPercent: it.DiscountPercent, SaleType: it.SaleType, Rate: it.Rate, RetailPrice: it.RetailPrice,
		RetailValue: it.RetailValueOverride, BuyerRegistered: inv.BuyerRegistrationType == domain.Registered,
		FurtherTaxRate: c.FurtherTaxRate, FurtherTaxAmount: it.FurtherTaxOverride,
		ExtraTaxRate: it.ExtraTaxRate, ExtraTaxAmount: it.ExtraTaxOverride, FEDRate: it.FEDRate, FEDAmount: it.FEDOverride,
		SalesTaxAmount: it.SalesTaxOverride,
	}
	switch it.FurtherTaxMode {
	case "yes":
		t := true
		li.FurtherTaxApplies = &t
	case "no":
		f := false
		li.FurtherTaxApplies = &f
	}
	if it.WithholdingOverride != nil {
		li.Withholding, li.WithholdingAmount = tax.WithholdAmount, it.WithholdingOverride
	} else {
		switch tax.WithholdingMode(inv.WithholdingMode) {
		case tax.WithholdFraction:
			li.Withholding, li.WithholdingFraction = tax.WithholdFraction, c.WithholdingFraction
		case tax.WithholdFull:
			li.Withholding = tax.WithholdFull
		}
	}
	res := tax.ComputeLine(li)
	it.Gross, it.Discount, it.ValueExclST, it.RetailValue = res.Gross, res.Discount, res.ValueExclST, res.RetailValue
	it.SalesTax, it.FurtherTax, it.ExtraTax, it.ExtraTaxEmpty = res.SalesTax, res.FurtherTax, res.ExtraTax, res.ExtraTaxEmpty
	it.FED, it.STWithheld, it.TotalValue = res.FED, res.STWithheld, res.TotalValue
	it.Warnings = res.Warnings
	if it.Rate == "" {
		it.Rate = res.Rate.Raw
	}
	return it, res, nil
}

// refContext builds the validation context from synced reference data.
func (s *Service) refContext(ctx context.Context, c *store.Company, env domain.Environment) validate.Context {
	vc := validate.Context{Env: env, Today: s.Now().In(PKT), SellerActivities: c.BusinessActivities}
	if v, err := decimal.NewFromString(strings.TrimSpace(s.Opts.CNICThreshold)); err == nil {
		vc.CNICThreshold = v
	}
	refEnv := env
	if env == domain.EnvSimulator {
		refEnv = domain.EnvSimulator
	}
	if l := s.cachedList(ctx, c.ID, refEnv, "uom"); len(l) > 0 {
		vc.KnownUOMs = l
	}
	if l := s.cachedList(ctx, c.ID, refEnv, "transtypecode"); len(l) > 0 {
		vc.KnownSaleTypes = l
	}
	if l := s.cachedList(ctx, c.ID, refEnv, "provinces"); len(l) > 0 {
		vc.KnownProvinces = l
	}
	return vc
}

// ValidateLocal checks an invoice against the local rules.
func (s *Service) ValidateLocal(ctx context.Context, c *store.Company, inv *store.Invoice, original *store.Invoice) validate.Result {
	p := s.BuildPayload(c, inv)
	vc := s.refContext(ctx, c, inv.Environment)
	if original != nil {
		vc.Original = &validate.OriginalInvoice{FBRNumber: original.FBRInvoiceNumber, Date: original.InvoiceDate,
			ValueExclST: original.Totals.ValueExclST, SalesTax: original.Totals.SalesTax, BuyerNTNCNIC: original.BuyerNTNCNIC}
	}
	res := validate.Payload(&p, vc)
	for i, it := range inv.Items {
		for _, w := range it.Warnings {
			res.Issues = append(res.Issues, validate.Issue{Line: i + 1, Field: "computation", Severity: validate.SevWarning, Message: w})
		}
		// SRO 1666(I)/2026 added the federal excise duty particulars to
		// rule 150R(13); FBR's API does not carry them, so they are printed.
		if it.FED.IsPositive() && (it.FEDType == "" || it.FEDSRO == "" || it.FEDSROSerial == "") {
			res.Issues = append(res.Issues, validate.Issue{Line: i + 1, Field: "fedType", Severity: validate.SevWarning,
				Message: "Enter the federal excise duty type and the FED Schedule/SRO reference and serial number for this line: rule 150R(13)(aa)-(ff) (SRO 1666(I)/2026) requires them on the invoice."})
		}
	}
	return res
}

// fbrText keeps descriptive text plain for FBR's parser, which integrators
// report rejecting control characters and escaped quotes: control characters
// become spaces, double quotes single quotes and backslashes slashes.
func fbrText(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '"':
			return '\''
		case r == '\\':
			return '/'
		case unicode.IsControl(r):
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

// BuildPayload maps an invoice to the DI JSON payload.
func (s *Service) BuildPayload(c *store.Company, inv *store.Invoice) fbr.InvoicePayload {
	p := fbr.InvoicePayload{
		InvoiceType: string(inv.DocType), InvoiceDate: inv.InvoiceDate,
		SellerNTNCNIC: inv.SellerNTNCNIC, SellerBusinessName: fbrText(inv.SellerName), SellerProvince: inv.SellerProvince, SellerAddress: fbrText(inv.SellerAddress),
		BuyerNTNCNIC: inv.BuyerNTNCNIC, BuyerBusinessName: fbrText(inv.BuyerName), BuyerProvince: inv.BuyerProvince, BuyerAddress: fbrText(inv.BuyerAddress),
		BuyerRegistrationType: string(inv.BuyerRegistrationType),
	}
	if inv.DocType == domain.DocDebitNote {
		p.InvoiceRefNo = inv.InvoiceRefNo
	} else if c != nil && c.SendInternalRef {
		p.InvoiceRefNo = inv.InternalNo
	}
	if inv.Environment == domain.EnvSandbox {
		p.ScenarioID = inv.ScenarioID
	}
	for _, it := range inv.Items {
		ip := fbr.ItemPayload{
			HSCode: it.HSCode, ProductDescription: fbrText(it.Description), Rate: it.Rate, UoM: it.UoM,
			Quantity: fbr.Q(it.Quantity), TotalValues: fbr.A(it.TotalValue), ValueSalesExcludingST: fbr.A(it.ValueExclST),
			FixedNotifiedValueOrRetailPrice: fbr.A(it.RetailValue), SalesTaxApplicable: fbr.A(it.SalesTax),
			SalesTaxWithheldAtSource: fbr.A(it.STWithheld), ExtraTax: fbr.A(it.ExtraTax), FurtherTax: fbr.A(it.FurtherTax),
			SROScheduleNo: it.SROScheduleNo, FEDPayable: fbr.A(it.FED), Discount: fbr.A(it.Discount),
			SaleType: it.SaleType, SROItemSerialNo: it.SROItemSerialNo,
		}
		if it.ExtraTaxEmpty {
			ip.ExtraTax = fbr.EmptyAmount()
		}
		p.Items = append(p.Items, ip)
	}
	return p
}

// docPrefix returns the internal number prefix for a document.
func docPrefix(c *store.Company, env domain.Environment, dt domain.DocType) string {
	pfx := c.InvoicePrefix
	if dt == domain.DocDebitNote {
		pfx = c.DebitNotePrefix
	}
	switch env {
	case domain.EnvSandbox:
		pfx = "SB-" + pfx
	case domain.EnvSimulator:
		pfx = "TR-" + pfx
	}
	return pfx
}

// CreateInvoice saves a new invoice (and submits it when requested).
func (s *Service) CreateInvoice(ctx context.Context, a Actor, companyID int64, in *InvoiceInput) (*store.Invoice, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	inv, _, err := s.Build(ctx, c, in)
	if err != nil {
		return nil, err
	}
	// Idempotency for ERP integrations: the same external reference returns
	// the existing invoice instead of creating a duplicate.
	if inv.ExternalRef != "" {
		if existing, err := s.Store.GetInvoiceByExternalRef(ctx, c.ID, inv.Environment, inv.ExternalRef); err == nil {
			return existing, nil
		}
	}
	inv.CreatedBy, inv.UpdatedBy = a.UserID, a.UserID
	err = s.Store.Tx(ctx, func(q store.Querier) error {
		n, err := store.NextNumber(ctx, q, c.ID, inv.Environment, inv.DocType, FiscalYear(inv.InvoiceDate))
		if err != nil {
			return err
		}
		inv.InternalNo = fmt.Sprintf("%s-%s-%06d", docPrefix(c, inv.Environment, inv.DocType), FiscalYear(inv.InvoiceDate), n)
		return store.InsertInvoice(ctx, q, inv)
	})
	if errors.Is(err, store.ErrConflict) && inv.ExternalRef != "" {
		if existing, e2 := s.Store.GetInvoiceByExternalRef(ctx, c.ID, inv.Environment, inv.ExternalRef); e2 == nil {
			return existing, nil
		}
	}
	if err != nil {
		return nil, err
	}
	s.Audit(ctx, a, c.ID, "invoice.create", "invoice", fmt.Sprint(inv.ID), map[string]any{"no": inv.InternalNo, "env": inv.Environment, "source": inv.Source,
		"total": inv.Totals.TotalValue.StringFixed(2)})
	if in.Submit {
		return s.Submit(ctx, a, c.ID, inv.ID, SubmitOptions{})
	}
	return s.Store.GetInvoice(ctx, c.ID, inv.ID)
}

// UpdateInvoice replaces an editable invoice's content (and submits it when
// requested — after the edit lock is released, since Submit takes it again).
func (s *Service) UpdateInvoice(ctx context.Context, a Actor, companyID, id int64, in *InvoiceInput) (*store.Invoice, error) {
	inv, err := s.updateInvoice(ctx, a, companyID, id, in)
	if err != nil {
		return nil, err
	}
	if in.Submit {
		return s.Submit(ctx, a, companyID, id, SubmitOptions{})
	}
	return inv, nil
}

func (s *Service) updateInvoice(ctx context.Context, a Actor, companyID, id int64, in *InvoiceInput) (*store.Invoice, error) {
	unlock := s.lockInvoice(id)
	defer unlock()
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	cur, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if !cur.Status.Editable() {
		return nil, Invalid("invoice %s is %s and can no longer be edited", cur.InternalNo, cur.Status)
	}
	in.Environment = cur.Environment
	if in.Source == "" {
		in.Source = cur.Source
	}
	inv, _, err := s.Build(ctx, c, in)
	if err != nil {
		return nil, err
	}
	if inv.DocType != cur.DocType {
		return nil, Invalid("the document type cannot be changed after creation")
	}
	inv.ID, inv.InternalNo, inv.CreatedBy, inv.UpdatedBy = cur.ID, cur.InternalNo, cur.CreatedBy, a.UserID
	inv.Status = domain.StatusDraft // any edit invalidates a previous FBR validation
	if err := s.Store.Tx(ctx, func(q store.Querier) error { return store.UpdateInvoiceContent(ctx, q, inv) }); err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil, Invalid("another invoice already uses external reference %q", inv.ExternalRef)
		}
		return nil, err
	}
	s.Audit(ctx, a, companyID, "invoice.update", "invoice", fmt.Sprint(id), map[string]any{"no": cur.InternalNo, "total": inv.Totals.TotalValue.StringFixed(2)})
	return s.Store.GetInvoice(ctx, companyID, id)
}

// DeleteInvoice removes a draft that was never reported to FBR.
func (s *Service) DeleteInvoice(ctx context.Context, a Actor, companyID, id int64) error {
	unlock := s.lockInvoice(id)
	defer unlock()
	inv, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return err
	}
	// Editable invoices (draft, validated, rejected) were never recorded by
	// FBR; anything else must be kept (six-year record retention).
	if !inv.Status.Editable() || inv.FBRInvoiceNumber != "" {
		return Invalid("invoice %s has been reported to FBR and cannot be deleted", inv.InternalNo)
	}
	if err := s.Store.DeleteInvoice(ctx, companyID, id); err != nil {
		return err
	}
	s.Audit(ctx, a, companyID, "invoice.delete", "invoice", fmt.Sprint(id), inv.InternalNo)
	return nil
}

// Compute builds an invoice without saving it (live totals for the UI/ERP).
func (s *Service) Compute(ctx context.Context, companyID int64, in *InvoiceInput) (*store.Invoice, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	inv, _, err := s.Build(ctx, c, in)
	return inv, err
}

// Payload returns the DI JSON that would be (or was) sent for an invoice.
func (s *Service) Payload(ctx context.Context, companyID, id int64) (json.RawMessage, error) {
	c, err := s.Store.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	inv, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if inv.PayloadJSON != "" {
		return json.RawMessage(inv.PayloadJSON), nil
	}
	p := s.BuildPayload(c, inv)
	b, err := json.MarshalIndent(p, "", "  ")
	return b, err
}

// sealHash computes the tamper-evident seal of an accepted invoice, chained
// to the company's previous accepted invoice.
func sealHash(prev string, inv *store.Invoice) string {
	payload := strings.Join([]string{
		fmt.Sprint(inv.CompanyID), string(inv.Environment), string(inv.DocType), inv.InternalNo, inv.InvoiceDate,
		inv.SellerNTNCNIC, inv.BuyerNTNCNIC, inv.FBRInvoiceNumber, inv.FBRDated, inv.PayloadHash,
		inv.Totals.ValueExclST.StringFixed(2), inv.Totals.SalesTax.StringFixed(2), inv.Totals.TotalValue.StringFixed(2),
	}, "|")
	return security.ChainHash(prev, payload)
}

// VerifySeal recomputes an accepted invoice's seal.
func VerifySeal(inv *store.Invoice) bool {
	if inv.SealHash == "" {
		return false
	}
	return sealHash(inv.PrevSealHash, inv) == inv.SealHash
}

// CancelWindow is the period after issuance during which an electronic
// invoice may be cancelled or edited through FBR's system without the
// Commissioner's approval (Sales Tax General Order 01 of 2026).
var CancelWindow = 72 * time.Hour

// fbrDatedLayouts are the date-time formats accepted for FBR's "dated"
// value (FBR returns the first; the others are tolerated from operators).
var fbrDatedLayouts = []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02T15:04"}

// ParseFBRDated parses FBR's issue date-time, which is Pakistan time.
func ParseFBRDated(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, l := range fbrDatedLayouts {
		if t, err := time.ParseInLocation(l, s, PKT); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// IssuedAt is when FBR issued the invoice: FBR's "dated" when known,
// otherwise the time the acceptance was recorded. The 72-hour cancellation
// window runs from it.
func IssuedAt(inv *store.Invoice) time.Time {
	if t, ok := ParseFBRDated(inv.FBRDated); ok {
		return t
	}
	return store.ParseTime(inv.AcceptedAt)
}

// CancelInput describes a cancellation.
type CancelInput struct {
	Reason string `json:"reason"`
	// Reference is FBR's/IRIS's confirmation of the cancellation.
	Reference string `json:"reference"`
	// CommissionerApproval is required after the 72-hour window.
	CommissionerApproval string `json:"commissionerApproval"`
	// UseAPI calls the configured FBR cancellation endpoint.
	UseAPI bool `json:"useApi"`
}

// CancelInvoice records the cancellation of an accepted invoice.
func (s *Service) CancelInvoice(ctx context.Context, a Actor, companyID, id int64, in CancelInput) (*store.Invoice, error) {
	unlock := s.lockInvoice(id)
	defer unlock()
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	inv, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != domain.StatusAccepted {
		return nil, Invalid("only invoices accepted by FBR can be cancelled")
	}
	if strings.TrimSpace(in.Reason) == "" {
		return nil, Invalid("a reason for cancellation is required")
	}
	issued := IssuedAt(inv)
	within := !issued.IsZero() && s.Now().Sub(issued) <= CancelWindow
	if !within && strings.TrimSpace(in.CommissionerApproval) == "" {
		return nil, Invalid("the invoice was issued more than 72 hours ago; under STGO 01 of 2026 cancellation requires the prior approval of the Commissioner Inland Revenue — enter the approval reference")
	}
	ref := strings.TrimSpace(in.Reference)
	if in.UseAPI && inv.Environment != domain.EnvSimulator {
		cl, err := s.Client(ctx, c, inv.Environment, &inv.ID)
		if err != nil {
			return nil, err
		}
		raw, err := cl.CancelInvoice(ctx, fbr.CancelRequest{InvoiceNumber: inv.FBRInvoiceNumber, SellerNTNCNIC: inv.SellerNTNCNIC, Reason: in.Reason})
		if err != nil {
			return nil, Invalid("FBR cancellation failed: %v", err)
		}
		ok, known, msg := fbr.CancelOutcome(raw)
		switch {
		case !known:
			return nil, Invalid("FBR's reply to the cancellation could not be interpreted (%s). Check the invoice on IRIS; if it shows as cancelled, record the cancellation with the IRIS reference.", truncate(msg, 300))
		case !ok:
			return nil, Invalid("FBR refused the cancellation: %s", msg)
		}
		if ref == "" {
			ref = truncate("Cancelled through the FBR API: "+strings.TrimSpace(string(raw)), 500)
		}
	}
	if ref == "" && inv.Environment != domain.EnvSimulator {
		return nil, Invalid("enter the cancellation reference shown on IRIS (or configure the FBR cancellation API)")
	}
	if !within {
		ref = strings.TrimSpace(ref + " | Commissioner approval: " + in.CommissionerApproval)
	}
	if err := s.Store.MarkCancelled(ctx, id, in.Reason, ref); err != nil {
		return nil, err
	}
	s.Audit(ctx, a, companyID, "invoice.cancel", "invoice", fmt.Sprint(id), map[string]any{"no": inv.InternalNo, "fbr": inv.FBRInvoiceNumber,
		"reason": in.Reason, "reference": ref, "within72h": within})
	return s.Store.GetInvoice(ctx, companyID, id)
}

// NewDebitNoteDraft creates a draft debit note referencing an accepted invoice,
// pre-filled with its lines (the operator then adjusts quantities/values).
func (s *Service) NewDebitNoteDraft(ctx context.Context, a Actor, companyID, originalID int64) (*store.Invoice, error) {
	orig, err := s.Store.GetInvoice(ctx, companyID, originalID)
	if err != nil {
		return nil, err
	}
	if orig.Status != domain.StatusAccepted || orig.DocType != domain.DocSaleInvoice {
		return nil, Invalid("debit notes can only be raised against sale invoices accepted by FBR")
	}
	in := &InvoiceInput{Environment: orig.Environment, DocType: string(domain.DocDebitNote), InvoiceDate: s.Today(), CustomerID: orig.CustomerID,
		Buyer: &BuyerInput{NTNCNIC: orig.BuyerNTNCNIC, Name: orig.BuyerName, Province: orig.BuyerProvince, Address: orig.BuyerAddress,
			RegistrationType: string(orig.BuyerRegistrationType)},
		WithholdingMode: &orig.WithholdingMode, RefInvoiceID: &orig.ID, ScenarioID: orig.ScenarioID,
		Notes: "Debit note against " + orig.InternalNo + " (FBR " + orig.FBRInvoiceNumber + ")", Source: "ui"}
	for _, it := range orig.Items {
		in.Items = append(in.Items, ItemInput{ProductID: it.ProductID, HSCode: it.HSCode, Description: it.Description, UoM: it.UoM,
			Quantity: it.Quantity, UnitPrice: it.UnitPrice, DiscountPercent: it.DiscountPercent, DiscountAmount: it.DiscountAmount,
			Value: it.ValueOverride, SaleType: it.SaleType, Rate: it.Rate, RetailPrice: it.RetailPrice, RetailValue: it.RetailValueOverride,
			FurtherTaxMode: it.FurtherTaxMode, ExtraTaxRate: it.ExtraTaxRate, FEDRate: it.FEDRate,
			SROScheduleNo: it.SROScheduleNo, SROItemSerialNo: it.SROItemSerialNo,
			FEDType: it.FEDType, FEDRateText: it.FEDRateText, FEDUnitPrice: it.FEDUnitPrice, FEDSRO: it.FEDSRO, FEDSROSerial: it.FEDSROSerial})
	}
	return s.CreateInvoice(ctx, a, companyID, in)
}
