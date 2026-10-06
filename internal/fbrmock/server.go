// Package fbrmock is an offline simulator of FBR's Digital Invoicing API.
//
// It is used (1) by the automated test-suite, (2) by the product's
// "Training Simulator" environment so staff can practise without reporting
// anything to FBR, and (3) as a standalone mock server ERP integrators can
// develop against (`einvoice mock-fbr`). Responses follow the shapes in
// PRAL's technical specification. Invoice numbers have the FBR format
// <seller NTN/CNIC>DI<unix-milliseconds>.
package fbrmock

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/tax"

	"github.com/shopspring/decimal"
)

// FaultKind selects an injected failure for the next post/validate call.
type FaultKind string

const (
	FaultNone        FaultKind = ""
	FaultTimeout     FaultKind = "timeout"     // sleep longer than the client timeout (after recording!)
	FaultDrop        FaultKind = "drop"        // record the invoice, then drop the connection
	FaultServerError FaultKind = "server_error" // HTTP 500
	FaultUnavailable FaultKind = "unavailable" // HTTP 503 (not processed)
	FaultUnauthorized FaultKind = "unauthorized"
)

// Server is the simulator. The zero value is not usable; call New.
type Server struct {
	mu sync.Mutex
	// Tokens maps bearer tokens to the seller registration number they were
	// issued for, per environment ("sandbox"/"production").
	Tokens map[string]TokenInfo
	// AcceptAnyToken makes every non-empty token valid for any seller
	// (simulator mode inside the product).
	AcceptAnyToken bool
	invoices       map[string]*fbr.InvoicePayload // by FBR invoice number
	invoiceDates   map[string]time.Time
	faults         []FaultKind
	// Delay for FaultTimeout.
	TimeoutDelay time.Duration
	// Now is the clock.
	Now func() time.Time
	seq int64
}

// TokenInfo describes a simulated security token.
type TokenInfo struct {
	SellerNTNCNIC string
	Sandbox       bool
}

// New returns a simulator.
func New() *Server {
	return &Server{
		Tokens:       map[string]TokenInfo{},
		invoices:     map[string]*fbr.InvoicePayload{},
		invoiceDates: map[string]time.Time{},
		TimeoutDelay: 3 * time.Second,
		Now:          time.Now,
	}
}

// AddToken registers a simulated token.
func (s *Server) AddToken(token, sellerNTN string, sandbox bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Tokens[token] = TokenInfo{SellerNTNCNIC: sellerNTN, Sandbox: sandbox}
}

// InjectFault queues a fault for the next invoice call(s).
func (s *Server) InjectFault(k FaultKind, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < count; i++ {
		s.faults = append(s.faults, k)
	}
}

// Invoice returns a recorded invoice (for tests).
func (s *Server) Invoice(fbrNo string) (*fbr.InvoicePayload, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.invoices[fbrNo]
	return p, ok
}

// Count returns how many invoices were recorded.
func (s *Server) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.invoices)
}

func (s *Server) nextFault() FaultKind {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.faults) == 0 {
		return FaultNone
	}
	f := s.faults[0]
	s.faults = s.faults[1:]
	return f
}

