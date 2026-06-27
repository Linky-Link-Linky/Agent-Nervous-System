package dashboard

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/client"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/model"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/poller"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

type focusPanel int

const (
	focusChain focusPanel = iota
	focusSnaps
	focusPolicy
	focusTokens
	focusMCP
)

type modalKind int

const (
	modalNone modalKind = iota
	modalHelp
	modalReceiptDetail
	modalConfirmTimeTravel
	modalConfirmRevoke
	modalConfirmToggle
)

type DashboardModel struct {
	width, height int
	focus         focusPanel
	modal         modalKind
	paused        bool
	demo          bool
	version       string
	poller        *poller.Poller
	client        client.Client
	chain         []*model.Receipt
	chainCursor   int
	chainScrolled bool
	chainVerify   string
	snaps         []*model.Snapshot
	snapCursor    int
	policies      []*model.Policy
	policyCursor  int
	tokens        []*model.Token
	tokenCursor   int
	tickCount     int
	mcpStatus     *model.MCPStatus
	mcpLog        []*model.MCPLogEntry
	mcpLogCursor  int
	daemon        *model.DaemonStatus
	daemonPulse   bool
	selectedReceipt *model.Receipt
	confirmTarget   string
	banner        string
	bannerErr     bool
	bannerTTL     int
	lastErr       string
	quitting      bool
	chainReqRate  []float64
}

type (
	MsgChain        struct{ Receipts []*model.Receipt }
	MsgSnapshots    struct{ Snaps []*model.Snapshot }
	MsgPolicies     struct{ Policies []*model.Policy }
	MsgTokens       struct{ Tokens []*model.Token }
	MsgMCPStatus    struct{ Status *model.MCPStatus }
	MsgMCPLog       struct{ Entries []*model.MCPLogEntry }
	MsgDaemon       struct{ Status *model.DaemonStatus }
	MsgPollError    struct{ Err error }
	MsgBanner       struct{ Text string; IsErr bool }
	MsgTick         struct{}
	MsgVerifyResult struct{ Verified bool; Count int }
)

func New(p *poller.Poller, c client.Client, demo bool, version string) DashboardModel {
	return DashboardModel{poller: p, client: c, demo: demo, version: version, focus: focusChain}
}

func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(listenChannels(m.poller), tickCmd())
}

func listenChannels(p *poller.Poller) tea.Cmd {
	return func() tea.Msg {
		select {
		case v := <-p.C.Chain:
			return MsgChain{v}
		case v := <-p.C.Snaps:
			return MsgSnapshots{v}
		case v := <-p.C.Policy:
			return MsgPolicies{v}
		case v := <-p.C.Token:
			return MsgTokens{v}
		case v := <-p.C.MCP:
			return MsgMCPStatus{v}
		case v := <-p.C.MCPLog:
			return MsgMCPLog{v}
		case v := <-p.C.Daemon:
			return MsgDaemon{v}
		case err := <-p.C.Errors:
			return MsgPollError{err}
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg { return MsgTick{} })
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if m.modal != modalNone {
			return m.updateModal(msg)
		}
		return m.updateGlobal(msg)
	case MsgTick:
		m.daemonPulse = !m.daemonPulse
		m.tickCount++
		if m.bannerTTL > 0 {
			m.bannerTTL--
		}
	case MsgChain:
		if !m.chainScrolled {
			m.chainCursor = 0
		}
		m.chain = msg.Receipts
		if len(msg.Receipts) > 0 {
			m.chainReqRate = append(m.chainReqRate, float64(len(msg.Receipts)))
			if len(m.chainReqRate) > 30 {
				m.chainReqRate = m.chainReqRate[1:]
			}
		}
		cmds = append(cmds, listenChannels(m.poller))
	case MsgSnapshots:
		m.snaps = msg.Snaps
		cmds = append(cmds, listenChannels(m.poller))
	case MsgPolicies:
		m.policies = msg.Policies
		cmds = append(cmds, listenChannels(m.poller))
	case MsgTokens:
		m.tokens = msg.Tokens
		cmds = append(cmds, listenChannels(m.poller))
	case MsgMCPStatus:
		m.mcpStatus = msg.Status
		cmds = append(cmds, listenChannels(m.poller))
	case MsgMCPLog:
		m.mcpLog = msg.Entries
		cmds = append(cmds, listenChannels(m.poller))
	case MsgDaemon:
		m.daemon = msg.Status
		cmds = append(cmds, listenChannels(m.poller))
	case MsgPollError:
		m.lastErr = msg.Err.Error()
		cmds = append(cmds, listenChannels(m.poller))
	case MsgBanner:
		m.banner, m.bannerErr, m.bannerTTL = msg.Text, msg.IsErr, 5
	case MsgVerifyResult:
		if msg.Verified {
			m.chainVerify = "verified"
		} else {
			m.chainVerify = "broken"
		}
	}
	return m, tea.Batch(cmds...)
}

