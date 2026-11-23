package workflow

import (
	"maps"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// StepContext holds the execution context reconstructed from workflow YAML.
type StepContext struct {
	WorkingDir string            `json:"working_dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Run        string            `json:"run,omitempty"`
}

type workflowFile struct {
	Env  map[string]string  `yaml:"env"`
	Jobs map[string]jobSpec `yaml:"jobs"`
}

type jobSpec struct {
	Name     string            `yaml:"name"`
	Env      map[string]string `yaml:"env"`
	Defaults struct {
		Run struct {
			WorkingDirectory string `yaml:"working-directory"`
		} `yaml:"run"`
	} `yaml:"defaults"`
	Steps []stepSpec `yaml:"steps"`
}

type stepSpec struct {
	Name             string            `yaml:"name"`
	Run              string            `yaml:"run"`
	WorkingDirectory string            `yaml:"working-directory"`
	Env              map[string]string `yaml:"env"`
}

// FindStepContext locates the matching job/step inside workflowDir YAML files
// and merges workflow, job, and step working-directory and env settings.
func FindStepContext(workflowDir, jobName, stepName string) (StepContext, error) {
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return StepContext{}, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(workflowDir, entry.Name()))
		if err != nil {
			continue
		}
		var wf workflowFile
		if err := yaml.Unmarshal(data, &wf); err != nil {
			return StepContext{}, err
		}

		for jobID, job := range wf.Jobs {
			if !matchJob(jobID, job.Name, jobName) {
				continue
			}
			for _, st := range job.Steps {
				if !matchStep(st.Name, st.Run, stepName) {
					continue
				}
				mergedEnv := make(map[string]string)
				maps.Copy(mergedEnv, wf.Env)
				maps.Copy(mergedEnv, job.Env)
				maps.Copy(mergedEnv, st.Env)

				wd := job.Defaults.Run.WorkingDirectory
				if st.WorkingDirectory != "" {
					wd = st.WorkingDirectory
				}
				return StepContext{
					WorkingDir: wd,
					Env:        mergedEnv,
					Run:        strings.TrimSpace(st.Run),
				}, nil
			}
		}
	}
	return StepContext{}, nil
}

func matchJob(jobID, jobDisplayName, target string) bool {
	if target == "" || target == "default" {
		return false
	}
	return strings.EqualFold(jobID, target) ||
		(jobDisplayName != "" && strings.EqualFold(jobDisplayName, target)) ||
		strings.HasPrefix(strings.ToLower(target), strings.ToLower(jobID)+" ")
}

func matchStep(stepName, stepRun, target string) bool {
	if target == "" {
		return false
	}
	if stepName != "" && strings.EqualFold(stepName, target) {
		return true
	}
	if strings.HasPrefix(target, "Run ") && stepRun != "" {
		firstLine := strings.SplitN(strings.TrimSpace(stepRun), "\n", 2)[0]
		return strings.EqualFold(strings.TrimPrefix(target, "Run "), strings.TrimSpace(firstLine))
	}
	return false
}