// Handler returns the HTTP handler implementing the DI endpoints.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /di_data/v1/di/postinvoicedata", s.invoice(false, true))
	mux.HandleFunc("POST /di_data/v1/di/postinvoicedata_sb", s.invoice(true, true))
	mux.HandleFunc("POST /di_data/v1/di/validateinvoicedata", s.invoice(false, false))
	mux.HandleFunc("POST /di_data/v1/di/validateinvoicedata_sb", s.invoice(true, false))
	mux.HandleFunc("GET /pdi/v1/provinces", s.auth(s.provinces))
	mux.HandleFunc("GET /pdi/v1/doctypecode", s.auth(s.docTypes))
	mux.HandleFunc("GET /pdi/v1/itemdesccode", s.auth(s.itemDescCodes))
	mux.HandleFunc("GET /pdi/v1/sroitemcode", s.auth(s.sroItemCodes))
	mux.HandleFunc("GET /pdi/v1/transtypecode", s.auth(s.transTypeCodes))
	mux.HandleFunc("GET /pdi/v1/uom", s.auth(s.uoms))
	mux.HandleFunc("GET /pdi/v1/SroSchedule", s.auth(s.sroSchedule))
	mux.HandleFunc("GET /pdi/v2/SaleTypeToRate", s.auth(s.saleTypeToRate))
	mux.HandleFunc("GET /pdi/v2/HS_UOM", s.auth(s.hsUOM))
	mux.HandleFunc("GET /pdi/v2/SROItem", s.auth(s.sroItems))
	mux.HandleFunc("POST /dist/v1/statl", s.auth(s.statl))
	mux.HandleFunc("POST /dist/v1/Get_Reg_Type", s.auth(s.regType))
	mux.HandleFunc("POST /mock/fault", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Kind  FaultKind `json:"kind"`
			Count int       `json:"count"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Count <= 0 {
			req.Count = 1
		}
		s.InjectFault(req.Kind, req.Count)
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

// Transport returns an http.RoundTripper that serves requests directly from
// the simulator without any network (used by the Training Simulator).
func (s *Server) Transport() http.RoundTripper {
	h := s.Handler()
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Result(), nil
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (s *Server) tokenInfo(r *http.Request) (TokenInfo, bool) {
	tok := bearer(r)
	if tok == "" {
		return TokenInfo{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if ti, ok := s.Tokens[tok]; ok {
		return ti, true
	}
	if s.AcceptAnyToken {
		return TokenInfo{}, true
	}
	return TokenInfo{}, false
}

func unauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]any{
		"fault": map[string]any{"code": 900901, "message": "Invalid Credentials", "description": "Invalid Credentials. Make sure you have provided the correct security credentials"},
	})
}

func (s *Server) auth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.tokenInfo(r); !ok {
			unauthorized(w)
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type itemStatus struct {
	ItemSNo    string `json:"itemSNo"`
	StatusCode string `json:"statusCode"`
	Status     string `json:"status"`
	InvoiceNo  string `json:"invoiceNo"`
	ErrorCode  string `json:"errorCode"`
	Error      string `json:"error"`
}

type validationResponse struct {
	StatusCode      string       `json:"statusCode"`
	Status          string       `json:"status"`
	ErrorCode       string       `json:"errorCode,omitempty"`
	Error           string       `json:"error"`
	InvoiceStatuses []itemStatus `json:"invoiceStatuses"`
}

type invoiceResponse struct {
	InvoiceNumber      string             `json:"invoiceNumber,omitempty"`
	Dated              string             `json:"dated"`
	ValidationResponse validationResponse `json:"validationResponse"`
}

var reDigits = regexp.MustCompile(`^\d+$`)

func validRegNo(s string) bool {
	if !reDigits.MatchString(s) {
		return false
	}
	return len(s) == 7 || len(s) == 9 || len(s) == 13
}

func (s *Server) invoice(sandbox, post bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fault := s.nextFault()
		if fault == FaultUnauthorized {
			unauthorized(w)
			return
		}
		ti, ok := s.tokenInfo(r)
		if !ok {
			unauthorized(w)
			return
		}
		if fault == FaultUnavailable {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var p fbr.InvoicePayload
		now := s.Now()
		dated := now.Format("2006-01-02 15:04:05")
		if err := fbr.DecodeLenient(body, &p); err != nil {
			writeJSON(w, http.StatusOK, invoiceResponse{Dated: dated, ValidationResponse: validationResponse{
				StatusCode: "01", Status: "Invalid", ErrorCode: "0000", Error: "Invalid JSON payload: " + err.Error()}})
			return
		}
		if fault == FaultServerError {
			http.Error(w, `{"message":"Internal Server Error"}`, http.StatusInternalServerError)
			return
		}

		// Header validation (stops at the first header error, as FBR does).
		if code, msg := s.headerError(&p, ti, sandbox, now); code != "" {
			writeJSON(w, http.StatusOK, invoiceResponse{Dated: dated, ValidationResponse: validationResponse{
				StatusCode: "01", Status: "Invalid", ErrorCode: code, Error: msg, InvoiceStatuses: nil}})
			return
		}

		statuses := make([]itemStatus, 0, len(p.Items))
		allValid := true
		for i := range p.Items {
			code, msg := itemError(&p, &p.Items[i])
			st := itemStatus{ItemSNo: strconv.Itoa(i + 1), StatusCode: "00", Status: "Valid"}
			if code != "" {
				allValid = false
				st.StatusCode, st.Status, st.ErrorCode, st.Error = "01", "Invalid", code, msg
			}
			statuses = append(statuses, st)
		}
		resp := invoiceResponse{Dated: dated, ValidationResponse: validationResponse{StatusCode: "00", Status: "Valid", InvoiceStatuses: statuses}}
		if !allValid {
			resp.ValidationResponse.StatusCode = "01"
			resp.ValidationResponse.Status = "Invalid"
			writeJSON(w, http.StatusOK, resp)
			return
		}
		if post {
			s.mu.Lock()
			s.seq++
			ms := now.UnixMilli() + s.seq
			s.mu.Unlock()
			invNo := fmt.Sprintf("%sDI%d", p.SellerNTNCNIC, ms)
			resp.InvoiceNumber = invNo
			for i := range resp.ValidationResponse.InvoiceStatuses {
				resp.ValidationResponse.InvoiceStatuses[i].InvoiceNo = fmt.Sprintf("%s-%d", invNo, i+1)
			}
			s.mu.Lock()
			cp := p
			s.invoices[invNo] = &cp
			s.invoiceDates[invNo] = now
			s.mu.Unlock()
		}
		switch fault {
		case FaultTimeout:
			time.Sleep(s.TimeoutDelay)
		case FaultDrop:
			if hj, ok := w.(http.Hijacker); ok {
				if conn, _, err := hj.Hijack(); err == nil {
					_ = conn.Close()
					return
				}
			}
			// Without a hijackable connection (in-memory transport) emulate
			// a broken response body.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"invoiceNumber":"`))
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func (s *Server) headerError(p *fbr.InvoicePayload, ti TokenInfo, sandbox bool, now time.Time) (string, string) {
	if !validRegNo(p.SellerNTNCNIC) {
		return "0401", "Unauthorized access: Provided seller registration number is not 13 digits (CNIC) or 7 digits (NTN) or the authorized token does not exist against seller registration number"
	}
	if ti.SellerNTNCNIC != "" && ti.SellerNTNCNIC != p.SellerNTNCNIC {
		return "0401", "Unauthorized access: Provided seller registration number is not 13 digits (CNIC) or 7 digits (NTN) or the authorized token does not exist against seller registration number"
	}
	if ti.SellerNTNCNIC != "" && ti.Sandbox != sandbox {
		return "0401", "Unauthorized access: the security token is not valid for this environment"
	}
	if p.InvoiceType == "" {
		return "0011", "Provide invoice type."
	}
	if !domain.DocType(p.InvoiceType).Valid() {
		return "0003", "Provided invoice type is not valid. Please refer to relevant reference API in the technical document for DI API for valid invoice types."
	}
	if p.InvoiceDate == "" {
		return "0042", "Provide invoice date."
	}
	d, err := time.Parse("2006-01-02", p.InvoiceDate)
	if err != nil {
		return "0005", "Provide invoice date in YYYY-MM-DD format."
	}
	if d.After(now.Add(24 * time.Hour)) {
		return "0043", "Invoice date cannot be in the future."
	}
	if strings.TrimSpace(p.SellerProvince) == "" {
		return "0073", "Provide Sale Origination Province of Supplier."
	}
	rt := domain.RegistrationType(p.BuyerRegistrationType)
	if p.BuyerRegistrationType == "" || !rt.Valid() {
		return "0012", "Provided buyer registration type is not valid. Please refer to relevant reference API in the technical document for DI API for valid buyer registration types."
	}
	if rt == domain.Registered && p.BuyerNTNCNIC == "" {
		return "0009", "Provide Buyer Registration No."
	}
	if p.BuyerNTNCNIC != "" && !validRegNo(p.BuyerNTNCNIC) {
		return "0002", "Buyer Registration No. is not in proper format. Please provide 13 digit CNIC or 7 digit NTN."
	}
	if p.BuyerNTNCNIC != "" && p.BuyerNTNCNIC == p.SellerNTNCNIC {
		return "0058", "Buyer and seller registration number cannot be the same."
	}
	if strings.TrimSpace(p.BuyerBusinessName) == "" {
		return "0010", "Provide Buyer Name."
	}
	if strings.TrimSpace(p.BuyerProvince) == "" {
		return "0074", "Provide Destination of Supply."
	}
	if sandbox && p.ScenarioID == "" {
		return "0000", "Provide scenario id for sandbox testing."
	}
	if domain.DocType(p.InvoiceType) == domain.DocDebitNote {
		if p.InvoiceRefNo == "" {
			return "0041", "Provide reference invoice number for debit note."
		}
		s.mu.Lock()
		orig, ok := s.invoices[p.InvoiceRefNo]
		origDate := s.invoiceDates[p.InvoiceRefNo]
		s.mu.Unlock()
		if !ok {
			return "0006", "Sale invoice does not exist against the provided reference number."
		}
		if orig.SellerNTNCNIC != p.SellerNTNCNIC {
			return "0006", "Sale invoice does not exist against the provided reference number."
		}
		if d.Before(time.Date(origDate.Year(), origDate.Month(), origDate.Day(), 0, 0, 0, 0, time.UTC)) {
			if od, err := time.Parse("2006-01-02", orig.InvoiceDate); err == nil && d.Before(od) {
				return "0035", "Debit note date cannot be earlier than the original invoice date."
			}
		}
	}
	if len(p.Items) == 0 {
		return "0000", "Provide at least one item."
	}
	return "", ""
}

