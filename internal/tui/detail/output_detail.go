package detail

import (
	"fmt"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

type OutputDetail struct {
	Node pipeline.Node
}

func NewOutputDetail(node pipeline.Node) OutputDetail {
	return OutputDetail{Node: node}
}

func (m OutputDetail) View() string {
	return fmt.Sprintf("Output detail\n\nName: %s\nStatus: %s\n\nEsc/b to go back", m.Node.Label, m.Node.Status)
}
