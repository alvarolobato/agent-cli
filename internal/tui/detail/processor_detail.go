package detail

import (
	"fmt"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

type ProcessorDetail struct {
	Node pipeline.Node
}

func NewProcessorDetail(node pipeline.Node) ProcessorDetail {
	return ProcessorDetail{Node: node}
}

func (m ProcessorDetail) View() string {
	return fmt.Sprintf("Processor detail\n\nName: %s\nStatus: %s\n\nEsc/b to go back", m.Node.Label, m.Node.Status)
}
