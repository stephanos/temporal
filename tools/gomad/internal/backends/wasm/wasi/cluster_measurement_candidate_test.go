//go:build gomad_cluster_candidate

package wasi

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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

func TestMatchedReleaseClusterCandidate(t *testing.T) {
	base := os.Getenv("GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT")
	if base == "" {
		t.Skip("set GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT for the matched release candidate")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, base)
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || base == string(filepath.Separator) || err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("measurement root must be an absolute clean external path")
	}
	root, err := diagnosticRunRoot(base, "matched-release-candidate-")
	if err != nil {
		t.Fatal(err)
	}
	admission := newDiagnosticAdmission(t, repo, root)
	clusterMeasurementCandidateAdmitHelper(t, admission, os.Getenv("GOMAD_WASM_TEST_HELPER"), os.Getenv("GOMAD_WASM_CLUSTER_HELPER_ADMISSION"))
	parent, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	admission.Inputs[parent] = fileHash(t, parent)
	t.Setenv("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", root)
	t.Setenv("GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE", filepath.Join(root, "compile-cache"))
	request, _ := temporalRecoveryRequestWithAdmission(t, "temporal_broker", admission)
	if request.diagnosticConfigurationRevision != "" || request.guestProfile != nil || request.Runtime == nil || request.Runtime.Diagnostics {
		t.Fatal("candidate producer requires the uninstrumented release execution profile")
	}
	var runs []clusterMeasurementCandidateRun
	choiceBaselines := map[uint64]Result{}
	choiceWitnesses := map[uint64]string{}
	t.Cleanup(func() {
		admission.check(t, root, "final", request, false)
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
				t.Error("three completed warm scenarios required for seed", seed)
			}
			summaries[stringSeed(seed)] = map[string]any{"authorized": clusterMeasurementCandidateDistribution(authorized), "whole_scenario_wall": clusterMeasurementCandidateDistribution(wholeWall), "parent_cpu": clusterMeasurementCandidateDistribution(parentCPU), "choice_witness_sha256": digest([]byte(choiceWitnesses[seed]))}
		}
		distinct := distinctChoiceTraces(choiceBaselines)
		if len(choiceBaselines) != 3 || distinct < 2 {
			t.Error("three measured seeds must retain distinct actual choice traces")
		}
		temporalJSON(t, filepath.Join(root, "matched-release-report.json"), map[string]any{
			"schema": "gomad3.matched-release-cluster/v1", "role": "single-helper-rust-candidate", "network_owner": "in-process-rust", "network_owner_implementation_sha256": ImplementationSHA256(),
			"source_commit": admission.SourceHead, "host_admission": admission.Receipt, "helper_admission": os.Getenv("GOMAD_WASM_CLUSTER_HELPER_ADMISSION"), "helper_features": []string{},
			"parent_executable_sha256": fileHash(t, parent), "identities": variantIdentities(request, Result{}), "module_receipts": admission.ModuleReceipts, "inputs": admission.Inputs,
			"seeds": []uint64{7, 19, 42}, "warm_repetitions": 3, "runs": runs, "summary": summaries, "distinct_choice_traces": distinct, "scenario_passed": !t.Failed() && len(runs) == 12,
			"primary_wall_scope":   "earliest actual cluster helper authorize-send through latest classified and owned-reaped helper completion across both processes and all four phases; includes interphase control/oracle/publication gaps and final World validation",
			"whole_scenario_scope": "fixture invocation plus cleanup, including initial helper preparation and final oracle/publication work; module/compiler preparation excluded and retained separately",
			"parent_cpu_scope":     "whole-scenario RUSAGE_SELF delta; no compiler/helper child CPU; not authorization-only CPU",
			"helper_cost_scope":    "one sample per actual completed helper process; never summed per-incarnation copies or overlapping wall durations; own-image RSS in metrics distinct from wait RSS high-water",
			"rerun":                map[string]any{"working_directory": filepath.Join(repo, "tools/gomad"), "command": []string{os.Getenv("GOMAD3_STOCK_GO"), "test", "-tags", "test_dep,gomad_cluster_candidate", "-count=1", "-timeout=6h", "-run", "^TestMatchedReleaseClusterCandidate$", "./internal/backends/wasm/wasi"}, "environment": map[string]string{"GOENV": "off", "GOTOOLCHAIN": "local", "GOWORK": "off", "GOROOT": "", "GOMAD3_STOCK_GO": os.Getenv("GOMAD3_STOCK_GO"), "GOMAD_WASM_TEST_HELPER": os.Getenv("GOMAD_WASM_TEST_HELPER"), "GOMAD_WASM_CLUSTER_HELPER_ADMISSION": os.Getenv("GOMAD_WASM_CLUSTER_HELPER_ADMISSION"), "GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT": base, "GOMAD_WASM_TOOLCHAIN_CACHE": os.Getenv("GOMAD_WASM_TOOLCHAIN_CACHE"), "TMPDIR": os.Getenv("TMPDIR")}},
		})
	})
	for _, seed := range []uint64{7, 19, 42} {
		for repetition := 0; repetition <= 3; repetition++ {
			t.Run(fmt.Sprintf("seed%d/rep%d", seed, repetition), func(t *testing.T) {
				evidence := filepath.Join(root, fmt.Sprintf("seed%d-rep%d", seed, repetition))
				if err := os.MkdirAll(evidence, 0700); err != nil {
					t.Fatal(err)
				}
				local := request
				control := *request.Runtime
				control.Seed, local.Runtime = seed, &control
				local.CompileCache = filepath.Join(root, fmt.Sprintf("seed%d-compile-cache", seed))
				if err := os.MkdirAll(local.CompileCache, 0700); err != nil {
					t.Fatal(err)
				}
				admission.check(t, evidence, "scenario-before", local, true)
				var completed []*ClusterResult
				row := clusterMeasurementCandidateRun{Seed: seed, Repetition: repetition, Warmup: repetition == 0}
				row.ParentBefore = hostexec.CurrentProcessResourceUsage()
				started := time.Now()
				t.Cleanup(func() {
					row.WholeScenarioWallNanos, row.ParentAfter = uint64(time.Since(started)), hostexec.CurrentProcessResourceUsage()
					cpuValid := row.ParentBefore != nil && row.ParentAfter != nil && row.ParentAfter.UserNanos >= row.ParentBefore.UserNanos && row.ParentAfter.SystemNanos >= row.ParentBefore.SystemNanos
					if !cpuValid {
						t.Error("whole-scenario parent CPU unavailable or regressed")
					} else {
						row.ParentUserNanos, row.ParentSystemNanos = row.ParentAfter.UserNanos-row.ParentBefore.UserNanos, row.ParentAfter.SystemNanos-row.ParentBefore.SystemNanos
					}
					if len(completed) != 2 || len(row.Pairs) != 4 {
						t.Error("two completed helper processes and all four scenario phases required")
					}
					if len(completed) > 0 {
						envelope := clusterMeasurementCandidateEnvelope(t, completed)
						row.FirstAuthorizationAt, row.LastClassifiedReapedAt = envelope["first_authorization_at"].(time.Time), envelope["last_classified_reaped_at"].(time.Time)
						row.AuthorizationToClassifiedReapedNanos = uint64(row.LastClassifiedReapedAt.Sub(row.FirstAuthorizationAt))
					}
					row.Passed = !t.Failed() && cpuValid && len(completed) == 2 && len(row.Pairs) == 4 && row.AuthorizationToClassifiedReapedNanos > 0
					runs = append(runs, row)
					temporalJSON(t, filepath.Join(evidence, "scenario-matched-release.json"), row)
					admission.check(t, evidence, "scenario-after", local, false)
				})
				observer := temporalPairObserver{
					Admission: func(t *testing.T, evidence, label string, requests []Request, before bool) {
						boundary := "after"
						if before {
							boundary = "before"
						}
						for i, request := range requests {
							admission.check(t, evidence, fmt.Sprintf("%s-%d-%s", label, i, boundary), request, before)
						}
					},
					Observe: func(t *testing.T, evidence, label string, requests []Request, results []Result, failures []error) {
						labels := []string{"cluster-crash", "cluster-reopened", "cluster-prefix-replay", "cluster-full-replay"}
						if len(row.Pairs) >= len(labels) || label != labels[len(row.Pairs)] || len(requests) != 2 || len(results) != 2 || len(failures) != 2 {
							t.Fatal("unexpected or incomplete candidate scenario phase", label)
						}
						var instances []map[string]any
						for i, result := range results {
							if failures[i] != nil || !result.Reaped || requests[i].Broker == nil {
								t.Fatalf("candidate incarnation result unavailable: %s %v", label, failures[i])
							}
							instances = append(instances, map[string]any{"node": requests[i].Broker.Node.ID, "incarnation": requests[i].Broker.Node.Incarnation, "termination": result.Termination, "exit_code": result.ExitCode, "incarnation_finished": result.Reaped,
								"peak_memory_bytes": result.PeakMemoryBytes, "fuel_remaining": result.FuelRemaining, "identities": variantIdentities(requests[i], result), "choice_sha256": digest(result.ChoiceTrace.Bytes), "interrupted_choice_sha256": digest(result.InterruptedChoiceBytes),
								"stdout_sha256": digest(result.Stdout), "stderr_sha256": digest(result.Stderr), "wasi_sha256": fmt.Sprintf("%x", result.TranscriptSHA256), "storage_sha256": digest(result.StorageTranscript), "network_sha256": digest(result.NetworkTranscript)})
						}
						if repetition == 1 && label == "cluster-reopened" {
							choiceBaselines[seed] = results[0]
						}
						if repetition > 0 && label == "cluster-reopened" {
							var witness string
							for _, result := range results {
								witness += digest(result.ChoiceTrace.Bytes) + ":" + digest(result.InterruptedChoiceBytes) + "\n"
							}
							if previous := choiceWitnesses[seed]; previous != "" && previous != witness {
								t.Error("same seed changed choice evidence across warm repetitions", seed)
							}
							choiceWitnesses[seed] = witness
						}
						pair := map[string]any{"label": label, "instances": instances, "network_owner": "in-process-rust", "network_owner_implementation_sha256": ImplementationSHA256()}
						row.Pairs = append(row.Pairs, pair)
						temporalJSON(t, filepath.Join(evidence, label+"-matched-release.json"), pair)
					},
					Cluster: func(t *testing.T, label string, result *ClusterResult) {
						labels := []string{"two-cluster", "two-cluster-replayed"}
						if len(completed) >= len(labels) || label != labels[len(completed)] {
							t.Fatal("unexpected or duplicate completed cluster process", label)
						}
						process := clusterMeasurementCandidateProcess(t, result)
						if repetition > 0 && (!result.Metrics.CacheHit || result.Metrics.CompileNanos != 0) {
							t.Fatal("warm candidate process compiled or missed its admitted cache")
						}
						process["label"] = label
						completed = append(completed, result)
						row.Processes = append(row.Processes, process)
						temporalJSON(t, filepath.Join(evidence, label+"-matched-process.json"), process)
					},
				}
				temporalIndependentClustersPartitionHealWholeClusterCrash(t, local, evidence, observer)
			})
		}
	}
}

