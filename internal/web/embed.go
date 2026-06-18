package web

import (
	"embed"
	"html/template"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

//go:embed templates/index.html.tmpl
var indexHTML string

//go:embed assets/app.css assets/poll.js
var assetsFS embed.FS

var tmplFuncs = template.FuncMap{
	"dash": func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	},
	"scan": func(s model.ScanState, n int) string {
		return s.Cell(n)
	},
}
