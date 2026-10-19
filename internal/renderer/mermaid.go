package renderer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rishirajsingh/wfl/pkg/types"
)

// RenderFlowToMermaid converts a single intent ValidatedBlock to a Mermaid
// sequenceDiagram markdown section.
func RenderFlowToMermaid(block types.ValidatedBlock) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "## Flow: %s (v%s)\n\n", block.Header.FlowName, block.Header.Version)
	sb.WriteString("```mermaid\nsequenceDiagram\n")

	// Collect comment lines from rawContent for Note rendering.
	comments := extractComments(block.Block.RawContent)
	for _, c := range comments {
		fmt.Fprintf(&sb, "    Note over all: %s\n", c)
	}

	// Build a map: blockLine → @context values, so each annotation is attached
	// to the arrow whose blockLine is closest and before it.
	contextByLine := map[int][]string{}
	for _, ann := range block.Annotations {
		if ann.Type == types.AnnotationContext {
			contextByLine[ann.BlockLine] = append(contextByLine[ann.BlockLine], ann.Value)
		}
	}

	// Build sorted arrow line list so we can find the "next arrow line" for each arrow.
	arrowLines := make([]int, len(block.Arrows))
	for i, a := range block.Arrows {
		arrowLines[i] = a.BlockLine
	}

	for i, arrow := range block.Arrows {
		label := arrow.Label
		if label == "" {
			label = arrow.To
		}
		fmt.Fprintf(&sb, "    %s->>%s: %s\n", arrow.From, arrow.To, label)

		// Attach @context annotations that appear on lines after this arrow
		// and before the next arrow (or end of block).
		nextArrowLine := -1
		if i+1 < len(block.Arrows) {
			nextArrowLine = arrowLines[i+1]
		}

		for line, vals := range contextByLine {
			if line > arrow.BlockLine && (nextArrowLine < 0 || line < nextArrowLine) {
				for _, v := range vals {
					fmt.Fprintf(&sb, "    Note right of %s: @context %s\n", arrow.To, v)
				}
				delete(contextByLine, line)
			}
		}
	}

	sb.WriteString("```\n")
	return sb.String()
}

// RenderFile renders all intent blocks in a WflFile to a .md file alongside the .wfl file.
// Flows are separated by a visual divider. Returns the output path.
func RenderFile(wf *types.WflFile) (string, error) {
	outPath := replaceExt(wf.FilePath, ".md")

	var sections []string
	for _, b := range wf.Blocks {
		if b.Header.Kind == types.BlockKindIntent {
			sections = append(sections, RenderFlowToMermaid(b))
		}
	}

	if len(sections) == 0 {
		return "", fmt.Errorf("no intent blocks found in %s", wf.FilePath)
	}

	// Separate flows with a clear visual divider.
	content := strings.Join(sections, "\n---\n\n")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", outPath, err)
	}
	return outPath, nil
}

// extractComments returns trimmed comment text from lines starting with #.
func extractComments(rawContent string) []string {
	var comments []string
	for _, line := range strings.Split(rawContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			if text != "" {
				comments = append(comments, text)
			}
		}
	}
	return comments
}

// replaceExt replaces the file extension of path with newExt.
func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)] + newExt
}
