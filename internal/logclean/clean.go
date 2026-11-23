package logclean

import (
	"regexp"
	"strings"
)

// StepLog groups cleaned log lines belonging to a single GitHub Actions job step.
type StepLog struct {
	Job   string   `json:"job"`
	Step  string   `json:"step"`
	Lines []string `json:"lines"`
}

var (
	ansiRE      = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	timestampRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z\s+`)
	secretREs   = []*regexp.Regexp{
		regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{20,}\b`),
		regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
		regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
		regexp.MustCompile(`\bsk-(?:proj-|ant-|live_)?[A-Za-z0-9_-]{20,}\b`),
		regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._-]{16,}\b`),
	}
)

// ParseFailedLog parses `gh run view --log-failed` output, stripping ANSI codes,
// UTC timestamps, workflow group markers, and secret tokens.
func ParseFailedLog(raw string) []StepLog {
	var steps []StepLog
	indexByKey := make(map[string]int)

	for _, rawLine := range strings.Split(raw, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		job, step, msg := splitTabRecord(line)
		msg = ansiRE.ReplaceAllString(msg, "")
		msg = timestampRE.ReplaceAllString(msg, "")
		msg = strings.TrimSpace(msg)

		if msg == "" || strings.HasPrefix(msg, "##[group]") || strings.HasPrefix(msg, "##[endgroup]") {
			continue
		}
		msg = redactTokens(msg)

		key := job + "\x00" + step
		idx, ok := indexByKey[key]
		if !ok {
			idx = len(steps)
			indexByKey[key] = idx
			steps = append(steps, StepLog{Job: job, Step: step})
		}
		steps[idx].Lines = append(steps[idx].Lines, msg)
	}
	return steps
}

func splitTabRecord(line string) (job, step, msg string) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) == 3 {
		return ansiRE.ReplaceAllString(parts[0], ""), parts[1], parts[2]
	}
	return "default", "failed-step", line
}

func redactTokens(s string) string {
	out := s
	for _, re := range secretREs {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	return out
}
