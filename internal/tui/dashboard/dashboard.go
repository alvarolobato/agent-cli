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
	cursor   int
	items    []string
	pipe     *pipeline.Pipeline
	usesOTel bool
}

// NewModel returns the initial dashboard state.
func NewModel(p *pipeline.Pipeline) Model {
	if p == nil {
		p = pipeline.ExamplePipeline()
	}
	usesOTel := hasOTelKinds(p)
	items := columnTitles(usesOTel)
	return Model{
		items:    items,
		pipe:     p,
		usesOTel: usesOTel,
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

func (m Model) SelectedColumn() int {
	return m.cursor
}

func (m Model) FirstNodeForColumn() *pipeline.Node {
	leftKind, rightKind := "input", "output"
	if m.usesOTel {
		leftKind, rightKind = "receiver", "exporter"
	}
	var kind string
	switch m.cursor {
	case 0:
		kind = leftKind
	case 1:
		kind = "processor"
	case 2:
		kind = rightKind
	default:
		return nil
	}
	nodes := m.nodesByKind(kind)
	if len(nodes) == 0 {
		return nil
	}
	node := nodes[0]
	return &node
}

func (m Model) View() string {
	leftKind, rightKind := "input", "output"
	if m.usesOTel {
		leftKind, rightKind = "receiver", "exporter"
	}
	left := renderColumn(m.items[0], m.nodesByKind(leftKind), m.cursor == 0)
	processors := renderColumn(m.items[1], m.nodesByKind("processor"), m.cursor == 1)
	right := renderColumn(m.items[2], m.nodesByKind(rightKind), m.cursor == 2)
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, processors, right)
	return metadataHeader(m.pipe) + content
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

func renderColumn(title string, nodes []pipeline.Node, selected bool) string {
	var b strings.Builder
	if selected {
		b.WriteString(titleStyle.Underline(true).Render(title))
	} else {
		b.WriteString(titleStyle.Render(title))
	}
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

func hasOTelKinds(p *pipeline.Pipeline) bool {
	if p == nil {
		return false
	}
	hasReceiver := false
	hasExporter := false
	for _, node := range p.Nodes {
		if node.Kind == "receiver" {
			hasReceiver = true
		}
		if node.Kind == "exporter" {
			hasExporter = true
		}
	}
	return hasReceiver || hasExporter
}

func columnTitles(usesOTel bool) []string {
	if usesOTel {
		return []string{"Receivers", "Processors", "Exporters"}
	}
	return []string{"Inputs", "Processors", "Outputs"}
}

func metadataHeader(p *pipeline.Pipeline) string {
	if p == nil || len(p.Metadata) == 0 {
		return ""
	}
	version := strings.TrimSpace(p.Metadata["agent_version"])
	flavor := strings.TrimSpace(p.Metadata["agent_flavor"])
	if version == "" && flavor == "" {
		return ""
	}
	parts := make([]string, 0, 2)
	if version != "" {
		parts = append(parts, "version "+version)
	}
	if flavor != "" {
		parts = append(parts, "flavor "+flavor)
	}
	return "Agent " + strings.Join(parts, " | ") + "\n\n"
}
