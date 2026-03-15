package output

import (
	"github.com/alvarolobato/agent-cli/internal/pipeline"
	lgtable "github.com/charmbracelet/lipgloss/table"
)

// RenderTable returns a table representation of pipeline nodes.
func RenderTable(p *pipeline.Pipeline) string {
	rows := make([][]string, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		rows = append(rows, []string{n.ID, n.Label, n.Kind, string(n.Status)})
	}

	t := lgtable.New().
		Headers("ID", "Label", "Kind", "Status").
		Rows(rows...)

	return t.String()
}
