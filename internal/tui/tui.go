// Package tui renders the interactive heads-up display with bubbletea.
package tui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/chadmayfield/gh-repos-hud/internal/ghclient"
	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// sortKey controls repo ordering within each org.
type sortKey int

const (
	sortHealth sortKey = iota // attention-first (default)
	sortName
	sortUndeployed
	sortAlerts
)

func (s sortKey) String() string {
	switch s {
	case sortName:
		return "name"
	case sortUndeployed:
		return "undeployed"
	case sortAlerts:
		return "alerts"
	default:
		return "health"
	}
}

// sortedRepoIndices returns repo indices ordered by the given key (name as a
// stable tiebreaker). Indices, not repos, so row lookups stay valid.
func sortedRepoIndices(repos []model.Repo, key sortKey) []int {
	idx := make([]int, len(repos))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ra, rb := repos[idx[a]], repos[idx[b]]
		switch key {
		case sortName:
			return ra.Name < rb.Name
		case sortUndeployed:
			if undeployRank(ra) != undeployRank(rb) {
				return undeployRank(ra) > undeployRank(rb)
			}
		case sortAlerts:
			if ra.Dependabot.Total() != rb.Dependabot.Total() {
				return ra.Dependabot.Total() > rb.Dependabot.Total()
			}
		default: // health: red>yellow>green>gray
			if ra.Health != rb.Health {
				return ra.Health > rb.Health
			}
		}
		return ra.Name < rb.Name
	})
	return idx
}

// undeployRank sorts ">=1/exact" above 0 above untagged.
func undeployRank(r model.Repo) int {
	switch {
	case r.Untagged:
		return -1
	case r.Undeployed != 0:
		return 1000000 - r.Undeployed // -1 (unknown >=1) ranks highest
	default:
		return 0
	}
}

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
	sortBy        sortKey

	watch         bool
	interval      time.Duration
	detail        bool
	detailData    *model.RepoDetail
	detailLoading bool
	loading       bool
	err           error

	scroll        int // index of the first visible row
	width, height int
}

// bodyCapacity is how many list rows fit between the sticky header and footer.
// Before the first WindowSizeMsg (height unset) it returns an effectively
// unlimited window so the initial paint isn't clipped.
func (m Model) bodyCapacity() int {
	if m.height <= 0 {
		return 100000
	}
	// title(1) + sticky header(1) + scroll indicator(1) + blank(1) + footer(4)
	c := m.height - 8
	if c < 1 {
		return 1
	}
	return c
}

// clampScroll keeps the cursor within the visible window and the window within
// bounds.
func (m *Model) clampScroll() {
	cap := m.bodyCapacity()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+cap {
		m.scroll = m.cursor - cap + 1
	}
	maxScroll := len(m.rows) - cap
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
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
		for _, ri := range sortedRepoIndices(o.Repos, m.sortBy) {
			r := o.Repos[ri]
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
	m.clampScroll()
}

// lowGraphQLThreshold is the headroom below which auto-refresh backs off.
const lowGraphQLThreshold = 100

// graphqlOK reports whether there's enough GraphQL headroom to auto-refresh.
func (m Model) graphqlOK() bool {
	return m.state == nil || m.state.RateLimit.GraphQLRemaining > lowGraphQLThreshold
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
