package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dev-shiki/gh-repro/internal/extract"
	"github.com/dev-shiki/gh-repro/internal/ghclient"
	"github.com/dev-shiki/gh-repro/internal/logclean"
	"github.com/dev-shiki/gh-repro/internal/workflow"
)

// Options configures a reproduction analysis run.
type Options struct {
	RunID       int64
	PR          int
	Repo        string
	LogFile     string
	WorkflowDir string
}

// Analyze resolves the failed CI log, matches workflow context, and builds local repro commands.
func Analyze(opts Options) ([]extract.Repro, error) {
	raw, err := loadLog(opts)
	if err != nil {
		return nil, err
	}

	steps := logclean.ParseFailedLog(raw)
	if len(steps) == 0 {
		return nil, fmt.Errorf("no failed step lines found in log")
	}

	wfDir := opts.WorkflowDir
	if wfDir == "" {
		wfDir = filepath.Join(".github", "workflows")
	}
	goMod := readGoModule("go.mod")

	var repros []extract.Repro
	for _, st := range steps {
		ctx, _ := workflow.FindStepContext(wfDir, st.Job, st.Step)
		r := extract.BuildRepro(st, ctx, goMod)
		if r.Command != "" || len(r.ErrorTail) > 0 {
			repros = append(repros, r)
		}
	}
	return repros, nil
}

// Execute runs the extracted command directly without shell interpolation.
func Execute(r extract.Repro) error {
	if r.Bin == "" {
		return fmt.Errorf("no executable command extracted for %s / %s", r.Job, r.Step)
	}
	cmd := exec.Command(r.Bin, r.Args...)
	if r.WorkingDir != "" {
		cmd.Dir = r.WorkingDir
	}
	if len(r.Env) > 0 {
		env := os.Environ()
		for k, v := range r.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func loadLog(opts Options) (string, error) {
	if opts.LogFile != "" {
		b, err := os.ReadFile(opts.LogFile)
		return string(b), err
	}
	runID := opts.RunID
	if runID == 0 {
		var err error
		runID, err = ghclient.ResolveFailingRunID(opts.Repo, opts.PR)
		if err != nil {
			return "", err
		}
	}
	return ghclient.FetchFailedLog(opts.Repo, runID)
}

func readGoModule(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
