package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestWASMWorldEncodingAdmissionAndOpaqueBytes(t *testing.T) {
	identity := strings.Repeat("a", 64)
	evidence := WASMExecutionEvidence{ConfigurationSHA256: "sha256:" + identity, World: []byte("GMWORLD2\x00"), Incarnations: []WASMIncarnationEvidence{{Handle: NodeHandle{Node: "a", Incarnation: 1}, ModuleSHA256: identity, HelperSHA256: identity, EnvironmentIdentity: identity, ChoiceTape: []byte{1}, WASISHA256: "sha256:" + identity, NetworkSHA256: "sha256:" + identity, StorageSHA256: "sha256:" + identity}}}
	encoded, err := json.Marshal(evidence)
	if err != nil || strings.Contains(string(encoded), "world_encoding") {
		t.Fatalf("absent encoding changed record bytes: %v", err)
	}
	if err := validateWASMExecution(&evidence, BackendWASM); err != nil {
		t.Fatal(err)
	}
	evidence.WorldEncoding = "unknown"
	if err := validateWASMExecution(&evidence, BackendWASM); err == nil {
		t.Fatal("unknown World encoding admitted")
	}
	evidence.WorldEncoding = WASMWorldCompressedSnapshotEncoding
	if err := validateWASMExecution(&evidence, BackendWASM); err != nil {
		t.Fatal(err)
	}
}

func TestWASMLiveEnumerationFailsClosedAfterHandleValidation(t *testing.T) {
	h := NodeHandle{Node: "a", Incarnation: 1}
	c := &inProcessCluster{wasm: &lifecycleAdmissionExecution{}, nodes: map[NodeID]*clusterNode{"a": {handle: h, state: NodeStateRunning}}, activity: make(chan struct{}, 1)}
	_, err := c.EnumerateCrashStates(t.Context(), h, "volume", VolumeCrashEnumerationLimits{}, nil)
	var unsupported *UnsupportedWASMCapabilityError
	if !errors.As(err, &unsupported) || unsupported.Backend != BackendWASM || unsupported.Capability != "live-volume-crash-enumeration" {
		t.Fatalf("unsupported WASM action reached native enumeration: %v", err)
	}
	_, err = c.EnumerateCrashStates(t.Context(), NodeHandle{Node: "a", Incarnation: 2}, "volume", VolumeCrashEnumerationLimits{}, nil)
	if !errors.Is(err, ErrStaleIncarnation) {
		t.Fatalf("capability rejection masked stale handle: %v", err)
	}
	c.nodes["a"].state = NodeStateStopped
	_, err = c.EnumerateCrashStates(t.Context(), h, "volume", VolumeCrashEnumerationLimits{}, nil)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("capability rejection masked stopped state: %v", err)
	}
}

type lifecycleAdmissionExecution struct{ effects int }

func (*lifecycleAdmissionExecution) Start(NodeSpec, NodeHandle) error { return nil }
func (*lifecycleAdmissionExecution) Wait(context.Context, NodeHandle) (WASMTerminal, error) {
	return WASMTerminal{}, nil
}
func (e *lifecycleAdmissionExecution) Stop(context.Context, NodeHandle) (WASMTerminal, error) {
	e.effects++
	return WASMTerminal{}, nil
}
func (e *lifecycleAdmissionExecution) Crash(context.Context, NodeHandle) (WASMTerminal, error) {
	e.effects++
	return WASMTerminal{}, nil
}
func (*lifecycleAdmissionExecution) Link(string, NodeID, NodeID, uint64, bool) error { return nil }
func (*lifecycleAdmissionExecution) Finish() (WASMEvidence, error)                   { return WASMEvidence{}, nil }

func (*lifecycleAdmissionExecution) Shutdown(context.Context) error { return nil }

func TestWASMShutdownDoesNotReserveModeledLifecycle(t *testing.T) {
	h := NodeHandle{Node: "a", Incarnation: 1}
	e := &lifecycleAdmissionExecution{}
	c := &inProcessCluster{wasm: e, nodes: map[NodeID]*clusterNode{"a": {handle: h, state: NodeStateRunning}}, ordered: []NodeID{"a"}, limits: Limits{}, activity: make(chan struct{}, 1)}
	if err := c.shutdownWASM(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(c.transitions) != 0 || e.effects != 0 || c.nodes["a"].state != NodeStateStopped {
		t.Fatal("mandatory cleanup reserved or applied modeled Stop/Crash")
	}
}

func TestWASMLifecycleRejectsBeforeExternalEffects(t *testing.T) {
	for _, action := range []LifecycleAction{LifecycleStop, LifecycleCrash} {
		for _, rejected := range []string{"capacity", "replay"} {
			t.Run(string(action)+"/"+rejected, func(t *testing.T) {
				h := NodeHandle{Node: "a", Incarnation: 1}
				e := &lifecycleAdmissionExecution{}
				limits := DefaultLimits()
				c := &inProcessCluster{wasm: e, nodes: map[NodeID]*clusterNode{"a": {handle: h, state: NodeStateRunning}}, limits: limits, activity: make(chan struct{}, 1)}
				if rejected == "capacity" {
					c.limits.ScenarioActions = 0
				} else {
					c.replay = &ReplayPlan{Transitions: []LifecycleTransition{{Ordinal: 0, Action: LifecycleWait, Handle: h, From: NodeStateRunning, To: NodeStateExited}}}
				}
				if _, err := c.endWASM(t.Context(), h, action); err == nil {
					t.Fatal("invalid lifecycle admitted")
				}
				if e.effects != 0 || c.nodes["a"].state != NodeStateRunning || c.nodes["a"].operation != nil || c.pendingOps != 0 {
					t.Fatal("rejected lifecycle changed external guest state")
				}
			})
		}
	}
}
