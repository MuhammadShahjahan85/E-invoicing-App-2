// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

package fbrmock

import "einvoicing/internal/domain"

// Reference data served by the simulator. IDs are simulator-specific; real
// IDs come from FBR when reference data is synced against sandbox/production.

type transType struct {
	ID   int
	Desc string
}

var transTypes = func() []transType {
	out := make([]transType, 0, len(domain.SaleTypes))
	for i, st := range domain.SaleTypes {
		out = append(out, transType{ID: 1001 + i, Desc: st.Name})
	}
	return out
}()

type rateRow struct {
	ID    int
	Desc  string
	Value float64
}

// ratesBySaleType lists the rate descriptions offered for each sale type.
var ratesBySaleType = map[string][]rateRow{
	domain.STStandard:        {{ID: 413, Desc: "18%", Value: 18}},
	domain.STReduced:         {{ID: 501, Desc: "1%", Value: 1}, {ID: 502, Desc: "5%", Value: 5}, {ID: 503, Desc: "10%", Value: 10}, {ID: 504, Desc: "12%", Value: 12}},
	domain.STZeroRated:       {{ID: 520, Desc: "0%", Value: 0}},
	domain.STExempt:          {{ID: 530, Desc: "Exempt", Value: 0}},
	domain.STThirdSchedule:   {{ID: 540, Desc: "18%", Value: 18}},
	domain.STSteel:           {{ID: 550, Desc: "18%", Value: 18}},
	domain.STShipBreaking:    {{ID: 551, Desc: "18%", Value: 18}},
	domain.STCottonGinners:   {{ID: 552, Desc: "18%", Value: 18}},
	domain.STTelecom:         {{ID: 553, Desc: "17%", Value: 17}, {ID: 554, Desc: "19.5%", Value: 19.5}},
	domain.STTollManufacture: {{ID: 555, Desc: "18%", Value: 18}},
	domain.STPetroleum:       {{ID: 556, Desc: "1.43%", Value: 1.43}},
	domain.STElectricityRetl: {{ID: 557, Desc: "5%", Value: 5}},
	domain.STGasToCNG:        {{ID: 558, Desc: "18%", Value: 18}},
	domain.STMobilePhones:    {{ID: 559, Desc: "18%", Value: 18}, {ID: 560, Desc: "25%", Value: 25}},
	domain.STProcessing:      {{ID: 561, Desc: "5%", Value: 5}},
	domain.STGoodsFED:        {{ID: 562, Desc: "8%", Value: 8}},
	domain.STServicesFED:     {{ID: 563, Desc: "8%", Value: 8}},
	domain.STServices:        {{ID: 564, Desc: "5%", Value: 5}, {ID: 565, Desc: "16%", Value: 16}},
	domain.STElectricVehicle: {{ID: 566, Desc: "1%", Value: 1}},
	domain.STCement:          {{ID: 567, Desc: "Rs.3", Value: 3}},
	domain.STPotassiumChlor:  {{ID: 734, Desc: "18% along with rupees 60 per kilogram", Value: 18}},
	domain.STCNGSales:        {{ID: 568, Desc: "Rs.200", Value: 200}},
	domain.STSRO297:          {{ID: 569, Desc: "25%", Value: 25}},
	domain.STNonAdjustable:   {{ID: 570, Desc: "0%", Value: 0}, {ID: 571, Desc: "1%", Value: 1}},
	domain.STDTRE:            {{ID: 572, Desc: "0%", Value: 0}},
	domain.STSIM:             {{ID: 573, Desc: "18%", Value: 18}},
	domain.STRerollableScrap: {{ID: 574, Desc: "18%", Value: 18}},
}

type sroSchedule struct {
	ID    int
	SerNo int
	Desc  string
}

// schedulesByRate maps rate ids to SRO / schedule references.
var schedulesByRate = map[int][]sroSchedule{
	501: {{ID: 389, SerNo: 1, Desc: "EIGHTH SCHEDULE Table 1"}, {ID: 390, SerNo: 2, Desc: "EIGHTH SCHEDULE Table 2"}},
	502: {{ID: 389, SerNo: 1, Desc: "EIGHTH SCHEDULE Table 1"}},
	503: {{ID: 389, SerNo: 1, Desc: "EIGHTH SCHEDULE Table 1"}},
	504: {{ID: 389, SerNo: 1, Desc: "EIGHTH SCHEDULE Table 1"}},
	520: {{ID: 7, SerNo: 1, Desc: "327(I)/2008"}, {ID: 8, SerNo: 2, Desc: "FIFTH SCHEDULE"}},
	530: {{ID: 391, SerNo: 1, Desc: "6th Schd Table I"}, {ID: 392, SerNo: 2, Desc: "6th Schd Table II"}, {ID: 393, SerNo: 3, Desc: "6th Schd Table III"}},
	556: {{ID: 394, SerNo: 1, Desc: "1450(I)/2021"}},
	557: {{ID: 394, SerNo: 1, Desc: "1450(I)/2021"}},
	559: {{ID: 395, SerNo: 1, Desc: "NINTH SCHEDULE"}},
	560: {{ID: 395, SerNo: 1, Desc: "NINTH SCHEDULE"}},
	564: {{ID: 396, SerNo: 1, Desc: "ICTO TABLE I"}},
	565: {{ID: 396, SerNo: 1, Desc: "ICTO TABLE I"}},
	566: {{ID: 393, SerNo: 1, Desc: "6th Schd Table III"}},
	734: {{ID: 389, SerNo: 1, Desc: "EIGHTH SCHEDULE Table 1"}},
	568: {{ID: 397, SerNo: 1, Desc: "581(1)/2024"}},
	569: {{ID: 398, SerNo: 1, Desc: "297(I)/2023-Table-I"}},
	570: {{ID: 389, SerNo: 1, Desc: "EIGHTH SCHEDULE Table 1"}},
	571: {{ID: 389, SerNo: 1, Desc: "EIGHTH SCHEDULE Table 1"}},
}

