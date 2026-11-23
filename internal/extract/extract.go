package extract

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dev-shiki/gh-repro/internal/logclean"
	"github.com/dev-shiki/gh-repro/internal/workflow"
)

// Repro describes a deterministic local reproduction command for a failed CI step.
type Repro struct {
	Runner     string            `json:"runner"`
	Job        string            `json:"job"`
	Step       string            `json:"step"`
	Bin        string            `json:"bin"`
	Args       []string          `json:"args"`
	Command    string            `json:"command"`
	Tests      []string          `json:"tests,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	ErrorTail  []string          `json:"error_tail,omitempty"`
}

var (
	pytestFailRE = regexp.MustCompile(`^FAILED\s+([^\s]+)`)
	goFailTestRE = regexp.MustCompile(`^---\s+FAIL:\s+([A-Za-z0-9_/.]+)`)
	goFailPkgRE  = regexp.MustCompile(`^FAIL\s+([^\s]+)\s+`)
	vitestFailRE = regexp.MustCompile(`^FAIL\s+([^\s>]+)\s*>\s*(.+)$`)
	cargoFailRE  = regexp.MustCompile(`^test\s+([^\s]+)\s+\.\.\.\s+FAILED$`)
)

// BuildRepro inspects cleaned step logs and workflow context to emit a minimal
// local test command (without shell interpolation on untrusted test names).
func BuildRepro(step logclean.StepLog, ctx workflow.StepContext, goModule string) Repro {
	tail := tailLines(step.Lines, 20)
	base := Repro{
		Job:        step.Job,
		Step:       step.Step,
		WorkingDir: ctx.WorkingDir,
		Env:        ctx.Env,
		ErrorTail:  tail,
	}

	if tests := matchAll(step.Lines, pytestFailRE); len(tests) > 0 {
		base.Runner = "pytest"
		base.Bin = "pytest"
		base.Args = tests
		base.Tests = tests
		base.Command = formatCommand(base.Bin, base.Args)
		return base
	}

	if tests := matchAll(step.Lines, goFailTestRE); len(tests) > 0 {
		pkg := "./..."
		if pkgs := matchAll(step.Lines, goFailPkgRE); len(pkgs) > 0 {
			pkg = toLocalGoPkg(pkgs[0], goModule)
		}
		escaped := make([]string, 0, len(tests))
		for _, t := range tests {
			escaped = append(escaped, regexp.QuoteMeta(t))
		}
		pattern := "^(" + strings.Join(escaped, "|") + ")$"
		base.Runner = "go"
		base.Bin = "go"
		base.Args = []string{"test", "-v", "-run", pattern, pkg}
		base.Tests = tests
		base.Command = formatCommand(base.Bin, base.Args)
		return base
	}

	if file, title := matchVitest(step.Lines); file != "" {
		runner := "vitest"
		bin := "npx"
		args := []string{"vitest", "run", file, "-t", title}
		if strings.Contains(strings.ToLower(ctx.Run), "jest") {
			runner = "jest"
			args = []string{"jest", file, "-t", title}
		}
		base.Runner = runner
		base.Bin = bin
		base.Args = args
		base.Tests = []string{file + " > " + title}
		base.Command = formatCommand(base.Bin, base.Args)
		return base
	}

	if tests := matchAll(step.Lines, cargoFailRE); len(tests) > 0 {
		base.Runner = "cargo"
		base.Bin = "cargo"
		base.Args = append([]string{"test"}, tests...)
		base.Tests = tests
		base.Command = formatCommand(base.Bin, base.Args)
		return base
	}

	base.Runner = "generic"
	if ctx.Run != "" {
		firstLine := strings.SplitN(strings.TrimSpace(ctx.Run), "\n", 2)[0]
		fields := strings.Fields(firstLine)
		if len(fields) > 0 {
			base.Bin = fields[0]
			base.Args = fields[1:]
			base.Command = formatCommand(base.Bin, base.Args)
		}
	}
	return base
}

func matchAll(lines []string, re *regexp.Regexp) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, line := range lines {
		m := re.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) < 2 {
			continue
		}
		val := strings.TrimSpace(m[1])
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, val)
	}
	return out
}

func matchVitest(lines []string) (file, title string) {
	for _, line := range lines {
		m := vitestFailRE.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) == 3 {
			return strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		}
	}
	return "", ""
}

func toLocalGoPkg(fullPkg, goModule string) string {
	if goModule != "" && strings.HasPrefix(fullPkg, goModule) {
		rel := strings.TrimPrefix(fullPkg, goModule)
		if rel == "" {
			return "."
		}
		return "." + rel
	}
	return fullPkg
}

func tailLines(lines []string, max int) []string {
	if len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
}

func formatCommand(bin string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, shellQuote(bin))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	safe := true
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("@%_+=:,./-^()", r)) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "'\"'\"'"))
}
