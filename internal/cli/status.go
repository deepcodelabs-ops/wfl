package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/internal/validator"
	"github.com/rishirajsingh/wfl/pkg/types"
	"github.com/spf13/cobra"
)

// StatusCmd returns the 'wfl status' command.
func StatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <file.wfl>",
		Short: "Show drift status for all flows in a workflow file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(args[0])
		},
	}
}

func runStatus(filePath string) error {
	wf, err := parser.ParseFile(filePath)
	if err != nil {
		return err
	}

	result := validator.Validate(wf)
	statuses := validator.ListAllStatuses(wf)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FLOW\tINTENT\tIMPL\tSTATUS")
	fmt.Fprintln(w, "----\t------\t----\t------")

	hasProblems := false
	for _, s := range statuses {
		implVer := s.ImplVersion
		if implVer == "" {
			implVer = "-"
		}
		status := string(s.Drift)
		if s.Drift == types.DriftStale || s.Drift == types.DriftHashMismatch {
			hasProblems = true
		}
		fmt.Fprintf(w, "%s\tv%s\t%s\t%s\n", s.FlowName, s.IntentVersion, implVer, status)
	}
	w.Flush()

	if result.HasHardErrors {
		fmt.Fprintln(os.Stderr, "\nHard errors:")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  [%s] %s\n", e.Code, e.Message)
		}
		return fmt.Errorf("status check failed with hard errors")
	}

	if hasProblems {
		return fmt.Errorf("one or more flows are stale or have hash mismatches")
	}

	return nil
}
