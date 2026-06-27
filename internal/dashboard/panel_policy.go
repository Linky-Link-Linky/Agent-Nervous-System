package dashboard

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderPolicyPanel() string {
	w := m.panel3Width()
	h := row2Height(m.height, m.width)
	style := theme.PanelStyle(m.focus == focusPolicy).Width(w).Height(h)
	title := theme.TitleStyle(m.focus == focusPolicy).Render(" ⚙ POLICIES ")
	if len(m.policies) == 0 {
		return style.Render(title + "\n\n  No policies configured.")
	}

	var b string
	b += title + "\n"

	b += fmt.Sprintf("%-20s %-6s %-4s %-9s %-7s %s\n",
		theme.TableHeader.Render("ID"),
		theme.TableHeader.Render("ON"),
		theme.TableHeader.Render("PRI"),
		theme.TableHeader.Render("SEV"),
		theme.TableHeader.Render("EFFECT"),
		theme.TableHeader.Render("NAME"))

	for i, p := range m.policies {
		selected := i == m.policyCursor && m.focus == focusPolicy
		rowStyle := theme.TableRowNormal
		if selected {
			rowStyle = theme.TableRowSelected
		}

		onStr := "○ OFF"
		if p.Enabled {
			onStr = "● ON"
		}

		var sevCol lipgloss.Color
		switch p.Severity {
		case "HIGH", "CRITICAL":
			sevCol = theme.Failure
		case "MEDIUM":
			sevCol = theme.Warn
		default:
			sevCol = theme.TextSecondary
		}

		var effCol lipgloss.Color
		switch p.Effect {
		case "deny":
			effCol = theme.Failure
		case "warn":
			effCol = theme.Warn
		default:
			effCol = theme.AccentB
		}

		line := fmt.Sprintf("%-20s %-6s %-4d %-9s %-7s %s",
			p.ShortID(),
			onStr,
			p.Priority,
			lipgloss.NewStyle().Foreground(sevCol).Render(p.Severity),
			lipgloss.NewStyle().Foreground(effCol).Render(p.Effect),
			p.ShortName())
		b += rowStyle.Render(line) + "\n"
	}

	var active, total int
	for _, p := range m.policies {
		total++
		if p.Enabled {
			active++
		}
	}
	b += fmt.Sprintf("\n  %d active / %d total — NociceptionError (0x1F) armed", active, total)
	return style.Render(b)
}
