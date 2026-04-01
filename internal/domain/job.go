package domain

import (
	"encoding/json"
	"time"
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

type Job struct {
	ID        string          `json:"id"`
	Plugin    string          `json:"plugin"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	Status    JobStatus       `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
}

type JobResult struct {
	Data       json.RawMessage `json:"data,omitempty"`
	Error      string          `json:"error,omitempty"`
	DurationMs int64           `json:"duration_ms"`
}

// JobRecord is a completed job stored for history/observability.
type JobRecord struct {
	Job
	Result    JobResult `json:"result"`
	UpdatedAt time.Time `json:"updated_at"`
}
