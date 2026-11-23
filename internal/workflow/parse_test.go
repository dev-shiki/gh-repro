package workflow

import (
	"path/filepath"
	"testing"
)

func TestFindStepContextMergesWorkingDirAndEnv(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata")
	ctx, err := FindStepContext(dir, "test-python", "Run pytest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.WorkingDir != "backend" {
		t.Fatalf("expected working-directory=backend, got %q", ctx.WorkingDir)
	}
	if ctx.Env["APP_ENV"] != "test" || ctx.Env["DATABASE_URL"] != "sqlite:///:memory:" {
		t.Fatalf("expected merged env, got %#v", ctx.Env)
	}
	if ctx.Run != "pytest -q" {
		t.Fatalf("expected run='pytest -q', got %q", ctx.Run)
	}
}
