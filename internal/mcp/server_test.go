package mcp

import (
	"encoding/json"
	"testing"
)

func TestMCPInitializeToolsListAndCall(t *testing.T) {
	initResp, send := HandleRequest([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`), nil)
	if !send {
		t.Fatal("expected response for initialize")
	}
	var parsed map[string]any
	if err := json.Unmarshal(initResp, &parsed); err != nil {
		t.Fatal(err)
	}
	res := parsed["result"].(map[string]any)
	if res["serverInfo"].(map[string]any)["name"] != "gh-repro" {
		t.Fatalf("unexpected serverInfo: %#v", res["serverInfo"])
	}

	listResp, send := HandleRequest([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`), nil)
	if !send || !json.Valid(listResp) {
		t.Fatal("expected valid tools/list response")
	}

	callResp, send := HandleRequest(
		[]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"reproduce_ci_failure","arguments":{"run_id":123}}}`),
		func(args map[string]any) (any, error) {
			return map[string]any{"command": "pytest tests/a.py::test_x"}, nil
		},
	)
	if !send {
		t.Fatal("expected tools/call response")
	}
	if !json.Valid(callResp) {
		t.Fatalf("invalid JSON response: %s", callResp)
	}
}
