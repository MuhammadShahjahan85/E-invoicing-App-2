// Package domain defines the core vocabulary of FBR Digital Invoicing (DI):
// environments, document types, buyer registration types, invoice statuses
// and the catalogue of FBR "sale types" (transaction types) together with the
// tax behaviour each one implies.
package domain

import "strings"

// Environment selects which FBR/PRAL endpoint set an invoice is reported to.
type Environment string

const (
	// EnvSandbox is PRAL's DI sandbox (postinvoicedata_sb / validateinvoicedata_sb).
	// Used for the mandatory scenario testing before a production token is issued.
	EnvSandbox Environment = "sandbox"
	// EnvProduction is the live DI system. Invoices reported here are legal tax invoices.
	EnvProduction Environment = "production"
	// EnvSimulator is a local, offline FBR simulator built into the product for
	// demonstrations and staff training. Nothing is sent to FBR.
	EnvSimulator Environment = "simulator"
)

// Valid reports whether e is a known environment.
func (e Environment) Valid() bool {
	switch e {
	case EnvSandbox, EnvProduction, EnvSimulator:
		return true
	}
	return false
}

// Label is a human readable label.
func (e Environment) Label() string {
	switch e {
	case EnvSandbox:
		return "FBR Sandbox (testing)"
	case EnvProduction:
		return "FBR Production (live)"
	case EnvSimulator:
		return "Training Simulator (offline)"
	}
	return string(e)
}

// DocType is the DI "invoiceType" value (doctypecode reference API).
type DocType string

const (
	// DocSaleInvoice is document type id 4 in the doctypecode reference API.
	DocSaleInvoice DocType = "Sale Invoice"
	// DocDebitNote is document type id 9. A debit note must carry the FBR
	// invoice number of the original invoice in invoiceRefNo.
	DocDebitNote DocType = "Debit Note"
)

// Valid reports whether d is accepted by the DI API.
func (d DocType) Valid() bool { return d == DocSaleInvoice || d == DocDebitNote }

// DocTypeID returns the numeric id FBR uses for the document type.
func (d DocType) DocTypeID() int {
	switch d {
	case DocSaleInvoice:
		return 4
	case DocDebitNote:
		return 9
	}
	return 0
}

// RegistrationType is the DI "buyerRegistrationType" value.
type RegistrationType string

const (
	Registered   RegistrationType = "Registered"
	Unregistered RegistrationType = "Unregistered"
)

// Valid reports whether r is accepted by the DI API.
func (r RegistrationType) Valid() bool { return r == Registered || r == Unregistered }

// NormalizeRegistrationType maps loose spellings ("registered", "REG",
// "unregistered", "un-registered") to the exact DI values.
func NormalizeRegistrationType(s string) RegistrationType {
	t := strings.ToLower(strings.TrimSpace(s))
	t = strings.ReplaceAll(t, "-", "")
	t = strings.ReplaceAll(t, " ", "")
	switch t {
	case "registered", "reg", "r", "active":
		return Registered
	case "unregistered", "unreg", "u", "notregistered", "inactive":
		return Unregistered
	}
	return RegistrationType(strings.TrimSpace(s))
}

// InvoiceStatus is the local lifecycle state of an invoice.
type InvoiceStatus string

const (
	// StatusDraft is editable and has never been accepted by FBR.
	StatusDraft InvoiceStatus = "DRAFT"
	// StatusValidated passed FBR's validateinvoicedata but is not yet posted.
	StatusValidated InvoiceStatus = "VALIDATED"
	// StatusQueued is waiting for the background submitter (e.g. offline).
	StatusQueued InvoiceStatus = "QUEUED"
	// StatusSubmitting is in flight to FBR (row is locked).
	StatusSubmitting InvoiceStatus = "SUBMITTING"
	// StatusAccepted carries an FBR invoice number. It is immutable.
	StatusAccepted InvoiceStatus = "ACCEPTED"
	// StatusRejected was refused by FBR validation; fix and resubmit.
	StatusRejected InvoiceStatus = "REJECTED"
	// StatusUncertain means the request reached FBR but no definitive answer
	// was received (timeout / connection drop). It must be reconciled against
	// IRIS before any retry so the sale is never reported twice.
	StatusUncertain InvoiceStatus = "UNCERTAIN"
	// StatusCancelled was cancelled through FBR within the permitted window
	// (STGO 01 of 2026: 72 hours) or with Commissioner approval.
	StatusCancelled InvoiceStatus = "CANCELLED"
)

