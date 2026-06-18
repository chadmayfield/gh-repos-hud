package ghclient

import (
	"context"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

type rateLimitResp struct {
	Resources struct {
		Core struct {
			Remaining int   `json:"remaining"`
			Limit     int   `json:"limit"`
			Reset     int64 `json:"reset"`
		} `json:"core"`
		GraphQL struct {
			Remaining int   `json:"remaining"`
			Limit     int   `json:"limit"`
			Reset     int64 `json:"reset"`
		} `json:"graphql"`
	} `json:"resources"`
}

// rateLimit reads current REST + GraphQL headroom. The /rate_limit endpoint
// does not itself count against any quota.
func (c *Client) rateLimit(ctx context.Context) model.RateLimit {
	r, err := restGet[rateLimitResp](ctx, c.rest, "rate_limit")
	if err != nil {
		return model.RateLimit{}
	}
	reset := time.Time{}
	if r.Resources.Core.Reset > 0 {
		reset = time.Unix(r.Resources.Core.Reset, 0)
	}
	return model.RateLimit{
		RESTRemaining:    r.Resources.Core.Remaining,
		RESTLimit:        r.Resources.Core.Limit,
		GraphQLRemaining: r.Resources.GraphQL.Remaining,
		GraphQLLimit:     r.Resources.GraphQL.Limit,
		ResetAt:          reset,
	}
}
