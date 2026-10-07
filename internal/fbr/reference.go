// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package fbr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Reference data records. PRAL's reference APIs use irregular key casing
// (e.g. "ratE_DESC", "hS_CODE", "srO_ITEM_ID"), so responses are decoded
// generically and fields are matched case-insensitively.

// ProvinceRef is one row of /pdi/v1/provinces.
type ProvinceRef struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// DocTypeRef is one row of /pdi/v1/doctypecode.
type DocTypeRef struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// HSCodeRef is one row of /pdi/v1/itemdesccode.
type HSCodeRef struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// SROItemRef is one row of /pdi/v1/sroitemcode or /pdi/v2/SROItem.
type SROItemRef struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// TransTypeRef is one row of /pdi/v1/transtypecode (sale types).
type TransTypeRef struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// UOMRef is one row of /pdi/v1/uom or /pdi/v2/HS_UOM.
type UOMRef struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// SROScheduleRef is one row of /pdi/v1/SroSchedule.
type SROScheduleRef struct {
	ID          int    `json:"id"`
	SerNo       int    `json:"serNo"`
	Description string `json:"description"`
}

// RateRef is one row of /pdi/v2/SaleTypeToRate.
type RateRef struct {
	ID          int     `json:"id"`
	Description string  `json:"description"`
	Value       float64 `json:"value"`
}

// StatlResult is the response of /dist/v1/statl (Sales Tax Active Taxpayer List).
type StatlResult struct {
	StatusCode string `json:"statusCode"`
	Status     string `json:"status"`
	Active     bool   `json:"active"`
	Raw        string `json:"raw"`
}

// RegTypeResult is the response of /dist/v1/Get_Reg_Type.
type RegTypeResult struct {
	StatusCode       string `json:"statusCode"`
	RegistrationNo   string `json:"registrationNo"`
	RegistrationType string `json:"registrationType"`
	Registered       bool   `json:"registered"`
	Raw              string `json:"raw"`
}

// row is a generically decoded JSON object with case-insensitive lookup.
type row map[string]any

func normKey(k string) string {
	k = strings.ToLower(k)
	k = strings.NewReplacer("_", "", " ", "", "-", "").Replace(k)
	return k
}

func (r row) get(keys ...string) any {
	for _, want := range keys {
		w := normKey(want)
		for k, v := range r {
			if normKey(k) == w {
				return v
			}
		}
	}
	return nil
}

