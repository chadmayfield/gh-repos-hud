package ghclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// Client wraps the gh-sourced REST and GraphQL clients. Auth comes from gh —
// no token is ever held or logged here.
type Client struct {
	rest *api.RESTClient
	gql  *api.GraphQLClient
}

// New builds a Client using the logged-in gh user's credentials.
func New() (*Client, error) {
	rest, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("gh REST client (run `gh auth login`?): %w", err)
	}
	gql, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("gh GraphQL client (run `gh auth login`?): %w", err)
	}
	return &Client{rest: rest, gql: gql}, nil
}

// errFeatureUnavailable marks a 403/404 — the feature is disabled or the token
// lacks scope. Callers degrade (show "?") instead of failing the whole run.
var errFeatureUnavailable = errors.New("feature unavailable")

// restListAll GETs every page of a JSON-array endpoint, following the Link
// header, and returns the combined slice plus the final response headers
// (used for rate-limit readings).
func restListAll[T any](ctx context.Context, c *api.RESTClient, path string) ([]T, http.Header, error) {
	var out []T
	var hdr http.Header
	next := path
	for next != "" {
		resp, err := c.RequestWithContext(ctx, http.MethodGet, next, nil)
		if err != nil {
			var httpErr *api.HTTPError
			if errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusForbidden || httpErr.StatusCode == http.StatusNotFound) {
				return out, hdr, fmt.Errorf("%s: %w", path, errFeatureUnavailable)
			}
			return out, hdr, err
		}
		hdr = resp.Header
		var page []T
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if len(body) > 0 {
			if err := json.Unmarshal(body, &page); err != nil {
				return out, hdr, fmt.Errorf("decode %s: %w", path, err)
			}
		}
		out = append(out, page...)
		next = nextPageLink(resp.Header.Get("Link"))
	}
	return out, hdr, nil
}

// restGet decodes a single JSON object endpoint, degrading on 403/404.
func restGet[T any](ctx context.Context, c *api.RESTClient, path string) (T, error) {
	var v T
	resp, err := c.RequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusForbidden || httpErr.StatusCode == http.StatusNotFound) {
			return v, fmt.Errorf("%s: %w", path, errFeatureUnavailable)
		}
		return v, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		if err := json.Unmarshal(body, &v); err != nil {
			return v, fmt.Errorf("decode %s: %w", path, err)
		}
	}
	return v, nil
}

// nextPageLink extracts the rel="next" URL from a Link header and returns it as
// a host-relative path+query (so it works for github.com and GHES alike).
func nextPageLink(link string) string {
	if link == "" {
		return ""
	}
	for _, part := range strings.Split(link, ",") {
		segs := strings.Split(strings.TrimSpace(part), ";")
		if len(segs) < 2 {
			continue
		}
		isNext := false
		for _, s := range segs[1:] {
			if strings.Contains(s, `rel="next"`) {
				isNext = true
			}
		}
		if !isNext {
			continue
		}
		raw := strings.TrimSpace(segs[0])
		raw = strings.TrimPrefix(raw, "<")
		raw = strings.TrimSuffix(raw, ">")
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		if u.RawQuery != "" {
			return u.Path + "?" + u.RawQuery
		}
		return u.Path
	}
	return ""
}
