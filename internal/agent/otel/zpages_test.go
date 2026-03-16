package otel

import "testing"

func TestParsePipelinezCollectorHTMLParsesEscapedLinks(t *testing.T) {
	html := `<html><body><a href="/debug/pipelinez?zpipelinename=metrics&amp;zcomponentname=otlp&amp;zcomponentkind=receiver">receiver</a></body></html>`

	pipelines, err := parsePipelinezCollectorHTML(html)
	if err != nil {
		t.Fatalf("parsePipelinezCollectorHTML() error = %v", err)
	}
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	if pipelines[0].Name != "metrics" {
		t.Fatalf("expected pipeline name metrics, got %q", pipelines[0].Name)
	}
	if len(pipelines[0].Receivers) != 1 || pipelines[0].Receivers[0].ID != "otlp" {
		t.Fatalf("expected receiver otlp, got %#v", pipelines[0].Receivers)
	}
}

func TestParsePipelinezHTMLFallsBackToCollectorLinks(t *testing.T) {
	html := `<html><body><a href="/debug/pipelinez?zpipelinename=traces&zcomponentname=batch&zcomponentkind=processor">processor</a></body></html>`

	pipelines, err := parsePipelinezHTML(html)
	if err != nil {
		t.Fatalf("parsePipelinezHTML() error = %v", err)
	}
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	if len(pipelines[0].Processors) != 1 || pipelines[0].Processors[0].ID != "batch" {
		t.Fatalf("expected processor batch, got %#v", pipelines[0].Processors)
	}
}
