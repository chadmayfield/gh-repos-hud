package tui

import (
	"strings"
	"testing"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func sampleState() *model.State {
	return &model.State{
		Owners: []model.Owner{{
			Name: "acme",
			Repos: []model.Repo{
				{Name: "acme-status-page", ShortSHA: "f6c3ac1", LatestTag: "v0.1.0", CI: model.CISuccess,
					Dependabot: model.AlertCounts{High: 1, Moderate: 1, Low: 1}, Health: model.HealthRed,
					CodeScanEnabled: true, SecretScanEnabled: true, Undeployed: 0},
				{Name: "acme-app", ShortSHA: "0df2c22", CI: model.CISuccess, Untagged: true, Health: model.HealthGreen},
			},
		}},
		RateLimit: model.RateLimit{RESTRemaining: 4966, RESTLimit: 5000, GraphQLRemaining: 4700, GraphQLLimit: 5000},
	}
}

func TestViewRendersRepos(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.state = sampleState()
	m.rebuildRows()

	out := m.View()
	for _, want := range []string{"acme", "acme-status-page", "acme-app", "[!!]", "[OK]", "REST 4966/5000", "q quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q\n---\n%s", want, out)
		}
	}
}

func TestRebuildRowsCollapse(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.state = sampleState()
	m.rebuildRows()
	if len(m.rows) != 3 { // 1 org + 2 repos
		t.Fatalf("expected 3 rows expanded, got %d", len(m.rows))
	}
	m.expanded["acme"] = false
	m.rebuildRows()
	if len(m.rows) != 1 { // just the org header
		t.Fatalf("expected 1 row collapsed, got %d", len(m.rows))
	}
}

func TestRebuildRowsOnlyAttention(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.state = sampleState()
	m.onlyAttention = true
	m.rebuildRows()
	// org header + the one red repo (green one filtered out)
	if len(m.rows) != 2 {
		t.Fatalf("expected 2 rows in attention mode, got %d", len(m.rows))
	}
}
