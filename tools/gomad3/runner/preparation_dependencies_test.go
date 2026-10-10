package runner

import (
	"bytes"
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

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/preparation"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

type preparationFixtureExecutor struct {
	run func(context.Context, execution.Spec) (execution.Result, error)
}

func (executor preparationFixtureExecutor) Run(ctx context.Context, request execution.Spec) (execution.Result, error) {
	return executor.run(ctx, request)
}

func TestPreparationDependenciesForwardRealFixtureInputs(t *testing.T) {
	preparer := newFakePreparer(t)
	var mu sync.Mutex
	var bootstrapSeeds, executionSeeds []uint64
	var outputs []*os.File
	var prepared target.Prepared
	executor := preparationFixtureExecutor{run: func(_ context.Context, request execution.Spec) (execution.Result, error) {
		if request.IO == nil || !bytes.Equal(request.IO.Config, []byte(scriptedBootstrapMarker)) {
			t.Errorf("forwarded bootstrap = %#v", request.IO)
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
			return execution.Result{}, err
		}
		if info.Mode().Perm() != 0o500 || uint64(info.Size()) != prepared.Size {
			t.Errorf("copied target mode/size = %v/%d", info.Mode(), info.Size())
		}
		if err := prepared.Verify(); err != nil {
			return execution.Result{}, err
		}
		var runOutputs []*os.File
		for _, writer := range []io.Writer{request.StdoutHead, request.StderrHead} {
			output, ok := writer.(*os.File)
			if !ok {
				t.Errorf("execution output = %T; want the real partial file", writer)
				continue
			}
			if _, err := output.Stat(); err != nil {
				return execution.Result{}, err
			}
			runOutputs = append(runOutputs, output)
		}
		mu.Lock()
		executionSeeds = append(executionSeeds, seedFromEnvironment(request.Env))
		outputs = append(outputs, runOutputs...)
		mu.Unlock()
		return processResult(0, "", ""), nil
	}}
	config, _ := testConfig(t, preparer, executor, "7-9", PolicyAll, 2)
	var campaignPath string
	config.Progress = func(event CampaignEvent) error {
		if event.Phase == ProgressPreparing {
			campaignPath = event.CampaignPath
		}
		return nil
	}
	dependencies := scriptedPreparationDependencies(t, preparer, executor)
	prepareFixture, bootstrapFixture := dependencies.prepare, dependencies.bootstrap
	dependencies.prepare = func(ctx context.Context, request preparation.Request) (target.Prepared, error) {
		want := config.Target
		want.PreparationRoot = filepath.Join(campaignPath, ".prepared")
		if !reflect.DeepEqual(request.Target, want) || !slices.Equal(request.Environment, config.Environment) || request.Preparer != preparer {
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
	dependencies.bootstrap = func(profile deterministicio.Spec, received target.Prepared, runner string, seed uint64) ([]byte, error) {
		if profile != deterministicio.Default() || !reflect.DeepEqual(received, prepared) || runner != config.RunnerBuild {
			t.Errorf("bootstrap inputs = %#v, %#v, %q", profile, received, runner)
		}
		mu.Lock()
		bootstrapSeeds = append(bootstrapSeeds, seed)
		mu.Unlock()
		return bootstrapFixture(profile, received, runner, seed)
	}
	result, err := exploreWith(context.Background(), config, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	slices.Sort(bootstrapSeeds)
	slices.Sort(executionSeeds)
	if preparer.calls != 1 || result.Attempted != 3 || result.Succeeded != 3 || !slices.Equal(bootstrapSeeds, []uint64{7, 8, 9}) || !slices.Equal(executionSeeds, bootstrapSeeds) {
		t.Fatalf("calls=%d result=%#v bootstrap=%v execution=%v", preparer.calls, result, bootstrapSeeds, executionSeeds)
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
	request := preparation.Request{Target: target.Spec{Kind: target.KindGoRun, Source: "fixture", PreparationRoot: t.TempDir()}, Environment: []string{"MODE=fixture"}}
	prepared := target.Prepared{Path: "fixture result", SHA256: "fixture digest", Size: 17}
	profile := deterministicio.Default()
	marker := []byte(scriptedBootstrapMarker)
	dependencies := executionDependencies{
		prepare: func(received context.Context, receivedRequest preparation.Request) (target.Prepared, error) {
			if received != ctx || !reflect.DeepEqual(receivedRequest, request) {
				t.Fatalf("prepare arguments = %v, %#v", received, receivedRequest)
			}
			return prepared, primary
		},
		bootstrap: func(received deterministicio.Spec, receivedPrepared target.Prepared, runner string, seed uint64) ([]byte, error) {
			if received != profile || !reflect.DeepEqual(receivedPrepared, prepared) || runner != "fixture runner" || seed != 42 {
				t.Fatalf("bootstrap arguments = %#v, %#v, %q, %d", received, receivedPrepared, runner, seed)
			}
			return marker, primary
		},
	}
	if result, err := dependencies.prepareTarget(ctx, request); !reflect.DeepEqual(result, prepared) || err != primary {
		t.Fatalf("prepare result = %#v, %v", result, err)
	}
	if result, err := dependencies.bootstrapFrame(profile, prepared, "fixture runner", 42); !bytes.Equal(result, marker) || err != primary {
		t.Fatalf("bootstrap result = %q, %v", result, err)
	}
}

func TestPreparationDependenciesFailuresStopAtOriginalStages(t *testing.T) {
	for _, operation := range []string{"prepare", "bootstrap"} {
		t.Run(operation, func(t *testing.T) {
			primary := errors.New("injected " + operation + " failed")
			preparer := newFakePreparer(t)
			executor := &fakeExecutor{}
			config, _ := testConfig(t, preparer, executor, "7", PolicyAll, 1)
			dependencies := scriptedPreparationDependencies(t, preparer, executor)
			bootstrapCalls := 0
			if operation == "prepare" {
				dependencies.prepare = func(context.Context, preparation.Request) (target.Prepared, error) { return target.Prepared{}, primary }
			}
			dependencies.bootstrap = func(_ deterministicio.Spec, prepared target.Prepared, _ string, _ uint64) ([]byte, error) {
				bootstrapCalls++
				partial := filepath.Join(filepath.Dir(filepath.Dir(prepared.Path)), ".partial", "00000000000000000000-7")
				for _, name := range []string{"stdout.head", "stderr.head"} {
					if _, err := os.Stat(filepath.Join(partial, name)); err != nil {
						t.Error(err)
					}
				}
				return nil, primary
			}
			result, err := exploreWith(context.Background(), config, dependencies)
			if !errors.Is(err, primary) || len(executor.directories()) != 0 || result.CampaignPath == "" {
				t.Fatalf("failed operation result = %#v, %v, executions=%v", result, err, executor.directories())
			}
			if operation == "prepare" {
				if err != primary || preparer.calls != 0 || bootstrapCalls != 0 || result.Attempted != 0 {
					t.Fatalf("preparation failure changed: %v calls=%d bootstrap=%d result=%#v", err, preparer.calls, bootstrapCalls, result)
				}
			} else {
				var hostError *HostError
				if !errors.As(err, &hostError) || hostError.Reason != "target_supervision" || preparer.calls != 1 || bootstrapCalls != 1 || result.Attempted != 1 {
					t.Fatalf("bootstrap failure changed: %v calls=%d bootstrap=%d result=%#v", err, preparer.calls, bootstrapCalls, result)
				}
			}
		})
	}
}

func TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard(t *testing.T) {
	profile := deterministicio.Default()
	_, guard := profile.BootstrapFrame(target.Prepared{}, "invalid", 0)
	if guard == nil || !strings.HasPrefix(guard.Error(), "deterministic I/O requires one of ") {
		t.Skip("unsupported-host refusal control requires an unqualified host")
	}
	for _, mode := range []string{"public", "executor only", "prepare only"} {
		t.Run(mode, func(t *testing.T) {
			preparer := newFakePreparer(t)
			executor := &fakeExecutor{}
			config, dependencies := testConfig(t, preparer, executor, "7", PolicyAll, 1)
			var err error
			if mode == "public" {
				_, err = Explore(context.Background(), config)
			} else {
				if mode == "prepare only" {
					dependencies = scriptedPreparationDependencies(t, preparer, executor)
					dependencies.bootstrap = nil
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
			} else if preparation.StageOf(err) != preparation.StageValidation {
				t.Fatalf("real preparation stage = %v", err)
			}
		})
	}
}
