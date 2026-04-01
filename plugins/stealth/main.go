package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	spectra "github.com/spectra-browser/spectra/sdk/go"
)

var stealthScripts = []string{
	`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`,
	`Object.defineProperty(navigator, 'plugins', {get: () => [1,2,3,4,5]})`,
	`Object.defineProperty(navigator, 'languages', {get: () => ['en-US','en']})`,
	`window.chrome = {runtime: {}, loadTimes: function(){}, csi: function(){}}`,
	`const origQuery = window.navigator.permissions.query;
	 window.navigator.permissions.query = (params) => (
		params.name === 'notifications' ?
		Promise.resolve({state: Notification.permission}) :
		origQuery(params)
	 )`,
	`const getParameter = WebGLRenderingContext.prototype.getParameter;
	 WebGLRenderingContext.prototype.getParameter = function(param) {
		if (param === 37445) return 'Intel Inc.';
		if (param === 37446) return 'Intel Iris OpenGL Engine';
		return getParameter.call(this, param);
	 }`,
}

type StealthParams struct {
	spectra.BrowserOptions
	URL string `json:"url"`
}

// stealthOpts returns BrowserOptions with stealth Chromium flags injected.
func stealthOpts(base spectra.BrowserOptions) spectra.BrowserOptions {
	if base.ExtraFlags == nil {
		base.ExtraFlags = map[string]string{}
	}
	base.ExtraFlags["disable-blink-features"] = "AutomationControlled"
	return base
}

func main() {
	p := spectra.NewPlugin("stealth")

	p.Handle("navigate", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p StealthParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.URL == "" {
			return nil, fmt.Errorf("url is required")
		}

		s, err := spectra.OpenBlankPage(stealthOpts(p.BrowserOptions))
		if err != nil {
			return nil, fmt.Errorf("open browser: %w", err)
		}
		defer s.Close()

		for _, script := range stealthScripts {
			s.Page.MustEvalOnNewDocument(script)
		}
		if err := s.Page.Navigate(p.URL); err != nil {
			return nil, fmt.Errorf("navigate: %w", err)
		}
		s.Page.MustWaitLoad()

		title := s.Page.MustEval(`() => document.title`).String()
		webdriver := s.Page.MustEval(`() => navigator.webdriver`).String()

		return map[string]interface{}{
			"title":     title,
			"url":       p.URL,
			"webdriver": webdriver,
			"stealth":   true,
		}, nil
	})

	p.Handle("screenshot", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p StealthParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.URL == "" {
			return nil, fmt.Errorf("url is required")
		}

		s, err := spectra.OpenBlankPage(stealthOpts(p.BrowserOptions))
		if err != nil {
			return nil, fmt.Errorf("open browser: %w", err)
		}
		defer s.Close()

		for _, script := range stealthScripts {
			s.Page.MustEvalOnNewDocument(script)
		}
		s.Page.Navigate(p.URL)
		s.Page.MustWaitLoad()

		data, err := s.Page.Screenshot(false, nil)
		if err != nil {
			return nil, fmt.Errorf("screenshot: %w", err)
		}

		return map[string]interface{}{
			"data":       base64.StdEncoding.EncodeToString(data),
			"size_bytes": len(data),
			"width":      p.BrowserOptions.ViewportWidth(),
			"height":     p.BrowserOptions.ViewportHeight(),
			"stealth":    true,
		}, nil
	})

	p.Handle("scrape", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p StealthParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.URL == "" {
			return nil, fmt.Errorf("url is required")
		}

		s, err := spectra.OpenBlankPage(stealthOpts(p.BrowserOptions))
		if err != nil {
			return nil, fmt.Errorf("open browser: %w", err)
		}
		defer s.Close()

		for _, script := range stealthScripts {
			s.Page.MustEvalOnNewDocument(script)
		}
		s.Page.Navigate(p.URL)
		s.Page.MustWaitLoad()

		return map[string]interface{}{
			"title":   s.Page.MustEval(`() => document.title`).String(),
			"text":    s.Page.MustEval(`() => document.body.innerText`).String(),
			"stealth": true,
		}, nil
	})

	p.Run()
}
