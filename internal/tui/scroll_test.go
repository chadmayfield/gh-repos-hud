package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func manyReposState(n int) *model.State {
	repos := make([]model.Repo, n)
	for i := range repos {
		repos[i] = model.Repo{Name: fmt.Sprintf("repo-%02d", i), CI: model.CISuccess, Health: model.HealthGreen}
	}
	return &model.State{Owners: []model.Owner{{Name: "org", Repos: repos}}}
}

func TestScrollWindow(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.state = manyReposState(30)
	m.height = 15 // bodyCapacity = 15 - 8 = 7
	m.rebuildRows()

	if got := m.bodyCapacity(); got != 7 {
		t.Fatalf("bodyCapacity = %d, want 7", got)
	}
	// Total rows = 1 org header + 30 repos = 31.
	if len(m.rows) != 31 {
		t.Fatalf("rows = %d, want 31", len(m.rows))
	}

	// Jump the cursor near the bottom; the window must follow.
	m.cursor = 28
	m.clampScroll()
	if m.scroll != 28-7+1 {
		t.Fatalf("scroll = %d, want %d", m.scroll, 28-7+1)
	}

	out := m.View()
	// Indicator reflects the window; a far-down repo is visible, a top one is not.
	if !strings.Contains(out, fmt.Sprintf("rows %d-%d of 31", m.scroll+1, m.scroll+7)) {
		t.Errorf("missing/incorrect scroll indicator in:\n%s", out)
	}
	if !strings.Contains(out, "repo-27") {
		t.Errorf("expected near-cursor repo repo-27 to be visible")
	}
	if strings.Contains(out, "repo-00") {
		t.Errorf("top repo repo-00 should be scrolled out of view")
	}
}
