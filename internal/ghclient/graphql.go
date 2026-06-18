package ghclient

import (
	"context"
	"fmt"
	"time"
)

// --- typed GraphQL response structs (decoded via encoding/json, which matches
// field names case-insensitively; explicit tags only where names differ). ---

type gqlRateLimit struct {
	Remaining int       `json:"remaining"`
	Limit     int       `json:"limit"`
	ResetAt   time.Time `json:"resetAt"`
}

type gqlCommit struct {
	Oid               string `json:"oid"`
	AbbreviatedOid    string `json:"abbreviatedOid"`
	StatusCheckRollup *struct {
		State string `json:"state"`
	} `json:"statusCheckRollup"`
}

type gqlTagTarget struct {
	TypeName string `json:"__typename"`
	Oid      string `json:"oid"` // set when the ref points straight at a Commit
	Target   *struct {
		Oid string `json:"oid"`
	} `json:"target"` // set when the ref points at an annotated Tag
}

// commitSHA resolves the underlying commit a tag ref points to.
func (t gqlTagTarget) commitSHA() string {
	if t.Target != nil && t.Target.Oid != "" {
		return t.Target.Oid
	}
	return t.Oid
}

type gqlPR struct {
	IsDraft   bool   `json:"isDraft"`
	Mergeable string `json:"mergeable"`
	Author    struct {
		Login    string `json:"login"`
		TypeName string `json:"__typename"`
	} `json:"author"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

type gqlRepoNode struct {
	Name                          string `json:"name"`
	URL                           string `json:"url"`
	IsArchived                    bool   `json:"isArchived"`
	IsPrivate                     bool   `json:"isPrivate"`
	IsFork                        bool   `json:"isFork"`
	HasVulnerabilityAlertsEnabled bool   `json:"hasVulnerabilityAlertsEnabled"`
	DefaultBranchRef              *struct {
		Name   string    `json:"name"`
		Target gqlCommit `json:"target"`
	} `json:"defaultBranchRef"`
	LatestRelease *struct {
		TagName string `json:"tagName"`
	} `json:"latestRelease"`
	Refs struct {
		Nodes []struct {
			Name   string       `json:"name"`
			Target gqlTagTarget `json:"target"`
		} `json:"nodes"`
	} `json:"refs"`
	VulnerabilityAlerts struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			SecurityVulnerability struct {
				Severity string `json:"severity"`
			} `json:"securityVulnerability"`
		} `json:"nodes"`
	} `json:"vulnerabilityAlerts"`
	PullRequests struct {
		TotalCount int     `json:"totalCount"`
		Nodes      []gqlPR `json:"nodes"`
	} `json:"pullRequests"`
}

type gqlRepoConnection struct {
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
	Nodes []gqlRepoNode `json:"nodes"`
}

type gqlReposResponse struct {
	RateLimit    gqlRateLimit `json:"rateLimit"`
	Organization *struct {
		Repositories gqlRepoConnection `json:"repositories"`
	} `json:"organization"`
	User *struct {
		Repositories gqlRepoConnection `json:"repositories"`
	} `json:"user"`
}

// fetchRepoNodes runs the org or user repo query, following pagination, and
// returns every repo node plus the latest rateLimit reading.
func (c *Client) fetchRepoNodes(ctx context.Context, login string, isUser bool) ([]gqlRepoNode, gqlRateLimit, error) {
	query := orgReposQuery
	if isUser {
		query = userReposQuery
	}

	var all []gqlRepoNode
	var rl gqlRateLimit
	var cursor *string

	for {
		vars := map[string]interface{}{"login": login, "cursor": cursor}
		var resp gqlReposResponse
		if err := c.gql.DoWithContext(ctx, query, vars, &resp); err != nil {
			return nil, rl, fmt.Errorf("graphql repos for %s: %w", login, err)
		}
		rl = resp.RateLimit

		conn := resp.Organization
		var c2 *gqlRepoConnection
		if conn != nil {
			c2 = &conn.Repositories
		} else if resp.User != nil {
			c2 = &resp.User.Repositories
		}
		if c2 == nil {
			break
		}
		all = append(all, c2.Nodes...)
		if !c2.PageInfo.HasNextPage || c2.PageInfo.EndCursor == "" {
			break
		}
		next := c2.PageInfo.EndCursor
		cursor = &next
	}
	return all, rl, nil
}
