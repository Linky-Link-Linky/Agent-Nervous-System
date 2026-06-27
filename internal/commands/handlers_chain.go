package commands

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	path "path/filepath"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/chain"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/identity"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
)

// --- chain ---

func cmdChain(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("chain", flag.ContinueOnError)
	n := fs.Int("n", 20, "Number of receipts to show")
	agentFilter := fs.String("agent", "", "Filter by agent ID")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}

	c, err := chain.Open("")
	if err != nil {
		return fmt.Errorf("opening chain: %v", err)
	}
	defer c.Close()

	limit := *n
	if limit < 0 {
		limit = 0
	}
	receipts, err := c.List(chain.QueryOptions{Limit: limit, AgentID: *agentFilter})
	if err != nil {
		return fmt.Errorf("querying chain: %v", err)
	}
	if len(receipts) == 0 {
		fmt.Fprint(w, "\nans: no receipts yet. Add @ans.trace to your agent tools to start recording.\n\n")
		return nil
	}
	for i, j := 0, len(receipts)-1; i < j; i, j = i+1, j-1 {
		receipts[i], receipts[j] = receipts[j], receipts[i]
	}
	pretty.PrintChain(w, receipts, noColor())
	return nil
}

// --- verify ---

func cmdVerify(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fullChain := fs.Bool("chain", false, "Verify entire chain integrity")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}

	if *fullChain {
		c, err := chain.Open("")
		if err != nil {
			return fmt.Errorf("opening chain: %v", err)
		}
		defer c.Close()
		pubkeys := make(map[string]ed25519.PublicKey)
		if ks, err := identity.NewKeystore(""); err == nil {
			ids, _ := ks.List()
			for _, id := range ids {
				ag, loadErr := ks.Load(id)
				if loadErr == nil {
					pubkeys[ag.ID] = ag.PublicKey
				}
			}
		} else {
			fmt.Fprintf(w, "ans: warning: keystore unavailable, signature verification skipped: %v\n", err)
		}
		result := c.VerifyChain(pubkeys)
		if result.Valid {
			fmt.Fprintf(w, "\n"+pretty.Green+pretty.Bold+"✓ Chain integrity verified"+pretty.Reset+" — %d receipts checked\n\n", result.TotalChecked)
		} else {
			fmt.Fprintf(w, "\n"+pretty.Red+pretty.Bold+"✗ Chain integrity FAILED"+pretty.Reset+" at index %d: %s\n\n", result.FirstBrokenAt, result.Error)
			return fmt.Errorf("chain verification failed")
		}
		return nil
	}

	receiptID := fs.Arg(0)
	if receiptID == "" {
		return fmt.Errorf("usage: ans verify <receipt_id>  or  ans verify --chain")
	}
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgVerify, daemon.VerifyReq{ReceiptID: receiptID}); err != nil {
		return fmt.Errorf("sending verify: %v", err)
	}
	var resp map[string]interface{}
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("reading verify response: %v", err)
	}
	pretty.PrintVerifyResult(w, resp, noColor())
	if valid, _ := resp["valid"].(bool); !valid {
		return fmt.Errorf("verification failed")
	}
	return nil
}

// --- agents ---

func cmdAgents(w io.Writer, args []string) error {
	_ = args
	ks, err := identity.NewKeystore("")
	if err != nil {
		return fmt.Errorf("opening keystore: %v", err)
	}
	ids, err := ks.List()
	if err != nil {
		return fmt.Errorf("listing agents: %v", err)
	}
	if len(ids) == 0 {
		pretty.Warn(w, "No agents registered yet")
		return nil
	}
	pretty.Header(w, "Registered Agents")
	for _, id := range ids {
		ag, err := ks.Load(id)
		if err != nil {
			pretty.Item(w, id, pretty.Red+"error: "+err.Error()+pretty.Reset)
			continue
		}
		pretty.Item(w, ag.ID, ag.Name+"  "+pretty.Dim+ag.Version+pretty.Reset)
	}
	fmt.Fprintln(w)
	return nil
}

// --- register ---

