package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

type stateLoadedMsg struct{ state *model.State }
type errMsg struct{ err error }
type tickMsg struct{}
type detailLoadedMsg struct{ detail model.RepoDetail }

// detailCmd lazily fetches per-repo detail (exact ahead_by, alerts, PRs).
func detailCmd(client *ghclient.Client, owner, name, tagSHA, headSHA string) tea.Cmd {
	return func() tea.Msg {
		return detailLoadedMsg{client.FetchRepoDetail(context.Background(), owner, name, tagSHA, headSHA)}
	}
}

// loadCmd fetches a fresh snapshot off the UI thread.
func loadCmd(client *ghclient.Client, opts ghclient.Options) tea.Cmd {
	return func() tea.Msg {
		st, err := client.FetchState(context.Background(), opts)
		if err != nil {
			return errMsg{err}
		}
		return stateLoadedMsg{st}
	}
}

// tickCmd schedules the next auto-refresh.
func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{} })
}

// freshOpts returns a copy that bypasses the cache (for explicit refresh / watch
// ticks — the user wants new data, not the cached snapshot).
func freshOpts(o ghclient.Options) ghclient.Options {
	o.NoCache = true
	return o
}
