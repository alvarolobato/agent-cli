package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alvarolobato/agent-cli/internal/agent/elasticagent"
	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/output"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var agentType string
	var format string
	var elasticConfigPath string
	var elasticStatusURL string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a pipeline-oriented status report",
		RunE: func(cmd *cobra.Command, args []string) error {
			model, err := statusPipeline(cmd, agentType, elasticConfigPath, elasticStatusURL)
			if err != nil {
				return err
			}

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
	cmd.Flags().StringVar(&elasticConfigPath, "elastic-config", "test/fixtures/elastic-agent.yml", "Path to elastic-agent.yml")
	cmd.Flags().StringVar(&elasticStatusURL, "elastic-url", "http://localhost:6791", "Elastic Agent status API base URL")

	return cmd
}

func statusPipeline(cmd *cobra.Command, agentType, elasticConfigPath, elasticStatusURL string) (*pipeline.Pipeline, error) {
	if agentType == "" {
		return pipeline.ExamplePipeline(), nil
	}
	if agentType != "elastic-agent" {
		return nil, fmt.Errorf("unsupported --agent value %q", agentType)
	}

	cfg, err := config.ParseElasticAgentConfig(elasticConfigPath)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	client := elasticagent.NewClient(elasticStatusURL, httpClient)
	adapter := elasticagent.NewAdapter(cfg, client)
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return adapter.Status(ctx)
}
