package dashboard

import (
	"strings"
	"testing"
	"time"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

func TestModelViewRendersPipelineColumns(t *testing.T) {
	model := NewModel(&pipeline.Pipeline{
		Name: "ea",
		Nodes: []pipeline.Node{
			{ID: "input.system-logs", Label: "system-logs", Kind: "input", Status: pipeline.Healthy, Metrics: &pipeline.NodeMetrics{EventsOutPerSec: 10}},
			{ID: "processor.batch", Label: "batch", Kind: "processor", Status: pipeline.Degraded, Metrics: &pipeline.NodeMetrics{EventsOutPerSec: 9}},
			{ID: "output.default", Label: "default", Kind: "output", Status: pipeline.Error},
		},
		UpdatedAt: time.Now().UTC(),
		Metadata:  map[string]string{"agent_version": "9.3.1", "agent_flavor": "servers"},
	})

	view := model.View()
	for _, token := range []string{"Inputs", "Processors", "Outputs", "system-logs", "batch", "default"} {
		if !strings.Contains(view, token) {
			t.Fatalf("expected %q in dashboard view, got:\n%s", token, view)
		}
	}
	if !strings.Contains(view, "version 9.3.1") {
		t.Fatalf("expected metadata header in dashboard view, got:\n%s", view)
	}
}

func TestModelViewRendersOTelColumns(t *testing.T) {
	model := NewModel(&pipeline.Pipeline{
		Name: "edot",
		Nodes: []pipeline.Node{
			{ID: "receiver.otlp", Label: "otlp", Kind: "receiver", Status: pipeline.Healthy},
			{ID: "processor.batch", Label: "batch", Kind: "processor", Status: pipeline.Healthy},
			{ID: "exporter.debug", Label: "debug", Kind: "exporter", Status: pipeline.Healthy},
		},
		UpdatedAt: time.Now().UTC(),
	})

	view := model.View()
	for _, token := range []string{"Receivers", "Processors", "Exporters", "otlp", "batch", "debug"} {
		if !strings.Contains(view, token) {
			t.Fatalf("expected %q in dashboard view, got:\n%s", token, view)
		}
	}
}
