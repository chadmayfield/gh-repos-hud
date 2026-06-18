package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
	"github.com/chadmayfield/gh-repos-hud/internal/tui"
)

func runRoot(ctx context.Context, opts ghclient.Options, interval time.Duration) error {
	client, err := ghclient.New()
	if err != nil {
		return err
	}

	// Interactive TUI by default; --json / --plain (or a non-TTY stdout) take
	// the non-interactive paths and fetch once up front.
	if !flagJSON && !flagPlain && isTTY() {
		m := tui.New(client, opts, flagWatch, interval)
		m.SetOnlyAttention(flagOnlyAttention)
		return tui.Run(m)
	}

	state, err := client.FetchState(ctx, opts)
	if err != nil {
		return err
	}
	if flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(state)
	}
	return renderText(os.Stdout, state)
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

func renderText(w *os.File, st *model.State) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, o := range st.Owners {
		label := o.Name
		if o.IsUser {
			label += " (personal)"
		}
		if o.Billing.Known && (o.Billing.SecretProtectionCommitters > 0 || o.Billing.CodeSecurityCommitters > 0) {
			label += fmt.Sprintf("  GHAS: secret=%d code=%d", o.Billing.SecretProtectionCommitters, o.Billing.CodeSecurityCommitters)
		}
		fmt.Fprintf(w, "\n%s\n", label)
		fmt.Fprintln(tw, "  HEALTH\tREPO\tCI\tSHA\tTAG\tDEP(C/H/M/L)\tCODE\tSECRET\tUNDEP\tPR(bot/hum)")
		for _, r := range o.Repos {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\t%d/%d/%d/%d\t%s\t%s\t%s\t%d/%d\n",
				r.Health.Glyph(), r.Name, r.CI.Short(), r.ShortSHA, tagOrDash(r.LatestTag),
				r.Dependabot.Critical, r.Dependabot.High, r.Dependabot.Moderate, r.Dependabot.Low,
				scanCell(r.CodeScanEnabled, r.CodeScanning), scanCell(r.SecretScanEnabled, r.SecretScanning),
				r.UndeployedLabel(), r.PRs.Bot, r.PRs.Human)
		}
		tw.Flush()
	}
	if len(st.Warnings) > 0 {
		fmt.Fprintf(w, "\n%d warning(s):\n", len(st.Warnings))
		for _, wn := range st.Warnings {
			fmt.Fprintf(w, "  %s/%s: %s\n", wn.Owner, wn.Feature, wn.Reason)
		}
	}
	cache := ""
	if st.FromCache {
		cache = fmt.Sprintf("  [cached %s ago; --no-cache to refresh]", time.Since(st.FetchedAt).Round(time.Second))
	}
	cost := ""
	if st.RateLimit.GraphQLCost > 0 {
		cost = fmt.Sprintf(" (cost %d pts)", st.RateLimit.GraphQLCost)
	}
	fmt.Fprintf(w, "\nrate: REST %d/%d  GraphQL %d/%d%s%s\n",
		st.RateLimit.RESTRemaining, st.RateLimit.RESTLimit,
		st.RateLimit.GraphQLRemaining, st.RateLimit.GraphQLLimit, cost, cache)
	return nil
}

func tagOrDash(t string) string {
	if t == "" {
		return "-"
	}
	return t
}

func scanCell(enabled bool, n int) string {
	if !enabled {
		return "?"
	}
	return fmt.Sprintf("%d", n)
}
