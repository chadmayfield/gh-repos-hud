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

	owners, err := c.DiscoverOwners(ctx, opts.IncludeOrgs, opts.IncludePersonal)
	if err != nil {
		return nil, err
	}

	results := make([]model.Owner, len(owners))
	warns := make([][]model.Warning, len(owners))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(6)
	for i, o := range owners {
		i, o := i, o
		g.Go(func() error {
			owner, w, err := c.fetchOwner(gctx, o, opts)
			if err != nil {
				return err
			}
			results[i] = owner
			warns[i] = w
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	state := &model.State{FetchedAt: time.Now()}
	for _, o := range results {
		if o.Name != "" {
			state.Owners = append(state.Owners, o)
		}
	}
	for _, w := range warns {
		state.Warnings = append(state.Warnings, w...)
	}
	state.RateLimit = c.rateLimit(ctx)
	saveCache(state)
	return state, nil
}

func (c *Client) fetchOwner(ctx context.Context, o ownerRef, opts Options) (model.Owner, []model.Warning, error) {
	nodes, _, err := c.fetchRepoNodes(ctx, o.Name, o.IsUser)
	if err != nil {
		return model.Owner{}, nil, err
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
				return model.Owner{}, nil, err
			}
		} else {
			codeCounts, codeAvail = m, true
		}

		if m, err := c.secretScanCounts(ctx, o.Name); err != nil {
			if errors.Is(err, errFeatureUnavailable) {
				warns = append(warns, model.Warning{Owner: o.Name, Feature: "secret-scanning", Reason: "disabled or no access"})
			} else {
				return model.Owner{}, nil, err
			}
		} else {
			secretCounts, secretAvail = m, true
		}

		if b, err := c.billing(ctx, o.Name); err != nil {
			if errors.Is(err, errFeatureUnavailable) {
				warns = append(warns, model.Warning{Owner: o.Name, Feature: "billing", Reason: "no GHAS or not an admin"})
			} else {
				return model.Owner{}, nil, err
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
	return owner, warns, nil
}
