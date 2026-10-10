package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad_wasm/toolchain"
)

func cooperativeRequest(t *testing.T, fixture string) Request {
	t.Helper()
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Fatal("qualified guest gate requires explicit GOMAD3_STOCK_GO")
	}
	rootOutput, err := exec.Command(compiler, "env", "GOROOT").Output()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "go")
	if err := os.CopyFS(root, os.DirFS(strings.TrimSpace(string(rootOutput)))); err != nil {
		t.Fatal(err)
	}
	native, err := filepath.Abs("../../gomad3/toolchain/runtime/overlay/src/runtime")
	if err != nil {
		t.Fatal(err)
	}
	overlay, identity, err := toolchain.BuildOverlay(root, native, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "runtime.wasm")
	command := exec.Command(filepath.Join(root, "bin/go"), "build", "-trimpath", "-overlay", overlay, "-o", module, fixture)
	command.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build actual cooperative guest: %v\n%s", err, output)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	helper := helperPath(t)
	config := testConfig()
	config.Profile = CooperativeProfile
	config.Clock.ReadStepNanos = 1000
	execution := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: identity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(identity))}
	return Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: time.Minute, Runtime: &RuntimeControl{Seed: 7, Diagnostics: true, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: execution, Limit: 1 << 20}}}
}

func TestCooperativeGoGuestForcesExactReplay(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	observed, err := Run(context.Background(), request)
	if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || *observed.ExitCode != 0 || !observed.Reaped {
		t.Fatalf("actual guest: %s %s err=%v stderr=%s", observed.Termination, observed.Message, err, observed.Stderr)
	}
	if observed.ChoiceTrace.Summary.Runnable == 0 || observed.ChoiceTrace.Summary.SelectPoll == 0 || len(observed.DiagnosticTrace.Records) != len(observed.ChoiceTrace.Records) || !bytes.Contains(observed.Stdout, []byte("timer")) {
		t.Fatalf("actual hook coverage: summary=%+v diagnostics=%d stdout=%s", observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records), observed.Stdout)
	}
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
	request.Runtime.Diagnostics = false
	replayed, err := Run(context.Background(), request)
	if err != nil || replayed.Termination != "exit" || !replayed.Reaped || !bytes.Equal(observed.ChoiceTrace.Bytes, replayed.ChoiceTrace.Bytes) || !bytes.Equal(observed.Stdout, replayed.Stdout) || !bytes.Equal(observed.Transcript, replayed.Transcript) {
		t.Fatalf("actual exact replay: %s %s err=%v stderr=%s trace=%x/%x evidence=%x/%x", replayed.Termination, replayed.Message, err, replayed.Stderr, observed.ChoiceTrace.SHA256, replayed.ChoiceTrace.SHA256, sha256.Sum256(observed.Transcript), sha256.Sum256(replayed.Transcript))
	}
	t.Logf("actual cooperative guest record/exact replay: module=%s helper=%s runtime=%s choices=%+v diagnostics=%d", request.ModuleSHA256, request.HelperSHA256, request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records))
	for _, mutation := range []struct {
		name   string
		reason choice.DivergenceReason
	}{{"missing", choice.DivergenceTapeExhausted}, {"extra", choice.DivergenceTapeUnconsumed}, {"kind", choice.DivergenceKind}, {"site", choice.DivergenceSite}, {"enabled", choice.DivergenceAlternativeSet}} {
		t.Run(mutation.name, func(t *testing.T) {
			plan := tape
			if mutation.name == "missing" {
				plan, err = tape.Prefix(0)
			} else {
				records := slices.Clone(observed.ChoiceTrace.Records)
				target := slices.IndexFunc(records, func(record choice.Record) bool { return record.Kind == choice.KindRunnable })
				if target < 0 {
					t.Fatal("fixture has no runnable record to alter")
				}
				switch mutation.name {
				case "extra":
					extra := records[target]
					extra.Ordinal = uint64(len(records))
					records = append(records, extra)
				case "kind":
					records[target].Kind = choice.KindSelectPoll
				case "site":
					records[target].Flags &^= choice.FlagSiteMissing
					records[target].SiteOffset++
				case "enabled":
					records[target].AlternativeSetDigest = sha256.Sum256([]byte("inapplicable alternatives"))
				}
				altered, buildErr := choice.BuildTrace(records, choice.TerminalComplete)
				if buildErr != nil {
					t.Fatal(buildErr)
				}
				plan, err = choice.ProjectReplayPlan(altered, request.Runtime.Choice.ExecutionIdentity)
			}
			if err != nil {
				t.Fatal(err)
			}
			request.Runtime.Choice.Tape = &plan
			result, runErr := Run(t.Context(), request)
			if runErr != nil || result.Termination != "divergence" || result.ChoiceDivergence == nil || result.ChoiceDivergence.Reason != mutation.reason || !result.Reaped {
				t.Fatalf("actual tape %s: %s %s divergence=%+v err=%v", mutation.name, result.Termination, result.Message, result.ChoiceDivergence, runErr)
			}
		})
	}
}

