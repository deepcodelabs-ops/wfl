package cli

import (
	"fmt"
	"os"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/internal/validator"
	"github.com/spf13/cobra"
)

// ValidateCmd returns the 'wfl validate' command.
func ValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <file.wfl>",
		Short: "Verify all @@intent hash locks in a workflow file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(args[0])
		},
	}
}

func runValidate(filePath string) error {
	wf, err := parser.ParseFile(filePath)
	if err != nil {
		return err
	}

	result := validator.Validate(wf)

	intentCount := 0
	for _, b := range result.Blocks {
		if b.Header.Kind == "intent" {
			intentCount++
			status := "OK"
			if !b.HashVerified {
				status = "FAIL"
			}
			fmt.Printf("[%s] flow:%s v%s\n", status, b.Header.FlowName, b.Header.Version)
		}
	}

	if result.HasHardErrors {
		fmt.Fprintln(os.Stderr, "\nErrors:")
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  [%s] %s\n", e.Code, e.Message)
		}
		return fmt.Errorf("validation failed: %d error(s)", len(result.Errors))
	}

	fmt.Printf("\nAll %d intent block(s) verified OK.\n", intentCount)
	return nil
}