func (m DashboardModel) updateGlobal(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg.String() {
	case "q", "Q", "ctrl+c":
		m.quitting = true
		cmds = append(cmds, tea.Quit)
	case "?":
		m.modal = modalHelp
	case "r", "R":
		m.poller.ForceRefresh()
		m.chainScrolled = false
	case "p", "P":
		m.paused = !m.paused
		if m.paused {
			m.poller.Pause()
		} else {
			m.poller.Resume()
		}
	case "tab":
		m.focus = (m.focus + 1) % 5
	case "shift+tab":
		m.focus = (m.focus + 4) % 5
	case "a", "A":
		m.focus = focusChain
	case "b", "B":
		m.focus = focusSnaps
	case "c", "C":
		m.focus = focusPolicy
	case "d", "D":
		m.focus = focusTokens
	case "e", "E":
		m.focus = focusMCP
	case "up", "k":
		m.moveCursor(-1)
		if m.focus == focusChain {
			m.chainScrolled = true
		}
	case "down", "j":
		m.moveCursor(1)
		if m.focus == focusChain {
			m.chainScrolled = true
		}
	case "enter":
		if m.focus == focusChain && len(m.chain) > 0 {
			m.selectedReceipt = m.chain[m.chainCursor]
			m.modal = modalReceiptDetail
		} else if m.focus == focusSnaps && len(m.snaps) > 0 {
			s := m.snaps[m.snapCursor]
			m.confirmTarget = fmt.Sprintf("restore snapshot %s (index %d)", s.ShortID(), s.ChainIndex)
			m.modal = modalConfirmTimeTravel
		}
	case "v", "V":
		if m.focus == focusChain {
			m.chainVerify = "checking"
			cmds = append(cmds, func() tea.Msg {
				ok, count, _ := m.client.VerifyChain()
				return MsgVerifyResult{Verified: ok, Count: count}
			})
		}
	case "t", "T":
		if m.focus == focusPolicy && len(m.policies) > 0 {
			m.confirmTarget = m.policies[m.policyCursor].ID
			m.modal = modalConfirmToggle
		}
	case "x", "X":
		if m.focus == focusTokens && len(m.tokens) > 0 {
			m.confirmTarget = m.tokens[m.tokenCursor].ID
			m.modal = modalConfirmRevoke
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *DashboardModel) moveCursor(delta int) {
	var n int
	var cursor *int
	switch m.focus {
	case focusChain:
		n = len(m.chain)
		cursor = &m.chainCursor
	case focusSnaps:
		n = len(m.snaps)
		cursor = &m.snapCursor
	case focusPolicy:
		n = len(m.policies)
		cursor = &m.policyCursor
	case focusTokens:
		n = len(m.tokens)
		cursor = &m.tokenCursor
	case focusMCP:
		n = len(m.mcpLog)
		cursor = &m.mcpLogCursor
	}
	if n == 0 || cursor == nil {
		return
	}
	*cursor += delta
	if *cursor < 0 {
		*cursor = 0
	}
	if *cursor >= n {
		*cursor = n - 1
	}
}

func (m DashboardModel) updateModal(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "Q", "n", "N":
		m.modal = modalNone
	case "enter", "y", "Y":
		var cmd tea.Cmd
		switch m.modal {
		case modalConfirmTimeTravel:
			if len(m.snaps) > m.snapCursor && m.snapCursor >= 0 {
				s := m.snaps[m.snapCursor]
				cmd = func() tea.Msg {
					m.client.TimeTravel(fmt.Sprintf("%d", s.ChainIndex), "filesystem")
					return MsgBanner{Text: fmt.Sprintf("Restored to index %d", s.ChainIndex)}
				}
			}
		case modalConfirmToggle:
			if len(m.policies) > m.policyCursor && m.policyCursor >= 0 {
				p := m.policies[m.policyCursor]
				cmd = func() tea.Msg {
					m.client.PolicyToggle(p.ID, !p.Enabled)
					return MsgBanner{Text: fmt.Sprintf("Policy %s toggled", p.ShortID())}
				}
			}
		case modalConfirmRevoke:
			if len(m.tokens) > m.tokenCursor && m.tokenCursor >= 0 {
				t := m.tokens[m.tokenCursor]
				cmd = func() tea.Msg {
					m.client.TokenRevoke(t.ID)
					return MsgBanner{Text: fmt.Sprintf("Token %s revoked", t.ShortID())}
				}
			}
		case modalReceiptDetail:
			if m.selectedReceipt != nil {
				cmd = func() tea.Msg {
					ok, _ := m.client.VerifyReceipt(m.selectedReceipt.ID)
					if ok {
						return MsgBanner{Text: "✓ Signature verified"}
					}
					return MsgBanner{Text: "✗ Invalid signature", IsErr: true}
				}
			}
		}
		m.modal = modalNone
		if cmd != nil {
			return m, cmd
		}
	}
	return m, nil
}

func (m DashboardModel) View() string {
	if m.width == 0 {
		return "Initialising…"
	}
	header := m.renderHeader()
	panelRow1 := lipgloss.JoinHorizontal(lipgloss.Top, m.renderChainPanel(), m.renderSnapshotPanel())
	panelRow2 := lipgloss.JoinHorizontal(lipgloss.Top, m.renderPolicyPanel(), m.renderTokenPanel(), m.renderMCPPanel())
	banner := m.renderBannerStr()
	statusBar := m.renderStatusBar()
	body := lipgloss.JoinVertical(lipgloss.Left, header, panelRow1, panelRow2, banner, statusBar)
	if m.modal != modalNone {
		return m.overlayModal(body)
	}
	return body
}

func (m DashboardModel) renderBannerStr() string {
	if m.bannerTTL <= 0 || m.banner == "" {
		return ""
	}
	s := theme.Success
	if m.bannerErr {
		s = theme.Failure
	}
	return "  " + lipgloss.NewStyle().Foreground(s).Render(m.banner)
}

func (m DashboardModel) chainPanelWidth() int { return m.width*55/100 - 1 }
func (m DashboardModel) snapPanelWidth() int   { return m.width - m.chainPanelWidth() - 2 }
func (m DashboardModel) panel3Width() int      { return m.width/3 - 1 }

func row1Height(totalH int) int {
	a := totalH - 6
	return a * 40 / 100
}

func row2Height(totalH int, _ int) int {
	a := totalH - 6
	return a - row1Height(totalH)
}

func (m DashboardModel) chainPanelHeight() int { return row1Height(m.height) }
func (m DashboardModel) snapPanelHeight() int   { return row1Height(m.height) }
