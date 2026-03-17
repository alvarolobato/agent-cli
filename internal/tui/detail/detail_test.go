package detail

import (
	"strings"
	"testing"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDetailScreensRender(t *testing.T) {
	node := pipeline.Node{Label: "sample", Status: pipeline.Healthy}
	if !strings.Contains(NewInputDetail(node).View(), "Input detail") {
		t.Fatalf("expected input detail view")
	}
	if !strings.Contains(NewProcessorDetail(node).View(), "Processor detail") {
		t.Fatalf("expected processor detail view")
	}
	if !strings.Contains(NewOutputDetail(node).View(), "Output detail") {
		t.Fatalf("expected output detail view")
	}
}

func TestRawConfigViewScrollableAndColored(t *testing.T) {
	m := NewRawConfig("outputs:\n  default:\n    type: elasticsearch\n")
	if !strings.Contains(m.View(), "Raw config") {
		t.Fatalf("expected raw config header")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	rendered := updated.View()
	if !strings.Contains(rendered, "outputs") {
		t.Fatalf("expected yaml content in raw config view, got: %s", rendered)
	}
}
