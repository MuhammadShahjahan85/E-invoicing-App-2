// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package dataimport

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// --- HTML ---

// readHTML reads the tables of a web page, including the ".xls" files
// some systems produce that are really HTML.
func readHTML(text string) (*Book, error) {
	doc, err := html.Parse(strings.NewReader(text))
	if err != nil {
		return nil, fmt.Errorf("cannot read the HTML file: %v", err)
	}
	b := &Book{Format: "web page (HTML) table", Kind: "html"}
	var walk func(n *html.Node, inTable bool)
	walk = func(n *html.Node, inTable bool) {
		if n.Type == html.ElementNode && n.Data == "table" && !inTable {
			sh := Sheet{Name: fmt.Sprintf("Table %d", len(b.Sheets)+1)}
			htmlRows(n, &sh)
			b.Sheets = append(b.Sheets, sh)
			inTable = true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inTable)
		}
	}
	walk(doc, false)
	if len(b.Sheets) == 0 {
		return nil, fmt.Errorf("the web page has no tables")
	}
	return finish(b)
}

func htmlRows(table *html.Node, sh *Sheet) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" && n != table {
			return // nested tables belong to a cell
		}
		if n.Type == html.ElementNode && n.Data == "tr" {
			var row []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					row = append(row, htmlText(c))
					for i := 1; i < atoiDef(htmlAttr(c, "colspan"), 1) && len(row) < MaxCols; i++ {
						row = append(row, "")
					}
				}
			}
			sh.addRow(row)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(table)
}

func htmlText(n *html.Node) string {
	var b strings.Builder
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			b.WriteString(n.Data)
		case n.Type == html.ElementNode && (n.Data == "br" || n.Data == "p" || n.Data == "div"):
			b.WriteByte(' ')
		case n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style"):
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

func htmlAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// --- JSON and XML as trees ---

// node is an element of a JSON or XML document. Arrays are flattened:
// each element becomes a child with the array's name, like repeated XML
// elements.
type node struct {
	name     string
	value    string
	leaf     bool
	attr     bool // an XML attribute
	children []*node
	parent   *node
}

func (n *node) add(c *node) {
	c.parent = n
	n.children = append(n.children, c)
}

func readJSON(text string) (*Book, error) {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	root := &node{name: "document"}
	count := 0
	if err := jsonValue(dec, root, "record", &count); err != nil {
		return nil, fmt.Errorf("cannot read the JSON file: %v", err)
	}
	b := &Book{Format: "JSON file", Kind: "json"}
	sh := flatten(root)
	if isDIPayload(sh) {
		b.Format = "FBR Digital Invoicing JSON"
	}
	b.Sheets = []Sheet{sh}
	return finish(b)
}

// jsonValue reads one JSON value and adds it to parent under name.
func jsonValue(dec *json.Decoder, parent *node, name string, count *int) error {
	*count++
	if *count > 500000 {
		return errors.New("the document is too large")
	}
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			n := &node{name: name}
			parent.add(n)
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return err
				}
				key, _ := kt.(string)
				if err := jsonValue(dec, n, key, count); err != nil {
					return err
				}
			}
			_, err := dec.Token() // }
			return err
		case '[':
			for dec.More() {
				if err := jsonValue(dec, parent, name, count); err != nil {
					return err
				}
			}
			_, err := dec.Token() // ]
			return err
		}
	case nil:
		parent.add(&node{name: name, leaf: true})
	case json.Number:
		parent.add(&node{name: name, leaf: true, value: t.String()})
	case bool:
		parent.add(&node{name: name, leaf: true, value: strconv.FormatBool(t)})
	case string:
		parent.add(&node{name: name, leaf: true, value: t})
	}
	return nil
}

