package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/spf13/cobra"
)

// AddFlowCmd returns the 'wfl add-flow' command.
func AddFlowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-flow <file.wfl> <FlowName>",
		Short: "Append a new flow skeleton to an existing workflow file",
		Long: `add-flow appends a new @@intent / @@impl / @@changelog block set
to an existing .wfl file. The intent block is scaffolded with a
computed hash so the file validates immediately.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddFlow(args[0], args[1])
		},
	}
}

func runAddFlow(filePath, flowName string) error {
	if !strings.HasSuffix(filePath, ".wfl") {
		return fmt.Errorf("file must have a .wfl extension")
	}

	// File must already exist.
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file %s does not exist — use 'wfl init' to create it first", filePath)
	}

	// Check that the flow name doesn't already exist in the file.
	wf, err := parser.ParseFile(filePath)
	if err != nil {
		return err
	}
	for _, b := range wf.Blocks {
		if b.Header.FlowName == flowName {
			return fmt.Errorf("flow %q already exists in %s", flowName, filePath)
		}
	}

	intentContent := "  # feat: initial " + flowName + " flow\n\n  actor -> service: action()"
	hash := parser.ComputeBlockHash(intentContent)

	skeleton := fmt.Sprintf(
		"\n# === %s ===\n\n@@intent v1.0.0 flow:%s\n%s\n@@end:sha256:%s\n\n@@impl v1.0.0 flow:%s\n  # agent-written after implementation\n@@end\n\n@@changelog flow:%s\n  v1.0.0 feat: initial %s flow\n@@end\n",
		flowName,
		flowName, intentContent, hash,
		flowName,
		flowName, flowName,
	)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	if _, err := f.WriteString(skeleton); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Added flow %q to %s\n", flowName, filePath)
	fmt.Printf("Hash:  %s...\n", hash[:16])
	return nil
}
