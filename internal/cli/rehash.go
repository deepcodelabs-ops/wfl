package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/pkg/types"
	"github.com/spf13/cobra"
)

// RehashCmd returns the 'wfl rehash' command.
func RehashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rehash <file.wfl>",
		Short: "Recompute and update all @@intent hash locks in-place",
		Long: `rehash recomputes the SHA256 hash for every @@intent block and
rewrites the @@end:sha256:... line in-place. Run this after editing
any @@intent block content.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRehash(args[0])
		},
	}
}

func runRehash(filePath string) error {
	wf, err := parser.ParseFile(filePath)
	if err != nil {
		return err
	}

	lines := make([]string, len(wf.RawLines))
	copy(lines, wf.RawLines)

	updated := 0
	for _, b := range wf.Blocks {
		if b.Header.Kind != types.BlockKindIntent {
			continue
		}

		newHash := parser.ComputeBlockHash(b.RawContent)

		// The @@end line is at b.EndLine (1-indexed).
		endIdx := b.EndLine - 1
		if endIdx < 0 || endIdx >= len(lines) {
			return fmt.Errorf("flow %q: @@end line index out of range", b.Header.FlowName)
		}

		oldLine := lines[endIdx]
		newLine := "@@end:sha256:" + newHash

		if oldLine == newLine {
			fmt.Printf("[unchanged] flow:%s v%s\n", b.Header.FlowName, b.Header.Version)
			continue
		}

		lines[endIdx] = newLine
		updated++

		if b.DeclaredHash == "" {
			fmt.Printf("[hashed]    flow:%s v%s  (was: no hash)\n", b.Header.FlowName, b.Header.Version)
		} else {
			fmt.Printf("[updated]   flow:%s v%s\n", b.Header.FlowName, b.Header.Version)
		}
	}

	if updated == 0 {
		fmt.Println("All hashes already up to date.")
		return nil
	}

	content := strings.Join(lines, "\n")
	// Preserve trailing newline if original had one.
	if len(wf.RawLines) > 0 {
		content += "\n"
	}
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	fmt.Printf("\nUpdated %d hash(es) in %s\n", updated, filePath)
	return nil
}
