package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alvarolobato/agent-cli/internal/agent/edot"
	"github.com/alvarolobato/agent-cli/internal/agent/elasticagent"
	"github.com/alvarolobato/agent-cli/internal/agent/otel"
	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/discovery"
	"github.com/alvarolobato/agent-cli/internal/metrics"
	"github.com/alvarolobato/agent-cli/internal/output"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var agentType string
	var format string
	var elasticConfigPath string
	var elasticStatusURL string
	var edotConfigPath string
	var edotZPagesURL string
	var edotMetricsURL string
	var edotHealthURL string
	var otelConfigPath string
	var otelZPagesURL string
	var otelMetricsURL string
	var otelHealthURL string
	var installPath string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a pipeline-oriented status report",
		RunE: func(cmd *cobra.Command, args []string) error {
			model, err := statusPipeline(cmd, statusOptions{
				agentType:         agentType,
				elasticConfig:     elasticConfigPath,
				elasticStatusURL:  elasticStatusURL,
				elasticURLSet:     cmd.Flags().Changed("elastic-url"),
				edotConfig:        edotConfigPath,
				edotZPagesURL:     edotZPagesURL,
				edotZPagesURLSet:  cmd.Flags().Changed("edot-zpages-url"),
				edotMetricsURL:    edotMetricsURL,
				edotMetricsURLSet: cmd.Flags().Changed("edot-metrics-url"),
				edotHealthURL:     edotHealthURL,
				edotHealthURLSet:  cmd.Flags().Changed("edot-health-url"),
				otelConfig:        otelConfigPath,
				otelZPagesURL:     otelZPagesURL,
				otelZPagesURLSet:  cmd.Flags().Changed("otel-zpages-url"),
				otelMetricsURL:    otelMetricsURL,
				otelMetricsURLSet: cmd.Flags().Changed("otel-metrics-url"),
				otelHealthURL:     otelHealthURL,
				otelHealthURLSet:  cmd.Flags().Changed("otel-health-url"),
				path:              installPath,
			})
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
				diagram := output.RenderPipeline(model)
				table := output.RenderTable(model)
				if strings.TrimSpace(diagram) != "" {
					cmd.Println(diagram)
					cmd.Println()
				}
				cmd.Println(table)
				return nil
			default:
				return errors.New("unsupported --format value (use: table|json)")
			}
		},
	}

	cmd.Flags().StringVar(&agentType, "agent", "", "Target a specific agent type")
	cmd.Flags().StringVar(&format, "format", "table", fmt.Sprintf("Output format (%s)", "table|json"))
	cmd.Flags().StringVar(&elasticConfigPath, "elastic-config", "", "Path to elastic-agent.yml (auto-detected when omitted)")
	cmd.Flags().StringVar(&elasticStatusURL, "elastic-url", "http://localhost:6791", "Elastic Agent status API base URL")
	cmd.Flags().StringVar(&edotConfigPath, "edot-config", "", "Path to EDOT/OTel collector YAML config")
	cmd.Flags().StringVar(&edotZPagesURL, "edot-zpages-url", "http://localhost:55679", "EDOT zpages base URL")
	cmd.Flags().StringVar(&edotMetricsURL, "edot-metrics-url", "http://localhost:8888/metrics", "EDOT Prometheus metrics endpoint")
	cmd.Flags().StringVar(&edotHealthURL, "edot-health-url", "http://localhost:13133/", "EDOT health_check endpoint")
	cmd.Flags().StringVar(&otelConfigPath, "otel-config", "", "Path to OTel collector YAML config")
	cmd.Flags().StringVar(&otelZPagesURL, "otel-zpages-url", "http://localhost:55679", "OTel zpages base URL")
	cmd.Flags().StringVar(&otelMetricsURL, "otel-metrics-url", "http://localhost:8888/metrics", "OTel Prometheus metrics endpoint")
	cmd.Flags().StringVar(&otelHealthURL, "otel-health-url", "http://localhost:13133/", "OTel health_check endpoint")
	bindPathFlags(cmd, &installPath)
	registerAgentFlagCompletion(cmd)

	return cmd
}

type statusOptions struct {
	agentType         string
	elasticConfig     string
	elasticOTelConfig string
	elasticStatusURL  string
	elasticURLSet     bool
	edotConfig        string
	edotZPagesURL     string
	edotZPagesURLSet  bool
	edotMetricsURL    string
	edotMetricsURLSet bool
	edotHealthURL     string
	edotHealthURLSet  bool
	otelConfig        string
	otelZPagesURL     string
	otelZPagesURLSet  bool
	otelMetricsURL    string
	otelMetricsURLSet bool
	otelHealthURL     string
	otelHealthURLSet  bool
	path              string
	discoveredMeta    map[string]string
}

var discoverAgents = func(ctx context.Context) ([]discovery.DiscoveredAgent, error) {
	return discovery.NewOrchestrator().DiscoverDetailed(ctx)
}

