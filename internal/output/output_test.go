package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

func fixturePipeline() *pipeline.Pipeline {
	return &pipeline.Pipeline{
		Name: "fixture",
		Nodes: []pipeline.Node{
			{ID: "in", Label: "input", Kind: "input", Status: pipeline.Healthy},
			{ID: "out", Label: "output", Kind: "output", Status: pipeline.Degraded},
		},
		Edges:     []pipeline.Edge{{From: "in", To: "out"}},
		UpdatedAt: time.Unix(100, 0).UTC(),
	}
}

func fixtureOTelPipeline() *pipeline.Pipeline {
	return &pipeline.Pipeline{
		Name: "edot",
		Nodes: []pipeline.Node{
			{ID: "receiver.otlp", Label: "otlp", Kind: "receiver", Status: pipeline.Healthy, Metrics: &pipeline.NodeMetrics{EventsInPerSec: 120}},
			{ID: "processor.batch", Label: "batch", Kind: "processor", Status: pipeline.Degraded, Metrics: &pipeline.NodeMetrics{DropCount: 2}},
			{ID: "exporter.elasticsearch/logs", Label: "elasticsearch/logs [Elastic]", Kind: "exporter", Status: pipeline.Error, Metrics: &pipeline.NodeMetrics{EventsOutPerSec: 118, ErrorCount: 1}},
		},
		Edges: []pipeline.Edge{
			{From: "receiver.otlp", To: "processor.batch"},
			{From: "processor.batch", To: "exporter.elasticsearch/logs"},
		},
		UpdatedAt: time.Unix(100, 0).UTC(),
	}
}

func TestRenderJSON(t *testing.T) {
	got, err := RenderJSON(fixturePipeline())
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	assertGolden(t, "ea-status.json", got)
}

func TestRenderTable(t *testing.T) {
	got := RenderTable(fixturePipeline())
	if !strings.Contains(got, "⚠") {
		t.Fatalf("expected table content, got: %s", got)
	}
	assertGolden(t, "ea-status-table.golden", got)
}

func TestRenderTableEDOT(t *testing.T) {
	got := RenderTable(fixtureOTelPipeline())
	if !strings.Contains(got, "otlp") {
		t.Fatalf("expected EDOT receiver component in table output, got: %s", got)
	}
	assertGolden(t, "edot-status-table.golden", got)
}

func assertGolden(t *testing.T, fileName, got string) {
	t.Helper()
	goldenPath := filepath.Join("..", "..", "test", "fixtures", "golden", fileName)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := strings.TrimSpace(string(wantBytes))
	if strings.TrimSpace(got) != want {
		t.Fatalf("golden mismatch for %s\nwant:\n%s\n\ngot:\n%s", fileName, want, got)
	}
}
