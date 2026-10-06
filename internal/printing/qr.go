// Package printing renders printable invoices (A4 and 80 mm thermal) with
// the FBR invoice number, the FBR-specified QR code and the Digital
// Invoicing logo.
package printing

import (
	"bytes"
	"fmt"
	"image/png"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// FBR's DI specification prescribes a QR code of Version 2.0 (25×25
// modules) printed at 1.0 × 1.0 inch, encoding the FBR invoice number.
const (
	QRVersion = 2
	QRModules = 25
)

// newQR builds a Version 2 symbol, preferring error-correction level M and
// falling back to L when the content is longer.
func newQR(content string) (*qrcode.QRCode, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty QR content")
	}
	for _, lvl := range []qrcode.RecoveryLevel{qrcode.Medium, qrcode.Low} {
		q, err := qrcode.NewWithForcedVersion(content, QRVersion, lvl)
		if err == nil {
			q.DisableBorder = true
			return q, nil
		}
	}
	return nil, fmt.Errorf("%q does not fit in a Version %d QR code", content, QRVersion)
}

// QRSVG returns an SVG of the QR code sized exactly 1in × 1in (the quiet
// zone is added by the surrounding layout as white margin).
func QRSVG(content string) (string, error) {
	q, err := newQR(content)
	if err != nil {
		return "", err
	}
	bm := q.Bitmap()
	n := len(bm)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="1in" height="1in" viewBox="0 0 %d %d" shape-rendering="crispEdges" data-qr-version="%d" data-qr-modules="%d" role="img" aria-label="FBR invoice QR code">`, n, n, QRVersion, n)
	b.WriteString(`<rect width="100%" height="100%" fill="#fff"/><path fill="#000" d="`)
	for y, row := range bm {
		for x, on := range row {
			if on {
				fmt.Fprintf(&b, "M%d %dh1v1h-1z", x, y)
			}
		}
	}
	b.WriteString(`"/></svg>`)
	return b.String(), nil
}

// QRPNG returns a PNG of the QR code; pixelsPerModule controls the size
// (e.g. 8 → 200 px for the 25-module symbol, about 1 inch at 200 dpi).
func QRPNG(content string, pixelsPerModule int) ([]byte, error) {
	q, err := newQR(content)
	if err != nil {
		return nil, err
	}
	if pixelsPerModule <= 0 {
		pixelsPerModule = 12
	}
	size := len(q.Bitmap()) * pixelsPerModule
	img := q.Image(size)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// QRModuleCount returns the number of modules per side (25 for Version 2).
func QRModuleCount(content string) (int, error) {
	q, err := newQR(content)
	if err != nil {
		return 0, err
	}
	return len(q.Bitmap()), nil
}
