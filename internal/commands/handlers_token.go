package commands

import (
	"flag"
	"fmt"
	"io"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
)

// --- token request ---

func cmdTokenRequest(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("token request", flag.ContinueOnError)
	resource := fs.String("resource", "", "Resource ARN or path (required)")
	action := fs.String("action", "read", "Action (read, write, etc.)")
	ttl := fs.Int("ttl", 60, "Token TTL in seconds (max 60)")
	_ = fs.Parse(args)
	if *resource == "" {
		return fmt.Errorf("usage: ans token request --resource <arn> [--action read] [--ttl 60]")
	}

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgTokenRequest, daemon.TokenRequestReq{
		AgentID: "_cli", Resource: *resource, Action: *action, TTLSeconds: *ttl, SingleUse: true,
	})
	var resp daemon.TokenRequestResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("token request failed: %v", err)
	}
	if !resp.Success {
		return fmt.Errorf("token provisioning failed: %s", resp.Message)
	}
	pretty.Done(w, "Token provisioned")
	pretty.Item(w, "Token ID", resp.TokenID)
	pretty.Item(w, "Type", resp.TokenType)
	pretty.Item(w, "Access Key", maskSecret(resp.AccessKey))
	pretty.Item(w, "Secret Key", maskSecret(resp.SecretKey))
	pretty.Item(w, "Bearer", maskSecret(resp.BearerToken))
	pretty.Item(w, "Resource", resp.Resource)
	pretty.Item(w, "Expires", fmt.Sprintf("%d ns", resp.ExpiresNS))
	fmt.Fprintln(w)
	return nil
}

// --- token list ---

func cmdTokenList(w io.Writer, args []string) error {
	_ = args
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgTokenList, daemon.TokenListReq{})
	var resp daemon.TokenListResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("token list failed: %v", err)
	}
	if len(resp.Tokens) == 0 {
		pretty.Warn(w, "No active tokens")
		return nil
	}
	pretty.Header(w, "Active Tokens")
	for _, t := range resp.Tokens {
		stateClr := pretty.Green
		if t.State == "revoked" || t.State == "expired" {
			stateClr = pretty.Red
		}
		pretty.Item(w, t.TokenID, fmt.Sprintf("%s  type=%s  resource=%s  state=%s%s%s", t.Provider, t.TokenType, t.Resource, stateClr, t.State, pretty.Reset))
	}
	fmt.Fprintln(w)
	return nil
}

// --- token revoke ---

func cmdTokenRevoke(w io.Writer, args []string) error {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("usage: ans token revoke <token-id>")
	}
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgTokenRevoke, daemon.TokenRevokeReq{TokenID: args[0]})
	var resp daemon.TokenRevokeResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("token revoke failed: %v", err)
	}
	if resp.Success {
		pretty.Done(w, fmt.Sprintf("Token %q revoked", args[0]))
	} else {
		return fmt.Errorf("revoking token: %s", resp.Message)
	}
	return nil
}
