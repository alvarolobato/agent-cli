package detail

import (
	"fmt"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

type InputDetail struct {
	Node pipeline.Node
}

func NewInputDetail(node pipeline.Node) InputDetail {
	return InputDetail{Node: node}
}

func (m InputDetail) View() string {
	return fmt.Sprintf("Input detail\n\nName: %s\nStatus: %s\n\nEsc/b to go back", m.Node.Label, m.Node.Status)
}
