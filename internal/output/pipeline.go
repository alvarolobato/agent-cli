package output

import (
	"fmt"
	"strings"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

// RenderPipeline renders a compact ASCII pipeline diagram.
func RenderPipeline(p *pipeline.Pipeline) string {
	if p == nil {
		return ""
	}

	leftKind, rightKind := "input", "output"
	leftTitle, rightTitle := "INPUTS", "OUTPUTS"
	if hasKind(p, "receiver") || hasKind(p, "exporter") {
		leftKind, rightKind = "receiver", "exporter"
		leftTitle, rightTitle = "RECEIVERS", "EXPORTERS"
	}

	left := nodesByKind(p, leftKind)
	mid := nodesByKind(p, "processor")
	right := nodesByKind(p, rightKind)
	rows := max(len(left), len(mid), len(right))
	if rows == 0 {
		return "No pipeline components discovered."
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%-34s %-34s %-34s\n", leftTitle, "PROCESSORS", rightTitle)
	b.WriteString(strings.Repeat("-", 104))
	b.WriteString("\n")
	for i := 0; i < rows; i++ {
		l := formatNodeAt(left, i)
		m := formatNodeAt(mid, i)
		r := formatNodeAt(right, i)
		_, _ = fmt.Fprintf(&b, "%-34s %-34s %-34s\n", l, m, r)
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatNodeAt(nodes []pipeline.Node, idx int) string {
	if idx >= len(nodes) {
		return ""
	}
	node := nodes[idx]
	metrics := "-"
	if node.Metrics != nil {
		metrics = fmt.Sprintf("in %.1f/s out %.1f/s err %.0f", node.Metrics.EventsInPerSec, node.Metrics.EventsOutPerSec, node.Metrics.ErrorCount)
	}
	return fmt.Sprintf("%s %s (%s)", healthIcon(node.Status), node.Label, metrics)
}

func nodesByKind(p *pipeline.Pipeline, kind string) []pipeline.Node {
	if p == nil {
		return nil
	}
	nodes := make([]pipeline.Node, 0)
	for _, node := range p.Nodes {
		if node.Kind == kind {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func hasKind(p *pipeline.Pipeline, kind string) bool {
	for _, node := range p.Nodes {
		if node.Kind == kind {
			return true
		}
	}
	return false
}

func max(values ...int) int {
	out := 0
	for _, value := range values {
		if value > out {
			out = value
		}
	}
	return out
}
