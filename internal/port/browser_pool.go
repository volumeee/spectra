package port

import "context"

// BrowserSession represents a single acquired browser instance from the pool.
type BrowserSession interface {
	CDPEndpoint() string
	Close()
}

type PoolStats struct {
	Active int `json:"active"`
	Idle   int `json:"idle"`
	Total  int `json:"total"`
	Max    int `json:"max"`
}

// BrowserPool manages a pool of Chromium browser instances.
// Plugins can acquire a session to reuse an existing browser instead of launching a new one.
type BrowserPool interface {
	Acquire(ctx context.Context) (BrowserSession, error)
	Release(session BrowserSession)
	Stats() PoolStats
	Shutdown(ctx context.Context) error
}
