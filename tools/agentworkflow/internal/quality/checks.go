package quality

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tools/agentworkflow/internal/process"
	"go.temporal.io/server/tools/agentworkflow/internal/workspace"
)

type Check struct {
	Name      string
	Command   []string
	Directory string
	Timeout   time.Duration
	Required  bool
}

type Options struct {
	DefaultTimeout time.Duration
	MaxOutputBytes int64
	Environment    []string
	Snapshot       workspace.Options
}

type Result struct {
	Name       string
	Command    []string
	Directory  string
	Required   bool
	Outcome    string
	ExitCode   int
	Duration   time.Duration
	Stdout     []byte
	Stderr     []byte
	Truncated  bool
	BeforeHash string
	AfterHash  string
}

func Run(ctx context.Context, candidate string, checks []Check, options Options) ([]Result, error) {
	if options.DefaultTimeout <= 0 || options.MaxOutputBytes <= 0 {
		return nil, errors.New("check bounds must be positive")
	}
	environment, err := buildEnvironment(options.Environment)
	if err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(checks))
	for _, check := range checks {
		if err := validateCheck(check); err != nil {
			return nil, err
		}
		directory, err := containedDirectory(candidate, check.Directory)
		if err != nil {
			return nil, fmt.Errorf("resolve check %q directory: %w", check.Name, err)
		}
		before, err := workspace.Snapshot(ctx, candidate, options.Snapshot)
		if err != nil {
			return nil, fmt.Errorf("snapshot before check %q: %w", check.Name, err)
		}
		timeout := check.Timeout
		if timeout == 0 {
			timeout = options.DefaultTimeout
		}
		processResult, processErr := (process.Runner{}).Run(ctx, process.Request{
			Command: check.Command, Directory: directory, Environment: append(environment, "PWD="+directory),
			Timeout: timeout, MaxOutputBytes: options.MaxOutputBytes,
		})
		after, snapshotErr := workspace.Snapshot(context.Background(), candidate, options.Snapshot)
		if snapshotErr != nil {
			return nil, errors.Join(fmt.Errorf("snapshot after check %q: %w", check.Name, snapshotErr), processErr)
		}
		result := Result{
			Name: check.Name, Command: append([]string(nil), check.Command...), Directory: check.Directory,
			Required: check.Required, ExitCode: processResult.ExitCode, Duration: processResult.Duration,
			Stdout: processResult.Stdout, Stderr: processResult.Stderr, Truncated: processResult.Truncated,
			BeforeHash: before, AfterHash: after,
		}
		switch {
		case before != after:
			result.Outcome = "mutated"
		case errors.Is(processErr, process.ErrTimeout):
			result.Outcome = "timed-out"
		case errors.Is(processErr, process.ErrCancelled):
			result.Outcome = "cancelled"
		case errors.Is(processErr, process.ErrOutputLimit):
			result.Outcome = "capacity-exhausted"
		case processErr != nil:
			result.Outcome = "infrastructure-failed"
		case processResult.ExitCode != 0:
			result.Outcome = "failed"
		default:
			result.Outcome = "passed"
		}
		results = append(results, result)
		if result.Outcome == "cancelled" {
			break
		}
	}
	return results, nil
}

func validateCheck(check Check) error {
	if strings.TrimSpace(check.Name) == "" {
		return errors.New("check name is required")
	}
	if len(check.Command) == 0 || strings.TrimSpace(check.Command[0]) == "" {
		return fmt.Errorf("check %q command is required", check.Name)
	}
	if check.Timeout < 0 {
		return fmt.Errorf("check %q timeout cannot be negative", check.Name)
	}
	return nil
}

func containedDirectory(root, relative string) (string, error) {
	if relative == "" {
		relative = "."
	}
	if filepath.IsAbs(relative) {
		return "", errors.New("check directory must be relative")
	}
	directory := filepath.Clean(filepath.Join(root, relative))
	contained, err := filepath.Rel(root, directory)
	if err != nil || contained == ".." || strings.HasPrefix(contained, ".."+string(filepath.Separator)) {
		return "", errors.Join(errors.New("check directory escapes candidate"), err)
	}
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return "", errors.Join(errors.New("check directory is not a directory"), err)
	}
	return directory, nil
}

func buildEnvironment(names []string) ([]string, error) {
	seen := make(map[string]struct{}, len(names))
	result := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || strings.ContainsRune(name, '=') {
			return nil, fmt.Errorf("environment name %q is invalid", name)
		}
		if _, found := seen[name]; found {
			continue
		}
		seen[name] = struct{}{}
		if value, found := os.LookupEnv(name); found {
			result = append(result, name+"="+value)
		}
	}
	return result, nil
}
