package runner

import (
	"context"
	"errors"
	backend "go.temporal.io/server/tools/gomad3/backends"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/native"

	"go.temporal.io/server/tools/gomad3/target"
)

// These tests pin the behavior that keys on whether an operation runs its
// executions through the supervisor process or through a substituted
// executor, so moving the substitution out of the public requests cannot
// change it.

func TestInjectionCharacterizationSimulationRoleKeysOnTheProcessExecutor(t *testing.T) {
	capability, err := simulationCapabilityForJob(processExecutor{provider: scriptProvider()}, runJob{})
	if err != nil || !reflect.DeepEqual(capability, &backend.SimulationCapability{Role: backend.SimulationRoleCoordinator}) {
		t.Fatalf("process executor simulation capability = %#v, %v", capability, err)
	}
	capability, err = simulationCapabilityForJob(&fakeExecutor{}, runJob{})
	if err != nil || capability != nil {
		t.Fatalf("substituted executor simulation capability = %#v, %v", capability, err)
	}
}

func TestInjectionCharacterizationReplayRequiresSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	artifactPath, observed := publishReplayArtifactForTarget(t, nil, replayArtifactTarget{})
	provider := native.New(native.Config{ToolchainRoot: toolchainRoot(t)})
	_, err := Replay(context.Background(), ReplaySpec{Backend: provider, ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t)})
	if err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("replay without supervisor or executor error = %v", err)
	}
	if _, err := Replay(context.Background(), ReplaySpec{Backend: provider, ArtifactPath: artifactPath, VerifyOnly: true, ToolchainRoot: toolchainRoot(t)}); err != nil {
		t.Fatalf("verify-only replay without supervisor: %v", err)
	}
	executor := &fakeReplayExecutor{result: observed}
	result, err := replayWith(context.Background(), ReplaySpec{Backend: provider, ArtifactPath: artifactPath, ToolchainRoot: toolchainRoot(t)}, executionDependencies{executor: executor})
	if err != nil || !result.Match {
		t.Fatalf("substituted replay without supervisor = %#v, %v", result, err)
	}
	description, err := provider.Describe(t.Context(), backend.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	if executor.calls != 1 || executor.request.Simulation != nil || executor.request.IO == nil || !reflect.DeepEqual(executor.request.IO.Profile, description.IOProfile) {
		t.Fatalf("substituted replay request calls=%d simulation=%#v profile=%#v", executor.calls, executor.request.Simulation, executor.request.IO)
	}
}

func TestInjectionCharacterizationMinimizeRequiresSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	spec, dependency := minimizationSpec(t, minimizationParent(t), t.TempDir())
	spec.Backend, dependency.executor = native.New(native.Config{ToolchainRoot: spec.ToolchainRoot}), nil
	if _, err := minimizeWith(context.Background(), spec, dependency); err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("minimize without supervisor or executor error = %v", err)
	}
	spec, dependency = minimizationSpec(t, minimizationParent(t), t.TempDir())
	spec.Backend = native.New(native.Config{ToolchainRoot: spec.ToolchainRoot})
	result, err := minimizeWith(context.Background(), spec, dependency)
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

func (executor *replayForwardingExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	if request.IO != nil && request.IO.Transcript != nil && request.IO.Transcript.Replay {
		executor.replays++
		return backend.ExecutionResult{}, errReplayReachedSubstitutedExecutor
	}
	return executor.candidates.Run(ctx, request)
}

func TestInjectionCharacterizationMinimizeDefaultReplayerUsesItsExecutor(t *testing.T) {
	spec, dependency := minimizationSpec(t, minimizationParent(t), t.TempDir())
	executor := &replayForwardingExecutor{}
	dependency.executor, spec.Replayer = executor, nil
	_, err := minimizeWith(context.Background(), spec, dependency)
	if !errors.Is(err, errReplayReachedSubstitutedExecutor) || !strings.HasPrefix(err.Error(), "replay minimization candidate: execute replay target: ") {
		t.Fatalf("minimize default replay error = %v", err)
	}
	if executor.replays != 1 || executor.candidates.calls == 0 {
		t.Fatalf("minimize executor replays=%d candidates=%d", executor.replays, executor.candidates.calls)
	}
}

