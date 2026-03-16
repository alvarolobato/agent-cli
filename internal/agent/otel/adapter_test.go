package otel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestAdapterStatusBuildsGenericPipelineLabels(t *testing.T) {
	cfg, err := config.ParseOTelCollectorConfig("../../../test/fixtures/otel-config.yaml")
	if err != nil {
		t.Fatalf("ParseOTelCollectorConfig() error = %v", err)
	}

	adapter := NewAdapterWithOptions(cfg, &fakeZPages{
		topology: &PipelineTopology{
			TracezReachable: true,
			Pipelines: []PipelineStatus{
				{
					Name:       "traces",
					Receivers:  []ComponentStatus{{ID: "otlp", Status: "StatusOK"}},
					Processors: []ComponentStatus{{ID: "batch", Status: "StatusOK"}},
					Exporters:  []ComponentStatus{{ID: "debug", Status: "StatusOK"}},
				},
			},
		},
	}, &fakeOTelMetrics{}, AdapterOptions{})

	got, err := adapter.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	nodesByID := map[string]pipeline.Node{}
	for _, node := range got.Nodes {
		nodesByID[node.ID] = node
	}
	if nodesByID["receiver.otlp"].Label != "OTLP receiver" {
		t.Fatalf("expected friendly receiver label, got %q", nodesByID["receiver.otlp"].Label)
	}
	if nodesByID["processor.batch"].Label != "Batch processor" {
		t.Fatalf("expected friendly processor label, got %q", nodesByID["processor.batch"].Label)
	}
	if nodesByID["exporter.debug"].Label != "Debug exporter" {
		t.Fatalf("expected friendly exporter label, got %q", nodesByID["exporter.debug"].Label)
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

	cfg, err := config.ParseOTelCollectorConfig("../../../test/fixtures/otel-config.yaml")
	if err != nil {
		t.Fatalf("ParseOTelCollectorConfig() error = %v", err)
	}
	cfg.Extensions = map[string]config.OTelComponent{
		"health_check": {
			Name: "health_check",
			Type: "health_check",
			Raw: map[string]interface{}{
				"endpoint": server.URL,
			},
		},
	}

	adapter := NewAdapterWithOptions(cfg, &fakeZPages{}, &fakeOTelMetrics{}, AdapterOptions{
		HTTPClient: server.Client(),
	})
	health, err := adapter.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if health != pipeline.Healthy {
		t.Fatalf("expected health healthy, got %q", health)
	}
}

func TestAdapterHealthNormalizesBindAllEndpointToLocalhost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"Server available"}`))
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(server.URL) error = %v", err)
	}

	cfg, err := config.ParseOTelCollectorConfig("../../../test/fixtures/otel-config.yaml")
	if err != nil {
		t.Fatalf("ParseOTelCollectorConfig() error = %v", err)
	}
	cfg.Extensions = map[string]config.OTelComponent{
		"health_check": {
			Name: "health_check",
			Type: "health_check",
			Raw: map[string]interface{}{
				"endpoint": "0.0.0.0:" + parsed.Port(),
			},
		},
	}

	adapter := NewAdapterWithOptions(cfg, &fakeZPages{}, &fakeOTelMetrics{}, AdapterOptions{
		HTTPClient: server.Client(),
	})
	health, err := adapter.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if health != pipeline.Healthy {
		t.Fatalf("expected health healthy, got %q", health)
	}
}

func TestHealthURLFallsBackOnInvalidEndpoint(t *testing.T) {
	cfg := &config.OTelCollectorConfig{
		Extensions: map[string]config.OTelComponent{
			"health_check": {
				Name: "health_check",
				Type: "health_check",
				Raw: map[string]interface{}{
					"endpoint": "://bad-url",
				},
			},
		},
	}
	const fallback = "http://localhost:13133/"
	got := healthURL(cfg, fallback)
	if strings.TrimSpace(got) != fallback {
		t.Fatalf("expected fallback %q, got %q", fallback, got)
	}
}