func TestCooperativeRuntimeProbes(t *testing.T) {
	request := cooperativeRequest(t, "../testdata/runtime_probes")
	request.Config.Environment = append(request.Config.Environment, "EXTRA_ENTROPY=no_")
	observed, err := Run(context.Background(), request)
	if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || (*observed.ExitCode != 0 && *observed.ExitCode != 17) || !observed.Reaped {
		t.Fatalf("actual runtime probes: %s %s err=%v stdout=%s stderr=%s", observed.Termination, observed.Message, err, observed.Stdout, observed.Stderr)
	}
	for _, witness := range []string{"entropy ", "map ", "runq ", "timer callbacks ", "timer parking", "note negative park wake", "collector automatic finalizer cleanup 1 1", "canary "} {
		if !bytes.Contains(observed.Stdout, []byte(witness)) {
			t.Fatalf("missing runtime witness %q: %s", witness, observed.Stdout)
		}
	}
	t.Logf("runtime probes stdout=%s summary=%+v diagnostics=%d", observed.Stdout, observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records))
	base := observed
	t.Run("diagnostic-divergence", func(t *testing.T) {
		records := slices.Clone(base.DiagnosticTrace.Records)
		for index := range records {
			records[index].RuntimeRandDraws++
		}
		altered, err := choice.BuildDiagnosticTrace(records, base.DiagnosticTrace.Capacity)
		if err != nil {
			t.Fatal(err)
		}
		difference, err := choice.DiffDiagnostics(base.DiagnosticTrace.Bytes, altered.Bytes)
		if err != nil || difference == nil || difference.Ordinal != 0 || !slices.Contains(difference.Fields, "runtime_rand_draws") {
			t.Fatalf("actual guest diagnostic mutation undetected: %+v %v", difference, err)
		}
	})
	t.Run("frozen-clock-parking", func(t *testing.T) {
		frozen := request
		frozen.Config.Clock.ReadStepNanos = 0
		result, err := Run(t.Context(), frozen)
		if err != nil || result.Termination != "exit" || !result.Reaped || !bytes.Contains(result.Stdout, []byte("timer parking")) || !bytes.Contains(result.Stdout, []byte("note negative park wake")) || !bytes.Contains(result.Stdout, []byte("collector automatic finalizer cleanup 1 1")) {
			t.Fatalf("frozen-clock idle handshake: %s %s err=%v stdout=%s stderr=%s", result.Termination, result.Message, err, result.Stdout, result.Stderr)
		}
		t.Logf("read-step=0 timer deadlines and negative note parking progressed: %s", result.Stdout)
	})
	for _, perturbation := range []string{"application-draws", "application-key", "runtime-seed"} {
		t.Run(perturbation, func(t *testing.T) {
			changed := request
			changed.Config.Environment = append([]string(nil), request.Config.Environment...)
			control := *request.Runtime
			changed.Runtime = &control
			switch perturbation {
			case "application-draws":
				changed.Config.Environment[len(changed.Config.Environment)-1] = "EXTRA_ENTROPY=yes"
			case "application-key":
				changed.Config.EntropyKey[31] = 99
			case "runtime-seed":
				changed.Runtime.Seed++
			}
			result, err := Run(context.Background(), changed)
			if err != nil || result.Termination != "exit" || !result.Reaped {
				t.Fatalf("entropy perturbation: %s %s %v %s", result.Termination, result.Message, err, result.Stderr)
			}
			first, rest, ok := bytes.Cut(result.Stdout, []byte{'\n'})
			baseFirst, baseRest, baseOK := bytes.Cut(base.Stdout, []byte{'\n'})
			if !ok || !baseOK {
				t.Fatal("missing entropy observation")
			}
			if perturbation == "runtime-seed" {
				if !bytes.Equal(first, baseFirst) || bytes.Equal(rest, baseRest) {
					t.Fatal("runtime seed changed application entropy or failed to vary runtime behavior")
				}
			} else {
				if perturbation == "application-key" && bytes.Equal(first, baseFirst) {
					t.Fatal("application entropy key had no effect")
				}
				if !bytes.Equal(rest, baseRest) || !bytes.Equal(result.ChoiceTrace.Bytes, base.ChoiceTrace.Bytes) || len(result.DiagnosticTrace.Records) != len(base.DiagnosticTrace.Records) {
					t.Fatal("application entropy perturbed runtime behavior")
				}
				for i, record := range result.DiagnosticTrace.Records {
					previous := base.DiagnosticTrace.Records[i]
					if record.RunqDraws != previous.RunqDraws || record.SchedulerDraws != previous.SchedulerDraws || record.SelectDraws != previous.SelectDraws || record.RuntimeRandDraws != previous.RuntimeRandDraws || record.RuntimeCheapRandDraws != previous.RuntimeCheapRandDraws || record.TimerDraws != previous.TimerDraws {
						t.Fatalf("application entropy perturbed seeded draw counters at %d", i)
					}
				}
			}
		})
	}
	tape, err := choice.ProjectReplayPlan(base.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	last := uint64(len(tape.Decisions) - 1)
	if tape.Decisions[last].Kind != choice.KindSelectPoll || tape.Decisions[last].Alternatives != 2 {
		t.Fatal("canary is not the final two-way branch")
	}
	prefix, err := choice.BuildRankPrefix(tape, last, 1-tape.Decisions[last].Selected)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModePrefix, &prefix
	alternate, err := Run(context.Background(), request)
	if err != nil || alternate.Termination != "exit" || alternate.ExitCode == nil || *alternate.ExitCode == *base.ExitCode || !alternate.Reaped {
		t.Fatalf("alternate forced canary: %s %s %v stdout=%s stderr=%s", alternate.Termination, alternate.Message, err, alternate.Stdout, alternate.Stderr)
	}
	alternateTape, err := choice.ProjectReplayPlan(alternate.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &alternateTape
	request.Runtime.Diagnostics = false
	exact, err := Run(context.Background(), request)
	if err != nil || exact.Termination != "exit" || exact.ExitCode == nil || *exact.ExitCode != *alternate.ExitCode || !bytes.Equal(exact.Stdout, alternate.Stdout) || !bytes.Equal(exact.ChoiceTrace.Bytes, alternate.ChoiceTrace.Bytes) || !bytes.Equal(exact.Transcript, alternate.Transcript) {
		t.Fatalf("alternate exact replay: %s %s %v stdout=%s stderr=%s", exact.Termination, exact.Message, err, exact.Stdout, exact.Stderr)
	}
	t.Logf("forced alternate canary exit=%d then full-tape exact replay; application/runtime entropy independence checked", *alternate.ExitCode)
}

func TestCooperativePreflightRejectsIncompleteRuntimeABI(t *testing.T) {
	request := fixtureRequest(t, `(module (memory (export "memory") 1) (func (export "_start")))`, nil)
	request.Config.Profile = CooperativeProfile
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	request.Runtime = &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}}
	if _, err := Run(context.Background(), request); err == nil {
		t.Fatal("cooperative profile authorized guest without runtime ABI")
	}
}
