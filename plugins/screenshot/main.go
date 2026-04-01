package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-rod/rod/lib/proto"
	spectra "github.com/spectra-browser/spectra/sdk/go"
)

type CaptureParams struct {
	spectra.BrowserOptions
	URL      string `json:"url"`
	FullPage bool   `json:"full_page"`
	Format   string `json:"format"`
	Quality  int    `json:"quality"`
}

func main() {
	p := spectra.NewPlugin("screenshot")

	p.Handle("capture", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p CaptureParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.URL == "" {
			return nil, fmt.Errorf("url is required")
		}
		if p.Format == "" {
			p.Format = "png"
		}
		if p.Quality == 0 {
			p.Quality = 90
		}

		s, err := spectra.OpenPage(p.URL, p.BrowserOptions)
		if err != nil {
			return nil, fmt.Errorf("open page: %w", err)
		}
		defer s.Close()

		quality := p.Quality
		req := &proto.PageCaptureScreenshot{
			Format:  proto.PageCaptureScreenshotFormatPng,
			Quality: &quality,
		}
		if p.Format == "jpeg" {
			req.Format = proto.PageCaptureScreenshotFormatJpeg
		}

		data, err := s.Page.Screenshot(p.FullPage, req)
		if err != nil {
			return nil, fmt.Errorf("screenshot: %w", err)
		}

		return map[string]interface{}{
			"data":       base64.StdEncoding.EncodeToString(data),
			"format":     p.Format,
			"width":      p.BrowserOptions.ViewportWidth(),
			"height":     p.BrowserOptions.ViewportHeight(),
			"size_bytes": len(data),
		}, nil
	})

	p.Run()
}
