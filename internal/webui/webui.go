// Package webui embeds the compiled single-page application (built from
// /web with `npm run build`, which writes into internal/webui/dist).
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the UI file system rooted at dist/.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
