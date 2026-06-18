package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

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
	title += "   sort:" + m.sortBy.String()
	if m.watch && !m.graphqlOK() {
		title += styleWarn.Render("   [auto-refresh paused: low GraphQL]")
	}
	b.WriteString(styleHeader.Render(title) + "\n")
	// Sticky column header (stays visible while the list scrolls below it).
	b.WriteString("  " + styleDim.Render(headerRow()) + "\n")

	cap := m.bodyCapacity()
	end := m.scroll + cap
	if end > len(m.rows) {
		end = len(m.rows)
	}
	for i := m.scroll; i < end; i++ {
		r := m.rows[i]
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

	b.WriteString(scrollIndicator(m.scroll, end, len(m.rows)) + "\n")
	b.WriteString(m.footer())
	return b.String()
}

func scrollIndicator(start, end, total int) string {
	if total == 0 {
		return styleDim.Render("  (no repos)")
	}
	up, down := "  ", "  "
	if start > 0 {
		up = styleKey.Render(" ^")
	}
	if end < total {
		down = styleKey.Render(" v")
	}
	return styleDim.Render(fmt.Sprintf("  rows %d-%d of %d", start+1, end, total)) + up + down
}

func headerRow() string {
	return fmt.Sprintf("%s %s %s %s %s %s %s %s %s %s %s",
		pad("", 4), pad("", 1), pad("REPO", 26), pad("CI", 4), pad("SHA", 8), pad("TAG", 11),
		pad("DEP C/H/M/L", 11), pad("CODE", 4), pad("SEC", 4), pad("UNDEP", 8), pad("PR b/h", 7))
}

