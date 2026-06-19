package model

import "testing"

func TestHealthGlyphAndString(t *testing.T) {
	cases := []struct {
		h     Health
		glyph string
		str   string
	}{
		{HealthGreen, "[OK]", "green"},
		{HealthYellow, "[~]", "yellow"},
		{HealthRed, "[!!]", "red"},
		{HealthGray, "[?]", "gray"},
	}
	for _, c := range cases {
		if got := c.h.Glyph(); got != c.glyph {
			t.Errorf("Glyph(%v)=%q want %q", c.h, got, c.glyph)
		}
		if got := c.h.String(); got != c.str {
			t.Errorf("String(%v)=%q want %q", c.h, got, c.str)
		}
	}
}

func TestCIStateStringAndShort(t *testing.T) {
	cases := []struct {
		c     CIState
		str   string
		short string
	}{
		{CINone, "none", "-"},
		{CISuccess, "success", "OK"},
		{CIPending, "pending", "..."},
		{CIFailure, "failure", "FAIL"},
	}
	for _, c := range cases {
		if got := c.c.String(); got != c.str {
			t.Errorf("String(%v)=%q want %q", c.c, got, c.str)
		}
		if got := c.c.Short(); got != c.short {
			t.Errorf("Short(%v)=%q want %q", c.c, got, c.short)
		}
	}
}

func TestScanStateStringAndCell(t *testing.T) {
	cases := []struct {
		s    ScanState
		n    int
		str  string
		cell string
	}{
		{ScanUnknown, 0, "unknown", "?"},
		{ScanOff, 0, "off", "off"},
		{ScanOn, 0, "on", "0"},
		{ScanOn, 5, "on", "5"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.str {
			t.Errorf("String(%v)=%q want %q", c.s, got, c.str)
		}
		if got := c.s.Cell(c.n); got != c.cell {
			t.Errorf("Cell(%v,%d)=%q want %q", c.s, c.n, got, c.cell)
		}
	}
}

func TestAlertCountsTotal(t *testing.T) {
	a := AlertCounts{Critical: 1, High: 2, Moderate: 3, Low: 4}
	if a.Total() != 10 {
		t.Errorf("Total()=%d want 10", a.Total())
	}
	if (AlertCounts{}).Total() != 0 {
		t.Error("empty AlertCounts Total should be 0")
	}
}
