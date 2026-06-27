package cli

import (
	"flag"
	"fmt"
	"strings"

	"ans/internal/client"
)

func runToken(args []string, c client.Client) {
	if len(args) == 0 {
		Fail("Usage: ans token <request|list|revoke> [flags]")
		exitErr(1)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "request":
		runTokenRequest(rest, c)
	case "list":
		runTokenList(c)
	case "revoke":
		runTokenRevoke(rest, c)
	default:
		Fail("unknown token subcommand: " + sub)
		exitErr(1)
	}
}

func runTokenRequest(args []string, c client.Client) {
	fs := flag.NewFlagSet("token request", flag.ExitOnError)
	resource := fs.String("resource", "", "resource ARN (required)")
	action := fs.String("action", "read", "action to permit")
	ttl := fs.Int("ttl", 3600, "TTL in seconds")
	fs.Parse(args)

	if *resource == "" {
		Fail("--resource is required")
		exitErr(1)
	}

	tok, err := c.TokenRequest(*resource, *action, *ttl)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Token granted")
	fmt.Printf("  ID:        %s\n", Purple(tok.ID))
	fmt.Printf("  Resource:  %s\n", tok.Resource)
	fmt.Printf("  Action:    %s\n", tok.Action)
	fmt.Printf("  Expires:   %s\n", tok.Expiry.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Token:     %s\n", Dim(trunc(tok.Token, 32)+"…"))
}

func runTokenList(c client.Client) {
	tokens, err := c.ListTokens()
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	if len(tokens) == 0 {
		Warn("No tokens")
		return
	}
	fmt.Println(Bold("ACTIVE TOKENS"))
	fmt.Println(Dim("────────────────────────────────────────────"))
	for _, t := range tokens {
		fmt.Printf("  %s  %-24s %-8s %s\n",
			Purple(trunc(t.ID, 8)),
			t.Resource,
			t.Permissions,
			t.ExpiresAt.Format("2006-01-02 15:04"))
	}
}

func runTokenRevoke(args []string, c client.Client) {
	if len(args) == 0 {
		Fail("Usage: ans token revoke <id>")
		exitErr(1)
	}
	if err := c.TokenRevoke(args[0]); err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Token revoked: " + args[0])
}

func trimQuotes(s string) string {
	return strings.Trim(s, "\"'")
}
