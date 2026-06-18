package tui

import (
	"fmt"
	"strings"

	"github.com/chadmayfield/gh-repos-hud/internal/model"
)

// View renders the whole screen.
func (m Model) View() string {
	if m.state == nil {
		if m.err != nil {
			return fmt.Sprintf("\n  error: %v\n\n  press q to quit\n", m.err)
		}
		return "\n  loading repo health...\n"
	}
	if m.detail {
		return m.viewDetail()
	}

	var b strings.Builder
	title := "gh-repos-hud"
	if m.loading {
		title += "  (refreshing...)"
	}
	if m.filtering || m.filter != "" {
		title += "   /" + m.filter
	}
	if m.onlyAttention {
		title += "   [attention-only]"
	}
	b.WriteString(styleHeader.Render(title) + "\n")
	b.WriteString(styleDim.Render("  "+headerRow()) + "\n")

	for i, r := range m.rows {
		gutter := "  "
		if i == m.cursor {
			gutter = styleKey.Render("> ")
		}
		if r.kind == rowOrg {
			b.WriteString(gutter + m.orgLine(r.ownerIdx) + "\n")
		} else {
			repo := m.state.Owners[r.ownerIdx].Repos[r.repoIdx]
			b.WriteString(gutter + repoLine(repo) + "\n")
		}
	}

	b.WriteString("\n" + m.footer())
	return b.String()
}

func headerRow() string {
	return fmt.Sprintf("%s %s %s %s %s %s %s %s %s %s",
		pad("", 4), pad("REPO", 26), pad("CI", 4), pad("SHA", 8), pad("TAG", 11),
		pad("DEP C/H/M/L", 11), pad("CODE", 4), pad("SEC", 4), pad("UNDEP", 8), pad("PR b/h", 7))
}

func (m Model) orgLine(oi int) string {
	o := m.state.Owners[oi]
	marker := "v"
	if !m.isExpanded(o.Name) {
		marker = ">"
	}
	name := o.Name
	if o.IsUser {
		name += " (personal)"
	}
	var g, y, rd int
	for _, r := range o.Repos {
		switch r.Health {
		case model.HealthGreen:
			g++
		case model.HealthYellow:
			y++
		case model.HealthRed:
			rd++
		}
	}
	tally := fmt.Sprintf("%d repos  R:%d Y:%d G:%d", len(o.Repos), rd, y, g)
	extra := ""
	if o.Billing.Known && (o.Billing.SecretProtectionCommitters > 0 || o.Billing.CodeSecurityCommitters > 0) {
		extra = fmt.Sprintf("  GHAS secret=%d code=%d", o.Billing.SecretProtectionCommitters, o.Billing.CodeSecurityCommitters)
	}
	return styleOrg.Render(marker+" "+name) + "   " + styleDim.Render(tally+extra)
}

func repoLine(r model.Repo) string {
	dep := fmt.Sprintf("%d/%d/%d/%d", r.Dependabot.Critical, r.Dependabot.High, r.Dependabot.Moderate, r.Dependabot.Low)
	pr := fmt.Sprintf("%d/%d", r.PRs.Bot, r.PRs.Human)
	return fmt.Sprintf("%s %s %s %s %s %s %s %s %s %s",
		glyph(r.Health), pad(r.Name, 26), ciStyled(r.CI), pad(r.ShortSHA, 8),
		pad(dashIfEmpty(r.LatestTag), 11), pad(dep, 11),
		pad(scanCell(r.CodeScanEnabled, r.CodeScanning), 4),
		pad(scanCell(r.SecretScanEnabled, r.SecretScanning), 4),
		pad(r.UndeployedLabel(), 8), pad(pr, 7))
}

func (m Model) viewDetail() string {
	r := m.selectedRepo()
	if r == nil {
		m2 := m
		m2.detail = false
		return m2.View()
	}
	var b strings.Builder
	b.WriteString(styleHeader.Render(r.Name) + "  " + styleDim.Render(r.URL) + "\n\n")
	fmt.Fprintf(&b, "  health        %s %s\n", glyph(r.Health), r.Health)
	fmt.Fprintf(&b, "  branch        %s @ %s   CI: %s\n", dashIfEmpty(r.DefaultBranch), r.ShortSHA, r.CI)
	fmt.Fprintf(&b, "  tag/release   %s / %s\n", dashIfEmpty(r.LatestTag), dashIfEmpty(r.LatestRelease))
	fmt.Fprintf(&b, "  undeployed    %s\n", r.UndeployedLabel())
	fmt.Fprintf(&b, "  dependabot    crit=%d high=%d mod=%d low=%d  (enabled=%v)\n",
		r.Dependabot.Critical, r.Dependabot.High, r.Dependabot.Moderate, r.Dependabot.Low, r.DependabotEnabled)
	fmt.Fprintf(&b, "  code scan     %s\n", scanCell(r.CodeScanEnabled, r.CodeScanning))
	fmt.Fprintf(&b, "  secret scan   %s\n", scanCell(r.SecretScanEnabled, r.SecretScanning))
	fmt.Fprintf(&b, "  open PRs      total=%d  bot=%d human=%d  mergeable=%d ci-green=%d draft=%d\n",
		r.PRs.Total, r.PRs.Bot, r.PRs.Human, r.PRs.Mergeable, r.PRs.CIGreen, r.PRs.Draft)
	b.WriteString("\n" + styleFooter.Render("  enter/esc back   o open in browser   q quit"))
	return b.String()
}

func (m Model) footer() string {
	rl := m.state.RateLimit
	var parts []string
	parts = append(parts, fmt.Sprintf("REST %d/%d  GraphQL %d/%d", rl.RESTRemaining, rl.RESTLimit, rl.GraphQLRemaining, rl.GraphQLLimit))
	if n := len(m.state.Warnings); n > 0 {
		parts = append(parts, styleWarn.Render(fmt.Sprintf("%d warning(s)", n)))
	}
	keys := "j/k move  enter drill  space fold  / filter  a attention  o open  r refresh  q quit"
	return styleFooter.Render("  "+strings.Join(parts, "   ")) + "\n" + styleFooter.Render("  "+keys)
}

// pad truncates or right-pads s to exactly n visible columns.
func pad(s string, n int) string {
	if len(s) > n {
		if n <= 1 {
			return s[:n]
		}
		return s[:n-1] + "~"
	}
	return s + strings.Repeat(" ", n-len(s))
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func scanCell(enabled bool, n int) string {
	if !enabled {
		return "?"
	}
	return fmt.Sprintf("%d", n)
}
