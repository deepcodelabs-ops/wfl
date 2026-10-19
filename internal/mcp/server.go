package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StartStdioServer starts the MCP server using the stdio transport.
// This is the standard mode used by Claude Desktop, Claude Code, and other
// MCP clients that launch the server as a subprocess.
// It blocks until the client disconnects or ctx is cancelled.
func StartStdioServer(ctx context.Context, workflowFilePath string) error {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "wfl",
		Version: "0.1.0",
	}, &mcp.ServerOptions{
		Instructions: "wfl workflow file system server. Use workflow.listFlows to discover flows, then workflow.getIntent/getContext/getSubtasks to read intent, and workflow.writeImpl to record implementation notes.",
	})

	RegisterTools(s, workflowFilePath)

	return s.Run(ctx, &mcp.StdioTransport{})
}
