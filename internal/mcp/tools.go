package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	wflcontext "github.com/rishirajsingh/wfl/internal/context"
	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/internal/validator"
	"github.com/rishirajsingh/wfl/pkg/types"
)

// flowInput is shared by all single-flow read tools.
type flowInput struct {
	FlowName         string `json:"flow_name"          jsonschema:"name of the flow"`
	WorkflowFilePath string `json:"workflow_file_path" jsonschema:"path to the .wfl file"`
}

// writeImplInput extends flowInput with the new impl content.
type writeImplInput struct {
	FlowName         string `json:"flow_name"          jsonschema:"name of the flow"`
	WorkflowFilePath string `json:"workflow_file_path" jsonschema:"path to the .wfl file"`
	Content          string `json:"content"            jsonschema:"content to write into the @@impl block"`
}

// listFlowsInput only needs the file path.
type listFlowsInput struct {
	WorkflowFilePath string `json:"workflow_file_path" jsonschema:"path to the .wfl file"`
}

// textResult wraps a JSON-serializable value as a TextContent response.
// Returns the three values required by ToolHandlerFor.
func textResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, nil, nil
}

// errorResult returns a tool-level error (not a protocol error).
// Returns the three values required by ToolHandlerFor.
func errorResult(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, nil, nil
}

// loadAndValidate is the common parse+validate cycle used by all tools.
func loadAndValidate(filePath string) (*types.WflFile, *types.ValidationResult, error) {
	wf, err := parser.ParseFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", filePath, err)
	}
	result := validator.Validate(wf)
	return wf, &result, nil
}

// RegisterTools adds all 8 wfl tools to the server.
func RegisterTools(s *mcp.Server, defaultFilePath string) {
	// workflow.getIntent
	mcp.AddTool(s, &mcp.Tool{
		Name:        "workflow.getIntent",
		Description: "Returns the @@intent block content for a named flow. Fails hard if the hash has been tampered.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in flowInput) (*mcp.CallToolResult, any, error) {
		wf, vr, err := loadAndValidate(in.WorkflowFilePath)
		if err != nil {
			return errorResult(err.Error())
		}
		if vr.HasHardErrors {
			msg := "hash verification failed:"
			for _, e := range vr.Errors {
				msg += "\n  " + e.Message
			}
			return errorResult(msg)
		}
		for _, b := range wf.Blocks {
			if b.Header.Kind == types.BlockKindIntent && b.Header.FlowName == in.FlowName {
				return textResult(map[string]any{
					"flowName":     b.Header.FlowName,
					"version":      b.Header.Version,
					"hashVerified": b.HashVerified,
					"content":      b.RawContent,
				})
			}
		}
		return errorResult(fmt.Sprintf("intent block for flow %q not found", in.FlowName))
	})

	// workflow.getImpl
	mcp.AddTool(s, &mcp.Tool{
		Name:        "workflow.getImpl",
		Description: "Returns the @@impl block content for a named flow.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in flowInput) (*mcp.CallToolResult, any, error) {
		wf, _, err := loadAndValidate(in.WorkflowFilePath)
		if err != nil {
			return errorResult(err.Error())
		}
		for _, b := range wf.Blocks {
			if b.Header.Kind == types.BlockKindImpl && b.Header.FlowName == in.FlowName {
				return textResult(map[string]any{
					"flowName": b.Header.FlowName,
					"version":  b.Header.Version,
					"content":  b.RawContent,
				})
			}
		}
		return textResult(map[string]any{
			"flowName":    in.FlowName,
			"implMissing": true,
		})
	})

	// workflow.getContext
	mcp.AddTool(s, &mcp.Tool{
		Name:        "workflow.getContext",
		Description: "Resolves @context annotations in the intent block to actual file excerpts. Returns @api and @db as metadata.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in flowInput) (*mcp.CallToolResult, any, error) {
		wf, vr, err := loadAndValidate(in.WorkflowFilePath)
		if err != nil {
			return errorResult(err.Error())
		}
		if vr.HasHardErrors {
			return errorResult("hash verification failed — cannot resolve context from tampered file")
		}
		for _, b := range wf.Blocks {
			if b.Header.Kind == types.BlockKindIntent && b.Header.FlowName == in.FlowName {
				workflowDir := filepath.Dir(in.WorkflowFilePath)
				resolved, metadata := wflcontext.ResolveAll(b, workflowDir)

				contexts := make([]map[string]any, 0, len(resolved))
				for _, rc := range resolved {
					contexts = append(contexts, map[string]any{
						"annotation": rc.Annotation.Value,
						"filePath":   rc.FilePath,
						"lineNumber": rc.LineNumber,
						"excerpt":    rc.Excerpt,
						"exists":     rc.Exists,
					})
				}
				return textResult(map[string]any{
					"flowName": in.FlowName,
					"contexts": contexts,
					"metadata": metadata,
				})
			}
		}
		return errorResult(fmt.Sprintf("intent block for flow %q not found", in.FlowName))
	})

	// workflow.getSubtasks
	mcp.AddTool(s, &mcp.Tool{
		Name:        "workflow.getSubtasks",
		Description: "Decomposes the intent block arrows into parallelizable subtask groups.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in flowInput) (*mcp.CallToolResult, any, error) {
		wf, _, err := loadAndValidate(in.WorkflowFilePath)
		if err != nil {
			return errorResult(err.Error())
		}
		for _, b := range wf.Blocks {
			if b.Header.Kind == types.BlockKindIntent && b.Header.FlowName == in.FlowName {
				subtasks := decomposeArrows(in.FlowName, b.Arrows)
				return textResult(map[string]any{
					"flowName": in.FlowName,
					"subtasks": subtasks,
				})
			}
		}
		return errorResult(fmt.Sprintf("intent block for flow %q not found", in.FlowName))
	})

	// workflow.getDependencies
	mcp.AddTool(s, &mcp.Tool{
		Name:        "workflow.getDependencies",
		Description: "Returns other flows that this flow depends on, inferred from cross-flow references and '# depends:' comments.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in flowInput) (*mcp.CallToolResult, any, error) {
		wf, _, err := loadAndValidate(in.WorkflowFilePath)
		if err != nil {
			return errorResult(err.Error())
		}
		deps := extractDependencies(wf, in.FlowName)
		return textResult(map[string]any{
			"flowName":     in.FlowName,
			"dependencies": deps,
		})
	})

	// workflow.writeImpl
	mcp.AddTool(s, &mcp.Tool{
		Name:        "workflow.writeImpl",
		Description: "Writes content to the @@impl block for a named flow. Refuses if the intent has breaking changes (isStale) or if the target is an @@intent block.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in writeImplInput) (*mcp.CallToolResult, any, error) {
		wf, _, err := loadAndValidate(in.WorkflowFilePath)
		if err != nil {
			return errorResult(err.Error())
		}

		// Guard: never write to an intent block.
		for _, b := range wf.Blocks {
			if b.Header.Kind == types.BlockKindIntent && b.Header.FlowName == in.FlowName {
				return errorResult("WRITE_REFUSED: agents may not write to @@intent blocks")
			}
		}

		// Guard: refuse if stale.
		status, err := validator.GetFlowStatus(wf, in.FlowName)
		if err != nil {
			return errorResult(fmt.Sprintf("flow %q not found", in.FlowName))
		}
		if status.IsStale {
			return errorResult("STALE_INTENT: the intent block has a breaking major version bump. Developer must bump the @@impl version to acknowledge before agent can proceed.")
		}

		// Find the impl block.
		var implBlock *types.ValidatedBlock
		for i := range wf.Blocks {
			if wf.Blocks[i].Header.Kind == types.BlockKindImpl && wf.Blocks[i].Header.FlowName == in.FlowName {
				implBlock = &wf.Blocks[i]
				break
			}
		}
		if implBlock == nil {
			return errorResult(fmt.Sprintf("impl block for flow %q not found", in.FlowName))
		}

		if err := WriteImplBlock(in.WorkflowFilePath, wf.RawLines, *implBlock, in.Content); err != nil {
			return errorResult(fmt.Sprintf("write failed: %v", err))
		}

		return textResult(map[string]any{
			"success":  true,
			"flowName": in.FlowName,
			"message":  "@@impl block updated successfully",
		})
	})

	// workflow.getStatus
	mcp.AddTool(s, &mcp.Tool{
		Name:        "workflow.getStatus",
		Description: "Returns the drift status for a named flow (version, hash, stale state).",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in flowInput) (*mcp.CallToolResult, any, error) {
		wf, _, err := loadAndValidate(in.WorkflowFilePath)
		if err != nil {
			return errorResult(err.Error())
		}
		status, err := validator.GetFlowStatus(wf, in.FlowName)
		if err != nil {
			return errorResult(err.Error())
		}
		return textResult(status)
	})

	// workflow.listFlows
	mcp.AddTool(s, &mcp.Tool{
		Name:        "workflow.listFlows",
		Description: "Lists all flows in a .wfl file with their version and drift status.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in listFlowsInput) (*mcp.CallToolResult, any, error) {
		wf, _, err := loadAndValidate(in.WorkflowFilePath)
		if err != nil {
			return errorResult(err.Error())
		}
		statuses := validator.ListAllStatuses(wf)
		return textResult(map[string]any{
			"workflowFile": in.WorkflowFilePath,
			"flows":        statuses,
		})
	})
}

