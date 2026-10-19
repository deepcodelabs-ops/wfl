package mcp

import (
	"fmt"
	"os"
	"strings"

	"github.com/rishirajsingh/wfl/pkg/types"
)

// WriteImplBlock replaces the content of an @@impl block in the .wfl file atomically.
// It writes to a .tmp file first, then os.Rename to replace the original.
// os.Rename is atomic on POSIX filesystems when src and dst are on the same partition —
// writing tmp to the same directory as the .wfl file guarantees this.
func WriteImplBlock(filePath string, rawLines []string, implBlock types.ValidatedBlock, newContent string) error {
	headerIdx := implBlock.StartLine - 1
	endIdx := implBlock.EndLine - 1

	if headerIdx < 0 || endIdx >= len(rawLines) || headerIdx >= endIdx {
		return fmt.Errorf("impl block line range [%d,%d] is out of bounds for file with %d lines",
			implBlock.StartLine, implBlock.EndLine, len(rawLines))
	}

	// Build replacement: keep header and @@end lines unchanged, replace content between them.
	replacement := []string{rawLines[headerIdx]}
	replacement = append(replacement, strings.Split(newContent, "\n")...)
	replacement = append(replacement, rawLines[endIdx])

	// Splice into a copy of rawLines.
	newLines := make([]string, 0, len(rawLines)-int(endIdx-headerIdx-1)+len(replacement))
	newLines = append(newLines, rawLines[:headerIdx]...)
	newLines = append(newLines, replacement...)
	newLines = append(newLines, rawLines[endIdx+1:]...)

	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename tmp to original: %w", err)
	}
	return nil
}
