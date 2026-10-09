// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// StockTransfer is a Stock Transfer Note: the non-fiscal document that must
// accompany goods moved from a factory to the registered person's own
// warehouse under the same STRN (Sales Tax General Order 25 of 2026).
type StockTransfer struct {
	ID           int64                `json:"id"`
	CompanyID    int64                `json:"companyId"`
	Seq          int                  `json:"seq"`
	Number       string               `json:"number"`
	DispatchedAt string               `json:"dispatchedAt"` // RFC 3339
	FromName     string               `json:"fromName"`
	FromAddress  string               `json:"fromAddress"`
	ToName       string               `json:"toName"`
	ToAddress    string               `json:"toAddress"`
	VehicleNo    string               `json:"vehicleNo"`
	DriverCNIC   string               `json:"driverCnic"`
	AuthorisedBy string               `json:"authorisedBy"`
	ReceivedBy   string               `json:"receivedBy"`
	ReceivedAt   string               `json:"receivedAt"`
	Notes        string               `json:"notes"`
	Status       string               `json:"status"` // DISPATCHED, RECEIVED, CANCELLED
	CancelReason string               `json:"cancelReason"`
	TotalValue   decimal.Decimal      `json:"totalValue"`
	CreatedAt    string               `json:"createdAt"`
	UpdatedAt    string               `json:"updatedAt"`
	Items        []*StockTransferItem `json:"items,omitempty"`
	ItemCount    int                  `json:"itemCount"`
}

// StockTransferItem is one line of a Stock Transfer Note.
type StockTransferItem struct {
	LineNo      int             `json:"lineNo"`
	ProductID   *int64          `json:"productId,omitempty"`
	Description string          `json:"description"`
	HSCode      string          `json:"hsCode"`
	Quantity    decimal.Decimal `json:"quantity"`
	UoM         string          `json:"uom"`
	ValueAtCost decimal.Decimal `json:"valueAtCost"`
}

const transferCols = `id, company_id, seq, number, dispatched_at, from_name, from_address, to_name, to_address, vehicle_no, driver_cnic,
	authorised_by, received_by, received_at, notes, status, cancel_reason, total_value, created_at, updated_at,
	(SELECT COUNT(*) FROM stock_transfer_items i WHERE i.transfer_id = stock_transfers.id)`

func scanTransfer(row interface{ Scan(...any) error }) (*StockTransfer, error) {
	var t StockTransfer
	var total int64
	err := row.Scan(&t.ID, &t.CompanyID, &t.Seq, &t.Number, &t.DispatchedAt, &t.FromName, &t.FromAddress, &t.ToName, &t.ToAddress,
		&t.VehicleNo, &t.DriverCNIC, &t.AuthorisedBy, &t.ReceivedBy, &t.ReceivedAt, &t.Notes, &t.Status, &t.CancelReason, &total,
		&t.CreatedAt, &t.UpdatedAt, &t.ItemCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t.TotalValue = Rupees(total)
	return &t, nil
}

// CreateStockTransfer stores a new note with the company's next number
// (STN-000001, STN-000002, ...).
func (s *Store) CreateStockTransfer(ctx context.Context, t *StockTransfer, userID *int64) error {
	return s.Tx(ctx, func(q Querier) error {
		var seq int
		if err := q.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) + 1 FROM stock_transfers WHERE company_id=?`, t.CompanyID).Scan(&seq); err != nil {
			return err
		}
		t.Seq, t.Number = seq, fmt.Sprintf("STN-%06d", seq)
		t.Status, t.CreatedAt = "DISPATCHED", now()
		t.UpdatedAt = t.CreatedAt
		total := decimal.Zero
		for _, it := range t.Items {
			total = total.Add(it.ValueAtCost)
		}
		t.TotalValue = total.Round(2)
		res, err := q.ExecContext(ctx, `INSERT INTO stock_transfers(company_id, seq, number, dispatched_at, from_name, from_address, to_name, to_address,
			vehicle_no, driver_cnic, authorised_by, notes, status, total_value, created_by, created_at, updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			t.CompanyID, t.Seq, t.Number, t.DispatchedAt, t.FromName, t.FromAddress, t.ToName, t.ToAddress, t.VehicleNo, t.DriverCNIC,
			t.AuthorisedBy, t.Notes, t.Status, Paisa(t.TotalValue), nullInt(userID), t.CreatedAt, t.UpdatedAt)
		if err != nil {
			return err
		}
		if t.ID, err = res.LastInsertId(); err != nil {
			return err
		}
		for i, it := range t.Items {
			it.LineNo = i + 1
			if _, err := q.ExecContext(ctx, `INSERT INTO stock_transfer_items(transfer_id, line_no, product_id, description, hs_code, quantity, uom, value_at_cost)
				VALUES(?,?,?,?,?,?,?,?)`, t.ID, it.LineNo, nullInt(it.ProductID), it.Description, it.HSCode, it.Quantity.String(), it.UoM, Paisa(it.ValueAtCost)); err != nil {
				return err
			}
		}
		t.ItemCount = len(t.Items)
		return nil
	})
}

