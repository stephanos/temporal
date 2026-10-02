package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestRunnerDeliversExplicitEnvironmentBeforeInitAndReplays(t *testing.T) {
	t.Setenv("GOMAD_ENV_HOST_ONLY", "must not reach target")
	for _, test := range []struct {
		name        string
		environment []string
		want        []string
	}{
		{name: "empty", want: []string{"TZ=UTC"}},
		{name: "explicit", environment: []string{"VALUE=spaces = unicode λ", "EMPTY="}, want: []string{"EMPTY=", "TZ=UTC", "VALUE=spaces = unicode λ"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := isolatedCampaign(t, conformanceTarget(t, "./environment"))
			config.Environment = test.environment
			config.ChoiceTraceLimit = 1 << 20
			config.KeepSuccesses = KeepSuccessesAll
			config.SuccessArtifactLimit = 1
			config.SuccessBytesLimit = 32 << 20
			summary := exploreIsolated(t, config)
			if summary.Succeeded != 1 || len(summary.SuccessArtifacts) != 1 {
				t.Fatalf("environment campaign = %#v", summary)
			}
			path := summary.SuccessArtifacts[0]
			stdout, err := os.ReadFile(filepath.Join(path, "stdout"))
			if err != nil {
				t.Fatal(err)
			}
			if want := fmt.Sprintf("init=%q main=%q\n", test.want, test.want); string(stdout) != want {
				t.Fatalf("target environment = %s, want %s", stdout, want)
			}
			replayed, err := Replay(context.Background(), ReplaySpec{
				ArtifactPath: path, ToolchainRoot: toolchainRoot(t), SupervisorCommand: config.SupervisorCommand,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !replayed.Match || replayed.ChoiceReplayStatus != ChoiceReplayExact {
				t.Fatalf("environment replay = %#v", replayed)
			}
		})
	}
}

func TestDirectTargetEnvironmentPreservesSeededScrubbingAndDisabledBehavior(t *testing.T) {
	spec := conformanceTarget(t, "./environment")
	spec.PreparationRoot = t.TempDir()
	prepared, err := target.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name        string
		environment []string
		want        []string
	}{
		{name: "seeded", environment: []string{"GOMADSEED=7", "TZ=UTC", "VALUE=host"}, want: []string{"TZ=UTC"}},
		{name: "disabled", environment: []string{"TZ=UTC", "VALUE=host"}, want: []string{"TZ=UTC", "VALUE=host"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.CommandContext(t.Context(), prepared.Path)
			command.Env = test.environment
			stdout, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("direct target: %v: %s", err, stdout)
			}
			if want := fmt.Sprintf("init=%q main=%q\n", test.want, test.want); string(stdout) != want {
				t.Fatalf("target environment = %s, want %s", stdout, want)
			}
		})
	}
}
