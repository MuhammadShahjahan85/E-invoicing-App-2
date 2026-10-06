package fbr

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/domain"
)

// Endpoints holds the DI URLs. Defaults follow PRAL's technical specification;
// every value can be overridden in configuration if PRAL changes them.
type Endpoints struct {
	BaseURL             string `json:"baseUrl"`
	PostPath            string `json:"postPath"`
	PostSandboxPath     string `json:"postSandboxPath"`
	ValidatePath        string `json:"validatePath"`
	ValidateSandboxPath string `json:"validateSandboxPath"`
	RefV1Path           string `json:"refV1Path"`
	RefV2Path           string `json:"refV2Path"`
	DistPath            string `json:"distPath"`
	// CancelPath / CancelSandboxPath are optional. STGO 01 of 2026 allows
	// cancellation within 72 hours "through the Board's system"; leave empty
	// to record cancellations made directly on IRIS instead.
	CancelPath        string `json:"cancelPath"`
	CancelSandboxPath string `json:"cancelSandboxPath"`
}

// DefaultEndpoints returns PRAL's published endpoints.
func DefaultEndpoints() Endpoints {
	return Endpoints{
		BaseURL:             "https://gw.fbr.gov.pk",
		PostPath:            "/di_data/v1/di/postinvoicedata",
		PostSandboxPath:     "/di_data/v1/di/postinvoicedata_sb",
		ValidatePath:        "/di_data/v1/di/validateinvoicedata",
		ValidateSandboxPath: "/di_data/v1/di/validateinvoicedata_sb",
		RefV1Path:           "/pdi/v1",
		RefV2Path:           "/pdi/v2",
		DistPath:            "/dist/v1",
	}
}

// Merge fills empty fields of e from d.
func (e Endpoints) Merge(d Endpoints) Endpoints {
	pick := func(a, b string) string {
		if strings.TrimSpace(a) == "" {
			return b
		}
		return a
	}
	return Endpoints{
		BaseURL:             strings.TrimRight(pick(e.BaseURL, d.BaseURL), "/"),
		PostPath:            pick(e.PostPath, d.PostPath),
		PostSandboxPath:     pick(e.PostSandboxPath, d.PostSandboxPath),
		ValidatePath:        pick(e.ValidatePath, d.ValidatePath),
		ValidateSandboxPath: pick(e.ValidateSandboxPath, d.ValidateSandboxPath),
		RefV1Path:           pick(e.RefV1Path, d.RefV1Path),
		RefV2Path:           pick(e.RefV2Path, d.RefV2Path),
		DistPath:            pick(e.DistPath, d.DistPath),
		CancelPath:          pick(e.CancelPath, d.CancelPath),
		CancelSandboxPath:   pick(e.CancelSandboxPath, d.CancelSandboxPath),
	}
}

// ErrKind classifies a failed call so callers can decide whether a retry is
// safe. This matters for postinvoicedata: retrying a request FBR may already
// have processed would report the same sale twice.
type ErrKind string

const (
	// ErrNotSent: the request never reached FBR (DNS, refused connection,
	// TLS failure before the body was written). Safe to retry.
	ErrNotSent ErrKind = "not_sent"
	// ErrUncertain: the request was sent but no usable answer came back
	// (timeout, connection reset, gateway timeout, 500). Reconcile before retrying.
	ErrUncertain ErrKind = "uncertain"
	// ErrUnavailable: FBR answered 502/503 (gateway/service unavailable),
	// which means the request was not processed. Safe to retry later.
	ErrUnavailable ErrKind = "unavailable"
	// ErrAuth: 401/403 — token missing, wrong, expired, or IP not whitelisted.
	ErrAuth ErrKind = "auth"
	// ErrClient: another 4xx; the request itself is wrong.
	ErrClient ErrKind = "client"
	// ErrDecode: a 2xx response that could not be parsed.
	ErrDecode ErrKind = "decode"
)

// CallError is returned for transport-level and HTTP-level failures.
// FBR validation rejections are not errors: they come back as a normal
// InvoiceResponse with IsValid() == false.
type CallError struct {
	Kind       ErrKind
	HTTPStatus int
	Body       string
	Err        error
}

