package ghclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRepoDetail(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/compare/aaa...bbb", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"ahead_by":3}`)
	})
	mux.HandleFunc("/repos/o/r/dependabot/alerts", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[{"html_url":"https://example/alert/1","security_advisory":{"severity":"high","summary":"bad dep"},"dependency":{"package":{"name":"left-pad"}}}]`)
	})
	mux.HandleFunc("/repos/o/r/pulls", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[{"number":7,"title":"fix things","html_url":"https://example/pull/7","draft":true}]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := testClient(t, srv)

	d := c.FetchRepoDetail(context.Background(), "o", "r", "aaa", "bbb")

	if d.AheadBy != 3 || !d.AheadKnown {
		t.Errorf("ahead: got %d known=%v, want 3/true", d.AheadBy, d.AheadKnown)
	}
	if len(d.Alerts) != 1 {
		t.Fatalf("alerts = %d, want 1", len(d.Alerts))
	}
	a := d.Alerts[0]
	if a.Package != "left-pad" || a.Severity != "high" || a.Summary != "bad dep" || a.URL != "https://example/alert/1" {
		t.Errorf("alert fields wrong: %+v", a)
	}
	if len(d.PRs) != 1 {
		t.Fatalf("PRs = %d, want 1", len(d.PRs))
	}
	p := d.PRs[0]
	if p.Number != 7 || p.Title != "fix things" || p.URL != "https://example/pull/7" || !p.Draft {
		t.Errorf("PR fields wrong: %+v", p)
	}
}

// TestFetchRepoDetailSameSHA covers the tagSHA==headSHA branch: nothing to
// compare, but AheadBy is known to be 0.
func TestFetchRepoDetailSameSHA(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/dependabot/alerts", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[]`)
	})
	mux.HandleFunc("/repos/o/r/pulls", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := testClient(t, srv)

	d := c.FetchRepoDetail(context.Background(), "o", "r", "same", "same")
	if d.AheadBy != 0 || !d.AheadKnown {
		t.Errorf("same-SHA ahead: got %d known=%v, want 0/true", d.AheadBy, d.AheadKnown)
	}
}

// TestFetchRepoDetailEmptySHA covers the empty-SHA branch: nothing to compare
// and AheadBy stays unknown.
func TestFetchRepoDetailEmptySHA(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/dependabot/alerts", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[]`)
	})
	mux.HandleFunc("/repos/o/r/pulls", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[]`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := testClient(t, srv)

	d := c.FetchRepoDetail(context.Background(), "o", "r", "", "head")
	if d.AheadKnown {
		t.Errorf("empty-SHA AheadKnown = true, want false")
	}
}
