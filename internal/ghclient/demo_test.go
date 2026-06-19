package ghclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func TestDemoState(t *testing.T) {
	st := demoState()
	if st == nil {
		t.Fatal("demoState returned nil")
	}

	if len(st.Owners) != 2 {
		t.Fatalf("owners = %d, want 2", len(st.Owners))
	}

	// Collect every repo and assert demo invariants.
	seen := map[model.Health]bool{}
	for _, o := range st.Owners {
		for _, r := range o.Repos {
			seen[r.Health] = true

			if !strings.Contains(r.URL, "acme-") {
				t.Errorf("repo %s URL %q does not contain \"acme-\"", r.Name, r.URL)
			}
			if strings.Contains(r.URL, "apiaryos") || strings.Contains(r.URL, "mmh") {
				t.Errorf("repo %s URL %q leaks a real owner", r.Name, r.URL)
			}
			if r.CodeScanName == "" {
				t.Errorf("repo %s CodeScanName empty", r.Name)
			}
			if r.SecretScanName == "" {
				t.Errorf("repo %s SecretScanName empty", r.Name)
			}
			if r.HealthName == "" {
				t.Errorf("repo %s HealthName empty", r.Name)
			}
			if r.PRs.Total != r.PRs.Bot+r.PRs.Human {
				t.Errorf("repo %s PRs.Total=%d, want Bot+Human=%d", r.Name, r.PRs.Total, r.PRs.Bot+r.PRs.Human)
			}
		}
	}

	// The demo data exercises red, yellow, and green directly. Gray is the
	// archived/degraded state, which ComputeHealth only returns for an archived
	// repo; no demo repo is archived, so it is verified separately below rather
	// than asserted here against data that cannot produce it.
	for _, h := range []model.Health{model.HealthRed, model.HealthYellow, model.HealthGreen} {
		if !seen[h] {
			t.Errorf("health state %v never appears across demo repos", h)
		}
	}
}

// TestHealthGrayReachable pins the fourth health state: an archived repo rolls
// up to gray. The demo never marks a repo archived, so this guards the path the
// demo cannot reach on its own.
func TestHealthGrayReachable(t *testing.T) {
	if got := model.ComputeHealth(model.Repo{Archived: true}); got != model.HealthGray {
		t.Errorf("archived repo health = %v, want gray", got)
	}
}

func TestFinalizeRepos(t *testing.T) {
	repos := finalizeRepos("acme-corp", true, []model.Repo{
		{Name: "svc", CI: model.CISuccess, LatestTag: "v1.0.0", PRs: model.PRStats{Bot: 2, Human: 3}},
		{Name: "empty"},
	})

	r := repos[0]
	if r.URL != "https://github.com/acme-corp/svc" {
		t.Errorf("URL = %q", r.URL)
	}
	if !r.Private {
		t.Error("Private not set")
	}
	if r.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %q, want main", r.DefaultBranch)
	}
	if r.LatestRelease != "v1.0.0" {
		t.Errorf("LatestRelease = %q, want v1.0.0", r.LatestRelease)
	}
	if !r.DependabotEnabled {
		t.Error("DependabotEnabled not set")
	}
	if r.PRs.Total != 5 {
		t.Errorf("PRs.Total = %d, want 5", r.PRs.Total)
	}
	if r.PRs.Mergeable != 3 {
		t.Errorf("PRs.Mergeable = %d, want 3 (defaults to Human)", r.PRs.Mergeable)
	}
	if r.PRs.CIGreen != 2 {
		t.Errorf("PRs.CIGreen = %d, want 2 (defaults to Bot)", r.PRs.CIGreen)
	}
	if r.CIName == "" || r.CodeScanName == "" || r.SecretScanName == "" || r.HealthName == "" {
		t.Errorf("derived names not stamped: %+v", r)
	}

	// The empty repo defaults ShortSHA to "-".
	if repos[1].ShortSHA != "-" {
		t.Errorf("empty ShortSHA = %q, want -", repos[1].ShortSHA)
	}
}

// TestNilClientDemo proves --demo needs no gh auth: a nil *Client must return a
// non-nil state with no error and no panic, because FetchState short-circuits
// before touching the receiver.
func TestNilClientDemo(t *testing.T) {
	var c *Client
	st, err := c.FetchState(context.Background(), Options{Demo: true})
	if err != nil {
		t.Fatalf("nil-client demo err = %v, want nil", err)
	}
	if st == nil {
		t.Fatal("nil-client demo state = nil, want non-nil")
	}
	if len(st.Owners) != 2 {
		t.Errorf("owners = %d, want 2", len(st.Owners))
	}
}

func TestDefaultOptions(t *testing.T) {
	o := DefaultOptions()
	if !o.IncludePersonal {
		t.Error("IncludePersonal = false, want true")
	}
	if !o.ExcludeArchived {
		t.Error("ExcludeArchived = false, want true")
	}
	if o.CacheTTL != 5*time.Minute {
		t.Errorf("CacheTTL = %v, want 5m", o.CacheTTL)
	}
}