var reHS = regexp.MustCompile(`^\d{4}\.\d{4}$`)

func itemError(p *fbr.InvoicePayload, it *fbr.ItemPayload) (string, string) {
	if strings.TrimSpace(it.HSCode) == "" {
		return "0019", "Provide HS Code."
	}
	if !reHS.MatchString(it.HSCode) {
		return "0052", "Provide proper HS Code."
	}
	if strings.TrimSpace(it.SaleType) == "" {
		return "0013", "Provide sale type."
	}
	st, known := domain.LookupSaleType(it.SaleType)
	if !known || st.Name != it.SaleType {
		return "0013", "Provided sale type is not valid. Please refer to relevant reference API in the technical document for DI API for valid sale types."
	}
	if strings.TrimSpace(it.Rate) == "" {
		return "0046", "Provide rate."
	}
	rate := tax.ParseRate(it.Rate)
	if !rate.Valid || !rateAllowed(st.Name, it.Rate) {
		return "0046", "Provided rate is not valid for the selected sale type. Please refer to SaleTypeToRate reference API."
	}
	if strings.TrimSpace(it.UoM) == "" {
		return "0023", "Provide UoM."
	}
	if strings.TrimSpace(it.ProductDescription) == "" {
		return "0024", "Provide product description."
	}
	if st.SRORequired && strings.TrimSpace(it.SROScheduleNo) == "" {
		return "0077", "Valid SRO/Schedule No. is mandatory where rate is not 18%."
	}
	if st.SRORequired && strings.TrimSpace(it.SROItemSerialNo) == "" {
		return "0078", "Provide SRO item serial no."
	}
	if st.ExtraTaxMustBeEmpty && !it.ExtraTax.Empty {
		return "0091", "Extra tax provided where sale is of reduced rate goods. Please verify if provided sale type is for Goods at reduced rate."
	}
	if st.Name == domain.STPotassiumChlor && it.UoM != "KG" {
		return "0165", "Provide UoM as KG."
	}
	if st.Basis == domain.BasisRetailPrice && it.FixedNotifiedValueOrRetailPrice.Value.IsZero() {
		return "0175", "Provide fixed / notified value or retail price."
	}
	base := it.ValueSalesExcludingST.Value
	if st.Basis == domain.BasisRetailPrice {
		base = it.FixedNotifiedValueOrRetailPrice.Value
	}
	expected := tax.R2(tax.PercentOf(base, rate.Percent).Add(it.Quantity.Value.Mul(rate.PerUnit)))
	if rate.Exempt {
		expected = decimal.Zero
	}
	if it.SalesTaxApplicable.Value.Sub(expected).Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		return "0102", "Provided sales tax amount does not match the calculated sales tax amount. Please ensure that the provided Sale Value is used to calculate the Sales Tax Amount for the provided Rate."
	}
	if it.SalesTaxWithheldAtSource.Value.IsPositive() && it.SalesTaxWithheldAtSource.Value.GreaterThan(it.SalesTaxApplicable.Value) && !rate.IsZeroRate() {
		return "0008", "ST withheld at source should either be zero or same as sales tax/fed in ST mode."
	}
	if domain.RegistrationType(p.BuyerRegistrationType) == domain.Registered && it.FurtherTax.Value.IsPositive() {
		return "0028", "Further tax is not applicable on supplies to registered persons."
	}
	return "", ""
}

