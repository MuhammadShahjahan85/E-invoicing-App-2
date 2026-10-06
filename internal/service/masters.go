package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/security"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"
	"einvoicing/internal/validate"
)

var reHS = regexp.MustCompile(`^\d{4}\.\d{4}$`)

// SaveCustomer validates and stores a customer.
func (s *Service) SaveCustomer(ctx context.Context, a Actor, c *store.Customer) (*store.Customer, error) {
	c.Name = cleanText(c.Name)
	if c.Name == "" {
		return nil, Invalid("customer name is required")
	}
	n, note := validate.NormalizeRegNo(c.NTNCNIC)
	c.NTNCNIC = n
	_ = note
	c.RegistrationType = domain.NormalizeRegistrationType(string(c.RegistrationType))
	if c.RegistrationType == "" {
		c.RegistrationType = domain.Unregistered
	}
	if !c.RegistrationType.Valid() {
		return nil, Invalid("registration type must be Registered or Unregistered")
	}
	if c.NTNCNIC != "" && !validate.ValidRegNoFormat(c.NTNCNIC) {
		return nil, Invalid("NTN must be 7 or 9 digits, CNIC 13 digits (got %q)", c.NTNCNIC)
	}
	if c.RegistrationType == domain.Registered && c.NTNCNIC == "" {
		return nil, Invalid("a registered buyer must have an NTN or CNIC")
	}
	c.Province = domain.NormalizeProvince(c.Province)
	if c.Province == "" {
		return nil, Invalid("province is required (destination of supply)")
	}
	switch tax.WithholdingMode(c.WithholdingMode) {
	case tax.WithholdNone, tax.WithholdFraction, tax.WithholdFull:
	default:
		return nil, Invalid("withholding mode must be empty, 'fraction' or 'full'")
	}
	isNew := c.ID == 0
	if isNew {
		c.Active = true
	}
	if err := s.Store.SaveCustomer(ctx, c); err != nil {
		return nil, err
	}
	act := "customer.update"
	if isNew {
		act = "customer.create"
	}
	s.Audit(ctx, a, c.CompanyID, act, "customer", fmt.Sprint(c.ID), map[string]any{"name": c.Name, "ntn": c.NTNCNIC, "type": c.RegistrationType})
	return c, nil
}

// SaveProduct validates and stores a product.
func (s *Service) SaveProduct(ctx context.Context, a Actor, p *store.Product) (*store.Product, error) {
	p.Description = cleanText(p.Description)
	p.HSCode = strings.TrimSpace(p.HSCode)
	if p.Description == "" {
		return nil, Invalid("description is required")
	}
	if !reHS.MatchString(p.HSCode) {
		return nil, Invalid("HS code must be in PCT format NNNN.NNNN (e.g. 0101.2100)")
	}
	if strings.TrimSpace(p.UoM) == "" {
		return nil, Invalid("unit of measure is required")
	}
	p.SaleType = domain.CanonicalSaleTypeName(p.SaleType)
	st, known := domain.LookupSaleType(p.SaleType)
	if p.SaleType == "" {
		return nil, Invalid("sale type is required")
	}
	if strings.TrimSpace(p.Rate) == "" && known {
		p.Rate = st.DefaultRate
	}
	if r := tax.ParseRate(p.Rate); !r.Valid {
		return nil, Invalid("rate %q is not valid", p.Rate)
	}
	if known && st.SRORequired && (strings.TrimSpace(p.SROScheduleNo) == "" || strings.TrimSpace(p.SROItemSerialNo) == "") {
		return nil, Invalid("%q requires the SRO/Schedule number and item serial number", st.Name)
	}
	if known && st.Exempt {
		p.Rate = "Exempt"
	}
	switch p.FurtherTaxMode {
	case "", "auto", "yes", "no":
	default:
		return nil, Invalid("further tax mode must be auto, yes or no")
	}
	if p.UnitPrice.IsNegative() || p.RetailPrice.IsNegative() {
		return nil, Invalid("prices cannot be negative")
	}
	isNew := p.ID == 0
	if isNew {
		p.Active = true
	}
	if err := s.Store.SaveProduct(ctx, p); err != nil {
		return nil, err
	}
	act := "product.update"
	if isNew {
		act = "product.create"
	}
	s.Audit(ctx, a, p.CompanyID, act, "product", fmt.Sprint(p.ID), map[string]any{"description": p.Description, "hs": p.HSCode, "saleType": p.SaleType, "rate": p.Rate})
	return p, nil
}

