package cli

import (
	"errors"
	"fmt"

	"github.com/alvarolobato/agent-cli/internal/output"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var agentType string
	var format string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a pipeline-oriented status report",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = agentType // Reserved for future agent filtering.

			model := pipeline.ExamplePipeline()
			switch format {
			case "json":
				out, err := output.RenderJSON(model)
				if err != nil {
					return err
				}
				cmd.Println(out)
				return nil
			case "table":
				cmd.Println(output.RenderTable(model))
				return nil
			default:
				return errors.New("unsupported --format value (use: table|json)")
			}
		},
	}

	cmd.Flags().StringVar(&agentType, "agent", "", "Target a specific agent type")
	cmd.Flags().StringVar(&format, "format", "table", fmt.Sprintf("Output format (%s)", "table|json"))

	return cmd
}
