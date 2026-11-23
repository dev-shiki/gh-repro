package extract

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dev-shiki/gh-repro/internal/logclean"
	"github.com/dev-shiki/gh-repro/internal/workflow"
)

func mustReadSteps(t *testing.T, fixture string) logclean.StepLog {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	steps := logclean.ParseFailedLog(string(b))
	if len(steps) == 0 {
		t.Fatalf("no steps parsed from %s", fixture)
	}
	return steps[0]
}

func TestBuildReproAcrossRunners(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		modPath  string
		stepCtx  workflow.StepContext
		wantBin  string
		wantArgs []string
	}{
		{
			name:     "pytest",
			fixture:  "pytest.log",
			stepCtx:  workflow.StepContext{WorkingDir: "backend", Run: "pytest -q"},
			wantBin:  "pytest",
			wantArgs: []string{"tests/test_auth.py::TestToken::test_refresh"},
		},
		{
			name:     "go test",
			fixture:  "go.log",
			modPath:  "github.com/datazip-inc/olake",
			stepCtx:  workflow.StepContext{Run: "go test ./..."},
			wantBin:  "go",
			wantArgs: []string{"test", "-v", "-run", "^(TestMissingConfigFlagPreRun)$", "./protocol"},
		},
		{
			name:     "vitest",
			fixture:  "vitest.log",
			stepCtx:  workflow.StepContext{WorkingDir: "web", Run: "npx vitest run"},
			wantBin:  "npx",
			wantArgs: []string{"vitest", "run", "src/auth/session.test.ts", "-t", "SessionStore > refreshes expired cookie"},
		},
		{
			name:     "cargo",
			fixture:  "cargo.log",
			stepCtx:  workflow.StepContext{Run: "cargo test --all-targets"},
			wantBin:  "cargo",
			wantArgs: []string{"test", "parser::tests::rejects_malformed_header"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			step := mustReadSteps(t, tc.fixture)
			got := BuildRepro(step, tc.stepCtx, tc.modPath)
			if got.Bin != tc.wantBin || !reflect.DeepEqual(got.Args, tc.wantArgs) {
				t.Fatalf("got (%s, %#v), want (%s, %#v)", got.Bin, got.Args, tc.wantBin, tc.wantArgs)
			}
			if got.Command == "" {
				t.Fatal("expected non-empty formatted Command")
			}
		})
	}
}

func TestBuildReproQuotesMaliciousTestNameSafely(t *testing.T) {
	step := logclean.StepLog{
		Job:  "test-web",
		Step: "Vitest",
		Lines: []string{
			"FAIL src/evil.test.ts > breaks $(whoami); rm -rf /",
		},
	}
	got := BuildRepro(step, workflow.StepContext{Run: "npx vitest run"}, "")
	if got.Bin != "npx" {
		t.Fatalf("expected npx, got %s", got.Bin)
	}
	if !strings.Contains(got.Command, "'breaks $(whoami); rm -rf /'") {
		t.Fatalf("expected single-quoted test name in Command, got %q", got.Command)
	}
}
