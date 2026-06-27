package commands

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/snapshot"
)

// --- snapshot take ---

func cmdSnapshotTake(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	agentID := fs.String("agent", "", "Agent ID to snapshot")
	snapType := fs.String("type", "filesystem", "Snapshot type: filesystem, memory, database")
	paths := fs.String("paths", "", "Comma-separated paths to snapshot (empty = full workspace)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}
	if *agentID == "" {
		*agentID = fs.Arg(0)
	}
	if *agentID == "" {
		return fmt.Errorf("usage: ans snapshot take --agent <id> [--type filesystem] [--paths a,b]")
	}

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgSnapshot, daemon.SnapshotReq{
		AgentID: *agentID, SnapType: *snapType, Paths: *paths,
	}); err != nil {
		return fmt.Errorf("sending snapshot request: %v", err)
	}
	var resp daemon.SnapshotResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("snapshot failed: %v", err)
	}
	pretty.Done(w, "Snapshot taken")
	pretty.Item(w, "ID", resp.SnapshotID[:16])
	pretty.Item(w, "Index", fmt.Sprintf("%d", resp.ChainIndex))
	pretty.Item(w, "Size", fmt.Sprintf("%d bytes", resp.SizeBytes))
	pretty.Item(w, "Hash", fmt.Sprintf("%x...", resp.Hash[:16]))
	fmt.Fprintln(w)
	return nil
}

// --- snapshot diff ---

func cmdSnapshotDiff(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("snapshot diff", flag.ContinueOnError)
	agentID := fs.String("agent", "", "Agent ID")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}
	if *agentID == "" {
		*agentID = fs.Arg(0)
	}
	if *agentID == "" {
		return fmt.Errorf("usage: ans snapshot diff --agent <id>")
	}

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgSnapshotDiff, daemon.SnapshotDiffReq{
		AgentID: *agentID, SnapType: string(snapshot.SnapFileSystem),
	})
	var resp daemon.SnapshotDiffResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("snapshot diff: %v", err)
	}
	if resp.Message != "" {
		fmt.Fprintln(w, resp.Message)
		return nil
	}
	pretty.Subheader(w, "File-level diff")
	if len(resp.Added) > 0 {
		fmt.Fprintf(w, "  %sAdded:%s %d files\n", pretty.Green, pretty.Reset, len(resp.Added))
		for _, f := range resp.Added {
			fmt.Fprintf(w, "    %s+%s %s\n", pretty.Green, pretty.Reset, f)
		}
	}
	if len(resp.Modified) > 0 {
		fmt.Fprintf(w, "  %sModified:%s %d files\n", pretty.Yellow, pretty.Reset, len(resp.Modified))
		for _, f := range resp.Modified {
			fmt.Fprintf(w, "    %s~%s %s\n", pretty.Yellow, pretty.Reset, f)
		}
	}
	if len(resp.Deleted) > 0 {
		fmt.Fprintf(w, "  %sDeleted:%s %d files\n", pretty.Red, pretty.Reset, len(resp.Deleted))
		for _, f := range resp.Deleted {
			fmt.Fprintf(w, "    %s-%s %s\n", pretty.Red, pretty.Reset, f)
		}
	}
	if len(resp.Added)+len(resp.Modified)+len(resp.Deleted) == 0 {
		pretty.Item(w, "Result", "No changes (snapshots are identical)")
	}
	return nil
}

// --- snapshots list ---

