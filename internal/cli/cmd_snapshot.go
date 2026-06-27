package cli

import (
	"flag"
	"fmt"
	"strconv"

	"ans/internal/client"
)

func runSnapshot(args []string, c client.Client) {
	if len(args) == 0 {
		Fail("Usage: ans snapshot <take|diff|list> [flags]")
		exitErr(1)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "take":
		runSnapshotTake(rest, c)
	case "diff":
		runSnapshotDiff(rest, c)
	case "list":
		runSnapshotList(c)
	default:
		Fail("unknown snapshot subcommand: " + sub)
		exitErr(1)
	}
}

func runSnapshotTake(args []string, c client.Client) {
	fs := flag.NewFlagSet("snapshot take", flag.ExitOnError)
	agentID := fs.String("agent", "", "agent ID")
	snapType := fs.String("type", "standard", "snapshot type (standard/emergency)")
	paths := fs.String("paths", "", "comma-separated paths")
	fs.Parse(args)

	snap, err := c.SnapshotTake(*agentID, *snapType, *paths)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Snapshot taken")
	fmt.Printf("  Slot:  %d\n", snap.Slot)
	fmt.Printf("  Type:  %s\n", snap.Type)
	fmt.Printf("  Paths: %s\n", Dim(snap.Paths))
}

func runSnapshotDiff(args []string, c client.Client) {
	fs := flag.NewFlagSet("snapshot diff", flag.ExitOnError)
	agentID := fs.String("agent", "", "agent ID")
	fs.Parse(args)

	diff, err := c.SnapshotDiff(*agentID)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	if len(diff.Added) == 0 && len(diff.Removed) == 0 && len(diff.Modified) == 0 {
		fmt.Println(Amber("No changes"))
		return
	}
	fmt.Println(Bold("SNAPSHOT DIFF"))
	fmt.Println(Dim("────────────────────────────────────"))
	for _, f := range diff.Added {
		fmt.Printf("  %s +%s\n", Green("+"), f)
	}
	for _, f := range diff.Removed {
		fmt.Printf("  %s -%s\n", Red("-"), f)
	}
	for _, f := range diff.Modified {
		fmt.Printf("  %s ~%s\n", Amber("~"), f)
	}
}

func runSnapshotList(c client.Client) {
	snaps, err := c.SnapshotList()
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	if len(snaps) == 0 {
		Warn("No snapshots")
		return
	}
	fmt.Println(Bold("SNAPSHOTS"))
	fmt.Println(Dim("────────────────────────────────────"))
	for _, s := range snaps {
		fmt.Printf("  Slot %d | %s | %s\n", s.Slot, s.Timestamp.Format("2006-01-02 15:04:05"), s.Type)
		fmt.Printf("         %s\n", Dim(s.Paths))
	}
}

func runTimeTravel(args []string, c client.Client) {
	if len(args) == 0 {
		Fail("Usage: ans time-travel <index>")
		exitErr(1)
	}
	index, err := strconv.Atoi(args[0])
	if err != nil {
		Fail("invalid index: " + args[0])
		exitErr(1)
	}
	if err := c.TimeTravel(index); err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Restored from snapshot " + args[0])
}

func runSnapshots(args []string, c client.Client) {
	runSnapshotList(c)
}
