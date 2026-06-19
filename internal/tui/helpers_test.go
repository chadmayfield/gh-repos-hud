package tui

import (
	"testing"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func TestSortKeyString(t *testing.T) {
	cases := map[sortKey]string{
		sortHealth:     "health",
		sortName:       "name",
		sortUndeployed: "undeployed",
		sortAlerts:     "alerts",
		sortKey(99):    "health", // default
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("sortKey(%d).String() = %q, want %q", k, got, want)
		}
	}
}

func TestUndeployRank(t *testing.T) {
	untagged := model.Repo{Untagged: true}
	if got := undeployRank(untagged); got != -1 {
		t.Errorf("untagged rank = %d, want -1", got)
	}
	deployed := model.Repo{Undeployed: 0}
	if got := undeployRank(deployed); got != 0 {
		t.Errorf("deployed rank = %d, want 0", got)
	}
	// Unknown ">=1" (Undeployed == -1) ranks above an exact count.
	unknown := undeployRank(model.Repo{Undeployed: -1})
	exact := undeployRank(model.Repo{Undeployed: 3})
	if unknown <= exact {
		t.Errorf("unknown (%d) should rank above exact (%d)", unknown, exact)
	}
	if exact <= 0 {
		t.Errorf("exact undeployed rank = %d, want > 0", exact)
	}
}

func TestSortedRepoIndices(t *testing.T) {
	repos := []model.Repo{
		{Name: "bravo", Health: model.HealthGreen, Undeployed: 0, Dependabot: model.AlertCounts{Low: 1}},
		{Name: "alpha", Health: model.HealthRed, Undeployed: 5, Dependabot: model.AlertCounts{Critical: 3}},
		{Name: "charlie", Health: model.HealthYellow, Undeployed: -1, Dependabot: model.AlertCounts{Moderate: 2}},
	}

	// Health: red(alpha) > yellow(charlie) > green(bravo).
	if got := sortedRepoIndices(repos, sortHealth); got[0] != 1 || got[1] != 2 || got[2] != 0 {
		t.Errorf("sortHealth = %v, want [1 2 0]", got)
	}
	// Name: alpha, bravo, charlie.
	if got := sortedRepoIndices(repos, sortName); got[0] != 1 || got[1] != 0 || got[2] != 2 {
		t.Errorf("sortName = %v, want [1 0 2]", got)
	}
	// Undeployed: charlie(-1 unknown) > alpha(5) > bravo(0).
	if got := sortedRepoIndices(repos, sortUndeployed); got[0] != 2 || got[1] != 1 || got[2] != 0 {
		t.Errorf("sortUndeployed = %v, want [2 1 0]", got)
	}
	// Alerts: alpha(3) > charlie(2) > bravo(1).
	if got := sortedRepoIndices(repos, sortAlerts); got[0] != 1 || got[1] != 2 || got[2] != 0 {
		t.Errorf("sortAlerts = %v, want [1 2 0]", got)
	}
}

func TestSortedRepoIndicesNameTiebreak(t *testing.T) {
	// Equal health falls back to name order.
	repos := []model.Repo{
		{Name: "zebra", Health: model.HealthGreen},
		{Name: "apple", Health: model.HealthGreen},
	}
	if got := sortedRepoIndices(repos, sortHealth); got[0] != 1 || got[1] != 0 {
		t.Errorf("tiebreak = %v, want [1 0]", got)
	}
}

func TestBodyCapacity(t *testing.T) {
	var m Model
	m.height = 0
	if got := m.bodyCapacity(); got != 100000 {
		t.Errorf("bodyCapacity(h=0) = %d, want 100000", got)
	}
	m.height = 20
	if got := m.bodyCapacity(); got != 12 {
		t.Errorf("bodyCapacity(h=20) = %d, want 12", got)
	}
	// Tiny height floors at 1.
	m.height = 3
	if got := m.bodyCapacity(); got != 1 {
		t.Errorf("bodyCapacity(h=3) = %d, want 1", got)
	}
}

func TestIsExpanded(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	// Default (no entry) is expanded.
	if !m.isExpanded("never-seen") {
		t.Error("unknown org should default to expanded")
	}
	m.expanded["folded"] = false
	if m.isExpanded("folded") {
		t.Error("explicitly collapsed org should report not expanded")
	}
	m.expanded["open"] = true
	if !m.isExpanded("open") {
		t.Error("explicitly expanded org should report expanded")
	}
}

func TestSelectedRepo(t *testing.T) {
	m := newRichModel()
	// cursor 0 is an org row -> nil.
	m.cursor = 0
	if m.selectedRepo() != nil {
		t.Error("selectedRepo on an org row should be nil")
	}
	// cursor 1 is a repo row.
	m.cursor = 1
	if r := m.selectedRepo(); r == nil || r.Name != "red-repo" {
		t.Errorf("selectedRepo on repo row = %v, want red-repo", r)
	}
	// Out-of-range cursor is nil.
	m.cursor = len(m.rows) + 5
	if m.selectedRepo() != nil {
		t.Error("selectedRepo with out-of-range cursor should be nil")
	}
}

func TestGraphqlOK(t *testing.T) {
	var m Model
	if !m.graphqlOK() {
		t.Error("nil-state model should report graphqlOK")
	}
	m.state = richState()
	m.state.RateLimit.GraphQLRemaining = lowGraphQLThreshold + 1
	if !m.graphqlOK() {
		t.Error("above-threshold should be ok")
	}
	m.state.RateLimit.GraphQLRemaining = lowGraphQLThreshold
	if m.graphqlOK() {
		t.Error("at-threshold should not be ok")
	}
}

func TestSetOnlyAttention(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.SetOnlyAttention(true)
	if !m.onlyAttention {
		t.Error("SetOnlyAttention(true) should set the flag")
	}
}
