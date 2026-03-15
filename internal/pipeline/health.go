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

// MapOTelRuntimeStatus normalizes OTel component runtime states.
func MapOTelRuntimeStatus(status string) HealthStatus {
	switch strings.TrimSpace(status) {
	case "StatusOK":
		return Healthy
	case "StatusRecoverableError":
		return Degraded
	case "StatusPermanentError", "StatusFatalError":
		return Error
	case "StatusStopped":
		return Disabled
	case "", "StatusNone", "StatusStarting", "StatusUnknown":
		return Unknown
	default:
		return mapStateToHealth(status)
	}
}

// AssessOTelComponentHealth combines runtime state with error/drop counters.
func AssessOTelComponentHealth(current HealthStatus, sendFailed, dropped float64, enabled bool) HealthStatus {
	if !enabled {
		return Disabled
	}
	if sendFailed > 0 {
		return Error
	}
	if dropped > 0 && current != Error {
		return Degraded
	}
	if current == Unknown {
		return Unknown
	}
	return current
}
