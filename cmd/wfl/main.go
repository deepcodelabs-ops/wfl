package main

import (
	"fmt"
	"os"

	"github.com/rishirajsingh/wfl/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "wfl",
		Short: "Workflow file system — developer intent + agent implementation",
		Long: `wfl manages structured .wfl workflow files that describe feature intent.

Developers write intent blocks; agents implement and write back to impl blocks.
The MCP server exposes chunked context to agents without reading the entire file.`,
		SilenceUsage: true,
	}

	root.AddCommand(cli.InitCmd())
	root.AddCommand(cli.AddFlowCmd())
	root.AddCommand(cli.RehashCmd())
	root.AddCommand(cli.CheckContextCmd())
	root.AddCommand(cli.StatusCmd())
	root.AddCommand(cli.RenderCmd())
	root.AddCommand(cli.ValidateCmd())
	root.AddCommand(cli.ServeCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
