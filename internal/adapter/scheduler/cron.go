package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
)

type CronScheduler struct {
	cron    *cron.Cron
	tasks   map[string]*taskEntry
	plugins port.PluginManager
	queue   port.JobQueue
	mu      sync.RWMutex
}

type taskEntry struct {
	task    domain.ScheduledTask
	entryID cron.EntryID
}

func NewCronScheduler(plugins port.PluginManager, queue port.JobQueue) *CronScheduler {
	return &CronScheduler{
		cron:    cron.New(),
		tasks:   make(map[string]*taskEntry),
		plugins: plugins,
		queue:   queue,
	}
}

func (s *CronScheduler) Add(_ context.Context, task *domain.ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	task.CreatedAt = time.Now()
	task.Enabled = true

	// Capture task ID for closure
	taskID := task.ID
	entryID, err := s.cron.AddFunc(task.CronExpr, func() {
		s.executeTask(taskID)
	})
	if err != nil {
		return err
	}

	s.tasks[task.ID] = &taskEntry{task: *task, entryID: entryID}
	slog.Info("scheduled task added", "id", task.ID, "cron", task.CronExpr, "plugin", task.Plugin)
	return nil
}

func (s *CronScheduler) Remove(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.tasks[id]
	if !ok {
		return nil
	}
	s.cron.Remove(entry.entryID)
	delete(s.tasks, id)
	return nil
}

func (s *CronScheduler) List(_ context.Context) ([]domain.ScheduledTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]domain.ScheduledTask, 0, len(s.tasks))
	for _, entry := range s.tasks {
		t := entry.task
		cronEntry := s.cron.Entry(entry.entryID)
		next := cronEntry.Next
		t.NextRun = &next
		list = append(list, t)
	}
	return list, nil
}

func (s *CronScheduler) Start(_ context.Context) error {
	s.cron.Start()
	slog.Info("scheduler started")
	return nil
}

func (s *CronScheduler) Stop(_ context.Context) error {
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("scheduler stopped")
	return nil
}

func (s *CronScheduler) executeTask(taskID string) {
	s.mu.RLock()
	entry, ok := s.tasks[taskID]
	if !ok {
		s.mu.RUnlock()
		return
	}
	task := entry.task
	s.mu.RUnlock()

	ctx := context.Background()
	job := &domain.Job{
		ID:        uuid.NewString(),
		Plugin:    task.Plugin,
		Method:    task.Method,
		Params:    task.Params,
		Status:    domain.JobStatusPending,
		CreatedAt: time.Now(),
	}

	_, err := s.queue.Enqueue(ctx, job, func(ctx context.Context, j *domain.Job) (*domain.JobResult, error) {
		return s.plugins.Execute(ctx, j.Plugin, j.Method, j.Params)
	})

	now := time.Now()
	s.mu.Lock()
	if e, ok := s.tasks[taskID]; ok {
		e.task.LastRun = &now
	}
	s.mu.Unlock()

	if err != nil {
		slog.Error("scheduled task failed", "id", taskID, "error", err)
	} else {
		slog.Info("scheduled task completed", "id", taskID)
	}
}
