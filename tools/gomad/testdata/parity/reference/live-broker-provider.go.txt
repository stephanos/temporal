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

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/moduleinfo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

func TestProviderClusterStartsOnlyRequestedIncarnations(t *testing.T) {
	request := guestRequest(t)
	request.Config.Args = []string{"service", "observation"}
	identity, err := BootConfiguration(request)
	if err != nil {
		t.Fatal(err)
	}
	spec := gomadsim.Spec{Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendWASM, Seed: 7, Limits: gomadsim.DefaultLimits(), Nodes: []gomadsim.NodeSpec{{ID: "a", Boot: "guest", Address: "10.0.0.1", Config: identity}, {ID: "b", Boot: "guest", Address: "10.0.0.2", Config: identity}}}
	spec.Nodes[1].ID = gomadsim.NodeID(strings.Repeat("b", 128))
	provider := &Provider{Boots: map[gomadsim.BootID]wasi.Request{"guest": request}}
	hard := spec
	hard.Fidelity = gomadsim.FidelityHardIsolation
	if _, err := provider.Open(t.Context(), hard); err == nil || !strings.Contains(err.Error(), "hard isolation") {
		t.Fatalf("one-helper execution silently weakened hard isolation: %v", err)
	}
	reference := &Provider{Boots: provider.Boots, goBrokerReference: true}
	if _, err := reference.Open(t.Context(), hard); err != nil {
		t.Fatalf("explicit isolated Go reference was refused: %v", err)
	}
	opened, err := provider.Open(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	execution := opened.(*execution)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = opened.Shutdown(ctx)
	})
	if execution.cluster != nil || execution.clusterRequest == nil {
		t.Fatal("Open launched a helper before the first requested start")
	}
	handle := gomadsim.NodeHandle{Node: "a", Incarnation: 1}
	if err := opened.Start(spec.Nodes[0], handle); err != nil {
		t.Fatal(err)
	}
	terminal, err := opened.Wait(t.Context(), handle)
	if err != nil || terminal.Failed {
		t.Fatalf("started guest: %+v %v", terminal, err)
	}
	evidence, err := opened.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.Execution.Incarnations) != 1 || evidence.Execution.Incarnations[0].Handle != handle {
		t.Fatal("never-started guest produced execution evidence")
	}
	if frames := execution.cluster.FrameCounts(); frames.Results != 1 || frames.GuestCallbacks != 0 {
		t.Fatalf("cluster transport frames: %+v", frames)
	}
}

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
			spec := gomadsim.Spec{Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendWASM, Fidelity: gomadsim.FidelitySimulationModel, Seed: 7, Limits: gomadsim.DefaultLimits(), Nodes: []gomadsim.NodeSpec{{ID: "a", Boot: "a", Address: "10.0.0.1", Config: identity}, {ID: "b", Boot: "b", Address: "10.0.0.2", Config: identity}, {ID: "dormant", Boot: "a", Address: "10.0.0.3", Config: identity}}}
			if mode == "action-capacity" {
				spec.Limits.ScenarioActions = 2
			}
			p := &Provider{Boots: map[gomadsim.BootID]wasi.Request{"a": request, "b": peer}, OutputStops: []OutputCrash{{Handle: gomadsim.NodeHandle{Node: "a", Incarnation: 1}, Marker: []byte("unreachable marker")}}}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			result, runErr := gomadsim.RunWASM(ctx, spec, func(ctx context.Context, c gomadsim.Cluster) error {
				a, err := c.Start(ctx, "a")
				if err != nil {
					return err
				}
				b, err := c.Start(ctx, "b")
				if err != nil {
					return err
				}
				if err := awaitProviderMarker(ctx, p, b, writer.seen); err != nil {
					return err
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
			if runErr == nil && (result.Outcome != gomadsim.OutcomeScenarioFailed || len(result.Record.Transitions) != 2) {
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

func TestProviderCanceledSplitRevocationCancelsAndReaps(t *testing.T) {
	for _, action := range []string{"crash", "stop"} {
		t.Run(action, func(t *testing.T) {
			broker, err := wasi.NewBroker(wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "a", Incarnation: 1}}, Limits: world.Limits{MaxRequests: 10000, MaxEvents: 10000, MaxQueuedEvents: 1000, MaxTransitions: 40000, MaxPayloadBytes: 32 << 20, MaxStringBytes: 1024}})
			if err != nil {
				t.Fatal(err)
			}
			handle := gomadsim.NodeHandle{Node: "a", Incarnation: 1}
			n := &node{handle: handle, done: make(chan struct{}), request: wasi.Request{Config: wasi.Config{Profile: wasi.CooperativeProfile, Storage: &wasi.StorageModel{}}, Broker: broker.Session("a")}, result: wasi.Result{Termination: "infrastructure"}}
			var reaped sync.Once
			reap := func() { reaped.Do(func() { n.result.Reaped = true; close(n.done) }) }
			defer reap()
			cancelled := make(chan struct{})
			n.cancel = func() { close(cancelled); reap() }
			e := &execution{broker: broker, nodes: map[gomadsim.NodeID]*node{"a": n}}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			returned := make(chan error, 1)
			go func() {
				var err error
				if action == "crash" {
					_, err = e.Crash(ctx, handle)
				} else {
					_, err = e.Stop(ctx, handle)
				}
				returned <- err
			}()
			select {
			case err := <-returned:
				if !errors.Is(err, context.Canceled) {
					t.Errorf("canceled revocation returned %v", err)
				}
			case <-time.After(time.Second):
				t.Error("canceled revocation waited for the 30-second grace")
				reap()
				<-returned
			}
			select {
			case <-cancelled:
			default:
				t.Error("canceled revocation did not cancel the helper")
			}
			if err := e.Shutdown(t.Context()); err != nil || !n.result.Reaped {
				t.Fatalf("mandatory cleanup lost helper ownership: %v", err)
			}
		})
	}
}

func TestProviderBindsEverySelectedBootNetworkLimit(t *testing.T) {
	for _, test := range []struct {
		name            string
		mismatch        int
		stallMismatch   bool
		networkFree     bool
		restartMismatch bool
		unusedMismatch  bool
		brokerOverride  bool
		brokerMismatch  bool
	}{
		{name: "matching", mismatch: -1},
		{name: "network free", mismatch: -1, networkFree: true},
		{name: "first byte mismatch", mismatch: 0},
		{name: "later byte mismatch", mismatch: 1},
		{name: "first stall mismatch", mismatch: 0, stallMismatch: true},
		{name: "later stall mismatch", mismatch: 1, stallMismatch: true},
		{name: "restart byte mismatch", mismatch: -1, restartMismatch: true},
		{name: "restart stall mismatch", mismatch: -1, stallMismatch: true, restartMismatch: true},
		{name: "unused mismatch", mismatch: -1, unusedMismatch: true},
		{name: "matching explicit broker", mismatch: -1, brokerOverride: true},
		{name: "broker byte mismatch", mismatch: -1, brokerOverride: true, brokerMismatch: true},
		{name: "broker stall mismatch", mismatch: -1, brokerOverride: true, brokerMismatch: true, stallMismatch: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := gomadsim.Spec{Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendWASM, Fidelity: gomadsim.FidelitySimulationModel, Seed: 17, Limits: gomadsim.DefaultLimits()}
			spec.Limits.NetworkConnectionBytes, spec.Limits.NetworkStallNanos = 123, 456
			provider := &Provider{Boots: make(map[gomadsim.BootID]wasi.Request)}
			for index, id := range []gomadsim.NodeID{"a", "b"} {
				request := wasi.Request{
					Runtime: &wasi.RuntimeControl{Seed: 17, Choice: &runnerbackend.ChoiceRequest{}},
					Config: wasi.Config{Profile: wasi.CooperativeProfile, Environment: []string{"PWD=/workspace"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace"},
						Limits:    wasi.Limits{OutputBytes: 1024, TranscriptBytes: 1 << 20, Calls: 128, Descriptors: 64, Files: 128, FilesystemBytes: 1 << 20, PendingEvents: 128},
						NetworkV2: &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 8, Connections: 8, Bytes: 1024, ConnectionBytes: 123, StallNanos: 456, Transitions: 128}}},
				}
				if test.networkFree {
					request.Config.NetworkV2 = nil
				} else if index == test.mismatch {
					if test.stallMismatch {
						request.Config.NetworkV2.Limits.StallNanos++
					} else {
						request.Config.NetworkV2.Limits.ConnectionBytes++
					}
				}
				identity, err := BootConfiguration(request)
				if err != nil {
					t.Fatal(err)
				}
				boot := gomadsim.BootID(id)
				provider.Boots[boot] = request
				spec.Nodes = append(spec.Nodes, gomadsim.NodeSpec{ID: id, Boot: boot, Address: fmt.Sprintf("10.0.0.%d", index+1), Config: identity})
			}
			if test.restartMismatch || test.unusedMismatch {
				request := provider.Boots["a"]
				model := *request.Config.NetworkV2
				request.Config.NetworkV2 = &model
				if test.stallMismatch {
					model.Limits.StallNanos++
				} else {
					model.Limits.ConnectionBytes++
				}
				provider.Boots["extra"] = request
				if test.restartMismatch {
					provider.IncarnationBoots = []IncarnationBoot{{Handle: gomadsim.NodeHandle{Node: "a", Incarnation: 2}, Boot: "extra"}}
				}
			}
			if test.brokerOverride {
				model := *provider.Boots["a"].Config.NetworkV2
				if test.brokerMismatch {
					if test.stallMismatch {
						model.Limits.StallNanos++
					} else {
						model.Limits.ConnectionBytes++
					}
				}
				provider.Broker.Network = &model
			}
			_, err := provider.Open(t.Context(), spec)
			if test.mismatch >= 0 || test.restartMismatch || test.brokerMismatch {
				if err == nil || !strings.Contains(err.Error(), "network limits") {
					t.Fatalf("identity-bound mismatch admitted: %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if test.brokerMismatch {
				if test.stallMismatch && provider.Broker.Network.Limits.StallNanos != 457 || !test.stallMismatch && provider.Broker.Network.Limits.ConnectionBytes != 124 {
					t.Fatal("broker identity-bound inputs overwritten")
				}
			}
		})
	}
}

// retainedTargetModule stands in for the Artifact target: a detached copy of
// the module, taken before the original path is made unavailable.
func retainedTargetModule(t *testing.T, r wasi.Request) RetainedModule {
	t.Helper()
	data, err := os.ReadFile(r.ModulePath)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(path, data, 0700); err != nil {
		t.Fatal(err)
	}
	return retainedModuleFile(path, "sha256:"+r.ModuleSHA256)
}

func installedHelper(r wasi.Request) InstalledHelper {
	return InstalledHelper{Path: r.HelperPath, SHA256: r.HelperSHA256, Engine: r.Engine}
}

func guestRequest(t *testing.T) wasi.Request {
	return guestRequestWithTestBinary(t, false)
}

func guestRequestWithTestBinary(t *testing.T, testBinary bool) wasi.Request {
	t.Helper()
	if !filepath.IsAbs(os.Getenv("GOMAD_STOCK_GO")) {
		t.Fatal("GOMAD_STOCK_GO must name stock Go 1.27.1")
	}
	root := stockgo.Root(t)
	native, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	overlay, key, err := toolchain.BuildOverlay(root, native, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "service.wasm")
	arguments := []string{"build", "-trimpath", "-overlay", overlay, "-o", module, "./testdata/service"}
	if testBinary {
		arguments = append([]string{"test", "-c"}, arguments[1:]...)
	}
	cmd := exec.Command(filepath.Join(root, "bin/go"), arguments...)
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
	imports, pages, err := moduleinfo.Inspect(data)
	if err != nil {
		t.Fatal(err)
	}
	helperPath := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if helperPath == "" {
		helperPath = "../wasmhost/target/release/gomad-wasmhost"
	}
	helper, err := filepath.Abs(helperPath)
	if err != nil {
		t.Fatal(err)
	}
	helperBytes, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual small guest module-sha256=%x module-bytes=%d helper-sha256=%x implementation=%s", sha256.Sum256(data), len(data), sha256.Sum256(helperBytes), ImplementationSHA256())
	config := wasi.Config{Profile: wasi.CooperativeProfile, Args: []string{"service", "server"}, Environment: []string{"PWD=/workspace", "TZ=UTC"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, EntropyKey: [32]byte{1, 2, 3}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 32 << 20, Calls: 100000, Descriptors: 128, Files: 1024, FilesystemBytes: 8 << 20, PendingEvents: 4096}, NetworkV2: &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 10000}}}
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

