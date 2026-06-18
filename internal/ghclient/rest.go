package ghclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// scanAlert is a throwaway element type: per-repo scan probes only need the
// page length (the open-alert count), not any field of each alert.
type scanAlert struct{}

// repoScanState probes one repo's scanning product (code-scanning or
// secret-scanning) and reports both whether it is enabled and, if so, how many
// alerts are open. This is the only reliable per-repo signal — the org-level
// aggregate returns alerts, so "0 alerts" there can't distinguish "enabled and
// clean" from "never turned on". Here the HTTP status itself carries that fact:
//
//	200       -> enabled; count = open alerts                 -> ScanOn
//	403 / 404 -> not enabled / no analysis / no token scope   -> ScanOff
//	otherwise -> transient (network / 5xx / secondary limit)  -> ScanUnknown
//
// It never returns an error: a probe failure is data ("off" or "?"), not a
// reason to abort the owner's fetch.
func (c *Client) repoScanState(ctx context.Context, owner, repo, product string) (model.ScanState, int) {
	path := fmt.Sprintf("repos/%s/%s/%s/alerts?state=open&per_page=100", owner, repo, product)
	alerts, _, err := restListAll[scanAlert](ctx, c.rest, path)
	if err != nil {
		if errors.Is(err, errFeatureUnavailable) {
			return model.ScanOff, 0
		}
		return model.ScanUnknown, 0
	}
	return model.ScanOn, len(alerts)
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
