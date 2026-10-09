-- Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
-- Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

-- Requirements of Chapter XIV of the Sales Tax Rules, 2006 (SRO 69(I)/2025 as
-- amended by SRO 1666(I)/2026) and Sales Tax General Orders 01 and 25 of 2026.

-- Registration number of the electronic invoicing software, printed on every
-- electronic invoice (rule 150R(13)(c)) and on the "Integrated with FBR"
-- signboard (rule 150R(11)).
ALTER TABLE companies ADD COLUMN software_reg_no TEXT NOT NULL DEFAULT '';

-- Digital signature recorded on each accepted invoice (rule 150R(4)(b)).
ALTER TABLE invoices ADD COLUMN signature TEXT NOT NULL DEFAULT '';

-- Advance receipt invoices: since the Finance Act 2026, section 23(1) of the
-- Sales Tax Act requires an invoice bearing an FBR invoice number for an
-- advance receipt, and rule 150S(2) (SRO 1666(I)/2026) requires it to be
-- issued electronically. advance_ref names, on the final invoice, the advance
-- receipt invoices it adjusts.
ALTER TABLE invoices ADD COLUMN advance_receipt INTEGER NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN advance_ref TEXT NOT NULL DEFAULT '';

-- Federal excise duty particulars added to rule 150R(13) by SRO
-- 1666(I)/2026: (aa) type, (bb) rate, (cc) price per unit, (dd) amount (the
-- line's fed), (ee) SRO/Schedule reference and (ff) serial number. Products
-- carry the defaults.
ALTER TABLE invoice_items ADD COLUMN fed_type TEXT NOT NULL DEFAULT '';
ALTER TABLE invoice_items ADD COLUMN fed_rate_text TEXT NOT NULL DEFAULT '';
ALTER TABLE invoice_items ADD COLUMN fed_unit_price TEXT NOT NULL DEFAULT '0';
ALTER TABLE invoice_items ADD COLUMN fed_sro TEXT NOT NULL DEFAULT '';
ALTER TABLE invoice_items ADD COLUMN fed_sro_serial TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN fed_type TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN fed_rate_text TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN fed_sro TEXT NOT NULL DEFAULT '';
ALTER TABLE products ADD COLUMN fed_sro_serial TEXT NOT NULL DEFAULT '';

-- The signature and the advance receipt particulars join the protected fields
-- of an accepted invoice. The signature may be set once, so that invoices
-- accepted before this version can be signed.
DROP TRIGGER IF EXISTS trg_invoices_immutable;
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
  OR NEW.advance_receipt IS NOT OLD.advance_receipt
  OR NEW.advance_ref IS NOT OLD.advance_ref
  OR (OLD.signature <> '' AND NEW.signature IS NOT OLD.signature)
  OR (OLD.status = 'CANCELLED' AND NEW.status IS NOT OLD.status)
  OR (OLD.status = 'ACCEPTED' AND NEW.status NOT IN ('ACCEPTED', 'CANCELLED'))
)
BEGIN
  SELECT RAISE(ABORT, 'invoice accepted by FBR is immutable');
END;

-- Closing on close of the day, week and month (rule 150R(4)(f)). Each closing
-- is a snapshot of the period's invoices, chained by hash and never changed.
CREATE TABLE closings (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id   INTEGER NOT NULL REFERENCES companies(id),
  environment  TEXT NOT NULL,
  kind         TEXT NOT NULL CHECK (kind IN ('day', 'week', 'month')),
  period_key   TEXT NOT NULL,
  period_start TEXT NOT NULL,
  period_end   TEXT NOT NULL,
  summary_json TEXT NOT NULL,
  hash         TEXT NOT NULL,
  prev_hash    TEXT NOT NULL DEFAULT '',
  created_at   TEXT NOT NULL,
  UNIQUE (company_id, environment, kind, period_key)
);
CREATE INDEX ix_closings_company ON closings(company_id, environment, kind, period_start);

CREATE TRIGGER trg_closings_noupdate BEFORE UPDATE ON closings
BEGIN
  SELECT RAISE(ABORT, 'closings cannot be changed');
END;

CREATE TRIGGER trg_closings_nodelete BEFORE DELETE ON closings
BEGIN
  SELECT RAISE(ABORT, 'closings cannot be deleted');
END;

-- Stock Transfer Notes for goods moved from a factory to the registered
-- person's own warehouse under the same STRN (Sales Tax General Order 25 of
-- 2026). Not tax invoices; kept for six years (section 24).
CREATE TABLE stock_transfers (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id    INTEGER NOT NULL REFERENCES companies(id),
  seq           INTEGER NOT NULL,
  number        TEXT NOT NULL,
  dispatched_at TEXT NOT NULL,
  from_name     TEXT NOT NULL,
  from_address  TEXT NOT NULL,
  to_name       TEXT NOT NULL,
  to_address    TEXT NOT NULL,
  vehicle_no    TEXT NOT NULL DEFAULT '',
  driver_cnic   TEXT NOT NULL DEFAULT '',
  authorised_by TEXT NOT NULL DEFAULT '',
  received_by   TEXT NOT NULL DEFAULT '',
  received_at   TEXT NOT NULL DEFAULT '',
  notes         TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL DEFAULT 'DISPATCHED' CHECK (status IN ('DISPATCHED', 'RECEIVED', 'CANCELLED')),
  cancel_reason TEXT NOT NULL DEFAULT '',
  total_value   INTEGER NOT NULL DEFAULT 0,
  created_by    INTEGER,
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL,
  UNIQUE (company_id, number)
);
CREATE INDEX ix_stock_transfers_company ON stock_transfers(company_id, dispatched_at);

CREATE TABLE stock_transfer_items (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  transfer_id   INTEGER NOT NULL REFERENCES stock_transfers(id),
  line_no       INTEGER NOT NULL,
  product_id    INTEGER REFERENCES products(id),
  description   TEXT NOT NULL,
  hs_code       TEXT NOT NULL DEFAULT '',
  quantity      TEXT NOT NULL,
  uom           TEXT NOT NULL DEFAULT '',
  value_at_cost INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX ix_stock_transfer_items ON stock_transfer_items(transfer_id);

CREATE TRIGGER trg_stock_transfers_nodelete BEFORE DELETE ON stock_transfers
BEGIN
  SELECT RAISE(ABORT, 'stock transfer notes must be kept for six years');
END;

CREATE TRIGGER trg_stock_transfer_items_nodelete BEFORE DELETE ON stock_transfer_items
BEGIN
  SELECT RAISE(ABORT, 'stock transfer notes must be kept for six years');
END;

-- Return filing dates extended by FBR for a tax period (orders under section
-- 74). Payment dates are not affected by such extensions.
CREATE TABLE return_extensions (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id  INTEGER NOT NULL REFERENCES companies(id),
  period      TEXT NOT NULL,
  filing_date TEXT NOT NULL,
  reference   TEXT NOT NULL DEFAULT '',
  created_by  INTEGER,
  created_at  TEXT NOT NULL,
  UNIQUE (company_id, period)
);