func (r row) str(keys ...string) string {
	switch v := r.get(keys...).(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func (r row) num(keys ...string) float64 {
	switch v := r.get(keys...).(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	}
	return 0
}

func (r row) int(keys ...string) int { return int(r.num(keys...)) }

func decodeRows(raw []byte) ([]row, error) {
	var rows []row
	if err := DecodeLenient(raw, &rows); err == nil {
		return rows, nil
	}
	// Some endpoints wrap the list: {"data":[...]} or {"Table":[...]}.
	var obj map[string]json.RawMessage
	if err := DecodeLenient(raw, &obj); err != nil {
		return nil, fmt.Errorf("unexpected reference response: %w", err)
	}
	for _, v := range obj {
		if err := DecodeLenient(v, &rows); err == nil {
			return rows, nil
		}
	}
	return nil, fmt.Errorf("reference response contains no list")
}

func (c *Client) refGet(ctx context.Context, op, path string, q url.Values) ([]row, []byte, error) {
	raw, _, err := c.do(ctx, op, http.MethodGet, c.url(path, q), nil)
	if err != nil {
		return nil, raw, err
	}
	rows, err := decodeRows(raw)
	if err != nil {
		return nil, raw, &CallError{Kind: ErrDecode, Body: string(raw), Err: err}
	}
	return rows, raw, nil
}

// fbrDate formats dates as PRAL's "dd-MMM-yyyy" (e.g. 04-Feb-2024).
func fbrDate(t time.Time) string { return t.Format("02-Jan-2006") }

// isoDate formats dates as YYYY-MM-DD.
func isoDate(t time.Time) string { return t.Format("2006-01-02") }

// Provinces calls /pdi/v1/provinces.
func (c *Client) Provinces(ctx context.Context) ([]ProvinceRef, error) {
	rows, _, err := c.refGet(ctx, "provinces", c.Endpoints.RefV1Path+"/provinces", nil)
	if err != nil {
		return nil, err
	}
	out := make([]ProvinceRef, 0, len(rows))
	for _, r := range rows {
		name := r.str("stateProvinceDesc", "description", "name")
		if name == "" {
			continue
		}
		out = append(out, ProvinceRef{Code: r.int("stateProvinceCode", "code", "id"), Name: name})
	}
	return out, nil
}

// DocTypes calls /pdi/v1/doctypecode.
func (c *Client) DocTypes(ctx context.Context) ([]DocTypeRef, error) {
	rows, _, err := c.refGet(ctx, "doctypecode", c.Endpoints.RefV1Path+"/doctypecode", nil)
	if err != nil {
		return nil, err
	}
	out := make([]DocTypeRef, 0, len(rows))
	for _, r := range rows {
		out = append(out, DocTypeRef{ID: r.int("docTypeId", "id"), Name: r.str("docDescription", "description", "name")})
	}
	return out, nil
}

// ItemDescCodes calls /pdi/v1/itemdesccode (the full HS code list).
func (c *Client) ItemDescCodes(ctx context.Context) ([]HSCodeRef, error) {
	rows, _, err := c.refGet(ctx, "itemdesccode", c.Endpoints.RefV1Path+"/itemdesccode", nil)
	if err != nil {
		return nil, err
	}
	out := make([]HSCodeRef, 0, len(rows))
	for _, r := range rows {
		code := r.str("hS_CODE", "hsCode", "code")
		if code == "" {
			continue
		}
		out = append(out, HSCodeRef{Code: code, Description: r.str("description", "desc")})
	}
	return out, nil
}

// SROItemCodes calls /pdi/v1/sroitemcode.
func (c *Client) SROItemCodes(ctx context.Context) ([]SROItemRef, error) {
	rows, _, err := c.refGet(ctx, "sroitemcode", c.Endpoints.RefV1Path+"/sroitemcode", nil)
	if err != nil {
		return nil, err
	}
	return parseSROItems(rows), nil
}

func parseSROItems(rows []row) []SROItemRef {
	out := make([]SROItemRef, 0, len(rows))
	for _, r := range rows {
		out = append(out, SROItemRef{ID: r.int("srO_ITEM_ID", "sroItemId", "id"), Description: r.str("srO_ITEM_DESC", "sroItemDesc", "description")})
	}
	return out
}

// TransTypes calls /pdi/v1/transtypecode (sale types).
func (c *Client) TransTypes(ctx context.Context) ([]TransTypeRef, error) {
	rows, _, err := c.refGet(ctx, "transtypecode", c.Endpoints.RefV1Path+"/transtypecode", nil)
	if err != nil {
		return nil, err
	}
	out := make([]TransTypeRef, 0, len(rows))
	for _, r := range rows {
		desc := r.str("transactioN_DESC", "transactionDesc", "description")
		if desc == "" {
			continue
		}
		out = append(out, TransTypeRef{ID: r.int("transactioN_TYPE_ID", "transactionTypeId", "id"), Description: desc})
	}
	return out, nil
}

// UOMs calls /pdi/v1/uom.
func (c *Client) UOMs(ctx context.Context) ([]UOMRef, error) {
	rows, _, err := c.refGet(ctx, "uom", c.Endpoints.RefV1Path+"/uom", nil)
	if err != nil {
		return nil, err
	}
	return parseUOMs(rows), nil
}

func parseUOMs(rows []row) []UOMRef {
	out := make([]UOMRef, 0, len(rows))
	for _, r := range rows {
		d := r.str("description", "uoM_DESC", "uom")
		if d == "" {
			continue
		}
		out = append(out, UOMRef{ID: r.int("uoM_ID", "uomId", "id"), Description: d})
	}
	return out
}

// SROSchedule calls /pdi/v1/SroSchedule for a rate id, date and province.
func (c *Client) SROSchedule(ctx context.Context, rateID int, date time.Time, provinceCode int) ([]SROScheduleRef, error) {
	q := url.Values{}
	q.Set("rate_id", strconv.Itoa(rateID))
	q.Set("date", fbrDate(date))
	q.Set("origination_supplier_csv", strconv.Itoa(provinceCode))
	rows, _, err := c.refGet(ctx, "SroSchedule", c.Endpoints.RefV1Path+"/SroSchedule", q)
	if err != nil {
		return nil, err
	}
	out := make([]SROScheduleRef, 0, len(rows))
	for _, r := range rows {
		out = append(out, SROScheduleRef{ID: r.int("srO_ID", "sroId", "id"), SerNo: r.int("serNo"), Description: r.str("srO_DESC", "sroDesc", "description")})
	}
	return out, nil
}

// SaleTypeToRate calls /pdi/v2/SaleTypeToRate. The returned descriptions are
// the exact strings to place in the "rate" field of an invoice line.
func (c *Client) SaleTypeToRate(ctx context.Context, date time.Time, transTypeID, provinceCode int) ([]RateRef, error) {
	q := url.Values{}
	q.Set("date", fbrDate(date))
	q.Set("transTypeId", strconv.Itoa(transTypeID))
	q.Set("originationSupplier", strconv.Itoa(provinceCode))
	rows, _, err := c.refGet(ctx, "SaleTypeToRate", c.Endpoints.RefV2Path+"/SaleTypeToRate", q)
	if err != nil {
		return nil, err
	}
	out := make([]RateRef, 0, len(rows))
	for _, r := range rows {
		out = append(out, RateRef{ID: r.int("ratE_ID", "rateId", "id"), Description: r.str("ratE_DESC", "rateDesc", "description"), Value: r.num("ratE_VALUE", "rateValue", "value")})
	}
	return out, nil
}

// HSUOM calls /pdi/v2/HS_UOM to get the unit(s) FBR accepts for an HS code.
func (c *Client) HSUOM(ctx context.Context, hsCode string, annexureID int) ([]UOMRef, error) {
	if annexureID == 0 {
		annexureID = 3
	}
	q := url.Values{}
	q.Set("hs_code", hsCode)
	q.Set("annexure_id", strconv.Itoa(annexureID))
	rows, _, err := c.refGet(ctx, "HS_UOM", c.Endpoints.RefV2Path+"/HS_UOM", q)
	if err != nil {
		return nil, err
	}
	return parseUOMs(rows), nil
}

// SROItems calls /pdi/v2/SROItem for an SRO id and date.
func (c *Client) SROItems(ctx context.Context, date time.Time, sroID int) ([]SROItemRef, error) {
	q := url.Values{}
	q.Set("date", isoDate(date))
	q.Set("sro_id", strconv.Itoa(sroID))
	rows, _, err := c.refGet(ctx, "SROItem", c.Endpoints.RefV2Path+"/SROItem", q)
	if err != nil {
		return nil, err
	}
	return parseSROItems(rows), nil
}

func (c *Client) distPost(ctx context.Context, op, path string, body any) (row, string, error) {
	raw, _, err := c.do(ctx, op, http.MethodPost, c.url(c.Endpoints.DistPath+path, nil), body)
	if err != nil {
		return nil, string(raw), err
	}
	var r row
	if err := DecodeLenient(raw, &r); err != nil {
		var rows []row
		if err2 := DecodeLenient(raw, &rows); err2 == nil && len(rows) > 0 {
			return rows[0], string(raw), nil
		}
		return nil, string(raw), &CallError{Kind: ErrDecode, Body: string(raw), Err: err}
	}
	return r, string(raw), nil
}

// STATL checks a registration number against the Sales Tax Active Taxpayer
// List on a date (/dist/v1/statl).
func (c *Client) STATL(ctx context.Context, regNo string, date time.Time) (StatlResult, error) {
	r, raw, err := c.distPost(ctx, "statl", "/statl", map[string]string{"regno": regNo, "date": isoDate(date)})
	if err != nil {
		return StatlResult{Raw: raw}, err
	}
	res := StatlResult{StatusCode: r.str("status code", "statusCode", "statuscode"), Status: r.str("status"), Raw: raw}
	st := strings.ToLower(strings.ReplaceAll(res.Status, "-", ""))
	res.Active = st == "active" || (res.StatusCode == "00" && !strings.Contains(st, "inactive"))
	return res, nil
}

// RegType returns whether a registration number is Registered or
// Unregistered for sales tax (/dist/v1/Get_Reg_Type).
func (c *Client) RegType(ctx context.Context, regNo string) (RegTypeResult, error) {
	r, raw, err := c.distPost(ctx, "Get_Reg_Type", "/Get_Reg_Type", map[string]string{"Registration_No": regNo})
	if err != nil {
		return RegTypeResult{Raw: raw}, err
	}
	res := RegTypeResult{
		StatusCode:       r.str("statuscode", "status code", "statusCode"),
		RegistrationNo:   r.str("REGISTRATION_NO", "registrationNo"),
		RegistrationType: r.str("REGISTRATION_TYPE", "registrationType"),
		Raw:              raw,
	}
	rt := strings.ToLower(strings.ReplaceAll(res.RegistrationType, " ", ""))
	res.Registered = rt == "registered"
	return res, nil
}
