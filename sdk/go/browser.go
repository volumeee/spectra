package spectra

import (
	"os"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/proto"
)

// BrowserOptions configures browser launch/connect behavior.
// All fields are optional — sensible defaults are applied.
type BrowserOptions struct {
	// CDPEndpoint is injected by Spectra core when share_pool=true.
	// If set, connects to existing browser instead of launching a new one.
	CDPEndpoint string `json:"_cdp_endpoint,omitempty"`

	// Width/Height set the viewport. Defaults to 1920x1080.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`

	// Headless controls headless mode. Defaults to true.
	Headless *bool `json:"headless,omitempty"`

	// ExtraFlags are additional Chromium flags.
	ExtraFlags map[string]string `json:"extra_flags,omitempty"`
}

// IsHeadless returns the effective headless setting (default: true).
func (o BrowserOptions) IsHeadless() bool {
	if o.Headless != nil {
		return *o.Headless
	}
	return true
}

// ViewportWidth returns effective width (default: 1920).
func (o BrowserOptions) ViewportWidth() int {
	if o.Width > 0 {
		return o.Width
	}
	return 1920
}

// ViewportHeight returns effective height (default: 1080).
func (o BrowserOptions) ViewportHeight() int {
	if o.Height > 0 {
		return o.Height
	}
	return 1080
}

// BrowserSession wraps a browser + page, handling ownership for cleanup.
type BrowserSession struct {
	Browser  *rod.Browser
	Page     *rod.Page
	owned    bool // true = we launched the browser, we close it
}

// Close closes the page and, if we own the browser, closes it too.
func (s *BrowserSession) Close() {
	if s.Page != nil {
		_ = s.Page.Close()
	}
	if s.owned && s.Browser != nil {
		s.Browser.MustClose()
	}
}

// OpenPage acquires a browser (from pool or new launch) and opens a page at the given URL.
// Viewport is set to opts.ViewportWidth() x opts.ViewportHeight().
// Caller must call session.Close() when done.
func OpenPage(url string, opts BrowserOptions) (*BrowserSession, error) {
	browser, owned, err := connectOrLaunchBrowser(opts)
	if err != nil {
		return nil, err
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		if owned {
			browser.MustClose()
		}
		return nil, err
	}

	_ = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  opts.ViewportWidth(),
		Height: opts.ViewportHeight(),
	})

	page.MustWaitLoad()

	return &BrowserSession{Browser: browser, Page: page, owned: owned}, nil
}

// OpenBlankPage acquires a browser and opens a blank page (for navigation later).
func OpenBlankPage(opts BrowserOptions) (*BrowserSession, error) {
	browser, owned, err := connectOrLaunchBrowser(opts)
	if err != nil {
		return nil, err
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		if owned {
			browser.MustClose()
		}
		return nil, err
	}

	_ = page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:  opts.ViewportWidth(),
		Height: opts.ViewportHeight(),
	})

	return &BrowserSession{Browser: browser, Page: page, owned: owned}, nil
}

// ConnectOrLaunch is the exported low-level escape hatch for plugins that need
// direct browser access: multi-page sessions, CDP events, network interception, etc.
// Returns (browser, owned, error) — if owned=true, caller must close the browser.
//
// Example:
//
//	browser, owned, err := spectra.ConnectOrLaunch(opts)
//	defer func() { if owned { browser.MustClose() } }()
//	page1, _ := browser.Page(proto.TargetCreateTarget{URL: "https://a.com"})
//	page2, _ := browser.Page(proto.TargetCreateTarget{URL: "https://b.com"})
func ConnectOrLaunch(opts BrowserOptions) (*rod.Browser, bool, error) {
	return connectOrLaunchBrowser(opts)
}

// connectOrLaunchBrowser is the internal implementation.
func connectOrLaunchBrowser(opts BrowserOptions) (*rod.Browser, bool, error) {
	if opts.CDPEndpoint != "" {
		b := rod.New().ControlURL(opts.CDPEndpoint)
		if err := b.Connect(); err == nil {
			return b, false, nil
		}
		// Fall through to launch if connect fails
	}

	headless := opts.IsHeadless()
	w, h := opts.ViewportWidth(), opts.ViewportHeight()

	l := launcher.New().
		Headless(headless).
		NoSandbox(true).
		Set("disable-gpu").
		Set("disable-dev-shm-usage")

	if !headless {
		l = l.Set("window-size", formatSize(w, h))
	}

	for k, v := range opts.ExtraFlags {
		l = l.Set(flags.Flag(k), v)
	}

	l.Logger(os.Stderr)

	u, err := l.Launch()
	if err != nil {
		return nil, true, err
	}

	b := rod.New().ControlURL(u)
	return b, true, b.Connect()
}

func formatSize(w, h int) string {
	return itoa(w) + "," + itoa(h)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
