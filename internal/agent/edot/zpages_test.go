package edot

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestZPagesClientGetPipelineTopologyJSON(t *testing.T) {
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
						"exporters":[{"id":"elasticsearch","status":"StatusRecoverableError","error":"timeout"}]
					}
				]
			}`))
		case "/debug/tracez":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewZPagesClient(server.URL, server.Client())
	topology, err := client.GetPipelineTopology(context.Background())
	if err != nil {
		t.Fatalf("GetPipelineTopology() error = %v", err)
	}

	if !topology.TracezReachable {
		t.Fatalf("expected tracez to be reachable")
	}
	if len(topology.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(topology.Pipelines))
	}
	p := topology.Pipelines[0]
	if p.Name != "traces" {
		t.Fatalf("expected pipeline traces, got %q", p.Name)
	}
	if len(p.Receivers) != 1 || p.Receivers[0].ID != "otlp" {
		t.Fatalf("expected receiver otlp, got %#v", p.Receivers)
	}
	if len(p.Processors) != 1 || p.Processors[0].ID != "batch" {
		t.Fatalf("expected processor batch, got %#v", p.Processors)
	}
	if len(p.Exporters) != 1 || p.Exporters[0].ID != "elasticsearch" {
		t.Fatalf("expected exporter elasticsearch, got %#v", p.Exporters)
	}
	if p.Exporters[0].Status != "StatusRecoverableError" {
		t.Fatalf("expected exporter status StatusRecoverableError, got %q", p.Exporters[0].Status)
	}
	if p.Exporters[0].Error != "timeout" {
		t.Fatalf("expected exporter error timeout, got %q", p.Exporters[0].Error)
	}
}

func TestZPagesClientGetPipelineTopologyHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/debug/pipelinez":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`
				<html><body>
					<table>
						<tr data-pipeline="metrics" data-kind="receiver" data-component="prometheus" data-status="StatusOK" data-error=""></tr>
						<tr data-pipeline="metrics" data-kind="processor" data-component="memory_limiter" data-status="StatusOK" data-error=""></tr>
						<tr data-pipeline="metrics" data-kind="exporter" data-component="debug" data-status="StatusOK" data-error=""></tr>
					</table>
				</body></html>
			`))
		case "/debug/tracez":
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewZPagesClient(server.URL, server.Client())
	topology, err := client.GetPipelineTopology(context.Background())
	if err != nil {
		t.Fatalf("GetPipelineTopology() error = %v", err)
	}

	if topology.TracezReachable {
		t.Fatalf("expected tracez to be unreachable")
	}
	if len(topology.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(topology.Pipelines))
	}
	p := topology.Pipelines[0]
	if p.Name != "metrics" {
		t.Fatalf("expected pipeline metrics, got %q", p.Name)
	}
	if len(p.Receivers) != 1 || p.Receivers[0].ID != "prometheus" {
		t.Fatalf("expected receiver prometheus, got %#v", p.Receivers)
	}
	if len(p.Processors) != 1 || p.Processors[0].ID != "memory_limiter" {
		t.Fatalf("expected processor memory_limiter, got %#v", p.Processors)
	}
	if len(p.Exporters) != 1 || p.Exporters[0].ID != "debug" {
		t.Fatalf("expected exporter debug, got %#v", p.Exporters)
	}
}

func TestParsePipelinezHTMLUnsupportedShapeError(t *testing.T) {
	_, err := parsePipelinezHTML(`<html><body><table><tr><td>plain-html-without-data-attrs</td></tr></table></body></html>`)
	if err == nil {
		t.Fatalf("expected parsePipelinezHTML to fail without fallback data attributes")
	}
	if !strings.Contains(err.Error(), "supported fallback expects") {
		t.Fatalf("expected explicit fallback limitation in error, got %q", err.Error())
	}
}
