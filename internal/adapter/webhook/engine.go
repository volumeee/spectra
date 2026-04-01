package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
)

type Engine struct {
	store      port.WebhookStore
	maxRetries int
	retryInterval time.Duration
	client     *http.Client
}

func NewEngine(store port.WebhookStore, maxRetries int, retryInterval time.Duration) *Engine {
	return &Engine{
		store:         store,
		maxRetries:    maxRetries,
		retryInterval: retryInterval,
		client:        &http.Client{Timeout: 10 * time.Second},
	}
}

func (e *Engine) HandleJobCompleted(job *domain.Job, result *domain.JobResult) {
	e.dispatch(domain.WebhookEventJobCompleted, map[string]interface{}{
		"job_id": job.ID, "plugin": job.Plugin, "method": job.Method, "result": result,
	})
}

func (e *Engine) HandleJobFailed(job *domain.Job, result *domain.JobResult) {
	e.dispatch(domain.WebhookEventJobFailed, map[string]interface{}{
		"job_id": job.ID, "plugin": job.Plugin, "method": job.Method, "error": result.Error,
	})
}

func (e *Engine) dispatch(event domain.WebhookEvent, data interface{}) {
	ctx := context.Background()
	subs, err := e.store.GetByEvent(ctx, event)
	if err != nil || len(subs) == 0 {
		return
	}

	payload := domain.WebhookPayload{Event: event, Timestamp: time.Now(), Data: data}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, sub := range subs {
		go e.send(sub, body)
	}
}

func (e *Engine) send(sub domain.WebhookSubscription, body []byte) {
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(e.retryInterval * time.Duration(attempt))
		}

		req, err := http.NewRequest(http.MethodPost, sub.TargetURL, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		if sub.Secret != "" {
			mac := hmac.New(sha256.New, []byte(sub.Secret))
			mac.Write(body)
			sig := hex.EncodeToString(mac.Sum(nil))
			req.Header.Set("X-Spectra-Signature", sig)
		}

		resp, err := e.client.Do(req)
		if err != nil {
			slog.Warn("webhook delivery failed", "url", sub.TargetURL, "attempt", attempt, "error", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			slog.Info("webhook delivered", "url", sub.TargetURL, "event", sub.Event)
			return
		}
		slog.Warn("webhook non-2xx", "url", sub.TargetURL, "status", resp.StatusCode, "attempt", attempt)
	}
	slog.Error("webhook delivery exhausted retries", "url", sub.TargetURL)
}
