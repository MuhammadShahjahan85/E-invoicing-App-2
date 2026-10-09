// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package domain

import (
	"strings"
	"testing"
)

// The matrix must cover every IRIS business activity and sector, and only
// name scenarios that exist.
func TestScenarioMatrixComplete(t *testing.T) {
	for _, act := range BusinessActivities {
		rows, ok := scenarioMatrix[act]
		if !ok {
			t.Errorf("no rows for %s", act)
			continue
		}
		for _, sec := range Sectors {
			ids, ok := rows[sec]
			if !ok || len(ids) == 0 {
				t.Errorf("no scenarios for %s / %s", act, sec)
			}
			for _, id := range ids {
				if _, ok := LookupScenario(id); !ok {
					t.Errorf("%s / %s: unknown scenario %s", act, sec, id)
				}
			}
		}
		if len(rows) != len(Sectors) {
			t.Errorf("%s: %d sectors", act, len(rows))
		}
	}
}

// Rows checked against section 10 of the DI API v1.12 specification.
func TestScenarioMatrixRows(t *testing.T) {
	cases := []struct{ act, sec, want string }{
		{"Manufacturer", "All Other Sectors", "SN001 SN002 SN005 SN006 SN007 SN015 SN016 SN017 SN021 SN022 SN024"},
		{"Manufacturer", "Steel", "SN003 SN004 SN011"},
		{"Manufacturer", "Pharmaceuticals", "SN001 SN002 SN005 SN006 SN007 SN015 SN016 SN017 SN021 SN022 SN024"},
		{"Importer", "Pharmaceuticals", "SN001 SN002 SN005 SN006 SN007 SN015 SN016 SN017 SN021 SN022 SN024 SN025"},
		{"Distributor", "All Other Sectors", "SN001 SN002 SN005 SN006 SN007 SN008 SN015 SN016 SN017 SN021 SN022 SN024 SN026 SN027 SN028"},
		{"Distributor", "FMCG", "SN008 SN026 SN027 SN028"},
		{"Distributor", "Wholesale / Retails", "SN001 SN002 SN008 SN026 SN027 SN028"},
		{"Wholesaler", "Steel", "SN003 SN004 SN008 SN011 SN026 SN027 SN028"},
		{"Retailer", "Steel", "SN003 SN004 SN011"},
		{"Retailer", "Wholesale / Retails", "SN008 SN026 SN027 SN028"},
		{"Service Provider", "All Other Sectors", "SN001 SN002 SN005 SN006 SN007 SN015 SN016 SN017 SN018 SN019 SN021 SN022 SN024"},
		{"Service Provider", "Wholesale / Retails", "SN008 SN018 SN019 SN026 SN027 SN028"},
		{"Exporter", "Textile", "SN001 SN002 SN005 SN006 SN007 SN009 SN015 SN016 SN017 SN021 SN022 SN024"},
		{"Other", "CNG Stations", "SN001 SN002 SN005 SN006 SN007 SN015 SN016 SN017 SN021 SN022 SN023 SN024"},
	}
	for _, c := range cases {
		if got := strings.Join(ApplicableScenarios(c.act, c.sec), " "); got != c.want {
			t.Errorf("%s / %s:\n got  %s\n want %s", c.act, c.sec, got, c.want)
		}
	}
	if got := strings.Join(ApplicableScenariosMulti([]string{"Manufacturer", "Retailer"}, []string{"Steel"}), " "); got != "SN003 SN004 SN011" {
		t.Errorf("multi: %s", got)
	}
}
