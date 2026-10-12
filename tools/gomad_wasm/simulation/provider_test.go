package simulation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio/volume"
	"go.temporal.io/server/tools/gomad3/record"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
	netmodel "go.temporal.io/server/tools/gomad3/simulation/network"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
	wasmbackend "go.temporal.io/server/tools/gomad_wasm/backend"
	"go.temporal.io/server/tools/gomad_wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad_wasm/toolchain"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

func TestProviderMandatoryCleanupReapsWithoutLifecycleActions(t *testing.T) {
	base := guestRequest(t)
	base.Config.Args = []string{"service", "durable"}
	for _, mode := range []string{"scenario-error", "action-capacity", "context-cancellation"} {
		t.Run(mode, func(t *testing.T) {
			request := base
			writer := &markerWriter{marker: []byte("fresh heap 1"), seen: make(chan struct{})}
			peer := request
			peer.StdoutMirror = writer
			identity, err := BootConfiguration(request)
			if err != nil {
				t.Fatal(err)
			}
			spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: 7, Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "a", Address: "10.0.0.1", Config: identity}, {ID: "b", Boot: "b", Address: "10.0.0.2", Config: identity}}}
			if mode == "action-capacity" {
				spec.Limits.ScenarioActions = 2
			}
			p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"a": request, "b": peer}, OutputStops: []OutputCrash{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 1}, Marker: []byte("unreachable marker")}}}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			result, runErr := gomad3sim.RunWASM(ctx, spec, func(ctx context.Context, c gomad3sim.Cluster) error {
				a, err := c.Start(ctx, "a")
				if err != nil {
					return err
				}
				if _, err := c.Start(ctx, "b"); err != nil {
					return err
				}
				select {
				case <-writer.seen:
				case <-ctx.Done():
					return ctx.Err()
				}
				if mode == "action-capacity" {
					return c.Crash(ctx, a)
				}
				if mode == "context-cancellation" {
					cancel()
					return ctx.Err()
				}
				return errors.New("actual scenario failure")
			}, p)
			if runErr != nil && mode != "context-cancellation" {
				for _, actual := range p.CompletedResults() {
					t.Logf("cleanup terminal %s/%d: %s %s reaped=%v interrupted=%v cancelled=%v", actual.Handle.Node, actual.Handle.Incarnation, actual.Result.Termination, actual.Result.Message, actual.Result.Reaped, actual.Result.RuntimeInterrupted, actual.Result.Cancelled)
				}
				t.Fatal(runErr)
			}
			if runErr == nil && (result.Outcome != gomad3sim.OutcomeScenarioFailed || len(result.Record.Transitions) != 2) {
				t.Fatal("cleanup fabricated lifecycle actions or suppressed actual failure")
			}
			results := p.CompletedResults()
			if len(results) != 2 {
				t.Fatal("cleanup lost helper results")
			}
			for _, actual := range results {
				if !actual.Result.Reaped || mode != "context-cancellation" && !actual.Result.RuntimeInterrupted {
					t.Fatalf("cleanup did not reap actual interrupted helper %s/%d: %s %s", actual.Handle.Node, actual.Handle.Incarnation, actual.Result.Termination, actual.Result.Message)
				}
			}
		})
	}
}

