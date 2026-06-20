package ghclient

import "context"

// ownerRef identifies a repo owner to scan.
type ownerRef struct {
	Name   string
	IsUser bool
}

// DiscoverOwners returns the personal account plus every org the user belongs
// to. If include is non-empty, it filters the *orgs* to those names only — the
// personal account is governed by includePersonal, not the org whitelist, so
// `--org foo` still shows your personal repos. Names in exclude are always
// dropped (personal included); includePersonal=false drops the personal account.
func (c *Client) DiscoverOwners(ctx context.Context, include, exclude []string, includePersonal bool) ([]ownerRef, error) {
	me, err := restGet[struct {
		Login string `json:"login"`
	}](ctx, c.rest, "user")
	if err != nil {
		return nil, err
	}

	orgs, _, err := restListAll[struct {
		Login string `json:"login"`
	}](ctx, c.rest, "user/orgs?per_page=100")
	if err != nil {
		return nil, err
	}

	want := map[string]bool{}
	for _, n := range include {
		want[n] = true
	}
	skip := map[string]bool{}
	for _, n := range exclude {
		skip[n] = true
	}
	var owners []ownerRef
	// Personal account: gated by includePersonal + exclude, NOT the org
	// whitelist — so naming orgs with --org doesn't silently hide your repos.
	if includePersonal && me.Login != "" && !skip[me.Login] {
		owners = append(owners, ownerRef{Name: me.Login, IsUser: true})
	}
	// Orgs: the include whitelist (if any) applies here.
	for _, o := range orgs {
		if (len(want) == 0 || want[o.Login]) && !skip[o.Login] {
			owners = append(owners, ownerRef{Name: o.Login, IsUser: false})
		}
	}
	return owners, nil
}
