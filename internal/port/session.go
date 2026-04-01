package port

import (
	"context"

	"github.com/spectra-browser/spectra/internal/domain"
)

// SessionManager manages long-lived named browser sessions.
type SessionManager interface {
	CreateSession(ctx context.Context, profileID string, ttlSeconds int) (*domain.BrowserSession, error)
	GetSession(ctx context.Context, id string) (*domain.BrowserSession, error)
	ListSessions(ctx context.Context) ([]domain.BrowserSession, error)
	DeleteSession(ctx context.Context, id string) error
	TouchSession(ctx context.Context, id, url, title string) error
}

// ProfileStore persists browser profiles (fingerprint identities).
type ProfileStore interface {
	CreateProfile(ctx context.Context, p *domain.BrowserProfile) error
	GetProfile(ctx context.Context, id string) (*domain.BrowserProfile, error)
	ListProfiles(ctx context.Context) ([]domain.BrowserProfile, error)
	DeleteProfile(ctx context.Context, id string) error
}

// ActionCache caches learned selectors per domain to reduce LLM calls.
type ActionCache interface {
	GetCached(ctx context.Context, domain, instruction string) (string, bool)
	SetCached(ctx context.Context, domain, instruction, selector string) error
	ClearCache(ctx context.Context, domain string) error
}
