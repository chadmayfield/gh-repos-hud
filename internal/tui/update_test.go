package tui

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// richState covers all health states across two owners (one org, one personal),
// with billing, scans, tags, undeployed counts, and PRs populated.
func richState() *model.State {
	return &model.State{
		Owners: []model.Owner{
			{
				Name:    "acme",
				Billing: model.Billing{Known: true, SecretProtectionCommitters: 2, CodeSecurityCommitters: 3},
				Repos: []model.Repo{
					{Name: "red-repo", URL: "https://example.com/red", ShortSHA: "aaaa111", LatestTag: "v1.0.0",
						CI: model.CIFailure, Health: model.HealthRed, Dependabot: model.AlertCounts{Critical: 1, High: 2},
						CodeScan: model.ScanOn, CodeScanning: 3, SecretScan: model.ScanOn, SecretScanning: 1,
						Undeployed: 5, DefaultBranch: "main", PRs: model.PRStats{Total: 2, Bot: 1, Human: 1}},
					{Name: "yellow-repo", URL: "https://example.com/yellow", ShortSHA: "bbbb222", LatestTag: "v0.9.0",
						CI: model.CIPending, Health: model.HealthYellow, Dependabot: model.AlertCounts{Moderate: 2, Low: 1},
						CodeScan: model.ScanOff, SecretScan: model.ScanUnknown, Undeployed: -1, DefaultBranch: "main"},
					{Name: "green-repo", URL: "https://example.com/green", ShortSHA: "cccc333", CI: model.CISuccess,
						Health: model.HealthGreen, Untagged: true, DefaultBranch: "main"},
					{Name: "gray-repo", ShortSHA: "dddd444", CI: model.CINone, Health: model.HealthGray,
						Archived: true, Private: true},
				},
			},
			{
				Name:   "octocat",
				IsUser: true,
				Repos: []model.Repo{
					{Name: "personal-tool", URL: "https://example.com/pt", ShortSHA: "eeee555",
						CI: model.CISuccess, Health: model.HealthGreen, Untagged: true},
				},
			},
		},
		RateLimit: model.RateLimit{RESTRemaining: 4900, RESTLimit: 5000, GraphQLRemaining: 4500, GraphQLLimit: 5000, GraphQLCost: 12},
		FetchedAt: time.Now().Add(-30 * time.Second),
		FromCache: true,
		Warnings:  []model.Warning{{Owner: "acme", Feature: "secret-scanning", Reason: "disabled"}},
	}
}

func newRichModel() Model {
	m := New(nil, ghclient.Options{}, false, 0)
	m.state = richState()
	m.height = 40
	m.width = 120
	m.rebuildRows()
	return m
}

// runeKey builds a KeyRunes message for a letter/string key.
func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// send runs Update and type-asserts the result back to a Model.
func send(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	res, cmd := m.Update(msg)
	mm, ok := res.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", res)
	}
	return mm, cmd
}

func TestNavigationKeys(t *testing.T) {
	m := newRichModel()
	m.cursor = 0

	m, _ = send(t, m, runeKey("j"))
	if m.cursor != 1 {
		t.Fatalf("j: cursor = %d, want 1", m.cursor)
	}
	m, _ = send(t, m, runeKey("k"))
	if m.cursor != 0 {
		t.Fatalf("k: cursor = %d, want 0", m.cursor)
	}
	// "k" at top clamps to 0.
	m, _ = send(t, m, runeKey("k"))
	if m.cursor != 0 {
		t.Fatalf("k clamp: cursor = %d, want 0", m.cursor)
	}

	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatalf("down: cursor = %d, want 1", m.cursor)
	}
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Fatalf("up: cursor = %d, want 0", m.cursor)
	}

	// G jumps to bottom, g to top.
	m, _ = send(t, m, runeKey("G"))
	if m.cursor != len(m.rows)-1 {
		t.Fatalf("G: cursor = %d, want %d", m.cursor, len(m.rows)-1)
	}
	m, _ = send(t, m, runeKey("g"))
	if m.cursor != 0 {
		t.Fatalf("g: cursor = %d, want 0", m.cursor)
	}
}

