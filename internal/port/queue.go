package port

import (
	"context"

	"github.com/spectra-browser/spectra/internal/domain"
)

type JobHandler func(ctx context.Context, job *domain.Job) (*domain.JobResult, error)

type JobEventCallback func(job *domain.Job, result *domain.JobResult)

type QueueStats struct {
	Running   int `json:"running"`
	Pending   int `json:"pending"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

type JobQueue interface {
	Enqueue(ctx context.Context, job *domain.Job, handler JobHandler) (*domain.JobResult, error)
	Stats() QueueStats
	OnJobCompleted(cb JobEventCallback)
	OnJobFailed(cb JobEventCallback)
	Shutdown(ctx context.Context) error
}
