// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package domain

import (
	"sort"
	"strings"
)

// Scenario is one of FBR's sandbox test scenarios (SN001–SN028). Before PRAL
// issues a production token, the taxpayer must successfully post an invoice
// in the sandbox for every scenario assigned to it in IRIS. The assignment
// depends on the business activity and sector selected during DI registration.
type Scenario struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	// Buyer used by FBR's published sample payload (sandbox test identities).
	BuyerNTNCNIC          string           `json:"buyerNTNCNIC"`
	BuyerName             string           `json:"buyerName"`
	BuyerRegistrationType RegistrationType `json:"buyerRegistrationType"`
	// InvoiceRefNo used in FBR's sample (sale invoices may carry the seller's
	// own reference; FBR ignores it for sale invoices).
	InvoiceRefNo string `json:"invoiceRefNo,omitempty"`
	// Item is FBR's published sample line ("DI Scenarios Description for
	// Sandbox Testing" v1.11).
	Item ScenarioItem `json:"item"`
}

// ScenarioItem is the sample line for a scenario. Amount fields reproduce the
// FBR sample exactly so the "FBR sample" run mode can send them verbatim.
type ScenarioItem struct {
	HSCode                          string  `json:"hsCode"`
	ProductDescription              string  `json:"productDescription"`
	Rate                            string  `json:"rate"`
	UoM                             string  `json:"uoM"`
	Quantity                        float64 `json:"quantity"`
	TotalValues                     float64 `json:"totalValues"`
	ValueSalesExcludingST           float64 `json:"valueSalesExcludingST"`
	FixedNotifiedValueOrRetailPrice float64 `json:"fixedNotifiedValueOrRetailPrice"`
	SalesTaxApplicable              float64 `json:"salesTaxApplicable"`
	SalesTaxWithheldAtSource        float64 `json:"salesTaxWithheldAtSource"`
	ExtraTax                        float64 `json:"extraTax"`
	FurtherTax                      float64 `json:"furtherTax"`
	SROScheduleNo                   string  `json:"sroScheduleNo"`
	FEDPayable                      float64 `json:"fedPayable"`
	Discount                        float64 `json:"discount"`
	SaleType                        string  `json:"saleType"`
	SROItemSerialNo                 string  `json:"sroItemSerialNo"`
}

