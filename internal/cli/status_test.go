package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alvarolobato/agent-cli/internal/discovery"
	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

func TestStatusPipelineElasticAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ea-1",
			"name":"standalone-agent",
			"status":{"overall":"HEALTHY","message":"ok"},
			"components":[
				{"id":"filestream-default","name":"filestream","status":{"overall":"HEALTHY","message":"ok"}},
				{"id":"output-default","name":"output-default","status":{"overall":"DEGRADED","message":"slow"}},
				{"id":"output-monitoring","name":"output-monitoring","status":{"overall":"FAILED","message":"failed"}}
			]
		}`))
	}))
	defer server.Close()

	cmd := newStatusCommand()
	pipe, err := statusPipeline(cmd, statusOptions{
		agentType:        "elastic-agent",
		elasticConfig:    "../../test/fixtures/elastic-agent.yml",
		elasticStatusURL: server.URL,
	})
	if err != nil {
		t.Fatalf("statusPipeline() error = %v", err)
	}

	if len(pipe.Nodes) == 0 {
		t.Fatalf("expected nodes in pipeline")
	}
	for _, n := range pipe.Nodes {
		if n.Kind == "output" && n.Label == "default" && n.Status != pipeline.Degraded {
			t.Fatalf("expected output default degraded, got %q", n.Status)
		}
	}
}

func TestStatusCommandJSONElasticAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"ea-1",
			"name":"standalone-agent",
			"status":{"overall":"HEALTHY","message":"ok"},
			"components":[]
		}`))
	}))
	defer server.Close()

	cmd := newStatusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--agent", "elastic-agent",
		"--format", "json",
		"--elastic-config", "../../test/fixtures/elastic-agent.yml",
		"--elastic-url", server.URL,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload := strings.TrimSpace(out.String())
	if !json.Valid([]byte(payload)) {
		t.Fatalf("expected valid JSON output, got %s", payload)
	}
}

func TestStatusCommandJSONEDOT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/debug/pipelinez":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"pipelines":[
					{
						"name":"traces",
						"receivers":[{"id":"otlp","status":"StatusOK"}],
						"processors":[{"id":"batch","status":"StatusOK"}],
						"exporters":[{"id":"debug","status":"StatusOK"}]
					}
				]
			}`))
		case "/debug/tracez":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(`otelcol_receiver_accepted_spans{receiver="otlp"} 100`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cmd := newStatusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--agent", "edot",
		"--format", "json",
		"--edot-config", "../../test/fixtures/edot-config.yaml",
		"--edot-zpages-url", server.URL,
		"--edot-metrics-url", server.URL + "/metrics",
		"--edot-health-url", server.URL + "/",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload := strings.TrimSpace(out.String())
	if !json.Valid([]byte(payload)) {
		t.Fatalf("expected valid JSON output, got %s", payload)
	}
	if !strings.Contains(payload, `"kind": "receiver"`) {
		t.Fatalf("expected EDOT receiver nodes in JSON output, got %s", payload)
	}
}

func TestStatusCommandJSONOTel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/debug/pipelinez":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"pipelines":[
					{
						"name":"metrics",
						"receivers":[{"id":"otlp","status":"StatusOK"}],
						"processors":[{"id":"batch","status":"StatusOK"}],
						"exporters":[{"id":"debug","status":"StatusOK"}]
					}
				]
			}`))
		case "/debug/tracez":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/metrics":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(`otelcol_receiver_accepted_metric_points{receiver="otlp"} 100`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cmd := newStatusCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--agent", "otel",
		"--format", "json",
		"--otel-config", "../../test/fixtures/otel-config.yaml",
		"--otel-zpages-url", server.URL,
		"--otel-metrics-url", server.URL + "/metrics",
		"--otel-health-url", server.URL + "/",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	payload := strings.TrimSpace(out.String())
	if !json.Valid([]byte(payload)) {
		t.Fatalf("expected valid JSON output, got %s", payload)
	}
	if !strings.Contains(payload, `"label": "OTLP receiver"`) {
		t.Fatalf("expected friendly OTLP label in OTel JSON output, got %s", payload)
	}
}

func TestAutoDetectStatusOptionsUsesDiscoveredConfigAndEndpoints(t *testing.T) {
	originalDiscover := discoverAgents
	t.Cleanup(func() { discoverAgents = originalDiscover })
	discoverAgents = func(context.Context) ([]discovery.DiscoveredAgent, error) {
		return []discovery.DiscoveredAgent{
			{
				AgentType:  "edot",
				ConfigPath: "/etc/edot/config.yaml",
				Endpoints: map[string]string{
					"zpages":  "http://localhost:55679",
					"metrics": "http://localhost:8888",
					"health":  "http://localhost:13133",
				},
				Source: "process",
			},
		}, nil
	}

	options, err := autoDetectStatusOptions(context.Background(), statusOptions{})
	if err != nil {
		t.Fatalf("autoDetectStatusOptions() error = %v", err)
	}
	if options.agentType != "edot" {
		t.Fatalf("expected edot agent type, got %q", options.agentType)
	}
	if options.edotConfig != "/etc/edot/config.yaml" {
		t.Fatalf("expected discovered config path, got %q", options.edotConfig)
	}
	if options.edotMetricsURL != "http://localhost:8888/metrics" {
		t.Fatalf("expected discovered metrics URL, got %q", options.edotMetricsURL)
	}
}

func TestAutoDetectStatusOptionsErrorsWhenNothingDiscovered(t *testing.T) {
	originalDiscover := discoverAgents
	t.Cleanup(func() { discoverAgents = originalDiscover })
	discoverAgents = func(context.Context) ([]discovery.DiscoveredAgent, error) {
		return nil, nil
	}

	_, err := autoDetectStatusOptions(context.Background(), statusOptions{})
	if err == nil {
		t.Fatalf("expected auto-detect error when no agents discovered")
	}
}
