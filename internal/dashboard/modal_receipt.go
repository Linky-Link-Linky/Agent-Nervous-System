package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderReceiptDetailModal() string {
	r := m.selectedReceipt
	if r == nil {
		return ""
	}
	lines := []string{
		theme.ModalTitle.Render("RECEIPT DETAIL"),
		"",
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("INDEX"), theme.ModalValue.Render(fmt.Sprintf("%d", r.Index))),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("HASH"), theme.CLIHash.Render(r.ID)),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("PREV HASH"), theme.CLIHash.Render(r.PrevHash)),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("AGENT"), theme.CLIAgentID.Render(r.AgentID)),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("ACTION TYPE"), r.ActionType),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("PHASE"), r.Phase),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("OUTCOME"), theme.OutcomeStyle(r.Outcome).Render(theme.OutcomeGlyph(r.Outcome))),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("DURATION"), theme.FormatDuration(r.DurationMS)),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("TIMESTAMP"), r.Timestamp.Format(time.RFC3339)),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("SNAPSHOT"), r.SnapshotID),
		fmt.Sprintf("  %-18s %s", theme.ModalLabel.Render("POLICY"), r.PolicyDecision),
		"",
		fmt.Sprintf("  %-18s", theme.ModalLabel.Render("SIGNATURE")),
		"  " + theme.CLIHash.Render(r.Signature),
		"",
		fmt.Sprintf("  %-18s", theme.ModalLabel.Render("PAYLOAD")),
		"  " + r.PayloadSummary,
		"",
	}
	content := strings.Join(lines, "\n")
	return theme.ModalBox.Width(60).Render(content)
}