func cmdRegister(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	name := fs.String("name", "", "Agent name (generated if empty)")
	versionFlag := fs.String("version", "1.0.0", "Agent version")
	owner := fs.String("owner", "", "Owner/creator of the agent")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}
	b := make([]byte, 4)
	rand.Read(b)
	rnd := hex.EncodeToString(b)
	if *name == "" {
		*name = "agent-" + rnd
	}

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgRegister, daemon.RegisterReq{
		Name: *name, Version: *versionFlag, Owner: *owner,
	}); err != nil {
		return fmt.Errorf("sending register: %v", err)
	}
	var resp daemon.RegisterResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("register failed: %v", err)
	}
	pretty.Done(w, "Agent registered")
	pretty.Item(w, "Agent ID", resp.AgentID)
	pretty.Item(w, "Name", *name)
	pretty.Item(w, "Version", *versionFlag)
	if *owner != "" {
		pretty.Item(w, "Owner", *owner)
	}
	fmt.Fprintln(w)
	return nil
}

// --- export ---

func cmdExport(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	format := fs.String("format", "jsonl", "Export format: jsonl, csv, txt, pdf")
	output := fs.String("output", "", "Output file (default stdout)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}

	c, err := chain.Open("")
	if err != nil {
		return fmt.Errorf("opening chain: %v", err)
	}
	defer c.Close()

	exportW := io.Writer(w)
	if *output != "" {
		clean := path.Clean(*output)
		if !path.IsAbs(clean) {
			abs, err := path.Abs(clean)
			if err != nil {
				return fmt.Errorf("resolving output path: %v", err)
			}
			clean = abs
		}
		abs, err := path.Abs(clean)
		if err != nil {
			return fmt.Errorf("resolving output path: %v", err)
		}
		f, err := os.Create(abs)
		if err != nil {
			return fmt.Errorf("creating output file: %v", err)
		}
		defer f.Close()
		exportW = f
	}

	var exportErr error
	switch *format {
	case "jsonl":
		exportErr = c.ExportJSONL(exportW, chain.QueryOptions{})
	case "csv":
		exportErr = c.ExportCSV(exportW, chain.QueryOptions{})
	case "txt", "text":
		exportErr = c.ExportTextReport(exportW)
	case "pdf":
		exportErr = c.ExportPDF(exportW)
	case "parquet":
		exportErr = c.ExportParquet(exportW, chain.QueryOptions{})
	default:
		return fmt.Errorf("unknown format %q -- use jsonl, csv, txt, pdf, or parquet", *format)
	}
	if exportErr != nil {
		return fmt.Errorf("export failed: %v", exportErr)
	}
	if *output != "" {
		pretty.Done(w, "Exported to "+*output)
	}
	return nil
}

// --- prune ---

func cmdPrune(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("prune", flag.ContinueOnError)
	upTo := fs.Uint64("up-to", 0, "Prune receipts with chain_index <= this value (required)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}
	if *upTo == 0 {
		return fmt.Errorf("usage: ans prune --up-to <chain_index>")
	}
	c, err := chain.Open("")
	if err != nil {
		return fmt.Errorf("opening chain: %v", err)
	}
	defer c.Close()
	anchor, err := c.Prune(*upTo)
	if err != nil {
		return fmt.Errorf("pruning chain: %v", err)
	}
	pretty.Done(w, fmt.Sprintf("Pruned %d receipts (index %d-%d)", anchor.Count, anchor.FromIndex, anchor.ToIndex))
	pretty.Item(w, "Merkle root", anchor.MerkleRoot)
	pretty.Item(w, "Anchor ID", fmt.Sprintf("%d", anchor.ID))
	fmt.Fprintln(w)
	return nil
}

// --- rotate ---

func cmdRotate(w io.Writer, args []string) error {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("usage: ans rotate <agent-id>")
	}
	agentID := args[0]
	ks, err := identity.NewKeystore("")
	if err != nil {
		return fmt.Errorf("opening keystore: %v", err)
	}
	newAgent, rec, err := ks.Rotate(agentID)
	if err != nil {
		return fmt.Errorf("rotating key: %v", err)
	}
	pretty.Done(w, "Key rotated successfully")
	pretty.Item(w, "Old agent ID", agentID)
	pretty.Item(w, "New agent ID", newAgent.ID)
	pretty.Item(w, "New public key", fmt.Sprintf("%x", newAgent.PublicKey))
	pretty.Item(w, "Rotation record", fmt.Sprintf("old_sig=%s... new_sig=%s...", safeSig(rec.OldSignature), safeSig(rec.NewSignature)))
	pretty.Warn(w, "Update your SDK configuration to use the new agent ID")
	return nil
}