func guestRequest(t *testing.T) wasi.Request {
	t.Helper()
	if !filepath.IsAbs(os.Getenv("GOMAD3_STOCK_GO")) {
		t.Fatal("GOMAD3_STOCK_GO must name stock Go 1.27.1")
	}
	root := stockgo.Root(t)
	native, err := filepath.Abs("../../gomad3/runtimesemantics")
	if err != nil {
		t.Fatal(err)
	}
	overlay, key, err := toolchain.BuildOverlay(root, native, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "service.wasm")
	cmd := exec.Command(filepath.Join(root, "bin/go"), "build", "-trimpath", "-overlay", overlay, "-o", module, "./testdata/service")
	cmd.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile real guest %v %s", err, output)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 8<<20 {
		t.Fatalf("small guest admission rejected module bytes=%d maximum=%d before helper dispatch", len(data), 8<<20)
	}
	imports, pages, err := wasmbackend.InspectModule(data)
	if err != nil {
		t.Fatal(err)
	}
	helper, err := filepath.Abs("../wasmhost/target/release/gomad3-wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	helperBytes, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual small guest module-sha256=%x module-bytes=%d helper-sha256=%x implementation=%s", sha256.Sum256(data), len(data), sha256.Sum256(helperBytes), ImplementationSHA256())
	config := wasi.Config{Profile: wasi.CooperativeProfile, Args: []string{"service", "server"}, Environment: []string{"PWD=/workspace", "TZ=UTC"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, EntropyKey: [32]byte{1, 2, 3}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 32 << 20, Calls: 100000, Descriptors: 128, Files: 1024, FilesystemBytes: 8 << 20, PendingEvents: 4096}, NetworkV2: &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Deliveries: 1024, Bytes: 8 << 20, Transitions: 10000}}}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: key, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(key))}
	return wasi.Request{HelperPath: helper, HelperSHA256: fmt.Sprintf("%x", sha256.Sum256(helperBytes)), ModulePath: module, ModuleSHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Engine: wasi.PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4_000_000_000, Config: config, Timeout: time.Minute, Runtime: &wasi.RuntimeControl{Seed: 7, Diagnostics: true, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8 << 20}}}
}

func logIncarnationSummaries(t *testing.T, p *Provider, phase string) {
	t.Helper()
	data, err := json.Marshal(p.CompletedSummaries())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual detached terminal/cost summaries phase=%s %s", phase, data)
}

type markerWriter struct {
	mu     sync.Mutex
	bytes  []byte
	marker []byte
	seen   chan struct{}
	once   sync.Once
}

func (w *markerWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.bytes = append(w.bytes, data...)
	if bytes.Contains(w.bytes, w.marker) {
		w.once.Do(func() { close(w.seen) })
	}
	return len(data), nil
}

