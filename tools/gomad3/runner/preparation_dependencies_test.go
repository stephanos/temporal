package runner

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

type preparationFixtureExecutor struct {
	run func(context.Context, backend.ExecutionRequest) (backend.ExecutionResult, error)
}

func (executor preparationFixtureExecutor) Run(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	return executor.run(ctx, request)
}

func TestPreparationDependenciesForwardRealFixtureInputs(t *testing.T) {
	preparer := newFakePreparer(t)
	var mu sync.Mutex
	var requestSeeds, executionSeeds []uint64
	var outputs []*os.File
	var prepared target.Prepared
	var description backend.Description
	var runnerBuild string
	executor := preparationFixtureExecutor{run: func(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
		if request.IO == nil || !reflect.DeepEqual(request.IO.Profile, description.IOProfile) || !reflect.DeepEqual(request.Target, prepared) || request.RunnerBuild != runnerBuild {
			t.Errorf("forwarded execution inputs = %#v", request)
		}
		if request.Command != prepared.Path || request.Argv0 != prepared.Argv[0] || !slices.Equal(request.Args, prepared.Argv[1:]) {
			t.Errorf("forwarded target = %#v", request)
		}
		data, err := os.ReadFile(request.Command)
		if err != nil || string(data) != "fake prepared target" {
			t.Errorf("copied target bytes = %q, %v", data, err)
		}
		info, err := os.Stat(request.Command)
		if err != nil {
			return backend.ExecutionResult{}, err
		}
		if info.Mode().Perm() != 0o500 || uint64(info.Size()) != prepared.Size {
			t.Errorf("copied target mode/size = %v/%d", info.Mode(), info.Size())
		}
		if err := verifyPrepared(prepared); err != nil {
			return backend.ExecutionResult{}, err
		}
		var runOutputs []*os.File
		for _, writer := range []io.Writer{request.StdoutHead, request.StderrHead} {
			output, ok := writer.(*os.File)
			if !ok {
				t.Errorf("execution output = %T; want the real partial file", writer)
				continue
			}
			if _, err := output.Stat(); err != nil {
				return backend.ExecutionResult{}, err
			}
			runOutputs = append(runOutputs, output)
		}
		mu.Lock()
		requestSeeds = append(requestSeeds, request.Seed)
		executionSeeds = append(executionSeeds, seedFromEnvironment(request.Env))
		outputs = append(outputs, runOutputs...)
		mu.Unlock()
		return processResult(0, "", ""), nil
	}}
	config, _ := testConfig(t, preparer, executor, "7-9", PolicyAll, 2)
	runnerBuild = config.RunnerBuild
	var err error
	description, err = config.Backend.Describe(t.Context(), backend.Configuration{Target: config.Target})
	if err != nil {
		t.Fatal(err)
	}
	var campaignPath string
	config.Progress = func(event CampaignEvent) error {
		if event.Phase == ProgressPreparing {
			campaignPath = event.CampaignPath
		}
		return nil
	}
	dependencies := scriptedPreparationDependencies(t, preparer, executor)
	prepareFixture := dependencies.prepare
	dependencies.prepare = func(ctx context.Context, request preparationRequest) (target.Prepared, error) {
		want := config.Target
		want.PreparationRoot = filepath.Join(campaignPath, ".prepared")
		if !reflect.DeepEqual(request.Target, want) || !slices.Equal(request.Environment, config.Environment) || request.Preparer != preparer || request.Provider != config.Backend {
			t.Fatalf("preparation inputs = %#v; want %#v", request, want)
		}
		if ctx.Err() != nil {
			t.Fatal(ctx.Err())
		}
		var err error
		prepared, err = prepareFixture(ctx, request)
		if err != nil {
			return prepared, err
		}
		wantPrepared := preparer.prepared
		wantPrepared.Path = filepath.Join(want.PreparationRoot, "target")
		wantPrepared.Adapters = []record.TargetAdapter{}
		if !reflect.DeepEqual(prepared, wantPrepared) {
			t.Fatalf("prepared fixture = %#v; want %#v", prepared, wantPrepared)
		}
		return prepared, nil
	}
	result, err := exploreWith(context.Background(), config, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	slices.Sort(requestSeeds)
	slices.Sort(executionSeeds)
	if preparer.calls != 1 || result.Attempted != 3 || result.Succeeded != 3 || !slices.Equal(requestSeeds, []uint64{7, 8, 9}) || !slices.Equal(executionSeeds, requestSeeds) {
		t.Fatalf("calls=%d result=%#v request=%v execution=%v", preparer.calls, result, requestSeeds, executionSeeds)
	}
	for _, output := range outputs {
		if _, err := output.Stat(); !errors.Is(err, os.ErrClosed) {
			t.Fatalf("execution output remained open: %v", err)
		}
	}
}

func TestPreparationDependenciesOperationErrorsRemainUnchanged(t *testing.T) {
	primary := errors.New("injected operation failed")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := preparationRequest{Target: target.Spec{Kind: target.KindGoRun, Source: "fixture", PreparationRoot: t.TempDir()}, Environment: []string{"MODE=fixture"}, Provider: scriptProvider()}
	prepared := target.Prepared{Path: "fixture result", SHA256: "fixture digest", Size: 17}
	dependencies := executionDependencies{
		prepare: func(received context.Context, receivedRequest preparationRequest) (target.Prepared, error) {
			if received != ctx || !reflect.DeepEqual(receivedRequest, request) {
				t.Fatalf("prepare arguments = %v, %#v", received, receivedRequest)
			}
			return prepared, primary
		},
	}
	if result, err := dependencies.prepareTarget(ctx, request); !reflect.DeepEqual(result, prepared) || err != primary {
		t.Fatalf("prepare result = %#v, %v", result, err)
	}
}

func TestPreparationDependenciesFailuresStopAtOriginalStages(t *testing.T) {
	for _, operation := range []string{"prepare", "execution"} {
		t.Run(operation, func(t *testing.T) {
			primary := errors.New("injected " + operation + " failed")
			preparer := newFakePreparer(t)
			executor := &fakeExecutor{}
			config, _ := testConfig(t, preparer, executor, "7", PolicyAll, 1)
			dependencies := scriptedPreparationDependencies(t, preparer, executor)
			executionCalls := 0
			if operation == "prepare" {
				dependencies.prepare = func(context.Context, preparationRequest) (target.Prepared, error) { return target.Prepared{}, primary }
			}
			dependencies.executor = preparationFixtureExecutor{run: func(_ context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
				executionCalls++
				prepared := request.Target
				partial := filepath.Join(filepath.Dir(filepath.Dir(prepared.Path)), ".partial", "00000000000000000000-7")
				for _, name := range []string{"stdout.head", "stderr.head"} {
					if _, err := os.Stat(filepath.Join(partial, name)); err != nil {
						t.Error(err)
					}
				}
				return backend.ExecutionResult{}, primary
			}}
			result, err := exploreWith(context.Background(), config, dependencies)
			if !errors.Is(err, primary) || len(executor.directories()) != 0 || result.CampaignPath == "" {
				t.Fatalf("failed operation result = %#v, %v, executions=%v", result, err, executor.directories())
			}
			if operation == "prepare" {
				if err != primary || preparer.calls != 0 || executionCalls != 0 || result.Attempted != 0 {
					t.Fatalf("preparation failure changed: %v calls=%d executions=%d result=%#v", err, preparer.calls, executionCalls, result)
				}
			} else {
				var hostError *HostError
				if !errors.As(err, &hostError) || hostError.Reason != "target_supervision" || preparer.calls != 1 || executionCalls != 1 || result.Attempted != 1 {
					t.Fatalf("execution failure changed: %v calls=%d executions=%d result=%#v", err, preparer.calls, executionCalls, result)
				}
			}
		})
	}
}

func TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard(t *testing.T) {
	profile := transcript.Default()
	_, guard := profile.BootstrapFrame(target.Prepared{}, "invalid", 0)
	if guard == nil || !strings.HasPrefix(guard.Error(), "deterministic I/O requires one of ") {
		t.Skip("unsupported-host refusal control requires an unqualified host")
	}
	for _, mode := range []string{"public", "executor only", "prepare only"} {
		t.Run(mode, func(t *testing.T) {
			preparer := newFakePreparer(t)
			executor := &fakeExecutor{}
			config, dependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
			config.Backend = native.New(native.Config{SupervisorCommand: []string{"unused"}, RunnerBuild: config.RunnerBuild})
			var err error
			if mode == "public" {
				_, err = Explore(context.Background(), config)
			} else {
				if mode == "prepare only" {
					dependencies = scriptedPreparationDependencies(t, preparer, executor)
					dependencies.executor = nil
				}
				_, err = exploreWith(context.Background(), config, dependencies)
			}
			if err == nil || !strings.Contains(err.Error(), guard.Error()) || preparer.calls != 1 || len(executor.directories()) != 0 {
				t.Fatalf("%s default = %v calls=%d executions=%v", mode, err, preparer.calls, executor.directories())
			}
			if mode == "prepare only" {
				var hostError *HostError
				if !errors.As(err, &hostError) || hostError.Reason != "target_supervision" {
					t.Fatalf("real bootstrap guard = %v", err)
				}
			} else {
				var preparationError *backend.PreparationError
				if !errors.As(err, &preparationError) || preparationError.Stage != "validation" {
					t.Fatalf("real preparation stage = %v", err)
				}
			}
		})
	}
}
