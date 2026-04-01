package main

import (
	"bytes"
	"context"
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

// ---- Shared types ----

// AIParams is the base for all ai/* methods.
type AIParams struct {
	spectra.BrowserOptions
	SessionID string `json:"session_id,omitempty"` // reuse existing session
	OpenAIKey string `json:"openai_api_key"`
	BaseURL   string `json:"base_url,omitempty"` // default: OpenAI
	Model     string `json:"model,omitempty"`    // default: gpt-4o
}

func (p *AIParams) defaults() {
	if p.Model == "" {
		p.Model = "gpt-4o"
	}
	if p.BaseURL == "" {
		p.BaseURL = "https://api.openai.com/v1"
	}
}

// ---- act ----

type ActParams struct {
	AIParams
	Instruction string `json:"instruction"` // "click the login button"
}

// ---- extract ----

type ExtractParams struct {
	AIParams
	Instruction string                 `json:"instruction"` // "get all product prices"
	Schema      map[string]interface{} `json:"schema"`      // JSON schema for output
}

// ---- observe ----

type ObserveParams struct {
	AIParams
}

// ---- execute (full autonomous agent) ----

type ExecuteParams struct {
	AIParams
	Task     string        `json:"task"`
	URL      string        `json:"url,omitempty"`
	MaxSteps int           `json:"max_steps,omitempty"`
	Config   *AgentConfig  `json:"config,omitempty"`
}

type AgentConfig struct {
	Planning       bool `json:"planning"`        // generate plan before execute
	SelfCorrection bool `json:"self_correction"` // retry on error
	Memory         bool `json:"memory"`          // use action cache
	HumanInLoop    bool `json:"human_in_loop"`   // escalate if stuck
}

// ---- LLM client ----

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func callLLM(ctx context.Context, baseURL, apiKey, model string, messages []llmMessage, maxTokens int) (string, error) {
	if maxTokens <= 0 {
		maxTokens = 512
	}
	body, _ := json.Marshal(map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": 0.1,
		"max_tokens":  maxTokens,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API %d: %s", resp.StatusCode, string(data))
	}

	var result struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &result); err != nil || len(result.Choices) == 0 {
		return "", fmt.Errorf("parse LLM response: %w", err)
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// ---- Accessibility tree extraction ----

// getA11yTree returns a compact accessibility tree string for the current page.
// This is far more token-efficient than full DOM text.
func getA11yTree(page *rod.Page) string {
	result, err := page.Eval(`() => {
		function walk(node, depth) {
			if (!node || depth > 5) return '';
			const role = node.role?.value || '';
			const name = node.name?.value || '';
			const desc = node.description?.value || '';
			if (!role && !name) return '';
			let line = '  '.repeat(depth) + role;
			if (name) line += ' "' + name.slice(0, 60) + '"';
			if (desc) line += ' (' + desc.slice(0, 40) + ')';
			let children = '';
			if (node.children) {
				for (const child of node.children) {
					children += walk(child, depth + 1);
				}
			}
			return line + '\n' + children;
		}
		// Use CDP accessibility tree via chrome.automation if available
		// Fallback: build from DOM
		const interactive = [];
		document.querySelectorAll('button,a,input,select,textarea,[role],[tabindex]').forEach(el => {
			const tag = el.tagName.toLowerCase();
			const role = el.getAttribute('role') || tag;
			const name = el.getAttribute('aria-label') || el.textContent?.trim().slice(0, 60) || el.getAttribute('placeholder') || el.getAttribute('name') || '';
			const id = el.id ? '#' + el.id : '';
			const cls = el.className ? '.' + el.className.split(' ')[0] : '';
			const selector = id || cls || tag;
			if (name || role !== tag) {
				interactive.push(role + ' "' + name + '" [' + selector + ']');
			}
		});
		return interactive.slice(0, 50).join('\n');
	}`)
	if err != nil || result == nil {
		return "accessibility tree unavailable"
	}
	return result.Value.String()
}

// ---- act implementation ----

func handleAct(ctx context.Context, p ActParams, page *rod.Page, cache actionCache) (interface{}, error) {
	a11y := getA11yTree(page)
	pageURL, _ := page.Eval(`() => window.location.hostname`)
	domain := ""
	if pageURL != nil {
		domain = pageURL.Value.String()
	}

	// Check cache first
	if cached, ok := cache.get(domain, p.Instruction); ok {
		slog.Debug("ai/act cache hit", "instruction", p.Instruction, "selector", cached)
		el, err := page.Timeout(5 * time.Second).Element(cached)
		if err == nil {
			if err := el.Click(proto.InputMouseButtonLeft, 1); err == nil {
				return map[string]interface{}{
					"success":    true,
					"selector":   cached,
					"cache_hit":  true,
					"instruction": p.Instruction,
				}, nil
			}
		}
		// Cache miss on execution — fall through to LLM
	}

	prompt := fmt.Sprintf(`You are a browser automation assistant. Given the accessibility tree of a web page, return ONLY a CSS selector for the element that matches the instruction.

Instruction: %s

Accessibility tree:
%s

Return ONLY the CSS selector, nothing else. Example: button#login or input[name="email"]`, p.Instruction, a11y)

	selector, err := callLLM(ctx, p.BaseURL, p.OpenAIKey, p.Model, []llmMessage{
		{Role: "user", Content: prompt},
	}, 64)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	selector = strings.TrimSpace(strings.Trim(selector, "`\"'"))

	el, err := page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found with selector %q: %w", selector, err)
	}

	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("click failed: %w", err)
	}

	// Cache the learned selector
	cache.set(domain, p.Instruction, selector)

	return map[string]interface{}{
		"success":     true,
		"selector":    selector,
		"cache_hit":   false,
		"instruction": p.Instruction,
	}, nil
}

