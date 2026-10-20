package measure

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

// clusterOwnerWaitingOwners are charged inside imports but are not host
// work: the baton wait while other instances run, and the profiler itself.
var clusterOwnerWaitingOwners = map[string]bool{"handoff_wait": true, "instrumentation": true}

// clusterOwnerProcess is one completed cluster helper process of a scenario
// run: its merged profile and per-thread reconciliation rows.
type clusterOwnerProcess struct {
	Label              string                      `json:"label"`
	HelperSHA256       string                      `json:"helper_sha256"`
	ExecuteWallNanos   uint64                      `json:"execute_wall_nanos"`
	Reconciliation     *diagnosticReconciliation   `json:"reconciliation"`
	Threads            []diagnosticThread          `json:"threads"`
	Owners             map[string]diagnosticSeries `json:"owners"`
	ABIOps             map[string]diagnosticOp     `json:"abi_ops"`
	HostWorkNanos      uint64                      `json:"host_work_execute_exclusive_nanos"`
	WaitingNanos       uint64                      `json:"waiting_execute_exclusive_nanos"`
	AuthorizedNanos    uint64                      `json:"authorization_to_terminal_nanos"`
	ReleaseWitness     string                      `json:"release_choice_witness_sha256"`
	DiagnosticWitness  string                      `json:"choice_witness_sha256"`
	ReleaseAuthorized  uint64                      `json:"release_authorization_to_terminal_nanos"`
	DiagnosticOverhead float64                     `json:"diagnostic_overhead_ratio"`
}

type clusterOwnerRun struct {
	Seed       uint64                `json:"seed"`
	Repetition int                   `json:"repetition"`
	Warmup     bool                  `json:"warmup"`
	Processes  []clusterOwnerProcess `json:"processes"`
}

type clusterOwnerEvidence struct {
	Incarnations map[string]wasi.Result
	Documents    map[string][]byte
}

func requireClusterOwnerEvidence(ctx context.Context, before, after clusterOwnerEvidence) error {
	if len(before.Incarnations) != len(after.Incarnations) || len(before.Documents) != len(after.Documents) {
		return fmt.Errorf("profiling changed cluster evidence inventory")
	}
	imageIdentity := func(image *wasi.StorageImage) string {
		if image == nil {
			return ""
		}
		return image.Identity()
	}
	for label, expected := range before.Incarnations {
		actual, ok := after.Incarnations[label]
		if !ok {
			return fmt.Errorf("profiling lost incarnation evidence %s", label)
		}
		if err := requireDiagnosticEvidence(ctx, expected, actual); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		if !bytes.Equal(expected.StorageTranscript, actual.StorageTranscript) || !bytes.Equal(expected.InterruptedChoiceBytes, actual.InterruptedChoiceBytes) || !bytes.Equal(expected.InterruptedDiagnosticBytes, actual.InterruptedDiagnosticBytes) || imageIdentity(expected.StorageImage) != imageIdentity(actual.StorageImage) {
			return fmt.Errorf("profiling changed incarnation storage or interrupted evidence: %s", label)
		}
	}
	for name, expected := range before.Documents {
		actual, ok := after.Documents[name]
		if !ok || !bytes.Equal(expected, actual) {
			return fmt.Errorf("profiling changed cluster document %s", name)
		}
	}
	return nil
}

// clusterOwnerRequestFor derives the request one role dispatches: the
// prepared scenario request bound to that role's own admitted helper, whose
// admission must carry exactly the role's features (none for the release
// reference, diagnostic-profile for the profiled runs). The runtime control is
// copied so the roles never share mutable state.
func clusterOwnerRequestFor(prepared wasi.Request, helpers map[string]diagnosticHelperAdmission, helper, helperSHA256 string, features []string) (wasi.Request, error) {
	admitted, ok := helpers[helper]
	if !ok {
		return wasi.Request{}, fmt.Errorf("cluster owner helper %s is not admitted", helper)
	}
	if !slices.Equal(admitted.Features, features) {
		return wasi.Request{}, fmt.Errorf("cluster owner helper %s admitted with features %v, want %v", helper, admitted.Features, features)
	}
	if len(helperSHA256) != 64 || prepared.Runtime == nil {
		return wasi.Request{}, fmt.Errorf("cluster owner request needs a helper digest and runtime control")
	}
	request := prepared
	control := *prepared.Runtime
	request.Runtime = &control
	request.HelperPath, request.HelperSHA256 = helper, helperSHA256
	return request, nil
}

