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

const clusterMeasurementBase = "15b491281cdafbfcd7e8b6c8ea4447ac8fbc6208"

var clusterMeasurementLabels = []string{"cluster-crash", "cluster-reopened", "cluster-prefix-replay", "cluster-full-replay"}

type clusterMeasurementInstance struct {
	Node, Termination, Error                        string
	AuthorizationOffsetNanos, ClassifiedOffsetNanos int64
	AuthorizationStartedAt, ExecutionClassifiedAt   time.Time
	AuthorizationToTerminalNanos                    uint64
	ValidatedTerminalObserved                       bool
	Metrics                                         *Telemetry
	HelperProcessUsage                              *hostexec.ResourceUsage
	PeakMemoryBytes                                 uint64
	ChoiceSHA256, InterruptedChoiceSHA256           string
	StdoutSHA256, StderrSHA256, WASISHA256          string
	StorageSHA256, NetworkSHA256                    string
}

type clusterMeasurementPair struct {
	Label                              string
	AuthorizationToClassifiedNanos     uint64
	DispatchWallNanos                  uint64
	ParentBefore, ParentAfter          *hostexec.ResourceUsage
	ParentUserNanos, ParentSystemNanos uint64
	Instances                          []clusterMeasurementInstance
}

func clusterMeasurementEnvelope(starts, ends []time.Time) (time.Time, time.Time, error) {
	if len(starts) != 2 || len(ends) != len(starts) {
		return time.Time{}, time.Time{}, fmt.Errorf("two actual instance boundaries required")
	}
	var first, last time.Time
	for i, start := range starts {
		if start.IsZero() || ends[i].IsZero() || !ends[i].After(start) {
			return time.Time{}, time.Time{}, fmt.Errorf("instance %d lacks a positive authorization/classified-reaped span", i)
		}
		if first.IsZero() || start.Before(first) {
			first = start
		}
		if ends[i].After(last) {
			last = ends[i]
		}
	}
	return first, last, nil
}

func TestClusterMeasurementEnvelope(t *testing.T) {
	start := time.Now()
	starts := []time.Time{start, start.Add(2 * time.Second)}
	ends := []time.Time{start.Add(5 * time.Second), start.Add(7 * time.Second)}
	first, last, err := clusterMeasurementEnvelope(starts, ends)
	if err != nil || last.Sub(first) != 7*time.Second {
		t.Fatalf("overlapping spans must yield 7s, not the summed 10s: %v %v", last.Sub(first), err)
	}
	for _, test := range []struct {
		name         string
		starts, ends []time.Time
	}{
		{"missing-instance", starts[:1], ends[:1]},
		{"missing-start", []time.Time{{}, starts[1]}, ends},
		{"missing-classification", starts, []time.Time{ends[0], {}}},
		{"nonpositive", starts, []time.Time{start, ends[1]}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := clusterMeasurementEnvelope(test.starts, test.ends); err == nil {
				t.Fatal("incomplete observation accepted")
			}
		})
	}
}

