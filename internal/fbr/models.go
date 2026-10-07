// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

// Package fbr is the client for FBR's Digital Invoicing (DI) web services
// operated by PRAL (Pakistan Revenue Automation (Pvt) Ltd), as described in
// PRAL's "Technical Specification for DI API" (v1.12).
package fbr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// InvoicePayload is the JSON body of postinvoicedata / validateinvoicedata.
// Field order follows PRAL's samples.
type InvoicePayload struct {
	InvoiceType           string        `json:"invoiceType"`
	InvoiceDate           string        `json:"invoiceDate"`
	SellerNTNCNIC         string        `json:"sellerNTNCNIC"`
	SellerBusinessName    string        `json:"sellerBusinessName"`
	SellerProvince        string        `json:"sellerProvince"`
	SellerAddress         string        `json:"sellerAddress"`
	BuyerNTNCNIC          string        `json:"buyerNTNCNIC"`
	BuyerBusinessName     string        `json:"buyerBusinessName"`
	BuyerProvince         string        `json:"buyerProvince"`
	BuyerAddress          string        `json:"buyerAddress"`
	BuyerRegistrationType string        `json:"buyerRegistrationType"`
	InvoiceRefNo          string        `json:"invoiceRefNo"`
	ScenarioID            string        `json:"scenarioId,omitempty"`
	Items                 []ItemPayload `json:"items"`
}

// ItemPayload is one line of the DI payload.
type ItemPayload struct {
	HSCode                          string `json:"hsCode"`
	ProductDescription              string `json:"productDescription"`
	Rate                            string `json:"rate"`
	UoM                             string `json:"uoM"`
	Quantity                        Qty    `json:"quantity"`
	TotalValues                     Amount `json:"totalValues"`
	ValueSalesExcludingST           Amount `json:"valueSalesExcludingST"`
	FixedNotifiedValueOrRetailPrice Amount `json:"fixedNotifiedValueOrRetailPrice"`
	SalesTaxApplicable              Amount `json:"salesTaxApplicable"`
	SalesTaxWithheldAtSource        Amount `json:"salesTaxWithheldAtSource"`
	ExtraTax                        Amount `json:"extraTax"`
	FurtherTax                      Amount `json:"furtherTax"`
	SROScheduleNo                   string `json:"sroScheduleNo"`
	FEDPayable                      Amount `json:"fedPayable"`
	Discount                        Amount `json:"discount"`
	SaleType                        string `json:"saleType"`
	SROItemSerialNo                 string `json:"sroItemSerialNo"`
}

// Amount is a monetary field. FBR expects JSON numbers with up to two
// decimals; for some fields (extraTax on reduced-rate goods) it requires an
// empty string instead of 0, which Empty expresses.
type Amount struct {
	Value decimal.Decimal
	Empty bool
}

// A builds an Amount.
func A(d decimal.Decimal) Amount { return Amount{Value: d.Round(2)} }

// EmptyAmount builds an Amount serialised as "".
func EmptyAmount() Amount { return Amount{Empty: true} }

// MarshalJSON implements json.Marshaler.
func (a Amount) MarshalJSON() ([]byte, error) {
	if a.Empty {
		return []byte(`""`), nil
	}
	return []byte(a.Value.Round(2).StringFixed(2)), nil
}

// UnmarshalJSON accepts numbers, numeric strings, "" and null.
func (a *Amount) UnmarshalJSON(b []byte) error {
	d, empty, err := parseFlexibleNumber(b)
	if err != nil {
		return fmt.Errorf("amount: %w", err)
	}
	a.Value, a.Empty = d, empty
	return nil
}

// Qty is a quantity with up to four decimals.
type Qty struct{ Value decimal.Decimal }

// Q builds a Qty.
func Q(d decimal.Decimal) Qty { return Qty{Value: d.Round(4)} }

// MarshalJSON implements json.Marshaler.
func (q Qty) MarshalJSON() ([]byte, error) {
	return []byte(q.Value.Round(4).StringFixed(4)), nil
}

// UnmarshalJSON accepts numbers, numeric strings, "" and null.
func (q *Qty) UnmarshalJSON(b []byte) error {
	d, _, err := parseFlexibleNumber(b)
	if err != nil {
		return fmt.Errorf("quantity: %w", err)
	}
	q.Value = d
	return nil
}

