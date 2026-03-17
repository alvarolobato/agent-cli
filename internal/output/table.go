package output

import (
	"fmt"
	"strings"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
	lgtable "github.com/charmbracelet/lipgloss/table"
)

// RenderTable returns a table representation of pipeline nodes.
func RenderTable(p *pipeline.Pipeline) string {
	if p == nil {
		return ""
	}
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

	meta := renderPipelineMetadata(p.Metadata)
	if meta == "" {
		return t.String()
	}
	return meta + "\n\n" + t.String()
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

func renderPipelineMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	details := make([]string, 0, 2)
	if version := strings.TrimSpace(metadata["agent_version"]); version != "" {
		details = append(details, "version "+version)
	}
	if flavor := strings.TrimSpace(metadata["agent_flavor"]); flavor != "" {
		details = append(details, "flavor "+flavor)
	}
	if len(details) == 0 {
		return ""
	}
	return "Agent: " + strings.Join(details, " | ")
}