func TestInjectionCharacterizationShardChecksToolchainOnlyForTheProcessExecutor(t *testing.T) {
	config, _ := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	planPath := filepath.Join(t.TempDir(), "campaign.plan.json")
	if _, err := CreateCampaignPlan(context.Background(), CampaignPlanSpec{Campaign: config, Output: planPath}); err != nil {
		t.Fatal(err)
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-toolchain")
	_, err := RunCampaignShard(context.Background(), CampaignShardSpec{
		Backend:  native.New(native.Config{ToolchainRoot: missingRoot, SupervisorCommand: []string{"unused"}}),
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		ToolchainRoot: missingRoot,
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("process shard with a missing toolchain error = %v", err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	result, err := runCampaignShardWith(context.Background(), CampaignShardSpec{
		Backend:  scriptProvider(),
		PlanPath: planPath, Shard: CampaignShard{Index: 0, Count: 1}, Artifacts: t.TempDir(), RunnerBuild: config.RunnerBuild,
		ToolchainRoot: missingRoot,
	}, executionDependencies{executor: executor})
	if err != nil || result.Succeeded != 1 || len(executor.requests) != 1 {
		t.Fatalf("substituted shard with a missing toolchain = %#v requests=%d, %v", result, len(executor.requests), err)
	}
}

func TestInjectionCharacterizationResumeChecksToolchainAndSupervisorOnlyForTheProcessExecutor(t *testing.T) {
	config, dependency := testConfig(t, newFakePreparer(t), &resumeInterruptExecutor{}, "7-8", PolicyAll, 1)
	ctx := cancelOnProgress(t, &config, func(event CampaignEvent) bool { return event.Succeeded == 1 })
	partial, err := exploreWith(ctx, config, dependency)
	if err == nil {
		t.Fatal("Explore() was not interrupted")
	}
	missingRoot := filepath.Join(t.TempDir(), "missing-toolchain")
	_, err = Resume(context.Background(), ResumeSpec{Backend: native.New(native.Config{ToolchainRoot: missingRoot}), CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot})
	if err == nil || err.Error() != "supervisor command is required" {
		t.Fatalf("resume without supervisor or executor error = %v", err)
	}
	_, err = Resume(context.Background(), ResumeSpec{Backend: native.New(native.Config{ToolchainRoot: missingRoot, SupervisorCommand: []string{"unused"}}), CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("process resume with a missing toolchain error = %v", err)
	}
	executor := &fakeExecutor{result: func(uint64) backend.ExecutionResult { return processResult(0, "", "") }}
	resumed, err := resumeWith(context.Background(), ResumeSpec{Backend: scriptProvider(), CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild, ToolchainRoot: missingRoot}, executionDependencies{executor: executor})
	if err != nil || resumed.Attempted != 2 || resumed.Succeeded != 2 || len(executor.requests) != 1 {
		t.Fatalf("substituted resume with a missing toolchain = %#v requests=%d, %v", resumed, len(executor.requests), err)
	}
}

func TestInjectionCharacterizationIsolatedExploreRejectsEverySubstitution(t *testing.T) {
	for name, configure := range map[string]func(*CampaignSpec, *executionDependencies){
		"executor": func(_ *CampaignSpec, dependency *executionDependencies) { dependency.executor = &fakeExecutor{} },
		"preparer": func(config *CampaignSpec, _ *executionDependencies) { config.Preparer = targetPreparer{} },
		"replayer": func(config *CampaignSpec, _ *executionDependencies) { config.Replayer = artifactReplayer{} },
	} {
		t.Run(name, func(t *testing.T) {
			config, dependency := testConfig(t, nil, nil, "1", PolicyAll, 1)
			config.Backend = native.New(native.Config{CoordinatorCommand: []string{"unused", "__coordinator"}})
			configure(&config, &dependency)
			if _, err := exploreWith(context.Background(), config, dependency); err == nil || err.Error() != "isolated Runner does not accept injected preparation or execution" {
				t.Fatalf("isolated Explore with an injected %s error = %v", name, err)
			}
		})
	}
}

func TestInjectionCharacterizationIsolatedPreparationDependencies(t *testing.T) {
	for _, test := range []struct {
		name          string
		prepare       bool
		execution     bool
		invalidSeeds  bool
		missingResume bool
	}{
		{name: "prepare only", prepare: true},
		{name: "execution only", execution: true},
		{name: "combined", prepare: true, execution: true},
		{name: "injection before seed validation", prepare: true, execution: true, invalidSeeds: true},
		{name: "resume preflight before injection", prepare: true, execution: true, missingResume: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			config, dependencies := testConfig(t, nil, nil, "7", PolicyAll, 1)
			marker := filepath.Join(t.TempDir(), "coordinator-started")
			command := filepath.Join(t.TempDir(), "coordinator")
			if err := os.WriteFile(command, []byte("#!/bin/sh\n: > \"$1\"\nexit 91\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			config.Backend = native.New(native.Config{CoordinatorCommand: []string{command, marker}})
			if test.prepare {
				dependencies.prepare = func(context.Context, preparationRequest) (target.Prepared, error) {
					calls.Add(1)
					return target.Prepared{}, errors.New("isolated preparation callback invoked")
				}
			}
			if test.execution {
				dependencies.executor = preparationFixtureExecutor{run: func(context.Context, backend.ExecutionRequest) (backend.ExecutionResult, error) {
					calls.Add(1)
					return backend.ExecutionResult{}, errors.New("isolated execution callback invoked")
				}}
			}
			if test.invalidSeeds {
				config.Seeds = "invalid"
			}
			if test.missingResume {
				config.ResumeCampaign = filepath.Join(t.TempDir(), "missing-campaign")
			}
			result, err := exploreWith(context.Background(), config, dependencies)
			if test.missingResume {
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("resume preflight error = %v", err)
				}
			} else if err == nil || err.Error() != "isolated Runner does not accept injected preparation or execution" {
				t.Fatalf("isolated injection error = %v", err)
			}
			if (dependencies.executor != nil) != test.execution || calls.Load() != 0 || !reflect.DeepEqual(result, CampaignResult{}) {
				t.Fatalf("isolated execution ran: executor=%v callbacks=%d result=%#v", dependencies.executor, calls.Load(), result)
			}
			if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("coordinator started: %v", err)
			}
		})
	}
}
