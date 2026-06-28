package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type ProxyModel struct {
	Running           bool
	ListenAddr        string
	TargetURL         string
	TotalRequests     int
	RateLimitHits     int
	PIIRedactions     int
	InjectionsBlocked int
	ReqPerMin         int
	RateLimit         int
	auditLog          []AuditEntry
	vp                viewport.Model
}

type AuditEntry struct {
	Method string
	Path   string
	Status int
	Time   string
}

func NewProxy() ProxyModel {
	entries := make([]AuditEntry, 20)
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for i := range entries {
		entries[i] = AuditEntry{
			Method: methods[i%len(methods)],
			Path:   "/api/v1/chain", Status: 200,
			Time: fmt.Sprintf("%02d:%02d", i, i*3%60),
		}
	}
	return ProxyModel{
		Running: true, ListenAddr: "127.0.0.1:9090", TargetURL: "http://localhost:8080",
		TotalRequests: 15420, RateLimitHits: 43, PIIRedactions: 12, InjectionsBlocked: 7,
		ReqPerMin: 120, RateLimit: 500,
		auditLog: entries,
	}
}

func (m ProxyModel) Init() tea.Cmd { return nil }
func (m ProxyModel) Update(msg tea.Msg) (ProxyModel, tea.Cmd) { return m, nil }

func (m ProxyModel) View(width, height int) string {
	t := styles.CurrentTheme

	// Status pill
	status := t.Badge(" RUNNING ", true)
	if !m.Running {
		status = t.Badge(" STOPPED ", false)
	}

	// Stats row
	stats := []string{
		fmt.Sprintf("Total: %d", m.TotalRequests),
		fmt.Sprintf("Rate-hits: %d", m.RateLimitHits),
		fmt.Sprintf("PII: %d", m.PIIRedactions),
		fmt.Sprintf("Blocked: %d", m.InjectionsBlocked),
	}
	var statCards []string
	for _, s := range stats {
		statCards = append(statCards, t.BoxStyle().Width(18).Render(
			lipgloss.NewStyle().Foreground(t.FgMuted).Render(s)))
	}
	statRow := lipgloss.JoinHorizontal(lipgloss.Top, statCards...)

	// Throughput gauge
	gaugeW := width - 8
	frac := float64(m.ReqPerMin) / float64(m.RateLimit)
	if frac > 1 {
		frac = 1
	}
	filled := int(float64(gaugeW) * frac)
	gauge := strings.Repeat("█", filled) + strings.Repeat("░", gaugeW-filled)
	gaugeView := t.BoxStyle().Render(lipgloss.JoinVertical(lipgloss.Left,
		t.TitleStyle().Render(fmt.Sprintf(" Throughput  %d/%d req/min", m.ReqPerMin, m.RateLimit)),
		lipgloss.NewStyle().Foreground(t.Accent).Render(gauge),
	))

	// Audit log
	var logLines []string
	for _, e := range m.auditLog {
		methodColor := t.Fg
		switch e.Method {
		case "GET":
			methodColor = t.Success
		case "POST":
			methodColor = t.Accent
		case "PUT":
			methodColor = t.Warning
		case "DELETE":
			methodColor = t.Danger
		}
		line := fmt.Sprintf("  %s %s  %d  %s",
			lipgloss.NewStyle().Foreground(methodColor).Bold(true).Render(e.Method),
			lipgloss.NewStyle().Foreground(t.Fg).Render(e.Path),
			e.Status,
			lipgloss.NewStyle().Foreground(t.FgMuted).Render(e.Time),
		)
		logLines = append(logLines, line)
	}
	m.vp.Width = width - 4
	m.vp.Height = height/2 - 2
	m.vp.SetContent(strings.Join(logLines, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("  %s  %s → %s", status,
			lipgloss.NewStyle().Foreground(t.FgMuted).Render(m.ListenAddr),
			lipgloss.NewStyle().Foreground(t.FgMuted).Render(m.TargetURL),
		),
		statRow,
		gaugeView,
		t.BoxStyle().Width(width).Render(m.vp.View()),
	)
}
