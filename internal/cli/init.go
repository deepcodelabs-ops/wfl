package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/spf13/cobra"
)

// InitCmd returns the 'wfl init' command.
func InitCmd() *cobra.Command {
	var flowName string

	cmd := &cobra.Command{
		Use:   "init <file.wfl>",
		Short: "Scaffold a new workflow file",
		Long: `init creates a new .wfl file with a scaffolded @@intent block and
a precomputed hash. The target directory is created automatically if
it does not exist.

Use --flow-name to set the flow name explicitly instead of deriving
it from the filename.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(args[0], flowName)
		},
	}

	cmd.Flags().StringVar(&flowName, "flow-name", "", "Flow name (default: derived from filename)")
	return cmd
}

func runInit(filePath, flowNameFlag string) error {
	if !strings.HasSuffix(filePath, ".wfl") {
		return fmt.Errorf("file must have a .wfl extension")
	}
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("file %s already exists", filePath)
	}

	// Auto-create parent directory if it doesn't exist.
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	flowName := flowNameFlag
	if flowName == "" {
		flowName = deriveFlowName(filepath.Base(filePath))
	}

	// Build the intent block content, compute its hash, then compose the full file.
	intentContent := "  # feat: initial " + flowName + " flow\n\n  actor -> service: action()"
	hash := parser.ComputeBlockHash(intentContent)

	content := fmt.Sprintf(
		"@@intent v1.0.0 flow:%s\n%s\n@@end:sha256:%s\n\n@@impl v1.0.0 flow:%s\n  # agent-written after implementation\n@@end\n\n@@changelog flow:%s\n  v1.0.0 feat: initial %s flow\n@@end\n",
		flowName, intentContent, hash,
		flowName,
		flowName, flowName,
	)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("Created %s\n", filePath)
	fmt.Printf("Flow:  %s\n", flowName)
	fmt.Printf("Hash:  %s...\n", hash[:16])
	return nil
}

// deriveFlowName converts a filename like "user_auth.wfl" → "UserAuth".
func deriveFlowName(base string) string {
	name := strings.TrimSuffix(base, ".wfl")
	// Split on underscores, hyphens, and spaces, then PascalCase each segment.
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var sb strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		runes := []rune(p)
		sb.WriteRune(unicode.ToUpper(runes[0]))
		for _, r := range runes[1:] {
			sb.WriteRune(r)
		}
	}
	result := sb.String()
	if result == "" {
		return "Flow"
	}
	return result
}