// Scenarios is the full catalogue in FBR order.
var Scenarios = []Scenario{
	{ID: "SN001", Title: "Sale of Standard Rate Goods to Registered Buyers",
		Description:  "Goods at the standard sales tax rate sold to a sales tax registered buyer, who may claim input tax.",
		BuyerNTNCNIC: "2046004", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Registered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Test product", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: 400, ValueSalesExcludingST: 1000, SalesTaxApplicable: 180, SaleType: STStandard}},
	{ID: "SN002", Title: "Sale of Standard Rate Goods to Unregistered Buyers",
		Description:  "Goods at the standard rate sold to a buyer not registered for sales tax; further tax applies.",
		BuyerNTNCNIC: "1234567", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Test product", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: 400, ValueSalesExcludingST: 1000, SalesTaxApplicable: 180, SaleType: STStandard}},
	{ID: "SN003", Title: "Sale of Steel (Melted and Re-Rolled) — Billets, Ingots and Long Bars",
		Description:  "Steel sector supplies by melters and re-rollers under sector specific rules.",
		BuyerNTNCNIC: "3710505701479", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "7214.1010", ProductDescription: "Steel long bars", Rate: "18%", UoM: "MT",
			Quantity: 1, ValueSalesExcludingST: 205000, SalesTaxApplicable: 36900, SaleType: STSteel}},
	{ID: "SN004", Title: "Sale of Steel Scrap by Ship Breakers",
		Description:  "Scrap recovered from ship breaking, treated separately for tax purposes.",
		BuyerNTNCNIC: "3710505701479", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "7204.1010", ProductDescription: "Ship breaking scrap", Rate: "18%", UoM: "MT",
			Quantity: 1, ValueSalesExcludingST: 175000, SalesTaxApplicable: 31500, SaleType: STShipBreaking}},
	{ID: "SN005", Title: "Sales of Reduced Rate Goods (Eighth Schedule)",
		Description:  "Goods taxed at a reduced rate listed in the Eighth Schedule.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0102.2930", ProductDescription: "Reduced rate product", Rate: "1%", UoM: "Numbers, pieces, units",
			Quantity: 1, ValueSalesExcludingST: 1000, SalesTaxApplicable: 10, SalesTaxWithheldAtSource: 50.23, FurtherTax: 120,
			SROScheduleNo: "EIGHTH SCHEDULE Table 1", FEDPayable: 50.36, Discount: 56.36, SaleType: STReduced, SROItemSerialNo: "82"}},
	{ID: "SN006", Title: "Sale of Exempt Goods (Sixth Schedule)",
		Description:  "Goods exempt from sales tax under the Sixth Schedule.",
		BuyerNTNCNIC: "2046004", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Registered,
		Item: ScenarioItem{HSCode: "0102.2930", ProductDescription: "Exempt product", Rate: "Exempt", UoM: "Numbers, pieces, units",
			Quantity: 1, ValueSalesExcludingST: 10, SalesTaxApplicable: 0, SalesTaxWithheldAtSource: 50.23, FurtherTax: 120,
			SROScheduleNo: "6th Schd Table I", FEDPayable: 50.36, Discount: 56.36, SaleType: STExempt, SROItemSerialNo: "100"}},
	{ID: "SN007", Title: "Sale of Zero-Rated Goods (Fifth Schedule)",
		Description:  "Goods charged at 0%; the supplier may still claim input tax.",
		BuyerNTNCNIC: "3710505701479", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Zero rated product", Rate: "0%", UoM: "Numbers, pieces, units",
			Quantity: 100, ValueSalesExcludingST: 100, SalesTaxApplicable: 0, SROScheduleNo: "327(I)/2008", SaleType: STZeroRated, SROItemSerialNo: "1"}},
	{ID: "SN008", Title: "Sale of 3rd Schedule Goods",
		Description:  "Goods taxed on the printed retail price rather than the transaction value.",
		BuyerNTNCNIC: "3710505701479", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Branded product", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: 100, TotalValues: 145, ValueSalesExcludingST: 0, FixedNotifiedValueOrRetailPrice: 1000, SalesTaxApplicable: 180, SaleType: STThirdSchedule}},
	{ID: "SN009", Title: "Purchase From Registered Cotton Ginners",
		Description:  "Cotton spinners' purchases from registered cotton ginners.",
		BuyerNTNCNIC: "2046004", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Registered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Cotton lint", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: 0, TotalValues: 2500, ValueSalesExcludingST: 2500, SalesTaxApplicable: 450, SaleType: STCottonGinners}},
	{ID: "SN010", Title: "Sale of Telecom Services by Mobile Operators",
		Description:  "Telecommunication services (calls, data, SMS) by mobile operators.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Telecom services", Rate: "17%", UoM: "Numbers, pieces, units",
			Quantity: 1000, ValueSalesExcludingST: 100, SalesTaxApplicable: 17, SaleType: STTelecom}},
	{ID: "SN011", Title: "Sale of Steel through Toll Manufacturing — Billets, Ingots and Long Bars",
		Description:  "Third party conversion of steel on behalf of another business.",
		BuyerNTNCNIC: "3710505701479", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "7214.9990", ProductDescription: "Toll manufactured steel", Rate: "18%", UoM: "MT",
			Quantity: 1, ValueSalesExcludingST: 205000, SalesTaxApplicable: 36900, SaleType: STTollManufacture}},
	{ID: "SN012", Title: "Sale of Petroleum Products",
		Description:  "Petroleum products at rates notified under the petroleum SRO.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Petroleum product", Rate: "1.43%", UoM: "Numbers, pieces, units",
			Quantity: 123, TotalValues: 132, ValueSalesExcludingST: 100, SalesTaxApplicable: 1.43, SalesTaxWithheldAtSource: 2,
			SROScheduleNo: "1450(I)/2021", SaleType: STPetroleum, SROItemSerialNo: "4"}},
	{ID: "SN013", Title: "Sale of Electricity to Retailers",
		Description:  "Supply of electricity to retailers.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Electricity", Rate: "5%", UoM: "Numbers, pieces, units",
			Quantity: 123, TotalValues: 212, ValueSalesExcludingST: 1000, SalesTaxApplicable: 50, SalesTaxWithheldAtSource: 11,
			SROScheduleNo: "1450(I)/2021", SaleType: STElectricityRetl, SROItemSerialNo: "4"}},
	{ID: "SN014", Title: "Sale of Gas to CNG Stations",
		Description:  "Natural gas supplied to CNG filling stations.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Natural gas", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: 123, ValueSalesExcludingST: 1000, SalesTaxApplicable: 180, SaleType: STGasToCNG}},
	{ID: "SN015", Title: "Sale of Mobile Phones",
		Description:  "Mobile handsets taxed under the Ninth Schedule.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Mobile phone", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: 123, ValueSalesExcludingST: 1234, SalesTaxApplicable: 222.12, SROScheduleNo: "NINTH SCHEDULE", SaleType: STMobilePhones, SROItemSerialNo: "1(A)"}},
	{ID: "SN016", Title: "Processing / Conversion of Goods",
		Description:  "Conversion of raw or semi-finished goods into finished products for another person.",
		BuyerNTNCNIC: "1000000000078", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Processing charges", Rate: "5%", UoM: "Numbers, pieces, units",
			Quantity: 1, ValueSalesExcludingST: 100, SalesTaxApplicable: 5, SaleType: STProcessing}},
	{ID: "SN017", Title: "Sale of Goods Where FED Is Charged in ST Mode",
		Description:  "Excisable goods where federal excise duty is collected in sales tax mode.",
		BuyerNTNCNIC: "7000009", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Excisable goods", Rate: "8%", UoM: "Numbers, pieces, units",
			Quantity: 1, ValueSalesExcludingST: 100, SalesTaxApplicable: 8, SaleType: STGoodsFED}},
	{ID: "SN018", Title: "Sale of Services Where FED Is Charged in ST Mode",
		Description:  "Services (e.g. advertisement, franchise, insurance) liable to FED invoiced under the sales tax framework.",
		BuyerNTNCNIC: "1000000000056", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Excisable services", Rate: "8%", UoM: "Numbers, pieces, units",
			Quantity: 20, ValueSalesExcludingST: 1000, SalesTaxApplicable: 80, SaleType: STServicesFED}},
	{ID: "SN019", Title: "Sale of Services (as per ICT Ordinance)",
		Description:  "Services taxable under the Islamabad Capital Territory (Tax on Services) Ordinance.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2900", ProductDescription: "Services", Rate: "5%", UoM: "Numbers, pieces, units",
			Quantity: 1, ValueSalesExcludingST: 100, SalesTaxApplicable: 5, SROScheduleNo: "ICTO TABLE I", SaleType: STServices, SROItemSerialNo: "1(ii)(ii)(a)"}},
	{ID: "SN020", Title: "Sale of Electric Vehicles",
		Description:  "Electric vehicles at the concessionary rate.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2900", ProductDescription: "Electric vehicle", Rate: "1%", UoM: "Numbers, pieces, units",
			Quantity: 122, ValueSalesExcludingST: 1000, SalesTaxApplicable: 10, SROScheduleNo: "6th Schd Table III", SaleType: STElectricVehicle, SROItemSerialNo: "20"}},
	{ID: "SN021", Title: "Sale of Cement / Concrete Block",
		Description:  "Cement and concrete blocks at a specific rate per unit.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Concrete blocks", Rate: "Rs.3", UoM: "Numbers, pieces, units",
			Quantity: 12, ValueSalesExcludingST: 123, SalesTaxApplicable: 36, SaleType: STCement}},
	{ID: "SN022", Title: "Sale of Potassium Chlorate",
		Description:  "Potassium chlorate at 18% plus Rs.60 per kilogram.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "3104.2000", ProductDescription: "Potassium chlorate", Rate: "18% along with rupees 60 per kilogram", UoM: "KG",
			Quantity: 1, ValueSalesExcludingST: 100, SalesTaxApplicable: 78, SROScheduleNo: "EIGHTH SCHEDULE Table 1", SaleType: STPotassiumChlor, SROItemSerialNo: "56"}},
	{ID: "SN023", Title: "Sale of CNG",
		Description:  "Compressed natural gas at the regional specific rate.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "CNG", Rate: "Rs.200", UoM: "Numbers, pieces, units",
			Quantity: 123, ValueSalesExcludingST: 234, SalesTaxApplicable: 24600, SROScheduleNo: "581(1)/2024", SaleType: STCNGSales, SROItemSerialNo: "Region-I"}},
	{ID: "SN024", Title: "Sale of Goods Listed in SRO 297(I)/2023",
		Description:  "Goods notified in SRO 297(I)/2023.",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "SRO 297 goods", Rate: "25%", UoM: "Numbers, pieces, units",
			Quantity: 123, ValueSalesExcludingST: 1000, SalesTaxApplicable: 250, SROScheduleNo: "297(I)/2023-Table-I", SaleType: STSRO297, SROItemSerialNo: "12"}},
	{ID: "SN025", Title: "Drugs Sold at Fixed ST Rate Under Serial 81 of Eighth Schedule Table 1",
		Description:  "Pharmaceutical products under serial 81 of the Eighth Schedule.",
		BuyerNTNCNIC: "1000000000078", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Unregistered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Medicine", Rate: "0%", UoM: "Numbers, pieces, units",
			Quantity: 1, ValueSalesExcludingST: 100, SalesTaxApplicable: 0, SROScheduleNo: "EIGHTH SCHEDULE Table 1", SaleType: STNonAdjustable, SROItemSerialNo: "81"}},
	{ID: "SN026", Title: "Sale of Goods at Standard Rate to End Consumers by Retailers",
		Description:  "Retail sale of standard rate goods to end consumers.",
		BuyerNTNCNIC: "1000000000078", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Registered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Retail goods", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: 123, ValueSalesExcludingST: 1000, SalesTaxApplicable: 180, SaleType: STStandard}},
	{ID: "SN027", Title: "Sale of 3rd Schedule Goods to End Consumers by Retailers",
		Description:  "Retail sale of Third Schedule goods on the printed retail price.",
		BuyerNTNCNIC: "7000006", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Registered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Branded retail goods", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: 1, ValueSalesExcludingST: 0, FixedNotifiedValueOrRetailPrice: 100, SalesTaxApplicable: 18, SaleType: STThirdSchedule}},
	{ID: "SN028", Title: "Sale of Goods at Reduced Rate to End Consumers by Retailers",
		Description:  "Retail sale of reduced rate goods (e.g. baby milk, books).",
		BuyerNTNCNIC: "1000000000000", BuyerName: "FERTILIZER MANUFAC IRS NEW", BuyerRegistrationType: Registered,
		Item: ScenarioItem{HSCode: "0101.2100", ProductDescription: "Reduced rate retail goods", Rate: "1%", UoM: "Numbers, pieces, units",
			Quantity: 1, ValueSalesExcludingST: 0, FixedNotifiedValueOrRetailPrice: 100, SalesTaxApplicable: 0,
			SROScheduleNo: "EIGHTH SCHEDULE Table 1", SaleType: STReduced, SROItemSerialNo: "70"}},
}

// LookupScenario returns the scenario with the given id (e.g. "SN005").
func LookupScenario(id string) (Scenario, bool) {
	id = strings.ToUpper(strings.TrimSpace(id))
	for _, s := range Scenarios {
		if s.ID == id {
			return s, true
		}
	}
	return Scenario{}, false
}

// scenarioMatrix is section 10 of PRAL's Technical Specification for DI API
// v1.12, "Applicable Scenarios based on Business Activity", row by row
// (rows 1-13, 16-28, ... 106-118 of the published table).
var scenarioMatrix = map[string]map[string][]string{
	"Manufacturer": {
		"All Other Sectors":        {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024"},
		"Steel":                    {"SN003", "SN004", "SN011"},
		"FMCG":                     {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN008"},
		"Textile":                  {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN009"},
		"Telecom":                  {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN010"},
		"Petroleum":                {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN012"},
		"Electricity Distribution": {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN013"},
		"Gas Distribution":         {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN014"},
		"Services":                 {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN018", "SN019"},
		"Automobile":               {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN020"},
		"CNG Stations":             {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN023"},
		"Pharmaceuticals":          {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024"},
		"Wholesale / Retails":      {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN026", "SN027", "SN028", "SN008"},
	},
	"Importer": {
		"All Other Sectors":        {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024"},
		"Steel":                    {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN003", "SN004", "SN011"},
		"FMCG":                     {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN008"},
		"Textile":                  {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN009"},
		"Telecom":                  {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN010"},
		"Petroleum":                {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN012"},
		"Electricity Distribution": {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN013"},
		"Gas Distribution":         {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN014"},
		"Services":                 {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN018", "SN019"},
		"Automobile":               {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN020"},
		"CNG Stations":             {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN023"},
		"Pharmaceuticals":          {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN025"},
		"Wholesale / Retails":      {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN026", "SN027", "SN028", "SN008"},
	},
	"Distributor": {
		"All Other Sectors":        {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN026", "SN027", "SN028", "SN008"},
		"Steel":                    {"SN003", "SN004", "SN011", "SN026", "SN027", "SN028", "SN008"},
		"FMCG":                     {"SN008", "SN026", "SN027", "SN028"},
		"Textile":                  {"SN009", "SN026", "SN027", "SN028", "SN008"},
		"Telecom":                  {"SN010", "SN026", "SN027", "SN028", "SN008"},
		"Petroleum":                {"SN012", "SN026", "SN027", "SN028", "SN008"},
		"Electricity Distribution": {"SN013", "SN026", "SN027", "SN028", "SN008"},
		"Gas Distribution":         {"SN014", "SN026", "SN027", "SN028", "SN008"},
		"Services":                 {"SN018", "SN019", "SN026", "SN027", "SN028", "SN008"},
		"Automobile":               {"SN020", "SN026", "SN027", "SN028", "SN008"},
		"CNG Stations":             {"SN023", "SN026", "SN027", "SN028", "SN008"},
		"Pharmaceuticals":          {"SN025", "SN026", "SN027", "SN028", "SN008"},
		"Wholesale / Retails":      {"SN001", "SN002", "SN026", "SN027", "SN028", "SN008"},
	},
	"Wholesaler": {
		"All Other Sectors":        {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN026", "SN027", "SN028", "SN008"},
		"Steel":                    {"SN003", "SN004", "SN011", "SN026", "SN027", "SN028", "SN008"},
		"FMCG":                     {"SN008", "SN026", "SN027", "SN028"},
		"Textile":                  {"SN009", "SN026", "SN027", "SN028", "SN008"},
		"Telecom":                  {"SN010", "SN026", "SN027", "SN028", "SN008"},
		"Petroleum":                {"SN012", "SN026", "SN027", "SN028", "SN008"},
		"Electricity Distribution": {"SN013", "SN026", "SN027", "SN028", "SN008"},
		"Gas Distribution":         {"SN014", "SN026", "SN027", "SN028", "SN008"},
		"Services":                 {"SN018", "SN019", "SN026", "SN027", "SN028", "SN008"},
		"Automobile":               {"SN020", "SN026", "SN027", "SN028", "SN008"},
		"CNG Stations":             {"SN023", "SN026", "SN027", "SN028", "SN008"},
		"Pharmaceuticals":          {"SN025", "SN026", "SN027", "SN028", "SN008"},
		"Wholesale / Retails":      {"SN001", "SN002", "SN026", "SN027", "SN028", "SN008"},
	},
	"Exporter": {
		"All Other Sectors":        {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024"},
		"Steel":                    {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN003", "SN004", "SN011"},
		"FMCG":                     {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN008"},
		"Textile":                  {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN009"},
		"Telecom":                  {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN010"},
		"Petroleum":                {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN012"},
		"Electricity Distribution": {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN013"},
		"Gas Distribution":         {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN014"},
		"Services":                 {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN018", "SN019"},
		"Automobile":               {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN020"},
		"CNG Stations":             {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN023"},
		"Pharmaceuticals":          {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN025"},
		"Wholesale / Retails":      {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN026", "SN027", "SN028", "SN008"},
	},
	"Retailer": {
		"All Other Sectors":        {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN026", "SN027", "SN028", "SN008"},
		"Steel":                    {"SN003", "SN004", "SN011"},
		"FMCG":                     {"SN026", "SN027", "SN028", "SN008"},
		"Textile":                  {"SN009", "SN026", "SN027", "SN028", "SN008"},
		"Telecom":                  {"SN010", "SN026", "SN027", "SN028", "SN008"},
		"Petroleum":                {"SN012", "SN026", "SN027", "SN028", "SN008"},
		"Electricity Distribution": {"SN013", "SN026", "SN027", "SN028", "SN008"},
		"Gas Distribution":         {"SN014", "SN026", "SN027", "SN028", "SN008"},
		"Services":                 {"SN018", "SN019", "SN026", "SN027", "SN028", "SN008"},
		"Automobile":               {"SN020", "SN026", "SN027", "SN028", "SN008"},
		"CNG Stations":             {"SN023", "SN026", "SN027", "SN028", "SN008"},
		"Pharmaceuticals":          {"SN025", "SN026", "SN027", "SN028", "SN008"},
		"Wholesale / Retails":      {"SN026", "SN027", "SN028", "SN008"},
	},
	"Service Provider": {
		"All Other Sectors":        {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN018", "SN019"},
		"Steel":                    {"SN003", "SN004", "SN011", "SN018", "SN019"},
		"FMCG":                     {"SN008", "SN018", "SN019"},
		"Textile":                  {"SN009", "SN018", "SN019"},
		"Telecom":                  {"SN010", "SN018", "SN019"},
		"Petroleum":                {"SN012", "SN018", "SN019"},
		"Electricity Distribution": {"SN013", "SN018", "SN019"},
		"Gas Distribution":         {"SN014", "SN018", "SN019"},
		"Services":                 {"SN018", "SN019"},
		"Automobile":               {"SN020", "SN018", "SN019"},
		"CNG Stations":             {"SN023", "SN018", "SN019"},
		"Pharmaceuticals":          {"SN025", "SN018", "SN019"},
		"Wholesale / Retails":      {"SN026", "SN027", "SN028", "SN008", "SN018", "SN019"},
	},
	"Other": {
		"All Other Sectors":        {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024"},
		"Steel":                    {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN003", "SN004", "SN011"},
		"FMCG":                     {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN008"},
		"Textile":                  {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN009"},
		"Telecom":                  {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN010"},
		"Petroleum":                {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN012"},
		"Electricity Distribution": {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN013"},
		"Gas Distribution":         {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN014"},
		"Services":                 {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN018", "SN019"},
		"Automobile":               {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN020"},
		"CNG Stations":             {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN023"},
		"Pharmaceuticals":          {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN025"},
		"Wholesale / Retails":      {"SN001", "SN002", "SN005", "SN006", "SN007", "SN015", "SN016", "SN017", "SN021", "SN022", "SN024", "SN026", "SN027", "SN028", "SN008"},
	},
}

// ApplicableScenarios returns the sandbox scenarios FBR's technical
// specification assigns to a business activity and sector. IRIS is
// authoritative: the scenarios it shows after DI registration override this
// suggestion and can be recorded per company.
func ApplicableScenarios(activity, sector string) []string {
	if sector == "" {
		sector = "All Other Sectors"
	}
	rows, ok := scenarioMatrix[activity]
	if !ok {
		rows = scenarioMatrix["Other"]
	}
	ids, ok := rows[sector]
	if !ok {
		ids = rows["All Other Sectors"]
	}
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return sortedKeys(set)
}

// ApplicableScenariosMulti unions the scenarios for several activities and sectors.
func ApplicableScenariosMulti(activities, sectors []string) []string {
	if len(sectors) == 0 {
		sectors = []string{"All Other Sectors"}
	}
	set := map[string]bool{}
	for _, a := range activities {
		for _, s := range sectors {
			for _, id := range ApplicableScenarios(a, s) {
				set[id] = true
			}
		}
	}
	return sortedKeys(set)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
