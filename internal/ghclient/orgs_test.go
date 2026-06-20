package ghclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func ownersServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"login":"me"}`)
	})
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `[{"login":"o1"},{"login":"o2"}]`)
	})
	return httptest.NewServer(mux)
}

func ownerNames(owners []ownerRef) map[string]bool {
	m := map[string]bool{}
	for _, o := range owners {
		m[o.Name] = true
	}
	return m
}

func TestDiscoverOwners(t *testing.T) {
	srv := ownersServer()
	defer srv.Close()
	c := testClient(t, srv)
	ctx := context.Background()

	t.Run("default", func(t *testing.T) {
		owners, err := c.DiscoverOwners(ctx, nil, nil, true)
		if err != nil {
			t.Fatal(err)
		}
		got := ownerNames(owners)
		if len(owners) != 3 || !got["me"] || !got["o1"] || !got["o2"] {
			t.Errorf("default owners = %+v, want me+o1+o2", owners)
		}
		for _, o := range owners {
			if o.Name == "me" && !o.IsUser {
				t.Error("me should be IsUser")
			}
			if o.Name == "o1" && o.IsUser {
				t.Error("o1 should not be IsUser")
			}
		}
	})

	t.Run("include filters orgs but keeps personal", func(t *testing.T) {
		// --org o1 narrows the orgs to o1, but the personal account stays
		// (the whitelist applies to orgs only, not to personal).
		owners, err := c.DiscoverOwners(ctx, []string{"o1"}, nil, true)
		if err != nil {
			t.Fatal(err)
		}
		got := ownerNames(owners)
		if len(owners) != 2 || !got["me"] || !got["o1"] || got["o2"] {
			t.Errorf("include owners = %+v, want me+o1 (personal kept, o2 dropped)", owners)
		}
	})

	t.Run("include with no personal = only named orgs", func(t *testing.T) {
		owners, err := c.DiscoverOwners(ctx, []string{"o1"}, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		got := ownerNames(owners)
		if len(owners) != 1 || !got["o1"] {
			t.Errorf("include+no-personal owners = %+v, want only o1", owners)
		}
	})

	t.Run("exclude drops named", func(t *testing.T) {
		owners, err := c.DiscoverOwners(ctx, nil, []string{"o2"}, true)
		if err != nil {
			t.Fatal(err)
		}
		got := ownerNames(owners)
		if got["o2"] {
			t.Errorf("exclude owners still has o2: %+v", owners)
		}
		if !got["me"] || !got["o1"] {
			t.Errorf("exclude dropped too much: %+v", owners)
		}
	})

	t.Run("no personal drops me", func(t *testing.T) {
		owners, err := c.DiscoverOwners(ctx, nil, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		got := ownerNames(owners)
		if got["me"] {
			t.Errorf("includePersonal=false still has me: %+v", owners)
		}
		if len(owners) != 2 {
			t.Errorf("owners = %d, want 2 orgs", len(owners))
		}
	})
}
