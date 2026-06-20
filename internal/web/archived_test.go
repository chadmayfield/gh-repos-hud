package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func TestHandleIndexArchived(t *testing.T) {
	s, err := newServer(nil, ghclient.Options{}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	s.state.Store(&model.State{
		FetchedAt: time.Now(),
		Owners: []model.Owner{{
			Name: "acme",
			Repos: []model.Repo{{
				Name: "frozen", Archived: true, Health: model.HealthGray,
				ShortSHA: "abc1234", URL: "https://github.com/acme/frozen",
			}},
		}},
		RateLimit: model.RateLimit{RESTRemaining: 5000, RESTLimit: 5000},
	})

	rr := httptest.NewRecorder()
	s.handleIndex(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rr.Body.String()
	if !strings.Contains(body, "[AR]") {
		t.Error("archived repo should render the [AR] glyph")
	}
	if !strings.Contains(body, `class="h-archived"`) {
		t.Error("archived repo row should carry the h-archived class")
	}
}