func TestPagingKeys(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.state = manyReposState(30)
	m.height = 15 // bodyCapacity = 7
	m.rebuildRows()
	m.cursor = 0

	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeySpace})
	if m.cursor != 7 {
		t.Fatalf("space: cursor = %d, want 7", m.cursor)
	}
	m, _ = send(t, m, runeKey("G"))
	last := len(m.rows) - 1
	// pgdown past the end clamps.
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyPgDown})
	if m.cursor != last {
		t.Fatalf("pgdown clamp: cursor = %d, want %d", m.cursor, last)
	}
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	if m.cursor != last-7 {
		t.Fatalf("pgup: cursor = %d, want %d", m.cursor, last-7)
	}
	// pgup past the top clamps to 0.
	m.cursor = 3
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	if m.cursor != 0 {
		t.Fatalf("pgup clamp: cursor = %d, want 0", m.cursor)
	}
}

func TestTabTogglesExpand(t *testing.T) {
	m := newRichModel()
	m.cursor = 0 // first org row "acme"
	if !m.isExpanded("acme") {
		t.Fatal("acme should start expanded")
	}
	m, _ = send(t, m, runeKey("tab"))
	if m.isExpanded("acme") {
		t.Fatal("tab should have collapsed acme")
	}
	m, _ = send(t, m, runeKey("tab"))
	if !m.isExpanded("acme") {
		t.Fatal("tab should have re-expanded acme")
	}
}

func TestEnterOnOrgTogglesExpand(t *testing.T) {
	m := newRichModel()
	m.cursor = 0 // org row
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.isExpanded("acme") {
		t.Fatal("enter on org should collapse it")
	}
	if m.detail {
		t.Fatal("enter on org must not enter detail")
	}
}

func TestEnterOnRepoEntersDetail(t *testing.T) {
	m := newRichModel()
	m.cursor = 1 // first repo under acme
	m, cmd := send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.detail {
		t.Fatal("enter on repo should set detail=true")
	}
	if !m.detailLoading {
		t.Fatal("non-demo enter should set detailLoading=true")
	}
	if cmd == nil {
		t.Fatal("enter on repo (live) should return a non-nil detail cmd")
	}
	// Exit detail again with enter.
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.detail {
		t.Fatal("enter in detail should exit detail")
	}
}

func TestEnterOnRepoDemoGuard(t *testing.T) {
	m := New(nil, ghclient.Options{Demo: true}, false, 0)
	m.state = richState()
	m.height = 40
	m.rebuildRows()
	m.cursor = 1 // a repo row
	m, cmd := send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.detail {
		t.Fatal("demo enter should set detail=true")
	}
	if m.detailLoading {
		t.Fatal("demo enter must leave detailLoading=false")
	}
	if cmd != nil {
		t.Fatal("demo enter must return a nil cmd (no live fetch)")
	}
}

func TestAttentionToggle(t *testing.T) {
	m := newRichModel()
	before := len(m.rows)
	m, _ = send(t, m, runeKey("a"))
	if !m.onlyAttention {
		t.Fatal("a should enable onlyAttention")
	}
	if len(m.rows) >= before {
		t.Fatalf("attention mode should drop green rows: before=%d after=%d", before, len(m.rows))
	}
	m, _ = send(t, m, runeKey("a"))
	if m.onlyAttention {
		t.Fatal("a should disable onlyAttention")
	}
}

func TestSortCycle(t *testing.T) {
	m := newRichModel()
	if m.sortBy != sortHealth {
		t.Fatalf("default sortBy = %v, want sortHealth", m.sortBy)
	}
	for _, want := range []sortKey{sortName, sortUndeployed, sortAlerts, sortHealth} {
		m, _ = send(t, m, runeKey("s"))
		if m.sortBy != want {
			t.Fatalf("s cycle: got %v want %v", m.sortBy, want)
		}
	}
}

func TestFilterEntry(t *testing.T) {
	m := newRichModel()
	m, _ = send(t, m, runeKey("/"))
	if !m.filtering {
		t.Fatal("/ should enter filtering mode")
	}
}

func TestOpenKeyReturnsCmd(t *testing.T) {
	m := newRichModel()
	m.cursor = 1 // repo with URL set
	_, cmd := send(t, m, runeKey("o"))
	if cmd == nil {
		t.Fatal("o on a repo with a URL should return a non-nil cmd")
	}
	// Never execute the returned cmd (it would shell out to a browser).

	// o on an org row (no URL) returns nil.
	m2 := newRichModel()
	m2.cursor = 0
	if _, cmd := send(t, m2, runeKey("o")); cmd != nil {
		t.Fatal("o on an org row should return nil")
	}
}

