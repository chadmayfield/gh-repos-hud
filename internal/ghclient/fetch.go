package ghclient

import (
	"context"
	"errors"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// Options controls a FetchState run.
type Options struct {
	IncludeOrgs     []string
	ExcludeOrgs     []string
	IncludePersonal bool
	ExcludeArchived bool
	NoCache         bool
	CacheTTL        time.Duration
}

// DefaultOptions returns sensible defaults: all orgs + personal, no archived,
// 5-minute cache.
func DefaultOptions() Options {
	return Options{IncludePersonal: true, ExcludeArchived: true, CacheTTL: 5 * time.Minute}
}

// FetchState builds the full HUD snapshot across every owner concurrently.
// Unless NoCache is set, a fresh-enough on-disk snapshot is reused to avoid
// re-spending the point-metered GraphQL budget.
func (c *Client) FetchState(ctx context.Context, opts Options) (*model.State, error) {
	if !opts.NoCache && opts.CacheTTL > 0 {
		if st, ok := loadCache(opts.CacheTTL); ok {
			return st, nil
		}
	}

	owners, err := c.DiscoverOwners(ctx, opts.IncludeOrgs, opts.ExcludeOrgs, opts.IncludePersonal)
	if err != nil {
		return nil, err
	}

	results := make([]model.Owner, len(owners))
	warns := make([][]model.Warning, len(owners))
	costs := make([]int, len(owners))
	ownerErrs := make([]error, len(owners))

	// Per-owner isolation: one owner failing (e.g. a rate-limit hit) becomes a
	// warning, not a whole-fetch abort, so the owners that loaded still show.
	g := new(errgroup.Group)
	g.SetLimit(6)
	for i, o := range owners {
		i, o := i, o
		g.Go(func() error {
			owner, w, cost, err := c.fetchOwner(ctx, o, opts)
			if err != nil {
				ownerErrs[i] = err
				return nil
			}
			results[i], warns[i], costs[i] = owner, w, cost
			return nil
		})
	}
	_ = g.Wait()

	state := &model.State{FetchedAt: time.Now()}
	for _, o := range results {
		if o.Name != "" {
			state.Owners = append(state.Owners, o)
		}
	}
	for _, w := range warns {
		state.Warnings = append(state.Warnings, w...)
	}
	for i, e := range ownerErrs {
		if e != nil {
			state.Warnings = append(state.Warnings, model.Warning{Owner: owners[i].Name, Feature: "fetch", Reason: e.Error()})
		}
	}
	state.RateLimit = c.rateLimit(ctx)
	for _, cost := range costs {
		state.RateLimit.GraphQLCost += cost
	}
	// Only cache a clean, unfiltered fetch — never a partial (rate-limited /
	// per-owner-failed) or org-filtered run, so the default never serves a
	// truncated snapshot later.
	clean := len(opts.IncludeOrgs) == 0 && len(opts.ExcludeOrgs) == 0
	for _, e := range ownerErrs {
		if e != nil {
			clean = false
		}
	}
	if clean {
		saveCache(state)
	}
	return state, nil
}

func (c *Client) fetchOwner(ctx context.Context, o ownerRef, opts Options) (model.Owner, []model.Warning, int, error) {
	nodes, rl, err := c.fetchRepoNodes(ctx, o.Name, o.IsUser)
	if err != nil {
		return model.Owner{}, nil, 0, err
	}

	var warns []model.Warning
	var codeCounts, secretCounts map[string]int
	codeAvail, secretAvail := false, false
	var billing model.Billing

	// Org-level scanning + billing don't exist for the personal account.
	if !o.IsUser {
		if m, err := c.codeScanCounts(ctx, o.Name); err != nil {
			if errors.Is(err, errFeatureUnavailable) {
				warns = append(warns, model.Warning{Owner: o.Name, Feature: "code-scanning", Reason: "disabled or no access"})
			} else {
				return model.Owner{}, nil, 0, err
			}
		} else {
			codeCounts, codeAvail = m, true
		}

		if m, err := c.secretScanCounts(ctx, o.Name); err != nil {
			if errors.Is(err, errFeatureUnavailable) {
				warns = append(warns, model.Warning{Owner: o.Name, Feature: "secret-scanning", Reason: "disabled or no access"})
			} else {
				return model.Owner{}, nil, 0, err
			}
		} else {
			secretCounts, secretAvail = m, true
		}

		if b, err := c.billing(ctx, o.Name); err != nil {
			if errors.Is(err, errFeatureUnavailable) {
				warns = append(warns, model.Warning{Owner: o.Name, Feature: "billing", Reason: "no GHAS or not an admin"})
			} else {
				return model.Owner{}, nil, 0, err
			}
		} else {
			billing = b
		}
	}

	owner := model.Owner{Name: o.Name, IsUser: o.IsUser, Billing: billing}
	for _, n := range nodes {
		if n.IsFork {
			continue
		}
		if opts.ExcludeArchived && n.IsArchived {
			continue
		}
		owner.Repos = append(owner.Repos, buildRepo(n, codeCounts[n.Name], secretCounts[n.Name], codeAvail, secretAvail))
	}

	// Attention-first: Red > Yellow > Green > Gray, then by name.
	sort.SliceStable(owner.Repos, func(i, j int) bool {
		a, b := owner.Repos[i], owner.Repos[j]
		if a.Health != b.Health {
			return a.Health > b.Health
		}
		return a.Name < b.Name
	})
	return owner, warns, rl.Cost, nil
}
