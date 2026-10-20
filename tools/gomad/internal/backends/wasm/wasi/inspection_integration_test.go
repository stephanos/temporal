package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func inspectionFixtureRequest(t *testing.T) Request {
	t.Helper()
	root := stockgo.Root(t)
	source, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	overlay, identity, err := toolchain.BuildOverlay(root, source, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "inspection.wasm")
	build := exec.Command(filepath.Join(root, "bin/go"), "build", "-p=1", "-trimpath", "-buildvcs=false", "-tags=gomad", "-overlay", overlay, "-o", module, "go.temporal.io/server/common/gomaddebug/testdata/fixture")
	build.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOMAXPROCS=1")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build actual inspection guest: %v\n%s", err, output)
	}
	request := cooperativeModuleRequest(t, module, identity)
	request.Runtime.Diagnostics = false
	return request
}

// Keep this raw control builder at the transport boundary: importing the cluster
// backend here would create a dependency cycle through its WASI client.
func inspectionFixtureControl(t *testing.T, evidence *cluster.WASMInspectionEvidence, boundary cluster.WASMInspectionBoundary, names []string, probes []uint32) []byte {
	t.Helper()
	commands := []map[string]uint32{}
	for _, name := range names {
		kind, id := uint32(0), uint32(0)
		switch name {
		case "stacks":
			kind = 1
		case "heap":
			kind = 2
		default:
			for _, registration := range evidence.Incarnations[0].Registry.Entries {
				if registration.Name == name {
					kind, id = 3, registration.ID
				}
			}
		}
		if kind == 0 {
			t.Fatalf("unknown fixture dump %q", name)
		}
		commands = append(commands, map[string]uint32{"kind": kind, "id": id})
	}
	digest := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	streams := []map[string]string{}
	registries := []cluster.WASMInspectionRegistry{}
	for _, cut := range boundary.Cuts {
		for _, incarnation := range evidence.Incarnations {
			if incarnation.Handle != cut.Handle {
				continue
			}
			s := incarnation.Streams
			streams = append(streams, map[string]string{"choice": digest(s.Choice[:cut.ChoiceBytes]), "wasi": digest(s.WASI[:cut.WASIBytes]), "network": digest(s.Network[:cut.NetworkBytes]), "storage": digest(s.Storage[:cut.StorageBytes]), "stdout": digest(s.Stdout[:cut.StdoutBytes]), "stderr": digest(s.Stderr[:cut.StderrBytes])})
			registry := cluster.WASMInspectionRegistry{Entries: []cluster.WASMInspectionRegistration{}}
			registry.Closed = incarnation.Registry.Closed && incarnation.Registry.ClosedOrder <= boundary.Order
			if registry.Closed {
				registry.ClosedOrder = incarnation.Registry.ClosedOrder
			}
			for _, registration := range incarnation.Registry.Entries {
				if registration.AvailableOrder <= boundary.Order {
					registry.Entries = append(registry.Entries, registration)
				}
			}
			registries = append(registries, registry)
		}
	}
	if probes == nil {
		probes = []uint32{}
	}
	control, err := json.Marshal(map[string]any{"target": boundary.Handle, "decision": boundary.Decision, "expected": boundary, "world_sha256": digest(evidence.World[:boundary.WorldBytes]), "lifecycle_sha256": digest(evidence.Lifecycle[:boundary.LifecycleBytes]), "stream_sha256": streams, "probes": probes, "commands": commands, "registries": registries})
	if err != nil {
		t.Fatal(err)
	}
	return control
}

