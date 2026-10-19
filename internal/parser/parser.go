package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rishirajsingh/wfl/pkg/types"
)

var (
	headerRegex     = regexp.MustCompile(`^@@(intent|impl|changelog)\s+v(\d+\.\d+\.\d+)\s+flow:(\w+)`)
	changelogRegex  = regexp.MustCompile(`^@@changelog\s+flow:(\w+)`)
	endRegex        = regexp.MustCompile(`^@@end(?::sha256:([a-f0-9]{64}))?`)
	annotationRegex = regexp.MustCompile(`^\s+@(context|api|db):\s+(.+)$`)
	arrowRegex      = regexp.MustCompile(`^\s*(\S+)\s+->\s+(\S+)(?::\s+(.*))?$`)
)

type parserState int

const (
	stateIdle    parserState = iota
	stateInBlock
)

// ParseFile reads a .wfl file from disk and parses it into a WflFile.
func ParseFile(filePath string) (*types.WflFile, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}
	return ParseString(string(data), filePath)
}

// ParseString parses .wfl content from a string. virtualPath is used for error messages
// and stored in WflFile.FilePath — it does not need to be a real file.
func ParseString(content, virtualPath string) (*types.WflFile, error) {
	// Normalize line endings before any processing.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	lines := strings.Split(content, "\n")
	// Remove a single trailing empty line that Split produces for files ending in \n.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var (
		blocks      []types.Block
		state       = stateIdle
		currentHdr  types.BlockHeader
		contentBuf  strings.Builder
		blockStart  int
		blockLines  []string // lines accumulated for annotation/arrow scanning
	)

	for i, line := range lines {
		lineNum := i + 1 // 1-indexed

		switch state {
		case stateIdle:
			if m := headerRegex.FindStringSubmatch(line); m != nil {
				state = stateInBlock
				currentHdr = types.BlockHeader{
					Kind:     types.BlockKind(m[1]),
					Version:  m[2],
					FlowName: m[3],
					RawLine:  line,
				}
				contentBuf.Reset()
				blockLines = nil
				blockStart = lineNum
				continue
			}
			// Changelog header without version
			if m := changelogRegex.FindStringSubmatch(line); m != nil {
				state = stateInBlock
				currentHdr = types.BlockHeader{
					Kind:     types.BlockKindChangelog,
					Version:  "0.0.0",
					FlowName: m[1],
					RawLine:  line,
				}
				contentBuf.Reset()
				blockLines = nil
				blockStart = lineNum
				continue
			}

		case stateInBlock:
			if m := endRegex.FindStringSubmatch(line); m != nil {
				rawContent := contentBuf.String()
				declaredHash := ""
				if len(m) > 1 {
					declaredHash = m[1]
				}

				annotations, arrows := parseBlockContent(blockLines, currentHdr.Kind)

				blocks = append(blocks, types.Block{
					Header:       currentHdr,
					RawContent:   rawContent,
					DeclaredHash: declaredHash,
					Annotations:  annotations,
					Arrows:       arrows,
					StartLine:    blockStart,
					EndLine:      lineNum,
				})

				state = stateIdle
				continue
			}

			// Accumulate raw content with "\n" between lines (not after last).
			if contentBuf.Len() > 0 {
				contentBuf.WriteString("\n")
			}
			contentBuf.WriteString(line)
			blockLines = append(blockLines, line)
		}
	}

	if state == stateInBlock {
		return nil, fmt.Errorf("parse %s: unclosed block for flow %q starting at line %d",
			virtualPath, currentHdr.FlowName, blockStart)
	}

	// Wrap raw blocks as ValidatedBlocks (hash/stale fields default false until Validate is called).
	validated := make([]types.ValidatedBlock, len(blocks))
	for i, b := range blocks {
		validated[i] = types.ValidatedBlock{Block: b}
	}

	return &types.WflFile{
		FilePath: virtualPath,
		Blocks:   validated,
		RawLines: lines,
	}, nil
}

// parseBlockContent scans the accumulated lines of a block for annotations and arrows.
// Only intent blocks contain meaningful arrows and annotations.
func parseBlockContent(lines []string, kind types.BlockKind) ([]types.Annotation, []types.FlowArrow) {
	var annotations []types.Annotation
	var arrows []types.FlowArrow

	if kind != types.BlockKindIntent {
		return annotations, arrows
	}

	for i, line := range lines {
		// Skip comment lines for arrow/annotation parsing.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		if m := annotationRegex.FindStringSubmatch(line); m != nil {
			annotations = append(annotations, types.Annotation{
				Type:      types.AnnotationType(m[1]),
				Value:     strings.TrimSpace(m[2]),
				BlockLine: i,
			})
			continue
		}

		if m := arrowRegex.FindStringSubmatch(line); m != nil {
			label := ""
			if len(m) > 3 {
				label = strings.TrimSpace(m[3])
			}
			arrows = append(arrows, types.FlowArrow{
				From:      m[1],
				To:        m[2],
				Label:     label,
				BlockLine: i,
			})
		}
	}

	return annotations, arrows
}
