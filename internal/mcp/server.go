package mcp

import (
	"bufio"
	"encoding/json"
	"io"
)

type rpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

var tools = []map[string]any{
	{
		"name":        "reproduce_ci_failure",
		"description": "Extract a minimal 1-line local repro command, working directory, env, and cleaned error tail from a failing GitHub Actions run (read-only, never executes).",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"run_id":       map[string]any{"type": "integer", "description": "GitHub Actions workflow run ID"},
				"pr":           map[string]any{"type": "integer", "description": "Optional PR number to inspect failing checks"},
				"repo":         map[string]any{"type": "string", "description": "Optional OWNER/REPO"},
				"log_file":     map[string]any{"type": "string", "description": "Optional local failed-log file path"},
				"workflow_dir": map[string]any{"type": "string", "default": ".github/workflows"},
			},
		},
	},
}

// HandleRequest processes a single JSON-RPC 2.0 MCP message.
func HandleRequest(raw []byte, runTool func(args map[string]any) (any, error)) ([]byte, bool) {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, false
	}

	switch req.Method {
	case "initialize":
		return marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "gh-repro", "version": "0.1.0"},
			},
		}), true
	case "notifications/initialized":
		return nil, false
	case "tools/list":
		return marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  map[string]any{"tools": tools},
		}), true
	case "tools/call":
		name, _ := req.Params["name"].(string)
		if name != "reproduce_ci_failure" || runTool == nil {
			return marshal(rpcErr(req.ID, -32601, "Unknown tool: "+name)), true
		}
		args, _ := req.Params["arguments"].(map[string]any)
		out, err := runTool(args)
		if err != nil {
			return marshal(rpcErr(req.ID, -32000, err.Error())), true
		}
		text, _ := json.MarshalIndent(out, "", "  ")
		return marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"content": []map[string]any{{"type": "text", "text": string(text)}},
			},
		}), true
	default:
		if req.ID == nil {
			return nil, false
		}
		return marshal(rpcErr(req.ID, -32601, "Method not found: "+req.Method)), true
	}
}

// ServeStdio runs the line-delimited JSON-RPC MCP loop.
func ServeStdio(in io.Reader, out io.Writer, runTool func(args map[string]any) (any, error)) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if resp, ok := HandleRequest(line, runTool); ok {
			if _, err := out.Write(append(resp, '\n')); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func rpcErr(id any, code int, msg string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": msg},
	}
}

func marshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
