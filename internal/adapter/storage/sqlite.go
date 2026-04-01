package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
)

// SQLiteStore implements port.WebhookStore, port.JobStore using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS webhooks (
			id         TEXT PRIMARY KEY,
			event      TEXT NOT NULL,
			target_url TEXT NOT NULL,
			secret     TEXT,
			active     INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS schedules (
			id         TEXT PRIMARY KEY,
			cron_expr  TEXT NOT NULL,
			plugin     TEXT NOT NULL,
			method     TEXT NOT NULL,
			params     TEXT,
			enabled    INTEGER NOT NULL DEFAULT 1,
			last_run   TEXT,
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS jobs (
			id          TEXT PRIMARY KEY,
			plugin      TEXT NOT NULL,
			method      TEXT NOT NULL,
			params      TEXT,
			status      TEXT NOT NULL,
			result_data TEXT,
			result_err  TEXT,
			duration_ms INTEGER,
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS browser_sessions (
			id         TEXT PRIMARY KEY,
			profile_id TEXT,
			url        TEXT,
			title      TEXT,
			metadata   TEXT,
			created_at TEXT NOT NULL,
			last_used  TEXT NOT NULL,
			expires_at TEXT
		);
		CREATE TABLE IF NOT EXISTS browser_profiles (
			id           TEXT PRIMARY KEY,
			name         TEXT NOT NULL,
			user_agent   TEXT,
			locale       TEXT,
			timezone     TEXT,
			proxy_url    TEXT,
			stealth_level TEXT,
			extra_flags  TEXT,
			created_at   TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS action_cache (
			domain      TEXT NOT NULL,
			instruction TEXT NOT NULL,
			selector    TEXT NOT NULL,
			updated_at  TEXT NOT NULL,
			PRIMARY KEY (domain, instruction)
		);
		CREATE INDEX IF NOT EXISTS idx_jobs_created ON jobs(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_webhooks_event ON webhooks(event);
	`)
	return err
}

// --- WebhookStore ---

func (s *SQLiteStore) CreateWebhook(_ context.Context, sub *domain.WebhookSubscription) error {
	_, err := s.db.Exec(
		`INSERT INTO webhooks(id,event,target_url,secret,active,created_at) VALUES(?,?,?,?,?,?)`,
		sub.ID, sub.Event, sub.TargetURL, sub.Secret, boolInt(sub.Active), sub.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) DeleteWebhook(_ context.Context, id string) error {
	_, err := s.db.Exec(`DELETE FROM webhooks WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) ListWebhooks(_ context.Context) ([]domain.WebhookSubscription, error) {
	rows, err := s.db.Query(`SELECT id,event,target_url,secret,active,created_at FROM webhooks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhooks(rows)
}

func (s *SQLiteStore) GetByEvent(_ context.Context, event domain.WebhookEvent) ([]domain.WebhookSubscription, error) {
	rows, err := s.db.Query(`SELECT id,event,target_url,secret,active,created_at FROM webhooks WHERE event=? AND active=1`, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhooks(rows)
}

func scanWebhooks(rows *sql.Rows) ([]domain.WebhookSubscription, error) {
	var list []domain.WebhookSubscription
	for rows.Next() {
		var sub domain.WebhookSubscription
		var active int
		var createdAt string
		if err := rows.Scan(&sub.ID, &sub.Event, &sub.TargetURL, &sub.Secret, &active, &createdAt); err != nil {
			return nil, err
		}
		sub.Active = active == 1
		sub.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		list = append(list, sub)
	}
	return list, rows.Err()
}

// --- JobStore ---

func (s *SQLiteStore) Save(_ context.Context, job *domain.Job, result *domain.JobResult) error {
	var dataStr, errStr string
	if result != nil {
		dataStr = string(result.Data)
		errStr = result.Error
	}
	params := string(job.Params)
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO jobs(id,plugin,method,params,status,result_data,result_err,duration_ms,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		job.ID, job.Plugin, job.Method, params, string(job.Status),
		dataStr, errStr, result.DurationMs,
		job.CreatedAt.Format(time.RFC3339), now,
	)
	return err
}

func (s *SQLiteStore) ListJobs(_ context.Context, limit int) ([]domain.JobRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(
		`SELECT id,plugin,method,params,status,result_data,result_err,duration_ms,created_at,updated_at
		 FROM jobs ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.JobRecord
	for rows.Next() {
		var r domain.JobRecord
		var params, resultData, resultErr, createdAt, updatedAt string
		var durationMs int64
		if err := rows.Scan(&r.ID, &r.Plugin, &r.Method, &params, &r.Status,
			&resultData, &resultErr, &durationMs, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		r.Params = json.RawMessage(params)
		r.Result = domain.JobResult{Data: json.RawMessage(resultData), Error: resultErr, DurationMs: durationMs}
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		list = append(list, r)
	}
	return list, rows.Err()
}

// --- ScheduleStore (part of SQLiteStore) ---

func (s *SQLiteStore) SaveSchedule(_ context.Context, task *domain.ScheduledTask) error {
	params := string(task.Params)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO schedules(id,cron_expr,plugin,method,params,enabled,created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		task.ID, task.CronExpr, task.Plugin, task.Method, params,
		boolInt(task.Enabled), task.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) DeleteSchedule(_ context.Context, id string) error {
	_, err := s.db.Exec(`DELETE FROM schedules WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) ListSchedules(_ context.Context) ([]domain.ScheduledTask, error) {
	rows, err := s.db.Query(`SELECT id,cron_expr,plugin,method,params,enabled,last_run,created_at FROM schedules`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.ScheduledTask
	for rows.Next() {
		var t domain.ScheduledTask
		var params, createdAt string
		var enabled int
		var lastRun sql.NullString
		if err := rows.Scan(&t.ID, &t.CronExpr, &t.Plugin, &t.Method, &params, &enabled, &lastRun, &createdAt); err != nil {
			return nil, err
		}
		t.Params = json.RawMessage(params)
		t.Enabled = enabled == 1
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if lastRun.Valid {
			lr, _ := time.Parse(time.RFC3339, lastRun.String)
			t.LastRun = &lr
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

func newID() string {
	return uuid.NewString()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// scanner is a common interface for *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// Compile-time interface checks — SQLiteStore directly implements all ports.
var (
	_ port.WebhookStore   = (*SQLiteStore)(nil)
	_ port.JobStore       = (*SQLiteStore)(nil)
	_ port.SessionManager = (*SQLiteStore)(nil)
	_ port.ProfileStore   = (*SQLiteStore)(nil)
	_ port.ActionCache    = (*SQLiteStore)(nil)
)

// --- SessionManager ---

func (s *SQLiteStore) CreateSession(_ context.Context, profileID string, ttlSeconds int) (*domain.BrowserSession, error) {
	id := newID()
	now := time.Now()
	var expiresAt *time.Time
	if ttlSeconds > 0 {
		t := now.Add(time.Duration(ttlSeconds) * time.Second)
		expiresAt = &t
	}
	var expiresStr interface{}
	if expiresAt != nil {
		expiresStr = expiresAt.Format(time.RFC3339)
	}
	_, err := s.db.Exec(
		`INSERT INTO browser_sessions(id,profile_id,url,title,metadata,created_at,last_used,expires_at) VALUES(?,?,?,?,?,?,?,?)`,
		id, profileID, "", "", "{}", now.Format(time.RFC3339), now.Format(time.RFC3339), expiresStr,
	)
	if err != nil {
		return nil, err
	}
	return &domain.BrowserSession{ID: id, ProfileID: profileID, CreatedAt: now, LastUsed: now, ExpiresAt: expiresAt}, nil
}

func (s *SQLiteStore) GetSession(_ context.Context, id string) (*domain.BrowserSession, error) {
	row := s.db.QueryRow(`SELECT id,profile_id,url,title,metadata,created_at,last_used,expires_at FROM browser_sessions WHERE id=?`, id)
	return scanSession(row)
}

func (s *SQLiteStore) ListSessions(_ context.Context) ([]domain.BrowserSession, error) {
	rows, err := s.db.Query(`SELECT id,profile_id,url,title,metadata,created_at,last_used,expires_at FROM browser_sessions ORDER BY last_used DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.BrowserSession
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *sess)
	}
	return list, rows.Err()
}

func (s *SQLiteStore) DeleteSession(_ context.Context, id string) error {
	_, err := s.db.Exec(`DELETE FROM browser_sessions WHERE id=?`, id)
	return err
}

func (s *SQLiteStore) TouchSession(_ context.Context, id, url, title string) error {
	_, err := s.db.Exec(`UPDATE browser_sessions SET url=?,title=?,last_used=? WHERE id=?`,
		url, title, time.Now().Format(time.RFC3339), id)
	return err
}

func scanSession(row scanner) (*domain.BrowserSession, error) {
	var sess domain.BrowserSession
	var profileID, url, title, metadata, createdAt, lastUsed string
	var expiresAt sql.NullString
	if err := row.Scan(&sess.ID, &profileID, &url, &title, &metadata, &createdAt, &lastUsed, &expiresAt); err != nil {
		return nil, err
	}
	sess.ProfileID = profileID
	sess.URL = url
	sess.Title = title
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	sess.LastUsed, _ = time.Parse(time.RFC3339, lastUsed)
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		sess.ExpiresAt = &t
	}
	json.Unmarshal([]byte(metadata), &sess.Metadata)
	return &sess, nil
}

// --- ProfileStore ---

func (s *SQLiteStore) CreateProfile(_ context.Context, p *domain.BrowserProfile) error {
	flags, _ := json.Marshal(p.ExtraFlags)
	_, err := s.db.Exec(
		`INSERT INTO browser_profiles(id,name,user_agent,locale,timezone,proxy_url,stealth_level,extra_flags,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.UserAgent, p.Locale, p.Timezone, p.ProxyURL, p.StealthLevel, string(flags), p.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetProfile(_ context.Context, id string) (*domain.BrowserProfile, error) {
	row := s.db.QueryRow(`SELECT id,name,user_agent,locale,timezone,proxy_url,stealth_level,extra_flags,created_at FROM browser_profiles WHERE id=?`, id)
	return scanProfile(row)
}

func (s *SQLiteStore) ListProfiles(_ context.Context) ([]domain.BrowserProfile, error) {
	rows, err := s.db.Query(`SELECT id,name,user_agent,locale,timezone,proxy_url,stealth_level,extra_flags,created_at FROM browser_profiles`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.BrowserProfile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

func (s *SQLiteStore) DeleteProfile(_ context.Context, id string) error {
	_, err := s.db.Exec(`DELETE FROM browser_profiles WHERE id=?`, id)
	return err
}

func scanProfile(row scanner) (*domain.BrowserProfile, error) {
	var p domain.BrowserProfile
	var flagsStr, createdAt string
	if err := row.Scan(&p.ID, &p.Name, &p.UserAgent, &p.Locale, &p.Timezone, &p.ProxyURL, &p.StealthLevel, &flagsStr, &createdAt); err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	json.Unmarshal([]byte(flagsStr), &p.ExtraFlags)
	return &p, nil
}

// --- ActionCache ---

func (s *SQLiteStore) GetCached(_ context.Context, domainKey, instruction string) (string, bool) {
	var selector string
	err := s.db.QueryRow(`SELECT selector FROM action_cache WHERE domain=? AND instruction=?`, domainKey, instruction).Scan(&selector)
	return selector, err == nil
}

func (s *SQLiteStore) SetCached(_ context.Context, domainKey, instruction, selector string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO action_cache(domain,instruction,selector,updated_at) VALUES(?,?,?,?)`,
		domainKey, instruction, selector, time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) ClearCache(_ context.Context, domainKey string) error {
	_, err := s.db.Exec(`DELETE FROM action_cache WHERE domain=?`, domainKey)
	return err
}