func TestEscBackspaceExitDetail(t *testing.T) {
	for _, k := range []tea.KeyType{tea.KeyEsc, tea.KeyBackspace} {
		m := newRichModel()
		m.detail = true
		m, _ = send(t, m, tea.KeyMsg{Type: k})
		if m.detail {
			t.Fatalf("key %v should exit detail", k)
		}
	}
}

func TestQuitKeys(t *testing.T) {
	for _, msg := range []tea.KeyMsg{runeKey("q"), {Type: tea.KeyCtrlC}} {
		m := newRichModel()
		_, cmd := send(t, m, msg)
		if cmd == nil {
			t.Fatalf("%v should return tea.Quit", msg)
		}
	}
}

func TestFilteringModeInput(t *testing.T) {
	m := newRichModel()
	m.filtering = true

	m, _ = send(t, m, runeKey("re"))
	if m.filter != "re" {
		t.Fatalf("filter = %q, want \"re\"", m.filter)
	}
	m, _ = send(t, m, runeKey("d"))
	if m.filter != "red" {
		t.Fatalf("filter = %q, want \"red\"", m.filter)
	}
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.filter != "re" {
		t.Fatalf("backspace: filter = %q, want \"re\"", m.filter)
	}
	// Backspace on empty filter is a no-op.
	m.filter = ""
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.filter != "" {
		t.Fatalf("backspace empty: filter = %q, want \"\"", m.filter)
	}

	// Enter exits filtering.
	m.filtering = true
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.filtering {
		t.Fatal("enter should exit filtering")
	}
	// Esc exits filtering.
	m.filtering = true
	m, _ = send(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.filtering {
		t.Fatal("esc should exit filtering")
	}
}

func TestUpdateStateLoadedMsg(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	if !m.loading {
		t.Fatal("new model should start loading")
	}
	m, _ = send(t, m, stateLoadedMsg{state: richState()})
	if m.loading {
		t.Fatal("stateLoadedMsg should clear loading")
	}
	if m.state == nil {
		t.Fatal("stateLoadedMsg should set state")
	}
	if len(m.rows) == 0 {
		t.Fatal("stateLoadedMsg should rebuild rows")
	}
}

func TestUpdateErrMsg(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	want := errors.New("boom")
	m, _ = send(t, m, errMsg{err: want})
	if m.err != want {
		t.Fatalf("err = %v, want %v", m.err, want)
	}
	if m.loading {
		t.Fatal("errMsg should clear loading")
	}
}

func TestUpdateDetailLoadedMsg(t *testing.T) {
	m := newRichModel()
	m.detailLoading = true
	d := model.RepoDetail{AheadBy: 3, AheadKnown: true,
		Alerts: []model.AlertDetail{{Package: "lodash", Severity: "high", Summary: "proto pollution"}},
		PRs:    []model.PRDetail{{Number: 7, Title: "fix things"}}}
	m, _ = send(t, m, detailLoadedMsg{detail: d})
	if m.detailLoading {
		t.Fatal("detailLoadedMsg should clear detailLoading")
	}
	if m.detailData == nil || m.detailData.AheadBy != 3 {
		t.Fatal("detailLoadedMsg should set detailData")
	}
}

func TestUpdateTickMsg(t *testing.T) {
	// watch=false: tick is inert.
	m := newRichModel()
	m, cmd := send(t, m, tickMsg{})
	if cmd != nil {
		t.Fatal("tick with watch=false should return nil cmd")
	}

	// watch=true, healthy headroom: schedules a refresh + next tick.
	mw := New(nil, ghclient.Options{}, true, time.Second)
	mw.state = richState()
	mw.loading = false
	mw.rebuildRows()
	mw, cmd = send(t, mw, tickMsg{})
	if cmd == nil {
		t.Fatal("tick with watch=true should return a batch cmd")
	}
	if !mw.loading {
		t.Fatal("tick with watch=true should set loading")
	}
}

func TestUpdateWindowSizeMsg(t *testing.T) {
	m := newRichModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 200, Height: 50})
	if m.width != 200 || m.height != 50 {
		t.Fatalf("size = %dx%d, want 200x50", m.width, m.height)
	}
}

func TestUpdateUnknownMsg(t *testing.T) {
	m := newRichModel()
	type weird struct{}
	if _, cmd := send(t, m, weird{}); cmd != nil {
		t.Fatal("unknown msg should be a no-op")
	}
}