func (e *CallError) Error() string {
	msg := string(e.Kind)
	if e.HTTPStatus != 0 {
		msg += fmt.Sprintf(" (HTTP %d)", e.HTTPStatus)
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	} else if e.Body != "" {
		b := e.Body
		if len(b) > 300 {
			b = b[:300] + "…"
		}
		msg += ": " + b
	}
	return msg
}

func (e *CallError) Unwrap() error { return e.Err }

// Retryable reports whether the same request may be sent again safely.
func (e *CallError) Retryable() bool {
	return e.Kind == ErrNotSent || e.Kind == ErrUnavailable
}

// KindOf extracts the ErrKind of an error, or "" when it is not a CallError.
func KindOf(err error) ErrKind {
	var ce *CallError
	if errors.As(err, &ce) {
		return ce.Kind
	}
	return ""
}

// CallLog describes one HTTP exchange with FBR. The token is never included.
type CallLog struct {
	Operation    string
	Method       string
	URL          string
	RequestBody  []byte
	ResponseBody []byte
	HTTPStatus   int
	Duration     time.Duration
	Err          error
	ErrKind      ErrKind
}

// Client talks to the DI API for one taxpayer (one token) and environment.
type Client struct {
	HTTP      *http.Client
	Endpoints Endpoints
	Token     string
	Env       domain.Environment
	UserAgent string
	// OnCall, when set, receives a log record for every exchange.
	OnCall func(CallLog)
	// Now is the clock (overridable in tests).
	Now func() time.Time
}

// NewHTTPClient builds an http.Client suitable for the PRAL gateway:
// TLS 1.2+, proxy from environment, sane timeouts.
func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConnsPerHost:   4,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{Transport: tr, Timeout: timeout}
}

// New creates a client.
func New(env domain.Environment, token string, ep Endpoints, hc *http.Client) *Client {
	if hc == nil {
		hc = NewHTTPClient(30 * time.Second)
	}
	return &Client{
		HTTP:      hc,
		Endpoints: ep.Merge(DefaultEndpoints()),
		Token:     strings.TrimSpace(token),
		Env:       env,
		UserAgent: brand.ShortName + "/" + brand.Version,
		Now:       time.Now,
	}
}

func (c *Client) url(path string, q url.Values) string {
	u := strings.TrimRight(c.Endpoints.BaseURL, "/") + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u
}

func (c *Client) sandbox() bool { return c.Env == domain.EnvSandbox }

// do performs an HTTP call and classifies failures.
func (c *Client) do(ctx context.Context, op, method, rawURL string, body any) ([]byte, int, error) {
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, 0, &CallError{Kind: ErrClient, Err: err}
		}
	}
	var rd io.Reader
	if reqBody != nil {
		rd = bytes.NewReader(reqBody)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, rd)
	if err != nil {
		return nil, 0, &CallError{Kind: ErrClient, Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", c.UserAgent)

	wrote := false
	trace := &httptrace.ClientTrace{WroteRequest: func(info httptrace.WroteRequestInfo) {
		if info.Err == nil {
			wrote = true
		}
	}}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	start := time.Now()
	resp, err := c.HTTP.Do(req)
	log := CallLog{Operation: op, Method: method, URL: rawURL, RequestBody: reqBody}
	if err != nil {
		kind := ErrUncertain
		if !wrote {
			kind = ErrNotSent
		}
		ce := &CallError{Kind: kind, Err: err}
		log.Duration, log.Err, log.ErrKind = time.Since(start), ce, kind
		c.emit(log)
		return nil, 0, ce
	}
	defer resp.Body.Close()
	respBody, rerr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	log.Duration = time.Since(start)
	log.HTTPStatus = resp.StatusCode
	log.ResponseBody = respBody
	if rerr != nil {
		ce := &CallError{Kind: ErrUncertain, HTTPStatus: resp.StatusCode, Err: rerr}
		log.Err, log.ErrKind = ce, ce.Kind
		c.emit(log)
		return nil, resp.StatusCode, ce
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.emit(log)
		return respBody, resp.StatusCode, nil
	}
	var kind ErrKind
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		kind = ErrAuth
	case resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable:
		kind = ErrUnavailable
	case resp.StatusCode == http.StatusTooManyRequests:
		kind = ErrUnavailable
	case resp.StatusCode >= 500:
		kind = ErrUncertain
	default:
		kind = ErrClient
	}
	ce := &CallError{Kind: kind, HTTPStatus: resp.StatusCode, Body: string(respBody)}
	log.Err, log.ErrKind = ce, kind
	c.emit(log)
	return respBody, resp.StatusCode, ce
}