// Editable reports whether the invoice content may still be changed.
func (s InvoiceStatus) Editable() bool {
	switch s {
	case StatusDraft, StatusValidated, StatusRejected:
		return true
	}
	return false
}

// Final reports whether FBR has issued a number for the invoice.
func (s InvoiceStatus) Final() bool { return s == StatusAccepted || s == StatusCancelled }

// TaxBasis tells the engine what value the sales-tax rate is applied to.
type TaxBasis string

const (
	// BasisValue applies the rate to the value of supply excluding tax
	// (section 2(46) Sales Tax Act, 1990 — net of trade discount).
	BasisValue TaxBasis = "value"
	// BasisRetailPrice applies the rate to the printed retail price
	// (Third Schedule goods, section 3(2)(a)).
	BasisRetailPrice TaxBasis = "retail_price"
)

// SaleType describes one FBR transaction type ("saleType" in the DI payload).
// Name must match the transtypecode reference API exactly, including FBR's own
// spelling (for example the pipe in "Goods as per SRO.297(|)/2023").
type SaleType struct {
	Name string `json:"name"`
	// TransTypeID is the FBR transactioN_TYPE_ID when known (filled by sync).
	TransTypeID int `json:"transTypeId,omitempty"`
	// Category is "goods" or "services".
	Category string `json:"category"`
	// DefaultRate is the most common FBR rate description for the type.
	DefaultRate string `json:"defaultRate"`
	// Basis is the sales tax base.
	Basis TaxBasis `json:"basis"`
	// SRORequired: FBR rejects the line without SRO/Schedule No and item
	// serial (errors 0077/0078).
	SRORequired bool `json:"sroRequired"`
	// SROTypical: an SRO/Schedule reference is normally expected; the UI
	// warns when missing but FBR decides.
	SROTypical bool `json:"sroTypical"`
	// ExtraTaxMustBeEmpty: FBR error 0091 — extraTax must be "" (not 0)
	// for reduced-rate goods.
	ExtraTaxMustBeEmpty bool `json:"extraTaxMustBeEmpty"`
	// FurtherTaxDefault: further tax under section 3(1A) is charged by
	// default when the buyer is unregistered.
	FurtherTaxDefault bool `json:"furtherTaxDefault"`
	// Exempt: rate is the literal "Exempt" and sales tax is zero.
	Exempt bool `json:"exempt"`
	// Scenario is the sandbox scenario that exercises this type, if any.
	Scenario string `json:"scenario,omitempty"`
	// Note is guidance for operators.
	Note string `json:"note,omitempty"`
}

// Sale type names exactly as published by FBR (transtypecode).
const (
	STStandard         = "Goods at standard rate (default)"
	STReduced          = "Goods at Reduced Rate"
	STZeroRated        = "Goods at zero-rate"
	STExempt           = "Exempt goods"
	STThirdSchedule    = "3rd Schedule Goods"
	STSteel            = "Steel melting and re-rolling"
	STShipBreaking     = "Ship breaking"
	STCottonGinners    = "Cotton ginners"
	STTelecom          = "Telecommunication services"
	STTollManufacture  = "Toll Manufacturing"
	STPetroleum        = "Petroleum Products"
	STElectricityRetl  = "Electricity Supply to Retailers"
	STGasToCNG         = "Gas to CNG stations"
	STMobilePhones     = "Mobile Phones"
	STProcessing       = "Processing/Conversion of Goods"
	STGoodsFED         = "Goods (FED in ST Mode)"
	STServicesFED      = "Services (FED in ST Mode)"
	STServices         = "Services"
	STElectricVehicle  = "Electric Vehicle"
	STCement           = "Cement /Concrete Block"
	STPotassiumChlor   = "Potassium Chlorate"
	STCNGSales         = "CNG Sales"
	STSRO297           = "Goods as per SRO.297(|)/2023"
	STNonAdjustable    = "Non-Adjustable Supplies"
	STDTRE             = "DTRE goods"
	STSIM              = "SIM"
	STRerollableScrap  = "Rerollable scrap by ship breakers"
)