func (m Model) orgLine(oi int) string {
	o := m.state.Owners[oi]
	marker := "[-]" // expanded (can collapse)
	if !m.isExpanded(o.Name) {
		marker = "[+]" // collapsed (can expand)
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
	// Colored glyph tally, matching the legend, instead of opaque R/Y/G letters.
	tally := fmt.Sprintf("%d repos   ", len(o.Repos)) +
		lipgloss.NewStyle().Foreground(colRed).Render(fmt.Sprintf("[!!] %d", rd)) + "  " +
		lipgloss.NewStyle().Foreground(colYellow).Render(fmt.Sprintf("[~] %d", y)) + "  " +
		lipgloss.NewStyle().Foreground(colGreen).Render(fmt.Sprintf("[OK] %d", g))
	if o.Billing.Known && (o.Billing.SecretProtectionCommitters > 0 || o.Billing.CodeSecurityCommitters > 0) {
		tally += styleDim.Render(fmt.Sprintf("   GHAS secret=%d code=%d", o.Billing.SecretProtectionCommitters, o.Billing.CodeSecurityCommitters))
	}
	return styleOrg.Render(marker+" "+name) + "   " + tally
}

func repoLine(r model.Repo) string {
	dep := fmt.Sprintf("%d/%d/%d/%d", r.Dependabot.Critical, r.Dependabot.High, r.Dependabot.Moderate, r.Dependabot.Low)
	pr := fmt.Sprintf("%d/%d", r.PRs.Bot, r.PRs.Human)
	return fmt.Sprintf("%s %s %s %s %s %s %s %s %s %s %s",
		glyph(r.Health), visGlyph(r.Private), pad(r.Name, 26), ciStyled(r.CI), pad(r.ShortSHA, 8),
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
	undep := r.UndeployedLabel()
	if m.detailData != nil && m.detailData.AheadKnown && !r.Untagged {
		undep = fmt.Sprintf("%d commit(s) since %s", m.detailData.AheadBy, dashIfEmpty(r.LatestTag))
	}
	fmt.Fprintf(&b, "  undeployed    %s\n", undep)
	fmt.Fprintf(&b, "  dependabot    crit=%d high=%d mod=%d low=%d  (enabled=%v)\n",
		r.Dependabot.Critical, r.Dependabot.High, r.Dependabot.Moderate, r.Dependabot.Low, r.DependabotEnabled)
	fmt.Fprintf(&b, "  code scan     %s\n", scanCell(r.CodeScanEnabled, r.CodeScanning))
	fmt.Fprintf(&b, "  secret scan   %s\n", scanCell(r.SecretScanEnabled, r.SecretScanning))
	fmt.Fprintf(&b, "  open PRs      total=%d  bot=%d human=%d  mergeable=%d ci-green=%d draft=%d\n",
		r.PRs.Total, r.PRs.Bot, r.PRs.Human, r.PRs.Mergeable, r.PRs.CIGreen, r.PRs.Draft)
	// Lazily-loaded alert + PR lists.
	b.WriteString("\n")
	if m.detailLoading {
		b.WriteString(styleDim.Render("  loading alerts + PRs...") + "\n")
	} else if d := m.detailData; d != nil {
		if len(d.Alerts) == 0 {
			b.WriteString(styleDim.Render("  no open alerts") + "\n")
		} else {
			b.WriteString(styleKey.Render("  open alerts:") + "\n")
			for _, a := range d.Alerts {
				sev := lipgloss.NewStyle().Foreground(sevColor(a.Severity)).Render(pad(strings.ToUpper(a.Severity), 8))
				fmt.Fprintf(&b, "    %s %s  %s\n", sev, pad(trunc(a.Package, 18), 18), trunc(a.Summary, m.width-36))
			}
		}
		if len(d.PRs) > 0 {
			b.WriteString(styleKey.Render("  open PRs:") + "\n")
			for _, p := range d.PRs {
				draft := ""
				if p.Draft {
					draft = styleDim.Render(" (draft)")
				}
				fmt.Fprintf(&b, "    #%-5d %s%s\n", p.Number, trunc(p.Title, m.width-18), draft)
			}
		}
	}
	b.WriteString("\n" + styleFooter.Render("  enter/esc back   o open in browser   q quit"))
	return b.String()
}

func sevColor(sev string) lipgloss.Color {
	switch strings.ToLower(sev) {
	case "critical", "high":
		return colRed
	case "moderate", "medium":
		return colYellow
	default:
		return colGray
	}
}

// trunc shortens s to at most n visible columns (no padding).
func trunc(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) > n {
		if n <= 1 {
			return s[:n]
		}
		return s[:n-1] + "~"
	}
	return s
}

func (m Model) footer() string {
	rl := m.state.RateLimit
	var parts []string
	if m.state.FromCache {
		parts = append(parts, styleWarn.Render(fmt.Sprintf("cached %s ago (r=refresh)", time.Since(m.state.FetchedAt).Round(time.Second))))
	}
	gqcost := ""
	if rl.GraphQLCost > 0 {
		gqcost = fmt.Sprintf(" (cost %d pts)", rl.GraphQLCost)
	}
	parts = append(parts, fmt.Sprintf("REST %d/%d  GraphQL %d/%d%s", rl.RESTRemaining, rl.RESTLimit, rl.GraphQLRemaining, rl.GraphQLLimit, gqcost))
	if n := len(m.state.Warnings); n > 0 {
		parts = append(parts, styleWarn.Render(fmt.Sprintf("%d warning(s)", n)))
	}
	legend := "health: " +
		lipgloss.NewStyle().Foreground(colGreen).Render("[OK] ok") + "  " +
		lipgloss.NewStyle().Foreground(colYellow).Render("[~] attention") + "  " +
		lipgloss.NewStyle().Foreground(colRed).Render("[!!] CI-fail or crit/high") + "  " +
		styleDim.Render("[?] unknown")
	cols := "cols: P=public repo   DEP=crit/high/mod/low   UNDEP=commits since last tag   PR=bot/human   CODE/SEC=scan alerts (? = off)"
	keys := "j/k move   space/pgdn page   g/G top/bottom   enter drill   tab fold   / filter   s sort   a attn   o open   r refresh   q quit"
	return styleFooter.Render("  ") + strings.Join(parts, "   ") + "\n" +
		"  " + legend + "\n" +
		styleFooter.Render("  "+cols) + "\n" +
		styleFooter.Render("  "+keys)
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
