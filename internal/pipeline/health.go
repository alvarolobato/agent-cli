package pipeline

// HealthStatus represents the current state of a pipeline component.
type HealthStatus string

const (
	Healthy  HealthStatus = "healthy"
	Degraded HealthStatus = "degraded"
	Error    HealthStatus = "error"
	Disabled HealthStatus = "disabled"
	Unknown  HealthStatus = "unknown"
)
