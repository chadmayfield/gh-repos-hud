package ghclient

import (
	"context"
	"fmt"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

type ghCompare struct {
	AheadBy int `json:"ahead_by"`
}

type ghAlert struct {
	HTMLURL          string `json:"html_url"`
	SecurityAdvisory struct {
		Severity string `json:"severity"`
		Summary  string `json:"summary"`
	} `json:"security_advisory"`
	Dependency struct {
		Package struct {
			Name string `json:"name"`
		} `json:"package"`
	} `json:"dependency"`
}

type ghPull struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Draft   bool   `json:"draft"`
}

// FetchRepoDetail lazily fetches per-repo detail (exact ahead_by, the open
// alert list, and open PRs) via REST. Used only when a repo is drilled into,
// so it costs nothing for the list view. Each piece degrades independently.
func (c *Client) FetchRepoDetail(ctx context.Context, owner, name, tagSHA, headSHA string) model.RepoDetail {
	var d model.RepoDetail

	switch {
	case tagSHA == "" || headSHA == "":
		// nothing to compare
	case tagSHA == headSHA:
		d.AheadBy, d.AheadKnown = 0, true
	default:
		cmp, err := restGet[ghCompare](ctx, c.rest,
			fmt.Sprintf("repos/%s/%s/compare/%s...%s", owner, name, tagSHA, headSHA))
		if err == nil {
			d.AheadBy, d.AheadKnown = cmp.AheadBy, true
		}
	}

	if alerts, _, err := restListAll[ghAlert](ctx, c.rest,
		fmt.Sprintf("repos/%s/%s/dependabot/alerts?state=open&per_page=100", owner, name)); err == nil {
		for _, a := range alerts {
			d.Alerts = append(d.Alerts, model.AlertDetail{
				Package:  a.Dependency.Package.Name,
				Severity: a.SecurityAdvisory.Severity,
				Summary:  a.SecurityAdvisory.Summary,
				URL:      a.HTMLURL,
			})
		}
	}

	if prs, _, err := restListAll[ghPull](ctx, c.rest,
		fmt.Sprintf("repos/%s/%s/pulls?state=open&per_page=50", owner, name)); err == nil {
		for _, p := range prs {
			d.PRs = append(d.PRs, model.PRDetail{Number: p.Number, Title: p.Title, URL: p.HTMLURL, Draft: p.Draft})
		}
	}

	return d
}
