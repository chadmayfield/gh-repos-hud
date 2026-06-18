package ghclient

import (
	"context"
	"fmt"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

type alertRepoRef struct {
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
}

// scanCounts aggregates open code-scanning or secret-scanning alerts by repo
// name for an org. Returns errFeatureUnavailable (wrapped) when the feature is
// off or the token lacks scope, so callers can degrade.
func (c *Client) scanCounts(ctx context.Context, org, product string) (map[string]int, error) {
	path := fmt.Sprintf("orgs/%s/%s/alerts?state=open&per_page=100", org, product)
	alerts, _, err := restListAll[alertRepoRef](ctx, c.rest, path)
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(alerts))
	for _, a := range alerts {
		if a.Repository.Name != "" {
			counts[a.Repository.Name]++
		}
	}
	return counts, nil
}

func (c *Client) codeScanCounts(ctx context.Context, org string) (map[string]int, error) {
	return c.scanCounts(ctx, org, "code-scanning")
}

func (c *Client) secretScanCounts(ctx context.Context, org string) (map[string]int, error) {
	return c.scanCounts(ctx, org, "secret-scanning")
}

type ghasBillingResp struct {
	TotalAdvancedSecurityCommitters int `json:"total_advanced_security_committers"`
}

// billing reads org-level GHAS spend (the paid-features guard). Degrades to
// Billing{Known:false} when unavailable (no GHAS / not an admin).
func (c *Client) billing(ctx context.Context, org string) (model.Billing, error) {
	var b model.Billing
	sp, err := restGet[ghasBillingResp](ctx, c.rest,
		fmt.Sprintf("orgs/%s/settings/billing/advanced-security?advanced_security_product=secret_protection", org))
	if err != nil {
		return b, err
	}
	cs, err := restGet[ghasBillingResp](ctx, c.rest,
		fmt.Sprintf("orgs/%s/settings/billing/advanced-security?advanced_security_product=code_security", org))
	if err != nil {
		return b, err
	}
	b.Known = true
	b.SecretProtectionCommitters = sp.TotalAdvancedSecurityCommitters
	b.CodeSecurityCommitters = cs.TotalAdvancedSecurityCommitters
	return b, nil
}
