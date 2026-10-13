// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package hscodes holds the complete list of HS (PCT) codes built into the
// software: every code of the current Pakistan Customs Tariff with its
// description, and every code FBR's IRIS accepts, including the services of
// chapter 98. pct.tsv.gz is made by scripts/hscodes/build.py from FBR's own
// publications; rebuild it when FBR issues a new tariff.
package hscodes

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"strings"
	"sync"
)

//go:embed pct.tsv.gz
var data []byte

// Entry is one HS code.
type Entry struct {
	Code        string // 0101.2100
	Description string // Pure-bred breeding animals
	Context     string // the heading and groups above the code
	InTariff    bool   // in the current Pakistan Customs Tariff
	InIRIS      bool   // in FBR's IRIS list
}

var (
	once    sync.Once
	entries []Entry
	index   map[string]int
	edition string
)

func load() {
	index = map[string]int{}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return
	}
	defer zr.Close()
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			// "# Pakistan Customs Tariff 2026-27 and FBR IRIS HS code list"
			if f := strings.Fields(line); len(f) > 4 {
				edition = f[4]
			}
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 4 {
			continue
		}
		index[f[0]] = len(entries)
		entries = append(entries, Entry{Code: f[0], Description: f[1], Context: f[2],
			InTariff: strings.Contains(f[3], "T") || strings.Contains(f[3], "S"), InIRIS: strings.Contains(f[3], "I")})
	}
}

// All returns every built-in code in code order.
func All() []Entry {
	once.Do(load)
	return entries
}

// Lookup returns a code's entry.
func Lookup(code string) (Entry, bool) {
	once.Do(load)
	i, ok := index[strings.TrimSpace(code)]
	if !ok {
		return Entry{}, false
	}
	return entries[i], true
}

// Edition is the tariff year of the built-in list, e.g. "2026-27".
func Edition() string {
	once.Do(load)
	return edition
}
