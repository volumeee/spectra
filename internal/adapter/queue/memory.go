package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
)

type MemoryQueue struct {
	maxConcurrent int
	maxPending    int
	sem           chan struct{}
	monitor       port.SystemMonitor // optional health check
	running       atomic.Int32
	pending       atomic.Int32
	completed     atomic.Int64
	failed        atomic.Int64
	timedout      atomic.Int64
	onCompleted   []port.JobEventCallback
	onFailed      []port.JobEventCallback
	mu            sync.RWMutex
	wg            sync.WaitGroup
}

func NewMemoryQueue(maxConcurrent, maxPending int, monitor port.SystemMonitor) *MemoryQueue {
	return &MemoryQueue{
		maxConcurrent: maxConcurrent,
		maxPending:    maxPending,
		sem:           make(chan struct{}, maxConcurrent),
		monitor:       monitor,
	}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job *domain.Job, handler port.JobHandler) (*domain.JobResult, error) {
	// Health check: reject if system is overloaded
	if q.monitor != nil {
		if overloaded, reason := q.monitor.Overloaded(); overloaded {
			slog.Warn("rejecting job: system overloaded", "reason", reason, "job_id", job.ID)
			return nil, fmt.Errorf("%w: %s", domain.ErrSystemOverloaded, reason)
		}
	}

	if int(q.pending.Load()) >= q.maxPending {
		return nil, domain.ErrQueueFull
	}

	q.pending.Add(1)
	defer q.pending.Add(-1)

	select {
	case q.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	q.wg.Add(1)
	defer func() {
		<-q.sem
		q.wg.Done()
	}()

	q.running.Add(1)
	job.Status = domain.JobStatusRunning

	result, err := handler(ctx, job)

	q.running.Add(-1)

	if err != nil {
		job.Status = domain.JobStatusFailed
		q.failed.Add(1)
		q.fireCallbacks(q.onFailed, job, result)
		return result, err
	}

	job.Status = domain.JobStatusCompleted
	q.completed.Add(1)
	q.fireCallbacks(q.onCompleted, job, result)
	return result, nil
}

func (q *MemoryQueue) Stats() port.QueueStats {
	return port.QueueStats{
		Running:   int(q.running.Load()),
		Pending:   int(q.pending.Load()),
		Completed: int(q.completed.Load()),
		Failed:    int(q.failed.Load()),
	}
}

func (q *MemoryQueue) OnJobCompleted(cb port.JobEventCallback) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.onCompleted = append(q.onCompleted, cb)
}

func (q *MemoryQueue) OnJobFailed(cb port.JobEventCallback) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.onFailed = append(q.onFailed, cb)
}

func (q *MemoryQueue) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		slog.Info("queue drained")
	case <-ctx.Done():
		slog.Warn("queue shutdown timeout, some jobs may be lost")
	}
	return nil
}

func (q *MemoryQueue) fireCallbacks(cbs []port.JobEventCallback, job *domain.Job, result *domain.JobResult) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	for _, cb := range cbs {
		go cb(job, result)
	}
}