func cmdSnapshots(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("snapshots", flag.ContinueOnError)
	agentFilter := fs.String("agent", "", "Filter by agent ID")
	snapType := fs.String("type", "filesystem", "Snapshot type")
	n := fs.Int("n", 20, "Number of snapshots to show")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}
	if *agentFilter == "" {
		if arg := fs.Arg(0); arg != "" {
			agentFilter = &arg
		}
	}
	if *agentFilter == "" {
		return fmt.Errorf("usage: ans snapshots --agent <id> [--type filesystem] [--n 20]")
	}

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgSnapshotList, daemon.SnapshotListReq{
		AgentID: *agentFilter, SnapType: *snapType, Limit: *n,
	}); err != nil {
		return fmt.Errorf("sending snapshot list request: %v", err)
	}
	var resp map[string]interface{}
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("reading snapshot list: %v", err)
	}
	snaps, _ := resp["snapshots"].([]interface{})
	if len(snaps) == 0 {
		pretty.Warn(w, "No snapshots found for agent "+*agentFilter)
		return nil
	}
	pretty.Header(w, fmt.Sprintf("Snapshots for %s", *agentFilter))
	for _, s := range snaps {
		snap, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		sid, _ := snap["snapshot_id"].(string)
		st, _ := snap["snap_type"].(string)
		ci, _ := snap["chain_index"].(float64)
		sz, _ := snap["size_bytes"].(float64)
		ts, _ := snap["timestamp_ns"].(float64)
		tsTime := time.Unix(0, int64(ts))
		sizeStr := fmt.Sprintf("%.1f KB", sz/1024)
		if sz < 1024 {
			sizeStr = fmt.Sprintf("%.0f B", sz)
		}
		idShort := sid
		if len(idShort) > 16 {
			idShort = idShort[:16]
		}
		pretty.Item(w, idShort, fmt.Sprintf("%s  index=%.0f  %s  %s", st, ci, sizeStr, tsTime.Format("15:04:05")))
	}
	pretty.Code(w, "ans time-travel <index> to restore")
	fmt.Fprintln(w)
	return nil
}

// --- time-travel ---

func cmdTimeTravel(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("time-travel", flag.ContinueOnError)
	snapType := fs.String("type", "filesystem", "Snapshot type: filesystem, memory, database")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}
	targetStr := fs.Arg(0)
	if targetStr == "" {
		return fmt.Errorf("usage: ans time-travel <chain-index-or-hash> [--type filesystem]")
	}

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()

	var targetIdx uint64
	if len(targetStr) == 64 && isHex(targetStr) {
		_ = daemon.WriteJSON(conn, daemon.MsgVerify, daemon.VerifyReq{ReceiptID: targetStr})
		var verifyResp daemon.VerifyResp
		if _, err := daemon.ReadJSON(conn, &verifyResp); err != nil {
			return fmt.Errorf("resolving receipt %q: %v", targetStr, err)
		}
		if verifyResp.ChainIndex == 0 && !verifyResp.Valid {
			return fmt.Errorf("receipt %q not found", targetStr)
		}
		targetIdx = verifyResp.ChainIndex
		pretty.Item(w, "Resolved receipt", fmt.Sprintf("%s -> chain index %d", targetStr[:16], targetIdx))
	} else {
		var err error
		targetIdx, err = strconv.ParseUint(targetStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid chain index or receipt hash: %v", err)
		}
	}

	if err := daemon.WriteJSON(conn, daemon.MsgRestore, daemon.RestoreReq{
		TargetIndex: targetIdx, SnapType: *snapType,
	}); err != nil {
		return fmt.Errorf("sending restore request: %v", err)
	}
	var resp daemon.RestoreResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("restore failed: %v", err)
	}
	if resp.Success {
		pretty.Ok(w, fmt.Sprintf("State restored to chain index %d", targetIdx))
	} else {
		pretty.Err(w, "Restore failed: "+resp.Message)
		return fmt.Errorf("restore failed")
	}
	return nil
}

// --- compensate ---

func cmdCompensate(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("compensate", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "Show what would be executed without running")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag error: %v", err)
	}
	targetStr := fs.Arg(0)
	if targetStr == "" {
		return fmt.Errorf("usage: ans compensate <chain-index> [--dry-run]")
	}
	targetIdx, err := strconv.ParseUint(targetStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chain index: %v", err)
	}

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgCompensate, daemon.CompensateReq{
		TargetIndex: targetIdx, DryRun: *dryRun,
	})
	var resp daemon.CompensateResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("compensation failed: %v", err)
	}
	for _, d := range resp.Details {
		pretty.Item(w, "  ", d)
	}
	if resp.Success {
		pretty.Ok(w, fmt.Sprintf("Compensation complete: %d run, %d failed", resp.ActionsRun, resp.ActionsFailed))
	} else {
		pretty.Err(w, fmt.Sprintf("Compensation had %d failures: %s", resp.ActionsFailed, resp.Message))
	}
	return nil
}
