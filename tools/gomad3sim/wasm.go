package gomad3sim

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// WASMProvider opens an isolated execution domain. Boot IDs name verified guest
// modules rather than host functions; lifecycle transitions remain cluster-owned.
type WASMProvider interface {
	Open(context.Context, Spec) (WASMExecution, error)
}

// UnsupportedWASMCapabilityError names an unsupported controller operation,
// without claiming that otherwise available guest execution is unavailable.
type UnsupportedWASMCapabilityError struct {
	Backend    Backend
	Capability string
}

func (err *UnsupportedWASMCapabilityError) Error() string {
	return fmt.Sprintf("%s does not support %s", err.Backend, err.Capability)
}

type WASMExecution interface {
	Start(NodeSpec, NodeHandle) error
	Wait(context.Context, NodeHandle) (WASMTerminal, error)
	Stop(context.Context, NodeHandle) (WASMTerminal, error)
	Crash(context.Context, NodeHandle) (WASMTerminal, error)
	Link(action string, from, to NodeID, delay uint64, symmetric bool) error
	Finish() (WASMEvidence, error)
	Shutdown(context.Context) error
}

type WASMTerminal struct {
	Failed      bool
	Reason      string
	Termination string
}
type WASMEvidence struct {
	Outputs   []OutputObservation
	Network   NetworkRecord
	Volumes   VolumeRecord
	Execution *WASMExecutionEvidence
}

// WASMWorldCompressedSnapshotEncoding names the World-owned bounded profile.
const WASMWorldCompressedSnapshotEncoding = "gomad3/world/compressed-snapshot/v1"

// WASMExecutionEvidence retains guest-local control independently from the
// cluster lifecycle tape. Payloads are detached bytes from their existing owners.
type WASMExecutionEvidence struct {
	ConfigurationSHA256 string                    `json:"configuration_sha256"`
	VirtualElapsedNanos uint64                    `json:"virtual_elapsed_nanos"`
	Incarnations        []WASMIncarnationEvidence `json:"incarnations"`
	World               []byte                    `json:"world"`
	WorldEncoding       string                    `json:"world_encoding,omitempty"`
}
type WASMIncarnationEvidence struct {
	Handle               NodeHandle `json:"handle"`
	ModuleSHA256         string     `json:"module_sha256"`
	HelperSHA256         string     `json:"helper_sha256"`
	EnvironmentIdentity  string     `json:"environment_identity"`
	ChoiceTape           []byte     `json:"choice_tape"`
	ChoiceTrace          []byte     `json:"choice_trace"`
	DiagnosticTrace      []byte     `json:"diagnostic_trace"`
	Interrupted          bool       `json:"interrupted"`
	WASISHA256           string     `json:"wasi_sha256"`
	NetworkSHA256        string     `json:"network_sha256"`
	StorageSHA256        string     `json:"storage_sha256"`
	StorageImageIdentity string     `json:"storage_image_identity,omitempty"`
}

func cloneWASMExecution(source *WASMExecutionEvidence) *WASMExecutionEvidence {
	if source == nil {
		return nil
	}
	cloned := *source
	cloned.World = append([]byte(nil), source.World...)
	cloned.Incarnations = append([]WASMIncarnationEvidence(nil), source.Incarnations...)
	for i := range cloned.Incarnations {
		cloned.Incarnations[i].ChoiceTape = append([]byte(nil), source.Incarnations[i].ChoiceTape...)
		cloned.Incarnations[i].ChoiceTrace = append([]byte(nil), source.Incarnations[i].ChoiceTrace...)
		cloned.Incarnations[i].DiagnosticTrace = append([]byte(nil), source.Incarnations[i].DiagnosticTrace...)
	}
	return &cloned
}