// SaleTypes is the built-in catalogue used until the reference data is synced
// from FBR. Rates are defaults only; the authoritative rate description for a
// given date and province comes from the SaleTypeToRate reference API.
var SaleTypes = []SaleType{
	{Name: STStandard, Category: "goods", DefaultRate: "18%", Basis: BasisValue, FurtherTaxDefault: true, Scenario: "SN001",
		Note: "Standard rate under section 3(1) of the Sales Tax Act, 1990."},
	{Name: STReduced, Category: "goods", DefaultRate: "1%", Basis: BasisValue, SRORequired: true, ExtraTaxMustBeEmpty: true, FurtherTaxDefault: true, Scenario: "SN005",
		Note: "Eighth Schedule reduced rates. Provide 'EIGHTH SCHEDULE Table 1' (or 2) and the serial number. Extra tax must be left empty (FBR error 0091)."},
	{Name: STZeroRated, Category: "goods", DefaultRate: "0%", Basis: BasisValue, SRORequired: true, Scenario: "SN007",
		Note: "Fifth Schedule / zero-rating SRO. Provide the SRO or schedule and serial number."},
	{Name: STExempt, Category: "goods", DefaultRate: "Exempt", Basis: BasisValue, SRORequired: true, Exempt: true, Scenario: "SN006",
		Note: "Sixth Schedule exemption. Rate must be 'Exempt'; provide the table and serial number."},
	{Name: STThirdSchedule, Category: "goods", DefaultRate: "18%", Basis: BasisRetailPrice, Scenario: "SN008",
		Note: "Tax is charged on the printed retail price (section 3(2)(a)). Enter the retail price; further tax is not charged by default."},
	{Name: STSteel, Category: "goods", DefaultRate: "18%", Basis: BasisValue, FurtherTaxDefault: true, Scenario: "SN003",
		Note: "Billets, ingots and long bars by steel melters/re-rollers."},
	{Name: STShipBreaking, Category: "goods", DefaultRate: "18%", Basis: BasisValue, FurtherTaxDefault: true, Scenario: "SN004"},
	{Name: STCottonGinners, Category: "goods", DefaultRate: "18%", Basis: BasisValue, Scenario: "SN009"},
	{Name: STTelecom, Category: "services", DefaultRate: "17%", Basis: BasisValue, Scenario: "SN010"},
	{Name: STTollManufacture, Category: "goods", DefaultRate: "18%", Basis: BasisValue, FurtherTaxDefault: true, Scenario: "SN011"},
	{Name: STPetroleum, Category: "goods", DefaultRate: "1.43%", Basis: BasisValue, SRORequired: true, Scenario: "SN012"},
	{Name: STElectricityRetl, Category: "goods", DefaultRate: "5%", Basis: BasisValue, SRORequired: true, Scenario: "SN013"},
	{Name: STGasToCNG, Category: "goods", DefaultRate: "18%", Basis: BasisValue, Scenario: "SN014"},
	{Name: STMobilePhones, Category: "goods", DefaultRate: "18%", Basis: BasisValue, SRORequired: true, FurtherTaxDefault: true, Scenario: "SN015",
		Note: "Ninth Schedule. Provide 'NINTH SCHEDULE' and the serial (e.g. 1(A))."},
	{Name: STProcessing, Category: "services", DefaultRate: "5%", Basis: BasisValue, Scenario: "SN016"},
	{Name: STGoodsFED, Category: "goods", DefaultRate: "8%", Basis: BasisValue, Scenario: "SN017"},
	{Name: STServicesFED, Category: "services", DefaultRate: "8%", Basis: BasisValue, Scenario: "SN018"},
	{Name: STServices, Category: "services", DefaultRate: "5%", Basis: BasisValue, SROTypical: true, Scenario: "SN019",
		Note: "Services taxable under the ICT (Tax on Services) Ordinance — schedule 'ICTO TABLE I' and serial."},
	{Name: STElectricVehicle, Category: "goods", DefaultRate: "1%", Basis: BasisValue, SRORequired: true, Scenario: "SN020"},
	{Name: STCement, Category: "goods", DefaultRate: "Rs.3", Basis: BasisValue, Scenario: "SN021",
		Note: "Specific (per unit) rate."},
	{Name: STPotassiumChlor, Category: "goods", DefaultRate: "18% along with rupees 60 per kilogram", Basis: BasisValue, SRORequired: true, Scenario: "SN022",
		Note: "Ad valorem plus Rs.60 per kilogram; UoM must be KG."},
	{Name: STCNGSales, Category: "goods", DefaultRate: "Rs.200", Basis: BasisValue, SRORequired: true, Scenario: "SN023",
		Note: "Specific rate per unit by region under the CNG SRO."},
	{Name: STSRO297, Category: "goods", DefaultRate: "25%", Basis: BasisValue, SRORequired: true, Scenario: "SN024"},
	{Name: STNonAdjustable, Category: "goods", DefaultRate: "0%", Basis: BasisValue, SRORequired: true, Scenario: "SN025",
		Note: "Drugs at fixed rate under serial 81 of the Eighth Schedule (Table 1)."},
	{Name: STDTRE, Category: "goods", DefaultRate: "0%", Basis: BasisValue, SROTypical: true,
		Note: "Supplies under the Duty and Tax Remission for Exports scheme."},
	{Name: STSIM, Category: "goods", DefaultRate: "18%", Basis: BasisValue},
	{Name: STRerollableScrap, Category: "goods", DefaultRate: "18%", Basis: BasisValue},
}