func statusPipeline(cmd *cobra.Command, options statusOptions) (*pipeline.Pipeline, error) {
	var err error
	options, err = resolveStatusOptionsFromPath(options)
	if err != nil {
		return nil, err
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if options.agentType == "" {
		options, err = autoDetectStatusOptions(ctx, options)
		if err != nil {
			return nil, err
		}
	}

	switch options.agentType {
	case "elastic-agent":
		configPath, err := resolveElasticConfigPath(options.elasticConfig)
		if err != nil {
			return nil, err
		}
		inspectResult, err := config.ParseElasticAgentConfigWithInspect(ctx, configPath, nil)
		if err != nil {
			return nil, err
		}
		cfg := inspectResult.Config
		if otelPath := strings.TrimSpace(options.elasticOTelConfig); otelPath != "" {
			otelCfg, err := config.ParseOTelCollectorConfig(otelPath)
			if err != nil {
				return nil, fmt.Errorf("parse supplemental elastic otel config: %w", err)
			}
			config.MergeFromOTelCollectorConfig(cfg, otelCfg)
		}

		httpClient := &http.Client{Timeout: 5 * time.Second}
		client := elasticagent.NewClient(options.elasticStatusURL, httpClient)
		adapter := elasticagent.NewAdapter(cfg, client)
		pipe, err := adapter.Status(ctx)
		if err != nil {
			return nil, err
		}
		attachDiscoveredMetadata(pipe, options.discoveredMeta)
		if pipe.Metadata == nil {
			pipe.Metadata = map[string]string{}
		}
		pipe.Metadata["config_source"] = inspectResult.Source
		return pipe, nil
	case "edot":
		if strings.TrimSpace(options.edotConfig) == "" {
			return nil, errors.New("edot config not found; pass --edot-config")
		}

		cfg, err := config.ParseOTelCollectorConfig(options.edotConfig)
		if err != nil {
			return nil, err
		}

		httpClient := &http.Client{Timeout: 5 * time.Second}
		zpagesClient := edot.NewZPagesClient(options.edotZPagesURL, httpClient)
		metricsCollector := metrics.NewCollector(httpClient)
		adapter := edot.NewAdapterWithOptions(cfg, zpagesClient, metricsCollector, edot.AdapterOptions{
			PrometheusURL:  options.edotMetricsURL,
			HealthCheckURL: options.edotHealthURL,
		})
		pipe, err := adapter.Status(ctx)
		if err != nil {
			return nil, err
		}
		attachDiscoveredMetadata(pipe, options.discoveredMeta)
		return pipe, nil
	case "otel":
		if strings.TrimSpace(options.otelConfig) == "" {
			return nil, errors.New("otel config not found; pass --otel-config")
		}

		cfg, err := config.ParseOTelCollectorConfig(options.otelConfig)
		if err != nil {
			return nil, err
		}

		httpClient := &http.Client{Timeout: 5 * time.Second}
		zpagesClient := otel.NewZPagesClient(options.otelZPagesURL, httpClient)
		metricsCollector := metrics.NewCollector(httpClient)
		adapter := otel.NewAdapterWithOptions(cfg, zpagesClient, metricsCollector, otel.AdapterOptions{
			PrometheusURL:  options.otelMetricsURL,
			HealthCheckURL: options.otelHealthURL,
		})
		pipe, err := adapter.Status(ctx)
		if err != nil {
			return nil, err
		}
		attachDiscoveredMetadata(pipe, options.discoveredMeta)
		return pipe, nil
	default:
		return nil, fmt.Errorf("unsupported --agent value %q", options.agentType)
	}
}

func autoDetectStatusOptions(ctx context.Context, options statusOptions) (statusOptions, error) {
	discovered, err := discoverAgents(ctx)
	if err != nil {
		return options, err
	}
	if len(discovered) == 0 {
		return options, errors.New("no local agents discovered; pass --agent and explicit config flags")
	}

	best, bestErr := selectPreferredAgent(discovered)
	if bestErr != nil {
		return options, bestErr
	}
	options.agentType = best.AgentType
	options.discoveredMeta = best.Metadata

	switch best.AgentType {
	case "elastic-agent":
		if strings.TrimSpace(options.elasticConfig) == "" {
			options.elasticConfig = best.ConfigPath
		}
		if endpoint, ok := best.Endpoints["status"]; ok && !options.elasticURLSet {
			options.elasticStatusURL = endpoint
		}
	case "edot":
		if strings.TrimSpace(options.edotConfig) == "" {
			options.edotConfig = best.ConfigPath
		}
		if endpoint, ok := best.Endpoints["zpages"]; ok && !options.edotZPagesURLSet {
			options.edotZPagesURL = endpoint
		}
		if endpoint, ok := best.Endpoints["metrics"]; ok && !options.edotMetricsURLSet {
			options.edotMetricsURL = endpoint + "/metrics"
		}
		if endpoint, ok := best.Endpoints["health"]; ok && !options.edotHealthURLSet {
			options.edotHealthURL = endpoint + "/"
		}
	case "otel":
		if strings.TrimSpace(options.otelConfig) == "" {
			options.otelConfig = best.ConfigPath
		}
		if endpoint, ok := best.Endpoints["zpages"]; ok && !options.otelZPagesURLSet {
			options.otelZPagesURL = endpoint
		}
		if endpoint, ok := best.Endpoints["metrics"]; ok && !options.otelMetricsURLSet {
			options.otelMetricsURL = endpoint + "/metrics"
		}
		if endpoint, ok := best.Endpoints["health"]; ok && !options.otelHealthURLSet {
			options.otelHealthURL = endpoint + "/"
		}
	}

	return options, nil
}

func resolveStatusOptionsFromPath(options statusOptions) (statusOptions, error) {
	path := strings.TrimSpace(options.path)
	if path == "" {
		return options, nil
	}

	discovered, err := discovery.DiscoverAgentAtPath(path)
	if err != nil {
		return options, err
	}

	if options.agentType != "" && options.agentType != discovered.Type {
		return options, fmt.Errorf("requested --agent %q but %q looks like %q", options.agentType, path, discovered.Type)
	}
	options.agentType = discovered.Type
	options.discoveredMeta = discovered.Metadata

	switch discovered.Type {
	case "elastic-agent":
		if strings.TrimSpace(options.elasticConfig) == "" {
			configPath := firstConfigPathByBaseName(discovered.ConfigPaths, "elastic-agent.yml", "elastic-agent.yaml")
			if configPath == "" {
				return options, fmt.Errorf("elastic agent config not found under %q", path)
			}
			options.elasticConfig = configPath
		}
		if strings.TrimSpace(options.elasticOTelConfig) == "" {
			options.elasticOTelConfig = firstConfigPathByBaseName(discovered.ConfigPaths, "otel.yml", "otel.yaml")
		}
	case "edot":
		if strings.TrimSpace(options.edotConfig) == "" {
			options.edotConfig = firstCollectorConfigPath(discovered.ConfigPaths)
		}
		if options.edotConfig == "" {
			return options, fmt.Errorf("edot config not found under %q", path)
		}
	case "otel":
		if strings.TrimSpace(options.otelConfig) == "" {
			options.otelConfig = firstCollectorConfigPath(discovered.ConfigPaths)
		}
		if options.otelConfig == "" {
			return options, fmt.Errorf("otel config not found under %q", path)
		}
	default:
		return options, fmt.Errorf("unsupported discovered agent type %q", discovered.Type)
	}

	return options, nil
}

func selectPreferredAgent(agents []discovery.DiscoveredAgent) (discovery.DiscoveredAgent, error) {
	best := agents[0]
	bestScore := discoveryPriority(best.AgentType)
	tied := []discovery.DiscoveredAgent{best}
	for i := 1; i < len(agents); i++ {
		score := discoveryPriority(agents[i].AgentType)
		if score < bestScore {
			best = agents[i]
			bestScore = score
			tied = []discovery.DiscoveredAgent{agents[i]}
			continue
		}
		if score == bestScore {
			tied = append(tied, agents[i])
		}
	}
	if len(tied) > 1 {
		typeLabel := tied[0].AgentType
		if strings.TrimSpace(typeLabel) == "" {
			typeLabel = "unknown"
		}
		return discovery.DiscoveredAgent{}, fmt.Errorf("multiple %s agents discovered; pass --agent and explicit config flags", typeLabel)
	}
	return best, nil
}

func discoveryPriority(agentType string) int {
	switch agentType {
	case "elastic-agent":
		return 0
	case "edot":
		return 1
	case "otel":
		return 2
	default:
		return 3
	}
}

func attachDiscoveredMetadata(pipe *pipeline.Pipeline, metadata map[string]string) {
	if pipe == nil || len(metadata) == 0 {
		return
	}
	if pipe.Metadata == nil {
		pipe.Metadata = map[string]string{}
	}
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		pipe.Metadata[key] = value
	}
}

func firstCollectorConfigPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	// Prefer conventional collector config names before falling back.
	if config := firstConfigPathByBaseName(paths, "config.yaml", "config.yml", "otel.yml", "otel.yaml"); config != "" {
		return config
	}
	for _, path := range paths {
		lower := strings.ToLower(filepath.Base(path))
		if strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml") {
			return path
		}
	}
	return ""
}

func firstConfigPathByBaseName(paths []string, names ...string) string {
	if len(paths) == 0 || len(names) == 0 {
		return ""
	}
	lookup := make(map[string]struct{}, len(names))
	for _, name := range names {
		lookup[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}
	for _, path := range paths {
		base := strings.ToLower(filepath.Base(path))
		if _, ok := lookup[base]; ok {
			return path
		}
	}
	return ""
}

func resolveElasticConfigPath(explicitPath string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}

	for _, candidate := range defaultElasticConfigPaths() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", errors.New("elastic agent config not found; pass --elastic-config")
}

func defaultElasticConfigPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{"/Library/Elastic/Agent/elastic-agent.yml"}
	case "windows":
		return []string{`C:\Program Files\Elastic\Agent\elastic-agent.yml`}
	default:
		return []string{"/opt/Elastic/Agent/elastic-agent.yml"}
	}
}
