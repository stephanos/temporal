package measure

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	wasmbackend "go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism/observed"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measurement"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

var stressAxes = []string{"cpu", "concurrent", "layout", "handoff"}

const stressChoiceTraceLimit = 32 << 20

type stressCase struct {
	Workload string `json:"workload"`
	Seed     uint64 `json:"seed"`
}

func stressCases() []stressCase {
	var cases []stressCase
	for index := range uint64(32) {
		cases = append(cases, stressCase{"runtime", index * 1000003})
	}
	for _, seed := range []uint64{7, 19, 42} {
		cases = append(cases, stressCase{"two-cluster", seed})
	}
	for _, workload := range []string{"workflow", "activity"} {
		for _, seed := range []uint64{0, 1000003, 2000006} {
			cases = append(cases, stressCase{workload, seed})
		}
	}
	return cases
}

type stressObservation struct {
	Run          determinism.Run   `json:"run"`
	Directory    string            `json:"directory"`
	Installation string            `json:"installation"`
	Fresh        bool              `json:"fresh_seeded_execution"`
	Digests      map[string]string `json:"digests"`
	Coordinates  bool              `json:"coordinates_available"`
}

type stressCell struct {
	stressCase
	Axis         string               `json:"axis"`
	Requested    int                  `json:"requested"`
	Rounds       int                  `json:"rounds"`
	Fanout       int                  `json:"fanout"`
	Completed    int                  `json:"completed"`
	Status       string               `json:"status"`
	Inapplicable bool                 `json:"inapplicable,omitempty"`
	Reason       string               `json:"reason,omitempty"`
	Runs         []stressObservation  `json:"runs"`
	Comparisons  []determinism.Report `json:"comparisons"`
}

// Concurrent repetitions are full cohorts; every member is retained and compared.
func stressNewCell(test stressCase, axis string, repetitions, parallel int) stressCell {
	fanout := 1
	if axis == "concurrent" {
		fanout = parallel
	}
	return stressCell{stressCase: test, Axis: axis, Requested: repetitions * fanout, Rounds: repetitions, Fanout: fanout}
}

type stressCohortKey struct{}

func stressCohortContext(ctx context.Context, fanout int) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	var mu sync.Mutex
	arrived := 0
	ready := make(chan struct{})
	wait := func() {
		mu.Lock()
		arrived++
		if arrived == fanout {
			close(ready)
		}
		mu.Unlock()
		select {
		case <-ready:
		case <-ctx.Done():
		}
	}
	return context.WithValue(ctx, stressCohortKey{}, wait), cancel
}

func stressCompare(reference, actual stressObservation) (determinism.Report, error) {
	if !reference.Fresh || !actual.Fresh {
		return determinism.Report{}, fmt.Errorf("%w: forced replay is not a stress cell", ErrRefused)
	}
	report, err := determinism.Compare(reference.Run, actual.Run)
	if err != nil {
		return report, fmt.Errorf("%w: %v", ErrRefused, err)
	}
	if !maps.Equal(reference.Digests, actual.Digests) && (report.Status == "identical" || !reference.Coordinates || !actual.Coordinates) {
		return determinism.Report{Schema: determinism.ReportSchema, Status: "undiagnosable", Reason: "canonical evidence differs without complete inspection coordinates; retain --diagnostics and inspection metadata"}, nil
	}
	if report.Reproducer != nil && len(report.Reproducer.ReplayCommand) >= 2 && reference.Installation != "" {
		report.Reproducer.ReplayCommand = append([]string{"gomad", "--installation", reference.Installation, "replay", "--stress-jitter"}, report.Reproducer.ReplayCommand[2:]...)
	}
	return report, nil
}

func (cell *stressCell) finish() {
	cell.Completed = len(cell.Runs)
	cell.Status = "pass"
	for _, report := range cell.Comparisons {
		if report.Status != "identical" {
			cell.Status = "divergence"
			return
		}
	}
	if cell.Completed != cell.Requested || len(cell.Comparisons) != cell.Requested || cell.Reason != "" {
		cell.Status = "skipped"
	}
}

