// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"einvoicing/internal/store"

	"github.com/shopspring/decimal"
)

// Sales Tax General Order 25 of 2026: moving goods from a factory to the
// registered person's own warehouse under the same STRN is not a supply, so
// no digital invoice is issued; the consignment must instead carry a
// sequentially numbered Stock Transfer Note in the prescribed format, be
// acknowledged on receipt, reconciled monthly and kept for six years. A
// warehouse with a separate STRN receives a taxable supply and needs a
// digital invoice.

// TransferInput is a new Stock Transfer Note.
type TransferInput struct {
	DispatchedAt string `json:"dispatchedAt"` // RFC 3339 or YYYY-MM-DDTHH:MM (Pakistan time); empty = now
	FromName     string `json:"fromName"`
	FromAddress  string `json:"fromAddress"`
	ToName       string `json:"toName"`
	ToAddress    string `json:"toAddress"`
	VehicleNo    string `json:"vehicleNo"`
	DriverCNIC   string `json:"driverCnic"`
	AuthorisedBy string `json:"authorisedBy"`
	Notes        string `json:"notes"`
	// SameSTRN confirms the warehouse is registered under the company's own
	// STRN; otherwise a sales tax invoice is required.
	SameSTRN bool                `json:"sameStrn"`
	Items    []TransferItemInput `json:"items"`
}

// TransferItemInput is one line of a new note.
type TransferItemInput struct {
	ProductID   *int64          `json:"productId"`
	Description string          `json:"description"`
	HSCode      string          `json:"hsCode"`
	Quantity    decimal.Decimal `json:"quantity"`
	UoM         string          `json:"uom"`
	ValueAtCost decimal.Decimal `json:"valueAtCost"`
}

func parseLocalTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return now, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, PKT); err == nil {
			return t, nil
		}
	}
	return time.Time{}, Invalid("date and time must be like 2026-10-09T14:30")
}

