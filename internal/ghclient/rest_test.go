package ghclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func TestBilling(t *testing.T) {
	mux := http.NewServeMux()
	// Both product queries hit the same path; differentiate on the query string.
	mux.HandleFunc("/orgs/o1/settings/billing/advanced-security", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("advanced_security_product") == "secret_protection" {
			io.WriteString(w, `{"total_advanced_security_committers":8}`)
			return
		}
		io.WriteString(w, `{"total_advanced_security_committers":5}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := testClient(t, srv)

	b, err := c.billing(context.Background(), "o1")
	if err != nil {
		t.Fatalf("billing: %v", err)
	}
	if !b.Known {
		t.Error("Known = false, want true")
	}
	if b.SecretProtectionCommitters != 8 {
		t.Errorf("SecretProtectionCommitters = %d, want 8", b.SecretProtectionCommitters)
	}
	if b.CodeSecurityCommitters != 5 {
		t.Errorf("CodeSecurityCommitters = %d, want 5", b.CodeSecurityCommitters)
	}
}

func TestBillingForbidden(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/orgs/o1/settings/billing/advanced-security", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"message":"forbidden"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := testClient(t, srv)

	_, err := c.billing(context.Background(), "o1")
	if !errors.Is(err, errFeatureUnavailable) {
		t.Fatalf("billing 403 err = %v, want errFeatureUnavailable", err)
	}
}

func TestRepoScanState(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantState model.ScanState
		wantCount int
	}{
		{"on with two alerts", http.StatusOK, `[{},{}]`, model.ScanOn, 2},
		{"off on 404", http.StatusNotFound, `{"message":"disabled"}`, model.ScanOff, 0},
		{"unknown on 500", http.StatusInternalServerError, `{"message":"boom"}`, model.ScanUnknown, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/o/r/code-scanning/alerts", func(w http.ResponseWriter, _ *http.Request) {
				if tt.status != http.StatusOK {
					w.WriteHeader(tt.status)
				}
				io.WriteString(w, tt.body)
			})
			srv := httptest.NewServer(mux)
			c := testClient(t, srv)

			state, count := c.repoScanState(context.Background(), "o", "r", "code-scanning")
			srv.Close()

			if state != tt.wantState || count != tt.wantCount {
				t.Errorf("got %v/%d, want %v/%d", state, count, tt.wantState, tt.wantCount)
			}
		})
	}
}

func TestNextPageLink(t *testing.T) {
	tests := []struct {
		name string
		link string
		want string
	}{
		{"empty", "", ""},
		{
			"next present with query",
			`<https://api.github.com/user/repos?page=2&per_page=100>; rel="next", <https://api.github.com/user/repos?page=5>; rel="last"`,
			"/user/repos?page=2&per_page=100",
		},
		{
			"no next",
			`<https://api.github.com/user/repos?page=1>; rel="prev"`,
			"",
		},
		{
			"multiple rels pick next",
			`<https://api.github.com/x?page=1>; rel="prev", <https://api.github.com/x?page=3>; rel="next", <https://api.github.com/x?page=9>; rel="last"`,
			"/x?page=3",
		},
		{
			"next with no query",
			`<https://api.github.com/path>; rel="next"`,
			"/path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nextPageLink(tt.link); got != tt.want {
				t.Errorf("nextPageLink(%q) = %q, want %q", tt.link, got, tt.want)
			}
		})
	}
}
