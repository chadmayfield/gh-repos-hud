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
	var billing model.Billing

	// Billing is an org-only, admin-only signal (it doesn't exist for the
	// personal account). Per-repo scan status, below, is queried for both.
	if !o.IsUser {
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
		owner.Repos = append(owner.Repos, buildRepo(n))
	}

	// Per-repo code/secret scan probes, concurrently. This is the accurate
	// signal — the same query for personal and org repos — that distinguishes
	// "enabled & clean" (0) from "not enabled" (off) from "couldn't tell" (?),
	// which the old org-aggregate could not. Two REST calls per repo; failures
	// degrade to off/? rather than failing the owner. SetLimit caps the burst
	// so a 60-repo owner doesn't trip GitHub's secondary (concurrency) limit.
	sg := new(errgroup.Group)
	sg.SetLimit(8)
	for i := range owner.Repos {
		r := &owner.Repos[i]
		sg.Go(func() error {
			r.CodeScan, r.CodeScanning = c.repoScanState(ctx, o.Name, r.Name, "code-scanning")
			return nil
		})
		sg.Go(func() error {
			r.SecretScan, r.SecretScanning = c.repoScanState(ctx, o.Name, r.Name, "secret-scanning")
			return nil
		})
	}
	_ = sg.Wait()

	// Roll up health now that scan counts exist (ComputeHealth factors open
	// scan alerts into yellow), and stamp the JSON status names.
	for i := range owner.Repos {
		r := &owner.Repos[i]
		r.CodeScanName = r.CodeScan.String()
		r.SecretScanName = r.SecretScan.String()
		r.Health = model.ComputeHealth(*r)
		r.HealthName = r.Health.String()
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
