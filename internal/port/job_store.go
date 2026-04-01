package port

import (
	"context"

	"github.com/spectra-browser/spectra/internal/domain"
)

// JobStore persists job history for observability and audit.
type JobStore interface {
	Save(ctx context.Context, job *domain.Job, result *domain.JobResult) error
	ListJobs(ctx context.Context, limit int) ([]domain.JobRecord, error)
}