func rateAllowed(saleType, rate string) bool {
	rows, ok := ratesBySaleType[saleType]
	if !ok {
		return true
	}
	for _, r := range rows {
		if strings.EqualFold(strings.TrimSpace(r.Desc), strings.TrimSpace(rate)) {
			return true
		}
	}
	return false
}

// --- reference endpoints ---

func (s *Server) provinces(w http.ResponseWriter, r *http.Request) {
	out := make([]map[string]any, 0, len(domain.Provinces))
	for _, p := range domain.Provinces {
		out = append(out, map[string]any{"stateProvinceCode": p.Code, "stateProvinceDesc": p.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) docTypes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]any{
		{"docTypeId": 4, "docDescription": "Sale Invoice"},
		{"docTypeId": 9, "docDescription": "Debit Note"},
	})
}

func (s *Server) itemDescCodes(w http.ResponseWriter, r *http.Request) {
	out := make([]map[string]any, 0, len(hsCodes))
	for _, h := range hsCodes {
		out = append(out, map[string]any{"hS_CODE": h.Code, "description": h.Desc})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) sroItemCodes(w http.ResponseWriter, r *http.Request) {
	var out []map[string]any
	for _, items := range itemsBySRO {
		for _, it := range items {
			out = append(out, map[string]any{"srO_ITEM_ID": it.ID, "srO_ITEM_DESC": it.Desc})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) transTypeCodes(w http.ResponseWriter, r *http.Request) {
	out := make([]map[string]any, 0, len(transTypes))
	for _, t := range transTypes {
		out = append(out, map[string]any{"transactioN_TYPE_ID": t.ID, "transactioN_DESC": t.Desc})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) uoms(w http.ResponseWriter, r *http.Request) {
	out := make([]map[string]any, 0, len(domain.UOMs))
	for i, u := range domain.UOMs {
		out = append(out, map[string]any{"uoM_ID": i + 1, "description": u})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) sroSchedule(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("rate_id"))
	out := []map[string]any{}
	for _, sc := range schedulesByRate[id] {
		out = append(out, map[string]any{"srO_ID": sc.ID, "serNo": sc.SerNo, "srO_DESC": sc.Desc})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) saleTypeToRate(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("transTypeId"))
	out := []map[string]any{}
	for _, t := range transTypes {
		if t.ID == id {
			for _, rr := range ratesBySaleType[t.Desc] {
				out = append(out, map[string]any{"ratE_ID": rr.ID, "ratE_DESC": rr.Desc, "ratE_VALUE": rr.Value})
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) hsUOM(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("hs_code")
	out := []map[string]any{}
	for _, h := range hsCodes {
		if h.Code == code {
			for i, u := range domain.UOMs {
				if u == h.UOM {
					out = append(out, map[string]any{"uoM_ID": i + 1, "description": u})
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) sroItems(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("sro_id"))
	out := []map[string]any{}
	for _, it := range itemsBySRO[id] {
		out = append(out, map[string]any{"srO_ITEM_ID": it.ID, "srO_ITEM_DESC": it.Desc})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) statl(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RegNo string `json:"regno"`
		Date  string `json:"date"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if registry[req.RegNo] {
		writeJSON(w, http.StatusOK, map[string]string{"status code": "01", "status": "Active"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status code": "02", "status": "In-Active"})
}

func (s *Server) regType(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RegNo string `json:"Registration_No"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	t := "Unregistered"
	if registry[req.RegNo] {
		t = "Registered"
	}
	writeJSON(w, http.StatusOK, map[string]string{"statuscode": "00", "REGISTRATION_NO": req.RegNo, "REGISTRATION_TYPE": t})
}
