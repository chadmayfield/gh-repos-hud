package web

import (
	"embed"
	"html/template"
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
	"scan": func(enabled bool, n int) string {
		if !enabled {
			return "?"
		}
		if n == 0 {
			return "0"
		}
		return itoa(n)
	},
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
