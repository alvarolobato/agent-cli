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

func TestModelNavigatesToDetailsAndBack(t *testing.T) {
	m := NewModel(false, 0, &pipeline.Pipeline{
		Nodes: []pipeline.Node{
			{ID: "input.logs", Label: "logs", Kind: "input", Status: pipeline.Healthy},
			{ID: "processor.batch", Label: "batch", Kind: "processor", Status: pipeline.Healthy},
			{ID: "output.default", Label: "default", Kind: "output", Status: pipeline.Healthy},
		},
		Metadata: map[string]string{"config_warnings": "sample warning"},
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if !strings.Contains(m.View(), "Input detail") {
		t.Fatalf("expected input detail screen, got:\n%s", m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if !strings.Contains(m.View(), "Enter detail") {
		t.Fatalf("expected dashboard after Esc, got:\n%s", m.View())
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	if !strings.Contains(m.View(), "Errors and warnings") {
		t.Fatalf("expected errors screen, got:\n%s", m.View())
	}
}

func TestModelLiveTickUpdatesTimestamp(t *testing.T) {
	m := NewModel(true, 2*time.Second, pipeline.ExamplePipeline())
	before := m.lastUpdated

	updated, _ := m.Update(tickMsg(time.Now().UTC().Add(2 * time.Second)))
	m = updated.(Model)
	if !m.lastUpdated.After(before) {
		t.Fatalf("expected lastUpdated to advance, before=%s after=%s", before, m.lastUpdated)
	}
}
