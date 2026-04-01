package port

import (
	"context"

	"github.com/spectra-browser/spectra/internal/domain"
)

type Scheduler interface {
	Add(ctx context.Context, task *domain.ScheduledTask) error
	Remove(ctx context.Context, id string) error
	List(ctx context.Context) ([]domain.ScheduledTask, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
