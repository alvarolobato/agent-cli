package pipeline

import "time"

// Pipeline is a graph model of an agent data flow.
type Pipeline struct {
	Name      string            `json:"name"`
	Nodes     []Node            `json:"nodes"`
	Edges     []Edge            `json:"edges"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Node is a pipeline vertex.
type Node struct {
	ID     string       `json:"id"`
	Label  string       `json:"label"`
	Kind   string       `json:"kind"`
	Status HealthStatus `json:"status"`
}

// Edge is a directional connection between nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ComponentStatus is a flattened status view used by outputs and UIs.
type ComponentStatus struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Kind    string       `json:"kind"`
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

// ExamplePipeline returns a small placeholder pipeline for skeleton commands.
func ExamplePipeline() *Pipeline {
	return &Pipeline{
		Name: "example",
		Nodes: []Node{
			{ID: "input.logs", Label: "logs input", Kind: "input", Status: Healthy},
			{ID: "processor.batch", Label: "batch", Kind: "processor", Status: Healthy},
			{ID: "output.es", Label: "elasticsearch", Kind: "output", Status: Healthy},
		},
		Edges: []Edge{
			{From: "input.logs", To: "processor.batch"},
			{From: "processor.batch", To: "output.es"},
		},
		UpdatedAt: time.Now().UTC(),
	}
}
