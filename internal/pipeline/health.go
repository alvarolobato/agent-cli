package pipeline

import "strings"

// HealthStatus represents the current state of a pipeline component.
type HealthStatus string

const (
	Healthy  HealthStatus = "healthy"
	Degraded HealthStatus = "degraded"
	Error    HealthStatus = "error"
	Disabled HealthStatus = "disabled"
	Unknown  HealthStatus = "unknown"
)

// AssessHealth normalizes a node status/state into one of the shared health enums.
func AssessHealth(node *Node) HealthStatus {
	if node == nil {
		return Unknown
	}
	return mapStateToHealth(string(node.Status))
}

func mapStateToHealth(state string) HealthStatus {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "HEALTHY", "RUNNING", "OK":
		return Healthy
	case "DEGRADED", "WARNING", "WARN":
		return Degraded
	case "FAILED", "ERROR":
		return Error
	case "DISABLED", "STOPPED":
		return Disabled
	default:
		return Unknown
	}
}