// clusterOwnerWitness is a scenario phase's per-incarnation choice evidence.
func clusterOwnerWitness(label string, requests []wasi.Request, results []wasi.Result) string {
	var witness strings.Builder
	for i, result := range results {
		fmt.Fprintf(&witness, "%s %s/%d %s %s\n", label, requests[i].Broker.Node.ID, requests[i].Broker.Node.Incarnation, digest(result.ChoiceTrace.Bytes), digest(result.InterruptedChoiceBytes))
	}
	return witness.String()
}

// clusterOwnerSplit separates host work from waiting time in a process's
// execute-exclusive owner wall.
func clusterOwnerSplit(owners map[string]diagnosticSeries) (work, waiting uint64) {
	for name, owner := range owners {
		if clusterOwnerWaitingOwners[name] {
			waiting += owner.ExecuteExclusiveWallNanos
		} else {
			work += owner.ExecuteExclusiveWallNanos
		}
	}
	return work, waiting
}

func measureClusterOwnerProfile(ctx context.Context) (resultErr error) {
	base := input(ctx, "GOMAD_WASM_CLUSTER_OWNER_PROFILE_ROOT")
	reference, referenceReceipt := input(ctx, "GOMAD_WASM_CLUSTER_OWNER_REFERENCE_HELPER"), input(ctx, "GOMAD_WASM_CLUSTER_HELPER_ADMISSION")
	diagnostic, diagnosticReceipt := input(ctx, "GOMAD_WASM_TEST_HELPER"), input(ctx, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION")
	repo := scopeFor(ctx).options.Repo
	relative, err := filepath.Rel(repo, base)
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: measurement root must be an absolute clean path outside the repository", ErrRefused)
	}
	for _, name := range []string{reference, referenceReceipt, diagnostic, diagnosticReceipt} {
		if !filepath.IsAbs(name) {
			return fmt.Errorf("%w: helpers and admission receipts must be absolute", ErrRefused)
		}
	}
	referenceSHA, err := fileHash(ctx, reference)
	if err != nil {
		return err
	}
	diagnosticSHA, err := fileHash(ctx, diagnostic)
	if err != nil {
		return err
	}
	if referenceSHA == diagnosticSHA {
		return fmt.Errorf("%w: reference helper must be the distinct release build", ErrRefused)
	}
	var seeds []uint64
	for _, field := range strings.Split(cmp.Or(input(ctx, "GOMAD_WASM_CLUSTER_OWNER_SEEDS"), "7,19,42"), ",") {
		seed, err := strconv.ParseUint(strings.TrimSpace(field), 10, 64)
		if err != nil {
			return fmt.Errorf("seeds: %w", err)
		}
		seeds = append(seeds, seed)
	}
	repetitions, err := strconv.Atoi(cmp.Or(input(ctx, "GOMAD_WASM_CLUSTER_OWNER_REPETITIONS"), "3"))
	if err != nil || repetitions < 1 || repetitions > 20 {
		return fmt.Errorf("%w: repetitions must be 1..20", ErrRefused)
	}
	root, err := diagnosticRunRoot(base, "cluster-owner-profile-")
	if err != nil {
		return err
	}
	admission, err := newDiagnosticAdmission(ctx, repo, root)
	if err != nil {
		return err
	}
	if err := clusterMeasurementCandidateAdmitHelper(ctx, admission, reference, referenceReceipt); err != nil {
		return err
	}
	if err := admission.addHelper(ctx, diagnostic, diagnosticReceipt, []string{"diagnostic-profile"}); err != nil {
		return err
	}
	setInput(ctx, "GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", root)
	setInput(ctx, "GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE", filepath.Join(root, "compile-cache"))
	prepared, _, err := temporalRecoveryRequestWithAdmission(ctx, "temporal_broker", admission)
	if err != nil {
		return err
	}
	if prepared.DiagnosticConfigurationRevision() != "" || prepared.GuestProfile() != nil || prepared.Runtime == nil || prepared.Runtime.Diagnostics {
		return fmt.Errorf("cluster owner profile requires the uninstrumented execution profile")
	}
	provenanceSHA, err := fileHash(ctx, filepath.Join(root, "temporal_broker-provenance.json"))
	if err != nil {
		return err
	}
	roles := map[string]wasi.Request{}
	for _, role := range []struct {
		name, helper, sha string
		features          []string
	}{
		{"reference", reference, referenceSHA, []string{}}, {"diagnostic", diagnostic, diagnosticSHA, []string{"diagnostic-profile"}},
	} {
		built, err := clusterOwnerRequestFor(prepared, admission.Helpers, role.helper, role.sha, role.features)
		if err != nil {
			return err
		}
		if err := admission.check(ctx, root, role.name+"-request", built, true); err != nil {
			return err
		}
		if err := temporalJSON(ctx, filepath.Join(root, "temporal_broker-"+role.name+"-provenance.json"), map[string]any{
			"role": role.name, "helper": built.HelperPath, "helper_sha256": built.HelperSHA256, "helper_features": role.features,
			"helper_admission": admission.Helpers[role.helper].Receipt, "module_sha256": built.ModuleSHA256, "runtime_identity": built.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, "engine": built.Engine, "shared_preparation_sha256": provenanceSHA,
		}); err != nil {
			return err
		}
		roles[role.name] = built
	}
	var runs []clusterOwnerRun
	defer func() {
		resultErr = errors.Join(resultErr, admission.check(ctx, root, "final", roles["diagnostic"], false))
		owners, err := compiledOwnerIdentities()
		resultErr = errors.Join(resultErr, err)
		cpuinfo, requiredHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
		resultErr = errors.Join(resultErr, temporalJSON(ctx, filepath.Join(root, "cluster-owner-profile.json"), map[string]any{
			"diagnostic_only": true, "selection": "TestTemporalIndependentClustersPartitionHealWholeClusterCrash", "seeds": seeds, "warm_repetitions": repetitions, "warmup_per_seed": 1,
			"source_commit": admission.SourceHead, "host_admission": admission.Receipt, "helper_admissions": map[string]string{"release": referenceReceipt, "diagnostic": diagnosticReceipt},
			"helper_sha256": diagnosticSHA, "reference_helper_sha256": referenceSHA, "identities": variantIdentities(roles["diagnostic"], wasi.Result{}), "module_receipts": admission.ModuleReceipts,
			"compiled_owner_identities": owners, "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "required_host_measured": requiredHost, "host_cpuinfo": cpuinfo, "host_boot_id": hostOwnerBootID(),
			"runs": runs, "complete": resultErr == nil && len(runs) == len(seeds)*(repetitions+1),
			"waiting_scope":       "handoff_wait (baton wait while other instances run) and instrumentation (profiler glue) are charged inside imports but reported as waiting, never host work; per-thread execute walls include them, so merged execute walls overlap in real time",
			"reconciliation_rule": "per process and per store thread: execute == guest residual + all owners (instrumentation included) within max(50us, 0.5% of execute wall)",
			"rerun":               rerunRecord(ctx, "go", repo, "cluster-owner-profile", scopeFor(ctx).options.Inputs, true),
		}))
	}()
	scenario := func(seed uint64, role wasi.Request, name string, canonical *clusterOwnerEvidence) (witness string, completed []*wasi.ClusterResult, scenarioErr error) {
		canonical.Incarnations = make(map[string]wasi.Result)
		canonical.Documents = make(map[string][]byte)
		evidence := filepath.Join(root, name)
		if err := os.MkdirAll(evidence, 0700); err != nil {
			return "", nil, err
		}
		local := role
		control := *role.Runtime
		control.Seed, local.Runtime = seed, &control
		helperRole := "diagnostic"
		if role.HelperPath == reference {
			helperRole = "reference"
		}
		local.CompileCache = filepath.Join(root, fmt.Sprintf("seed%d-%s-compile-cache", seed, helperRole))
		if err := os.MkdirAll(local.CompileCache, 0700); err != nil {
			return "", nil, err
		}
		if err := admission.check(ctx, evidence, "scenario-before", local, true); err != nil {
			return "", nil, err
		}
		defer func() {
			scenarioErr = errors.Join(scenarioErr, admission.check(ctx, evidence, "scenario-after", local, false))
		}()
		var choices strings.Builder
		err := temporalIndependentClustersPartitionHealWholeClusterCrash(ctx, local, evidence, temporalPairObserver{
			Admission: func(ctx context.Context, evidence, label string, requests []wasi.Request, before bool) error {
				for i, request := range requests {
					if err := admission.check(ctx, evidence, fmt.Sprintf("%s-%d-%t", label, i, before), request, before); err != nil {
						return err
					}
				}
				return nil
			},
			Observe: func(ctx context.Context, evidence, label string, requests []wasi.Request, results []wasi.Result, failures []error) error {
				for i := range results {
					if failures[i] != nil || requests[i].Broker == nil {
						return fmt.Errorf("%s: incarnation result unavailable: %s %v", name, label, failures[i])
					}
					key := fmt.Sprintf("%s/%s/%d", label, requests[i].Broker.Node.ID, requests[i].Broker.Node.Incarnation)
					if _, exists := canonical.Incarnations[key]; exists {
						return fmt.Errorf("duplicate cluster incarnation evidence %s", key)
					}
					canonical.Incarnations[key] = results[i]
				}
				choices.WriteString(clusterOwnerWitness(label, requests, results))
				return nil
			},
			Cluster: func(ctx context.Context, label string, result *wasi.ClusterResult) error {
				if err := temporalWrite(ctx, filepath.Join(evidence, label+"-helper-stderr.txt"), result.HelperStderr()); err != nil {
					return err
				}
				completed = append(completed, result)
				return nil
			},
		})
		if err != nil {
			return "", completed, err
		}
		if len(completed) == 0 {
			return "", nil, fmt.Errorf("%s: no completed in-process cluster helper", name)
		}
		for _, document := range []string{"two-cluster-world.json", "two-cluster-network-record.json", "two-cluster-replayed-world.json", "two-cluster-replayed-network-record.json"} {
			data, err := os.ReadFile(filepath.Join(evidence, document))
			if err != nil {
				return "", completed, err
			}
			canonical.Documents[document] = data
		}
		if err := temporalWrite(ctx, filepath.Join(evidence, "choice-witness.txt"), []byte(choices.String())); err != nil {
			return "", completed, err
		}
		return choices.String(), completed, nil
	}
	for _, seed := range seeds {
		var releaseEvidence clusterOwnerEvidence
		releaseWitness, released, err := scenario(seed, roles["reference"], fmt.Sprintf("seed%d-reference", seed), &releaseEvidence)
		if err != nil {
			return err
		}
		for _, result := range released {
			if len(result.HelperStderr()) != 0 {
				return fmt.Errorf("release helper emitted diagnostic bytes")
			}
		}
		for repetition := -1; repetition < repetitions; repetition++ {
			name := fmt.Sprintf("seed%d-rep%d", seed, repetition)
			if repetition < 0 {
				name = fmt.Sprintf("seed%d-warmup", seed)
			}
			var diagnosticEvidence clusterOwnerEvidence
			witness, completed, err := scenario(seed, roles["diagnostic"], name, &diagnosticEvidence)
			if err != nil {
				return err
			}
			if err := requireClusterOwnerEvidence(ctx, releaseEvidence, diagnosticEvidence); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if witness != releaseWitness {
				return fmt.Errorf("%s: choice witness differs from the release run", name)
			}
			if len(completed) != len(released) {
				return fmt.Errorf("%s: %d cluster processes, release had %d", name, len(completed), len(released))
			}
			run := clusterOwnerRun{Seed: seed, Repetition: repetition, Warmup: repetition < 0}
			for i, result := range completed {
				report, err := decodeDiagnosticReport(result.HelperStderr())
				if err != nil {
					return fmt.Errorf("%s process %d: %w", name, i, err)
				}
				if err := requireAttributedProfile(report); err != nil {
					return fmt.Errorf("%s process %d: %w", name, i, err)
				}
				if len(report.Threads) == 0 {
					return fmt.Errorf("%s process %d: merged profile without store-thread rows", name, i)
				}
				work, waiting := clusterOwnerSplit(report.Owners)
				process := clusterOwnerProcess{Label: fmt.Sprint(i), HelperSHA256: diagnosticSHA, ExecuteWallNanos: report.Phases["execute"].WallNanos, Reconciliation: report.Reconciliation,
					Threads: report.Threads, Owners: report.Owners, ABIOps: report.ABIOps, HostWorkNanos: work, WaitingNanos: waiting, AuthorizedNanos: result.AuthorizationToTerminalNanos,
					ReleaseWitness: digest([]byte(releaseWitness)), DiagnosticWitness: digest([]byte(witness)), ReleaseAuthorized: released[i].AuthorizationToTerminalNanos}
				if released[i].AuthorizationToTerminalNanos != 0 {
					process.DiagnosticOverhead = float64(result.AuthorizationToTerminalNanos) / float64(released[i].AuthorizationToTerminalNanos)
				}
				run.Processes = append(run.Processes, process)
			}
			runs = append(runs, run)
			if err := temporalJSON(ctx, filepath.Join(root, name, "cluster-owner-run.json"), run); err != nil {
				return err
			}
		}
	}
	return nil
}
