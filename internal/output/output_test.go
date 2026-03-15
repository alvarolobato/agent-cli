package output

import (
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

func TestRenderJSON(t *testing.T) {
	got, err := RenderJSON(fixturePipeline())
	if err != nil {
		t.Fatalf("RenderJSON() error = %v", err)
	}
	if !strings.Contains(got, "\"name\": \"fixture\"") {
		t.Fatalf("expected pipeline name in JSON, got: %s", got)
	}
}

func TestRenderTable(t *testing.T) {
	got := RenderTable(fixturePipeline())
	if !strings.Contains(got, "input") || !strings.Contains(got, "degraded") {
		t.Fatalf("expected table content, got: %s", got)
	}
}