func clusterMeasurementCandidateAdmitHelper(t *testing.T, admission *diagnosticAdmission, helper, receipt string) {
	t.Helper()
	if !filepath.IsAbs(helper) || !filepath.IsAbs(receipt) {
		t.Fatal("explicit absolute release helper and admission receipt required")
	}
	inputs := terminalExitHelperInputs(t, receipt, Request{HelperPath: helper, HelperSHA256: fileHash(t, helper)}, filepath.Join(admission.Repo, "tools/gomad/internal/backends/wasm/wasmhost"))
	var admitted terminalExitHelperAdmission
	if err := json.Unmarshal(mustTerminalExitRead(t, receipt), &admitted); err != nil {
		t.Fatal(err)
	}
	var build terminalExitHelperBuild
	if err := json.Unmarshal(mustTerminalExitRead(t, admitted.BuildReceipt), &build); err != nil {
		t.Fatal(err)
	}
	if len(build.Command) == 0 || !filepath.IsAbs(build.Command[0]) || build.Inputs[build.Command[0]] == "" || !slices.Equal(build.Command, terminalExitCargoCommand(build.Command[0], build.Crate, []string{})) {
		t.Fatal("release measurement requires the exact pinned single-job Cargo command")
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(filepath.Dir(admitted.BuildReceipt), name)
		if build.Inputs[path] != fileHash(t, path) {
			t.Fatal("missing authenticated raw release compiler evidence", name)
		}
	}
	var execution struct {
		Command  []string
		ExitCode int
	}
	if err := json.Unmarshal(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admitted.BuildReceipt), "cargo-build.json")), &execution); err != nil || execution.ExitCode != 0 || !slices.Equal(execution.Command, build.Command) {
		t.Fatal("release command differs from retained execution", err)
	}
	if string(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admitted.BuildReceipt), "rustc-version.stdout"))) != build.RustcVersion || string(mustTerminalExitRead(t, filepath.Join(filepath.Dir(admitted.BuildReceipt), "cargo-version.stdout"))) != build.CargoVersion {
		t.Fatal("release compiler versions differ from retained execution")
	}
	admission.Helpers[helper] = diagnosticHelperAdmission{Receipt: receipt, Features: []string{}}
	maps.Copy(admission.Inputs, inputs)
}

