package dashboard

import (
	"fmt"
	"strings"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
	"github.com/charmbracelet/lipgloss"
)

var (
	columnStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Width(34)
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	nodeStyle   = lipgloss.NewStyle().PaddingLeft(1)
)

// Model renders a pipeline-first dashboard screen.
type Model struct {
	cursor int
	items  []string
	pipe   *pipeline.Pipeline
}

// NewModel returns the initial dashboard state.
func NewModel(p *pipeline.Pipeline) Model {
	items := []string{
		"Inputs",
		"Processors",
		"Outputs",
	}
	if p == nil {
		p = pipeline.ExamplePipeline()
	}
	return Model{
		items: items,
		pipe:  p,
	}
}

func (m *Model) MoveUp() {
	if len(m.items) == 0 {
		return
	}
	m.cursor--
	if m.cursor < 0 {
		m.cursor = len(m.items) - 1
	}
}

func (m *Model) MoveDown() {
	if len(m.items) == 0 {
		return
	}
	m.cursor++
	if m.cursor >= len(m.items) {
		m.cursor = 0
	}
}

func (m Model) View() string {
	inputs := renderColumn(m.items[0], m.nodesByKind("input"))
	processors := renderColumn(m.items[1], m.nodesByKind("processor"))
	outputs := renderColumn(m.items[2], m.nodesByKind("output"))
	return lipgloss.JoinHorizontal(lipgloss.Top, inputs, processors, outputs)
}

func (m Model) nodesByKind(kind string) []pipeline.Node {
	if m.pipe == nil {
		return nil
	}
	nodes := make([]pipeline.Node, 0, len(m.pipe.Nodes))
	for _, n := range m.pipe.Nodes {
		if n.Kind == kind {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func renderColumn(title string, nodes []pipeline.Node) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	if len(nodes) == 0 {
		b.WriteString(nodeStyle.Render("-"))
		return columnStyle.Render(strings.TrimRight(b.String(), "\n"))
	}

	for _, n := range nodes {
		events := "-"
		if n.Metrics != nil {
			events = fmt.Sprintf("%.2f/s", n.Metrics.EventsOutPerSec)
		}
		b.WriteString(nodeStyle.Render(fmt.Sprintf("%s %s (%s)", healthIcon(n.Status), n.Label, events)))
		b.WriteString("\n")
	}

	return columnStyle.Render(strings.TrimRight(b.String(), "\n"))
}

func healthIcon(status pipeline.HealthStatus) string {
	switch status {
	case pipeline.Healthy:
		return "✓"
	case pipeline.Degraded:
		return "⚠"
	case pipeline.Error:
		return "✗"
	case pipeline.Disabled:
		return "○"
	default:
		return "?"
	}
}
