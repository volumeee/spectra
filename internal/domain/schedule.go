package domain

import (
	"encoding/json"
	"time"
)

type ScheduledTask struct {
	ID        string          `json:"id"`
	CronExpr  string          `json:"cron"`
	Plugin    string          `json:"plugin"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
	Enabled   bool            `json:"enabled"`
	LastRun   *time.Time      `json:"last_run,omitempty"`
	NextRun   *time.Time      `json:"next_run,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
