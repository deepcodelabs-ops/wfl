package cli

import (
	"fmt"
	"os"
	"path/filepath"

	wflcontext "github.com/rishirajsingh/wfl/internal/context"
	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/pkg/types"
	"github.com/spf13/cobra"
)

// CheckContextCmd returns the 'wfl check-context' command.
func CheckContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-context <file.wfl>",
		Short: "Verify all @context annotation paths exist and point to valid lines",
		Long: `check-context resolves every @context annotation in all intent blocks,
reports which files exist and which are missing, and warns when a
referenced line number exceeds the file's length (stale line number).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckContext(args[0])
		},
	}
}

func runCheckContext(filePath string) error {
	wf, err := parser.ParseFile(filePath)
	if err != nil {
		return err
	}

	workflowDir := filepath.Dir(filePath)
	missingCount := 0
	staleCount := 0
	okCount := 0

	for _, b := range wf.Blocks {
		if b.Header.Kind != types.BlockKindIntent {
			continue
		}

		contextAnns := []types.Annotation{}
		for _, ann := range b.Annotations {
			if ann.Type == types.AnnotationContext {
				contextAnns = append(contextAnns, ann)
			}
		}

		if len(contextAnns) == 0 {
			continue
		}

		fmt.Printf("flow:%s\n", b.Header.FlowName)

		for _, ann := range contextAnns {
			rc := wflcontext.ResolveContextAnnotation(ann, workflowDir)
			if !rc.Exists {
				fmt.Fprintf(os.Stderr, "  [MISSING] %s\n", ann.Value)
				fmt.Fprintf(os.Stderr, "            resolved to: %s\n", rc.FilePath)
				missingCount++
				continue
			}

			// Check if line number is within bounds.
			lineCount := countLines(rc.FilePath)
			if rc.LineNumber > lineCount {
				fmt.Fprintf(os.Stderr, "  [STALE]   %s\n", ann.Value)
				fmt.Fprintf(os.Stderr, "            line %d exceeds file length (%d lines)\n", rc.LineNumber, lineCount)
				staleCount++
				continue
			}

			fmt.Printf("  [OK]      %s\n", ann.Value)
			okCount++
		}
	}

	fmt.Printf("\n%d OK, %d missing, %d stale\n", okCount, missingCount, staleCount)

	if missingCount > 0 || staleCount > 0 {
		return fmt.Errorf("check-context failed: %d missing, %d stale annotation(s)", missingCount, staleCount)
	}
	return nil
}

func countLines(filePath string) int {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}
	count := 1
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	return count
}
