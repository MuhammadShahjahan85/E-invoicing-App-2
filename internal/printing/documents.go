// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package printing

import (
	"io"
	"strings"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"
)

func karachi() *time.Location {
	loc, err := time.LoadLocation("Asia/Karachi")
	if err != nil {
		return time.FixedZone("PKT", 5*3600)
	}
	return loc
}

// RenderSignboard writes the "Integrated with FBR" signboard that rule
// 150R(11) requires at each integrated outlet or point of sale.
func RenderSignboard(w io.Writer, c *store.Company, outlet string, o Options) error {
	logo, mime := o.FBRLogo, o.FBRLogoMime
	if len(logo) == 0 {
		logo, mime = DefaultFBRLogo, DefaultFBRLogoMime
	}
	return templates.ExecuteTemplate(w, "signboard.html", map[string]any{
		"Company": c, "Outlet": strings.TrimSpace(outlet), "FBRLogo": dataURI(logo, mime), "ProductName": brand.ProductName,
		"ShowToolbar": o.ShowToolbar, "Nonce": o.Nonce,
	})
}

type transferLine struct {
	No                            int
	Description, HSCode, Qty, UoM string
	Value                         string
}

// RenderStockTransfer writes a Stock Transfer Note in the format of
// Annexure-A to Sales Tax General Order 25 of 2026: a despatch copy and a
// warehouse copy.
func RenderStockTransfer(w io.Writer, c *store.Company, t *store.StockTransfer, o Options) error {
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	loc := karachi()
	v := map[string]any{
		"Company": c, "T": t, "Nonce": o.Nonce, "ShowToolbar": o.ShowToolbar, "ProductName": brand.ProductName,
		"Dispatched": shortTime(t.DispatchedAt, loc), "PrintedAt": o.Now.In(loc).Format("02-Jan-2006 15:04"),
		"Total": tax.FormatAmount(t.TotalValue), "Copies": []string{"DESPATCH COPY", "WAREHOUSE COPY"},
	}
	if t.ReceivedAt != "" {
		v["Received"] = shortTime(t.ReceivedAt, loc)
	}
	var lines []transferLine
	for _, it := range t.Items {
		lines = append(lines, transferLine{No: it.LineNo, Description: it.Description, HSCode: it.HSCode, Qty: tax.FormatQty(it.Quantity),
			UoM: it.UoM, Value: tax.FormatAmount(it.ValueAtCost)})
	}
	v["Lines"] = lines
	return templates.ExecuteTemplate(w, "stock_transfer.html", v)
}
