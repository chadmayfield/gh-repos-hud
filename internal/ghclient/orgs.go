package ghclient

import "context"

// ownerRef identifies a repo owner to scan.
type ownerRef struct {
	Name   string
	IsUser bool
}

// DiscoverOwners returns the personal account plus every org the user belongs
// to. If include is non-empty, only those owner names are kept. If
// includePersonal is false, the personal account is dropped.
func (c *Client) DiscoverOwners(ctx context.Context, include []string, includePersonal bool) ([]ownerRef, error) {
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
	keep := func(name string) bool { return len(want) == 0 || want[name] }

	var owners []ownerRef
	if includePersonal && me.Login != "" && keep(me.Login) {
		owners = append(owners, ownerRef{Name: me.Login, IsUser: true})
	}
	for _, o := range orgs {
		if keep(o.Login) {
			owners = append(owners, ownerRef{Name: o.Login, IsUser: false})
		}
	}
	return owners, nil
}