// CreateStockTransfer records and numbers a Stock Transfer Note.
func (s *Service) CreateStockTransfer(ctx context.Context, a Actor, companyID int64, in TransferInput) (*store.StockTransfer, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if !in.SameSTRN {
		return nil, Invalid("a Stock Transfer Note is only for goods moved to your own warehouse under the same STRN (Sales Tax General Order 25 of 2026). If the warehouse holds a separate STRN the movement is a taxable supply: issue a sales tax invoice instead")
	}
	at, err := parseLocalTime(in.DispatchedAt, s.Now())
	if err != nil {
		return nil, err
	}
	if at.After(s.Now().Add(24 * time.Hour)) {
		return nil, Invalid("the dispatch time cannot be in the future")
	}
	t := &store.StockTransfer{CompanyID: c.ID, DispatchedAt: at.UTC().Format(time.RFC3339),
		FromName: truncate(cleanText(in.FromName), 200), FromAddress: truncate(cleanText(in.FromAddress), 300),
		ToName: truncate(cleanText(in.ToName), 200), ToAddress: truncate(cleanText(in.ToAddress), 300),
		VehicleNo: truncate(strings.ToUpper(cleanText(in.VehicleNo)), 30), AuthorisedBy: truncate(cleanText(in.AuthorisedBy), 120),
		Notes: truncate(strings.TrimSpace(in.Notes), 500)}
	if t.FromName == "" {
		t.FromName = c.Name
	}
	if t.FromAddress == "" {
		t.FromAddress = strings.TrimSpace(strings.Join([]string{c.Address, c.City}, ", "))
		t.FromAddress = strings.Trim(t.FromAddress, ", ")
	}
	if t.ToName == "" || t.ToAddress == "" {
		return nil, Invalid("enter the receiving warehouse's name and address")
	}
	if cnic := strings.NewReplacer("-", "", " ", "").Replace(in.DriverCNIC); cnic != "" {
		if len(cnic) != 13 || strings.Trim(cnic, "0123456789") != "" {
			return nil, Invalid("the driver's CNIC must have 13 digits")
		}
		t.DriverCNIC = cnic[:5] + "-" + cnic[5:12] + "-" + cnic[12:]
	}
	if len(in.Items) == 0 {
		return nil, Invalid("add at least one line")
	}
	if len(in.Items) > 200 {
		return nil, Invalid("a note can have at most 200 lines")
	}
	for i, it := range in.Items {
		line := &store.StockTransferItem{ProductID: it.ProductID, Description: truncate(cleanText(it.Description), 300),
			HSCode: truncate(strings.TrimSpace(it.HSCode), 20), Quantity: it.Quantity, UoM: truncate(strings.TrimSpace(it.UoM), 60),
			ValueAtCost: it.ValueAtCost.Round(2)}
		if it.ProductID != nil {
			p, err := s.Store.GetProduct(ctx, c.ID, *it.ProductID)
			if err != nil {
				return nil, Invalid("line %d: product not found", i+1)
			}
			if line.Description == "" {
				line.Description = p.Description
			}
			if line.HSCode == "" {
				line.HSCode = p.HSCode
			}
			if line.UoM == "" {
				line.UoM = p.UoM
			}
		}
		switch {
		case line.Description == "":
			return nil, Invalid("line %d: enter a description of the goods", i+1)
		case !line.Quantity.IsPositive():
			return nil, Invalid("line %d: the quantity must be more than zero", i+1)
		case line.ValueAtCost.IsNegative():
			return nil, Invalid("line %d: the value at cost cannot be negative", i+1)
		}
		t.Items = append(t.Items, line)
	}
	if err := s.Store.CreateStockTransfer(ctx, t, a.UserID); err != nil {
		return nil, err
	}
	s.Audit(ctx, a, c.ID, "stock_transfer.create", "stock_transfer", fmt.Sprint(t.ID), map[string]any{"no": t.Number, "to": t.ToName,
		"lines": len(t.Items), "value": t.TotalValue.StringFixed(2)})
	return s.Store.GetStockTransfer(ctx, c.ID, t.ID)
}

// ReceiveStockTransfer records the warehouse's acknowledgement of a note.
func (s *Service) ReceiveStockTransfer(ctx context.Context, a Actor, companyID, id int64, by, at string) (*store.StockTransfer, error) {
	by = truncate(cleanText(by), 120)
	if by == "" {
		return nil, Invalid("enter the name and designation of the person who received the goods")
	}
	t, err := parseLocalTime(at, s.Now())
	if err != nil {
		return nil, err
	}
	if err := s.Store.MarkTransferReceived(ctx, companyID, id, by, t.UTC().Format(time.RFC3339)); err != nil {
		if err == store.ErrConflict {
			return nil, Invalid("only a note in transit can be marked received")
		}
		return nil, err
	}
	s.Audit(ctx, a, companyID, "stock_transfer.received", "stock_transfer", fmt.Sprint(id), map[string]any{"by": by})
	return s.Store.GetStockTransfer(ctx, companyID, id)
}

// CancelStockTransfer cancels a note issued in error; it stays in the
// register.
func (s *Service) CancelStockTransfer(ctx context.Context, a Actor, companyID, id int64, reason string) (*store.StockTransfer, error) {
	reason = cleanText(reason)
	if reason == "" {
		return nil, Invalid("enter the reason for cancelling the note")
	}
	if err := s.Store.CancelStockTransfer(ctx, companyID, id, truncate(reason, 300)); err != nil {
		if err == store.ErrConflict {
			return nil, Invalid("only a note in transit can be cancelled")
		}
		return nil, err
	}
	s.Audit(ctx, a, companyID, "stock_transfer.cancel", "stock_transfer", fmt.Sprint(id), map[string]any{"reason": reason})
	return s.Store.GetStockTransfer(ctx, companyID, id)
}
