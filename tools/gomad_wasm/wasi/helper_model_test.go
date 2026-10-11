package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"os"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
)

func TestHelperModelPreservesGoOwnerEvidence(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	requireSmallHelperParityModule(t, request)
	for _, frozen := range []bool{false, true} {
		t.Run(map[bool]string{false: "read-step", true: "frozen"}[frozen], func(t *testing.T) {
			if frozen {
				request.Config.Clock.ReadStepNanos = 0
			}
			remote := request
			remote.remoteModel = true
			reference, err := Run(t.Context(), remote)
			if err != nil {
				t.Fatal(err)
			}
			requireExit(t, "Go reference", reference, 0)
			local, err := Run(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			if diff := sameEvidence(reference, local); diff != "" || !bytes.Equal(reference.DiagnosticTrace.Bytes, local.DiagnosticTrace.Bytes) || reference.EnvironmentIdentity != local.EnvironmentIdentity || reference.Calls != local.Calls || reference.TimeAdvances != local.TimeAdvances {
				t.Fatalf("helper evidence differs in %s: %s %s diagnostics=%v calls=%d/%d advances=%d/%d", diff, local.Termination, local.Message, bytes.Equal(reference.DiagnosticTrace.Bytes, local.DiagnosticTrace.Bytes), reference.Calls, local.Calls, reference.TimeAdvances, local.TimeAdvances)
			}
			if local.Telemetry == nil || local.Telemetry.PrepareNanos == 0 || local.Telemetry.InstantiateNanos == 0 || local.Telemetry.ExecuteNanos == 0 || local.HelperResourceUsage == nil || reference.AuthorizationToTerminalNanos == 0 || local.AuthorizationToTerminalNanos == 0 {
				t.Fatalf("measured helper phase/resource telemetry missing: %+v %+v", local.Telemetry, local.HelperResourceUsage)
			}
			t.Logf("measured authorized execution Go-owner=%dns helper-owner=%dns (includes fresh instantiation and evidence flush)", reference.AuthorizationToTerminalNanos, local.AuthorizationToTerminalNanos)
			tape, err := choice.ProjectReplayPlan(reference.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []choice.Mode{choice.ModeReplay, choice.ModePrefix} {
				forced := request
				control := *request.Runtime
				choices := *control.Choice
				choices.Mode, choices.Tape = mode, &tape
				control.Choice = &choices
				forced.Runtime = &control
				replayed, err := Run(context.Background(), forced)
				if err != nil {
					t.Fatal(err)
				}
				if diff := sameEvidence(reference, replayed); diff != "" {
					t.Fatalf("forced mode %d differs in %s: %s %s", mode, diff, replayed.Termination, replayed.Message)
				}
			}
		})
	}
}

func TestHelperModelPreservesMaximumRemoteReply(t *testing.T) {
	imports := []Import{}
	for _, name := range []string{"config", "decision", "observation", "finish", "idle"} {
		imports = append(imports, Import{"gomad_wasm_v1", name, []string{"i32", "i32"}, []string{"i32"}})
	}
	imports = append(imports, Import{"wasi_snapshot_preview1", "random_get", []string{"i32", "i32"}, []string{"i32"}})
	request := fixtureRequest(t, `(module
 (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "decision" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "observation" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "idle" (func (param i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "random_get" (func $random (param i32 i32) (result i32)))
 (memory (export "memory") 65)
 (data (i32.const 16) "\00\00\00\01")
 (func (export "_start")
 i32.const 0 i32.const 16 call $config drop
 i32.const 64 i32.const 4194304 call $random drop
 i32.const 16 i32.const 16 call $finish drop))`, imports)
	request.InitialMemoryPages = 65
	request.Config.Profile = CooperativeProfile
	request.Config.Clock.ReadStepNanos = 0
	request.Config.Limits.TranscriptBytes = 64 << 20
	request.Runtime = &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 8192, ExecutionIdentity: choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}}}
	remote := request
	remote.remoteModel = true
	reference, err := Run(t.Context(), remote)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "maximum Go reply", reference, 0)
	local, err := Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if difference := sameEvidence(reference, local); difference != "" {
		t.Fatalf("maximum reply differs in %s: %s %s", difference, local.Termination, local.Message)
	}
	if len(local.Transcript) < bufferLimit {
		t.Fatal("maximum byte reply was not observed")
	}
}

