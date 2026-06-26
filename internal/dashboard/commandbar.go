package dashboard

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/commands"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var ansiRE = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]|\033][^\a]*(\a|\033\\)`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

type commandBar struct {
	flex     *tview.Flex
	input    *tview.InputField
	output   *tview.TextView
	app      *tview.Application
	provider providers.DashboardProvider
	history  []string
	histPos  int
	outputs  []string
}

func newCommandBar(app *tview.Application, provider providers.DashboardProvider) *commandBar {
	input := tview.NewInputField().
		SetLabel("[#2ecc71]>[-] ").
		SetFieldWidth(0).
		SetPlaceholder("Type a command (Enter to run, Esc to clear, Up/Down for history)")
	input.SetBackgroundColor(bgColor)
	input.SetPlaceholderTextColor(tcell.NewRGBColor(0x94, 0xA3, 0xB8))
	input.SetFieldTextColor(foreground)
	input.SetLabelColor(primaryColor)

	output := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetScrollable(true)
	output.SetBackgroundColor(bgColor)
	output.SetText("")

	cb := &commandBar{
		flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		input:    input,
		output:   output,
		app:      app,
		provider: provider,
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			raw := strings.TrimSpace(input.GetText())
			if raw != "" {
				cb.addHistory(raw)
				cb.execute(raw)
			}
			input.SetText("")
			app.SetFocus(input)
		}
		if key == tcell.KeyEscape {
			input.SetText("")
			app.SetFocus(input)
		}
	})

	input.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyUp {
			cb.histPrev()
			return nil
		}
		if ev.Key() == tcell.KeyDown {
			cb.histNext()
			return nil
		}
		return ev
	})

	hintText := "[#94a3b8]Type a command and press Enter  |  Hotkeys (empty input): [#2ecc71]1[-]status [#2ecc71]2[-]chain [#2ecc71]3[-]agents [#2ecc71]4[-]verify [#2ecc71]s[-]snap [#2ecc71]h[-]help [#2ecc71]q[-]quit[-]"
	cb.outputs = []string{hintText}

	cb.flex.AddItem(input, 1, 0, false)
	cb.flex.AddItem(output, 0, 1, false)

	return cb
}

func (c *commandBar) addHistory(cmd string) {
	c.history = append(c.history, cmd)
	c.histPos = len(c.history)
	if len(c.history) > 50 {
		c.history = c.history[len(c.history)-50:]
	}
}

func (c *commandBar) histPrev() {
	if len(c.history) == 0 || c.histPos <= 0 {
		return
	}
	c.histPos--
	c.input.SetText(c.history[c.histPos])
}

func (c *commandBar) histNext() {
	if c.histPos >= len(c.history)-1 {
		c.input.SetText("")
		c.histPos = len(c.history)
		return
	}
	c.histPos++
	c.input.SetText(c.history[c.histPos])
}

func escBrackets(s string) string {
	return strings.ReplaceAll(s, "[", "[[")
}

func (c *commandBar) execute(raw string) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return
	}

	if parts[0] == "ans" && len(parts) > 1 {
		parts = parts[1:]
	}

	cmdName := parts[0]

	if cmdName == "help" || cmdName == "--help" || cmdName == "-h" {
		if len(parts) > 1 {
			c.runCmdAsync(raw, parts, 10*time.Second)
		} else {
			c.showLocalHelp()
		}
		return
	}
	if cmdName == "clear" || cmdName == "cls" {
		c.outputs = nil
		c.output.SetText("")
		return
	}

	c.runCmdAsync(raw, parts, 30*time.Second)
}

func (c *commandBar) runCmdAsync(raw string, parts []string, timeout time.Duration) {
	pending := fmt.Sprintf("[#2ecc71]>[-] [#94a3b8]%s[-]\n[#f59e0b]  running...[-]", escBrackets(raw))
	c.showOutput(pending)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan string, 1)

		go func() {
			var buf strings.Builder
			err := commands.DispatchTo(&buf, parts)
			result := buf.String()
			if err != nil && err.Error() != "" {
				if result != "" && !strings.HasSuffix(result, "\n") {
					result += "\n"
				}
				result += err.Error()
			}
			if result == "" {
				result = "(ok)"
			}
			done <- result
		}()

		var result string
		select {
		case result = <-done:
		case <-ctx.Done():
			result = "command timed out"
		}

		plain := stripANSI(result)
		if len(plain) > 2000 {
			plain = plain[:2000] + "\n... (truncated)"
		}

		display := fmt.Sprintf("[#2ecc71]>[-] [#e2e8f0]%s[-]\n[#e2e8f0]%s[-]", escBrackets(raw), escBrackets(plain))

		c.app.QueueUpdateDraw(func() {
			c.showOutput(display)
			c.provider.RecentEvents()
		})
	}()
}

func (c *commandBar) showLocalHelp() {
	help := `[#2ecc71]Hotkeys (with empty input)[-]
  [#e2e8f0]1[-] [#94a3b8]status[-]     [#e2e8f0]2[-] [#94a3b8]chain --n 5[-]  [#e2e8f0]3[-] [#94a3b8]agents[-]     [#e2e8f0]4[-] [#94a3b8]verify --chain[-]
  [#e2e8f0]s[-] [#94a3b8]snapshot[-]    [#e2e8f0]h[-] [#94a3b8]this help[-]    [#e2e8f0]c[-] [#94a3b8]clear[-]      [#e2e8f0]q[-] [#94a3b8]quit[-]

[#2ecc71]CLI commands (type in the bar and press Enter)[-]

[#94a3b8]Setup[-]
  [#e2e8f0]init, start, stop, status, doctor, update, uninstall[-]

[#94a3b8]Chain & Receipts[-]
  [#e2e8f0]chain, verify, agents, register, export, prune, rotate[-]

[#94a3b8]Time-Travel & Snapshots[-]
  [#e2e8f0]time-travel, snapshot take/diff/list, snapshots[-]

[#94a3b8]Policy & Tokens[-]
  [#e2e8f0]policy add/list/remove/eval, token request/list/revoke[-]

[#94a3b8]MCP Proxy[-]
  [#e2e8f0]mcp start/stop/status/log[-]

[#94a3b8]Other[-]
  [#e2e8f0]version, dashboard, clear[-]

[#94a3b8]Tip:[-] Run [#e2e8f0]help <command>[-] for details (e.g. [#e2e8f0]help chain[-])
`
	c.showOutput(help)
}

func (c *commandBar) showOutput(text string) {
	c.outputs = append(c.outputs, text)
	if len(c.outputs) > 10 {
		c.outputs = c.outputs[len(c.outputs)-10:]
	}
	c.output.SetText(strings.Join(c.outputs, "\n[#334155]────────────────────────────────────────────────────[-]\n"))
}
