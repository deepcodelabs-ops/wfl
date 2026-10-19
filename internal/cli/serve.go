package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	wflmcp "github.com/rishirajsingh/wfl/internal/mcp"
	"github.com/spf13/cobra"
)

// ServeCmd returns the 'wfl serve' command.
func ServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve <file.wfl>",
		Short: "Start MCP server (stdio mode) pointed at a workflow file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(args[0])
		},
	}
}

func runServe(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("file not found: %s", absPath)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fmt.Fprintf(os.Stderr, "wfl MCP server starting (stdio mode)\nworkflow: %s\n", absPath)

	return wflmcp.StartStdioServer(ctx, absPath)
}
