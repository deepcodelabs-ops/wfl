package cli

import (
	"fmt"

	"github.com/rishirajsingh/wfl/internal/parser"
	"github.com/rishirajsingh/wfl/internal/renderer"
	"github.com/rishirajsingh/wfl/internal/validator"
	"github.com/spf13/cobra"
)

// RenderCmd returns the 'wfl render' command.
func RenderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "render <file.wfl>",
		Short: "Render all flows as Mermaid diagrams to a .md file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(args[0])
		},
	}
}

func runRender(filePath string) error {
	wf, err := parser.ParseFile(filePath)
	if err != nil {
		return err
	}

	result := validator.Validate(wf)
	if result.HasHardErrors {
		return fmt.Errorf("refusing to render: hash verification failed — run 'wfl validate' for details")
	}

	outPath, err := renderer.RenderFile(wf)
	if err != nil {
		return err
	}

	fmt.Printf("Rendered to %s\n", outPath)
	return nil
}