func TestCooperativeInspectionEveryDecision(t *testing.T) {
	request := inspectionFixtureRequest(t)
	recorded, err := Run(context.Background(), request)
	if err != nil || recorded.Termination != "exit" || recorded.ExitCode == nil || *recorded.ExitCode != 1 || !recorded.Reaped {
		t.Fatalf("record fixture: %s %s err=%v stderr=%s", recorded.Termination, recorded.Message, err, recorded.Stderr)
	}
	if len(recorded.DebugOutput) != 0 || len(recorded.DiagnosticTrace.Records) != 0 {
		t.Fatal("recording enabled debug work or optional diagnostics")
	}
	evidence, err := cluster.DecodeWASMInspection(recorded.InspectionMetadata)
	if err != nil {
		t.Fatalf("record metadata: %v", err)
	}
	if !evidence.Complete || len(evidence.Boundaries) == 0 || !evidence.Incarnations[0].Registry.Closed {
		t.Fatalf("incomplete dormant recording: %+v", evidence)
	}
	tape, err := choice.ProjectReplayPlan(recorded.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	setChoiceReplay(t, request.Runtime.Choice, choice.ModeReplay, recorded.ChoiceTrace.Bytes, tape)
	replayed, err := Run(context.Background(), request)
	if err != nil || replayed.Termination != "exit" || !bytes.Equal(recorded.ChoiceTrace.Bytes, replayed.ChoiceTrace.Bytes) || !bytes.Equal(recorded.Transcript, replayed.Transcript) || !bytes.Equal(recorded.InspectionMetadata, replayed.InspectionMetadata) {
		t.Fatalf("ordinary exact replay: %s %s err=%v", replayed.Termination, replayed.Message, err)
	}
	seen := map[choice.Kind]int{}
	observationOffset := false
	for ordinal, boundary := range evidence.Boundaries {
		record := recorded.ChoiceTrace.Records[boundary.Trace]
		if boundary.Decision != uint64(ordinal) || record.Flags&choice.FlagDecision == 0 {
			t.Fatalf("tape ordinal confused with trace observation: %+v %+v", boundary, record)
		}
		observationOffset = observationOffset || boundary.Trace > boundary.Decision
		seen[record.Kind]++
		t.Run(fmt.Sprintf("kind%d-%d", record.Kind, boundary.Decision), func(t *testing.T) {
			request.Runtime.Inspection = inspectionFixtureControl(t, evidence, boundary, []string{"stacks", "heap"}, nil)
			result, err := Run(t.Context(), request)
			if err != nil || result.Termination != "inspected_prefix" || !result.Reaped {
				t.Fatalf("stop: %s %s err=%v stderr=%s", result.Termination, result.Message, err, result.Stderr)
			}
			observed, err := cluster.DecodeWASMInspection(result.InspectionMetadata)
			if err != nil {
				t.Fatal(err)
			}
			if err := cluster.VerifyWASMInspectionPrefix(evidence, observed, boundary); err != nil {
				t.Fatal(err)
			}
			output, err := DecodeDebugOutput(result.DebugOutput)
			if err != nil || len(output.Stacks) == 0 || len(output.HeapProfile) == 0 {
				t.Fatalf("built-in dumps: stacks=%d heap=%d err=%v", len(output.Stacks), len(output.HeapProfile), err)
			}
			if boundary.Decision == 0 {
				profile := filepath.Join(t.TempDir(), "heap.pprof")
				if err := os.WriteFile(profile, output.HeapProfile, 0600); err != nil {
					t.Fatal(err)
				}
				command := exec.Command(filepath.Join(stockgo.Root(t), "bin/go"), "tool", "pprof", "-top", profile)
				if out, err := command.CombinedOutput(); err != nil {
					t.Fatalf("actual guest heap pprof: %v\n%s", err, out)
				}
			}
		})
	}
	if seen[choice.KindRunnable] == 0 || seen[choice.KindSelectPoll] == 0 || !observationOffset {
		t.Fatalf("fixture omitted decision kinds: %v", seen)
	}
	last := evidence.Boundaries[len(evidence.Boundaries)-1]
	t.Run("registered-dumps-frozen-GC", func(t *testing.T) {
		request.Runtime.Inspection = inspectionFixtureControl(t, evidence, last, []string{"fixture.snapshot", "fixture.panic", "fixture.truncated", "fixture.gc"}, nil)
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination != "inspected_prefix" || !result.Reaped {
			t.Fatalf("registered dumps: %s %s err=%v stderr=%s", result.Termination, result.Message, err, result.Stderr)
		}
		observed, err := cluster.DecodeWASMInspection(result.InspectionMetadata)
		if err != nil {
			t.Fatal(err)
		}
		if err := cluster.VerifyWASMInspectionPrefix(evidence, observed, last); err != nil {
			t.Fatal(err)
		}
		output, err := DecodeDebugOutput(result.DebugOutput)
		if err != nil || len(output.Dumps) != 4 {
			t.Fatalf("registered output: dumps=%d err=%v", len(output.Dumps), err)
		}
		if output.Dumps[0].Status != "ok" || !bytes.Contains(output.Dumps[0].Data, []byte("application lock is held")) || !bytes.Contains(output.Dumps[0].Data, []byte("progress observed")) || output.Dumps[1].Status != "panic" || output.Dumps[2].Status != "truncated" || output.Dumps[3].Status != "ok" || !bytes.Contains(output.Dumps[3].Data, []byte("collection completed")) {
			t.Fatalf("dump isolation/GC: %+v", output.Dumps)
		}
	})
	probeID := func(name string) uint32 {
		for _, registration := range evidence.Incarnations[0].Registry.Entries {
			if registration.Kind == "probe" && registration.Name == name {
				return registration.ID
			}
		}
		t.Fatalf("missing probe %s", name)
		return 0
	}
	t.Run("allocation-free-probes", func(t *testing.T) {
		request.Runtime.Inspection = inspectionFixtureControl(t, evidence, last, nil, []uint32{probeID("fixture.progress"), probeID("fixture.oversized")})
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination != "inspected_prefix" {
			t.Fatalf("enabled probes: %s %s err=%v", result.Termination, result.Message, err)
		}
		observed, err := cluster.DecodeWASMInspection(result.InspectionMetadata)
		if err != nil {
			t.Fatal(err)
		}
		if err := cluster.VerifyWASMInspectionPrefix(evidence, observed, last); err != nil {
			t.Fatal(err)
		}
		output, err := DecodeDebugOutput(result.DebugOutput)
		if err != nil || len(output.Probes) != 8 || len(output.Drops) != 8 {
			t.Fatalf("probe/drop output: probes=%d drops=%d err=%v", len(output.Probes), len(output.Drops), err)
		}
		for i, probe := range output.Probes {
			if len(probe.Fields) != 3 || probe.Fields[0].Uint64 != uint64(i+1) || probe.Decision > last.Decision || probe.Trace > last.Trace {
				t.Fatalf("typed probe or coordinates: %+v", probe)
			}
		}
	})
	t.Run("allocating-probe-diverges-without-diagnostics", func(t *testing.T) {
		request.Runtime.Inspection = inspectionFixtureControl(t, evidence, last, nil, []uint32{probeID("fixture.allocating")})
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination != "instrumentation_divergence" || !result.Reaped || !bytes.Equal(result.Stdout, recorded.Stdout) || len(result.DiagnosticTrace.Records) != 0 {
			t.Fatalf("allocating probe witness: %s %s err=%v stdout=%s", result.Termination, result.Message, err, result.Stdout)
		}
		if len(result.DebugOutput) != 0 {
			t.Fatal("allocating probe reached dump execution")
		}
		observed, err := cluster.DecodeWASMInspection(result.InspectionMetadata)
		if err != nil {
			t.Fatal(err)
		}
		if len(observed.Boundaries) == 0 {
			t.Fatal("allocating probe omitted the rejected boundary witness")
		}
		actual := observed.Boundaries[len(observed.Boundaries)-1].Cuts[0].Witness
		expected := last.Cuts[0].Witness
		if actual.GCCycles != expected.GCCycles || actual.AllocatedBytes <= expected.AllocatedBytes {
			t.Fatalf("allocating probe changed GC or omitted allocation charge: actual=%+v expected=%+v", actual, expected)
		}
	})
	t.Run("inspection-fuel-pressure", func(t *testing.T) {
		request.Runtime.Inspection = inspectionFixtureControl(t, evidence, last, []string{"fixture.timeout"}, nil)
		request.Fuel = request.Fuel - recorded.FuelRemaining + 1000000
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination != "probe_fuel_limit" || !result.Reaped || result.WatchdogTimeout || result.FuelRemaining != 0 {
			t.Fatalf("inspection fuel pressure: %s %s err=%v reaped=%v timeout=%v remaining=%d", result.Termination, result.Message, err, result.Reaped, result.WatchdogTimeout, result.FuelRemaining)
		}
	})
	t.Run("nonyielding-dump-reaped", func(t *testing.T) {
		request.Runtime.Inspection = inspectionFixtureControl(t, evidence, last, []string{"fixture.timeout"}, nil)
		request.Timeout = 10 * time.Second
		request.Fuel = 1 << 50
		result, err := Run(t.Context(), request)
		if err != nil || result.Termination == "inspected_prefix" || !result.Reaped || !result.WatchdogTimeout {
			t.Fatalf("nonyielding dump timeout: %s %s err=%v reaped=%v timeout=%v", result.Termination, result.Message, err, result.Reaped, result.WatchdogTimeout)
		}
	})
	t.Logf("actual diagnostics-off record/full replay/inspection: decisions=%d kinds=%v module=%s helper=%s", len(evidence.Boundaries), seen, request.ModuleSHA256, request.HelperSHA256)
}
