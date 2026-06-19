package ghclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// orgRepoGQLMixed returns three repos in one org: a normal "keeper" repo, a
// fork (must be skipped), and an archived repo (skipped under ExcludeArchived).
const orgRepoGQLMixed = `{"data":{"organization":{"repositories":{
  "pageInfo":{"hasNextPage":false,"endCursor":""},
  "nodes":[
    {
      "name":"keeper","url":"https://github.com/org1/keeper","isArchived":false,"isPrivate":true,"isFork":false,
      "hasVulnerabilityAlertsEnabled":true,
      "defaultBranchRef":{"name":"main","target":{"oid":"deadbee","abbreviatedOid":"deadbee","statusCheckRollup":{"state":"SUCCESS"}}},
      "latestRelease":{"tagName":"v1.0.0"},
      "refs":{"nodes":[]},
      "vulnerabilityAlerts":{"totalCount":0,"nodes":[]},
      "pullRequests":{"totalCount":0,"nodes":[]}
    },
    {
      "name":"a-fork","url":"https://github.com/org1/a-fork","isArchived":false,"isPrivate":false,"isFork":true,
      "hasVulnerabilityAlertsEnabled":false,
      "defaultBranchRef":{"name":"main","target":{"oid":"f00","abbreviatedOid":"f00","statusCheckRollup":{"state":"SUCCESS"}}},
      "refs":{"nodes":[]},
      "vulnerabilityAlerts":{"totalCount":0,"nodes":[]},
      "pullRequests":{"totalCount":0,"nodes":[]}
    },
    {
      "name":"old-archived","url":"https://github.com/org1/old-archived","isArchived":true,"isPrivate":true,"isFork":false,
      "hasVulnerabilityAlertsEnabled":false,
      "defaultBranchRef":{"name":"main","target":{"oid":"a1b","abbreviatedOid":"a1b","statusCheckRollup":{"state":"SUCCESS"}}},
      "refs":{"nodes":[]},
      "vulnerabilityAlerts":{"totalCount":0,"nodes":[]},
      "pullRequests":{"totalCount":0,"nodes":[]}
    }
  ]
}},"rateLimit":{"remaining":4999,"limit":5000,"resetAt":"2026-01-01T00:00:00Z","cost":3}}}`

func mixedFixtureServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `{"login":"me"}`) })
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `[{"login":"org1"}]`) })
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "user(login") {
			io.WriteString(w, userRepoGQL)
			return
		}
		io.WriteString(w, orgRepoGQLMixed)
	})
	// Only the keeper repo should ever be probed (fork + archived are filtered
	// out before the scan stage). Code-scanning: on with two alerts. Secret-
	// scanning: 404 -> off.
	mux.HandleFunc("/repos/org1/keeper/code-scanning/alerts", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[{},{}]`)
	})
	mux.HandleFunc("/repos/org1/keeper/secret-scanning/alerts", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"message":"disabled"}`)
	})
	mux.HandleFunc("/orgs/org1/settings/billing/advanced-security", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"total_advanced_security_committers":0}`)
	})
	mux.HandleFunc("/rate_limit", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"resources":{"core":{"remaining":4990,"limit":5000,"reset":0},"graphql":{"remaining":4998,"limit":5000,"reset":0}}}`)
	})
	return httptest.NewServer(mux)
}

// TestFetchStateFiltersForksAndArchived asserts that a fork and an archived repo
// are dropped, leaving only the keeper, and that the keeper's scan tri-state and
// counts come from the per-repo probes (code on/2, secret off/0).
func TestFetchStateFiltersForksAndArchived(t *testing.T) {
	srv := mixedFixtureServer()
	defer srv.Close()
	c := testClient(t, srv)

	st, err := c.FetchState(context.Background(), Options{
		IncludePersonal: true, ExcludeArchived: true, NoCache: true, CacheTTL: time.Minute,
	})
	if err != nil {
		t.Fatalf("FetchState: %v", err)
	}

	var org *model.Owner
	for i := range st.Owners {
		if st.Owners[i].Name == "org1" {
			org = &st.Owners[i]
		}
	}
	if org == nil {
		t.Fatalf("org1 not found; owners=%+v", st.Owners)
	}
	if len(org.Repos) != 1 {
		t.Fatalf("org1 repos = %d, want 1 (fork + archived dropped)", len(org.Repos))
	}
	r := org.Repos[0]
	if r.Name != "keeper" {
		t.Fatalf("kept repo = %q, want keeper", r.Name)
	}
	if r.CodeScan != model.ScanOn || r.CodeScanning != 2 {
		t.Errorf("code scan = %v count=%d, want on/2", r.CodeScan, r.CodeScanning)
	}
	if r.SecretScan != model.ScanOff || r.SecretScanning != 0 {
		t.Errorf("secret scan = %v count=%d, want off/0", r.SecretScan, r.SecretScanning)
	}
}