type sroItem struct {
	ID   int
	Desc string
}

var itemsBySRO = map[int][]sroItem{
	389: {{ID: 724, Desc: "56"}, {ID: 725, Desc: "70"}, {ID: 726, Desc: "81"}, {ID: 727, Desc: "82"}},
	390: {{ID: 730, Desc: "1"}, {ID: 731, Desc: "2"}},
	7:   {{ID: 740, Desc: "1"}, {ID: 741, Desc: "2"}},
	8:   {{ID: 742, Desc: "1"}},
	391: {{ID: 750, Desc: "100"}, {ID: 751, Desc: "101"}},
	392: {{ID: 752, Desc: "1"}},
	393: {{ID: 753, Desc: "20"}},
	394: {{ID: 760, Desc: "4"}},
	395: {{ID: 770, Desc: "1(A)"}, {ID: 771, Desc: "1(B)"}},
	396: {{ID: 780, Desc: "1(ii)(ii)(a)"}},
	397: {{ID: 790, Desc: "Region-I"}, {ID: 791, Desc: "Region-II"}},
	398: {{ID: 800, Desc: "12"}},
}

type hsCode struct {
	Code string
	Desc string
	UOM  string
}

var hsCodes = []hsCode{
	{"0101.2100", "PURE-BRED BREEDING HORSES", "Numbers, pieces, units"},
	{"0101.2900", "OTHER HORSES", "Numbers, pieces, units"},
	{"0102.2930", "LIVE BOVINE ANIMALS - OTHER", "Numbers, pieces, units"},
	{"0401.1000", "MILK AND CREAM, FAT CONTENT NOT EXCEEDING 1%", "Liter"},
	{"1006.3010", "RICE, SEMI-MILLED OR WHOLLY MILLED - BASMATI", "KG"},
	{"1101.0010", "WHEAT FLOUR", "KG"},
	{"1507.9000", "SOYA-BEAN OIL - OTHER", "KG"},
	{"1701.9910", "REFINED SUGAR", "KG"},
	{"2202.1010", "AERATED WATERS", "Liter"},
	{"2523.2900", "PORTLAND CEMENT - OTHER", "KG"},
	{"2710.1210", "MOTOR SPIRIT", "Liter"},
	{"3004.9099", "MEDICAMENTS - OTHER", "Numbers, pieces, units"},
	{"3104.2000", "POTASSIUM CHLORIDE", "KG"},
	{"3401.1100", "SOAP FOR TOILET USE", "KG"},
	{"3923.2100", "SACKS AND BAGS OF POLYMERS OF ETHYLENE", "KG"},
	{"4011.1000", "NEW PNEUMATIC TYRES FOR MOTOR CARS", "Numbers, pieces, units"},
	{"4901.9900", "PRINTED BOOKS - OTHER", "Numbers, pieces, units"},
	{"5208.1100", "WOVEN FABRICS OF COTTON - PLAIN WEAVE", "Square Metre"},
	{"5904.9000", "FLOOR COVERINGS - OTHER", "Square Metre"},
	{"6109.1000", "T-SHIRTS OF COTTON", "Numbers, pieces, units"},
	{"6810.1100", "BUILDING BLOCKS AND BRICKS OF CEMENT", "Numbers, pieces, units"},
	{"7204.1010", "WASTE AND SCRAP OF CAST IRON - RE-ROLLABLE", "MT"},
	{"7214.1010", "BARS AND RODS OF IRON - FORGED", "MT"},
	{"7214.9990", "BARS AND RODS OF IRON - OTHER", "MT"},
	{"8415.1010", "AIR CONDITIONING MACHINES - WINDOW OR WALL TYPES", "Numbers, pieces, units"},
	{"8471.3010", "PORTABLE COMPUTERS (LAPTOPS)", "Numbers, pieces, units"},
	{"8517.1219", "MOBILE PHONES - OTHER", "Numbers, pieces, units"},
	{"8703.8090", "ELECTRIC VEHICLES - OTHER", "Numbers, pieces, units"},
	{"9804.0000", "SERVICES", "Numbers, pieces, units"},
	{"9805.1000", "SERVICES PROVIDED BY HOTELS AND RESTAURANTS", "Numbers, pieces, units"},
	{"9814.1000", "IT AND IT ENABLED SERVICES", "Numbers, pieces, units"},
}

// registry is the simulator's taxpayer register used by STATL / Get_Reg_Type
// and by registered-buyer checks. NTNs not listed are treated as unregistered.
var registry = map[string]bool{
	"2046004":       true,
	"7000006":       true,
	"7000008":       true,
	"7000009":       false,
	"0786909":       true,
	"8885801":       true,
	"1000000000078": true,
	"1000000000056": false,
	"1000000000000": false,
	"1234567":       false,
	"3710505701479": false,
	"4130276175937": true,
}

// SeedHSCode is an HS code with description and typical UoM.
type SeedHSCode struct {
	Code, Description, UOM string
}

// SeedHSCodes returns the simulator's HS code list (also used to seed the
// local HS code table before the full list is synced from FBR).
func SeedHSCodes() []SeedHSCode {
	out := make([]SeedHSCode, 0, len(hsCodes))
	for _, h := range hsCodes {
		out = append(out, SeedHSCode{Code: h.Code, Description: h.Desc, UOM: h.UOM})
	}
	return out
}
