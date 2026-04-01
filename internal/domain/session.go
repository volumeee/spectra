package domain

import "time"

// BrowserSession represents a long-lived named browser session.
// Sessions persist state (cookies, localStorage) across multiple requests.
type BrowserSession struct {
	ID        string            `json:"id"`
	ProfileID string            `json:"profile_id,omitempty"`
	URL       string            `json:"url"`
	Title     string            `json:"title"`
	CreatedAt time.Time         `json:"created_at"`
	LastUsed  time.Time         `json:"last_used"`
	ExpiresAt *time.Time        `json:"expires_at,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// BrowserProfile is a persistent browser identity (UA, locale, timezone, fingerprint seed).
// Reusing a profile gives consistent fingerprinting across sessions.
type BrowserProfile struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	UserAgent  string            `json:"user_agent,omitempty"`
	Locale     string            `json:"locale,omitempty"`
	Timezone   string            `json:"timezone,omitempty"`
	ProxyURL   string            `json:"proxy_url,omitempty"`
	StealthLevel string          `json:"stealth_level,omitempty"` // "basic" | "advanced" | "maximum"
	ExtraFlags map[string]string `json:"extra_flags,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}