func awaitProviderMarker(ctx context.Context, p *Provider, handle gomadsim.NodeHandle, marker <-chan struct{}) error {
	p.mu.Lock()
	e := p.latest
	p.mu.Unlock()
	driverCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	var driverErr error
	go func() {
		_, driverErr = e.Wait(driverCtx, handle)
		close(done)
	}()
	defer func() { cancel(); <-done }()
	select {
	case <-marker:
		return nil
	case <-done:
		select {
		case <-marker:
			return nil
		default:
		}
		if driverErr != nil {
			return driverErr
		}
		return errors.New("WASM guest terminated before the expected output marker")
	case <-ctx.Done():
		return ctx.Err()
	}
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
	spec := gomadsim.Spec{Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendWASM, Fidelity: gomadsim.FidelitySimulationModel, Seed: 7, Limits: gomadsim.DefaultLimits(), Nodes: []gomadsim.NodeSpec{{ID: "a", Boot: "durable", Address: "10.0.0.1", Config: a}, {ID: "b", Boot: "park", Address: "10.0.0.2", Config: b}}}
	p := &Provider{Boots: map[gomadsim.BootID]wasi.Request{"durable": first, "park": peer}, OutputStops: []OutputCrash{{Handle: gomadsim.NodeHandle{Node: "a", Incarnation: 1}, Marker: []byte("committed durable-a")}}}
	observed, err := gomadsim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomadsim.Cluster) error {
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
	if observed.Outcome != gomadsim.OutcomeCompleted || observed.Record.Nodes[0].State != gomadsim.NodeStateStopped || !observed.WASM.Incarnations[0].Interrupted {
		t.Fatalf("typed stopped guest lost modeled interruption: %+v", observed.Record.Nodes)
	}
	payload, err := p.RetentionPayload(spec, observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	module := retainedTargetModule(t, first)
	if err := os.Rename(first.ModulePath, first.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_STOCK_GO", "/original/compiler/unavailable")
	replayed, results, err := ReplayRetainedResults(t.Context(), payload, module, t.TempDir(), installedHelper(first))
	if err != nil {
		t.Fatal(err)
	}
	left, err := gomadsim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	right, err := gomadsim.EncodeClusterRecord(replayed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) || len(results) != 2 || results[0].WallElapsedNanos == 0 || !results[0].Result.RuntimeInterrupted || !results[0].Result.Reaped {
		t.Fatal("typed stopped guest did not source-lessly replay its sealed boundary")
	}
}

func TestProviderGroupedAcknowledgmentCrashSourceLessReplay(t *testing.T) {
	base := guestRequest(t)
	boots := map[gomadsim.BootID]wasi.Request{}
	spec := gomadsim.Spec{Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendWASM, Fidelity: gomadsim.FidelitySimulationModel, Seed: 7, Limits: gomadsim.DefaultLimits()}
	var bindings []IncarnationBoot
	for i, id := range []gomadsim.NodeID{"a", "b"} {
		r := base
		r.Config.Args = []string{"service", "durable"}
		r.Config.Storage = &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: string(id), Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
		commit, recover := gomadsim.BootID(string(id)+"-commit"), gomadsim.BootID(string(id)+"-recover")
		boots[commit] = r
		identity, err := BootConfiguration(r)
		if err != nil {
			t.Fatal(err)
		}
		spec.Nodes = append(spec.Nodes, gomadsim.NodeSpec{ID: id, Boot: commit, Address: fmt.Sprintf("10.0.0.%d", i+1), Config: identity, Volumes: []gomadsim.VolumeMount{{Volume: gomadsim.VolumeID(id), Path: "/workspace"}}})
		spec.Volumes = append(spec.Volumes, gomadsim.VolumeSpec{ID: gomadsim.VolumeID(id), CapacityBytes: 8 << 20})
		r.Config.Args = []string{"service", "recover"}
		boots[recover] = r
		bindings = append(bindings, IncarnationBoot{Handle: gomadsim.NodeHandle{Node: id, Incarnation: 2}, Boot: recover})
	}
	group := OutputCrashGroup{Handles: []gomadsim.NodeHandle{{Node: "a", Incarnation: 1}, {Node: "b", Incarnation: 1}}, Marker: []byte("committed durable-a")}
	p := &Provider{Boots: boots, IncarnationBoots: bindings, OutputCrashGroups: []OutputCrashGroup{group}}
	observed, err := gomadsim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomadsim.Cluster) error {
		for _, id := range []gomadsim.NodeID{"a", "b"} {
			if _, err := c.Start(ctx, id); err != nil {
				return err
			}
		}
		for _, handle := range group.Handles {
			if err := c.Crash(ctx, handle); err != nil {
				return err
			}
		}
		for _, id := range []gomadsim.NodeID{"a", "b"} {
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
	if observed.Outcome != gomadsim.OutcomeCompleted || len(observed.WASM.Incarnations) != 4 || len(observed.Record.Volumes.Snapshot.Volumes) != 2 {
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
	module := retainedTargetModule(t, base)
	if err := os.Rename(base.ModulePath, base.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_STOCK_GO", "/original/compiler/unavailable")
	replayed, original, completed, err := ReplayBoundRetained(t.Context(), payload, module, t.TempDir(), installedHelper(base))
	if err != nil {
		t.Fatal(err)
	}
	if completed == nil {
		t.Fatal("bound grouped replay lost completed observations")
	}
	logIncarnationSummaries(t, completed, "grouped-replay")
	left, err := gomadsim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	right, err := gomadsim.EncodeClusterRecord(replayed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) || record.HashBytes(left) != original {
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
	spec := gomadsim.Spec{Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendWASM, Fidelity: gomadsim.FidelitySimulationModel, Seed: 7, Limits: gomadsim.DefaultLimits(), Volumes: []gomadsim.VolumeSpec{{ID: "a-store", CapacityBytes: 8 << 20}}, Nodes: []gomadsim.NodeSpec{{ID: "a", Boot: "durable", Address: "10.0.0.1", Config: a, Volumes: []gomadsim.VolumeMount{{Volume: "a-store", Path: "/workspace"}}}, {ID: "b", Boot: "park", Address: "10.0.0.2", Config: b}}}
	var recordedProvider *Provider
	run := func() (gomadsim.Result, error) {
		writer := &markerWriter{marker: []byte("committed durable-a"), seen: make(chan struct{})}
		first := request
		first.StdoutMirror = writer
		p := &Provider{Boots: map[gomadsim.BootID]wasi.Request{"durable": first, "park": peer, "recover": recovery}, IncarnationBoots: []IncarnationBoot{{Handle: gomadsim.NodeHandle{Node: "a", Incarnation: 2}, Boot: "recover"}}, OutputCrashes: []OutputCrash{{Handle: gomadsim.NodeHandle{Node: "a", Incarnation: 1}, Marker: []byte("committed durable-a")}}}
		recordedProvider = p
		return gomadsim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomadsim.Cluster) error {
			a, err := c.Start(ctx, "a")
			if err != nil {
				return err
			}
			if err := awaitProviderMarker(ctx, p, a, writer.seen); err != nil {
				return err
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
	if observed.Outcome != gomadsim.OutcomeCompleted || len(observed.Record.Volumes.Snapshot.Volumes) != 1 {
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
	module := retainedTargetModule(t, request)
	if err := os.Rename(request.ModulePath, request.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_STOCK_GO", "/original/compiler/unavailable")
	replayed, err := ReplayRetained(t.Context(), payload, module, t.TempDir(), installedHelper(request))
	if err != nil {
		t.Fatal(err)
	}
	left, err := gomadsim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	right, err := gomadsim.EncodeClusterRecord(replayed.Record)
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
	spec := gomadsim.Spec{Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendWASM, Fidelity: gomadsim.FidelitySimulationModel, Seed: 7, Limits: gomadsim.DefaultLimits(), Nodes: []gomadsim.NodeSpec{{ID: "a", Boot: "server", Address: "10.0.0.1", Config: a}, {ID: "b", Boot: "client", Address: "10.0.0.2", Config: b}}}
	spec.Links = []gomadsim.LinkSpec{{From: "a", To: "b", Enabled: true}, {From: "b", To: "a", Enabled: true}}
	p := &Provider{Boots: map[gomadsim.BootID]wasi.Request{"server": server, "client": client}}
	if profile != "" {
		p.Broker.WorldProfile = profile
		p.Broker.Limits = world.CompressedSnapshotProfile().Limits
	}
	scenario := func(ctx context.Context, c gomadsim.Cluster) error {
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
	observed, err := gomadsim.RunWASM(t.Context(), spec, scenario, p)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Outcome != gomadsim.OutcomeCompleted || len(observed.Record.Transitions) != 8 {
		t.Fatalf("isolated lifecycle result %+v", observed)
	}
	if observed.Record.WASM.WorldEncoding != profile {
		t.Fatal("explicit World profile was not retained")
	}
	logIncarnationSummaries(t, p, "lifecycle-record")
	for _, output := range observed.Outputs {
		if output.Stream == gomadsim.OutputStdout && (!bytes.Contains(output.Bytes, []byte("fresh heap 1")) || !bytes.Contains(output.Bytes, []byte("ordered bytes ordered-byte"))) {
			t.Fatalf("incarnation %v reused heap or changed stream: %s", output.Handle, output.Bytes)
		}
	}
	payload, err := p.RetentionPayload(spec, observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	targetModule := retainedTargetModule(t, server)
	// Retention references the module by digest: the evidence never reads the
	// module, so its bytes do not depend on module size or availability.
	detached := &Provider{Boots: map[gomadsim.BootID]wasi.Request{}, Broker: p.Broker, OutputCrashes: p.OutputCrashes, OutputStops: p.OutputStops, IncarnationBoots: p.IncarnationBoots, OutputCrashGroups: p.OutputCrashGroups, Oracles: p.Oracles}
	for id, boot := range p.Boots {
		boot.ModulePath = filepath.Join(t.TempDir(), "unavailable-module")
		detached.Boots[id] = boot
	}
	without, err := detached.RetentionPayload(spec, observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	encodedRecord, err := gomadsim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained evidence=%d bytes record=%d bytes", len(payload.Data), len(encodedRecord))
	if !bytes.Equal(without.Data, payload.Data) || uint64(len(payload.Data)) > uint64(len(encodedRecord))*4/3+64<<10 {
		t.Fatal("retained evidence embeds or depends on module bytes")
	}
	for _, test := range []struct {
		name   string
		mutate func(*retainedExecution)
	}{
		{"missing-action", func(b *retainedExecution) { b.Lifecycle = b.Lifecycle[:len(b.Lifecycle)-1] }},
		{"mutated-action", func(b *retainedExecution) { b.Lifecycle[0].Handle.Incarnation++ }},
		{"missing-module", func(b *retainedExecution) { b.ModuleSHA256 = "" }},
		{"foreign-module", func(b *retainedExecution) { b.ModuleSHA256 = strings.Repeat("0", 64) }},
		{"boot-module-differs", func(b *retainedExecution) { b.Boots[0].Request.ModuleSHA256 = strings.Repeat("0", 64) }},
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
			var observed gomadsim.ClusterRecord
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
		{"trailing-world-byte", func(b *retainedExecution) {
			mutateRetainedWorld(t, b, func(data []byte) []byte {
				return append(bytes.Clone(data), 0)
			})
		}},
		{"invalid-world-header", func(b *retainedExecution) {
			mutateRetainedWorld(t, b, func(data []byte) []byte {
				changed := bytes.Clone(data)
				changed[0] ^= 1
				return changed
			})
		}},
		{"corrupt-world-digest", func(b *retainedExecution) {
			mutateRetainedWorld(t, b, func(data []byte) []byte {
				changed := bytes.Clone(data)
				changed[32] ^= 1
				return changed
			})
		}},
		{"changed-runtime-binding", func(b *retainedExecution) {
			b.Boots[0].Request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey += "-changed"
		}},
		{"changed-original-record", func(b *retainedExecution) { b.Record = append(b.Record, '\n') }},
		{"changed-root-identity", func(b *retainedExecution) {
			var observed gomadsim.ClusterRecord
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
			if _, _, err := ReopenProvider(bad, targetModule, directory, installedHelper(server)); err == nil {
				t.Fatal("mutated retained inputs were admitted")
			}
			if _, _, _, err := ReplayBoundRetained(t.Context(), bad, targetModule, directory, installedHelper(server)); err == nil {
				t.Fatal("bound replay admitted mutated retained inputs")
			}
			entries, err := os.ReadDir(directory)
			if err != nil || len(entries) != 0 {
				t.Fatalf("rejection wrote helper inputs before validation: %v %v", entries, err)
			}
		})
	}
	foreign := installedHelper(server)
	foreign.SHA256 = strings.Repeat("f", 64)
	directory := t.TempDir()
	if _, _, _, err := ReplayBoundRetained(t.Context(), payload, targetModule, directory, foreign); err == nil || !strings.Contains(err.Error(), server.HelperSHA256) || !strings.Contains(err.Error(), foreign.SHA256) {
		t.Fatalf("bound replay admitted a helper other than the installation's: %v", err)
	}
	if entries, err := os.ReadDir(directory); err != nil || len(entries) != 0 {
		t.Fatalf("foreign installed helper wrote executable inputs: %v %v", entries, err)
	}
	for _, primary := range []string{"", "sha256:" + strings.Repeat("0", 64)} {
		directory := t.TempDir()
		foreignPrimary := targetModule
		foreignPrimary.SHA256 = primary
		if _, _, _, err := ReplayBoundRetained(t.Context(), payload, foreignPrimary, directory, installedHelper(server)); err == nil {
			t.Fatal("bound replay accepted foreign or missing Artifact primary")
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 0 {
			t.Fatalf("unbound primary wrote executable inputs: %v %v", entries, err)
		}
	}
	module := retainedTargetModule(t, server)
	if err := os.Rename(server.ModulePath, server.ModulePath+".unavailable"); err != nil {
		t.Fatal(err)
	}
	// Reopening receives no source or stock toolchain. Executable inputs are
	// copied from the existing Artifact payload into fresh private files.
	t.Setenv("GOMAD_STOCK_GO", "/original/compiler/unavailable")
	replayed, original, completed, err := ReplayBoundRetained(t.Context(), payload, module, t.TempDir(), installedHelper(server))
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := decodeRetained(payload)
	if err != nil || original != record.HashBytes(bundle.Record) || completed == nil || len(completed.CompletedSummaries()) != 4 {
		t.Fatalf("bound replay lost original canonical record digest or terminal summaries: %v", err)
	}
	logIncarnationSummaries(t, completed, "lifecycle-replay")
	left, err := gomadsim.EncodeClusterRecord(observed.Record)
	if err != nil {
		t.Fatal(err)
	}
	right, err := gomadsim.EncodeClusterRecord(replayed.Record)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) {
		t.Fatal("actual WASM lifecycle replay changed canonical record")
	}
	called := false
	if _, err = gomadsim.Run(t.Context(), spec, func(context.Context, gomadsim.Cluster) error { called = true; return nil }); err == nil || called {
		t.Fatal("WASM without an injected engine did not fail closed")
	}
}

func mutateRetainedWorld(t *testing.T, bundle *retainedExecution, mutate func([]byte) []byte) {
	t.Helper()
	var observed gomadsim.ClusterRecord
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

// An unbound (driver-initiated) crash or stop of a split-owned storage node is
// delivered in band, so the helper flushes its recovery image and the next
// incarnation recovers the committed value (fn-176.6).
func TestProviderUnboundRevocationOfSplitStorageNodeRestarts(t *testing.T) {
	request := guestRequest(t)
	request.Config.Args = []string{"service", "durable"}
	request.Config.Storage = &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: "a-store", Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
	recovery := request
	recovery.Config.Args = []string{"service", "recover"}
	identity, err := BootConfiguration(request)
	if err != nil {
		t.Fatal(err)
	}
	spec := gomadsim.Spec{Schema: gomadsim.SpecSchema, Backend: gomadsim.BackendWASM, Fidelity: gomadsim.FidelitySimulationModel, Seed: 7, Limits: gomadsim.DefaultLimits(), Volumes: []gomadsim.VolumeSpec{{ID: "a-store", CapacityBytes: 8 << 20}}, Nodes: []gomadsim.NodeSpec{{ID: "a", Boot: "durable", Address: "10.0.0.1", Config: identity, Volumes: []gomadsim.VolumeMount{{Volume: "a-store", Path: "/workspace"}}}}}
	dormant := request
	dormant.Config.Storage = nil
	dormant.Config.Args = []string{"service", "park"}
	dormantIdentity, err := BootConfiguration(dormant)
	if err != nil {
		t.Fatal(err)
	}
	spec.Nodes = append(spec.Nodes, gomadsim.NodeSpec{ID: "dormant", Boot: "dormant", Address: "10.0.0.2", Config: dormantIdentity})
	for _, action := range []string{"crash", "stop"} {
		t.Run(action, func(t *testing.T) {
			writer := &markerWriter{marker: []byte("committed durable-a"), seen: make(chan struct{})}
			first := request
			first.StdoutMirror = writer
			if !wasi.SplitOwnsStorage(wasi.Request{Config: first.Config, Broker: &wasi.BrokerSession{}}) {
				t.Fatal("cooperative broker storage run is not split-owned")
			}
			p := &Provider{Boots: map[gomadsim.BootID]wasi.Request{"durable": first, "recover": recovery, "dormant": dormant}, IncarnationBoots: []IncarnationBoot{{Handle: gomadsim.NodeHandle{Node: "a", Incarnation: 2}, Boot: "recover"}}}
			observed, err := gomadsim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomadsim.Cluster) error {
				a, err := c.Start(ctx, "a")
				if err != nil {
					return err
				}
				if err := awaitProviderMarker(ctx, p, a, writer.seen); err != nil {
					return err
				}
				if action == "crash" {
					err = c.Crash(ctx, a)
				} else {
					err = c.Stop(ctx, a)
				}
				if err != nil {
					return err
				}
				if a, err = c.Restart(ctx, "a"); err != nil {
					return err
				}
				_, err = c.Wait(ctx, a)
				return err
			}, p)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, output := range observed.Outputs {
				found = found || output.Handle.Incarnation == 2 && bytes.Contains(output.Bytes, []byte("recovered durable-a"))
			}
			if !found {
				t.Fatalf("unbound %s lost the durable value: %+v", action, observed.Outputs)
			}
		})
	}
}