func validateWASMExecution(evidence *WASMExecutionEvidence, backend Backend) error {
	if evidence == nil {
		return nil
	}
	if backend != BackendWASM || len(evidence.Incarnations) == 0 || len(evidence.World) == 0 || !validSHA256(evidence.ConfigurationSHA256) {
		return errors.New("invalid WASM execution evidence")
	}
	if evidence.WorldEncoding != "" && evidence.WorldEncoding != WASMWorldCompressedSnapshotEncoding {
		return errors.New("unsupported WASM World encoding")
	}
	var world json.RawMessage
	if err := json.Unmarshal(evidence.World, &world); err != nil {
		return fmt.Errorf("WASM World evidence: %w", err)
	}
	canonical, err := json.Marshal(world)
	if err != nil || !bytes.Equal(canonical, evidence.World) {
		return errors.New("noncanonical WASM World evidence")
	}
	for i, n := range evidence.Incarnations {
		if n.Handle.Node == "" || n.Handle.Incarnation == 0 || i > 0 && (evidence.Incarnations[i-1].Handle.Node > n.Handle.Node || evidence.Incarnations[i-1].Handle.Node == n.Handle.Node && evidence.Incarnations[i-1].Handle.Incarnation >= n.Handle.Incarnation) {
			return errors.New("unordered WASM incarnation evidence")
		}
		if !validSHA256("sha256:"+n.ModuleSHA256) || !validSHA256("sha256:"+n.HelperSHA256) || !validSHA256("sha256:"+n.EnvironmentIdentity) || len(n.ChoiceTape) == 0 || !validSHA256(n.WASISHA256) || !validSHA256(n.NetworkSHA256) || !validSHA256(n.StorageSHA256) {
			return errors.New("invalid WASM incarnation identities or tape")
		}
	}
	return nil
}

func RunWASM(ctx context.Context, spec Spec, scenario Scenario, provider WASMProvider) (Result, error) {
	if ctx == nil || scenario == nil {
		return Result{}, errors.New("WASM simulation context or scenario is nil")
	}
	if spec.Backend != BackendWASM || provider == nil {
		return Result{}, &BackendUnavailableError{Backend: spec.Backend}
	}
	if err := ValidateSpec(spec); err != nil {
		return Result{}, err
	}
	execution, err := provider.Open(ctx, spec)
	if err != nil {
		return Result{}, err
	}
	if execution == nil {
		return Result{}, &BackendUnavailableError{Backend: BackendWASM}
	}
	return run(ctx, spec, scenario, execution)
}

func (cluster *inProcessCluster) endWASM(ctx context.Context, handle NodeHandle, action LifecycleAction) (NodeResult, error) {
	cluster.mu.Lock()
	node, err := cluster.currentNode(handle, NodeStateRunning)
	if err != nil {
		cluster.mu.Unlock()
		return NodeResult{}, err
	}
	if node.operation != nil {
		cluster.mu.Unlock()
		return NodeResult{}, ErrInvalidTransition
	}
	var entry *transitionEntry
	switch action {
	case LifecycleStop:
		entry, err = cluster.reserveStopTransition(handle)
	case LifecycleCrash:
		var transition LifecycleTransition
		transition, err = cluster.prepareTransition(action, handle, NodeStateRunning, NodeStateCrashed)
		if err == nil {
			entry = &transitionEntry{transition: transition}
			cluster.transitions = append(cluster.transitions, entry)
		}
	case LifecycleWait:
	default:
		err = ErrInvalidTransition
	}
	if err != nil {
		cluster.mu.Unlock()
		return NodeResult{}, err
	}
	op := &nodeOperation{action: action, done: make(chan struct{})}
	node.operation = op
	cluster.pendingOps++
	cluster.mu.Unlock()
	var terminal WASMTerminal
	switch action {
	case LifecycleWait:
		terminal, err = cluster.wasm.Wait(ctx, handle)
	case LifecycleStop:
		terminal, err = cluster.wasm.Stop(ctx, handle)
	case LifecycleCrash:
		terminal, err = cluster.wasm.Crash(ctx, handle)
	default:
		err = ErrInvalidTransition
	}
	cluster.mu.Lock()
	defer cluster.mu.Unlock()
	defer cluster.finishOperationLocked(op)
	if err != nil {
		if entry != nil {
			cluster.transitions = cluster.transitions[:len(cluster.transitions)-1]
		}
		return NodeResult{}, err
	}
	to := NodeStateExited
	if action == LifecycleStop {
		to = NodeStateStopped
	}
	if action == LifecycleCrash {
		to = NodeStateCrashed
	}
	if terminal.Failed && action != LifecycleCrash {
		to = NodeStateFailed
	}
	if entry != nil {
		actual := entry.transition
		actual.To = to
		if cluster.replay != nil && entry.transition != actual {
			err := transitionDivergence(actual.Ordinal, entry.transition, actual)
			cluster.transitions = cluster.transitions[:len(cluster.transitions)-1]
			cluster.replayFailure = err
			return NodeResult{}, err
		}
		entry.transition, entry.resolved = actual, true
	} else {
		transition, err := cluster.prepareTransition(action, handle, NodeStateRunning, to)
		if err != nil {
			return NodeResult{}, err
		}
		cluster.commitTransition(transition)
	}
	return cluster.commitTerminal(node, to, boundedTerminalText(terminal.Reason)), nil
}

