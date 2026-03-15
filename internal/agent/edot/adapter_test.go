package edot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvarolobato/agent-cli/internal/config"
	"github.com/alvarolobato/agent-cli/internal/metrics"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

type fakeZPages struct {
	topology *PipelineTopology
	err      error
}

func (f *fakeZPages) GetPipelineTopology(_ context.Context) (*PipelineTopology, error) {
	return f.topology, f.err
}

type fakeOTelMetrics struct {
	snapshot *metrics.OTelSnapshot
}

func (f *fakeOTelMetrics) CollectOTelPrometheus(_ context.Context, _ string) (*metrics.OTelSnapshot, error) {
	return f.snapshot, nil
}

func TestAdapterStatusBuildsPipelineWithMetrics(t *testing.T) {
	cfg, err := config.ParseOTelCollectorConfig("../../../test/fixtures/edot-config.yaml")
	if err != nil {
		t.Fatalf("ParseOTelCollectorConfig() error = %v", err)
	}

	adapter := newAdapterWithDeps(
		cfg,
		&fakeZPages{
			topology: &PipelineTopology{
				TracezReachable: true,
				Pipelines: []PipelineStatus{
					{
						Name:       "logs",
						Receivers:  []ComponentStatus{{ID: "otlp", Status: "StatusOK"}},
						Processors: []ComponentStatus{{ID: "ecsformatprocessor/logs", Status: "StatusOK"}, {ID: "batch", Status: "StatusOK"}},
						Exporters:  []ComponentStatus{{ID: "elasticsearch/logs", Status: "StatusOK"}},
					},
				},
			},
		},
		&fakeOTelMetrics{
			snapshot: &metrics.OTelSnapshot{
				Receivers: map[string]metrics.OTelComponentMetrics{
					"otlp": {Accepted: 120},
				},
				Processors: map[string]metrics.OTelComponentMetrics{
					"batch": {Dropped: 2},
				},
				Exporters: map[string]metrics.OTelComponentMetrics{
					"elasticsearch/logs": {Sent: 118, SendFailed: 1},
				},
			},
		},
		nil,
		"http://localhost:8888/metrics",
		"http://localhost:13133/",
	)

	got, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if got.Metadata["tracez_reachable"] != "true" {
		t.Fatalf("expected tracez_reachable metadata true, got %q", got.Metadata["tracez_reachable"])
	}

	nodesByID := map[string]pipeline.Node{}
	for _, node := range got.Nodes {
		nodesByID[node.ID] = node
	}
	if len(nodesByID) == 0 {
		t.Fatalf("expected nodes to be present")
	}

	receiver := nodesByID["receiver.otlp"]
	if receiver.Metrics == nil || receiver.Metrics.EventsInPerSec != 120 {
		t.Fatalf("expected receiver metrics accepted=120, got %#v", receiver.Metrics)
	}

	processor := nodesByID["processor.batch"]
	if processor.Metrics == nil || processor.Metrics.DropCount != 2 {
		t.Fatalf("expected processor drop_count=2, got %#v", processor.Metrics)
	}
	if processor.Status != pipeline.Degraded {
		t.Fatalf("expected processor degraded from drop count, got %q", processor.Status)
	}

	exporter := nodesByID["exporter.elasticsearch/logs"]
	if exporter.Metrics == nil || exporter.Metrics.EventsOutPerSec != 118 {
		t.Fatalf("expected exporter events_out_per_sec=118, got %#v", exporter.Metrics)
	}
	if exporter.Status != pipeline.Error {
		t.Fatalf("expected exporter error from send_failed, got %q", exporter.Status)
	}
	if !strings.Contains(exporter.Label, "[Elastic]") {
		t.Fatalf("expected elastic-branded label, got %q", exporter.Label)
	}
}

func TestAdapterHealthUsesHealthCheckEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("expected / path, got %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"Server available"}`))
	}))
	defer server.Close()

	cfg, err := config.ParseOTelCollectorConfig("../../../test/fixtures/edot-config.yaml")
	if err != nil {
		t.Fatalf("ParseOTelCollectorConfig() error = %v", err)
	}
	cfg.Extensions["health_check"] = config.OTelComponent{
		Name: "health_check",
		Type: "health_check",
		Raw: map[string]interface{}{
			"endpoint": strings.TrimPrefix(server.URL, "http://"),
		},
	}

	adapter := newAdapterWithDeps(cfg, &fakeZPages{}, &fakeOTelMetrics{}, server.Client(), "", "")

	health, err := adapter.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if health != pipeline.Healthy {
		t.Fatalf("expected health healthy, got %q", health)
	}
}
