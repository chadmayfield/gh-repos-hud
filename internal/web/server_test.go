package web

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	s, err := newServer(nil, ghclient.Options{}, 30*time.Second)
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	s.state.Store(&model.State{
		FetchedAt: time.Now(),
		Owners: []model.Owner{{
			Name: "acme",
			Repos: []model.Repo{
				{Name: "acme-status-page", URL: "https://github.com/acme/x", ShortSHA: "f6c3ac1",
					LatestTag: "v0.1.0", CI: model.CISuccess, Health: model.HealthRed,
					Dependabot: model.AlertCounts{High: 1}, CodeScanEnabled: true, SecretScanEnabled: true},
			},
		}},
		RateLimit: model.RateLimit{RESTRemaining: 4966, RESTLimit: 5000},
	})
	return s
}

func TestHandleIndex(t *testing.T) {
	s := testServer(t)
	rr := httptest.NewRecorder()
	s.handleIndex(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != 200 {
		t.Fatalf("status = %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"gh-repos-hud", "acme", "acme-status-page", `class="h-red"`, "[!!]", "REST 4966/5000"} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing %q", want)
		}
	}
}

func TestHandleState(t *testing.T) {
	s := testServer(t)
	rr := httptest.NewRecorder()
	s.handleState(rr, httptest.NewRequest("GET", "/api/state.json", nil))
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	if !strings.Contains(rr.Body.String(), `"acme-status-page"`) {
		t.Errorf("state json missing repo name")
	}
}
