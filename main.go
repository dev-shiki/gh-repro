package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dev-shiki/gh-repro/internal/app"
	"github.com/dev-shiki/gh-repro/internal/mcp"
)

func main() {
	runID := flag.Int64("run-id", 0, "GitHub Actions workflow run ID")
	pr := flag.Int("pr", 0, "Pull request number to inspect")
	repo := flag.String("repo", "", "Repository in OWNER/REPO format")
	logFile := flag.String("log-file", "", "Offline path to a `gh run view --log-failed` file")
	wfDir := flag.String("workflow-dir", filepath.Join(".github", "workflows"), "Path to workflow YAML directory")
	asJSON := flag.Bool("json", false, "Emit JSON output")
	execRun := flag.Bool("run", false, "Execute the first extracted reproduction command locally")
	mcpMode := flag.Bool("mcp", false, "Run as a stdio MCP server")
	flag.Parse()

	if *mcpMode {
		err := mcp.ServeStdio(os.Stdin, os.Stdout, func(args map[string]any) (any, error) {
			opts := app.Options{
				RunID:       int64From(args["run_id"]),
				PR:          int(int64From(args["pr"])),
				Repo:        strFrom(args["repo"]),
				LogFile:     strFrom(args["log_file"]),
				WorkflowDir: strFrom(args["workflow_dir"]),
			}
			return app.Analyze(opts)
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "mcp error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	repros, err := app.Analyze(app.Options{
		RunID:       *runID,
		PR:          *pr,
		Repo:        *repo,
		LogFile:     *logFile,
		WorkflowDir: *wfDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "gh-repro: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(repros)
	} else {
		for i, r := range repros {
			fmt.Printf("[%d] %s / %s (%s)\n", i+1, r.Job, r.Step, r.Runner)
			if r.WorkingDir != "" {
				fmt.Printf("    cwd: %s\n", r.WorkingDir)
			}
			if len(r.Env) > 0 {
				fmt.Printf("    env: %v\n", r.Env)
			}
			fmt.Printf("    cmd: %s\n", r.Command)
			if len(r.ErrorTail) > 0 {
				fmt.Printf("    err: %s\n", strings.Join(r.ErrorTail, "\n         "))
			}
		}
	}

	if *execRun && len(repros) > 0 {
		if err := app.Execute(repros[0]); err != nil {
			os.Exit(1)
		}
	}
}

func int64From(v any) int64 {
	if f, ok := v.(float64); ok {
		return int64(f)
	}
	return 0
}

func strFrom(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