func (cluster *inProcessCluster) shutdownWASM(ctx context.Context) error {
	cluster.mu.Lock()
	cluster.closing = true
	var handles []NodeHandle
	for _, id := range cluster.ordered {
		if n := cluster.nodes[id]; n.state == NodeStateRunning {
			handles = append(handles, n.handle)
		}
	}
	cluster.mu.Unlock()
	err := cluster.wasm.Shutdown(ctx)
	if quiescenceErr := cluster.waitForQuiescence(context.WithoutCancel(ctx)); quiescenceErr != nil {
		return errors.Join(err, quiescenceErr)
	}
	for _, h := range handles {
		terminal, current := cluster.wasm.Wait(context.WithoutCancel(ctx), h)
		err = errors.Join(err, current)
		cluster.mu.Lock()
		n, lookupErr := cluster.currentNode(h, NodeStateRunning)
		if lookupErr == nil && current == nil {
			to, reason := NodeStateStopped, ""
			if terminal.Termination == "crash" {
				to = NodeStateCrashed
			}
			if terminal.Failed && terminal.Termination != "crash" && terminal.Termination != "stop" {
				to, reason = NodeStateFailed, boundedTerminalText(terminal.Reason)
			}
			cluster.commitTerminal(n, to, reason)
		}
		cluster.mu.Unlock()
	}
	return err
}

func BindWASMVolumeEvidence(transitions []VolumeTransition, states []VolumeStateSnapshot, limits Limits) (VolumeRecord, error) {
	record := cloneVolumeRecord(VolumeRecord{Transitions: transitions, Snapshot: VolumeSnapshot{Volumes: states}})
	sort.SliceStable(record.Transitions, func(i, j int) bool {
		return volumeTransitionLane(record.Transitions[i]) < volumeTransitionLane(record.Transitions[j])
	})
	for i := range record.Transitions {
		record.Transitions[i].Ordinal = uint64(i)
	}
	sort.Slice(record.Snapshot.Volumes, func(i, j int) bool {
		a, b := record.Snapshot.Volumes[i], record.Snapshot.Volumes[j]
		return string(a.Node)+"\x00"+string(a.Volume) < string(b.Node)+"\x00"+string(b.Volume)
	})
	for i := range record.Snapshot.Volumes {
		record.Snapshot.Volumes[i].Identity = volumeStateSnapshotIdentity(record.Snapshot.Volumes[i])
	}
	record.Snapshot.TransitionSHA256 = volumeTransitionsIdentity(record.Transitions)
	record.Snapshot.Identity = volumeRunSnapshotIdentity(record.Snapshot)
	return record, validateVolumeRecord(record, limits)
}
