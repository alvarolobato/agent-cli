package elasticagent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/metrics"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

func TestAdapterStatusBuildsPipelineFromConfigAndRuntime(t *testing.T) {
	cfg, err := config.ParseElasticAgentConfig("../../../test/fixtures/elastic-agent.yml")
	if err != nil {
		t.Fatalf("ParseElasticAgentConfig() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			t.Fatalf("expected /api/status path, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ea-1",
			"name":"standalone-agent",
			"status":{"overall":"DEGRADED","message":"warnings"},
			"components":[
				{"id":"filestream-default","name":"filestream","status":{"overall":"HEALTHY","message":"ok"}},
				{"id":"httpjson-monitoring","name":"httpjson","status":{"overall":"FAILED","message":"bad"}},
				{"id":"output-default","name":"output-default","status":{"overall":"HEALTHY","message":"ok"}},
				{"id":"output-monitoring","name":"output-monitoring","status":{"overall":"DEGRADED","message":"slow"}}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())
	adapter := NewAdapter(cfg, client)

	got, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if got.Name != "standalone-agent" {
		t.Fatalf("expected pipeline name standalone-agent, got %q", got.Name)
	}
	if len(got.Nodes) != 4 {
		t.Fatalf("expected 4 nodes (2 inputs + 2 outputs), got %d", len(got.Nodes))
	}
	if len(got.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(got.Edges))
	}

	nodesByID := make(map[string]pipeline.Node, len(got.Nodes))
	for _, n := range got.Nodes {
		nodesByID[n.ID] = n
	}

	if nodesByID["input.system-logs"].Status != pipeline.Healthy {
		t.Fatalf("expected system-logs input healthy, got %q", nodesByID["input.system-logs"].Status)
	}
	if nodesByID["input.api-events"].Status != pipeline.Disabled {
		t.Fatalf("expected api-events input disabled, got %q", nodesByID["input.api-events"].Status)
	}
	if nodesByID["output.default"].Status != pipeline.Healthy {
		t.Fatalf("expected output.default healthy, got %q", nodesByID["output.default"].Status)
	}
	if nodesByID["output.monitoring"].Status != pipeline.Degraded {
		t.Fatalf("expected output.monitoring degraded, got %q", nodesByID["output.monitoring"].Status)
	}

	edges := map[string]bool{}
	for _, e := range got.Edges {
		edges[e.From+"->"+e.To] = true
	}
	if !edges["input.system-logs->output.default"] {
		t.Fatalf("expected edge input.system-logs->output.default")
	}
	if !edges["input.api-events->output.monitoring"] {
		t.Fatalf("expected edge input.api-events->output.monitoring")
	}
}

func TestAdapterHealthUsesRuntimeOverall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ea-1",
			"name":"standalone-agent",
			"status":{"overall":"FAILED","message":"boom"},
			"components":[]
		}`))
	}))
	defer server.Close()

	cfg := &config.ElasticAgentConfig{}
	client := NewClient(server.URL, server.Client())
	adapter := NewAdapter(cfg, client)

	health, err := adapter.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if health != pipeline.Error {
		t.Fatalf("expected health error, got %q", health)
	}
}

type fakeMetricsCollector struct {
	snapshot *metrics.Snapshot
}

func (f *fakeMetricsCollector) CollectBeatStats(_ context.Context, endpoint string) (*metrics.Snapshot, error) {
	if endpoint == "" {
		return nil, nil
	}
	return f.snapshot, nil
}

func TestAdapterStatusPopulatesInputMetrics(t *testing.T) {
	cfg, err := config.ParseElasticAgentConfig("../../../test/fixtures/elastic-agent.yml")
	if err != nil {
		t.Fatalf("ParseElasticAgentConfig() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ea-1",
			"name":"standalone-agent",
			"status":{"overall":"HEALTHY","message":"ok"},
			"components":[]
		}`))
	}))
	defer server.Close()

	adapter := newAdapterWithDeps(
		cfg,
		NewClient(server.URL, server.Client()),
		&fakeMetricsCollector{
			snapshot: &metrics.Snapshot{
				EventsInPerSec:  120,
				EventsOutPerSec: 118,
				ErrorCount:      2,
				DropCount:       1,
			},
		},
		func(config.ElasticInput) string { return "http://localhost:5066/stats" },
	)

	got, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	nodesByID := make(map[string]pipeline.Node, len(got.Nodes))
	for _, n := range got.Nodes {
		nodesByID[n.ID] = n
	}

	enabledInput := nodesByID["input.system-logs"]
	if enabledInput.Metrics == nil {
		t.Fatalf("expected metrics for enabled input")
	}
	if enabledInput.Metrics.EventsInPerSec != 120 {
		t.Fatalf("expected input metrics events_in_per_sec=120, got %v", enabledInput.Metrics.EventsInPerSec)
	}
	if enabledInput.Metrics.EventsOutPerSec != 118 {
		t.Fatalf("expected input metrics events_out_per_sec=118, got %v", enabledInput.Metrics.EventsOutPerSec)
	}
	if enabledInput.Metrics.ErrorCount != 2 {
		t.Fatalf("expected input metrics error_count=2, got %v", enabledInput.Metrics.ErrorCount)
	}
	if enabledInput.Metrics.DropCount != 1 {
		t.Fatalf("expected input metrics drop_count=1, got %v", enabledInput.Metrics.DropCount)
	}

	disabledInput := nodesByID["input.api-events"]
	if disabledInput.Metrics != nil {
		t.Fatalf("expected disabled input metrics to be nil")
	}
}
