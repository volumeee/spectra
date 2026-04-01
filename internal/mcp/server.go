package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spectra-browser/spectra/internal/port"
)

func NewMCPServer(plugins port.PluginManager) *server.MCPServer {
	s := server.NewMCPServer("Spectra", "0.2.0", server.WithToolCapabilities(true))

	s.AddTool(mcp.Tool{
		Name:        "spectra_screenshot",
		Description: "Take a screenshot of a web page",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"url":       map[string]string{"type": "string", "description": "URL to screenshot"},
				"width":     map[string]interface{}{"type": "integer", "description": "Viewport width", "default": 1280},
				"height":    map[string]interface{}{"type": "integer", "description": "Viewport height", "default": 720},
				"full_page": map[string]interface{}{"type": "boolean", "description": "Full page screenshot", "default": false},
			},
			Required: []string{"url"},
		},
	}, toolHandler(plugins, "screenshot", "capture"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_pdf",
		Description: "Generate a PDF from a web page",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"url":       map[string]string{"type": "string", "description": "URL to convert to PDF"},
				"landscape": map[string]interface{}{"type": "boolean", "description": "Landscape orientation", "default": false},
			},
			Required: []string{"url"},
		},
	}, toolHandler(plugins, "pdf", "generate"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_scrape",
		Description: "Scrape and extract structured data from a web page",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"url":      map[string]string{"type": "string", "description": "URL to scrape"},
				"wait_for": map[string]string{"type": "string", "description": "CSS selector to wait for"},
			},
			Required: []string{"url"},
		},
	}, toolHandler(plugins, "scrape", "extract"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_record",
		Description: "Record a multi-step browser session with screenshots and video",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"url":         map[string]string{"type": "string", "description": "Starting URL"},
				"steps":       map[string]interface{}{"type": "array", "description": "Steps to execute"},
				"output_mode": map[string]interface{}{"type": "string", "description": "frames|screencast|both", "default": "both"},
			},
			Required: []string{"url"},
		},
	}, toolHandler(plugins, "recorder", "record"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_ai_act",
		Description: "Perform a single browser action using natural language (e.g. 'click the login button'). Uses accessibility tree for efficiency.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"instruction":    map[string]string{"type": "string", "description": "Natural language instruction"},
				"openai_api_key": map[string]string{"type": "string", "description": "OpenAI-compatible API key"},
				"session_id":     map[string]string{"type": "string", "description": "Optional: reuse existing session"},
			},
			Required: []string{"instruction", "openai_api_key"},
		},
	}, toolHandler(plugins, "ai", "act"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_ai_extract",
		Description: "Extract structured data from a web page using natural language and a JSON schema.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"instruction":    map[string]string{"type": "string", "description": "What to extract"},
				"schema":         map[string]string{"type": "object", "description": "JSON schema for output"},
				"openai_api_key": map[string]string{"type": "string", "description": "OpenAI-compatible API key"},
			},
			Required: []string{"instruction", "openai_api_key"},
		},
	}, toolHandler(plugins, "ai", "extract"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_ai_observe",
		Description: "List all possible actions on the current page without executing anything.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"openai_api_key": map[string]string{"type": "string", "description": "OpenAI-compatible API key"},
				"session_id":     map[string]string{"type": "string", "description": "Optional: session to observe"},
			},
			Required: []string{"openai_api_key"},
		},
	}, toolHandler(plugins, "ai", "observe"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_ai_execute",
		Description: "Full autonomous AI agent that browses the web to complete complex tasks. Supports planning, self-correction, and memory.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"task":           map[string]string{"type": "string", "description": "Natural language task description"},
				"url":            map[string]string{"type": "string", "description": "Optional starting URL"},
				"openai_api_key": map[string]string{"type": "string", "description": "OpenAI-compatible API key"},
				"model":          map[string]interface{}{"type": "string", "description": "LLM model", "default": "gpt-4o"},
				"max_steps":      map[string]interface{}{"type": "integer", "description": "Max agent steps", "default": 20},
			},
			Required: []string{"task", "openai_api_key"},
		},
	}, toolHandler(plugins, "ai", "execute"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_ai_plan",
		Description: "Generate an execution plan for a browser task without running it (dry run).",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"task":           map[string]string{"type": "string", "description": "Task to plan"},
				"openai_api_key": map[string]string{"type": "string", "description": "OpenAI-compatible API key"},
			},
			Required: []string{"task", "openai_api_key"},
		},
	}, toolHandler(plugins, "ai", "plan"))

	s.AddTool(mcp.Tool{
		Name:        "spectra_execute",
		Description: "Execute any Spectra plugin method",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"plugin": map[string]string{"type": "string", "description": "Plugin name"},
				"method": map[string]string{"type": "string", "description": "Method name"},
				"params": map[string]string{"type": "object", "description": "Method parameters"},
			},
			Required: []string{"plugin", "method"},
		},
	}, executeHandler(plugins))

	s.AddTool(mcp.Tool{
		Name:        "spectra_plugins",
		Description: "List all available Spectra plugins",
		InputSchema: mcp.ToolInputSchema{Type: "object", Properties: map[string]interface{}{}},
	}, listPluginsHandler(plugins))

	return s
}

func toolHandler(plugins port.PluginManager, pluginName, method string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params, _ := json.Marshal(req.Params.Arguments)
		result, err := plugins.Execute(ctx, pluginName, method, params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(result.Data)), nil
	}
}

func executeHandler(plugins port.PluginManager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments
		pluginName, _ := args["plugin"].(string)
		method, _ := args["method"].(string)
		if pluginName == "" || method == "" {
			return mcp.NewToolResultError("plugin and method are required"), nil
		}
		params, _ := json.Marshal(args["params"])
		result, err := plugins.Execute(ctx, pluginName, method, params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(result.Data)), nil
	}
}

func listPluginsHandler(plugins port.PluginManager) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		list := plugins.List()
		data, _ := json.MarshalIndent(list, "", "  ")
		return mcp.NewToolResultText(string(data)), nil
	}
}
