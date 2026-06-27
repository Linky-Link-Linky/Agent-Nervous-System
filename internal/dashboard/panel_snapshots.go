package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func snapSizeStr(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%dB", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
}

func (m DashboardModel) renderSnapshotPanel() string {
	w := m.snapPanelWidth()
	h := m.snapPanelHeight()
	style := theme.PanelStyle(m.focus == focusSnaps).Width(w).Height(h)
	title := theme.TitleStyle(m.focus == focusSnaps).Render(" ⏱ SNAPSHOTS & TIME-TRAVEL ")
	if len(m.snaps) == 0 {
		return style.Render(title + "\n\n  No snapshots yet.")
	}

	var b strings.Builder
	b.WriteString(title + "\n")

	header := lipgloss.JoinHorizontal(lipgloss.Left,
		theme.TableHeader.Render(theme.PadRight("SNAP ID", 12)),
		theme.TableHeader.Render(theme.PadRight("TYPE", 12)),
		theme.TableHeader.Render(theme.PadRight("IDX", 9)),
		theme.TableHeader.Render(theme.PadRight("SIZE", 9)),
		theme.TableHeader.Render(theme.PadRight("TIME", 8)),
		theme.TableHeader.Render(theme.PadRight("AGENT", 13)),
	)
	b.WriteString(header + "\n")
	b.WriteString(theme.TableHeader.Render(strings.Repeat("─", w-4)) + "\n")

	n := len(m.snaps)
	if n > 10 {
		n = 10
	}
	rows := m.snaps[:n]

	for i, r := range rows {
		selected := i == m.snapCursor && m.focus == focusSnaps
		rowStyle := theme.TableRowNormal
		if selected {
			rowStyle = theme.TableRowSelected
		}
		id := theme.CLIHash.Render(theme.PadRight(r.ShortID(), 12))
		typ := theme.PadRight(r.Type, 12)
		idx := theme.PadRight(fmt.Sprintf("%d", r.ChainIndex), 9)
		sz := theme.PadRight(snapSizeStr(r.SizeBytes), 9)
		tm := r.Timestamp.Format("15:04:05")
		ts := theme.PadRight(tm, 8)
		agent := theme.CLIAgentID.Render(theme.PadRight(r.AgentID, 13))
		line := rowStyle.Render(id + " " + typ + " " + idx + " " + sz + " " + ts + " " + agent)
		b.WriteString(line + "\n")
	}

	var total int64
	oldest := time.Now()
	newest := time.Time{}
	for _, s := range m.snaps {
		total += s.SizeBytes
		if s.Timestamp.Before(oldest) {
			oldest = s.Timestamp
		}
		if s.Timestamp.After(newest) {
			newest = s.Timestamp
		}
	}
	b.WriteString(fmt.Sprintf("\n  TOTAL %d snapshots  SIZE %s", len(m.snaps), snapSizeStr(total)))
	return style.Render(b.String())
}
