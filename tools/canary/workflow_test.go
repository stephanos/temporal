package canary_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/policy"
	"gopkg.in/yaml.v3"
)

type workflow struct {
	Name        string                 `yaml:"name"`
	On          map[string]any         `yaml:"on"`
	Permissions map[string]string      `yaml:"permissions"`
	Concurrency concurrency            `yaml:"concurrency"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

type concurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress *bool  `yaml:"cancel-in-progress"`
}

type workflowJob struct {
	If             string            `yaml:"if"`
	RunsOn         string            `yaml:"runs-on"`
	Environment    string            `yaml:"environment"`
	TimeoutMinutes int               `yaml:"timeout-minutes"`
	Permissions    map[string]string `yaml:"permissions"`
	Steps          []workflowStep    `yaml:"steps"`
}

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	If   string            `yaml:"if"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
	With map[string]any    `yaml:"with"`
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// The protected workflow is manual, in one non-cancelling concurrency group, on main only, in the
// production-canary environment, read-only, bounded, pinned, and runs exactly the untagged
// binary's two closed modes, reconcile and the upload always; the policy names this very file.
func TestTheProductionCanaryWorkflowIsManualAndProtected(t *testing.T) {
	canary, err := policy.Embedded()
	require.NoError(t, err)
	encoded, err := os.ReadFile(filepath.Join(repositoryRoot(t), canary.WorkflowPath))
	require.NoError(t, err, "the policy's workflow path is the workflow file")
	var parsed workflow
	require.NoError(t, yaml.Unmarshal(encoded, &parsed))

	require.Equal(t, map[string]any{"workflow_dispatch": nil}, parsed.On, "a manual dispatch is the only trigger")
	require.Equal(t, map[string]string{"contents": "read"}, parsed.Permissions)
	require.Equal(t, "umpire-production-canary", parsed.Concurrency.Group)
	require.NotNil(t, parsed.Concurrency.CancelInProgress)
	require.False(t, *parsed.Concurrency.CancelInProgress, "no job ever cancels another's Run")
	require.Len(t, parsed.Jobs, 1)
	job := parsed.Jobs["canary"]
	require.Equal(t, "github.ref == '"+canary.TrustedRef+"'", job.If)
	require.Equal(t, "production-canary", job.Environment)
	require.Equal(t, 30, job.TimeoutMinutes)
	require.Empty(t, job.Permissions, "the job takes no permission beyond the workflow's read")
	require.Regexp(t, `^ubuntu-\d`, job.RunsOn, "a fresh GitHub-hosted runner per dispatch")

	pinned := regexp.MustCompile(`^[\w.-]+/[\w.-]+@[0-9a-f]{40}$`)
	var runs, reconciles, uploads int
	for _, step := range job.Steps {
		if step.Uses != "" {
			require.Regexp(t, pinned, step.Uses, "every action is pinned to a commit")
		}
		for name, value := range step.Env {
			require.True(t, strings.HasPrefix(name, "UMPIRE_CANARY_"), name)
			require.Equal(t, "${{ secrets."+name+" }}", value, "credentials and coordinates come only from the environment's secrets")
		}
		switch {
		case strings.Contains(step.Run, "umpire-canary run "):
			runs++
			require.Empty(t, step.If)
			requireOnlyTheClosedFlags(t, step.Run, "run")
			require.Contains(t, step.Run, "> canary-output/run-summary.json 2> canary-output/run-progress.log")
			requireEveryVariable(t, step.Env)
		case strings.Contains(step.Run, "umpire-canary reconcile "):
			reconciles++
			require.Equal(t, "always()", step.If, "reconcile runs whatever run did")
			requireOnlyTheClosedFlags(t, step.Run, "reconcile")
			require.Contains(t, step.Run, "> canary-output/reconcile-summary.json 2> canary-output/reconcile-progress.log")
			requireEveryVariable(t, step.Env)
		case strings.HasPrefix(step.Uses, "actions/upload-artifact@"):
			uploads++
			require.Equal(t, "always()", step.If)
			require.Equal(t, "canary-output/", step.With["path"])
		case step.Run != "":
			require.Equal(t, "make canary-build", step.Run, "the only other command builds the untagged binary")
		default:
			require.NotEmpty(t, step.Uses, "a step either runs a command or uses a pinned action")
		}
	}
	require.Equal(t, []int{1, 1, 1}, []int{runs, reconciles, uploads})
}

func requireOnlyTheClosedFlags(t *testing.T, command, mode string) {
	t.Helper()
	line := strings.TrimSpace(command[strings.Index(command, "./.build/umpire-canary "+mode):])
	line = strings.SplitN(line, " >", 2)[0]
	require.Equal(t, "./.build/umpire-canary "+mode+` --output canary-output --recovery "$RUNNER_TEMP/umpire-canary-recovery.json"`, line)
	require.NotContains(t, command, "canary_harness", "the workflow never builds or runs the harness")
}

func requireEveryVariable(t *testing.T, env map[string]string) {
	t.Helper()
	for _, name := range []string{
		authority.VariableTLSCert, authority.VariableTLSKey, authority.VariableAPIKey, authority.VariableGRPC,
		authority.VariableNamespace, authority.VariableTaskQueue, authority.VariableHandlerQueue, authority.VariableEndpoint,
	} {
		require.Contains(t, env, name)
	}
	require.Len(t, env, 8)
}
