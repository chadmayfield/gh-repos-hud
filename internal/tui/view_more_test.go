package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func TestViewLoadingNilState(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	out := m.View()
	if !strings.Contains(out, "loading repo health") {
		t.Fatalf("nil-state view should show loading:\n%s", out)
	}
}

func TestViewErrorNilState(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.err = errors.New("kaboom")
	out := m.View()
	for _, want := range []string{"error:", "kaboom", "press q to quit"} {
		if !strings.Contains(out, want) {
			t.Fatalf("error view missing %q:\n%s", want, out)
		}
	}
}

func TestViewNormalRichState(t *testing.T) {
	m := newRichModel()
	m.loading = true // exercise "(refreshing...)"
	out := m.View()
	for _, want := range []string{
		"gh-repos-hud", "(refreshing...)", "sort:health",
		"acme", "octocat", "(personal)",
		"red-repo", "yellow-repo", "green-repo",
		"[!!]", "[~]", "[OK]",
		"GHAS secret=2 code=3", // billing tally
		"REST 4900/5000", "GraphQL 4500/5000", "(cost 12 pts)",
		"cached", "1 warning(s)",
		"q quit",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("normal view missing %q:\n%s", want, out)
		}
	}
}

func TestViewFilteredAndAttention(t *testing.T) {
	m := newRichModel()
	m.filter = "red"
	m.onlyAttention = true
	m.rebuildRows()
	out := m.View()
	for _, want := range []string{"/red", "[attention-only]", "red-repo"} {
		if !strings.Contains(out, want) {
			t.Fatalf("filtered/attention view missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "green-repo") {
		t.Fatalf("filtered view should not contain green-repo:\n%s", out)
	}
}

func TestViewCollapsedOrg(t *testing.T) {
	m := newRichModel()
	m.expanded["acme"] = false
	m.rebuildRows()
	out := m.View()
	if !strings.Contains(out, "[+]") {
		t.Fatalf("collapsed org should render [+] marker:\n%s", out)
	}
	if strings.Contains(out, "red-repo") {
		t.Fatalf("collapsed org should hide its repos:\n%s", out)
	}
}

func TestViewAutoRefreshPaused(t *testing.T) {
	m := New(nil, ghclient.Options{}, true, 0)
	st := richState()
	st.RateLimit.GraphQLRemaining = 10 // below lowGraphQLThreshold
	m.state = st
	m.rebuildRows()
	out := m.View()
	if !strings.Contains(out, "auto-refresh paused") {
		t.Fatalf("low-GraphQL watch view should warn about paused refresh:\n%s", out)
	}
}

func TestViewDetailPopulated(t *testing.T) {
	m := newRichModel()
	m.cursor = 1 // red-repo
	m.detail = true
	m.detailData = &model.RepoDetail{
		AheadBy: 5, AheadKnown: true,
		Alerts: []model.AlertDetail{
			{Package: "lodash", Severity: "critical", Summary: "prototype pollution"},
			{Package: "minimist", Severity: "moderate", Summary: "argument injection"},
		},
		PRs: []model.PRDetail{
			{Number: 12, Title: "bump deps"},
			{Number: 13, Title: "wip refactor", Draft: true},
		},
	}
	out := m.viewDetail()
	for _, want := range []string{
		"red-repo", "https://example.com/red",
		"health", "branch", "tag/release", "undeployed",
		"dependabot", "code scan", "secret scan", "open PRs",
		"open alerts:", "CRITICAL", "lodash", "MODERATE", "minimist",
		"#12", "bump deps", "#13", "(draft)",
		"enter/esc back", "o open in browser", "q quit",
		"commit(s) since", // AheadKnown branch
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("detail view missing %q:\n%s", want, out)
		}
	}
}

func TestViewDetailNoAlerts(t *testing.T) {
	m := newRichModel()
	m.cursor = 1
	m.detail = true
	m.detailData = &model.RepoDetail{Alerts: nil, PRs: nil}
	out := m.viewDetail()
	if !strings.Contains(out, "no open alerts") {
		t.Fatalf("empty-alert detail should say so:\n%s", out)
	}
}

func TestViewDetailNilData(t *testing.T) {
	m := newRichModel()
	m.cursor = 1
	m.detail = true
	m.detailLoading = true
	m.detailData = nil
	out := m.viewDetail()
	if !strings.Contains(out, "loading alerts + PRs") {
		t.Fatalf("loading detail should show loading line:\n%s", out)
	}
}

func TestViewDetailNonRepoCursorFallsBack(t *testing.T) {
	m := newRichModel()
	m.cursor = 0 // org row -> selectedRepo() is nil
	m.detail = true
	out := m.viewDetail()
	// Falls back to the normal list view.
	if !strings.Contains(out, "gh-repos-hud") {
		t.Fatalf("detail on a non-repo row should fall back to list view:\n%s", out)
	}
}

func TestViewViaUpdateDetailDispatch(t *testing.T) {
	// View() routes to viewDetail when m.detail is set.
	m := newRichModel()
	m.cursor = 1
	m.detail = true
	m.detailData = &model.RepoDetail{}
	out := m.View()
	if !strings.Contains(out, "enter/esc back") {
		t.Fatalf("View() should dispatch to viewDetail:\n%s", out)
	}
}
