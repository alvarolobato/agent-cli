package output

import (
	"encoding/json"

	"github.com/alvarolobato/agent-cli/internal/pipeline"
)

// RenderJSON returns an indented JSON view of a pipeline model.
func RenderJSON(p *pipeline.Pipeline) (string, error) {
	payload, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
