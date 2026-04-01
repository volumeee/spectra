package port

// SystemMonitor checks if the system is overloaded (CPU/memory).
// Used by the queue to reject requests when resources are exhausted.
type SystemMonitor interface {
	// Overloaded returns true and a reason string if system is under pressure.
	Overloaded() (bool, string)
	// Stats returns current CPU and memory usage percentages.
	Stats() MonitorStats
}

type MonitorStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	Overloaded    bool    `json:"overloaded"`
	Reason        string  `json:"reason,omitempty"`
}
