package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

// Global flags (bound in init).
var (
	flagJSON          bool
	flagNoColor       bool
	flagWatch         bool
	flagOnlyAttention bool
	flagNoCache       bool
	flagOrgs          []string
	flagRefresh       int // seconds
)

var rootCmd = &cobra.Command{
	Use:   "repos-hud",
	Short: "Heads-up display of repo health across your GitHub orgs",
	Long: "gh-repos-hud shows every repo in the organizations you belong to (plus your\n" +
		"personal repos), grouped by org, with CI status, current SHA/tag, undeployed\n" +
		"changes, Dependabot/code-scanning/secret-scanning alerts, and open PRs.\n\n" +
		"Auth is sourced from gh — no token is ever embedded. Default launches a TUI;\n" +
		"use `gh repos-hud serve` for a local web dashboard or --json for raw output.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRoot(cmd.Context())
	},
}

// Execute runs the root command with the given context.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	f := rootCmd.PersistentFlags()
	f.BoolVar(&flagJSON, "json", false, "output machine-readable JSON instead of the TUI")
	f.BoolVar(&flagNoColor, "no-color", false, "disable colored output")
	f.BoolVar(&flagWatch, "watch", false, "auto-refresh on an interval")
	f.BoolVar(&flagOnlyAttention, "only-attention", false, "show only repos needing attention (non-green)")
	f.BoolVar(&flagNoCache, "no-cache", false, "bypass the short-TTL cache and re-fetch")
	f.StringSliceVar(&flagOrgs, "org", nil, "limit to these orgs (repeatable); default: all you belong to")
	f.IntVar(&flagRefresh, "refresh", 30, "refresh interval in seconds (watch/serve)")
}