func TestBrokerCrashRetainsInterruptedRuntimePrefix(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	requireSmallHelperParityModule(t, request)
	request.Config.Clock.ReadStepNanos = 0
	config := brokerTestConfig()
	config.Nodes = config.Nodes[:1]
	broker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	request.Broker = broker.Session("a")
	if err := request.Broker.CrashOnOutput([]byte("timer")); err != nil {
		t.Fatal(err)
	}
	crashed, err := Run(t.Context(), request)
	if err != nil || crashed.Termination != "crash" || !crashed.Reaped || !crashed.RuntimeInterrupted || len(crashed.InterruptedChoiceBytes) == 0 || len(crashed.ChoiceTrace.Bytes) != 0 {
		t.Fatalf("modeled broker crash prefix missing or claimed complete: termination=%s reaped=%v interrupted=%v bytes=%d complete=%d err=%v message=%s", crashed.Termination, crashed.Reaped, crashed.RuntimeInterrupted, len(crashed.InterruptedChoiceBytes), len(crashed.ChoiceTrace.Bytes), err, crashed.Message)
	}
	tape, err := choice.ProjectInterruptedReplayPlan(crashed.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if crashed.Telemetry == nil || !crashed.Telemetry.ResourceUsageAvailable || crashed.Telemetry.ExecuteNanos == 0 || crashed.AuthorizationToTerminalNanos == 0 || crashed.PeakMemoryBytes == 0 || crashed.FuelRemaining == request.Fuel || crashed.HelperResourceUsage == nil {
		t.Fatalf("modeled broker interruption lost actual helper costs: metrics=%+v wait=%+v execution=%d peak=%d fuel=%d", crashed.Telemetry, crashed.HelperResourceUsage, crashed.AuthorizationToTerminalNanos, crashed.PeakMemoryBytes, crashed.FuelRemaining)
	}
	replayBroker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	request.Broker = replayBroker.Session("a")
	if err := request.Broker.CrashOnOutput([]byte("timer")); err != nil {
		t.Fatal(err)
	}
	control, choices := *request.Runtime, *request.Runtime.Choice
	choices.Mode, choices.Tape = choice.ModePrefix, &tape
	control.Choice, request.Runtime = &choices, &control
	replayed, err := Run(t.Context(), request)
	if err != nil || replayed.Termination != "crash" || !replayed.Reaped || !replayed.RuntimeInterrupted || len(replayed.ChoiceTrace.Bytes) != 0 || replayed.Message != crashed.Message || !bytes.Equal(replayed.InterruptedChoiceBytes, crashed.InterruptedChoiceBytes) || !bytes.Equal(replayed.InterruptedDiagnosticBytes, crashed.InterruptedDiagnosticBytes) || !bytes.Equal(replayed.Transcript, crashed.Transcript) || !bytes.Equal(replayed.NetworkTranscript, crashed.NetworkTranscript) || !bytes.Equal(replayed.Stdout, crashed.Stdout) || !bytes.Equal(replayed.Stderr, crashed.Stderr) || replayed.EnvironmentIdentity != crashed.EnvironmentIdentity {
		t.Fatalf("fresh forced interrupted prefix differs: termination=%s interrupted=%v err=%v message=%s", replayed.Termination, replayed.RuntimeInterrupted, err, replayed.Message)
	}
	t.Logf("actual fresh interrupted-prefix replay: choices=%d transcript=%x diagnostics=%x", len(crashed.InterruptedChoiceBytes), sha256.Sum256(crashed.Transcript), sha256.Sum256(crashed.InterruptedDiagnosticBytes))
}

func requireSmallHelperParityModule(t *testing.T, request Request) {
	t.Helper()
	info, err := os.Stat(request.ModulePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 8<<20 {
		t.Fatalf("short parity gate unexpectedly prepared large module: %d bytes", info.Size())
	}
	t.Logf("small parity module=%s bytes=%d helper=%s runtime=%s model=%s", request.ModuleSHA256, info.Size(), request.HelperSHA256, request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, ImplementationSHA256())
}
