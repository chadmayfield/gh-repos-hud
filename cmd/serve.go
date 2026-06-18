package cmd

import (
	"github.com/spf13/cobra"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/web"
)

var flagPort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the local web dashboard at http://127.0.0.1:PORT",
	Long: "Starts a loopback-only web server that renders the same HUD as the TUI\n" +
		"and auto-refreshes. Bound to 127.0.0.1 only; never exposed externally.",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ghclient.New()
		if err != nil {
			return err
		}
		opts, interval, port := resolveOptions(cmd)
		// Default: on-demand fetch per request (cached); --watch enables the
		// background poller + page auto-refresh.
		return web.Serve(cmd.Context(), client, opts, port, interval, flagWatch)
	},
}

func init() {
	serveCmd.Flags().IntVar(&flagPort, "port", 8787, "port to listen on (127.0.0.1 only)")
	rootCmd.AddCommand(serveCmd)
}
