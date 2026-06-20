package tui

import (
	"strings"
	"testing"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func archivedState() *model.State {
	return &model.State{
		Owners: []model.Owner{{
			Name: "acme",
			Repos: []model.Repo{
				{Name: "frozen-repo", Archived: true, Health: model.HealthGray, ShortSHA: "abc1234"},
				{Name: "broken-repo", Health: model.HealthRed, ShortSHA: "def5678"},
				{Name: "live-repo", Health: model.HealthGreen, ShortSHA: "0a1b2c3"},
			},
		}},
		RateLimit: model.RateLimit{RESTRemaining: 5000, RESTLimit: 5000},
	}
}

func TestViewShowsArchivedGlyphAndLegend(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.state = archivedState()
	m.rebuildRows()
	out := m.View()
	if !strings.Contains(out, "[AR]") {
		t.Errorf("View() should show [AR] for the archived repo\n---\n%s", out)
	}
	if !strings.Contains(out, "[AR] archived") {
		t.Error("footer legend should document [AR] archived")
	}
}

func TestOnlyAttentionHidesArchived(t *testing.T) {
	m := New(nil, ghclient.Options{}, false, 0)
	m.onlyAttention = true
	m.state = archivedState()
	m.rebuildRows()
	out := m.View()
	if strings.Contains(out, "frozen-repo") {
		t.Error("attention mode should hide archived repos")
	}
	if strings.Contains(out, "live-repo") {
		t.Error("attention mode should hide green repos")
	}
	if !strings.Contains(out, "broken-repo") {
		t.Error("attention mode should still show red repos")
	}
}
