package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

func TestTagOrDash(t *testing.T) {
	if got := tagOrDash(""); got != "-" {
		t.Errorf("tagOrDash(\"\")=%q want -", got)
	}
	if got := tagOrDash("v1.2.3"); got != "v1.2.3" {
		t.Errorf("tagOrDash(v1.2.3)=%q", got)
	}
}

func TestSetVersionInfo(t *testing.T) {
	SetVersionInfo("1.2.3", "abc123", "2026-01-01")
	if versionStr != "1.2.3" || commitStr != "abc123" || dateStr != "2026-01-01" {
		t.Errorf("version info not set: %q %q %q", versionStr, commitStr, dateStr)
	}
	if rootCmd.Version != "1.2.3" {
		t.Errorf("rootCmd.Version=%q", rootCmd.Version)
	}
}

func TestRenderText(t *testing.T) {
	st := &model.State{
		Owners: []model.Owner{{
			Name:    "acme",
			Billing: model.Billing{Known: true, SecretProtectionCommitters: 3, CodeSecurityCommitters: 2},
			Repos: []model.Repo{
				{Name: "api", Private: true, ShortSHA: "abc1234", CI: model.CISuccess, LatestTag: "v1.0.0",
					Dependabot: model.AlertCounts{High: 1}, CodeScan: model.ScanOn, CodeScanning: 2,
					SecretScan: model.ScanOff, Health: model.HealthRed, PRs: model.PRStats{Bot: 1, Human: 2}},
				{Name: "site", Private: false, ShortSHA: "def5678", CI: model.CINone, Untagged: true,
					CodeScan: model.ScanUnknown, SecretScan: model.ScanOn, Health: model.HealthGreen},
			},
		}},
		Warnings:  []model.Warning{{Owner: "acme", Feature: "billing", Reason: "no GHAS"}},
		RateLimit: model.RateLimit{RESTRemaining: 4990, RESTLimit: 5000, GraphQLRemaining: 4900, GraphQLLimit: 5000, GraphQLCost: 5},
	}
	var buf bytes.Buffer
	if err := renderText(&buf, st); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"acme", "GHAS: secret=3 code=2", "api", "site (pub)", "v1.0.0",
		"off", "untagged", "1 warning(s)", "no GHAS", "rate: REST 4990/5000", "cost 5 pts",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("renderText output missing %q\n---\n%s", want, out)
		}
	}
}

func TestResolveOptionsDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // no config file -> defaults
	opts, interval, port := resolveOptions(rootCmd)
	if !opts.IncludePersonal || !opts.ExcludeArchived {
		t.Errorf("resolveOptions defaults wrong: %+v", opts)
	}
	if interval == 0 || port == 0 {
		t.Errorf("interval/port should be non-zero: %v %d", interval, port)
	}
}

func TestResolveOptionsArchived(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	flagArchived = true
	t.Cleanup(func() { flagArchived = false })
	opts, _, _ := resolveOptions(rootCmd)
	if opts.ExcludeArchived {
		t.Error("--archived should set ExcludeArchived=false")
	}
}