func readXML(text string) (*Book, error) {
	dec := xml.NewDecoder(strings.NewReader(text))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity
	root := &node{name: "document"}
	cur := root
	var text0 strings.Builder
	count := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if count == 0 {
				return nil, fmt.Errorf("cannot read the XML file: %v", err)
			}
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			count++
			if count > 500000 {
				return nil, errors.New("the XML document is too large")
			}
			n := &node{name: t.Name.Local}
			cur.add(n)
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
					continue
				}
				n.add(&node{name: a.Name.Local, leaf: true, attr: true, value: a.Value})
			}
			cur = n
			text0.Reset()
		case xml.CharData:
			text0.Write(t)
		case xml.EndElement:
			elems, attrs := false, false
			for _, c := range cur.children {
				if c.attr {
					attrs = true
				} else {
					elems = true
				}
			}
			if !elems {
				v := strings.TrimSpace(text0.String())
				if attrs {
					// <amount currency="PKR">100</amount>: the text joins
					// the attributes as a value named after the element.
					if v != "" {
						cur.add(&node{name: cur.name, leaf: true, attr: true, value: v})
					}
				} else {
					cur.leaf, cur.value = true, v
				}
			}
			text0.Reset()
			if cur.parent != nil {
				cur = cur.parent
			}
		}
	}
	b := &Book{Format: "XML file", Kind: "xml"}
	b.Sheets = []Sheet{flatten(root)}
	return finish(b)
}

// flatten turns a document tree into a table. The rows are the most
// numerous kind of record (an element with values of its own); each row
// also carries the values of its ancestors, so an invoice's buyer and date
// are repeated on each of its lines.
func flatten(root *node) Sheet {
	type group struct {
		nodes  []*node
		leaves int
		depth  int
	}
	groups := map[string]*group{}
	var order []string
	var walk func(n *node, path string, depth int)
	walk = func(n *node, path string, depth int) {
		leaves := 0
		for _, c := range n.children {
			if c.leaf {
				leaves++
			}
		}
		if leaves > 0 && n != root {
			g := groups[path]
			if g == nil {
				g = &group{depth: depth}
				groups[path] = g
				order = append(order, path)
			}
			g.nodes = append(g.nodes, n)
			g.leaves = max(g.leaves, leaves)
		}
		for _, c := range n.children {
			if !c.leaf {
				walk(c, path+"/"+c.name, depth+1)
			}
		}
	}
	walk(root, "", 0)
	var best *group
	for _, p := range order {
		g := groups[p]
		switch {
		case best == nil:
			best = g
		case len(g.nodes) > len(best.nodes):
			best = g
		case len(g.nodes) == len(best.nodes) && g.depth > best.depth && g.leaves >= 2:
			best = g
		}
	}
	sh := Sheet{Name: "Data"}
	if best == nil {
		return sh
	}
	var cols []string
	idx := map[string]int{}
	var rows []map[string]string
	for _, rec := range best.nodes {
		row := map[string]string{}
		set := func(name, v string) {
			if _, ok := row[name]; ok {
				return
			}
			row[name] = v
			if _, ok := idx[name]; !ok {
				idx[name] = len(cols)
				cols = append(cols, name)
			}
		}
		// The record's own values, then single nested objects (dotted
		// names), then the ancestors' values.
		var own func(n *node, prefix string)
		own = func(n *node, prefix string) {
			counts := map[string]int{}
			for _, c := range n.children {
				counts[c.name]++
			}
			for _, c := range n.children {
				switch {
				case c.leaf:
					set(prefix+c.name, c.value)
				case counts[c.name] == 1 && len(prefix) < 200:
					own(c, prefix+c.name+".")
				}
			}
		}
		own(rec, "")
		for a := rec.parent; a != nil && a != root; a = a.parent {
			for _, c := range a.children {
				if c.leaf {
					if _, ok := row[c.name]; ok {
						set(a.name+"."+c.name, c.value)
					} else {
						set(c.name, c.value)
					}
				}
			}
		}
		rows = append(rows, row)
		if len(rows) > MaxRows {
			break
		}
	}
	if len(cols) > MaxCols {
		cols = cols[:MaxCols]
	}
	sh.addRow(cols)
	for _, r := range rows {
		cells := make([]string, len(cols))
		for i, c := range cols {
			cells[i] = r[c]
		}
		sh.addRow(cells)
	}
	return sh
}

// isDIPayload recognises FBR Digital Invoicing invoice JSON.
func isDIPayload(sh Sheet) bool {
	if len(sh.Rows) == 0 {
		return false
	}
	have := map[string]bool{}
	for _, h := range sh.Rows[0] {
		have[strings.ToLower(h)] = true
	}
	return have["hscode"] && have["productdescription"] && (have["valuesalesexcludingst"] || have["salestaxapplicable"])
}
