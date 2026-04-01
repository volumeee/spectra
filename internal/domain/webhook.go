package domain

import "time"

type WebhookEvent string

const (
	WebhookEventJobCompleted WebhookEvent = "job.completed"
	WebhookEventJobFailed    WebhookEvent = "job.failed"
	WebhookEventPluginCrash  WebhookEvent = "plugin.crashed"
)

type WebhookSubscription struct {
	ID        string       `json:"id"`
	Event     WebhookEvent `json:"event"`
	TargetURL string       `json:"target_url"`
	Secret    string       `json:"secret,omitempty"`
	Active    bool         `json:"active"`
	CreatedAt time.Time    `json:"created_at"`
}

type WebhookPayload struct {
	Event     WebhookEvent    `json:"event"`
	Timestamp time.Time       `json:"timestamp"`
	Data      interface{}     `json:"data"`
}