func TestProviderStopReapsGuestAndPeerContinues(t *testing.T) {
	first := guestRequest(t)
	first.Config.Args = []string{"service", "durable"}
	peer := first
	peer.Config.Args = []string{"service", "park"}
	a, err := BootConfiguration(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := BootConfiguration(peer)
	if err != nil {
		t.Fatal(err)
	}
	spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: 7, Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "durable", Address: "10.0.0.1", Config: a}, {ID: "b", Boot: "park", Address: "10.0.0.2", Config: b}}}
	p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"durable": first, "park": peer}, OutputStops: []OutputCrash{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 1}, Marker: []byte("committed durable-a")}}}
	observed, err := gomad3sim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomad3sim.Cluster) error {
		a, err := c.Start(ctx, "a")
		if err != nil {
			return err
		}
		b, err := c.Start(ctx, "b")
		if err != nil {
			return err
		}
		if err := c.Stop(ctx, a); err != nil {
			return err
		}
		_, err = c.Wait(ctx, b)
		return err
	}, p)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Outcome != gomad3sim.OutcomeCompleted || observed.Record.Nodes[0].State != gomad3sim.NodeStateStopped || !observed.WASM.Incarnations[0].Interrupted {
		t.Fatalf("typed stopped guest lost modeled interruption: %+v", observed.Record.Nodes)
	}
	payload, err := p.RetentionPayload(spec, observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(first.ModulePath, first.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD3_STOCK_GO", "/original/compiler/unavailable")
	replayed, results, err := ReplayRetainedResults(t.Context(), payload, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	left, err := gomad3sim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	right, err := gomad3sim.EncodeClusterRecord(replayed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) || len(results) != 2 || results[0].WallElapsedNanos == 0 || !results[0].Result.RuntimeInterrupted || !results[0].Result.Reaped {
		t.Fatal("typed stopped guest did not source-lessly replay its sealed boundary")
	}
}

func TestProviderGroupedAcknowledgmentCrashSourceLessReplay(t *testing.T) {
	base := guestRequest(t)
	boots := map[gomad3sim.BootID]wasi.Request{}
	spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: 7, Limits: gomad3sim.DefaultLimits()}
	var bindings []IncarnationBoot
	for i, id := range []gomad3sim.NodeID{"a", "b"} {
		r := base
		r.Config.Args = []string{"service", "durable"}
		r.Config.Storage = &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: string(id), Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
		commit, recover := gomad3sim.BootID(string(id)+"-commit"), gomad3sim.BootID(string(id)+"-recover")
		boots[commit] = r
		identity, err := BootConfiguration(r)
		if err != nil {
			t.Fatal(err)
		}
		spec.Nodes = append(spec.Nodes, gomad3sim.NodeSpec{ID: id, Boot: commit, Address: fmt.Sprintf("10.0.0.%d", i+1), Config: identity, Volumes: []gomad3sim.VolumeMount{{Volume: gomad3sim.VolumeID(id), Path: "/workspace"}}})
		spec.Volumes = append(spec.Volumes, gomad3sim.VolumeSpec{ID: gomad3sim.VolumeID(id), CapacityBytes: 8 << 20})
		r.Config.Args = []string{"service", "recover"}
		boots[recover] = r
		bindings = append(bindings, IncarnationBoot{Handle: gomad3sim.NodeHandle{Node: id, Incarnation: 2}, Boot: recover})
	}
	group := OutputCrashGroup{Handles: []gomad3sim.NodeHandle{{Node: "a", Incarnation: 1}, {Node: "b", Incarnation: 1}}, Marker: []byte("committed durable-a")}
	p := &Provider{Boots: boots, IncarnationBoots: bindings, OutputCrashGroups: []OutputCrashGroup{group}}
	observed, err := gomad3sim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomad3sim.Cluster) error {
		for _, id := range []gomad3sim.NodeID{"a", "b"} {
			if _, err := c.Start(ctx, id); err != nil {
				return err
			}
		}
		for _, handle := range group.Handles {
			if err := c.Crash(ctx, handle); err != nil {
				return err
			}
		}
		for _, id := range []gomad3sim.NodeID{"a", "b"} {
			if _, err := c.Restart(ctx, id); err != nil {
				return err
			}
		}
		for _, binding := range bindings {
			if _, err := c.Wait(ctx, binding.Handle); err != nil {
				return err
			}
		}
		return nil
	}, p)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Outcome != gomad3sim.OutcomeCompleted || len(observed.WASM.Incarnations) != 4 || len(observed.Record.Volumes.Snapshot.Volumes) != 2 {
		t.Fatal("independent grouped crash/recovery did not complete")
	}
	logIncarnationSummaries(t, p, "grouped-record")
	for _, result := range p.CompletedResults() {
		if !result.Result.Reaped {
			t.Fatal("grouped helper not reaped")
		}
		if result.Handle.Incarnation == 1 && !bytes.Contains(result.Result.Stdout, group.Marker) {
			t.Fatal("group crash preceded a required acknowledgment")
		}
		if result.Handle.Incarnation == 2 && !bytes.Contains(result.Result.Stdout, []byte("fresh heap 1\nrecovered durable-a")) {
			t.Fatal("independent recovered incarnation reused heap or lost volume")
		}
	}
	payload, err := p.RetentionPayload(spec, observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(base.ModulePath, base.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD3_STOCK_GO", "/original/compiler/unavailable")
	replayed, original, completed, err := ReplayBoundRetained(t.Context(), payload, "sha256:"+base.ModuleSHA256, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if completed == nil {
		t.Fatal("bound grouped replay lost completed observations")
	}
	logIncarnationSummaries(t, completed, "grouped-replay")
	left, err := gomad3sim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	right, err := gomad3sim.EncodeClusterRecord(replayed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) || !bytes.Equal(left, original) {
		t.Fatal("grouped acknowledgment crash changed sealed replay")
	}
}

func TestProviderActualCrashDurableRestart(t *testing.T) {
	request := guestRequest(t)
	request.Config.Args = []string{"service", "durable"}
	request.Config.Storage = &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: "a-store", Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
	peer := request
	peer.Config = request.Config
	peer.Config.Args = []string{"service", "park"}
	peer.Config.Storage = nil
	recovery := request
	recovery.Config.Args = []string{"service", "recover"}
	a, err := BootConfiguration(request)
	if err != nil {
		t.Fatal(err)
	}
	b, err := BootConfiguration(peer)
	if err != nil {
		t.Fatal(err)
	}
	spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: 7, Limits: gomad3sim.DefaultLimits(), Volumes: []gomad3sim.VolumeSpec{{ID: "a-store", CapacityBytes: 8 << 20}}, Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "durable", Address: "10.0.0.1", Config: a, Volumes: []gomad3sim.VolumeMount{{Volume: "a-store", Path: "/workspace"}}}, {ID: "b", Boot: "park", Address: "10.0.0.2", Config: b}}}
	var recordedProvider *Provider
	run := func() (gomad3sim.Result, error) {
		writer := &markerWriter{marker: []byte("committed durable-a"), seen: make(chan struct{})}
		first := request
		first.StdoutMirror = writer
		p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"durable": first, "park": peer, "recover": recovery}, IncarnationBoots: []IncarnationBoot{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 2}, Boot: "recover"}}, OutputCrashes: []OutputCrash{{Handle: gomad3sim.NodeHandle{Node: "a", Incarnation: 1}, Marker: []byte("committed durable-a")}}}
		recordedProvider = p
		return gomad3sim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomad3sim.Cluster) error {
			a, err := c.Start(ctx, "a")
			if err != nil {
				return err
			}
			select {
			case <-writer.seen:
			case <-ctx.Done():
				return ctx.Err()
			}
			if err = c.Crash(ctx, a); err != nil {
				return err
			}
			a, err = c.Restart(ctx, "a")
			if err != nil {
				return err
			}
			b, err := c.Start(ctx, "b")
			if err != nil {
				return err
			}
			if _, err = c.Wait(ctx, a); err != nil {
				return err
			}
			_, err = c.Wait(ctx, b)
			return err
		}, p)
	}
	observed, err := run()
	if err != nil {
		t.Fatal(err)
	}
	if observed.Outcome != gomad3sim.OutcomeCompleted || len(observed.Record.Volumes.Snapshot.Volumes) != 1 {
		t.Fatalf("durable isolated crash result %+v", observed)
	}
	found := false
	for _, output := range observed.Outputs {
		if output.Handle.Node == "a" && output.Handle.Incarnation == 2 && bytes.Contains(output.Bytes, []byte("fresh heap 1\nrecovered durable-a")) {
			found = true
		}
	}
	if !found {
		t.Fatalf("fresh durable recovery output %+v", observed.Outputs)
	}
	payload, err := recordedProvider.RetentionPayload(spec, observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(request.ModulePath, request.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD3_STOCK_GO", "/original/compiler/unavailable")
	replayed, err := ReplayRetained(t.Context(), payload, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	left, err := gomad3sim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	right, err := gomad3sim.EncodeClusterRecord(replayed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) {
		t.Fatal("actual hard-crash durable restart changed canonical replay record")
	}
}

func TestProviderIndependentGuestsLifecycleReplay(t *testing.T) {
	testProviderIndependentGuestsLifecycleReplay(t, "")
}

func TestProviderCompressedIndependentGuestsLifecycleReplay(t *testing.T) {
	testProviderIndependentGuestsLifecycleReplay(t, world.CompressedSnapshotSchema)
}

func testProviderIndependentGuestsLifecycleReplay(t *testing.T, profile string) {
	t.Helper()
	server := guestRequest(t)
	client := server
	client.Config = server.Config
	client.Config.Args = []string{"service", "client", "10.0.0.1"}
	a, err := BootConfiguration(server)
	if err != nil {
		t.Fatal(err)
	}
	b, err := BootConfiguration(client)
	if err != nil {
		t.Fatal(err)
	}
	spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: 7, Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "server", Address: "10.0.0.1", Config: a}, {ID: "b", Boot: "client", Address: "10.0.0.2", Config: b}}}
	spec.Links = []gomad3sim.LinkSpec{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}}
	p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"server": server, "client": client}}
	if profile != "" {
		p.Broker.WorldProfile = profile
		p.Broker.Limits = world.CompressedSnapshotProfile().Limits
	}
	scenario := func(ctx context.Context, c gomad3sim.Cluster) error {
		a, err := c.Start(ctx, "a")
		if err != nil {
			return err
		}
		b, err := c.Start(ctx, "b")
		if err != nil {
			return err
		}
		if _, err = c.Wait(ctx, b); err != nil {
			return err
		}
		if _, err = c.Wait(ctx, a); err != nil {
			return err
		}
		a, err = c.Restart(ctx, "a")
		if err != nil {
			return err
		}
		b, err = c.Restart(ctx, "b")
		if err != nil {
			return err
		}
		if _, err = c.Wait(ctx, b); err != nil {
			return err
		}
		_, err = c.Wait(ctx, a)
		return err
	}
	observed, err := gomad3sim.RunWASM(t.Context(), spec, scenario, p)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Outcome != gomad3sim.OutcomeCompleted || len(observed.Record.Transitions) != 8 {
		t.Fatalf("isolated lifecycle result %+v", observed)
	}
	if observed.Record.WASM.WorldEncoding != profile {
		t.Fatal("explicit World profile was not retained")
	}
	logIncarnationSummaries(t, p, "lifecycle-record")
	for _, output := range observed.Outputs {
		if output.Stream == gomad3sim.OutputStdout && (!bytes.Contains(output.Bytes, []byte("fresh heap 1")) || !bytes.Contains(output.Bytes, []byte("ordered bytes ordered-byte"))) {
			t.Fatalf("incarnation %v reused heap or changed stream: %s", output.Handle, output.Bytes)
		}
	}
	payload, err := p.RetentionPayload(spec, observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*retainedExecution)
	}{
		{"missing-action", func(b *retainedExecution) { b.Lifecycle = b.Lifecycle[:len(b.Lifecycle)-1] }},
		{"mutated-action", func(b *retainedExecution) { b.Lifecycle[0].Handle.Incarnation++ }},
		{"missing-module", func(b *retainedExecution) { delete(b.Inputs, b.Boots[0].Request.ModuleSHA256) }},
		{"changed-limits", func(b *retainedExecution) { b.Spec.Limits.NetworkTransitions++ }},
		{"changed-environment", func(b *retainedExecution) { b.Boots[0].Request.Config.Clock.EpochNanos++ }},
		{"changed-helper-budget", func(b *retainedExecution) { b.Boots[0].Request.Timeout++ }},
		{"changed-choice-budget", func(b *retainedExecution) { b.Boots[0].Request.Runtime.Choice.Limit++ }},
		{"malformed-storage-selection", func(b *retainedExecution) {
			b.Boots[0].Request.StorageCrashSelection = map[string][]uint64{"unknown": {0, 0}}
		}},
		{"changed-imports", func(b *retainedExecution) { b.Boots[0].Request.Imports[0].Name += "-changed" }},
		{"missing-model-implementation", func(b *retainedExecution) { b.Implementation = "" }},
		{"changed-model-implementation", func(b *retainedExecution) { b.Implementation = "sha256:" + strings.Repeat("0", 64) }},
		{"changed-world-profile", func(b *retainedExecution) { b.Broker.WorldProfile = "unknown" }},
		{"changed-world-encoding", func(b *retainedExecution) {
			var observed gomad3sim.ClusterRecord
			if err := json.Unmarshal(b.Record, &observed); err != nil {
				t.Fatal(err)
			}
			observed.WASM.WorldEncoding = "unknown"
			var err error
			b.Record, err = json.Marshal(observed)
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"raw-noncanonical-world", func(b *retainedExecution) {
			mutateRetainedWorld(t, b, func(data []byte) []byte {
				var snapshot world.Snapshot
				if err := json.Unmarshal(data, &snapshot); err != nil {
					t.Fatal(err)
				}
				encoded, err := json.Marshal(snapshot)
				if err != nil {
					t.Fatal(err)
				}
				return encoded
			})
		}},
		{"unknown-world-field", func(b *retainedExecution) {
			mutateRetainedWorld(t, b, func(data []byte) []byte {
				return append(append(bytes.Clone(data[:len(data)-1]), []byte(`,"unknown":true`)...), '}')
			})
		}},
		{"corrupt-world-state", func(b *retainedExecution) {
			mutateRetainedWorld(t, b, func(data []byte) []byte {
				var snapshot map[string]json.RawMessage
				if err := json.Unmarshal(data, &snapshot); err != nil {
					t.Fatal(err)
				}
				snapshot["stateDigest"] = json.RawMessage(`"` + strings.Repeat("0", 64) + `"`)
				encoded, err := json.Marshal(snapshot)
				if err != nil {
					t.Fatal(err)
				}
				return encoded
			})
		}},
		{"changed-runtime-binding", func(b *retainedExecution) {
			b.Boots[0].Request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey += "-changed"
		}},
		{"changed-original-record", func(b *retainedExecution) { b.Record = append(b.Record, '\n') }},
		{"changed-root-identity", func(b *retainedExecution) {
			var observed gomad3sim.ClusterRecord
			if err := json.Unmarshal(b.Record, &observed); err != nil {
				t.Fatal(err)
			}
			observed.Identity = "sha256:" + strings.Repeat("0", 64)
			var err error
			b.Record, err = json.Marshal(observed)
			if err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var changed retainedExecution
			if err := json.Unmarshal(payload.Data, &changed); err != nil {
				t.Fatal(err)
			}
			test.mutate(&changed)
			data, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			bad := payload
			bad.Data = data
			bad.Reference.SHA256 = record.HashBytes(data)
			bad.Reference.Bytes = record.Uint64String(len(data))
			directory := t.TempDir()
			if _, _, err := VerifyRetained(bad); err == nil {
				t.Fatal("verify-only admitted mutated retained inputs")
			}
			if _, err := RetainedStorageCrashSelections(bad); err == nil {
				t.Fatal("selection projection admitted mutated retained inputs")
			}
			if _, _, err := ReopenProvider(bad, directory); err == nil {
				t.Fatal("mutated retained inputs were admitted")
			}
			if _, _, _, err := ReplayBoundRetained(t.Context(), bad, "sha256:"+server.ModuleSHA256, directory); err == nil {
				t.Fatal("bound replay admitted mutated retained inputs")
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 0 {
				t.Fatalf("rejection wrote helper inputs before validation: %v %v", entries, err)
			}
		})
	}
	for _, primary := range []string{"", "sha256:" + strings.Repeat("0", 64)} {
		directory := t.TempDir()
		if _, _, _, err := ReplayBoundRetained(t.Context(), payload, primary, directory); err == nil {
			t.Fatal("bound replay accepted foreign or missing Artifact primary")
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 0 {
			t.Fatalf("unbound primary wrote executable inputs: %v %v", entries, err)
		}
	}
	if err := os.Rename(server.ModulePath, server.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	// Reopening receives no source or stock toolchain. Executable inputs are
	// copied from the existing Artifact payload into fresh private files.
	t.Setenv("GOMAD3_STOCK_GO", "/original/compiler/unavailable")
	replayed, original, completed, err := ReplayBoundRetained(t.Context(), payload, "sha256:"+server.ModuleSHA256, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := decodeRetained(payload)
	if err != nil || !bytes.Equal(original, bundle.Record) || completed == nil || len(completed.CompletedSummaries()) != 4 {
		t.Fatalf("bound replay lost original canonical bytes or terminal summaries: %v", err)
	}
	logIncarnationSummaries(t, completed, "lifecycle-replay")
	left, err := gomad3sim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	right, err := gomad3sim.EncodeClusterRecord(replayed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) {
		t.Fatal("actual WASM lifecycle replay changed canonical record")
	}
	called := false
	if _, err = gomad3sim.Run(t.Context(), spec, func(context.Context, gomad3sim.Cluster) error { called = true; return nil }); err == nil || called {
		t.Fatal("WASM without an injected engine did not fail closed")
	}
}

func mutateRetainedWorld(t *testing.T, bundle *retainedExecution, mutate func([]byte) []byte) {
	t.Helper()
	var observed gomad3sim.ClusterRecord
	if err := json.Unmarshal(bundle.Record, &observed); err != nil {
		t.Fatal(err)
	}
	observed.WASM.World = mutate(observed.WASM.World)
	encoded, err := json.Marshal(observed)
	if err != nil {
		t.Fatal(err)
	}
	bundle.Record = encoded
}
