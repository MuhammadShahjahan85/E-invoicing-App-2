// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package hscodes

import (
	"regexp"
	"testing"
)

func TestBuiltInList(t *testing.T) {
	all := All()
	if len(all) < 7500 {
		t.Fatalf("only %d codes built in", len(all))
	}
	if Edition() == "" {
		t.Fatal("no tariff edition")
	}
	format := regexp.MustCompile(`^\d{4}\.\d{4}$`)
	chapters := map[string]bool{}
	for i, e := range all {
		if !format.MatchString(e.Code) {
			t.Fatalf("bad code %q", e.Code)
		}
		if e.Description == "" {
			t.Fatalf("%s has no description", e.Code)
		}
		if i > 0 && all[i-1].Code >= e.Code {
			t.Fatalf("not in order at %s", e.Code)
		}
		chapters[e.Code[:2]] = true
	}
	// Every chapter of the HS (77 is reserved) and the services of chapter 98.
	for c := 1; c <= 98; c++ {
		ch := []byte{byte('0' + c/10), byte('0' + c%10)}
		if c != 77 && !chapters[string(ch)] {
			t.Errorf("chapter %s missing", ch)
		}
	}
	for code, want := range map[string]string{
		"0101.2100": "Pure-bred breeding animals",
		"7214.9990": "",
		"2501.0021": "Pink rock salt",
		"9801.1000": "Services provided or rendered by hotels",
	} {
		e, ok := Lookup(code)
		if !ok {
			t.Errorf("%s not found", code)
			continue
		}
		if want != "" && e.Description != want {
			t.Errorf("%s: %q, want %q", code, e.Description, want)
		}
	}
	if e, _ := Lookup("0102.2110"); e.Context != "Live bovine animals › Cattle › Pure-bred breeding animals" {
		t.Errorf("context of 0102.2110: %q", e.Context)
	}
	if _, ok := Lookup("0000.0000"); ok {
		t.Error("unknown code found")
	}
}