func stressPositive(ctx context.Context, key string, fallback int) (int, error) {
	value := input(ctx, key)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > 256 {
		return 0, fmt.Errorf("%w: --%s must be 1..256", ErrRefused, measurement.Flag(key))
	}
	return n, nil
}

func stressSource(ctx context.Context) (map[string]string, error) {
	repo := scopeFor(ctx).options.Repo
	command := exec.CommandContext(ctx, "git", "-C", repo, "ls-files", "-z", "--", "*.go", "*.txt", "*.rs", "*.toml", "*.lock", "*go.mod", "*go.sum", "*json")
	listed, err := command.Output()
	if err != nil {
		return nil, err
	}
	inputs := map[string]string{}
	for _, name := range bytes.Split(bytes.TrimRight(listed, "\x00"), []byte{0}) {
		path := string(name)
		if strings.HasPrefix(path, ".flow/") {
			continue
		}
		hash, err := fileHash(ctx, filepath.Join(repo, path))
		if err != nil {
			return nil, err
		}
		inputs[path] = hash
	}
	return inputs, nil
}

func measureStress(ctx context.Context) (runErr error) {
	root := input(ctx, "GOMAD_WASM_STRESS_ROOT")
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		return fmt.Errorf("%w: stress root must be an absolute clean retained directory", ErrRefused)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: stress root must not already exist", ErrRefused)
	}
	if input(ctx, "GOMAD_WASM_STRESS_HOST_DESCRIPTION") == "" {
		return fmt.Errorf("%w: provide the host identity and quiet-host admission", ErrRefused)
	}
	repetitions, err := stressPositive(ctx, "GOMAD_WASM_STRESS_REPETITIONS", 3)
	if err != nil {
		return err
	}
	parallel, err := stressPositive(ctx, "GOMAD_WASM_STRESS_PARALLEL", 2)
	if err != nil {
		return err
	}
	workers, err := stressPositive(ctx, "GOMAD_WASM_STRESS_CPU_WORKERS", max(1, runtime.NumCPU()-1))
	if err != nil {
		return err
	}
	if repetitions < 2 || parallel < 2 {
		return fmt.Errorf("%w: repetitions and concurrent fan-out must be at least two", ErrRefused)
	}
	if only := input(ctx, "GOMAD_WASM_STRESS_ONLY"); only != "" && only != "runtime" && only != "two-cluster" && only != "workflow" && only != "activity" {
		return fmt.Errorf("%w: unknown --stress-only workload %q", ErrRefused, only)
	}
	if seed := input(ctx, "GOMAD_WASM_STRESS_SEED"); seed != "" {
		if _, err := strconv.ParseUint(seed, 10, 64); err != nil {
			return fmt.Errorf("%w: invalid --stress-seed", ErrRefused)
		}
	}
	if axis := input(ctx, "GOMAD_WASM_STRESS_AXIS"); axis != "" && axis != "reference" && axis != "cpu" && axis != "concurrent" && axis != "layout" && axis != "handoff" {
		return fmt.Errorf("%w: unknown --stress-axis %q", ErrRefused, axis)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	options := scopeFor(ctx).options
	commandInputs := maps.Clone(options.Inputs)
	commandInputs["GOMAD_WASM_STRESS_ROOT"] = root + "-rerun"
	runCommand := measurement.Command("go", options.Repo, "stress", commandInputs)
	setInput(ctx, "GOMAD_WASM_PROTOTYPE_ROOT", root)
	setInput(ctx, "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", root)
	sources, err := stressSource(ctx)
	if err != nil {
		return err
	}
	encodedSources, err := json.Marshal(sources)
	if err != nil {
		return err
	}
	sourceID := digest(encodedSources)
	if err := temporalJSON(ctx, filepath.Join(root, "source-manifest.json"), sources); err != nil {
		return err
	}
	hostBoot, hostCPU, hostErr := warmPreparationHostEvidence(ctx)
	if hostErr != nil {
		return fmt.Errorf("%w: current host CPU evidence is unavailable", ErrRefused)
	}
	var cells []stressCell
	var references []stressObservation
	defer func() {
		seen := map[string]bool{}
		for _, cell := range cells {
			seen[fmt.Sprintf("%s/%d/%s", cell.Workload, cell.Seed, cell.Axis)] = true
		}
		for _, test := range stressCases() {
			for _, axis := range stressAxes {
				if !seen[fmt.Sprintf("%s/%d/%s", test.Workload, test.Seed, axis)] {
					cell := stressNewCell(test, axis, repetitions, parallel)
					cell.Reason = "producer stopped before execution"
					cell.finish()
					cells = append(cells, cell)
				}
			}
		}
		complete := runErr == nil
		for _, cell := range cells {
			complete = complete && (cell.Status == "pass" || cell.Inapplicable)
		}
		current, err := stressSource(ctx)
		if err != nil {
			runErr = errors.Join(runErr, err)
			complete = false
		} else if !maps.Equal(sources, current) {
			runErr = errors.Join(runErr, fmt.Errorf("%w: candidate sources changed during stress", ErrRefused))
			complete = false
		}
		runErr = errors.Join(runErr, temporalJSON(ctx, filepath.Join(root, "report.json"), map[string]any{"schema": "gomad.wasm-determinism-stress/v1", "complete": complete, "cells": cells, "references": references, "source_sha256": sourceID, "host_cpu": hostCPU, "host_boot": hostBoot, "host_description": input(ctx, "GOMAD_WASM_STRESS_HOST_DESCRIPTION"), "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "repetitions": repetitions, "cpu_workers": workers, "concurrent_instances": parallel, "rerun": runCommand}))
	}()
	requests := map[string]wasi.Request{}
	var failures error
	for _, test := range stressCases() {
		if only, seed := input(ctx, "GOMAD_WASM_STRESS_ONLY"), input(ctx, "GOMAD_WASM_STRESS_SEED"); only != "" && only != test.Workload || seed != "" && seed != strconv.FormatUint(test.Seed, 10) {
			for _, axis := range stressAxes {
				cell := stressNewCell(test, axis, repetitions, parallel)
				cell.Reason = "excluded by --stress-only"
				cell.finish()
				cells = append(cells, cell)
			}
			continue
		}
		request, ok := requests[test.Workload]
		if !ok {
			request, err = stressRequest(ctx, root, test.Workload)
			if err != nil {
				return err
			}
			requests[test.Workload] = request
		}
		request.Runtime.Seed = test.Seed
		base := filepath.Join(root, fmt.Sprintf("%s-%d", test.Workload, test.Seed))
		reference, err := stressExecute(ctx, test, request, sourceID, "reference", 0, base, runCommand)
		if err != nil {
			return err
		}
		references = append(references, stressSummary(reference))
		for _, axis := range stressAxes {
			cell := stressNewCell(test, axis, repetitions, parallel)
			if selected := input(ctx, "GOMAD_WASM_STRESS_AXIS"); selected != "" && selected != axis {
				cell.Reason = "excluded by --stress-axis"
				cell.finish()
				cells = append(cells, cell)
				continue
			}
			if axis == "handoff" && test.Workload != "two-cluster" {
				cell.Inapplicable = true
				cell.Reason = "helper-owned single guest has no cluster baton or remote replies to perturb"
				cell.finish()
				cells = append(cells, cell)
				continue
			}
			stop := func() {}
			if axis == "cpu" {
				stop = stressCPULoad(ctx, workers)
			}
			results := make([]stressObservation, cell.Requested)
			errorsByRun := make([]error, cell.Requested)
			for first := 0; first < cell.Requested; first += cell.Fanout {
				cohort, cancel := stressCohortContext(ctx, cell.Fanout)
				var group sync.WaitGroup
				for index := first; index < first+cell.Fanout; index++ {
					group.Go(func() {
						results[index], errorsByRun[index] = stressExecute(cohort, test, request, sourceID, axis, index, base, runCommand)
						if errorsByRun[index] != nil {
							cancel()
						}
					})
				}
				group.Wait()
				cancel()
			}
			stop()
			for index, observation := range results {
				if errorsByRun[index] != nil {
					cell.Reason += fmt.Sprintf("execution %d: %v; ", index, errorsByRun[index])
					failures = errors.Join(failures, errorsByRun[index])
					continue
				}
				cell.Runs = append(cell.Runs, stressSummary(observation))
				comparison, err := stressCompare(reference, observation)
				if err != nil {
					failures = errors.Join(failures, err)
					cell.Reason += err.Error()
					continue
				}
				cell.Comparisons = append(cell.Comparisons, comparison)
			}
			cell.finish()
			if cell.Status != "pass" {
				failures = errors.Join(failures, fmt.Errorf("stress %s/%d/%s: %s", test.Workload, test.Seed, axis, cell.Status))
			}
			cells = append(cells, cell)
			logf(ctx, "%s/%d/%s: %s (%d/%d)", test.Workload, test.Seed, axis, cell.Status, cell.Completed, cell.Requested)
			if err := temporalJSON(ctx, filepath.Join(root, "report.json"), map[string]any{"schema": "gomad.wasm-determinism-stress/v1", "complete": false, "cells": cells, "references": references, "source_sha256": sourceID, "host_cpu": hostCPU, "host_description": input(ctx, "GOMAD_WASM_STRESS_HOST_DESCRIPTION"), "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "rerun": runCommand}); err != nil {
				return err
			}
		}
	}
	return failures
}

func stressCPULoad(ctx context.Context, workers int) func() {
	load, cancel := context.WithCancel(ctx)
	var group sync.WaitGroup
	for range workers {
		group.Go(func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			for load.Err() == nil {
				value := uint64(1)
				for range 100000 {
					value = value*6364136223846793005 + 1
				}
				runtime.KeepAlive(value)
			}
		})
	}
	return func() { cancel(); group.Wait() }
}

func stressRequest(ctx context.Context, root, workload string) (wasi.Request, error) {
	helper, err := helperPath(ctx)
	if err != nil {
		return wasi.Request{}, err
	}
	helperSHA, err := fileHash(ctx, helper)
	if err != nil {
		return wasi.Request{}, err
	}
	request := wasi.Request{HelperPath: helper, HelperSHA256: helperSHA, Engine: wasi.PinnedEngine(), Config: standaloneConfig(), MemoryBytes: 4 << 30, Fuel: 1 << 50, Timeout: 15 * time.Minute, CompileCache: filepath.Join(root, "compile-cache"), Runtime: &wasi.RuntimeControl{Diagnostics: true, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: stressChoiceTraceLimit}}}
	request.Config.Profile = wasi.CooperativeProfile
	switch workload {
	case "runtime":
		request.Config.Environment = append(request.Config.Environment, "EXTRA_ENTROPY=no_")
		request.Config.Limits.TranscriptBytes = record.DefaultIOTranscriptBytes
		request.Runtime.Choice.Limit = 1 << 20
		request.MemoryBytes, request.Fuel = 256<<20, 4000000000
		request.Timeout = time.Minute
	case "two-cluster":
		request.Config, err = temporalRecoveryConfig(scopeFor(ctx).options.Repo)
		if err != nil {
			return request, err
		}
		request.Config.Args = append([]string{"replication.test"}, stressClusterArgs...)
		request.Config.NetworkV2 = stressClusterNetwork()
		request.Config.Profile = wasi.CooperativeProfile
		spec, err := stressTarget(ctx, workload, request)
		if err != nil {
			return request, err
		}
		if err := temporalStageBrokerFixture(ctx, scopeFor(ctx).options.Repo, strings.TrimPrefix(spec.Source, "./")); err != nil {
			return request, err
		}
	case "workflow", "activity":
		selection := "^TestActivityTestSuite$"
		if workload == "workflow" {
			selection = "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"
		}
		config, configErr := prototypeTemporalConfig(ctx, scopeFor(ctx).options.Repo, "/workspace/tests", "-test.run="+selection, "-persistenceType=sql", "-persistenceDriver=sqlite")
		if configErr != nil {
			return request, configErr
		}
		request.Config = config
	default:
		return request, fmt.Errorf("unknown stress workload %q", workload)
	}
	if workload != "runtime" {
		request.Config.Clock.ReadStepNanos = 0
	}
	request.SetStressHostConditions("", false)
	return request, nil
}

func stressExecute(ctx context.Context, test stressCase, original wasi.Request, sourceID, axis string, repetition int, base string, command []string) (stressObservation, error) {
	parent := scopeFor(ctx)
	child := &runScope{runResources: parent.runResources, options: parent.options, name: parent.name}
	child.options.Inputs = maps.Clone(parent.options.Inputs)
	var observation stressObservation
	err := child.execute(context.WithValue(ctx, runKey{}, child), func(ctx context.Context) error {
		var err error
		observation, err = stressExecuteOne(ctx, test, original, sourceID, axis, repetition, base, command)
		return err
	})
	return observation, err
}

func stressExecuteOne(ctx context.Context, test stressCase, original wasi.Request, sourceID, axis string, repetition int, base string, command []string) (stressObservation, error) {
	request := original
	runtimeControl := *original.Runtime
	choiceControl := *original.Runtime.Choice
	runtimeControl.Choice = &choiceControl
	request.Runtime = &runtimeControl
	if request.Runtime.Choice.Mode != choice.ModeRecord || request.Runtime.Choice.Tape != nil {
		return stressObservation{}, fmt.Errorf("%w: stress requires fresh seeded execution", ErrRefused)
	}
	root := filepath.Join(base, fmt.Sprintf("%s-%03d", axis, repetition))
	if err := os.MkdirAll(root, 0700); err != nil {
		return stressObservation{}, err
	}
	padding := ""
	if axis == "layout" {
		padding = strings.Repeat("x", 1+(repetition%16)*4096)
		request.CompileCache = filepath.Join(filepath.Dir(original.CompileCache), strings.Repeat("p", 1+(repetition%4)*40), "compile-cache")
	}
	request.SetStressHostConditions(padding, axis == "handoff")
	rerun := append([]string(nil), command...)
	for index := 0; index+1 < len(rerun); index++ {
		if rerun[index] == "--stress-root" {
			rerun[index+1] = root + "-rerun"
		}
	}
	rerun = append(rerun, "--stress-only", test.Workload, "--stress-seed", strconv.FormatUint(test.Seed, 10), "--stress-axis", axis)
	compiler, err := fileHash(ctx, input(ctx, "GOMAD_STOCK_GO"))
	if err != nil {
		return stressObservation{}, err
	}
	observation := stressObservation{Directory: root, Fresh: true, Digests: map[string]string{}, Run: determinism.Run{Workload: test.Workload, Seed: test.Seed, Perturbation: axis, Command: rerun,
		Identities: map[string]string{"source": sourceID, "helper": request.HelperSHA256, "module": request.ModuleSHA256, "engine": request.DiagnosticConfigurationRevision(), "engine_version": request.Engine.Version, "compiler": compiler, "model": wasi.ImplementationSHA256(), "runtime": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, "host": runtime.GOOS + "/" + runtime.GOARCH}}}
	if err := os.MkdirAll(request.CompileCache, 0700); err != nil {
		return observation, err
	}
	spec, err := stressTarget(ctx, test.Workload, request)
	if err != nil {
		return observation, err
	}
	stress := wasmbackend.StressExecution{Padding: padding, Jitter: axis == "handoff"}
	if wait, ok := ctx.Value(stressCohortKey{}).(func()); ok {
		stress.ConfigureRequest = func(*wasi.Request) { wait() }
	}
	if test.Workload == "two-cluster" {
		stress.BindScenario = stressBindCluster
	}
	var collectErr error
	canary := false
	provider, err := stressRunnerProvider(ctx, input(ctx, "GOMAD_WASM_STRESS_ROOT"), request, stress, func(execution wasmbackend.ExecutionObservation) {
		observation.Run.Identities["declared_inputs"] = string(execution.DeclaredInputsSHA256)
		observation.Run.Identities["module"] = strings.TrimPrefix(execution.Target.SHA256, "sha256:")
		observation.Run.Identities["runtime"] = execution.Target.BuildKey
		implementation, err := choice.ImplementationIdentity(execution.Target.BuildKey)
		if err != nil {
			collectErr = err
			return
		}
		module, err := hex.DecodeString(strings.TrimPrefix(execution.Target.SHA256, "sha256:"))
		if err != nil || len(module) != 32 {
			collectErr = errors.New("prepared stress target has invalid module identity")
			return
		}
		identity := choice.ExecutionIdentity{ToolchainBuildKey: execution.Target.BuildKey, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: implementation}
		copy(identity.TargetSHA256[:], module)
		request.Runtime.Choice.ExecutionIdentity = identity
		if execution.Spec != nil {
			bound, err := json.Marshal(execution.Spec)
			if err != nil {
				collectErr = err
				return
			}
			observation.Run.Identities["bound_inputs"] = digest(bound)
		}
		for _, incarnation := range execution.Incarnations {
			if err := stressCollect(ctx, root, string(incarnation.Handle.Node), uint64(incarnation.Handle.Incarnation), incarnation.Result, request.Runtime.Choice, &observation); err != nil {
				collectErr = err
				return
			}
		}
		var metadata []byte
		if test.Workload == "two-cluster" {
			if err := stressClusterOracles(ctx, root, execution.Incarnations); err != nil {
				collectErr = err
				return
			}
			if execution.Cluster == nil {
				collectErr = errors.New("cluster stress run lacks shared evidence")
				return
			}
			metadata = execution.Cluster.InspectionMetadata
			observation.Digests["world"] = digest(execution.Cluster.World)
			collectErr = temporalWrite(ctx, filepath.Join(root, "world.bin"), execution.Cluster.World)
			if collectErr != nil {
				return
			}
		} else {
			if len(execution.Incarnations) != 1 {
				collectErr = errors.New("single guest stress run has wrong incarnation cardinality")
				return
			}
			if err := stressRequireExit(ctx, test.Workload, execution.Incarnations[0].Result); err != nil {
				collectErr = err
				return
			}
			canary = test.Workload == "runtime" && *execution.Incarnations[0].Result.ExitCode == 17
			metadata = execution.Incarnations[0].Result.InspectionMetadata
		}
		if err := temporalWrite(ctx, filepath.Join(root, "inspection.json"), metadata); err != nil {
			collectErr = err
			return
		}
		collectErr = stressInspection(metadata, &observation)
	})
	if err != nil {
		return observation, err
	}
	observation.Installation, err = stressRunnerInstallation(input(ctx, "GOMAD_WASM_STRESS_ROOT"), request)
	if err != nil {
		return observation, err
	}
	campaign, err := stressRunnerCampaign(ctx, input(ctx, "GOMAD_WASM_STRESS_ROOT"), provider, spec, request)
	if err != nil {
		return observation, err
	}
	if collectErr != nil {
		return observation, collectErr
	}
	if len(observation.Run.Incarnations) == 0 {
		return observation, errors.New("stress campaign did not observe its fresh execution")
	}
	observation.Run.ArtifactDir, err = stressArtifactPath(campaign, canary)
	if err != nil {
		return observation, err
	}
	opened, err := artifact.OpenArtifact(observation.Run.ArtifactDir)
	if err != nil {
		return observation, err
	}
	manifest := opened.Manifest()
	if err := opened.Close(); err != nil {
		return observation, err
	}
	if err := stressArtifactOutcome(manifest, canary); err != nil {
		return observation, err
	}
	encoded, err := json.Marshal(observation.Digests)
	if err != nil {
		return observation, err
	}
	observation.Run.EvidenceDigest = digest(encoded)
	return observation, temporalJSON(ctx, filepath.Join(root, "run.json"), stressSummary(observation))
}

func stressRequireExit(ctx context.Context, workload string, result wasi.Result) error {
	if result.Cancelled || result.WatchdogTimeout {
		return errors.New("stress execution was cancelled or timed out")
	}
	codes := []uint32{0}
	if workload == "runtime" {
		codes = append(codes, 17)
	}
	if err := requireExit(ctx, workload, result, codes...); err != nil {
		return err
	}
	if workload == "runtime" && *result.ExitCode == 17 && !bytes.Contains(result.Stdout, []byte("canary 2\n")) {
		return errors.New("runtime exit17 lacks its intentional canary witness")
	}
	return nil
}

func stressArtifactPath(campaign runner.CampaignResult, canary bool) (string, error) {
	if campaign.Attempted == 1 && campaign.Watchdogs == 0 && campaign.Cancelled == 0 && campaign.ReplayDivergences == 0 {
		if !canary && campaign.Succeeded == 1 && campaign.Failures == 0 && len(campaign.SuccessArtifacts) == 1 && len(campaign.Artifacts) == 0 {
			return campaign.SuccessArtifacts[0], nil
		}
		if canary && campaign.Succeeded == 0 && campaign.Failures == 1 && len(campaign.Artifacts) == 1 && len(campaign.SuccessArtifacts) == 0 {
			return campaign.Artifacts[0], nil
		}
	}
	return "", errors.New("stress campaign did not retain exactly one expected application outcome")
}

func stressArtifactOutcome(manifest record.ExecutionRecord, canary bool) error {
	outcome := manifest.Outcome
	if outcome.Termination == "exit" && outcome.ExitCode != nil {
		if !canary && manifest.ArtifactKind == record.ArtifactSuccess && outcome.Domain == "success" && *outcome.ExitCode == 0 {
			return nil
		}
		if canary && manifest.ArtifactKind == record.ArtifactTargetFailure && outcome.Domain == "target" && outcome.Reason == "nonzero_exit" && *outcome.ExitCode == 17 {
			return nil
		}
	}
	return errors.New("retained stress artifact does not contain the expected application outcome")
}

func stressCollect(ctx context.Context, root, name string, incarnation uint64, result wasi.Result, control *runnerbackend.ChoiceRequest, observation *stressObservation) error {
	prefix := fmt.Sprintf("%s-%d", name, incarnation)
	observation.Run.Identities["environment/"+prefix] = result.EnvironmentIdentity
	trace, diagnostic := result.ChoiceTrace.Bytes, result.DiagnosticTrace.Bytes
	tape, err := choice.ProjectReplayPlan(result.ChoiceTrace, control.ExecutionIdentity)
	if result.RuntimeInterrupted {
		trace, diagnostic = result.InterruptedChoiceBytes, result.InterruptedDiagnosticBytes
		tape, err = choice.ProjectInterruptedReplayPlan(trace, control.Limit, control.ExecutionIdentity)
	}
	if err != nil {
		return err
	}
	inc := determinism.Incarnation{Instance: name, Incarnation: incarnation, Diagnostic: diagnostic, Stdout: result.Stdout, Stderr: result.Stderr}
	observation.Run.Incarnations = append(observation.Run.Incarnations, inc)
	storageIdentity := ""
	if result.StorageImage != nil {
		storageIdentity = result.StorageImage.Identity()
	}
	terminal, err := json.Marshal(struct {
		Termination, Message, Environment string
		StorageIdentity                   string
		ExitCode                          *uint32
		VirtualTime                       uint64
	}{result.Termination, result.Message, result.EnvironmentIdentity, storageIdentity, result.ExitCode, result.VirtualTimeElapsedNanos})
	if err != nil {
		return err
	}
	storage, err := json.Marshal(result.StorageSnapshots)
	if err != nil {
		return err
	}
	for key, data := range map[string][]byte{"choice-trace": trace, "choice-tape": tape.Bytes, "diagnostic": diagnostic, "wasi": result.Transcript, "network": result.NetworkTranscript, "volume": result.StorageTranscript, "storage-snapshots": storage, "stdout": result.Stdout, "stderr": result.Stderr, "terminal": terminal} {
		observation.Digests[prefix+"/"+key] = digest(data)
		if err := temporalWrite(ctx, filepath.Join(root, prefix+"-"+key+".bin"), data); err != nil {
			return err
		}
	}
	return nil
}

func stressSummary(observation stressObservation) stressObservation {
	observation.Run.Incarnations = nil
	observation.Run.WorldOrder = nil
	observation.Run.Boundaries = nil
	observation.Run.OrderLog = nil
	return observation
}

func stressInspection(data []byte, observation *stressObservation) error {
	if len(data) == 0 {
		return nil
	}
	observation.Digests["inspection"] = digest(data)
	run, err := observed.FromInspection(observation.Run, data)
	if err != nil {
		return err
	}
	observation.Run = run
	observation.Coordinates = true
	return nil
}
