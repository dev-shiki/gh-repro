package ghclient

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var runURLRE = regexp.MustCompile(`/actions/runs/(\d+)`)

// ResolveFailingRunID finds the latest failed workflow run ID from a PR or current HEAD commit.
func ResolveFailingRunID(repo string, pr int) (int64, error) {
	if pr > 0 {
		args := []string{"pr", "checks", strconv.Itoa(pr), "--json", "name,state,link"}
		if repo != "" {
			args = append(args, "-R", repo)
		}
		out, err := runGH(args...)
		if err != nil {
			return 0, err
		}
		var checks []struct {
			State string `json:"state"`
			Link  string `json:"link"`
		}
		if err := json.Unmarshal(out, &checks); err != nil {
			return 0, err
		}
		for _, c := range checks {
			if strings.EqualFold(c.State, "FAILURE") || strings.EqualFold(c.State, "ERROR") {
				if m := runURLRE.FindStringSubmatch(c.Link); len(m) == 2 {
					return strconv.ParseInt(m[1], 10, 64)
				}
			}
		}
		return 0, fmt.Errorf("no failing GitHub Actions run found on PR #%d", pr)
	}

	args := []string{"run", "list", "--status", "failure", "--limit", "1", "--json", "databaseId"}
	if repo != "" {
		args = append(args, "-R", repo)
	}
	out, err := runGH(args...)
	if err != nil {
		return 0, err
	}
	var runs []struct {
		DatabaseID int64 `json:"databaseId"`
	}
	if err := json.Unmarshal(out, &runs); err != nil {
		return 0, err
	}
	if len(runs) == 0 {
		return 0, fmt.Errorf("no recent failed workflow runs found")
	}
	return runs[0].DatabaseID, nil
}

// FetchFailedLog retrieves `gh run view <runID> --log-failed`.
func FetchFailedLog(repo string, runID int64) (string, error) {
	args := []string{"run", "view", strconv.FormatInt(runID, 10), "--log-failed"}
	if repo != "" {
		args = append(args, "-R", repo)
	}
	out, err := runGH(args...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func runGH(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh %s failed: %w\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}
