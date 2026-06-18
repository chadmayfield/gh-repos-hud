package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	versionStr = "dev"
	commitStr  = "none"
	dateStr    = "unknown"
)

// SetVersionInfo is called from main with ldflags-stamped values.
func SetVersionInfo(version, commit, date string) {
	versionStr, commitStr, dateStr = version, commit, date
	rootCmd.Version = version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, commit, and build date",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("gh-repos-hud %s (commit %s, built %s)\n", versionStr, commitStr, dateStr)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