// decomposeArrows converts flow arrows into ordered subtask groups.
// Consecutive arrows with the same `From` participant are grouped together (sequential).
// Arrows with different `From` participants at the same depth can be parallelized.
func decomposeArrows(flowName string, arrows []types.FlowArrow) []types.Subtask {
	subtasks := make([]types.Subtask, 0, len(arrows))
	for i, a := range arrows {
		label := a.Label
		if label == "" {
			label = fmt.Sprintf("%s -> %s", a.From, a.To)
		}
		subtasks = append(subtasks, types.Subtask{
			ID:          fmt.Sprintf("%s-%d", flowName, i+1),
			Description: fmt.Sprintf("%s -> %s: %s", a.From, a.To, label),
			From:        a.From,
			To:          a.To,
			Label:       label,
			Order:       i + 1,
		})
	}
	return subtasks
}

// extractDependencies scans a WflFile for flows that the given flow depends on.
// It looks for "# depends: OtherFlow" comments in the intent block's raw content.
func extractDependencies(wf *types.WflFile, flowName string) []string {
	var deps []string
	seen := map[string]bool{}

	for _, b := range wf.Blocks {
		if b.Header.Kind != types.BlockKindIntent || b.Header.FlowName != flowName {
			continue
		}
		for _, line := range splitLines(b.RawContent) {
			var dep string
			if _, err := fmt.Sscanf(trimLeft(line)+"", "# depends: %s", &dep); err == nil && dep != "" {
				if !seen[dep] {
					deps = append(deps, dep)
					seen[dep] = true
				}
			}
		}
	}
	return deps
}

func splitLines(s string) []string {
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func trimLeft(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[i:]
		}
	}
	return ""
}