// ---- extract implementation ----

func handleExtract(ctx context.Context, p ExtractParams, page *rod.Page) (interface{}, error) {
	a11y := getA11yTree(page)
	schemaJSON, _ := json.Marshal(p.Schema)

	prompt := fmt.Sprintf(`You are a data extraction assistant. Extract structured data from the web page based on the instruction and schema.

Instruction: %s

Schema (JSON): %s

Accessibility tree:
%s

Also available: full page text below.

Return ONLY valid JSON matching the schema, nothing else.`, p.Instruction, string(schemaJSON), a11y)

	// Also get page text for extraction context
	pageText, _ := page.Eval(`() => document.body.innerText.slice(0, 4000)`)
	if pageText != nil {
		prompt += "\n\nPage text:\n" + pageText.Value.String()
	}

	result, err := callLLM(ctx, p.BaseURL, p.OpenAIKey, p.Model, []llmMessage{
		{Role: "user", Content: prompt},
	}, 1024)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	var extracted interface{}
	if err := json.Unmarshal([]byte(cleanJSON(result)), &extracted); err != nil {
		return map[string]interface{}{"raw": result}, nil
	}
	return extracted, nil
}

// ---- observe implementation ----

func handleObserve(ctx context.Context, p ObserveParams, page *rod.Page) (interface{}, error) {
	a11y := getA11yTree(page)

	prompt := fmt.Sprintf(`You are a browser automation assistant. List the possible actions a user can take on this page.

Return a JSON array of action objects: [{"action":"click","description":"...","selector":"..."}]
Include only interactive elements (buttons, links, inputs, forms).
Maximum 20 actions.

Accessibility tree:
%s`, a11y)

	result, err := callLLM(ctx, p.BaseURL, p.OpenAIKey, p.Model, []llmMessage{
		{Role: "user", Content: prompt},
	}, 512)
	if err != nil {
		return nil, fmt.Errorf("LLM error: %w", err)
	}

	var actions interface{}
	json.Unmarshal([]byte(cleanJSON(result)), &actions)

	return map[string]interface{}{
		"url":     evalStr(page, `() => window.location.href`),
		"title":   evalStr(page, `() => document.title`),
		"actions": actions,
		"a11y":    a11y,
	}, nil
}

// ---- execute (full autonomous agent) ----