func parseFlexibleNumber(b []byte) (decimal.Decimal, bool, error) {
	s := strings.TrimSpace(string(b))
	if s == "null" || s == `""` {
		return decimal.Zero, s == `""`, nil
	}
	if strings.HasPrefix(s, `"`) {
		var str string
		if err := json.Unmarshal(b, &str); err != nil {
			return decimal.Zero, false, err
		}
		str = strings.TrimSpace(strings.ReplaceAll(str, ",", ""))
		if str == "" {
			return decimal.Zero, true, nil
		}
		d, err := decimal.NewFromString(str)
		return d, false, err
	}
	d, err := decimal.NewFromString(s)
	return d, false, err
}

// InvoiceResponse is the response of postinvoicedata and validateinvoicedata.
type InvoiceResponse struct {
	InvoiceNumber      FlexString          `json:"invoiceNumber"`
	Dated              FlexString          `json:"dated"`
	ValidationResponse *ValidationResponse `json:"validationResponse"`
}

// ValidationResponse carries the header and per-item validation outcome.
type ValidationResponse struct {
	StatusCode      FlexString      `json:"statusCode"`
	Status          FlexString      `json:"status"`
	ErrorCode       FlexString      `json:"errorCode"`
	Error           FlexString      `json:"error"`
	InvoiceStatuses []InvoiceStatus `json:"invoiceStatuses"`
}

// InvoiceStatus is the per-item outcome. On success InvoiceNo is the item
// level FBR number ("<invoiceNumber>-<itemSNo>").
type InvoiceStatus struct {
	ItemSNo    FlexString `json:"itemSNo"`
	StatusCode FlexString `json:"statusCode"`
	Status     FlexString `json:"status"`
	InvoiceNo  FlexString `json:"invoiceNo"`
	ErrorCode  FlexString `json:"errorCode"`
	Error      FlexString `json:"error"`
}

// IsValid reports whether FBR accepted the invoice (header status "00"/
// "Valid" and every item valid).
func (r *InvoiceResponse) IsValid() bool {
	if r == nil || r.ValidationResponse == nil {
		return false
	}
	vr := r.ValidationResponse
	if string(vr.StatusCode) != "00" {
		return false
	}
	if s := strings.ToLower(string(vr.Status)); s != "" && s != "valid" {
		return false
	}
	for _, it := range vr.InvoiceStatuses {
		if string(it.StatusCode) != "" && string(it.StatusCode) != "00" {
			return false
		}
	}
	return true
}

// Errors flattens header and item errors into readable strings.
func (r *InvoiceResponse) Errors() []ErrorItem {
	var out []ErrorItem
	if r == nil || r.ValidationResponse == nil {
		return out
	}
	vr := r.ValidationResponse
	if vr.Error != "" || (vr.ErrorCode != "" && string(vr.StatusCode) != "00") {
		out = append(out, ErrorItem{Code: string(vr.ErrorCode), Message: string(vr.Error)})
	}
	for _, it := range vr.InvoiceStatuses {
		if (string(it.StatusCode) != "" && string(it.StatusCode) != "00") || it.Error != "" {
			n, _ := strconv.Atoi(string(it.ItemSNo))
			out = append(out, ErrorItem{Item: n, Code: string(it.ErrorCode), Message: string(it.Error)})
		}
	}
	return out
}

// ErrorItem is one FBR validation error. Item is 0 for header errors.
type ErrorItem struct {
	Item    int    `json:"item"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ErrorItem) String() string {
	prefix := ""
	if e.Item > 0 {
		prefix = fmt.Sprintf("Item %d: ", e.Item)
	}
	if e.Code != "" {
		return fmt.Sprintf("%s[%s] %s", prefix, e.Code, e.Message)
	}
	return prefix + e.Message
}

// FlexString decodes JSON strings, numbers, booleans and null as a string.
type FlexString string

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = FlexString(s)
		return nil
	}
	*f = FlexString(string(b))
	return nil
}

// String implements fmt.Stringer.
func (f FlexString) String() string { return string(f) }

// sanitizeJSON repairs the trailing commas FBR occasionally returns
// ({"a":1,} or [1,2,]) so the response can still be parsed.
func sanitizeJSON(b []byte) []byte {
	var out bytes.Buffer
	inStr, esc := false, false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inStr {
			out.WriteByte(c)
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			out.WriteByte(c)
			continue
		}
		if c == ',' {
			j := i + 1
			for j < len(b) && (b[j] == ' ' || b[j] == '\n' || b[j] == '\r' || b[j] == '\t') {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue
			}
		}
		out.WriteByte(c)
	}
	return out.Bytes()
}

// DecodeLenient unmarshals JSON, retrying after repairing trailing commas.
func DecodeLenient(b []byte, v any) error {
	if err := json.Unmarshal(b, v); err == nil {
		return nil
	}
	return json.Unmarshal(sanitizeJSON(b), v)
}
