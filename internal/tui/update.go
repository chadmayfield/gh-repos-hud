package tui

import (
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and key input.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampScroll()
		return m, nil

	case stateLoadedMsg:
		m.state = msg.state
		m.err = nil
		m.loading = false
		m.rebuildRows()
		m.clampScroll()
		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case detailLoadedMsg:
		d := msg.detail
		m.detailData = &d
		m.detailLoading = false
		return m, nil

	case tickMsg:
		var cmds []tea.Cmd
		// Back off auto-refresh when GraphQL headroom is low (manual 'r' still works).
		if m.watch && !m.loading && m.graphqlOK() {
			m.loading = true
			cmds = append(cmds, loadCmd(m.client, freshOpts(m.opts)))
		}
		if m.watch {
			cmds = append(cmds, tickCmd(m.interval))
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Filter input mode captures most keys.
	if m.filtering {
		switch msg.Type {
		case tea.KeyEnter, tea.KeyEsc:
			m.filtering = false
		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
			}
			m.rebuildRows()
		case tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.rebuildRows()
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		if m.detail {
			m.detail = false
		}
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.clampScroll()
		return m, nil
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
		m.clampScroll()
		return m, nil
	case "pgup", "ctrl+u":
		m.cursor -= m.bodyCapacity()
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.clampScroll()
		return m, nil
	case "pgdown", "ctrl+d", " ":
		// space pages down (org-fold moved to tab/enter to free it for paging)
		m.cursor += m.bodyCapacity()
		if m.cursor > len(m.rows)-1 {
			m.cursor = len(m.rows) - 1
		}
		m.clampScroll()
		return m, nil
	case "g", "home":
		m.cursor = 0
		m.clampScroll()
		return m, nil
	case "G", "end":
		m.cursor = len(m.rows) - 1
		m.clampScroll()
		return m, nil
	case "enter":
		if m.detail {
			m.detail = false
			return m, nil
		}
		if len(m.rows) == 0 {
			return m, nil
		}
		r := m.rows[m.cursor]
		if r.kind == rowOrg {
			name := m.state.Owners[r.ownerIdx].Name
			m.expanded[name] = !m.isExpanded(name)
			m.rebuildRows()
			m.clampScroll()
			return m, nil
		}
		// Drill into a repo: open the pane and lazily fetch its detail.
		m.detail = true
		m.detailData = nil
		// Demo mode has no live client, so skip the detail fetch and just show
		// the summary the snapshot already carries.
		if m.opts.Demo {
			m.detailLoading = false
			return m, nil
		}
		m.detailLoading = true
		owner := m.state.Owners[r.ownerIdx].Name
		repo := m.state.Owners[r.ownerIdx].Repos[r.repoIdx]
		return m, detailCmd(m.client, owner, repo.Name, repo.TagSHA, repo.HeadSHA)
	case "tab":
		if len(m.rows) > 0 {
			name := m.state.Owners[m.rows[m.cursor].ownerIdx].Name
			m.expanded[name] = !m.isExpanded(name)
			m.rebuildRows()
			m.clampScroll()
		}
		return m, nil
	case "a":
		m.onlyAttention = !m.onlyAttention
		m.rebuildRows()
		return m, nil
	case "s":
		m.sortBy = (m.sortBy + 1) % 4
		m.rebuildRows()
		m.clampScroll()
		return m, nil
	case "r":
		if !m.loading {
			m.loading = true
			return m, loadCmd(m.client, freshOpts(m.opts))
		}
		return m, nil
	case "/":
		m.filtering = true
		return m, nil
	case "o":
		if repo := m.selectedRepo(); repo != nil && repo.URL != "" {
			return m, openURLCmd(repo.URL)
		}
		return m, nil
	}
	return m, nil
}

// openURLCmd opens a URL in the default browser without blocking the UI.
func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		_ = cmd.Start()
		return nil
	}
}
