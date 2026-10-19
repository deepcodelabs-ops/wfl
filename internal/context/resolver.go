package context

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rishirajsingh/wfl/pkg/types"
)

const contextWindow = 20

// ResolveContextAnnotation resolves a @context annotation by reading the referenced file.
// workflowDir is the directory of the .wfl file — relative paths in @context are resolved from there.
func ResolveContextAnnotation(ann types.Annotation, workflowDir string) types.ResolvedContext {
	// Split on the LAST ":" to separate path from line number.
	// Handles paths with colons (e.g. Windows drive letters).
	idx := strings.LastIndex(ann.Value, ":")
	if idx < 0 {
		return types.ResolvedContext{Annotation: ann, Exists: false}
	}

	relPath := ann.Value[:idx]
	lineNum, err := strconv.Atoi(strings.TrimSpace(ann.Value[idx+1:]))
	if err != nil {
		return types.ResolvedContext{Annotation: ann, Exists: false}
	}

	absPath := filepath.Join(workflowDir, relPath)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return types.ResolvedContext{
			Annotation: ann,
			FilePath:   absPath,
			LineNumber: lineNum,
			Exists:     false,
		}
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	start := lineNum - 1 - contextWindow
	if start < 0 {
		start = 0
	}
	end := lineNum - 1 + contextWindow + 1
	if end > len(lines) {
		end = len(lines)
	}

	return types.ResolvedContext{
		Annotation: ann,
		FilePath:   absPath,
		LineNumber: lineNum,
		Excerpt:    lines[start:end],
		Exists:     true,
	}
}

// ResolveAll resolves all @context annotations for a block.
// @api and @db annotations are returned as pass-through metadata keyed by type.
func ResolveAll(block types.ValidatedBlock, workflowDir string) ([]types.ResolvedContext, map[string][]string) {
	var resolved []types.ResolvedContext
	metadata := map[string][]string{}

	for _, ann := range block.Annotations {
		switch ann.Type {
		case types.AnnotationContext:
			resolved = append(resolved, ResolveContextAnnotation(ann, workflowDir))
		case types.AnnotationAPI:
			metadata["api"] = append(metadata["api"], ann.Value)
		case types.AnnotationDB:
			metadata["db"] = append(metadata["db"], ann.Value)
		}
	}

	return resolved, metadata
}
