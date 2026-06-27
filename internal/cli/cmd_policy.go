package cli

import (
	"flag"
	"fmt"
	"strings"

	"ans/internal/client"
)

func runPolicy(args []string, c client.Client) {
	if len(args) == 0 {
		Fail("Usage: ans policy <add|list|remove|eval> [flags]")
		exitErr(1)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "add":
		runPolicyAdd(rest, c)
	case "list":
		runPolicyList(rest, c)
	case "remove":
		runPolicyRemove(rest, c)
	case "eval":
		runPolicyEval(rest, c)
	default:
		Fail("unknown policy subcommand: " + sub)
		exitErr(1)
	}
}

func runPolicyAdd(args []string, c client.Client) {
	if len(args) == 0 {
		Fail("Usage: ans policy add <file.json|file.yaml>")
		exitErr(1)
	}
	policy, err := c.PolicyAdd(args[0])
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Policy added: " + policy.ID)
}

func runPolicyList(args []string, c client.Client) {
	fs := flag.NewFlagSet("policy list", flag.ExitOnError)
	enabled := fs.Bool("enabled", false, "only enabled")
	fs.Parse(args)

	policies, err := c.PolicyList(*enabled)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	if len(policies) == 0 {
		Warn("No policies")
		return
	}
	fmt.Println(Bold("POLICIES"))
	fmt.Println(Dim("────────────────────────────────────────────"))
	for _, p := range policies {
		status := Green("allow")
		if p.Effect == "deny" {
			status = Red("deny")
		}
		active := ""
		if !p.Enabled {
			active = Dim(" (disabled)")
		}
		fmt.Printf("  %s  %s  %s %s%s\n",
			Purple(trunc(p.ID, 10)),
			p.ActionType,
			status,
			p.PayloadSummary,
			active)
	}
}

func runPolicyRemove(args []string, c client.Client) {
	if len(args) == 0 {
		Fail("Usage: ans policy remove <id>")
		exitErr(1)
	}
	if err := c.PolicyRemove(args[0]); err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Removed: " + args[0])
}

func runPolicyEval(args []string, c client.Client) {
	fs := flag.NewFlagSet("policy eval", flag.ExitOnError)
	actionType := fs.String("action-type", "", "action type URI")
	payload := fs.String("payload-summary", "", "payload summary text")
	fs.Parse(args)

	if *actionType == "" {
		Fail("--action-type is required")
		exitErr(1)
	}

	suggestion := *payload
	if suggestion == "" {
		suggestion = strings.Join(fs.Args(), " ")
	}

	result, err := c.PolicyEval(*actionType, suggestion)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}

	if result.Allowed {
		OK(fmt.Sprintf("ALLOW — matched %s", result.MatchedPolicy))
	} else {
		Fail(fmt.Sprintf("DENY — matched %s", result.MatchedPolicy))
		exitErr(1)
	}
}
