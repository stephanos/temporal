package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

// These tests pin the behavior that keys on whether an operation runs its
// executions through the supervisor process or through a substituted
// executor, so moving the substitution out of the public requests cannot
// change it.

func TestInjectionCharacterizationSimulationRoleKeysOnTheProcessExecutor(t *testing.T) {
	capability, err := simulationCapabilityForJob(processExecutor{}, runJob{})
	if err != nil || !reflect.DeepEqual(capability, &execution.SimulationCapability{Role: execution.SimulationRoleCoordinator}) {
		t.Fatalf("process executor simulation capability = %#v, %v", capability, err)
	}
	capability, err = simulationCapabilityForJob(&fakeExecutor{}, runJob{})
	if err != nil || capability != nil {
		t.Fatalf("substituted executor simulation capability = %#v, %v", capability, err)
	}
}

func TestInjectionCharacterizationReplayRequiresSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	artifactPath, observed := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{})
	_, err := Replay(context.Background(), ReplaySpec{ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t)})
	if err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("replay without supervisor or executor error = %v", err)
	}
	if _, err := Replay(context.Background(), ReplaySpec{ArtifactPath: artifactPath, VerifyOnly: true, ToolchainRoot: toolchainRoot(t)}); err != nil {
		t.Fatalf("verify-only replay without supervisor: %v", err)
	}
	// A substituted replay needs no supervisor command. It still needs a
	// bootstrap command: without one, replayBootstrapCommand derives it from
	// the absent supervisor command and indexes past its end.
	executor := &fakeReplayExecutor{result: observed}
	result, err := Replay(context.Background(), ReplaySpec{ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t), BootstrapCommand: []string{"bootstrap"}, Executor: executor})
	if err != nil || !result.Match {
		t.Fatalf("substituted replay without supervisor = %#v, %v", result, err)
	}
	// Only the process executor asks the supervisor for the coordinator
	// simulation role when the artifact carries no simulation plan.
	if executor.calls != 1 || executor.request.Simulation != nil || len(executor.request.SupervisorCommand) != 0 || !reflect.DeepEqual(executor.request.BootstrapCommand, []string{"bootstrap"}) {
		t.Fatalf("substituted replay request calls=%d simulation=%#v supervisor=%q bootstrap=%q", executor.calls, executor.request.Simulation, executor.request.SupervisorCommand, executor.request.BootstrapCommand)
	}
}

func TestInjectionCharacterizationMinimizeRequiresSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	spec := minimizationSpec(t, minimizationParent(t), t.TempDir())
	spec.SupervisorCommand, spec.Executor = nil, nil
	if _, err := Minimize(context.Background(), spec); err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("minimize without supervisor or executor error = %v", err)
	}
	spec = minimizationSpec(t, minimizationParent(t), t.TempDir())
	spec.SupervisorCommand = nil
	result, err := Minimize(context.Background(), spec)
	if err != nil || !result.Changed {
		t.Fatalf("substituted minimize without supervisor = %#v, %v", result, err)
	}
}

var errReplayReachedSubstitutedExecutor = errors.New("replay reached the substituted executor")

// replayForwardingExecutor evaluates minimization candidates like
// minimizationExecutor and fails every replay execution, which it recognizes
// by its replayed I/O transcript.
type replayForwardingExecutor struct {
	candidates minimizationExecutor
	replays    int
}

func (executor *replayForwardingExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	if request.IO != nil && request.IO.Transcript != nil && request.IO.Transcript.Replay {
		executor.replays++
		return execution.Result{}, errReplayReachedSubstitutedExecutor
	}
	return executor.candidates.Run(ctx, request)
}

func TestInjectionCharacterizationMinimizeDefaultReplayerUsesItsExecutor(t *testing.T) {
	spec := minimizationSpec(t, minimizationParent(t), t.TempDir())
	executor := &replayForwardingExecutor{}
	spec.Executor, spec.Replayer = executor, nil
	_, err := Minimize(context.Background(), spec)
	if !errors.Is(err, errReplayReachedSubstitutedExecutor) || !strings.HasPrefix(err.Error(), "replay minimization candidate: execute replay target: ") {
		t.Fatalf("minimize default replay error = %v", err)
	}
	if executor.replays != 1 || executor.candidates.calls == 0 {
		t.Fatalf("minimize executor replays=%d candidates=%d", executor.replays, executor.candidates.calls)
	}
}

func TestInjectionCharacterizationShardChecksToolchainOnlyForTheProcessExecutor(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	if _, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}); err != nil {
		t.Fatal(err)
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-toolchain")
	_, err := RunCampaignShard(context.Background(), CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		ToolchainRoot: missingRoot, SupervisorCommand: []string{"unused"},
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("process shard with a missing toolchain error = %v", err)
	}
	executor := &fakeExecutor{result: func(uint64) execution.Result { return processResult(0, "", "") }}
	result, err := RunCampaignShard(context.Background(), CampaignShardSpec{
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		ToolchainRoot: missingRoot, Executor: executor,
	})
	if err != nil || result.Succeeded != 1 || len(executor.requests) != 1 {
		t.Fatalf("substituted shard with a missing toolchain = %#v requests=%d, %v", result, len(executor.requests), err)
	}
}

func TestInjectionCharacterizationResumeChecksToolchainAndSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &resumeInterruptExecutor{}, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(event CampaignEvent) bool { return event.Succeeded == 1 })
	partial, err := Explore(ctx, config)
	if err == nil {
		t.Fatal("Explore() was not interrupted")
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-toolchain")
	_, err = Resume(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot})
	if err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("resume without supervisor or executor error = %v", err)
	}
	_, err = Resume(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot, SupervisorCommand: []string{"unused"}})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("process resume with a missing toolchain error = %v", err)
	}
	executor := &fakeExecutor{result: func(uint64) execution.Result { return processResult(0, "", "") }}
	resumed, err := Resume(context.Background(), ResumeSpec{CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot, Executor: executor})
	if err != nil || resumed.Attempted != 2 || resumed.Succeeded != 2 || len(executor.requests) != 1 {
		t.Fatalf("substituted resume with a missing toolchain = %#v requests=%d, %v", resumed, len(executor.requests), err)
	}
}

func TestInjectionCharacterizationIsolatedExploreRejectsEverySubstitution(t *testing.T) {
	for name, configure := range map[string]func(*CampaignSpec){
		"executor": func(spec *CampaignSpec) { spec.Executor = &fakeExecutor{} },
		"preparer": func(spec *CampaignSpec) { spec.Preparer = targetPreparer{} },
		"replayer": func(spec *CampaignSpec) { spec.Replayer = artifactReplayer{} },
	} {
		t.Run(name, func(t *testing.T) {
			config := testConfig(t, nil, nil, "1", PolicyAll, 1)
			config.CoordinatorCommand = []string{"unused", "__coordinator"}
			configure(&config)
			if _, err := Explore(context.Background(), config); err == nil || err.Error() != "isolated Runner does not accept injected preparation or execution" {
				t.Fatalf("isolated Explore with an injected %s error = %v", name, err)
			}
		})
	}
}
