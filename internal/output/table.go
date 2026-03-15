package output

import (
	"fmt"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
	lgtable "github.com/charmbracelet/lipgloss/table"
)

// RenderTable returns a table representation of pipeline nodes.
func RenderTable(p *pipeline.Pipeline) string {
	rows := make([][]string, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		eventsPerSec := "-"
		errorCount := "-"
		if n.Metrics != nil {
			eventsPerSec = fmt.Sprintf("%.2f", n.Metrics.EventsOutPerSec)
			errorCount = fmt.Sprintf("%.0f", n.Metrics.ErrorCount)
		}
		rows = append(rows, []string{
			n.Label,
			healthIcon(n.Status),
			string(n.Status),
			eventsPerSec,
			errorCount,
		})
	}

	t := lgtable.New().
		Headers("Component", "Health", "Status", "Events/s", "Errors").
		Rows(rows...)

	return t.String()
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
