package cli

import (
	"flag"
	"fmt"
	"strconv"

	"ans/internal/client"
)

func runCompensate(args []string, c client.Client) {
	fs := flag.NewFlagSet("compensate", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "preview without executing")
	fs.Parse(args)

	if len(fs.Args()) == 0 {
		Fail("Usage: ans compensate <index> [--dry-run]")
		exitErr(1)
	}

	index, err := strconv.Atoi(fs.Args()[0])
	if err != nil {
		Fail("invalid index: " + fs.Args()[0])
		exitErr(1)
	}

	if *dryRun {
		plan, err := c.CompensateDryRun(index)
		if err != nil {
			Fail(err.Error())
			exitErr(1)
		}
		fmt.Println(Amber("DRY RUN — Compensation for chain index " + fs.Args()[0]))
		fmt.Println(Dim("──────────────────────────────────────────"))
		for _, step := range plan.Steps {
			if step.HasComp {
				fmt.Printf("  [%d] %s → restore from backup\n", step.ChainIndex, step.ActionType)
				fmt.Printf("       cmd: %s\n", Cyan(step.Command))
			} else {
				fmt.Printf("  [%d] %s → (no compensation registered)\n", step.ChainIndex, step.ActionType)
			}
		}
		fmt.Println()
		OK(fmt.Sprintf("%d would execute, %d skipped (no compensation registered)", plan.WouldRun, plan.Skipped))
		return
	}

	fmt.Printf("Execute compensation for chain index %d? Type 'yes' to confirm: ", index)
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		Warn("Cancelled")
		return
	}

	result, err := c.Compensate(index)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	if result.Failed == 0 {
		OK(fmt.Sprintf("Compensation complete: %d run, %d failed", result.Ran, result.Failed))
	} else {
		Fail(fmt.Sprintf("Compensation failed: %d run, %d failed", result.Ran, result.Failed))
		for _, step := range result.Steps {
			if step.ExitCode != 0 {
				fmt.Printf("  [%d] %s → exit %d\n", step.ChainIndex, step.Command, step.ExitCode)
				fmt.Printf("       stderr: %s\n", Red(step.Stderr))
			}
		}
		exitErr(1)
	}
}
