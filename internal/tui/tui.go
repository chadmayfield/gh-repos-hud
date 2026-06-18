// Package tui renders the interactive heads-up display with bubbletea.
package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

type rowKind int

const (
	rowOrg rowKind = iota
	rowRepo
)

// row is one navigable line: an org header or (when expanded) a repo.
type row struct {
	kind     rowKind
	ownerIdx int
	repoIdx  int
}

// Model is the bubbletea state.
type Model struct {
	client *ghclient.Client
	opts   ghclient.Options

	state    *model.State
	rows     []row
	cursor   int
	expanded map[string]bool

	filter        string
	filtering     bool
	onlyAttention bool

	watch    bool
	interval time.Duration
	detail   bool
	loading  bool
	err      error

	width, height int
}

// New builds the initial model.
func New(client *ghclient.Client, opts ghclient.Options, watch bool, interval time.Duration) Model {
	return Model{
		client:   client,
		opts:     opts,
		expanded: map[string]bool{},
		watch:    watch,
		interval: interval,
		loading:  true,
	}
}

// SetOnlyAttention pre-filters to non-green repos.
func (m *Model) SetOnlyAttention(v bool) { m.onlyAttention = v }

// Init kicks off the first load (and the refresh tick when watching).
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{loadCmd(m.client, m.opts)}
	if m.watch {
		cmds = append(cmds, tickCmd(m.interval))
	}
	return tea.Batch(cmds...)
}

// Run starts the program.
func Run(m Model) error {
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// rebuildRows recomputes the visible row list from state + expand/filter state.
func (m *Model) rebuildRows() {
	m.rows = m.rows[:0]
	if m.state == nil {
		return
	}
	flt := strings.ToLower(strings.TrimSpace(m.filter))
	for oi, o := range m.state.Owners {
		m.rows = append(m.rows, row{kind: rowOrg, ownerIdx: oi})
		if !m.isExpanded(o.Name) {
			continue
		}
		for ri, r := range o.Repos {
			if m.onlyAttention && r.Health == model.HealthGreen {
				continue
			}
			if flt != "" && !strings.Contains(strings.ToLower(r.Name), flt) {
				continue
			}
			m.rows = append(m.rows, row{kind: rowRepo, ownerIdx: oi, repoIdx: ri})
		}
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// isExpanded reports whether an org is expanded (default: expanded).
func (m *Model) isExpanded(org string) bool {
	v, ok := m.expanded[org]
	return !ok || v
}

func (m *Model) selectedRepo() *model.Repo {
	if len(m.rows) == 0 || m.cursor >= len(m.rows) {
		return nil
	}
	r := m.rows[m.cursor]
	if r.kind != rowRepo {
		return nil
	}
	return &m.state.Owners[r.ownerIdx].Repos[r.repoIdx]
}