// SaveCompany validates and stores company details.
func (s *Service) SaveCompany(ctx context.Context, a Actor, c *store.Company) (*store.Company, error) {
	c.Name = cleanText(c.Name)
	if c.Name == "" {
		return nil, Invalid("business name is required (as registered with FBR)")
	}
	n, _ := validate.NormalizeRegNo(c.NTNCNIC)
	c.NTNCNIC = n
	if !validate.ValidRegNoFormat(c.NTNCNIC) {
		return nil, Invalid("seller NTN must be 7 or 9 digits, or a 13-digit CNIC for individuals")
	}
	c.Province = domain.NormalizeProvince(c.Province)
	if c.Province == "" {
		return nil, Invalid("province is required (origination of supply)")
	}
	if strings.TrimSpace(c.Address) == "" {
		return nil, Invalid("business address is required")
	}
	if c.Environment == "" {
		c.Environment = domain.EnvSimulator
	}
	if !c.Environment.Valid() {
		return nil, Invalid("unknown environment")
	}
	for _, act := range c.BusinessActivities {
		if !contains(domain.BusinessActivities, act) {
			return nil, Invalid("unknown business activity %q", act)
		}
	}
	for _, sec := range c.Sectors {
		if !contains(domain.Sectors, sec) {
			return nil, Invalid("unknown sector %q", sec)
		}
	}
	for _, sc := range c.AssignedScenarios {
		if _, ok := domain.LookupScenario(sc); !ok {
			return nil, Invalid("unknown scenario %q", sc)
		}
	}
	if c.FurtherTaxRate.IsNegative() || c.FurtherTaxRate.GreaterThan(tax.Hundred) {
		return nil, Invalid("further tax rate must be between 0 and 100")
	}
	if c.WithholdingFraction.IsNegative() || c.WithholdingFraction.GreaterThan(tax.MustD("1")) {
		return nil, Invalid("withholding fraction must be between 0 and 1")
	}
	for _, p := range s.Provinces(ctx, c.Environment) {
		if strings.EqualFold(p.Name, c.Province) {
			c.ProvinceCode = p.Code
		}
	}
	if c.ID == 0 {
		c.Active = true
		id, err := s.Store.CreateCompany(ctx, c)
		if err != nil {
			if err == store.ErrConflict {
				return nil, Invalid("a company with NTN/CNIC %s already exists", c.NTNCNIC)
			}
			return nil, err
		}
		c.ID = id
		s.Audit(ctx, a, id, "company.create", "company", fmt.Sprint(id), map[string]any{"name": c.Name, "ntn": c.NTNCNIC})
	} else {
		prev, err := s.Store.GetCompany(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		if c.Environment == domain.EnvProduction && prev.Environment != domain.EnvProduction {
			if !prev.HasProductionToken {
				return nil, Invalid("cannot switch to production: save the production security token issued on IRIS first")
			}
			if err := s.Opts.License.CheckProduction(c.NTNCNIC); err != nil {
				return nil, Invalid("cannot switch to production: %v", err)
			}
		}
		if err := s.Store.UpdateCompany(ctx, c); err != nil {
			if err == store.ErrConflict {
				return nil, Invalid("a company with NTN/CNIC %s already exists", c.NTNCNIC)
			}
			return nil, err
		}
		changes := map[string]any{"name": c.Name}
		if prev.Environment != c.Environment {
			changes["environment"] = fmt.Sprintf("%s -> %s", prev.Environment, c.Environment)
		}
		if prev.NTNCNIC != c.NTNCNIC {
			changes["ntn"] = fmt.Sprintf("%s -> %s", prev.NTNCNIC, c.NTNCNIC)
		}
		s.Audit(ctx, a, c.ID, "company.update", "company", fmt.Sprint(c.ID), changes)
	}
	return s.Store.GetCompany(ctx, c.ID)
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// TokenInput sets an FBR security token.
type TokenInput struct {
	Environment domain.Environment `json:"environment"`
	Token       string             `json:"token"`
	Expiry      string             `json:"expiry"` // YYYY-MM-DD (tokens are valid for five years)
	Clear       bool               `json:"clear"`
}

// SetToken encrypts and stores an FBR token. Tokens are never returned by the API.
func (s *Service) SetToken(ctx context.Context, a Actor, companyID int64, in TokenInput) (*store.Company, error) {
	if in.Environment != domain.EnvSandbox && in.Environment != domain.EnvProduction {
		return nil, Invalid("tokens are set for the sandbox or production environment")
	}
	tok := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(in.Token), "Bearer "))
	if !in.Clear && len(tok) < 8 {
		return nil, Invalid("the token looks too short; paste the full security token from IRIS")
	}
	if in.Expiry != "" {
		if _, err := time.Parse("2006-01-02", in.Expiry); err != nil {
			return nil, Invalid("expiry must be YYYY-MM-DD")
		}
	}
	enc := ""
	if !in.Clear {
		var err error
		if enc, err = s.Vault.Encrypt(tok); err != nil {
			return nil, err
		}
	}
	if err := s.Store.SetCompanyToken(ctx, companyID, in.Environment, enc); err != nil {
		return nil, err
	}
	c, err := s.Store.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if in.Environment == domain.EnvSandbox {
		c.SandboxTokenExpiry = in.Expiry
	} else {
		c.ProductionTokenExpiry = in.Expiry
	}
	if err := s.Store.UpdateCompany(ctx, c); err != nil {
		return nil, err
	}
	action := "company.token_set"
	if in.Clear {
		action = "company.token_cleared"
	}
	s.Audit(ctx, a, companyID, action, "company", fmt.Sprint(companyID), map[string]any{"environment": in.Environment, "token": security.Mask(tok)})
	return s.Store.GetCompany(ctx, companyID)
}

