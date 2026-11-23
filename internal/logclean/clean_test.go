package logclean

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFailedLogStripsANSIAndRedactsTokens(t *testing.T) {
	rawBytes, err := os.ReadFile(filepath.Join("..", "..", "testdata", "pytest.log"))
	if err != nil {
		t.Fatal(err)
	}
	raw := "\x1b[31m" + string(rawBytes)
	steps := ParseFailedLog(raw)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	s := steps[0]
	if s.Job != "test-python" || s.Step != "Run pytest" {
		t.Fatalf("unexpected job/step: %+v", s)
	}
	if len(s.Lines) != 2 {
		t.Fatalf("expected 2 cleaned lines (group/endgroup dropped), got %d: %#v", len(s.Lines), s.Lines)
	}
	if contains(s.Lines[0], "ghp_") {
		t.Fatalf("expected ghp_ token to be redacted, got %q", s.Lines[0])
	}
	if !contains(s.Lines[0], "[REDACTED]") {
		t.Fatalf("expected [REDACTED] placeholder, got %q", s.Lines[0])
	}
	if contains(s.Lines[0], "\x1b[") {
		t.Fatalf("expected ANSI codes stripped, got %q", s.Lines[0])
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && indexString(s, sub) >= 0
}

func indexString(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
