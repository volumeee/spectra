package port

import (
	"context"

	"github.com/spectra-browser/spectra/internal/domain"
)

type WebhookStore interface {
	CreateWebhook(ctx context.Context, sub *domain.WebhookSubscription) error
	DeleteWebhook(ctx context.Context, id string) error
	ListWebhooks(ctx context.Context) ([]domain.WebhookSubscription, error)
	GetByEvent(ctx context.Context, event domain.WebhookEvent) ([]domain.WebhookSubscription, error)
}
