-- Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
-- Veridian E-invoicing PK is proprietary software; see the LICENSE file.

-- Initial schema for Veridian E-invoicing PK.
-- Monetary amounts are stored as INTEGER paisa (1 rupee = 100) so SQL sums
-- are exact. Quantities and unit prices are stored as decimal TEXT.
-- Timestamps are RFC 3339 UTC text; dates are YYYY-MM-DD.

CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE companies (
  id                     INTEGER PRIMARY KEY AUTOINCREMENT,
  name                   TEXT NOT NULL,
  ntn_cnic               TEXT NOT NULL,
  strn                   TEXT NOT NULL DEFAULT '',
  province               TEXT NOT NULL,
  province_code          INTEGER NOT NULL DEFAULT 0,
  address                TEXT NOT NULL,
  city                   TEXT NOT NULL DEFAULT '',
  phone                  TEXT NOT NULL DEFAULT '',
  email                  TEXT NOT NULL DEFAULT '',
  business_activities    TEXT NOT NULL DEFAULT '[]',
  sectors                TEXT NOT NULL DEFAULT '[]',
  assigned_scenarios     TEXT NOT NULL DEFAULT '[]',
  environment            TEXT NOT NULL DEFAULT 'simulator',
  sandbox_token_enc      TEXT NOT NULL DEFAULT '',
  production_token_enc   TEXT NOT NULL DEFAULT '',
  sandbox_token_expiry   TEXT NOT NULL DEFAULT '',
  production_token_expiry TEXT NOT NULL DEFAULT '',
  invoice_prefix         TEXT NOT NULL DEFAULT 'INV',
  debit_note_prefix      TEXT NOT NULL DEFAULT 'DN',
  further_tax_rate       TEXT NOT NULL DEFAULT '4',
  withholding_fraction   TEXT NOT NULL DEFAULT '0.2',
  send_internal_ref      INTEGER NOT NULL DEFAULT 0,
  validate_before_post   INTEGER NOT NULL DEFAULT 1,
  logo                   BLOB,
  logo_mime              TEXT NOT NULL DEFAULT '',
  print_settings         TEXT NOT NULL DEFAULT '{}',
  active                 INTEGER NOT NULL DEFAULT 1,
  created_at             TEXT NOT NULL,
  updated_at             TEXT NOT NULL
);
CREATE UNIQUE INDEX ux_companies_ntn ON companies(ntn_cnic);

CREATE TABLE users (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  username             TEXT NOT NULL UNIQUE COLLATE NOCASE,
  full_name            TEXT NOT NULL DEFAULT '',
  email                TEXT NOT NULL DEFAULT '',
  password_hash        TEXT NOT NULL,
  role                 TEXT NOT NULL,
  all_companies        INTEGER NOT NULL DEFAULT 0,
  active               INTEGER NOT NULL DEFAULT 1,
  must_change_password INTEGER NOT NULL DEFAULT 0,
  failed_logins        INTEGER NOT NULL DEFAULT 0,
  locked_until         TEXT NOT NULL DEFAULT '',
  last_login_at        TEXT NOT NULL DEFAULT '',
  created_at           TEXT NOT NULL,
  updated_at           TEXT NOT NULL
);

CREATE TABLE user_companies (
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  company_id INTEGER NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, company_id)
);

