package dashboard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var ansiRE = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]|\033][^\a]*(\a|\033\\)`)

type commandBar struct {
	flex      *tview.Flex
	input     *tview.InputField
	output    *tview.TextView
	app       *tview.Application
	provider  providers.DashboardProvider
	inputMode bool
	history   []string
	histPos   int
}

func newCommandBar(app *tview.Application, provider providers.DashboardProvider) *commandBar {
	input := tview.NewInputField().
		SetLabel("[#2ecc71]>[-] ").
		SetFieldWidth(0).
		SetPlaceholder("register, chain, status, help, etc.  (Esc to cancel)").
		SetPlaceholderTextColor(tcell.NewRGBColor(0x4C, 0x1D, 0x95))
	input.SetBackgroundColor(bgColor)

	output := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetScrollable(true)
	output.SetBackgroundColor(bgColor)
	output.SetText("[#94a3b8]Press [:#2ecc71]:[-] or [:#2ecc71]/[-] for commands  |  [#94a3b8]Hotkeys: [:#2ecc71]1[-]status [:#2ecc71]2[-]chain [:#2ecc71]3[-]agents [:#2ecc71]4[-]verify [:#2ecc71]s[-]snap [:#2ecc71]h[-]help  [:#2ecc71]q[-]quit[-]\n")

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
			cb.inputMode = false
			cb.app.SetFocus(nil)
		}
		if key == tcell.KeyEscape {
			cb.inputMode = false
			cb.app.SetFocus(nil)
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

	cb.flex.AddItem(input, 1, 0, false)
	cb.flex.AddItem(output, 0, 1, false)

	return cb
}

func (c *commandBar) activate() {
	c.inputMode = true
	c.app.SetFocus(c.input)
}

func (c *commandBar) deactivate() {
	c.inputMode = false
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

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func escBrackets(s string) string {
	return strings.ReplaceAll(s, "[", "[[")
}

func (c *commandBar) execute(raw string) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return
	}

	// Strip leading "ans" if user types "ans chain" thinking it's a shell
	if parts[0] == "ans" && len(parts) > 1 {
		parts = parts[1:]
	}

	cmdName := parts[0]

	if cmdName == "help" || cmdName == "--help" || cmdName == "-h" {
		if len(parts) > 1 {
			// Delegate to ans <subcommand> --help
			rest := strings.Join(parts[1:], " ") + " --help"
			self, err := os.Executable()
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, self, strings.Fields(rest)...)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				cmd.Run()
				out := stdout.String()
				if out == "" {
					out = stderr.String()
				}
				plain := stripANSI(out)
				if plain == "" {
					plain = "(no help available)"
				}
				plain = escBrackets(plain)
				c.showOutput(fmt.Sprintf("[#e2e8f0]%s[-]", plain))
				return
			}
		}
		c.showLocalHelp()
		return
	}

	// Built-in clear: clear the output area
	if cmdName == "clear" || cmdName == "cls" {
		c.output.SetText("")
		return
	}

	self, err := os.Executable()
	if err != nil {
		c.showOutput(fmt.Sprintf("[#f472b6]error:[-] [#94a3b8]cannot find binary: %v[-]", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, self, parts...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	out := stdout.String()
	errOut := stderr.String()

	plain := out
	if plain == "" {
		plain = errOut
	}
	if runErr != nil && plain == "" {
		plain = runErr.Error()
	}
	if ctx.Err() == context.DeadlineExceeded {
		plain = "command timed out after 30s"
	}

	plain = stripANSI(plain)

	if plain == "" {
		plain = "(ok)"
	}
	if len(plain) > 2000 {
		plain = plain[:2000] + "\n... (truncated)"
	}

	plain = escBrackets(plain)
	cmdDisplay := escBrackets(raw)

	display := fmt.Sprintf("[#2ecc71]>[-] [#e2e8f0]%s[-]\n[#e2e8f0]%s[-]", cmdDisplay, plain)
	c.showOutput(display)

	c.provider.RecentEvents()
	c.app.QueueUpdateDraw(func() {})
}

func (c *commandBar) showLocalHelp() {
	help := `[#2ecc71]Hotkeys (press while not typing)[-]
  [#e2e8f0]1[-] [#94a3b8]status[-]     [#e2e8f0]2[-] [#94a3b8]chain --n 5[-]  [#e2e8f0]3[-] [#94a3b8]agents[-]     [#e2e8f0]4[-] [#94a3b8]verify --chain[-]
  [#e2e8f0]s[-] [#94a3b8]snapshot[-]    [#e2e8f0]h[-] [#94a3b8]this help[-]    [#e2e8f0]c[-] [#94a3b8]clear[-]      [#e2e8f0]q[-] [#94a3b8]quit[-]
  [#e2e8f0]:[/][-][#94a3b8] command bar[-]

[#2ecc71]CLI commands (type after pressing : or /)[-]

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
	c.output.SetText(text)
}