// LookupSaleType finds a sale type by exact name, then case/space-insensitively.
// It also tolerates "SRO.297(I)/2023" for FBR's "SRO.297(|)/2023".
func LookupSaleType(name string) (SaleType, bool) {
	for _, st := range SaleTypes {
		if st.Name == name {
			return st, true
		}
	}
	key := canonicalSaleTypeKey(name)
	for _, st := range SaleTypes {
		if canonicalSaleTypeKey(st.Name) == key {
			return st, true
		}
	}
	return SaleType{}, false
}

// CanonicalSaleTypeName returns FBR's exact spelling for a loosely typed name,
// or the input unchanged when it is not in the catalogue.
func CanonicalSaleTypeName(name string) string {
	if st, ok := LookupSaleType(name); ok {
		return st.Name
	}
	return strings.TrimSpace(name)
}

func canonicalSaleTypeKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "(i)", "(|)")
	s = strings.ReplaceAll(s, "(1)", "(|)")
	var b strings.Builder
	for _, r := range s {
		if r == ' ' || r == '-' || r == '/' || r == '_' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Provinces is the seed list of FBR province descriptions (provinces API).
// Synced data from FBR replaces it.
var Provinces = []Province{
	{Code: 2, Name: "BALOCHISTAN"},
	{Code: 4, Name: "AZAD JAMMU AND KASHMIR"},
	{Code: 5, Name: "CAPITAL TERRITORY"},
	{Code: 6, Name: "KHYBER PAKHTUNKHWA"},
	{Code: 7, Name: "PUNJAB"},
	{Code: 8, Name: "SINDH"},
	{Code: 9, Name: "GILGIT BALTISTAN"},
}

// Province is one entry of the provinces reference API.
type Province struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// NormalizeProvince maps common spellings ("Islamabad", "ICT", "KPK",
// "Sindh") to the FBR description. Unknown values are upper-cased.
func NormalizeProvince(s string) string {
	t := strings.ToUpper(strings.TrimSpace(s))
	t = strings.Join(strings.Fields(t), " ")
	switch t {
	case "ISLAMABAD", "ICT", "ISLAMABAD CAPITAL TERRITORY", "FEDERAL CAPITAL", "CAPITAL", "FEDERAL":
		return "CAPITAL TERRITORY"
	case "KPK", "KP", "NWFP", "KHYBER PAKHTOONKHWA", "KHYBER-PAKHTUNKHWA":
		return "KHYBER PAKHTUNKHWA"
	case "AJK", "AZAD KASHMIR", "AZAD JAMMU & KASHMIR", "AZAD JAMMU KASHMIR":
		return "AZAD JAMMU AND KASHMIR"
	case "GB", "GILGIT-BALTISTAN", "GILGIT":
		return "GILGIT BALTISTAN"
	case "BALUCHISTAN", "BALOCHISTAN":
		return "BALOCHISTAN"
	case "PUNJAB", "PANJAB":
		return "PUNJAB"
	case "SINDH", "SIND":
		return "SINDH"
	}
	return t
}

// UOMs is the seed list of FBR unit of measure descriptions (uom API).
// The values are case sensitive.
var UOMs = []string{
	"Numbers, pieces, units", "KG", "Kilogram", "MT", "Liter", "Gram", "Gallon", "Dozen",
	"Meter", "Square Metre", "Square Foot", "SqY", "Cubic Metre", "Pcs", "Pair", "Packs",
	"SET", "Bag", "40KG", "Barrels", "Carat", "KWH", "1000 kWh", "Mega Watt", "MMBTU",
	"Thousand Unit", "Timber Logs", "Bill of lading", "Pound", "NO", "Others",
}

// BusinessActivities are the IRIS "business activity" choices that drive the
// sandbox scenarios assigned to a taxpayer.
var BusinessActivities = []string{
	"Manufacturer", "Importer", "Distributor", "Wholesaler", "Exporter", "Retailer", "Service Provider", "Other",
}

// Sectors are the IRIS sector choices.
var Sectors = []string{
	"All Other Sectors", "Steel", "FMCG", "Textile", "Telecom", "Petroleum", "Electricity Distribution",
	"Gas Distribution", "Services", "Automobile", "CNG Stations", "Pharmaceuticals", "Wholesale / Retails",
}
