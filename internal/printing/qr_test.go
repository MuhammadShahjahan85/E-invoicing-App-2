// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package printing

import (
	"strings"
	"testing"
)

func TestQRVersion2(t *testing.T) {
	for _, no := range []string{"7000007DI1747119701593", "4210112345671DI1747119701593", "0786909DI1747119701593"} {
		n, err := QRModuleCount(no)
		if err != nil {
			t.Fatalf("%s: %v", no, err)
		}
		if n != QRModules {
			t.Errorf("%s: %d modules, want %d", no, n, QRModules)
		}
		svg, err := QRSVG(no)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(svg, `width="1in" height="1in"`) || !strings.Contains(svg, `viewBox="0 0 25 25"`) {
			t.Errorf("unexpected svg header: %.120s", svg)
		}
		png, err := QRPNG(no, 8)
		if err != nil || len(png) < 100 {
			t.Errorf("png: %v", err)
		}
	}
	if _, err := QRSVG(strings.Repeat("x", 200)); err == nil {
		t.Error("expected overflow error")
	}
}
