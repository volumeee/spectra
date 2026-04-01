package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	spectra "github.com/spectra-browser/spectra/sdk/go"
)

type ExecuteParams struct {
	spectra.BrowserOptions
	Task      string `json:"task"`
	URL       string `json:"url,omitempty"`
	OpenAIKey string `json:"openai_api_key"`
	BaseURL   string `json:"base_url,omitempty"`
	Model     string `json:"model,omitempty"`
	MaxSteps  int    `json:"max_steps,omitempty"`
}

// Action is what the LLM decides to do next.
type Action struct {
	Type     string `json:"type"`               // navigate|click|type|scroll|read_dom|screenshot|wait|done
	Selector string `json:"selector,omitempty"` // CSS selector for click/type
	Value    string `json:"value,omitempty"`    // URL for navigate, text for type, result for done
	Reason   string `json:"reason,omitempty"`   // LLM's reasoning
}

type ActionLog struct {
	Step     int    `json:"step"`
	Action   Action `json:"action"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	PageURL  string `json:"page_url"`
	PageTitle string `json:"page_title"`
}

const systemPrompt = `You are a browser automation agent. You control a web browser to complete tasks.

You respond with a single JSON action object. Available actions:
- {"type":"navigate","value":"https://...","reason":"..."}
- {"type":"click","selector":"CSS_SELECTOR","reason":"..."}
- {"type":"type","selector":"CSS_SELECTOR","value":"TEXT","reason":"..."}
- {"type":"scroll","reason":"..."}
- {"type":"read_dom","reason":"..."}
- {"type":"screenshot","reason":"..."}
- {"type":"wait","reason":"..."}
- {"type":"done","value":"RESULT","reason":"..."}

Rules:
1. Always respond with valid JSON only — no markdown, no explanation outside JSON.
2. Use "done" when the task is complete, with the result in "value".
3. Use specific CSS selectors (id, class, input[name=...]).
4. After each action you will receive the current page state.
5. Maximum steps will be enforced externally.`

func main() {
	p := spectra.NewPlugin("ai-browse")

	p.Handle("execute", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var ep ExecuteParams
		if err := json.Unmarshal(params, &ep); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if ep.Task == "" {
			return nil, fmt.Errorf("task is required")
		}
		if ep.OpenAIKey == "" {
			return nil, fmt.Errorf("openai_api_key is required")
		}
		if ep.Model == "" {
			ep.Model = "gpt-4o"
		}
		if ep.MaxSteps <= 0 {
			ep.MaxSteps = 20
		}
		if ep.BaseURL == "" {
			ep.BaseURL = "https://api.openai.com/v1"
		}

		s, err := spectra.OpenBlankPage(ep.BrowserOptions)
		if err != nil {
			return nil, fmt.Errorf("browser: %w", err)
		}
		defer s.Close()

		if ep.URL != "" {
			if err := s.Page.Navigate(ep.URL); err != nil {
				return nil, fmt.Errorf("navigate to start URL: %w", err)
			}
			s.Page.MustWaitLoad()
		}

		agent := &agentLoop{
			page:     s.Page,
			ep:       ep,
			client:   &http.Client{Timeout: 30 * time.Second},
			messages: []llmMessage{{Role: "system", Content: systemPrompt}},
		}

		result, actionLog, err := agent.run(ctx)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"result":      result,
			"action_log":  actionLog,
			"total_steps": len(actionLog),
			"task":        ep.Task,
			"model":       ep.Model,
		}, nil
	})

	p.Run()
}

type agentLoop struct {
	page     *rod.Page
	ep       ExecuteParams
	client   *http.Client
	messages []llmMessage
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (a *agentLoop) run(ctx context.Context) (string, []ActionLog, error) {
	var actionLog []ActionLog

	// Initial user message
	a.messages = append(a.messages, llmMessage{
		Role:    "user",
		Content: fmt.Sprintf("Task: %s\n\nCurrent page: %s", a.ep.Task, a.pageState()),
	})

	for step := 0; step < a.ep.MaxSteps; step++ {
		select {
		case <-ctx.Done():
			return "", actionLog, ctx.Err()
		default:
		}

		// Ask LLM for next action
		action, err := a.askLLM(ctx)
		if err != nil {
			return "", actionLog, fmt.Errorf("LLM error at step %d: %w", step, err)
		}

		slog.Debug("ai-browse action", "step", step, "type", action.Type, "reason", action.Reason)

		// Execute action
		pageURL := a.pageState()
		pageTitle := ""
		if t, err := a.page.Eval(`() => document.title`); err == nil {
			pageTitle = t.Value.String()
		}

		var execErr error
		var observation string

		switch action.Type {
		case "done":
			actionLog = append(actionLog, ActionLog{Step: step, Action: *action, Success: true, PageURL: pageURL, PageTitle: pageTitle})
			return action.Value, actionLog, nil

		case "navigate":
			execErr = a.page.Navigate(action.Value)
			if execErr == nil {
				a.page.MustWaitLoad()
				observation = "Navigated to " + action.Value + ". " + a.pageState()
			}

		case "click":
			el, err := a.page.Timeout(5 * time.Second).Element(action.Selector)
			if err != nil {
				execErr = fmt.Errorf("element not found: %s", action.Selector)
			} else {
				execErr = el.Click(proto.InputMouseButtonLeft, 1)
				if execErr == nil {
					time.Sleep(500 * time.Millisecond)
					observation = "Clicked. " + a.pageState()
				}
			}

		case "type":
			el, err := a.page.Timeout(5 * time.Second).Element(action.Selector)
			if err != nil {
				execErr = fmt.Errorf("element not found: %s", action.Selector)
			} else {
				execErr = el.Input(action.Value)
				if execErr == nil {
					observation = "Typed text."
				}
			}

		case "scroll":
			a.page.Mouse.Scroll(0, 400, 1)
			observation = "Scrolled down."

		case "read_dom":
			dom, err := a.page.Eval(`() => document.body.innerText.slice(0, 3000)`)
			if err != nil {
				execErr = err
			} else {
				observation = "DOM content:\n" + dom.Value.String()
			}

		case "screenshot":
			snap, err := a.page.Screenshot(false, nil)
			if err != nil {
				execErr = err
			} else {
				observation = "Screenshot taken. Size: " + fmt.Sprintf("%d bytes", len(snap))
				_ = base64.StdEncoding.EncodeToString(snap) // available if needed
			}

		case "wait":
			time.Sleep(1 * time.Second)
			observation = "Waited 1 second."

		default:
			execErr = fmt.Errorf("unknown action type: %s", action.Type)
		}

		logEntry := ActionLog{
			Step: step, Action: *action,
			Success: execErr == nil,
			PageURL: pageURL, PageTitle: pageTitle,
		}
		if execErr != nil {
			logEntry.Error = execErr.Error()
			observation = "Error: " + execErr.Error()
		}
		actionLog = append(actionLog, logEntry)

		// Feed observation back to LLM
		if observation == "" {
			observation = a.pageState()
		}
		a.messages = append(a.messages, llmMessage{
			Role:    "assistant",
			Content: mustMarshal(action),
		})
		a.messages = append(a.messages, llmMessage{
			Role:    "user",
			Content: "Result: " + observation,
		})
	}

	return "", actionLog, fmt.Errorf("max steps (%d) reached without completing task", a.ep.MaxSteps)
}

func (a *agentLoop) askLLM(ctx context.Context) (*Action, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"model":       a.ep.Model,
		"messages":    a.messages,
		"temperature": 0.1,
		"max_tokens":  256,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(a.ep.BaseURL, "/")+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.ep.OpenAIKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(data))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("empty LLM response")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	// Strip markdown code blocks if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var action Action
	if err := json.Unmarshal([]byte(content), &action); err != nil {
		return nil, fmt.Errorf("parse action JSON: %w (content: %s)", err, content)
	}
	return &action, nil
}

func (a *agentLoop) pageState() string {
	url, _ := a.page.Eval(`() => window.location.href`)
	title, _ := a.page.Eval(`() => document.title`)
	if url == nil || title == nil {
		return "unknown page"
	}
	return fmt.Sprintf("URL: %s | Title: %s", url.Value.String(), title.Value.String())
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