// ConnectionTest is the result of testing FBR connectivity.
type ConnectionTest struct {
	Environment domain.Environment `json:"environment"`
	OK          bool               `json:"ok"`
	Steps       []TestStep         `json:"steps"`
}

// TestStep is one check of a connection test.
type TestStep struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// TestConnection checks token presence, reference API access and a sample
// validateinvoicedata call (nothing is recorded by FBR).
func (s *Service) TestConnection(ctx context.Context, a Actor, companyID int64, env domain.Environment) (*ConnectionTest, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if env == "" {
		env = c.Environment
	}
	res := &ConnectionTest{Environment: env, OK: true}
	step := func(name string, ok bool, msg string) {
		res.Steps = append(res.Steps, TestStep{Name: name, OK: ok, Message: msg})
		if !ok {
			res.OK = false
		}
	}
	switch env {
	case domain.EnvSandbox:
		step("Security token", c.HasSandboxToken, map[bool]string{true: "Sandbox token configured", false: "No sandbox token — obtain it from IRIS > Digital Invoicing"}[c.HasSandboxToken])
	case domain.EnvProduction:
		step("Security token", c.HasProductionToken, map[bool]string{true: "Production token configured", false: "No production token — it is issued after the sandbox scenarios pass"}[c.HasProductionToken])
		if err := s.Opts.License.CheckProduction(c.NTNCNIC); err != nil {
			step("Licence", false, err.Error())
		} else {
			step("Licence", true, "Production use licensed")
		}
	default:
		step("Simulator", true, "Training simulator — nothing is sent to FBR")
	}
	if !res.OK {
		return res, nil
	}
	cl, err := s.Client(ctx, c, env, nil)
	if err != nil {
		step("Client", false, err.Error())
		return res, nil
	}
	start := time.Now()
	prov, err := cl.Provinces(ctx)
	if err != nil {
		hint := ""
		if fbr.KindOf(err) == fbr.ErrAuth {
			hint = " — check the token and that this server's public IP is whitelisted by PRAL"
		}
		step("Reference API (provinces)", false, err.Error()+hint)
		return res, nil
	}
	step("Reference API (provinces)", true, fmt.Sprintf("%d provinces received in %d ms", len(prov), time.Since(start).Milliseconds()))

	sample := &fbr.InvoicePayload{InvoiceType: string(domain.DocSaleInvoice), InvoiceDate: s.Today(), SellerNTNCNIC: c.NTNCNIC,
		SellerBusinessName: c.Name, SellerProvince: c.Province, SellerAddress: c.Address, BuyerBusinessName: "Connection test",
		BuyerProvince: c.Province, BuyerAddress: c.Address, BuyerRegistrationType: string(domain.Unregistered),
		Items: []fbr.ItemPayload{{HSCode: "0101.2100", ProductDescription: "Connection test", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: fbr.Q(tax.MustD("1")), ValueSalesExcludingST: fbr.A(tax.MustD("100")), SalesTaxApplicable: fbr.A(tax.MustD("18")),
			FurtherTax: fbr.A(tax.MustD("4")), TotalValues: fbr.A(tax.MustD("122")), SaleType: domain.STStandard}}}
	if env == domain.EnvSandbox {
		sample.ScenarioID = "SN002"
	}
	vr, err := cl.ValidateInvoice(ctx, sample)
	switch {
	case err != nil:
		step("Invoice validation API", false, err.Error())
	case vr.IsValid():
		step("Invoice validation API", true, "Sample invoice validated by FBR (not recorded)")
	default:
		var msgs []string
		for _, e := range vr.Errors() {
			msgs = append(msgs, e.String())
		}
		// Reaching FBR's validator proves connectivity and authorisation; a
		// header error such as 0401 points at a token/NTN mismatch.
		ok := true
		for _, e := range vr.Errors() {
			if e.Code == "0401" {
				ok = false
			}
		}
		step("Invoice validation API", ok, "FBR answered: "+strings.Join(msgs, "; "))
	}
	s.Audit(ctx, a, companyID, "company.connection_test", "company", fmt.Sprint(companyID), res)
	return res, nil
}
