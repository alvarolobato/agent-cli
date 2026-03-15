package edot

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/alvarolobato/agent-cli/internal/agent/otel"
	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/metrics"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

const (
	agentID   = "edot-collector"
	agentType = "edot"
)

type zpagesAPI interface {
	GetPipelineTopology(ctx context.Context) (*PipelineTopology, error)
}

type otelMetricsAPI interface {
	CollectOTelPrometheus(ctx context.Context, endpoint string) (*metrics.OTelSnapshot, error)
}

// Adapter maps EDOT config and runtime APIs into the shared OTel pipeline model.
type Adapter struct {
	base *otel.Adapter
}

// AdapterOptions overrides default runtime endpoints.
type AdapterOptions struct {
	PrometheusURL  string
	HealthCheckURL string
}

// NewAdapter creates an EDOT adapter with defaults for local collector endpoints.
func NewAdapter(cfg *config.OTelCollectorConfig, zpagesClient *ZPagesClient, metricsCollector *metrics.Collector) *Adapter {
	return NewAdapterWithOptions(cfg, zpagesClient, metricsCollector, AdapterOptions{})
}

// NewAdapterWithOptions creates an EDOT adapter with configurable endpoints.
func NewAdapterWithOptions(
	cfg *config.OTelCollectorConfig,
	zpagesClient *ZPagesClient,
	metricsCollector *metrics.Collector,
	options AdapterOptions,
) *Adapter {
	return newAdapterWithDeps(cfg, zpagesClient, metricsCollector, nil, options.PrometheusURL, options.HealthCheckURL)
}

func newAdapterWithDeps(
	cfg *config.OTelCollectorConfig,
	zpagesClient zpagesAPI,
	metricsCollector otelMetricsAPI,
	httpClient *http.Client,
	prometheusURL string,
	healthCheckURL string,
) *Adapter {
	base := otel.NewAdapterWithOptions(cfg, newZPagesBridge(zpagesClient), metricsCollector, otel.AdapterOptions{
		PrometheusURL:     prometheusURL,
		HealthCheckURL:    healthCheckURL,
		HTTPClient:        httpClient,
		AgentID:           agentID,
		AgentType:         agentType,
		PipelineName:      "edot",
		NodeLabeler:       elasticNodeLabel,
		MetadataAugmenter: elasticMetadata,
	})
	return &Adapter{base: base}
}

// ID returns the adapter identifier.
func (a *Adapter) ID() string {
	return a.base.ID()
}

// Type returns the adapter type string.
func (a *Adapter) Type() string {
	return a.base.Type()
}

// Status builds a pipeline graph from config wiring and runtime status/metrics.
func (a *Adapter) Status(ctx context.Context) (*pipeline.Pipeline, error) {
	return a.base.Status(ctx)
}

// Health returns top-level health using the health_check endpoint.
func (a *Adapter) Health(ctx context.Context) (pipeline.HealthStatus, error) {
	return a.base.Health(ctx)
}

func elasticNodeLabel(componentName, _, _ string, descriptor otel.ComponentDescriptor) string {
	label := strings.TrimSpace(descriptor.Name)
	if label == "" {
		label = componentName
	}
	if isElasticComponent(componentName) {
		return label + " [Elastic]"
	}
	return label
}

func isElasticComponent(name string) bool {
	value := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(value, "elastic") || strings.Contains(value, "ecsformatprocessor")
}

func elasticMetadata(nodes []pipeline.Node) map[string]string {
	elasticNodes := make([]string, 0)
	for _, node := range nodes {
		if isElasticComponent(node.ID) {
			elasticNodes = append(elasticNodes, node.ID)
		}
	}
	if len(elasticNodes) == 0 {
		return nil
	}
	sort.Strings(elasticNodes)
	return map[string]string{
		"elastic_components": strings.Join(elasticNodes, ","),
	}
}

type zpagesBridge struct {
	zpages zpagesAPI
}

func newZPagesBridge(zpages zpagesAPI) *zpagesBridge {
	return &zpagesBridge{zpages: zpages}
}

func (b *zpagesBridge) GetPipelineTopology(ctx context.Context) (*otel.PipelineTopology, error) {
	if b == nil || b.zpages == nil {
		return nil, nil
	}
	topology, err := b.zpages.GetPipelineTopology(ctx)
	if err != nil {
		return nil, err
	}
	return convertTopology(topology), nil
}

func convertTopology(topology *PipelineTopology) *otel.PipelineTopology {
	if topology == nil {
		return nil
	}
	out := &otel.PipelineTopology{
		TracezReachable: topology.TracezReachable,
		Pipelines:       make([]otel.PipelineStatus, 0, len(topology.Pipelines)),
	}
	for _, p := range topology.Pipelines {
		receivers := make([]otel.ComponentStatus, 0, len(p.Receivers))
		for _, component := range p.Receivers {
			receivers = append(receivers, convertComponent(component))
		}
		processors := make([]otel.ComponentStatus, 0, len(p.Processors))
		for _, component := range p.Processors {
			processors = append(processors, convertComponent(component))
		}
		exporters := make([]otel.ComponentStatus, 0, len(p.Exporters))
		for _, component := range p.Exporters {
			exporters = append(exporters, convertComponent(component))
		}
		out.Pipelines = append(out.Pipelines, otel.PipelineStatus{
			Name:       p.Name,
			Receivers:  receivers,
			Processors: processors,
			Exporters:  exporters,
		})
	}
	return out
}

func convertComponent(component ComponentStatus) otel.ComponentStatus {
	return otel.ComponentStatus{
		ID:     component.ID,
		Kind:   component.Kind,
		Status: component.Status,
		Error:  component.Error,
	}
}
