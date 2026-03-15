package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModelImplementsTeaModelAndRenders(t *testing.T) {
	m := NewModel(false, time.Second, &pipeline.Pipeline{
		Name: "ea",
		Nodes: []pipeline.Node{
			{ID: "input.system-logs", Label: "system-logs", Kind: "input", Status: pipeline.Healthy},
			{ID: "output.default", Label: "default", Kind: "output", Status: pipeline.Healthy},
		},
	})

	var _ tea.Model = m

	view := m.View()
	if !strings.Contains(view, "agent-cli TUI") {
		t.Fatalf("expected TUI header, got:\n%s", view)
	}
}
