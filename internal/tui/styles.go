package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

var (
	colRed    = lipgloss.Color("1")
	colGreen  = lipgloss.Color("2")
	colYellow = lipgloss.Color("3")
	colBlue   = lipgloss.Color("4")
	colGray   = lipgloss.Color("8")

	styleHeader   = lipgloss.NewStyle().Bold(true).Foreground(colBlue)
	styleOrg      = lipgloss.NewStyle().Bold(true).Underline(true)
	styleFooter   = lipgloss.NewStyle().Foreground(colGray)
	styleSelected = lipgloss.NewStyle().Reverse(true)
	styleDim      = lipgloss.NewStyle().Foreground(colGray)
	styleWarn     = lipgloss.NewStyle().Foreground(colYellow)
	styleKey      = lipgloss.NewStyle().Bold(true)
)

// healthColor maps a rollup to a lipgloss color.
func healthColor(h model.Health) lipgloss.Color {
	switch h {
	case model.HealthGreen:
		return colGreen
	case model.HealthYellow:
		return colYellow
	case model.HealthRed:
		return colRed
	default:
		return colGray
	}
}

// glyph renders the colored ASCII health marker, padded to a fixed width so
// columns stay aligned ([OK]/[!!] are 4 wide, [~]/[?] are 3).
func glyph(h model.Health) string {
	return lipgloss.NewStyle().Foreground(healthColor(h)).Render(pad(h.Glyph(), 4))
}

// ciStyled colors a CI short label.
func ciStyled(c model.CIState) string {
	col := colGray
	switch c {
	case model.CISuccess:
		col = colGreen
	case model.CIFailure:
		col = colRed
	case model.CIPending:
		col = colYellow
	}
	return lipgloss.NewStyle().Foreground(col).Render(pad(c.Short(), 4))
}
