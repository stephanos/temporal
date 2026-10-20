package measure

import (
	"context"
	"encoding/json"
	errutil "errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	wasi "go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
)

type clusterMeasurementCandidateRun struct {
	Seed                                         uint64
	Repetition                                   int
	Warmup, Passed                               bool
	FirstAuthorizationAt, LastClassifiedReapedAt time.Time
	AuthorizationToClassifiedReapedNanos         uint64
	WholeScenarioWallNanos                       uint64
	ParentBefore, ParentAfter                    *hostexec.ResourceUsage
	ParentUserNanos, ParentSystemNanos           uint64
	Pairs, Processes                             []map[string]any
}

func measureMatchedReleaseCandidate(runCtx context.Context) (measureErr error) {
	base := input(runCtx, "GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT")
	if base == "" {
		measureErr = errutil.Join(measureErr, fmt.Errorf("%w: %v", ErrRefused, errutil.New("set GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT for the matched release candidate")))
		return
	}
	repo, err := absolute(runCtx, "../../../../../..")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	relative, err := filepath.Rel(repo, base)
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || base == string(filepath.Separator) || err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		measureErr = errutil.Join(measureErr, errutil.New("measurement root must be an absolute clean external path"))
		return
	}
	root, err := diagnosticRunRoot(base, "matched-release-candidate-")
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	admission, diagnosticAdmissionErr := newDiagnosticAdmission(runCtx, repo, root)
	if diagnosticAdmissionErr != nil {
		measureErr = errutil.Join(measureErr, diagnosticAdmissionErr)
		return
	}
	clusterMeasurementCandidateAdmitHelperErr := clusterMeasurementCandidateAdmitHelper(runCtx, admission, input(runCtx, "GOMAD_WASM_TEST_HELPER"), input(runCtx, "GOMAD_WASM_CLUSTER_HELPER_ADMISSION"))
	if clusterMeasurementCandidateAdmitHelperErr != nil {
		measureErr = errutil.Join(measureErr, clusterMeasurementCandidateAdmitHelperErr)
		return
	}
	parent, err := os.Executable()
	if err != nil {
		measureErr = errutil.Join(measureErr, err)
		return
	}
	parentSHA256, parentSHA256Err := fileHash(runCtx, parent)
	if parentSHA256Err != nil {
		measureErr = errutil.Join(measureErr, parentSHA256Err)
		return
	}
	admission.Inputs[parent] = parentSHA256
	setInput(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", root)
	setInput(runCtx, "GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE", filepath.Join(root, "compile-cache"))
	recoveryRequestWithAdmission, recoveryRequestWithAdmission2, recoveryRequestWithAdmissionErr := temporalRecoveryRequestWithAdmission(runCtx, "temporal_broker", admission)
	if recoveryRequestWithAdmissionErr != nil {
		measureErr = errutil.Join(measureErr, recoveryRequestWithAdmissionErr)
		return
	}
	request, _ := recoveryRequestWithAdmission, recoveryRequestWithAdmission2
	if request.DiagnosticConfigurationRevision() != "" || request.GuestProfile() != nil || request.Runtime == nil || request.Runtime.Diagnostics {
		measureErr = errutil.Join(measureErr, errutil.New("candidate producer requires the uninstrumented release execution profile"))
		return
	}
	var runs []clusterMeasurementCandidateRun
	choiceBaselines := map[uint64]wasi.Result{}
	choiceWitnesses := map[uint64]string{}
	addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
		checkErr := admission.check(runCtx, root, "final", request, false)
		if checkErr != nil {
			measureErr = errutil.Join(measureErr, checkErr)
			return
		}
		summaries := map[string]any{}
		for _, seed := range []uint64{7, 19, 42} {
			var authorized, wholeWall, parentCPU []uint64
			for _, run := range runs {
				if run.Seed != seed || run.Warmup || !run.Passed {
					continue
				}
				authorized = append(authorized, run.AuthorizationToClassifiedReapedNanos)
				wholeWall = append(wholeWall, run.WholeScenarioWallNanos)
				parentCPU = append(parentCPU, run.ParentUserNanos+run.ParentSystemNanos)
			}
			if len(authorized) != 3 {
				measureErr = errutil.Join(measureErr, errutil.New(fmt.Sprint("three completed warm scenarios required for seed", seed)))
			}
			summaries[stringSeed(seed)] = map[string]any{"authorized": clusterMeasurementCandidateDistribution(authorized), "whole_scenario_wall": clusterMeasurementCandidateDistribution(wholeWall), "parent_cpu": clusterMeasurementCandidateDistribution(parentCPU), "choice_witness_sha256": digest([]byte(choiceWitnesses[seed]))}
		}
		distinct := distinctChoiceTraces(choiceBaselines)
		if len(choiceBaselines) != 3 || distinct < 2 {
			measureErr = errutil.Join(measureErr, errutil.New("three measured seeds must retain distinct actual choice traces"))
		}
		parentSHA2562, parentSHA256Err2 := fileHash(runCtx, parent)
		if parentSHA256Err2 != nil {
			measureErr = errutil.Join(measureErr, parentSHA256Err2)
			return
		}
		writeReportErr := temporalJSON(runCtx, filepath.Join(root, "matched-release-report.json"), map[string]any{"schema": "gomad.matched-release-cluster/v1", "role": "single-helper-rust-candidate", "network_owner": "in-process-rust", "network_owner_implementation_sha256": request.HelperSHA256, "source_commit": admission.SourceHead, "host_admission": admission.Receipt, "helper_admission": input(runCtx, "GOMAD_WASM_CLUSTER_HELPER_ADMISSION"), "helper_features": []string{}, "parent_executable_sha256": parentSHA2562, "identities": variantIdentities(request, wasi.Result{}), "module_receipts": admission.ModuleReceipts, "inputs": admission.Inputs, "seeds": []uint64{7, 19, 42}, "warm_repetitions": 3, "runs": runs, "summary": summaries, "distinct_choice_traces": distinct, "scenario_passed": measureErr == nil && !failed(runCtx) && len(runs) == 12, "primary_wall_scope": "earliest actual cluster helper authorize-send through latest classified and owned-reaped helper completion across both processes and all four phases; includes interphase control/oracle/publication gaps and final World validation", "whole_scenario_scope": "fixture invocation plus cleanup, including initial helper preparation and final oracle/publication work; module/compiler preparation excluded and retained separately", "parent_cpu_scope": "whole-scenario RUSAGE_SELF delta; no compiler/helper child CPU; not authorization-only CPU", "helper_cost_scope": "one sample per actual completed helper process; never summed per-incarnation copies or overlapping wall durations; own-image RSS in metrics distinct from wait RSS high-water", "rerun": rerunRecord(runCtx, input(runCtx, "GOMAD_STOCK_GO"), repo, "matched-release-candidate", map[string]string{"GOENV": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "GOROOT": "", "GOMAD_STOCK_GO": input(runCtx, "GOMAD_STOCK_GO"), "GOMAD_WASM_TEST_HELPER": input(runCtx, "GOMAD_WASM_TEST_HELPER"), "GOMAD_WASM_CLUSTER_HELPER_ADMISSION": input(runCtx, "GOMAD_WASM_CLUSTER_HELPER_ADMISSION"), "GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT": base, "GOMAD_WASM_TOOLCHAIN_CACHE": input(runCtx, "GOMAD_WASM_TOOLCHAIN_CACHE"), "TMPDIR": input(runCtx, "TMPDIR")}, false)})
		if writeReportErr != nil {
			measureErr = errutil.Join(measureErr, writeReportErr)
			return
		}
		return
	})
	for _, seed := range []uint64{7, 19, 42} {
		repetition := 0
		for ; repetition <= 3; repetition++ {
			runStep2, runStepErr := runStep(runCtx, fmt.Sprintf("seed%d/rep%d", seed, repetition), func(runCtx context.Context) (measureErr error) {
				evidence := filepath.Join(root, fmt.Sprintf("seed%d-rep%d", seed, repetition))
				err := os.MkdirAll(evidence, 0700)
				if err != nil {
					measureErr = errutil.Join(measureErr, err)
					return
				}
				local := request
				control := *request.Runtime
				control.Seed, local.Runtime = seed, &control
				local.CompileCache = filepath.Join(root, fmt.Sprintf("seed%d-compile-cache", seed))
				err = os.MkdirAll(local.CompileCache, 0700)
				if err != nil {
					measureErr = errutil.Join(measureErr, err)
					return
				}
				checkErr2 := admission.check(runCtx, evidence, "scenario-before", local, true)
				if checkErr2 != nil {
					measureErr = errutil.Join(measureErr, checkErr2)
					return
				}
				var completed []*wasi.ClusterResult
				row := clusterMeasurementCandidateRun{Seed: seed, Repetition: repetition, Warmup: repetition == 0}
				row.ParentBefore = hostexec.CurrentProcessResourceUsage()
				started := time.Now()
				addCleanup(runCtx, func(runCtx context.Context) (measureErr error) {
					row.WholeScenarioWallNanos, row.ParentAfter = uint64(time.Since(started)), hostexec.CurrentProcessResourceUsage()
					cpuValid := row.ParentBefore != nil && row.ParentAfter != nil && row.ParentAfter.UserNanos >= row.ParentBefore.UserNanos && row.ParentAfter.SystemNanos >= row.ParentBefore.SystemNanos
					if !cpuValid {
						measureErr = errutil.Join(measureErr, errutil.New("whole-scenario parent CPU unavailable or regressed"))
					} else {
						row.ParentUserNanos, row.ParentSystemNanos = row.ParentAfter.UserNanos-row.ParentBefore.UserNanos, row.ParentAfter.SystemNanos-row.ParentBefore.SystemNanos
					}
					if len(completed) != 2 || len(row.Pairs) != 4 {
						measureErr = errutil.Join(measureErr, errutil.New("two completed helper processes and all four scenario phases required"))
					}
					if len(completed) > 0 {
						envelope, clusterMeasurementCandidateEnvelopeErr := clusterMeasurementCandidateEnvelope(runCtx, completed, local.HelperSHA256)
						if clusterMeasurementCandidateEnvelopeErr != nil {
							measureErr = errutil.Join(measureErr, clusterMeasurementCandidateEnvelopeErr)
							return
						}
						row.FirstAuthorizationAt, row.LastClassifiedReapedAt = envelope["first_authorization_at"].(time.Time), envelope["last_classified_reaped_at"].(time.Time)
						row.AuthorizationToClassifiedReapedNanos = uint64(row.LastClassifiedReapedAt.Sub(row.FirstAuthorizationAt))
					}
					row.Passed = measureErr == nil && !failed(runCtx) && cpuValid && len(completed) == 2 && len(row.Pairs) == 4 && row.AuthorizationToClassifiedReapedNanos > 0
					runs = append(runs, row)
					writeReportErr2 := temporalJSON(runCtx, filepath.Join(evidence, "scenario-matched-release.json"), row)
					if writeReportErr2 != nil {
						measureErr = errutil.Join(measureErr, writeReportErr2)
						return
					}
					checkErr3 := admission.check(runCtx, evidence, "scenario-after", local, false)
					if checkErr3 != nil {
						measureErr = errutil.Join(measureErr, checkErr3)
						return
					}
					return
				})
				observer := temporalPairObserver{Admission: func(runCtx context.Context, evidence, label string, requests []wasi.Request, before bool) (measureErr error) {
					boundary := "after"
					if before {
						boundary = "before"
					}
					for i, request := range requests {
						checkErr4 := admission.check(runCtx, evidence, fmt.Sprintf("%s-%d-%s", label, i, boundary), request, before)
						if checkErr4 != nil {
							measureErr = errutil.Join(measureErr, checkErr4)
							return
						}
					}
					return
				}, Observe: func(runCtx context.Context, evidence, label string, requests []wasi.Request, results []wasi.Result, failures []error) (measureErr error) {
					labels := []string{"cluster-crash", "cluster-reopened", "cluster-prefix-replay", "cluster-full-replay"}
					if len(row.Pairs) >= len(labels) || label != labels[len(row.Pairs)] || len(requests) != 2 || len(results) != 2 || len(failures) != 2 {
						measureErr = errutil.Join(measureErr, errutil.New(fmt.Sprint("unexpected or incomplete candidate scenario phase", label)))
						return
					}
					var instances []map[string]any
					for i, result := range results {
						if failures[i] != nil || !result.Reaped || requests[i].Broker == nil {
							measureErr = errutil.Join(measureErr, fmt.Errorf("candidate incarnation result unavailable: %s %v", label, failures[i]))
							return
						}
						instances = append(instances, map[string]any{"node": requests[i].Broker.Node.ID, "incarnation": requests[i].Broker.Node.Incarnation, "termination": result.Termination, "exit_code": result.ExitCode, "incarnation_finished": result.Reaped, "peak_memory_bytes": result.PeakMemoryBytes, "fuel_remaining": result.FuelRemaining, "identities": variantIdentities(requests[i], result), "choice_sha256": digest(result.ChoiceTrace.Bytes), "interrupted_choice_sha256": digest(result.InterruptedChoiceBytes), "stdout_sha256": digest(result.Stdout), "stderr_sha256": digest(result.Stderr), "wasi_sha256": fmt.Sprintf("%x", result.TranscriptSHA256), "storage_sha256": digest(result.StorageTranscript), "network_sha256": digest(result.NetworkTranscript)})
					}
					if repetition == 1 && label == "cluster-reopened" {
						choiceBaselines[seed] = results[0]
					}
					if repetition > 0 && label == "cluster-reopened" {
						var witness string
						for _, result := range results {
							witness += digest(result.ChoiceTrace.Bytes) + ":" + digest(result.InterruptedChoiceBytes) + "\n"
						}
						previous := choiceWitnesses[seed]
						if previous != "" && previous != witness {
							measureErr = errutil.Join(measureErr, errutil.New(fmt.Sprint("same seed changed choice evidence across warm repetitions", seed)))
						}
						choiceWitnesses[seed] = witness
					}
					pair := map[string]any{"label": label, "instances": instances, "network_owner": "in-process-rust", "network_owner_implementation_sha256": requests[0].HelperSHA256}
					row.Pairs = append(row.Pairs, pair)
					writeReportErr3 := temporalJSON(runCtx, filepath.Join(evidence, label+"-matched-release.json"), pair)
					if writeReportErr3 != nil {
						measureErr = errutil.Join(measureErr, writeReportErr3)
						return
					}
					return
				}, Cluster: func(runCtx context.Context, label string, result *wasi.ClusterResult) (measureErr error) {
					labels := []string{"two-cluster", "two-cluster-replayed"}
					if len(completed) >= len(labels) || label != labels[len(completed)] {
						measureErr = errutil.Join(measureErr, errutil.New(fmt.Sprint("unexpected or duplicate completed cluster process", label)))
						return
					}
					process, clusterMeasurementCandidateProcessErr := clusterMeasurementCandidateProcess(runCtx, result, local.HelperSHA256)
					if clusterMeasurementCandidateProcessErr != nil {
						measureErr = errutil.Join(measureErr, clusterMeasurementCandidateProcessErr)
						return
					}
					if repetition > 0 && (!result.Metrics.CacheHit || result.Metrics.CompileNanos != 0) {
						measureErr = errutil.Join(measureErr, errutil.New("warm candidate process compiled or missed its admitted cache"))
						return
					}
					process["label"] = label
					completed = append(completed, result)
					row.Processes = append(row.Processes, process)
					writeReportErr4 := temporalJSON(runCtx, filepath.Join(evidence, label+"-matched-process.json"), process)
					if writeReportErr4 != nil {
						measureErr = errutil.Join(measureErr, writeReportErr4)
						return
					}
					return
				}}
				independentClustersPartitionHealWholeClusterCrashErr := temporalIndependentClustersPartitionHealWholeClusterCrash(runCtx, local, evidence, observer)
				if independentClustersPartitionHealWholeClusterCrashErr != nil {
					measureErr = errutil.Join(measureErr, independentClustersPartitionHealWholeClusterCrashErr)
					return
				}
				return
			})
			if runStepErr != nil {
				measureErr = errutil.Join(measureErr, runStepErr)
				return
			}
			_ = runStep2
		}
	}
	return
}
func clusterMeasurementCandidateAdmitHelper(runCtx context.Context, admission *diagnosticAdmission, helper, receipt string) (measureErr error) {
	if !filepath.IsAbs(helper) || !filepath.IsAbs(receipt) {
		measureErr = errutil.New("explicit absolute release helper and admission receipt required")
		return
	}
	helperSHA256, helperSHA256Err := fileHash(runCtx, helper)
	if helperSHA256Err != nil {
		measureErr = helperSHA256Err
		return
	}
	inputs, terminalExitHelperInputsErr := terminalExitHelperInputs(runCtx, receipt, wasi.Request{HelperPath: helper, HelperSHA256: helperSHA256}, filepath.Join(admission.Repo, "tools/gomad/internal/backends/wasm/wasmhost"))
	if terminalExitHelperInputsErr != nil {
		measureErr = terminalExitHelperInputsErr
		return
	}
	var admitted terminalExitHelperAdmission
	mustTerminalExitRead2, mustTerminalExitReadErr := mustTerminalExitRead(runCtx, receipt)
	if mustTerminalExitReadErr != nil {
		measureErr = mustTerminalExitReadErr
		return
	}
	err := json.Unmarshal(mustTerminalExitRead2, &admitted)
	if err != nil {
		measureErr = err
		return
	}
	var build terminalExitHelperBuild
	mustTerminalExitRead3, mustTerminalExitReadErr2 := mustTerminalExitRead(runCtx, admitted.BuildReceipt)
	if mustTerminalExitReadErr2 != nil {
		measureErr = mustTerminalExitReadErr2
		return
	}
	err = json.Unmarshal(mustTerminalExitRead3, &build)
	if err != nil {
		measureErr = err
		return
	}
	if len(build.Command) == 0 || !filepath.IsAbs(build.Command[0]) || build.Inputs[build.Command[0]] == "" || !slices.Equal(build.Command, terminalExitCargoCommand(build.Command[0], build.Crate, []string{})) {
		measureErr = errutil.New("release measurement requires the exact pinned single-job Cargo command")
		return
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(filepath.Dir(admitted.BuildReceipt), name)
		pathSHA256, pathSHA256Err := fileHash(runCtx, path)
		if pathSHA256Err != nil {
			measureErr = pathSHA256Err
			return
		}
		if build.Inputs[path] != pathSHA256 {
			measureErr = errutil.New(fmt.Sprint("missing authenticated raw release compiler evidence", name))
			return
		}
	}
	var execution struct {
		Command  []string
		ExitCode int
	}
	mustTerminalExitRead4, mustTerminalExitReadErr3 := mustTerminalExitRead(runCtx, filepath.Join(filepath.Dir(admitted.BuildReceipt), "cargo-build.json"))
	if mustTerminalExitReadErr3 != nil {
		measureErr = mustTerminalExitReadErr3
		return
	}
	err = json.Unmarshal(mustTerminalExitRead4, &execution)
	if err != nil || execution.ExitCode != 0 || !slices.Equal(execution.Command, build.Command) {
		measureErr = errutil.New(fmt.Sprint("release command differs from retained execution", err))
		return
	}
	mustTerminalExitRead5, mustTerminalExitReadErr4 := mustTerminalExitRead(runCtx, filepath.Join(filepath.Dir(admitted.BuildReceipt), "rustc-version.stdout"))
	if mustTerminalExitReadErr4 != nil {
		measureErr = mustTerminalExitReadErr4
		return
	}
	mustTerminalExitRead6, mustTerminalExitReadErr5 := mustTerminalExitRead(runCtx, filepath.Join(filepath.Dir(admitted.BuildReceipt), "cargo-version.stdout"))
	if mustTerminalExitReadErr5 != nil {
		measureErr = mustTerminalExitReadErr5
		return
	}
	if string(mustTerminalExitRead5) != build.RustcVersion || string(mustTerminalExitRead6) != build.CargoVersion {
		measureErr = errutil.New("release compiler versions differ from retained execution")
		return
	}
	admission.Helpers[helper] = diagnosticHelperAdmission{Receipt: receipt, Features: []string{}}
	maps.Copy(admission.Inputs, inputs)
	return
}
func clusterMeasurementCandidateDistribution(values []uint64) map[string]any {
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	if len(ordered) == 0 {
		return map[string]any{"samples": values, "available": false}
	}
	return map[string]any{"samples": values, "available": true, "min_nanos": ordered[0], "median_nanos": ordered[len(ordered)/2], "max_nanos": ordered[len(ordered)-1]}
}
func clusterMeasurementCandidateProcess(runCtx context.Context, result *wasi.ClusterResult, helperSHA256 string) (measureValue0 map[string]any, measureErr error) {
	if result == nil || result.Metrics == nil || result.Scheduler == nil || result.HelperResourceUsage == nil || result.AuthorizationStartedAt.IsZero() || !result.TerminalObservedAt.After(result.AuthorizationStartedAt) || result.AuthorizationToTerminalNanos == 0 || result.WallElapsedNanos == 0 {
		measureErr = errutil.New("candidate measurement requires an actual completed and owned-reaped cluster process")
		return
	}
	if result.Frames.GuestCallbacks != 0 || !result.Metrics.ResourceUsageAvailable || result.Metrics.PeakRSSBytes == 0 || result.Scheduler.OperationSamples == 0 || result.Scheduler.RunnableSamples > result.Scheduler.OperationSamples || result.Scheduler.OverlapNanos > result.Scheduler.ActiveNanos {
		measureErr = errutil.New("candidate scheduler, ownership or resource observation invalid")
		return
	}
	return map[string]any{"network_owner": "in-process-rust", "network_owner_implementation_sha256": helperSHA256, "authorization_started_at": result.AuthorizationStartedAt, "classified_reaped_at": result.TerminalObservedAt, "authorization_to_classified_reaped_nanos": result.TerminalObservedAt.Sub(result.AuthorizationStartedAt).Nanoseconds(), "cluster_authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "cluster_wall_elapsed_nanos": result.WallElapsedNanos, "metrics": result.Metrics, "helper_process_usage": result.HelperResourceUsage, "frames": result.Frames, "scheduler": result.Scheduler, "scope": "one completed helper process; cluster authorization and elapsed time include control/oracle pauses and preparation according to their own boundaries; never sum per-incarnation copies of these metrics", "guest_scope": "execute wall minus timed host-import spans, an upper bound including ABI and unattributed runtime cost; not guest-only CPU", "handoff_scope": "handoff_nanos measures grant-to-wake latency; divide only by matching handoff_samples, not guest idle handoff count; fibers decision uses idle handoff count times current host switch cost"}, measureErr
}
func clusterMeasurementCandidateEnvelope(runCtx context.Context, results []*wasi.ClusterResult, helperSHA256 string) (measureValue0 map[string]any, measureErr error) {
	if len(results) == 0 {
		measureErr = errutil.New("actual completed helper processes required")
		return
	}
	var first, last time.Time
	var processes []map[string]any
	for _, result := range results {
		clusterMeasurementCandidateProcess2, clusterMeasurementCandidateProcessErr := clusterMeasurementCandidateProcess(runCtx, result, helperSHA256)
		if clusterMeasurementCandidateProcessErr != nil {
			measureErr = clusterMeasurementCandidateProcessErr
			return
		}
		processes = append(processes, clusterMeasurementCandidateProcess2)
		if first.IsZero() || result.AuthorizationStartedAt.Before(first) {
			first = result.AuthorizationStartedAt
		}
		if result.TerminalObservedAt.After(last) {
			last = result.TerminalObservedAt
		}
	}
	return map[string]any{"first_authorization_at": first, "last_classified_reaped_at": last, "authorization_to_classified_reaped_nanos": last.Sub(first).Nanoseconds(), "processes": processes}, measureErr
}
