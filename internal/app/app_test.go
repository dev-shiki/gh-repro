package app

import (
	"path/filepath"
	"testing"
)

func TestAnalyzeEndToEndWithWorkflowContext(t *testing.T) {
	wfDir := filepath.Join("..", "..", "testdata")
	repros, err := Analyze(Options{
		LogFile:     filepath.Join(wfDir, "pytest.log"),
		WorkflowDir: wfDir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repros) != 1 {
		t.Fatalf("expected 1 repro, got %d", len(repros))
	}
	r := repros[0]
	if r.WorkingDir != "backend" {
		t.Fatalf("expected WorkingDir=backend, got %q", r.WorkingDir)
	}
	if r.Env["APP_ENV"] != "test" {
		t.Fatalf("expected APP_ENV=test, got %#v", r.Env)
	}
	if r.Command != "pytest tests/test_auth.py::TestToken::test_refresh" {
		t.Fatalf("unexpected command: %q", r.Command)
	}
}
