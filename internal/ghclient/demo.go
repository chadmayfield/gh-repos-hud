package ghclient

import (
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// demoState returns a synthetic snapshot for `--demo`: screenshots and trying
// the HUD without touching real data. Every org and repo here is fictional; the
// numbers are chosen to exercise each column and all four health states (red,
// yellow, green, gray). It is rendered through the real front-ends, so it stays
// accurate to the actual UI.
func demoState() *model.State {
	st := &model.State{
		FetchedAt: time.Now(),
		RateLimit: model.RateLimit{
			RESTRemaining: 4987, RESTLimit: 5000,
			GraphQLRemaining: 4972, GraphQLLimit: 5000, GraphQLCost: 6,
		},
		Owners: []model.Owner{
			{
				Name:    "acme-corp",
				Billing: model.Billing{Known: true, SecretProtectionCommitters: 8, CodeSecurityCommitters: 8},
				Repos: finalizeRepos("acme-corp", true, []model.Repo{
					// RED: CI failing + critical/high Dependabot alerts.
					{Name: "payments-api", ShortSHA: "9f3a1c7", CI: model.CIFailure, LatestTag: "v2.4.1",
						Dependabot: model.AlertCounts{Critical: 1, High: 2, Moderate: 1},
						CodeScan:   model.ScanOn, CodeScanning: 3, SecretScan: model.ScanOn,
						Undeployed: 4, PRs: model.PRStats{Bot: 2, Human: 1}},
					// RED: open high-severity alert (CI green otherwise).
					{Name: "ledger-core", ShortSHA: "1b8d40e", CI: model.CISuccess, LatestTag: "v4.0.0",
						Dependabot: model.AlertCounts{High: 1},
						CodeScan:   model.ScanOn, CodeScanning: 2, SecretScan: model.ScanOn, SecretScanning: 1,
						PRs: model.PRStats{Bot: 1}},
					// YELLOW: undeployed changes + moderate alerts.
					{Name: "checkout-web", ShortSHA: "4c2e9aa", CI: model.CISuccess, LatestTag: "v1.8.0",
						Dependabot: model.AlertCounts{Moderate: 2, Low: 1},
						CodeScan:   model.ScanOff, SecretScan: model.ScanOn,
						Undeployed: -1, PRs: model.PRStats{Bot: 1, Human: 2}},
					// YELLOW: CI in progress.
					{Name: "fraud-detector", ShortSHA: "7a0f521", CI: model.CIPending, LatestTag: "v0.9.2",
						CodeScan: model.ScanOn, SecretScan: model.ScanOff},
					// GREEN: clean, deployed, scanning on.
					{Name: "auth-service", ShortSHA: "e51d8b3", CI: model.CISuccess, LatestTag: "v3.1.0",
						CodeScan: model.ScanOn, SecretScan: model.ScanOn},
					// GREEN: untagged is informational, not a problem.
					{Name: "infra-terraform", ShortSHA: "c9740f2", CI: model.CISuccess, Untagged: true,
						CodeScan: model.ScanOn, SecretScan: model.ScanOn},
				}),
			},
			{
				Name:   "acme-labs",
				IsUser: true,
				Repos: finalizeRepos("acme-labs", false, []model.Repo{
					// YELLOW: undeployed commits since the last tag.
					{Name: "dashboard-ui", Private: false, ShortSHA: "2d6b119", CI: model.CISuccess, LatestTag: "v0.3.0",
						Dependabot: model.AlertCounts{Low: 2},
						CodeScan:   model.ScanOff, SecretScan: model.ScanOn,
						Undeployed: 2, PRs: model.PRStats{Human: 1}},
					// GREEN: public, secret scanning free + clean.
					{Name: "cli-helpers", Private: false, ShortSHA: "8e1aa07", CI: model.CISuccess, LatestTag: "v1.2.0",
						CodeScan: model.ScanOn, SecretScan: model.ScanOn},
					// GREEN: untagged side project.
					{Name: "ml-toolkit", Private: false, ShortSHA: "f47c3d0", CI: model.CISuccess, Untagged: true,
						CodeScan: model.ScanOff, SecretScan: model.ScanOn},
					// GREEN: a quiet side project — no CI run, no alerts, nothing to flag.
					{Name: "scratchpad", Private: false, ShortSHA: "0a55e2b", CI: model.CINone, Untagged: true,
						CodeScan: model.ScanOff, SecretScan: model.ScanOff},
				}),
			},
		},
		Warnings: []model.Warning{
			{Owner: "acme-labs", Feature: "billing", Reason: "no GHAS or not an admin"},
		},
	}
	return st
}

// finalizeRepos fills the derived fields (URL, status names, PR totals, health)
// the same way the real fetch path does, so the demo renders identically.
// private sets each repo's visibility (corp repos private, personal repos public).
func finalizeRepos(owner string, private bool, repos []model.Repo) []model.Repo {
	for i := range repos {
		r := &repos[i]
		r.Private = private
		r.URL = "https://github.com/" + owner + "/" + r.Name
		if r.DefaultBranch == "" {
			r.DefaultBranch = "main"
		}
		if r.ShortSHA == "" {
			r.ShortSHA = "-"
		}
		if r.LatestTag != "" && r.LatestRelease == "" {
			r.LatestRelease = r.LatestTag
		}
		r.DependabotEnabled = true
		r.PRs.Total = r.PRs.Bot + r.PRs.Human
		if r.PRs.Mergeable == 0 {
			r.PRs.Mergeable = r.PRs.Human
		}
		if r.PRs.CIGreen == 0 {
			r.PRs.CIGreen = r.PRs.Bot
		}
		r.CIName = r.CI.String()
		r.CodeScanName = r.CodeScan.String()
		r.SecretScanName = r.SecretScan.String()
		r.Health = model.ComputeHealth(*r)
		r.HealthName = r.Health.String()
	}
	return repos
}