func clusterMeasurementReleaseHelper(t *testing.T, admission *diagnosticAdmission, helper, receipt string) {
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
	var execution struct {
		Command  []string
		ExitCode int
	}
	for _, name := range []string{"cargo-build.stdout", "cargo-build.json", "rustc-version.stdout", "cargo-version.stdout"} {
		path := filepath.Join(filepath.Dir(admitted.BuildReceipt), name)
		if build.Inputs[path] != fileHash(t, path) {
			t.Fatal("missing authenticated raw release compiler evidence", name)
		}
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

func TestMatchedReleaseClusterBaseline(t *testing.T) {
	base := os.Getenv("GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT")
	if base == "" {
		t.Skip("set GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT for the matched release cluster baseline")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, base)
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || base == string(filepath.Separator) || err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("measurement root must be an absolute clean external path")
	}
	root, err := diagnosticRunRoot(base, "matched-release-baseline-")
	if err != nil {
		t.Fatal(err)
	}
	admission := newDiagnosticAdmission(t, repo, root)
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	clusterMeasurementReleaseHelper(t, admission, helper, os.Getenv("GOMAD_WASM_CLUSTER_HELPER_ADMISSION"))
	if _, err := diagnosticGit(repo, "merge-base", "--is-ancestor", clusterMeasurementBase, "HEAD"); err != nil {
		t.Fatal("baseline instrumentation must descend from retained fn172 baseline", err)
	}
	patch, err := diagnosticGit(repo, "diff", clusterMeasurementBase, "HEAD", "--", "tools/gomad")
	if err != nil {
		t.Fatal(err)
	}
	temporalWrite(t, filepath.Join(root, "instrumentation.patch"), []byte(patch))
	parent, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	admission.Inputs[parent] = fileHash(t, parent)
	t.Setenv("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", root)
	t.Setenv("GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE", filepath.Join(root, "compile-cache"))
	request, _ := temporalRecoveryRequestWithAdmission(t, "temporal_broker", admission)
	var runs []map[string]any
	choiceBaselines := map[uint64]Result{}
	t.Cleanup(func() {
		admission.check(t, root, "final", request, false)
		summary := clusterMeasurementSummary(t, runs)
		distinct := distinctChoiceTraces(choiceBaselines)
		if len(choiceBaselines) != 3 || distinct < 2 {
			t.Error("three measured seeds must retain distinct actual choice traces")
		}
		temporalJSON(t, filepath.Join(root, "matched-release-report.json"), map[string]any{
			"schema": "gomad3.matched-release-cluster/v1", "role": "fn172-go-broker-baseline", "network_owner": "go-reference", "network_owner_implementation_sha256": ImplementationSHA256(),
			"retained_base_commit": clusterMeasurementBase, "source_commit": admission.SourceHead, "instrumentation_patch_sha256": digest([]byte(patch)), "host_admission": admission.Receipt,
			"helper_admission": os.Getenv("GOMAD_WASM_CLUSTER_HELPER_ADMISSION"), "helper_features": []string{}, "parent_executable_sha256": fileHash(t, parent),
			"identities": variantIdentities(request, Result{}), "module_receipts": admission.ModuleReceipts, "inputs": admission.Inputs,
			"seeds": []uint64{7, 19, 42}, "warm_repetitions": 3, "runs": runs, "summary": summary, "distinct_choice_traces": distinct, "scenario_passed": !t.Failed() && len(runs) == 12,
			"primary_wall_scope": "per complete four-pair scenario: earliest actual instance authorization through latest final classified and owned-reaped result; includes every interphase preparation, control, oracle and publication gap; overlapping helpers never summed",
			"pair_wall_scope":    "per pair: earliest actual instance authorization through latest final classified and owned-reaped result; primary is not the sum of these pairs",
			"parent_cpu_scope":   "RUSAGE_SELF delta around pair dispatch, including preparation, validation, helper reaping and fixture publication; no compiler/helper child CPU; not authorization-only CPU",
			"helper_cost_scope":  "per-helper process metrics and wait usage; CPU distinct from parent; overlapping process walls not cluster elapsed; wait RSS may inherit compiler high-water",
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
				control.Seed = seed
				local.Runtime = &control
				local.CompileCache = filepath.Join(root, fmt.Sprintf("seed%d-compile-cache", seed))
				if err := os.MkdirAll(local.CompileCache, 0700); err != nil {
					t.Fatal(err)
				}
				var pairs []clusterMeasurementPair
				var before *hostexec.ResourceUsage
				var started time.Time
				scenarioStarted, scenarioBefore := time.Now(), hostexec.CurrentProcessResourceUsage()
				t.Cleanup(func() {
					var first, last time.Time
					for _, pair := range pairs {
						for _, instance := range pair.Instances {
							if first.IsZero() || instance.AuthorizationStartedAt.Before(first) {
								first = instance.AuthorizationStartedAt
							}
							if instance.ExecutionClassifiedAt.After(last) {
								last = instance.ExecutionClassifiedAt
							}
						}
					}
					runs = append(runs, map[string]any{"seed": seed, "repetition": repetition, "cold_or_warmup": repetition == 0, "pairs": pairs,
						"first_authorization_at": first, "last_classified_reaped_at": last, "authorization_to_classified_reaped_nanos": last.Sub(first).Nanoseconds(),
						"whole_scenario_wall_nanos": time.Since(scenarioStarted).Nanoseconds(), "parent_before": scenarioBefore, "parent_after": hostexec.CurrentProcessResourceUsage(),
						"passed": !t.Failed() && len(pairs) == len(clusterMeasurementLabels)})
				})
				observer := temporalPairObserver{
					Admission: func(t *testing.T, evidence, label string, requests []Request, begin bool) {
						if !begin {
							return
						}
						for i, request := range requests {
							admission.check(t, evidence, fmt.Sprintf("%s-%d-before", label, i), request, true)
						}
						before, started = hostexec.CurrentProcessResourceUsage(), time.Now()
					},
					Observe: func(t *testing.T, evidence, label string, requests []Request, results []Result, failures []error) {
						after, wall := hostexec.CurrentProcessResourceUsage(), time.Since(started)
						pair := clusterMeasurementCollectPair(t, label, requests, results, failures)
						if repetition == 1 && label == "cluster-reopened" {
							choiceBaselines[seed] = results[0]
						}
						if repetition > 0 {
							for _, instance := range pair.Instances {
								if instance.Metrics != nil && (!instance.Metrics.CacheHit || instance.Metrics.CompileNanos != 0) {
									t.Fatal("warm repetition compiled or missed the admitted cache")
								}
							}
						}
						if before == nil || after == nil || after.UserNanos < before.UserNanos || after.SystemNanos < before.SystemNanos {
							t.Fatal("parent RUSAGE_SELF observation unavailable or regressed")
						}
						pair.DispatchWallNanos, pair.ParentBefore, pair.ParentAfter = uint64(wall), before, after
						pair.ParentUserNanos, pair.ParentSystemNanos = after.UserNanos-before.UserNanos, after.SystemNanos-before.SystemNanos
						if len(pairs) >= len(clusterMeasurementLabels) || label != clusterMeasurementLabels[len(pairs)] {
							t.Fatal("unexpected or duplicate measurement phase", label)
						}
						pairs = append(pairs, pair)
						temporalJSON(t, filepath.Join(evidence, label+"-matched-release.json"), pair)
						for i, request := range requests {
							admission.check(t, evidence, fmt.Sprintf("%s-%d-after", label, i), request, false)
						}
					},
				}
				temporalIndependentClustersPartitionHealWholeClusterCrash(t, local, evidence, observer)
			})
		}
	}
}

func clusterMeasurementCollectPair(t *testing.T, label string, requests []Request, results []Result, failures []error) clusterMeasurementPair {
	t.Helper()
	if len(requests) != 2 || len(results) != 2 || len(failures) != 2 {
		t.Fatal("matched release measurement requires two actual instance results")
	}
	starts, ends := make([]time.Time, 2), make([]time.Time, 2)
	for i, result := range results {
		starts[i], ends[i] = result.authorizationStartedAt, result.executionClassifiedAt
	}
	first, last, err := clusterMeasurementEnvelope(starts, ends)
	if err != nil {
		t.Fatal(err)
	}
	pair := clusterMeasurementPair{Label: label, AuthorizationToClassifiedNanos: uint64(last.Sub(first))}
	for i, result := range results {
		if failures[i] != nil || !result.Reaped || result.HelperResourceUsage == nil {
			t.Fatalf("incomplete release instance %d: termination=%s err=%v", i, result.Termination, failures[i])
		}
		pair.Instances = append(pair.Instances, clusterMeasurementInstance{
			Node: requests[i].Broker.Node.ID, Termination: result.Termination, AuthorizationOffsetNanos: starts[i].Sub(first).Nanoseconds(), ClassifiedOffsetNanos: ends[i].Sub(first).Nanoseconds(),
			AuthorizationStartedAt: starts[i], ExecutionClassifiedAt: ends[i],
			AuthorizationToTerminalNanos: result.AuthorizationToTerminalNanos, Metrics: result.Telemetry, HelperProcessUsage: result.HelperResourceUsage, PeakMemoryBytes: result.PeakMemoryBytes,
			ValidatedTerminalObserved: !result.terminalValidatedAt.IsZero(),
			ChoiceSHA256:              digest(result.ChoiceTrace.Bytes), InterruptedChoiceSHA256: digest(result.InterruptedChoiceBytes), StdoutSHA256: digest(result.Stdout), StderrSHA256: digest(result.Stderr), WASISHA256: fmt.Sprintf("%x", result.TranscriptSHA256), StorageSHA256: digest(result.StorageTranscript), NetworkSHA256: digest(result.NetworkTranscript),
		})
	}
	return pair
}

func clusterMeasurementSummary(t *testing.T, runs []map[string]any) map[string]any {
	t.Helper()
	summaries := map[string]any{}
	for _, seed := range []uint64{7, 19, 42} {
		var authorized, wholeWall, parentCPU []int64
		var schedule string
		for _, run := range runs {
			if run["seed"] != seed || run["repetition"] == 0 || run["passed"] != true {
				continue
			}
			authorized = append(authorized, run["authorization_to_classified_reaped_nanos"].(int64))
			wholeWall = append(wholeWall, run["whole_scenario_wall_nanos"].(int64))
			before, after := run["parent_before"].(*hostexec.ResourceUsage), run["parent_after"].(*hostexec.ResourceUsage)
			if before == nil || after == nil || after.UserNanos < before.UserNanos || after.SystemNanos < before.SystemNanos {
				t.Error("whole-scenario parent CPU unavailable or regressed")
				continue
			}
			parentCPU = append(parentCPU, int64(after.UserNanos-before.UserNanos+after.SystemNanos-before.SystemNanos))
			pairs := run["pairs"].([]clusterMeasurementPair)
			var witness string
			for _, instance := range pairs[1].Instances {
				witness += instance.ChoiceSHA256 + ":" + instance.InterruptedChoiceSHA256 + "\n"
			}
			if schedule != "" && schedule != witness {
				t.Error("same seed changed choice evidence across warm repetitions", seed)
			}
			schedule = witness
		}
		summaries[stringSeed(seed)] = map[string]any{"authorized": clusterMeasurementDistribution(authorized), "whole_scenario_wall": clusterMeasurementDistribution(wholeWall), "parent_cpu": clusterMeasurementDistribution(parentCPU), "choice_witness_sha256": digest([]byte(schedule))}
		if len(authorized) != 3 || len(parentCPU) != 3 {
			t.Error("three completed warm measurements required for seed", seed)
		}
	}
	return summaries
}

func clusterMeasurementDistribution(values []int64) map[string]any {
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	if len(ordered) == 0 {
		return map[string]any{"samples": values, "available": false}
	}
	return map[string]any{"samples": values, "available": true, "min_nanos": ordered[0], "median_nanos": ordered[len(ordered)/2], "max_nanos": ordered[len(ordered)-1]}
}
