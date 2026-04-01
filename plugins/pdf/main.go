package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-rod/rod/lib/proto"
	spectra "github.com/spectra-browser/spectra/sdk/go"
)

type GenerateParams struct {
	spectra.BrowserOptions
	URL             string  `json:"url"`
	Landscape       bool    `json:"landscape"`
	PrintBackground bool    `json:"print_background"`
	MarginTop       float64 `json:"margin_top"`
	MarginBottom    float64 `json:"margin_bottom"`
	MarginLeft      float64 `json:"margin_left"`
	MarginRight     float64 `json:"margin_right"`
}

func main() {
	p := spectra.NewPlugin("pdf")

	p.Handle("generate", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p GenerateParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.URL == "" {
			return nil, fmt.Errorf("url is required")
		}

		s, err := spectra.OpenPage(p.URL, p.BrowserOptions)
		if err != nil {
			return nil, fmt.Errorf("open page: %w", err)
		}
		defer s.Close()

		reader, err := s.Page.PDF(&proto.PagePrintToPDF{
			Landscape:       p.Landscape,
			PrintBackground: p.PrintBackground,
			MarginTop:       &p.MarginTop,
			MarginBottom:    &p.MarginBottom,
			MarginLeft:      &p.MarginLeft,
			MarginRight:     &p.MarginRight,
		})
		if err != nil {
			return nil, fmt.Errorf("pdf: %w", err)
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read pdf: %w", err)
		}

		return map[string]interface{}{
			"data":       base64.StdEncoding.EncodeToString(data),
			"size_bytes": len(data),
		}, nil
	})

	p.Run()
}
