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