type ActionLog struct {
	Step        int    `json:"step"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	CacheHit    bool   `json:"cache_hit,omitempty"`
	PageURL     string `json:"page_url"`
}

const agentSystemPrompt = `You are a browser automation agent. Complete the given task by controlling a web browser.

Respond with a single JSON action:
{"type":"navigate","value":"https://...","reason":"..."}
{"type":"act","instruction":"click the login button","reason":"..."}
{"type":"type","selector":"CSS_SELECTOR","value":"TEXT","reason":"..."}
{"type":"scroll","reason":"..."}
{"type":"extract","instruction":"get data","schema":{},"reason":"..."}
{"type":"wait","reason":"..."}
{"type":"done","result":"FINAL_RESULT","reason":"..."}

Rules:
1. Respond with valid JSON ONLY.
2. Use "act" for clicking/interacting — describe what to do in natural language.
3. Use "done" when task is complete with the result.
4. Be efficient — minimize steps.`

func handleExecute(ctx context.Context, p ExecuteParams, page *rod.Page, cache actionCache) (interface{}, error) {
	if p.MaxSteps <= 0 {
		p.MaxSteps = 20
	}
	cfg := p.Config
	if cfg == nil {
		cfg = &AgentConfig{Planning: true, SelfCorrection: true, Memory: true}
	}

	messages := []llmMessage{{Role: "system", Content: agentSystemPrompt}}
	var actionLog []ActionLog
	corrections := 0
	cacheHits := 0
	llmCalls := 0

	// Planning phase
	var plan []string
	if cfg.Planning {
		planPrompt := fmt.Sprintf("Task: %s\n\nCurrent page: %s\n\nCreate a numbered plan (max 10 steps). Return JSON array of strings.",
			p.Task, pageState(page))
		planResult, err := callLLM(ctx, p.BaseURL, p.OpenAIKey, p.Model, []llmMessage{
			{Role: "user", Content: planPrompt},
		}, 512)
		llmCalls++
		if err == nil {
			planResult = cleanJSON(planResult)
			json.Unmarshal([]byte(planResult), &plan)
		}
	}

	// Initial context
	messages = append(messages, llmMessage{
		Role:    "user",
		Content: fmt.Sprintf("Task: %s\n\nCurrent page: %s", p.Task, pageState(page)),
	})

	for step := 0; step < p.MaxSteps; step++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Get next action from LLM
		raw, err := callLLM(ctx, p.BaseURL, p.OpenAIKey, p.Model, messages, 256)
		llmCalls++
		if err != nil {
			return nil, fmt.Errorf("LLM error at step %d: %w", step, err)
		}

		raw = cleanJSON(raw)
		var action map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &action); err != nil {
			return nil, fmt.Errorf("parse action at step %d: %w", step, err)
		}

		actionType, _ := action["type"].(string)
		reason, _ := action["reason"].(string)
		log := ActionLog{Step: step, Type: actionType, Description: reason}

		var observation string
		var execErr error

		switch actionType {
		case "done":
			result, _ := action["result"].(string)
			log.Success = true
			actionLog = append(actionLog, log)
			return map[string]interface{}{
				"result":      result,
				"plan":        plan,
				"action_log":  actionLog,
				"total_steps": step + 1,
				"llm_calls":   llmCalls,
				"cache_hits":  cacheHits,
				"corrections": corrections,
				"task":        p.Task,
			}, nil

		case "navigate":
			val, _ := action["value"].(string)
			execErr = page.Navigate(val)
			if execErr == nil {
				page.MustWaitLoad()
				observation = "Navigated. " + pageState(page)
			}

		case "act":
			instruction, _ := action["instruction"].(string)
			actResult, err := handleAct(ctx, ActParams{
				AIParams:    p.AIParams,
				Instruction: instruction,
			}, page, cache)
			if err != nil {
				execErr = err
			} else {
				if r, ok := actResult.(map[string]interface{}); ok {
					if r["cache_hit"] == true {
						cacheHits++
					}
				}
				time.Sleep(300 * time.Millisecond)
				observation = "Action done. " + pageState(page)
			}

		case "type":
			sel, _ := action["selector"].(string)
			val, _ := action["value"].(string)
			el, err := page.Timeout(5 * time.Second).Element(sel)
			if err != nil {
				execErr = fmt.Errorf("element not found: %s", sel)
			} else {
				execErr = el.Input(val)
				if execErr == nil {
					observation = "Typed text."
				}
			}

		case "scroll":
			page.Mouse.Scroll(0, 400, 1)
			observation = "Scrolled."

		case "extract":
			instruction, _ := action["instruction"].(string)
			schema, _ := action["schema"].(map[string]interface{})
			extracted, err := handleExtract(ctx, ExtractParams{
				AIParams:    p.AIParams,
				Instruction: instruction,
				Schema:      schema,
			}, page)
			if err != nil {
				execErr = err
			} else {
				b, _ := json.Marshal(extracted)
				observation = "Extracted: " + string(b)[:min(200, len(string(b)))]
			}

		case "wait":
			time.Sleep(1 * time.Second)
			observation = "Waited."

		default:
			execErr = fmt.Errorf("unknown action: %s", actionType)
		}

		log.PageURL = pageState(page)
		if execErr != nil {
			log.Success = false
			log.Error = execErr.Error()
			observation = "Error: " + execErr.Error()
			if cfg.SelfCorrection {
				corrections++
				observation += " Please try a different approach."
			}
		} else {
			log.Success = true
		}
		actionLog = append(actionLog, log)

		messages = append(messages,
			llmMessage{Role: "assistant", Content: raw},
			llmMessage{Role: "user", Content: "Result: " + observation},
		)
	}

	return nil, fmt.Errorf("max steps (%d) reached", p.MaxSteps)
}

// ---- action cache (in-memory, per-process) ----
// For persistent cache, wire to SQLiteStore.ActionCache

type actionCache struct {
	m map[string]string
}

func newActionCache() actionCache {
	return actionCache{m: make(map[string]string)}
}

func (c actionCache) get(domain, instruction string) (string, bool) {
	v, ok := c.m[domain+"|"+instruction]
	return v, ok
}

func (c actionCache) set(domain, instruction, selector string) {
	c.m[domain+"|"+instruction] = selector
}

// ---- helpers ----

// evalStr safely evaluates a JS expression and returns the string result.
func evalStr(page *rod.Page, js string) string {
	v, err := page.Eval(js)
	if err != nil || v == nil {
		return ""
	}
	return v.Value.String()
}

func pageState(page *rod.Page) string {
	return fmt.Sprintf("URL: %s | Title: %s",
		evalStr(page, `() => window.location.href`),
		evalStr(page, `() => document.title`),
	)
}

func cleanJSON(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---- main ----

func main() {
	p := spectra.NewPlugin("ai")
	cache := newActionCache()

	// act — single action via accessibility tree + LLM + cache
	p.Handle("act", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p ActParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.Instruction == "" {
			return nil, fmt.Errorf("instruction is required")
		}
		if p.OpenAIKey == "" {
			return nil, fmt.Errorf("openai_api_key is required")
		}
		p.defaults()

		s, err := spectra.OpenBlankPage(p.BrowserOptions)
		if err != nil {
			return nil, err
		}
		defer s.Close()

		return handleAct(ctx, p, s.Page, cache)
	})

	// extract — structured data extraction via schema + LLM
	p.Handle("extract", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p ExtractParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.OpenAIKey == "" {
			return nil, fmt.Errorf("openai_api_key is required")
		}
		p.defaults()

		s, err := spectra.OpenBlankPage(p.BrowserOptions)
		if err != nil {
			return nil, err
		}
		defer s.Close()

		return handleExtract(ctx, p, s.Page)
	})

	// observe — list possible actions on current page (no execute)
	p.Handle("observe", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p ObserveParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.OpenAIKey == "" {
			return nil, fmt.Errorf("openai_api_key is required")
		}
		p.defaults()

		s, err := spectra.OpenBlankPage(p.BrowserOptions)
		if err != nil {
			return nil, err
		}
		defer s.Close()

		return handleObserve(ctx, p, s.Page)
	})

	// execute — full autonomous agent with planning, memory, self-correction
	p.Handle("execute", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p ExecuteParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.Task == "" {
			return nil, fmt.Errorf("task is required")
		}
		if p.OpenAIKey == "" {
			return nil, fmt.Errorf("openai_api_key is required")
		}
		p.defaults()

		s, err := spectra.OpenBlankPage(p.BrowserOptions)
		if err != nil {
			return nil, err
		}
		defer s.Close()

		if p.URL != "" {
			if err := s.Page.Navigate(p.URL); err != nil {
				return nil, fmt.Errorf("navigate: %w", err)
			}
			s.Page.MustWaitLoad()
		}

		return handleExecute(ctx, p, s.Page, cache)
	})

	// plan — generate execution plan without running (dry run)
	p.Handle("plan", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p ExecuteParams
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if p.Task == "" {
			return nil, fmt.Errorf("task is required")
		}
		if p.OpenAIKey == "" {
			return nil, fmt.Errorf("openai_api_key is required")
		}
		p.defaults()

		prompt := fmt.Sprintf(`Task: %s

Create a detailed step-by-step plan to complete this browser automation task.
Return a JSON object: {"plan": ["step 1", "step 2", ...], "estimated_steps": N, "complexity": "low|medium|high"}`, p.Task)

		result, err := callLLM(ctx, p.BaseURL, p.OpenAIKey, p.Model, []llmMessage{
			{Role: "user", Content: prompt},
		}, 512)
		if err != nil {
			return nil, err
		}

		result = cleanJSON(result)
		var plan interface{}
		json.Unmarshal([]byte(result), &plan)
		return plan, nil
	})

	p.Run()
}
