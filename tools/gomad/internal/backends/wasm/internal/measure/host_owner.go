package measure

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

// hostOwnerSelections are the single-guest selections of fn-172/fn-174.
var hostOwnerSelections = map[string]string{
	"activity": "^TestActivityTestSuite$",
	"workflow": "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$",
}

// hostOwnerGuestTags build the qualified slim Temporal guest
// (backend.QualifiedGuestBuildTags; backend imports this package).
var hostOwnerGuestTags = []string{"disable_grpc_modules", "gomad_wasm_slim", "sqlite3_dotlk", "test_dep"}

func measureHostOwnerProfile(ctx context.Context) error {
	root := input(ctx, "GOMAD_WASM_HOST_OWNER_PROFILE_ROOT")
	reference := input(ctx, "GOMAD_WASM_HOST_OWNER_REFERENCE_HELPER")
	for _, name := range []string{root, reference, input(ctx, "GOMAD_WASM_TEST_HELPER"), input(ctx, "GOMAD_STOCK_GO")} {
		if !filepath.IsAbs(name) {
			return fmt.Errorf("%w: host owner profile paths must be absolute", ErrRefused)
		}
	}
	diagnostic, err := helperPath(ctx)
	if err != nil {
		return err
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
	selectionName := cmp.Or(input(ctx, "GOMAD_WASM_HOST_OWNER_SELECTION"), "activity")
	selection, ok := hostOwnerSelections[selectionName]
	if !ok {
		return fmt.Errorf("%w: selection must be activity or workflow, got %q", ErrRefused, selectionName)
	}
	var seeds []uint64
	for _, field := range strings.Split(cmp.Or(input(ctx, "GOMAD_WASM_HOST_OWNER_SEEDS"), "0,1000003,2000006"), ",") {
		seed, err := strconv.ParseUint(strings.TrimSpace(field), 10, 64)
		if err != nil {
			return fmt.Errorf("seeds: %w", err)
		}
		seeds = append(seeds, seed)
	}
	repetitions, err := strconv.Atoi(cmp.Or(input(ctx, "GOMAD_WASM_HOST_OWNER_REPETITIONS"), "3"))
	if err != nil || repetitions < 1 || repetitions > 20 {
		return fmt.Errorf("%w: repetitions must be 1..20", ErrRefused)
	}
	repo := scopeFor(ctx).options.Repo
	commit, err := exec.CommandContext(ctx, "git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		return err
	}
	status, err := exec.CommandContext(ctx, "git", "-C", repo, "status", "--porcelain", "--", "tools/gomad").Output()
	if err != nil {
		return err
	}
	if len(status) != 0 {
		return fmt.Errorf("%w: host owner profile requires committed tools/gomad sources:\n%s", ErrRefused, status)
	}
	setInput(ctx, "GOMAD_WASM_PROTOTYPE_ROOT", root)
	evidence := filepath.Join(root, "evidence")
	if err := os.MkdirAll(evidence, 0700); err != nil {
		return err
	}
	module := input(ctx, "GOMAD_WASM_HOST_OWNER_MODULE")
	if module == "" {
		module = filepath.Join(root, "tests.wasm")
		_, seconds, err := prototypeBuild(ctx, repo, "test", "./tests", module, hostOwnerGuestTags...)
		if err != nil {
			return err
		}
		logf(ctx, "built Temporal guest in %.1fs", seconds)
	} else if !filepath.IsAbs(module) {
		return fmt.Errorf("%w: GOMAD_WASM_HOST_OWNER_MODULE must be absolute", ErrRefused)
	}
	request := func(seed uint64, run, helper string) (wasi.Request, error) {
		config, err := prototypeTemporalConfig(ctx, repo, "/workspace/tests", "-test.run="+run, "-persistenceType=sql", "-persistenceDriver=sqlite")
		if err != nil {
			return wasi.Request{}, err
		}
		request, err := prototypeRequest(ctx, module, config, seed, 30*time.Minute)
		if err != nil {
			return wasi.Request{}, err
		}
		role, sha := "diagnostic", diagnosticSHA
		if helper == reference {
			role, sha = "reference", referenceSHA
		}
		request.HelperPath, request.HelperSHA256 = helper, sha
		request.CompileCache = filepath.Join(root, "compile-cache-"+role)
		return request, nil
	}
	run := func(seed uint64, selection, helper, name string, warm bool) (wasi.Result, error) {
		prepared, err := request(seed, selection, helper)
		if err != nil {
			return wasi.Result{}, err
		}
		return runDiagnosticProfile(ctx, evidence, name, prepared, helper == diagnostic, warm)
	}
	for _, helper := range []string{reference, diagnostic} {
		role := "diagnostic"
		if helper == reference {
			role = "reference"
		}
		if _, err := run(0, "^$", helper, role+"-prepare", false); err != nil {
			return err
		}
	}
	var rows []hostOwnerRow
	for _, seed := range seeds {
		label := selectionName + "-seed" + stringSeed(seed)
		released, err := run(seed, selection, reference, label+"-reference", true)
		if err != nil {
			return err
		}
		if !bytes.Contains(released.Stdout, []byte("=== RUN   "+strings.Trim(strings.Split(selection, "/")[0], "^$"))) {
			return fmt.Errorf("host owner selection did not run")
		}
		witness := digest(released.ChoiceTrace.Bytes)
		if err := temporalWrite(ctx, filepath.Join(evidence, label+"-reference-choice.bin"), released.ChoiceTrace.Bytes); err != nil {
			return err
		}
		for repetition := -1; repetition < repetitions; repetition++ {
			name := label + "-rep" + strconv.Itoa(repetition)
			if repetition < 0 {
				name = label + "-warmup"
			}
			result, err := run(seed, selection, diagnostic, name, true)
			if err != nil {
				return err
			}
			if err := requireDiagnosticEvidence(ctx, released, result); err != nil {
				return err
			}
			if got := digest(result.ChoiceTrace.Bytes); got != witness {
				return fmt.Errorf("%s: choice witness %s differs from release %s", name, got, witness)
			}
			report, err := decodeDiagnosticReport(result.HelperStderr())
			if err != nil {
				return err
			}
			if err := requireAttributedProfile(report); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			row := hostOwnerRow{Seed: seed, Repetition: repetition, Warmup: repetition < 0, ChoiceWitnessSHA256: witness, ReleaseWitnessSHA256: witness, Calls: result.Calls,
				ReleaseAuthorizedNanos: released.AuthorizationToTerminalNanos, AuthorizedNanos: result.AuthorizationToTerminalNanos,
				Reconciliation: report.Reconciliation, ABIOps: report.ABIOps, Owners: report.Owners, Nested: report.Nested, ExecuteWallNanos: report.Phases["execute"].WallNanos}
			if released.AuthorizationToTerminalNanos != 0 {
				row.DiagnosticOverheadRatio = float64(result.AuthorizationToTerminalNanos) / float64(released.AuthorizationToTerminalNanos)
			}
			if err := temporalJSON(ctx, filepath.Join(evidence, name+"-owner-profile.json"), row); err != nil {
				return err
			}
			rows = append(rows, row)
		}
	}
	owners, err := compiledOwnerIdentities()
	if err != nil {
		return err
	}
	moduleSHA, err := fileHash(ctx, module)
	if err != nil {
		return err
	}
	cpuinfo, requiredHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	return temporalJSON(ctx, filepath.Join(root, "host-owner-profile.json"), map[string]any{
		"diagnostic_only": true, "selection": selectionName, "test_run": selection, "seeds": seeds, "warm_repetitions": repetitions, "warmup_per_seed": 1,
		"source_commit": strings.TrimSpace(string(commit)), "helper_sha256": diagnosticSHA, "reference_helper_sha256": referenceSHA,
		"module_sha256": moduleSHA, "guest_build_tags": hostOwnerGuestTags, "engine": wasi.PinnedEngine(), "model_sha256": wasi.ImplementationSHA256(), "compiled_owner_identities": owners,
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "required_host_measured": requiredHost, "host_cpuinfo": cpuinfo, "host_boot_id": hostOwnerBootID(),
		"reconciliation_rule": "|execute - (attributed host + guest residual)| <= max(50us, 0.5% of execute wall); guest residual is measured between import trampolines",
		"scope":               "unsampled diagnostic-profile helper; owner wall is exclusive of nested owner spans; per-call quantiles are log-linear bucket bounds (relative error < 6.25%); reference runs use the release helper with separate compile caches",
		"rows":                rows, "owner_summary": hostOwnerSummary(rows),
		"rerun": rerunRecord(ctx, "go", repo, "host-owner-profile", scopeFor(ctx).options.Inputs, true),
	})
}

type hostOwnerRow struct {
	Seed                    uint64                      `json:"seed"`
	Repetition              int                         `json:"repetition"`
	Warmup                  bool                        `json:"warmup"`
	ChoiceWitnessSHA256     string                      `json:"choice_witness_sha256"`
	ReleaseWitnessSHA256    string                      `json:"release_choice_witness_sha256"`
	Calls                   uint64                      `json:"calls"`
	ReleaseAuthorizedNanos  uint64                      `json:"release_authorization_to_terminal_nanos"`
	AuthorizedNanos         uint64                      `json:"authorization_to_terminal_nanos"`
	DiagnosticOverheadRatio float64                     `json:"diagnostic_overhead_ratio"`
	Reconciliation          *diagnosticReconciliation   `json:"reconciliation"`
	ABIOps                  map[string]diagnosticOp     `json:"abi_ops"`
	Owners                  map[string]diagnosticSeries `json:"owners"`
	Nested                  map[string]diagnosticNested `json:"nested"`
	ExecuteWallNanos        uint64                      `json:"execute_wall_nanos"`
}

type hostOwnerShare struct {
	Runs                 int     `json:"runs"`
	MedianExclusiveNanos uint64  `json:"median_execute_exclusive_wall_nanos"`
	MinExclusiveNanos    uint64  `json:"min_execute_exclusive_wall_nanos"`
	MaxExclusiveNanos    uint64  `json:"max_execute_exclusive_wall_nanos"`
	MedianShareOfExecute float64 `json:"median_share_of_execute"`
	MinShareOfExecute    float64 `json:"min_share_of_execute"`
	MaxShareOfExecute    float64 `json:"max_share_of_execute"`
	MedianCalls          uint64  `json:"median_count"`
}

// hostOwnerSummary is each owner's (and the guest residual's) median and
// range over warm repetitions, per seed and across all seeds, of
// execute-exclusive wall and its share of execute wall. An owner absent
// from a run counts as zero there.
func hostOwnerSummary(rows []hostOwnerRow) map[string]map[string]hostOwnerShare {
	groups := map[string][]hostOwnerRow{}
	for _, row := range rows {
		if !row.Warmup {
			groups["all"] = append(groups["all"], row)
			groups["seed"+stringSeed(row.Seed)] = append(groups["seed"+stringSeed(row.Seed)], row)
		}
	}
	median := func(values []float64) float64 {
		slices.Sort(values)
		if len(values)%2 == 1 {
			return values[len(values)/2]
		}
		return (values[len(values)/2-1] + values[len(values)/2]) / 2
	}
	summary := map[string]map[string]hostOwnerShare{}
	for group, members := range groups {
		names := map[string]bool{"guest_residual": true}
		for _, row := range members {
			for name := range row.Owners {
				names[name] = true
			}
		}
		summary[group] = map[string]hostOwnerShare{}
		for name := range names {
			var walls, shares, counts []float64
			for _, row := range members {
				wall, count := row.Owners[name].ExecuteExclusiveWallNanos, row.Owners[name].Count
				if name == "guest_residual" {
					wall, count = row.Reconciliation.GuestResidualWallNanos, 0
				}
				walls, counts = append(walls, float64(wall)), append(counts, float64(count))
				shares = append(shares, float64(wall)/float64(max(row.ExecuteWallNanos, 1)))
			}
			share := hostOwnerShare{Runs: len(members), MedianExclusiveNanos: uint64(median(slices.Clone(walls))), MinExclusiveNanos: uint64(slices.Min(walls)), MaxExclusiveNanos: uint64(slices.Max(walls)),
				MedianShareOfExecute: median(slices.Clone(shares)), MinShareOfExecute: slices.Min(shares), MaxShareOfExecute: slices.Max(shares), MedianCalls: uint64(median(counts))}
			summary[group][name] = share
		}
	}
	return summary
}

func hostOwnerBootID() string {
	data, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
