// Command gh-repos-hud is a gh CLI extension: a heads-up display of repo
// health across the GitHub organizations the logged-in gh user belongs to.
//
// It never embeds a token — auth is sourced from gh via the go-gh library.
// Run as `gh repos-hud` (TUI), `gh repos-hud serve` (web), or with --json.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/chadmayfield/gh-repos-hud/cmd"
)

// Set via ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)

	// Structured logging to stderr; warn by default, debug when HUD_DEBUG is set.
	logLevel := slog.LevelWarn
	if os.Getenv("HUD_DEBUG") != "" {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := cmd.Execute(ctx); err != nil {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "Interrupted")
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
