package dashboard

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

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
}

func newCommandBar(app *tview.Application, provider providers.DashboardProvider) *commandBar {
	input := tview.NewInputField().
		SetLabel("[#a855f7]>[-] ").
		SetFieldWidth(0).
		SetPlaceholder("register, chain, status, help, start, etc.  (Esc to cancel)").
		SetPlaceholderTextColor(tcell.NewRGBColor(0x4C, 0x1D, 0x95))
	input.SetBackgroundColor(bgColor)

	output := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetScrollable(true)
	output.SetBackgroundColor(bgColor)
	output.SetText("[#94a3b8]Press [:#a855f7]:[-] to enter a command, [#94a3b8]Esc to return[-]\n")

	cb := &commandBar{
		flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		input:    input,
		output:   output,
		app:      app,
		provider: provider,
	}

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			cmd := strings.TrimSpace(input.GetText())
			if cmd != "" {
				cb.execute(cmd)
			}
			input.SetText("")
			cb.inputMode = false
		}
		if key == tcell.KeyEscape {
			cb.inputMode = false
		}
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

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func (c *commandBar) execute(cmdText string) {
	self, err := os.Executable()
	if err != nil {
		c.showOutput(fmt.Sprintf("[#f472b6]error:[-] [#94a3b8]cannot find binary: %v[-]", err))
		return
	}

	args := strings.Fields(cmdText)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(self, args...)
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
	plain = stripANSI(plain)

	if plain == "" {
		plain = "(ok)"
	}

	// Truncate to avoid overwhelming the display
	if len(plain) > 2000 {
		plain = plain[:2000] + "\n... (truncated)"
	}

	// Escape brackets for tview color tag parsing
	plain = strings.ReplaceAll(plain, "[", "[[")
	cmdDisplay := strings.ReplaceAll(cmdText, "[", "[[")

	display := fmt.Sprintf("[#a855f7]>[-] [#e2e8f0]%s[-]\n[#e2e8f0]%s[-]", cmdDisplay, plain)
	c.showOutput(display)

	// Refresh panels so data updates after command
	c.provider.RecentEvents()
	c.app.QueueUpdateDraw(func() {})
}

func (c *commandBar) showOutput(text string) {
	c.output.SetText(text)
}
