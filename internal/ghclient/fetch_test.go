package ghclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// rewriteTransport sends every request to the test server, ignoring the host
// go-gh computes (api.github.com / graphql).
type rewriteTransport struct{ scheme, host string }

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.scheme
	req.URL.Host = t.host
	req.Host = t.host
	return http.DefaultTransport.RoundTrip(req)
}

func testClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	u, _ := url.Parse(server.URL)
	tr := &rewriteTransport{scheme: u.Scheme, host: u.Host}
	rest, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "x", Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "x", Transport: tr})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{rest: rest, gql: gql}
}

const orgRepoGQL = `{"data":{"organization":{"repositories":{
  "pageInfo":{"hasNextPage":false,"endCursor":""},
  "nodes":[{
    "name":"repo1","url":"https://github.com/org1/repo1","isArchived":false,"isPrivate":true,"isFork":false,
    "hasVulnerabilityAlertsEnabled":true,
    "defaultBranchRef":{"name":"main","target":{"oid":"abc1234","abbreviatedOid":"abc1234","statusCheckRollup":{"state":"SUCCESS"}}},
    "latestRelease":{"tagName":"v1.0.0"},
    "refs":{"nodes":[{"name":"v1.0.0","target":{"__typename":"Commit","oid":"abc1234"}},{"name":"mvp-1","target":{"__typename":"Commit","oid":"old"}}]},
    "vulnerabilityAlerts":{"totalCount":1,"nodes":[{"securityVulnerability":{"severity":"HIGH"}}]},
    "pullRequests":{"totalCount":1,"nodes":[{"isDraft":false,"mergeable":"MERGEABLE","author":{"login":"dependabot[bot]","__typename":"Bot"},"commits":{"nodes":[{"commit":{"statusCheckRollup":{"state":"SUCCESS"}}}]}}]}
  }]
}},"rateLimit":{"remaining":4999,"limit":5000,"resetAt":"2026-01-01T00:00:00Z","cost":3}}}`

const userRepoGQL = `{"data":{"user":{"repositories":{"pageInfo":{"hasNextPage":false,"endCursor":""},"nodes":[]}},"rateLimit":{"remaining":4998,"limit":5000,"cost":1}}}`

func fixtureServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `{"login":"me"}`) })
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `[{"login":"org1"}]`) })
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "user(login") {
			io.WriteString(w, userRepoGQL)
			return
		}
		io.WriteString(w, orgRepoGQL)
	})
	mux.HandleFunc("/orgs/org1/code-scanning/alerts", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `[]`) })
	mux.HandleFunc("/orgs/org1/secret-scanning/alerts", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden) // disabled -> should degrade to a warning
		io.WriteString(w, `{"message":"secret scanning disabled"}`)
	})
	mux.HandleFunc("/orgs/org1/settings/billing/advanced-security", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"total_advanced_security_committers":0}`)
	})
	mux.HandleFunc("/rate_limit", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"resources":{"core":{"remaining":4990,"limit":5000,"reset":0},"graphql":{"remaining":4998,"limit":5000,"reset":0}}}`)
	})
	return httptest.NewServer(mux)
}

func TestFetchStateMerge(t *testing.T) {
	srv := fixtureServer()
	defer srv.Close()
	c := testClient(t, srv)

	st, err := c.FetchState(context.Background(), Options{IncludePersonal: true, ExcludeArchived: true, NoCache: true, CacheTTL: time.Minute})
	if err != nil {
		t.Fatalf("FetchState: %v", err)
	}

	// Two owners: me (user, 0 repos) + org1 (1 repo).
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
		t.Fatalf("org1 repos = %d, want 1", len(org.Repos))
	}
	r := org.Repos[0]
	if r.Name != "repo1" || r.ShortSHA != "abc1234" {
		t.Errorf("repo basics wrong: %+v", r)
	}
	if r.CI != model.CISuccess {
		t.Errorf("CI = %v, want success", r.CI)
	}
	if r.Dependabot.High != 1 || r.Health != model.HealthRed {
		t.Errorf("alert/health wrong: high=%d health=%v", r.Dependabot.High, r.Health)
	}
	if r.LatestTag != "v1.0.0" || r.Undeployed != 0 {
		t.Errorf("tag/undeployed wrong: tag=%q undeployed=%d (mvp-1 should be ignored)", r.LatestTag, r.Undeployed)
	}
	if r.PRs.Bot != 1 || r.PRs.Mergeable != 1 || r.PRs.CIGreen != 1 {
		t.Errorf("PR classification wrong: %+v", r.PRs)
	}
	// GraphQL cost summed across owners (org 3 + user 1).
	if st.RateLimit.GraphQLCost != 4 {
		t.Errorf("GraphQLCost = %d, want 4", st.RateLimit.GraphQLCost)
	}
	// Secret-scanning 403 degraded to a warning, not a failure.
	foundWarn := false
	for _, w := range st.Warnings {
		if w.Feature == "secret-scanning" {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Errorf("expected a secret-scanning degradation warning; warnings=%+v", st.Warnings)
	}
}