func clusterMeasurementCandidateDistribution(values []uint64) map[string]any {
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	if len(ordered) == 0 {
		return map[string]any{"samples": values, "available": false}
	}
	return map[string]any{"samples": values, "available": true, "min_nanos": ordered[0], "median_nanos": ordered[len(ordered)/2], "max_nanos": ordered[len(ordered)-1]}
}

func clusterMeasurementCandidateProcess(t *testing.T, result *ClusterResult) map[string]any {
	t.Helper()
	if result == nil || result.Metrics == nil || result.Scheduler == nil || result.HelperResourceUsage == nil || result.AuthorizationStartedAt.IsZero() || !result.TerminalObservedAt.After(result.AuthorizationStartedAt) || result.AuthorizationToTerminalNanos == 0 || result.WallElapsedNanos == 0 {
		t.Fatal("candidate measurement requires an actual completed and owned-reaped cluster process")
	}
	if result.Frames.GuestCallbacks != 0 || !result.Metrics.ResourceUsageAvailable || result.Metrics.PeakRSSBytes == 0 || result.Scheduler.OperationSamples == 0 || result.Scheduler.RunnableSamples > result.Scheduler.OperationSamples || result.Scheduler.OverlapNanos > result.Scheduler.ActiveNanos {
		t.Fatal("candidate scheduler, ownership or resource observation invalid")
	}
	return map[string]any{
		"network_owner": "in-process-rust", "network_owner_implementation_sha256": ImplementationSHA256(),
		"authorization_started_at": result.AuthorizationStartedAt, "classified_reaped_at": result.TerminalObservedAt, "authorization_to_classified_reaped_nanos": result.TerminalObservedAt.Sub(result.AuthorizationStartedAt).Nanoseconds(),
		"cluster_authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "cluster_wall_elapsed_nanos": result.WallElapsedNanos,
		"metrics": result.Metrics, "helper_process_usage": result.HelperResourceUsage, "frames": result.Frames, "scheduler": result.Scheduler,
		"scope":         "one completed helper process; cluster authorization and elapsed time include control/oracle pauses and preparation according to their own boundaries; never sum per-incarnation copies of these metrics",
		"guest_scope":   "execute wall minus timed host-import spans, an upper bound including ABI and unattributed runtime cost; not guest-only CPU",
		"handoff_scope": "handoff_nanos measures grant-to-wake latency; divide only by matching handoff_samples, not guest idle handoff count; fibers decision uses idle handoff count times current host switch cost",
	}
}

func clusterMeasurementCandidateEnvelope(t *testing.T, results []*ClusterResult) map[string]any {
	t.Helper()
	if len(results) == 0 {
		t.Fatal("actual completed helper processes required")
	}
	var first, last time.Time
	var processes []map[string]any
	for _, result := range results {
		processes = append(processes, clusterMeasurementCandidateProcess(t, result))
		if first.IsZero() || result.AuthorizationStartedAt.Before(first) {
			first = result.AuthorizationStartedAt
		}
		if result.TerminalObservedAt.After(last) {
			last = result.TerminalObservedAt
		}
	}
	return map[string]any{"first_authorization_at": first, "last_classified_reaped_at": last, "authorization_to_classified_reaped_nanos": last.Sub(first).Nanoseconds(), "processes": processes}
}
