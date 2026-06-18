package ghclient

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// versionTagRe matches release-style tags (v1.2.3, 1.2, v0.1.0) but not
// milestone tags like mvp-1. Only these count as deployment baselines.
var versionTagRe = regexp.MustCompile(`^v?\d+\.\d+`)

func isVersionTag(name string) bool { return versionTagRe.MatchString(name) }

// semverNumRe extracts the leading major.minor.patch from a tag name.
var semverNumRe = regexp.MustCompile(`^v?(\d+)\.(\d+)(?:\.(\d+))?`)

// semverKey parses a tag into a comparable [major, minor, patch]. ok is false
// for non-version tags. Pre-release/build suffixes are ignored.
func semverKey(name string) ([3]int, bool) {
	m := semverNumRe.FindStringSubmatch(name)
	if m == nil {
		return [3]int{}, false
	}
	var k [3]int
	for i := 0; i < 3; i++ {
		if m[i+1] != "" {
			k[i], _ = strconv.Atoi(m[i+1])
		}
	}
	return k, true
}

func semverGreater(a, b [3]int) bool {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

var botLogins = map[string]bool{
	"dependabot": true, "dependabot[bot]": true,
	"renovate": true, "renovate[bot]": true,
	"github-actions[bot]": true,
}

func ciFromState(s string) model.CIState {
	switch s {
	case "SUCCESS":
		return model.CISuccess
	case "FAILURE", "ERROR":
		return model.CIFailure
	case "PENDING", "EXPECTED":
		return model.CIPending
	default:
		return model.CINone
	}
}

func ciFromRollup(r *struct {
	State string `json:"state"`
}) model.CIState {
	if r == nil {
		return model.CINone
	}
	return ciFromState(r.State)
}

// buildRepo maps a GraphQL node plus the REST scan counts into a model.Repo,
// computing every derived field. codeAvail/secretAvail report whether the
// org-level scanning endpoints were reachable (used to mark coverage).
func buildRepo(n gqlRepoNode, codeCount, secretCount int, codeAvail, secretAvail bool) model.Repo {
	r := model.Repo{
		Name:              n.Name,
		URL:               n.URL,
		Private:           n.IsPrivate,
		Archived:          n.IsArchived,
		DependabotEnabled: n.HasVulnerabilityAlertsEnabled,
		CodeScanning:      codeCount,
		SecretScanning:    secretCount,
		CodeScanEnabled:   codeAvail,
		SecretScanEnabled: secretAvail,
	}

	if n.DefaultBranchRef != nil {
		r.DefaultBranch = n.DefaultBranchRef.Name
		r.ShortSHA = n.DefaultBranchRef.Target.AbbreviatedOid
		r.HeadSHA = n.DefaultBranchRef.Target.Oid
		r.CI = ciFromRollup(n.DefaultBranchRef.Target.StatusCheckRollup)
	}
	if r.ShortSHA == "" {
		r.ShortSHA = "-"
	}

	// Dependabot severity breakdown (total authoritative from totalCount).
	for _, a := range n.VulnerabilityAlerts.Nodes {
		switch strings.ToUpper(a.SecurityVulnerability.Severity) {
		case "CRITICAL":
			r.Dependabot.Critical++
		case "HIGH":
			r.Dependabot.High++
		case "MODERATE", "MEDIUM":
			r.Dependabot.Moderate++
		case "LOW":
			r.Dependabot.Low++
		}
	}

	// Latest *version* tag (semver-ish only) and its commit SHA. Milestone
	// tags like mvp-1 are ignored so they don't show as perpetually undeployed.
	if n.LatestRelease != nil && isVersionTag(n.LatestRelease.TagName) {
		r.LatestRelease = n.LatestRelease.TagName
	}
	// Pick the HIGHEST semver tag (not the newest by commit date) as the
	// deployment baseline — otherwise an out-of-order tag misreads "latest".
	var tagName, tagSHA string
	var best [3]int
	for _, ref := range n.Refs.Nodes {
		if k, ok := semverKey(ref.Name); ok {
			if tagName == "" || semverGreater(k, best) {
				best, tagName, tagSHA = k, ref.Name, ref.Target.commitSHA()
			}
		}
	}
	r.LatestTag = tagName
	r.TagSHA = tagSHA

	// Undeployed: commits on default branch since the latest version tag.
	switch {
	case tagName == "":
		r.Untagged = true
		r.Undeployed = 0
	case r.HeadSHA != "" && tagSHA != "" && r.HeadSHA == tagSHA:
		r.Undeployed = 0
	default:
		r.Undeployed = -1 // >=1, exact count resolved lazily via compare
	}

	r.PRs = classifyPRs(n.PullRequests.TotalCount, n.PullRequests.Nodes)
	r.CIName = r.CI.String()
	r.Health = model.ComputeHealth(r)
	r.HealthName = r.Health.String()
	return r
}

func classifyPRs(total int, nodes []gqlPR) model.PRStats {
	s := model.PRStats{Total: total}
	for _, pr := range nodes {
		isBot := pr.Author.TypeName == "Bot" || botLogins[strings.ToLower(pr.Author.Login)]
		if isBot {
			s.Bot++
		} else {
			s.Human++
		}
		if pr.IsDraft {
			s.Draft++
		}
		if pr.Mergeable == "MERGEABLE" {
			s.Mergeable++
		}
		if len(pr.Commits.Nodes) > 0 {
			rollup := pr.Commits.Nodes[0].Commit.StatusCheckRollup
			if rollup != nil && rollup.State == "SUCCESS" {
				s.CIGreen++
			}
		}
	}
	return s
}