CREATE TABLE sessions (
  token_hash   TEXT PRIMARY KEY,
  user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  csrf_token   TEXT NOT NULL,
  created_at   TEXT NOT NULL,
  expires_at   TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  ip           TEXT NOT NULL DEFAULT '',
  user_agent   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX ix_sessions_user ON sessions(user_id);

CREATE TABLE api_keys (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id   INTEGER NOT NULL REFERENCES companies(id),
  name         TEXT NOT NULL,
  prefix       TEXT NOT NULL,
  key_hash     TEXT NOT NULL UNIQUE,
  scopes       TEXT NOT NULL DEFAULT 'invoices',
  created_by   INTEGER,
  created_at   TEXT NOT NULL,
  last_used_at TEXT NOT NULL DEFAULT '',
  revoked_at   TEXT NOT NULL DEFAULT ''
);

CREATE TABLE customers (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id        INTEGER NOT NULL REFERENCES companies(id),
  code              TEXT NOT NULL DEFAULT '',
  name              TEXT NOT NULL,
  ntn_cnic          TEXT NOT NULL DEFAULT '',
  strn              TEXT NOT NULL DEFAULT '',
  registration_type TEXT NOT NULL DEFAULT 'Unregistered',
  province          TEXT NOT NULL DEFAULT '',
  address           TEXT NOT NULL DEFAULT '',
  city              TEXT NOT NULL DEFAULT '',
  phone             TEXT NOT NULL DEFAULT '',
  email             TEXT NOT NULL DEFAULT '',
  withholding_mode  TEXT NOT NULL DEFAULT '',
  statl_status      TEXT NOT NULL DEFAULT '',
  fbr_reg_type      TEXT NOT NULL DEFAULT '',
  status_checked_at TEXT NOT NULL DEFAULT '',
  notes             TEXT NOT NULL DEFAULT '',
  active            INTEGER NOT NULL DEFAULT 1,
  created_at        TEXT NOT NULL,
  updated_at        TEXT NOT NULL
);
CREATE INDEX ix_customers_company ON customers(company_id, name);
CREATE INDEX ix_customers_ntn ON customers(company_id, ntn_cnic);

CREATE TABLE products (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id         INTEGER NOT NULL REFERENCES companies(id),
  code               TEXT NOT NULL DEFAULT '',
  description        TEXT NOT NULL,
  hs_code            TEXT NOT NULL,
  uom                TEXT NOT NULL,
  sale_type          TEXT NOT NULL,
  rate               TEXT NOT NULL,
  sro_schedule_no    TEXT NOT NULL DEFAULT '',
  sro_item_serial_no TEXT NOT NULL DEFAULT '',
  unit_price         TEXT NOT NULL DEFAULT '0',
  retail_price       TEXT NOT NULL DEFAULT '0',
  further_tax_mode   TEXT NOT NULL DEFAULT 'auto',
  extra_tax_rate     TEXT NOT NULL DEFAULT '0',
  fed_rate           TEXT NOT NULL DEFAULT '0',
  active             INTEGER NOT NULL DEFAULT 1,
  created_at         TEXT NOT NULL,
  updated_at         TEXT NOT NULL
);
CREATE INDEX ix_products_company ON products(company_id, description);

CREATE TABLE invoices (
  id                      INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id              INTEGER NOT NULL REFERENCES companies(id),
  environment             TEXT NOT NULL,
  doc_type                TEXT NOT NULL,
  internal_no             TEXT NOT NULL,
  invoice_date            TEXT NOT NULL,
  status                  TEXT NOT NULL,
  customer_id             INTEGER REFERENCES customers(id),
  seller_ntn_cnic         TEXT NOT NULL,
  seller_name             TEXT NOT NULL,
  seller_province         TEXT NOT NULL,
  seller_address          TEXT NOT NULL,
  buyer_ntn_cnic          TEXT NOT NULL DEFAULT '',
  buyer_name              TEXT NOT NULL,
  buyer_province          TEXT NOT NULL,
  buyer_address           TEXT NOT NULL DEFAULT '',
  buyer_registration_type TEXT NOT NULL,
  withholding_mode        TEXT NOT NULL DEFAULT '',
  invoice_ref_no          TEXT NOT NULL DEFAULT '',
  ref_invoice_id          INTEGER REFERENCES invoices(id),
  scenario_id             TEXT NOT NULL DEFAULT '',
  external_ref            TEXT NOT NULL DEFAULT '',
  source                  TEXT NOT NULL DEFAULT 'ui',
  notes                   TEXT NOT NULL DEFAULT '',
  total_gross             INTEGER NOT NULL DEFAULT 0,
  total_discount          INTEGER NOT NULL DEFAULT 0,
  total_value_excl_st     INTEGER NOT NULL DEFAULT 0,
  total_retail_value      INTEGER NOT NULL DEFAULT 0,
  total_sales_tax         INTEGER NOT NULL DEFAULT 0,
  total_further_tax       INTEGER NOT NULL DEFAULT 0,
  total_extra_tax         INTEGER NOT NULL DEFAULT 0,
  total_fed               INTEGER NOT NULL DEFAULT 0,
  total_st_withheld       INTEGER NOT NULL DEFAULT 0,
  total_value             INTEGER NOT NULL DEFAULT 0,
  amount_payable          INTEGER NOT NULL DEFAULT 0,
  fbr_invoice_number      TEXT NOT NULL DEFAULT '',
  fbr_dated               TEXT NOT NULL DEFAULT '',
  fbr_status_code         TEXT NOT NULL DEFAULT '',
  fbr_errors              TEXT NOT NULL DEFAULT '',
  last_error              TEXT NOT NULL DEFAULT '',
  validation_json         TEXT NOT NULL DEFAULT '',
  submit_attempts         INTEGER NOT NULL DEFAULT 0,
  next_attempt_at         TEXT NOT NULL DEFAULT '',
  payload_json            TEXT NOT NULL DEFAULT '',
  payload_hash            TEXT NOT NULL DEFAULT '',
  seal_hash               TEXT NOT NULL DEFAULT '',
  prev_seal_hash          TEXT NOT NULL DEFAULT '',
  print_count             INTEGER NOT NULL DEFAULT 0,
  cancelled_at            TEXT NOT NULL DEFAULT '',
  cancel_reason           TEXT NOT NULL DEFAULT '',
  cancel_reference        TEXT NOT NULL DEFAULT '',
  created_by              INTEGER,
  updated_by              INTEGER,
  created_at              TEXT NOT NULL,
  updated_at              TEXT NOT NULL,
  submitted_at            TEXT NOT NULL DEFAULT '',
  accepted_at             TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX ux_invoices_no ON invoices(company_id, environment, internal_no);
CREATE UNIQUE INDEX ux_invoices_extref ON invoices(company_id, environment, external_ref) WHERE external_ref <> '';
CREATE UNIQUE INDEX ux_invoices_fbr ON invoices(fbr_invoice_number) WHERE fbr_invoice_number <> '';
CREATE INDEX ix_invoices_status ON invoices(company_id, environment, status);
CREATE INDEX ix_invoices_date ON invoices(company_id, environment, invoice_date);
CREATE INDEX ix_invoices_queue ON invoices(status, next_attempt_at);

CREATE TABLE invoice_items (
  id                    INTEGER PRIMARY KEY AUTOINCREMENT,
  invoice_id            INTEGER NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
  line_no               INTEGER NOT NULL,
  product_id            INTEGER REFERENCES products(id),
  hs_code               TEXT NOT NULL,
  description           TEXT NOT NULL,
  uom                   TEXT NOT NULL,
  quantity              TEXT NOT NULL,
  unit_price            TEXT NOT NULL,
  discount_percent      TEXT NOT NULL DEFAULT '0',
  discount_amount       TEXT NOT NULL DEFAULT '0',
  value_override        TEXT NOT NULL DEFAULT '',
  sale_type             TEXT NOT NULL,
  rate                  TEXT NOT NULL,
  retail_price          TEXT NOT NULL DEFAULT '0',
  retail_value_override TEXT NOT NULL DEFAULT '',
  further_tax_mode      TEXT NOT NULL DEFAULT 'auto',
  further_tax_override  TEXT NOT NULL DEFAULT '',
  extra_tax_rate        TEXT NOT NULL DEFAULT '0',
  extra_tax_override    TEXT NOT NULL DEFAULT '',
  fed_rate              TEXT NOT NULL DEFAULT '0',
  fed_override          TEXT NOT NULL DEFAULT '',
  withholding_override  TEXT NOT NULL DEFAULT '',
  sales_tax_override    TEXT NOT NULL DEFAULT '',
  sro_schedule_no       TEXT NOT NULL DEFAULT '',
  sro_item_serial_no    TEXT NOT NULL DEFAULT '',
  gross                 INTEGER NOT NULL DEFAULT 0,
  discount              INTEGER NOT NULL DEFAULT 0,
  value_excl_st         INTEGER NOT NULL DEFAULT 0,
  retail_value          INTEGER NOT NULL DEFAULT 0,
  sales_tax             INTEGER NOT NULL DEFAULT 0,
  further_tax           INTEGER NOT NULL DEFAULT 0,
  extra_tax             INTEGER NOT NULL DEFAULT 0,
  extra_tax_empty       INTEGER NOT NULL DEFAULT 0,
  fed                   INTEGER NOT NULL DEFAULT 0,
  st_withheld           INTEGER NOT NULL DEFAULT 0,
  total_value           INTEGER NOT NULL DEFAULT 0,
  fbr_item_invoice_no   TEXT NOT NULL DEFAULT '',
  fbr_status_code       TEXT NOT NULL DEFAULT '',
  fbr_error_code        TEXT NOT NULL DEFAULT '',
  fbr_error             TEXT NOT NULL DEFAULT ''
);
CREATE INDEX ix_items_invoice ON invoice_items(invoice_id, line_no);

CREATE TABLE number_series (
  company_id  INTEGER NOT NULL,
  environment TEXT NOT NULL,
  doc_type    TEXT NOT NULL,
  fiscal_year TEXT NOT NULL,
  next_no     INTEGER NOT NULL,
  PRIMARY KEY (company_id, environment, doc_type, fiscal_year)
);

CREATE TABLE fbr_calls (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id    INTEGER NOT NULL,
  invoice_id    INTEGER,
  environment   TEXT NOT NULL,
  operation     TEXT NOT NULL,
  method        TEXT NOT NULL,
  url           TEXT NOT NULL,
  request_body  TEXT NOT NULL DEFAULT '',
  response_body TEXT NOT NULL DEFAULT '',
  http_status   INTEGER NOT NULL DEFAULT 0,
  duration_ms   INTEGER NOT NULL DEFAULT 0,
  error_kind    TEXT NOT NULL DEFAULT '',
  error         TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL
);
CREATE INDEX ix_fbr_calls_invoice ON fbr_calls(invoice_id);
CREATE INDEX ix_fbr_calls_company ON fbr_calls(company_id, created_at);

CREATE TABLE ref_cache (
  cache_key  TEXT PRIMARY KEY,
  kind       TEXT NOT NULL,
  data       TEXT NOT NULL,
  source     TEXT NOT NULL,
  fetched_at TEXT NOT NULL
);

CREATE TABLE ref_hs_codes (
  code        TEXT PRIMARY KEY,
  description TEXT NOT NULL,
  source      TEXT NOT NULL
);

CREATE TABLE audit_log (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  ts         TEXT NOT NULL,
  user_id    INTEGER,
  username   TEXT NOT NULL DEFAULT '',
  company_id INTEGER,
  action     TEXT NOT NULL,
  entity     TEXT NOT NULL DEFAULT '',
  entity_id  TEXT NOT NULL DEFAULT '',
  details    TEXT NOT NULL DEFAULT '',
  ip         TEXT NOT NULL DEFAULT '',
  prev_hash  TEXT NOT NULL,
  hash       TEXT NOT NULL
);
CREATE INDEX ix_audit_company ON audit_log(company_id, ts);

CREATE TABLE scenario_runs (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id         INTEGER NOT NULL,
  scenario_id        TEXT NOT NULL,
  invoice_id         INTEGER,
  mode               TEXT NOT NULL,
  status             TEXT NOT NULL,
  fbr_invoice_number TEXT NOT NULL DEFAULT '',
  message            TEXT NOT NULL DEFAULT '',
  run_at             TEXT NOT NULL,
  run_by             INTEGER
);
CREATE INDEX ix_scenario_runs ON scenario_runs(company_id, scenario_id, run_at);

CREATE TABLE incidents (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id       INTEGER NOT NULL,
  kind             TEXT NOT NULL,
  description      TEXT NOT NULL,
  started_at       TEXT NOT NULL,
  ended_at         TEXT NOT NULL DEFAULT '',
  auto_detected    INTEGER NOT NULL DEFAULT 0,
  reported_at      TEXT NOT NULL DEFAULT '',
  report_reference TEXT NOT NULL DEFAULT '',
  created_by       INTEGER,
  created_at       TEXT NOT NULL,
  updated_at       TEXT NOT NULL
);
CREATE INDEX ix_incidents_company ON incidents(company_id, started_at);

-- Integrity guards ---------------------------------------------------------

-- Invoices accepted by FBR are immutable: only status (to CANCELLED), the
-- cancellation fields, print counter and timestamps may change.
CREATE TRIGGER trg_invoices_immutable
BEFORE UPDATE ON invoices
WHEN OLD.status IN ('ACCEPTED', 'CANCELLED') AND (
     NEW.internal_no IS NOT OLD.internal_no
  OR NEW.doc_type IS NOT OLD.doc_type
  OR NEW.environment IS NOT OLD.environment
  OR NEW.invoice_date IS NOT OLD.invoice_date
  OR NEW.seller_ntn_cnic IS NOT OLD.seller_ntn_cnic
  OR NEW.buyer_ntn_cnic IS NOT OLD.buyer_ntn_cnic
  OR NEW.buyer_name IS NOT OLD.buyer_name
  OR NEW.buyer_registration_type IS NOT OLD.buyer_registration_type
  OR NEW.total_value_excl_st IS NOT OLD.total_value_excl_st
  OR NEW.total_sales_tax IS NOT OLD.total_sales_tax
  OR NEW.total_value IS NOT OLD.total_value
  OR NEW.fbr_invoice_number IS NOT OLD.fbr_invoice_number
  OR NEW.fbr_dated IS NOT OLD.fbr_dated
  OR NEW.payload_json IS NOT OLD.payload_json
  OR NEW.payload_hash IS NOT OLD.payload_hash
  OR NEW.seal_hash IS NOT OLD.seal_hash
  OR (OLD.status = 'CANCELLED' AND NEW.status IS NOT OLD.status)
  OR (OLD.status = 'ACCEPTED' AND NEW.status NOT IN ('ACCEPTED', 'CANCELLED'))
)
BEGIN
  SELECT RAISE(ABORT, 'invoice accepted by FBR is immutable');
END;

CREATE TRIGGER trg_invoices_nodelete
BEFORE DELETE ON invoices
WHEN OLD.status NOT IN ('DRAFT', 'VALIDATED', 'REJECTED') OR OLD.fbr_invoice_number <> ''
BEGIN
  SELECT RAISE(ABORT, 'invoices reported to FBR cannot be deleted');
END;

CREATE TRIGGER trg_items_immutable
BEFORE UPDATE ON invoice_items
WHEN (SELECT status FROM invoices WHERE id = OLD.invoice_id) IN ('ACCEPTED', 'CANCELLED')
BEGIN
  SELECT RAISE(ABORT, 'lines of an invoice accepted by FBR are immutable');
END;

CREATE TRIGGER trg_items_nodelete
BEFORE DELETE ON invoice_items
WHEN (SELECT status FROM invoices WHERE id = OLD.invoice_id) IN ('ACCEPTED', 'CANCELLED', 'UNCERTAIN', 'SUBMITTING', 'QUEUED')
BEGIN
  SELECT RAISE(ABORT, 'lines of a reported invoice cannot be deleted');
END;

CREATE TRIGGER trg_audit_noupdate BEFORE UPDATE ON audit_log
BEGIN
  SELECT RAISE(ABORT, 'audit log is append-only');
END;

CREATE TRIGGER trg_audit_nodelete BEFORE DELETE ON audit_log
BEGIN
  SELECT RAISE(ABORT, 'audit log is append-only');
END;
