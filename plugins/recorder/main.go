package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	spectra "github.com/spectra-browser/spectra/sdk/go"
)

type RecordParams struct {
	spectra.BrowserOptions
	URL        string `json:"url"`
	Steps      []Step `json:"steps"`
	OutputMode string `json:"output_mode"` // "frames" | "both"
}

type Step struct {
	Action   string `json:"action"`
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`
	Delay    int    `json:"delay,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
	OnError  string `json:"on_error,omitempty"` // "continue" | "stop"
}

type StepResult struct {
	Step        int    `json:"step"`
	Action      string `json:"action"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	Screenshot  string `json:"screenshot,omitempty"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	TimestampMs int64  `json:"timestamp_ms"`
}

func main() {
	p := spectra.NewPlugin("recorder")

	p.Handle("record", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var rp RecordParams
		if err := json.Unmarshal(params, &rp); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if rp.URL == "" {
			return nil, fmt.Errorf("url is required")
		}
		if rp.OutputMode == "" {
			rp.OutputMode = "both"
		}

		s, err := spectra.OpenPage(rp.URL, rp.BrowserOptions)
		if err != nil {
			return nil, fmt.Errorf("open page: %w", err)
		}
		defer s.Close()

		start := time.Now()
		var stepResults []StepResult
		var allFrames []string

		snap, _ := s.Page.Screenshot(false, nil)
		stepResults = append(stepResults, StepResult{
			Step: 0, Action: "initial", Success: true,
			Screenshot:  base64.StdEncoding.EncodeToString(snap),
			URL:         s.Page.MustEval(`() => window.location.href`).String(),
			Title:       s.Page.MustEval(`() => document.title`).String(),
			TimestampMs: time.Since(start).Milliseconds(),
		})
		if rp.OutputMode == "frames" || rp.OutputMode == "both" {
			allFrames = append(allFrames, base64.StdEncoding.EncodeToString(snap))
		}

		for i, step := range rp.Steps {
			sr := StepResult{Step: i + 1, Action: step.Action}
			if stepErr := executeStep(s.Page, step); stepErr != nil {
				sr.Success = false
				sr.Error = stepErr.Error()
				if errSnap, err := s.Page.Screenshot(false, nil); err == nil {
					sr.Screenshot = base64.StdEncoding.EncodeToString(errSnap)
				}
				stepResults = append(stepResults, sr)
				if step.OnError != "continue" {
					break
				}
			} else {
				sr.Success = true
			}

			if step.Delay > 0 {
				time.Sleep(time.Duration(step.Delay) * time.Millisecond)
			}

			snap, _ = s.Page.Screenshot(false, nil)
			sr.Screenshot = base64.StdEncoding.EncodeToString(snap)
			sr.URL = s.Page.MustEval(`() => window.location.href`).String()
			sr.Title = s.Page.MustEval(`() => document.title`).String()
			sr.TimestampMs = time.Since(start).Milliseconds()
			stepResults = append(stepResults, sr)

			if rp.OutputMode == "frames" || rp.OutputMode == "both" {
				allFrames = append(allFrames, base64.StdEncoding.EncodeToString(snap))
			}
		}

		result := map[string]interface{}{
			"steps":       stepResults,
			"total_steps": len(rp.Steps),
			"duration_ms": time.Since(start).Milliseconds(),
			"output_mode": rp.OutputMode,
			"width":       rp.BrowserOptions.ViewportWidth(),
			"height":      rp.BrowserOptions.ViewportHeight(),
		}
		if rp.OutputMode == "frames" || rp.OutputMode == "both" {
			result["frames"] = allFrames
			result["frame_count"] = len(allFrames)
		}
		return result, nil
	})

	p.Run()
}

func executeStep(page *rod.Page, step Step) error {
	timeout := time.Duration(step.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	switch step.Action {
	case "navigate":
		if err := page.Navigate(step.Value); err != nil {
			return err
		}
		page.MustWaitLoad()
	case "click":
		el, err := page.Timeout(timeout).Element(step.Selector)
		if err != nil {
			return fmt.Errorf("element not found: %s", step.Selector)
		}
		return el.Click(proto.InputMouseButtonLeft, 1)
	case "type":
		el, err := page.Timeout(timeout).Element(step.Selector)
		if err != nil {
			return fmt.Errorf("element not found: %s", step.Selector)
		}
		return el.Input(step.Value)
	case "hover":
		el, err := page.Timeout(timeout).Element(step.Selector)
		if err != nil {
			return fmt.Errorf("element not found: %s", step.Selector)
		}
		return el.Hover()
	case "select":
		el, err := page.Timeout(timeout).Element(step.Selector)
		if err != nil {
			return fmt.Errorf("element not found: %s", step.Selector)
		}
		return el.Select([]string{step.Value}, true, rod.SelectorTypeText)
	case "evaluate_js":
		_, err := page.Eval(step.Value)
		return err
	case "wait_for":
		_, err := page.Timeout(timeout).Element(step.Selector)
		return err
	case "assert_text":
		el, err := page.Timeout(timeout).Element(step.Selector)
		if err != nil {
			return fmt.Errorf("element not found: %s", step.Selector)
		}
		text, err := el.Text()
		if err != nil {
			return err
		}
		if text != step.Value {
			return fmt.Errorf("assert failed: expected %q, got %q", step.Value, text)
		}
	case "scroll":
		page.Mouse.Scroll(0, 300, 1)
	case "wait", "screenshot":
		if step.Delay > 0 {
			time.Sleep(time.Duration(step.Delay) * time.Millisecond)
		}
	}
	return nil
}