// GetStockTransfer returns a note with its lines.
func (s *Store) GetStockTransfer(ctx context.Context, companyID, id int64) (*StockTransfer, error) {
	t, err := scanTransfer(s.DB.QueryRowContext(ctx, `SELECT `+transferCols+` FROM stock_transfers WHERE company_id=? AND id=?`, companyID, id))
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT line_no, product_id, description, hs_code, quantity, uom, value_at_cost FROM stock_transfer_items
		WHERE transfer_id=? ORDER BY line_no`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it StockTransferItem
		var pid sql.NullInt64
		var qty string
		var v int64
		if err := rows.Scan(&it.LineNo, &pid, &it.Description, &it.HSCode, &qty, &it.UoM, &v); err != nil {
			return nil, err
		}
		it.ProductID, it.Quantity, it.ValueAtCost = intPtr(pid), dec(qty), Rupees(v)
		t.Items = append(t.Items, &it)
	}
	return t, rows.Err()
}

// TransferFilter selects notes for the register.
type TransferFilter struct {
	From, To, Status, Q string
	Limit, Offset       int
}

// ListStockTransfers returns the register of notes, newest first.
func (s *Store) ListStockTransfers(ctx context.Context, companyID int64, f TransferFilter) ([]*StockTransfer, int, error) {
	where := []string{"company_id=?"}
	args := []any{companyID}
	if f.From != "" {
		where, args = append(where, "substr(dispatched_at,1,10)>=?"), append(args, f.From)
	}
	if f.To != "" {
		where, args = append(where, "substr(dispatched_at,1,10)<=?"), append(args, f.To)
	}
	if f.Status != "" {
		where, args = append(where, "status=?"), append(args, f.Status)
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		like := "%" + q + "%"
		where = append(where, "(number LIKE ? OR to_name LIKE ? OR vehicle_no LIKE ? OR id IN (SELECT transfer_id FROM stock_transfer_items WHERE description LIKE ?))")
		args = append(args, like, like, like, like)
	}
	cond := strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM stock_transfers WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+transferCols+` FROM stock_transfers WHERE `+cond+` ORDER BY seq DESC LIMIT ? OFFSET ?`,
		append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*StockTransfer{}
	for rows.Next() {
		t, err := scanTransfer(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// MarkTransferReceived records the warehouse's acknowledgement.
func (s *Store) MarkTransferReceived(ctx context.Context, companyID, id int64, by, at string) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE stock_transfers SET status='RECEIVED', received_by=?, received_at=?, updated_at=?
		WHERE company_id=? AND id=? AND status='DISPATCHED'`, by, at, now(), companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrConflict
	}
	return nil
}

// CancelStockTransfer marks a note cancelled (it is kept in the register).
func (s *Store) CancelStockTransfer(ctx context.Context, companyID, id int64, reason string) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE stock_transfers SET status='CANCELLED', cancel_reason=?, updated_at=?
		WHERE company_id=? AND id=? AND status='DISPATCHED'`, reason, now(), companyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrConflict
	}
	return nil
}
