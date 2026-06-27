package dashboard

import (
	"fmt"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderTokenPanel() string {
	w := m.panel3Width()
	h := row2Height(m.height, m.width)
	style := theme.PanelStyle(m.focus == focusTokens).Width(w).Height(h)
	title := theme.TitleStyle(m.focus == focusTokens).Render(" ⚡ EPHEMERAL TOKENS ")
	if len(m.tokens) == 0 {
		return style.Render(title + "\n\n  No active tokens.")
	}

	var b string
	b += title + "\n"

	b += fmt.Sprintf("%-12s %-8s %-20s %-10s %s\n",
		theme.TableHeader.Render("CRED ID"),
		theme.TableHeader.Render("TYPE"),
		theme.TableHeader.Render("RESOURCE"),
		theme.TableHeader.Render("TTL"),
		theme.TableHeader.Render("AGENT"))

	for i, t := range m.tokens {
		selected := i == m.tokenCursor && m.focus == focusTokens
		rowStyle := theme.TableRowNormal
		if selected {
			rowStyle = theme.TableRowSelected
		}

		ttl := t.TTLSeconds()
		ttlBar := TTLBar(ttl, 60)

		line := fmt.Sprintf("%-12s %-8s %-20s %-10s %s",
			t.ShortID(),
			t.TypeIcon()+" "+t.Type,
			t.ShortResource(),
			fmt.Sprintf("%ds %s", ttl, ttlBar),
			t.AgentID)
		b += rowStyle.Render(line) + "\n"
	}

	active := 0
	for _, t := range m.tokens {
		if !t.Revoked {
			active++
		}
	}
	b += fmt.Sprintf("\n  %d active tokens — max TTL 60s — zero-trust", active)
	return style.Render(b)
}
