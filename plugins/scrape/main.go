package main

import (
	"context"
	"encoding/json"
	"fmt"

	spectra "github.com/spectra-browser/spectra/sdk/go"
)

type ExtractParams struct {
	spectra.BrowserOptions
	URL       string            `json:"url"`
	Selectors map[string]string `json:"selectors"`
	WaitFor   string            `json:"wait_for"`
	ExecuteJS string            `json:"execute_js"`
}

func main() {
	p := spectra.NewPlugin("scrape")

	p.Handle("extract", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p ExtractParams
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

		if p.WaitFor != "" {
			s.Page.MustElement(p.WaitFor)
		}
		if p.ExecuteJS != "" {
			s.Page.MustEval(p.ExecuteJS)
		}

		title := s.Page.MustEval(`() => document.title`).String()
		text := s.Page.MustEval(`() => document.body.innerText`).String()
		description := s.Page.MustEval(`() => {
			const m = document.querySelector('meta[name="description"]');
			return m ? m.content : '';
		}`).String()

		links := s.Page.MustEval(`() => Array.from(document.querySelectorAll('a[href]')).map(a => a.href).slice(0, 100)`).Arr()
		images := s.Page.MustEval(`() => Array.from(document.querySelectorAll('img[src]')).map(i => i.src).slice(0, 50)`).Arr()

		linkStrs := make([]string, len(links))
		for i, l := range links {
			linkStrs[i] = l.String()
		}
		imgStrs := make([]string, len(images))
		for i, img := range images {
			imgStrs[i] = img.String()
		}

		meta := s.Page.MustEval(`() => {
			const metas = {};
			document.querySelectorAll('meta[property], meta[name]').forEach(m => {
				const key = m.getAttribute('property') || m.getAttribute('name');
				metas[key] = m.content;
			});
			return metas;
		}`).Map()
		metaMap := make(map[string]string)
		for k, v := range meta {
			metaMap[k] = v.String()
		}

		custom := make(map[string]string)
		for key, selector := range p.Selectors {
			if el, err := s.Page.Element(selector); err == nil {
				t, _ := el.Text()
				custom[key] = t
			}
		}

		return map[string]interface{}{
			"title":       title,
			"description": description,
			"text":        text,
			"links":       linkStrs,
			"images":      imgStrs,
			"meta":        metaMap,
			"custom":      custom,
		}, nil
	})

	p.Run()
}
