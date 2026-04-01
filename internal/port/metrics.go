package port

// MetricsCollector records operational statistics for observability.
type MetricsCollector interface {
	RecordRequest(plugin, method string, durationMs int64, success bool)
	Snapshot() MetricsSnapshot
}

type MetricsSnapshot struct {
	TotalRequests   int64                     `json:"total_requests"`
	TotalSuccess    int64                     `json:"total_success"`
	TotalFailed     int64                     `json:"total_failed"`
	TotalTimedOut   int64                     `json:"total_timed_out"`
	AvgDurationMs   float64                   `json:"avg_duration_ms"`
	ByPlugin        map[string]PluginMetrics  `json:"by_plugin"`
}

type PluginMetrics struct {
	Requests    int64   `json:"requests"`
	Success     int64   `json:"success"`
	Failed      int64   `json:"failed"`
	AvgDuration float64 `json:"avg_duration_ms"`
}