func (c *Client) emit(l CallLog) {
	if c.OnCall != nil {
		c.OnCall(l)
	}
}

func (c *Client) invoiceCall(ctx context.Context, op, path string, p *InvoicePayload) (*InvoiceResponse, error) {
	if c.Token == "" {
		return nil, &CallError{Kind: ErrAuth, Err: errors.New("no FBR security token configured for this environment")}
	}
	body := *p
	if !c.sandbox() {
		body.ScenarioID = "" // scenarioId is a sandbox-only field
	}
	raw, status, err := c.do(ctx, op, http.MethodPost, c.url(path, nil), &body)
	if err != nil {
		// Some gateways return a JSON validation body with a non-2xx code;
		// surface it if it parses.
		var ce *CallError
		if errors.As(err, &ce) && ce.Kind == ErrClient && len(raw) > 0 {
			var r InvoiceResponse
			if DecodeLenient(raw, &r) == nil && r.ValidationResponse != nil {
				return &r, nil
			}
		}
		return nil, err
	}
	var r InvoiceResponse
	if err := DecodeLenient(raw, &r); err != nil {
		kind := ErrDecode
		if op == "postinvoicedata" {
			kind = ErrUncertain
		}
		return nil, &CallError{Kind: kind, HTTPStatus: status, Body: string(raw), Err: err}
	}
	if r.ValidationResponse == nil {
		kind := ErrDecode
		if op == "postinvoicedata" {
			kind = ErrUncertain
		}
		return nil, &CallError{Kind: kind, HTTPStatus: status, Body: string(raw), Err: errors.New("response has no validationResponse")}
	}
	return &r, nil
}

// PostInvoice reports an invoice (postinvoicedata / postinvoicedata_sb).
// A nil error with r.IsValid()==false is an FBR validation rejection.
func (c *Client) PostInvoice(ctx context.Context, p *InvoicePayload) (*InvoiceResponse, error) {
	path := c.Endpoints.PostPath
	if c.sandbox() {
		path = c.Endpoints.PostSandboxPath
	}
	return c.invoiceCall(ctx, "postinvoicedata", path, p)
}

// ValidateInvoice checks an invoice without recording it
// (validateinvoicedata / validateinvoicedata_sb).
func (c *Client) ValidateInvoice(ctx context.Context, p *InvoicePayload) (*InvoiceResponse, error) {
	path := c.Endpoints.ValidatePath
	if c.sandbox() {
		path = c.Endpoints.ValidateSandboxPath
	}
	return c.invoiceCall(ctx, "validateinvoicedata", path, p)
}

// CancelSupported reports whether a cancellation endpoint is configured.
func (c *Client) CancelSupported() bool {
	if c.sandbox() {
		return c.Endpoints.CancelSandboxPath != ""
	}
	return c.Endpoints.CancelPath != ""
}

// CancelRequest is the body sent to a configured cancellation endpoint.
type CancelRequest struct {
	InvoiceNumber string `json:"invoiceNumber"`
	SellerNTNCNIC string `json:"sellerNTNCNIC"`
	Reason        string `json:"reason"`
}

// CancelInvoice calls the configured cancellation endpoint. The response is
// returned raw because its format is not part of DI API v1.12.
func (c *Client) CancelInvoice(ctx context.Context, req CancelRequest) ([]byte, error) {
	if !c.CancelSupported() {
		return nil, &CallError{Kind: ErrClient, Err: errors.New("cancellation endpoint not configured; cancel the invoice on IRIS and record it here")}
	}
	path := c.Endpoints.CancelPath
	if c.sandbox() {
		path = c.Endpoints.CancelSandboxPath
	}
	raw, _, err := c.do(ctx, "cancelinvoice", http.MethodPost, c.url(path, nil), req)
	return raw, err
}
