package cluster

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/history"
)

const SpecSchema = "gomad3.simulation-spec/v8"
const ClusterRecordSchema = "gomad3.cluster-record/v8"
const ClusterReplaySchema = "gomad3.cluster-replay/v8"

const MaximumClusterRecordBytes = 128 << 20
const MaximumTerminalReasonBytes = 4096

const (
	BackendInProcess Backend = "in_process"
	BackendProcess   Backend = "process"
	BackendWASM      Backend = "wasm"
)

const (
	FidelitySimulationModel Fidelity = "simulation_model"
	FidelityHardIsolation   Fidelity = "hard_isolation"
)

const (
	NodeStateDefined NodeState = "defined"
	NodeStateRunning NodeState = "running"
	NodeStateExited  NodeState = "exited"
	NodeStateStopped NodeState = "stopped"
	NodeStateCrashed NodeState = "crashed"
	NodeStateFailed  NodeState = "failed"
)

const (
	LifecycleStart   LifecycleAction = "start"
	LifecycleWait    LifecycleAction = "wait"
	LifecycleStop    LifecycleAction = "stop"
	LifecycleCrash   LifecycleAction = "crash"
	LifecycleRestart LifecycleAction = "restart"
)

const (
	ReplayDimensionTransition  ReplayDimension = "transition"
	ReplayDimensionNetwork     ReplayDimension = "network"
	ReplayDimensionVolume      ReplayDimension = "volume"
	ReplayDimensionFault       ReplayDimension = "fault"
	ReplayDimensionScenario    ReplayDimension = "scenario"
	ReplayDimensionExploration ReplayDimension = "exploration"
	ReplayDimensionEvidence    ReplayDimension = "evidence"
	ReplayDimensionTerminal    ReplayDimension = "terminal"
)

const (
	OutcomeCompleted      Outcome = "completed"
	OutcomeScenarioFailed Outcome = "scenario_failed"
	OutcomeOracleFailed   Outcome = "oracle_failed"
	OutcomeReplayDiverged Outcome = "replay_diverged"
)

const (
	OutputStdout OutputStream = "stdout"
	OutputStderr OutputStream = "stderr"
)

const (
	LimitationSharedPackageGlobals         Limitation = "shared_package_globals"
	LimitationRevokedGoroutinesMayRemain   Limitation = "revoked_goroutines_may_remain"
	LimitationCPULoopsRequireWatchdog      Limitation = "cpu_loops_require_outer_watchdog"
	LimitationHardIsolationRequiresProcess Limitation = "hard_isolation_requires_process_backend"
)

const LeakRevokedGoroutineMayRemain LeakKind = "revoked_goroutine_may_remain"

var ErrInvalidTransition = errors.New("invalid simulation lifecycle transition")
var ErrStaleIncarnation = errors.New("stale simulation node incarnation")
var ErrReplayDiverged = errors.New("simulation lifecycle replay diverged")
var ErrBackendUnavailable = errors.New("simulation backend is unavailable")

type Backend string
type Fidelity string
type BootID = history.BootID
type NodeID = history.NodeID
type VolumeID = history.VolumeID
type NodeState string
type LifecycleAction string
type ReplayDimension string
type Outcome string
type OutputStream = history.OutputStream
type Limitation string
type LeakKind string
type NetworkTransitionKind = history.NetworkTransitionKind
type NetworkOutcome = history.NetworkOutcome
type VolumeTransitionKind = history.VolumeTransitionKind
type VolumeOutcome = history.VolumeOutcome
type VolumeCrashCapacity = history.VolumeCrashCapacity

const (
	VolumeOperationAllocate      VolumeTransitionKind = "allocate"
	VolumeOperationResize        VolumeTransitionKind = "resize"
	VolumeOperationWrite         VolumeTransitionKind = "write"
	VolumeOperationMetadata      VolumeTransitionKind = "metadata"
	VolumeOperationNamespace     VolumeTransitionKind = "namespace"
	VolumeOperationFileSync      VolumeTransitionKind = "file_sync"
	VolumeOperationDirectorySync VolumeTransitionKind = "directory_sync"
	VolumeOperationFlush         VolumeTransitionKind = "flush"
	VolumeOperationCrash         VolumeTransitionKind = "crash"
)

const (
	VolumeOutcomeOK       VolumeOutcome = "ok"
	VolumeOutcomeCapacity VolumeOutcome = "capacity"
	VolumeOutcomeStale    VolumeOutcome = "stale"
)

const (
	VolumeCrashCapacityStates     VolumeCrashCapacity = "states"
	VolumeCrashCapacityOperations VolumeCrashCapacity = "operations"
	VolumeCrashCapacityDepth      VolumeCrashCapacity = "depth"
	VolumeCrashCapacityBytes      VolumeCrashCapacity = "bytes"
	VolumeCrashCapacityWall       VolumeCrashCapacity = "wall"
)

const (
	NetworkListen           NetworkTransitionKind = "listen"
	NetworkDial             NetworkTransitionKind = "dial"
	NetworkAccept           NetworkTransitionKind = "accept"
	NetworkWrite            NetworkTransitionKind = "write"
	NetworkClose            NetworkTransitionKind = "close"
	NetworkReset            NetworkTransitionKind = "reset"
	NetworkListenerClose    NetworkTransitionKind = "listener_close"
	NetworkPartition        NetworkTransitionKind = "partition"
	NetworkHeal             NetworkTransitionKind = "heal"
	NetworkDelay            NetworkTransitionKind = "delay"
	NetworkDisconnect       NetworkTransitionKind = "disconnect"
	NetworkReconnect        NetworkTransitionKind = "reconnect"
	NetworkDirectionalDelay NetworkTransitionKind = "directional_delay"
	NetworkStop             NetworkTransitionKind = "stop"
	NetworkCrash            NetworkTransitionKind = "crash"
)

const (
	NetworkOutcomeOK          NetworkOutcome = "ok"
	NetworkOutcomeClosed      NetworkOutcome = "closed"
	NetworkOutcomeRefused     NetworkOutcome = "refused"
	NetworkOutcomeDeadline    NetworkOutcome = "deadline"
	NetworkOutcomeHeld        NetworkOutcome = "held"
	NetworkOutcomeTimeout     NetworkOutcome = "timeout"
	NetworkOutcomeReset       NetworkOutcome = "reset"
	NetworkOutcomeCapacity    NetworkOutcome = "capacity"
	NetworkOutcomeUnsupported NetworkOutcome = "unsupported"
)

type NodeHandle = history.NodeHandle

type NodeResult struct {
	Handle NodeHandle `json:"handle"`
	State  NodeState  `json:"state"`
	Reason string     `json:"reason,omitempty"`
}

type LifecycleTransition struct {
	Ordinal uint64          `json:"ordinal"`
	Action  LifecycleAction `json:"action"`
	Handle  NodeHandle      `json:"handle"`
	From    NodeState       `json:"from"`
	To      NodeState       `json:"to"`
}

type OutputObservation = history.OutputObservation

type LeakDiagnostic struct {
	Handle NodeHandle `json:"handle"`
	Kind   LeakKind   `json:"kind"`
}

type NetworkEndpoint = history.NetworkEndpoint

type NetworkTransition = history.NetworkTransition

type NetworkNodeSnapshot = history.NetworkNodeSnapshot

type NetworkLinkSnapshot = history.NetworkLinkSnapshot

type NetworkListenerSnapshot = history.NetworkListenerSnapshot

type NetworkConnectionSnapshot = history.NetworkConnectionSnapshot

type NetworkSnapshot = history.NetworkSnapshot

type NetworkRecord = history.NetworkRecord

type VolumeTransition = history.VolumeTransition

type VolumeEntrySnapshot = history.VolumeEntrySnapshot

type VolumeStateSnapshot = history.VolumeStateSnapshot

type VolumeSnapshot = history.VolumeSnapshot

type VolumeRecord = history.VolumeRecord

type VolumeCrashEnumerationLimits = history.VolumeCrashEnumerationLimits

type VolumeCrashFrontier = history.VolumeCrashFrontier

type VolumeCrashEntry = history.VolumeCrashEntry

type VolumeCrashState = history.VolumeCrashState

type VolumeCrashEnumeration = history.VolumeCrashEnumeration

type ClusterStaticIdentities = history.ClusterStaticIdentities

type ClusterModelIdentities = history.ClusterModelIdentities

type FaultRealization struct {
	Ordinal  uint64      `json:"ordinal"`
	Action   FaultAction `json:"action"`
	Matched  FaultMatch  `json:"matched"`
	Target   NodeHandle  `json:"target,omitempty"`
	Identity string      `json:"identity"`
}

type Observation struct {
	Ordinal    uint64     `json:"ordinal"`
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Handle     NodeHandle `json:"handle,omitempty"`
	Value      []byte     `json:"value"`
	FullSHA256 string     `json:"full_sha256"`
	Identity   string     `json:"identity"`
}

type BackendUnavailableError struct {
	Backend Backend
}

func (err *BackendUnavailableError) Error() string {
	return ErrBackendUnavailable.Error() + ": " + string(err.Backend)
}

func (err *BackendUnavailableError) Unwrap() error {
	return ErrBackendUnavailable
}

type ReplayDivergence struct {
	Dimension                   ReplayDimension      `json:"dimension"`
	Ordinal                     uint64               `json:"ordinal"`
	Expected                    LifecycleTransition  `json:"expected"`
	Actual                      LifecycleTransition  `json:"actual"`
	ExpectedSHA256              string               `json:"expected_sha256,omitempty"`
	ActualSHA256                string               `json:"actual_sha256,omitempty"`
	ExpectedNetwork             *NetworkTransition   `json:"expected_network,omitempty"`
	ActualNetwork               *NetworkTransition   `json:"actual_network,omitempty"`
	ExpectedVolume              *VolumeTransition    `json:"expected_volume,omitempty"`
	ActualVolume                *VolumeTransition    `json:"actual_volume,omitempty"`
	ExpectedFault               *FaultAction         `json:"expected_fault,omitempty"`
	ActualFault                 *FaultAction         `json:"actual_fault,omitempty"`
	ExpectedScenario            *ScenarioDecision    `json:"expected_scenario,omitempty"`
	ActualScenario              *ScenarioDecision    `json:"actual_scenario,omitempty"`
	ExpectedExplorationOverride *ExplorationOverride `json:"expected_exploration_override,omitempty"`
	ExpectedExploration         *ExplorationDecision `json:"expected_exploration,omitempty"`
	ActualExploration           *ExplorationDecision `json:"actual_exploration,omitempty"`
}

type ReplayDivergenceError struct {
	Divergence ReplayDivergence
}

func (err *ReplayDivergenceError) Error() string {
	return fmt.Sprintf("%v: dimension=%s ordinal=%d expected_sha256=%s actual_sha256=%s expected_network=%+v actual_network=%+v expected_volume=%+v actual_volume=%+v expected_fault=%+v actual_fault=%+v expected_scenario=%+v actual_scenario=%+v expected_exploration_override=%+v expected_exploration=%+v actual_exploration=%+v", ErrReplayDiverged, err.Divergence.Dimension, err.Divergence.Ordinal, err.Divergence.ExpectedSHA256, err.Divergence.ActualSHA256, err.Divergence.ExpectedNetwork, err.Divergence.ActualNetwork, err.Divergence.ExpectedVolume, err.Divergence.ActualVolume, err.Divergence.ExpectedFault, err.Divergence.ActualFault, err.Divergence.ExpectedScenario, err.Divergence.ActualScenario, err.Divergence.ExpectedExplorationOverride, err.Divergence.ExpectedExploration, err.Divergence.ActualExploration)
}

func (err *ReplayDivergenceError) Unwrap() error {
	return ErrReplayDiverged
}

type ClusterRecord struct {
	Schema               string                  `json:"schema"`
	Backend              Backend                 `json:"backend"`
	Fidelity             Fidelity                `json:"fidelity"`
	Seed                 uint64                  `json:"seed"`
	Limits               Limits                  `json:"limits"`
	Static               ClusterStaticIdentities `json:"static_identities"`
	Models               ClusterModelIdentities  `json:"model_identities"`
	NodeSpecs            []NodeSpec              `json:"node_specs"`
	LinkSpecs            []LinkSpec              `json:"link_specs"`
	VolumeSpecs          []VolumeSpec            `json:"volume_specs"`
	SpecSHA256           string                  `json:"spec_sha256"`
	Outcome              Outcome                 `json:"outcome"`
	Reason               string                  `json:"reason,omitempty"`
	FailureIdentity      string                  `json:"failure_identity,omitempty"`
	Nodes                []NodeResult            `json:"nodes"`
	Transitions          []LifecycleTransition   `json:"transitions"`
	FaultPlan            FaultPlan               `json:"fault_plan"`
	Faults               []FaultRealization      `json:"faults"`
	ExplorationPlan      *ExplorationPlan        `json:"exploration_plan,omitempty"`
	ExplorationDecisions []ExplorationDecision   `json:"exploration_decisions,omitempty"`
	ScenarioChoices      ScenarioChoicePlan      `json:"scenario_choice_plan"`
	Scenarios            []ScenarioDecision      `json:"scenario_tape"`
	History              []HistoryOperation      `json:"history"`
	Observations         []Observation           `json:"observations"`
	Oracles              []OracleResult          `json:"oracles"`
	Network              NetworkRecord           `json:"network"`
	Volumes              VolumeRecord            `json:"volumes"`
	WASM                 *WASMExecutionEvidence  `json:"wasm_execution,omitempty"`
	Outputs              []OutputObservation     `json:"outputs"`
	Limitations          []Limitation            `json:"limitations"`
	Leaks                []LeakDiagnostic        `json:"leaks"`
	Divergence           *ReplayDivergence       `json:"divergence,omitempty"`
	Identity             string                  `json:"identity"`
}

type ReplayPlan struct {
	Schema               string                  `json:"schema"`
	SpecSHA256           string                  `json:"spec_sha256"`
	Static               ClusterStaticIdentities `json:"static_identities"`
	Models               ClusterModelIdentities  `json:"model_identities"`
	Outcome              Outcome                 `json:"outcome"`
	Reason               string                  `json:"reason,omitempty"`
	FailureIdentity      string                  `json:"failure_identity,omitempty"`
	Nodes                []NodeResult            `json:"nodes"`
	Transitions          []LifecycleTransition   `json:"transitions"`
	FaultPlan            FaultPlan               `json:"fault_plan"`
	Faults               []FaultRealization      `json:"faults"`
	ExplorationPlan      *ExplorationPlan        `json:"exploration_plan,omitempty"`
	ExplorationDecisions []ExplorationDecision   `json:"exploration_decisions,omitempty"`
	ScenarioChoices      ScenarioChoicePlan      `json:"scenario_choice_plan"`
	Scenarios            []ScenarioDecision      `json:"scenario_tape"`
	History              []HistoryOperation      `json:"history"`
	Observations         []Observation           `json:"observations"`
	Oracles              []OracleResult          `json:"oracles"`
	Network              NetworkRecord           `json:"network"`
	Volumes              VolumeRecord            `json:"volumes"`
	WASM                 *WASMExecutionEvidence  `json:"wasm_execution,omitempty"`
	Outputs              []OutputObservation     `json:"outputs"`
	Leaks                []LeakDiagnostic        `json:"leaks"`
	Identity             string                  `json:"identity"`
}

type Result struct {
	Outcome         Outcome                `json:"outcome"`
	Reason          string                 `json:"reason,omitempty"`
	FailureIdentity string                 `json:"failure_identity,omitempty"`
	Nodes           []NodeResult           `json:"nodes"`
	Outputs         []OutputObservation    `json:"outputs"`
	Leaks           []LeakDiagnostic       `json:"leaks"`
	Network         NetworkRecord          `json:"network"`
	Volumes         VolumeRecord           `json:"volumes"`
	WASM            *WASMExecutionEvidence `json:"wasm_execution,omitempty"`
	Faults          []FaultRealization     `json:"faults"`
	Scenarios       []ScenarioDecision     `json:"scenario_tape"`
	History         []HistoryOperation     `json:"history"`
	Observations    []Observation          `json:"observations"`
	Oracles         []OracleResult         `json:"oracles"`
	Divergence      *ReplayDivergence      `json:"divergence,omitempty"`
	Record          ClusterRecord          `json:"record"`
}

type NodeContext = history.NodeContext

type BootFunc func(context.Context, NodeContext) error

type Cluster interface {
	Start(context.Context, NodeID) (NodeHandle, error)
	Wait(context.Context, NodeHandle) (NodeResult, error)
	Stop(context.Context, NodeHandle) error
	Crash(context.Context, NodeHandle) error
	Restart(context.Context, NodeID) (NodeHandle, error)
	Partition(context.Context, NodeID, NodeID) error
	Heal(context.Context, NodeID, NodeID) error
	SetDelay(context.Context, NodeID, NodeID, uint64) error
	ApplyFault(context.Context, FaultAction) (FaultRealization, error)
	TriggerFault(context.Context, FaultMatch) (FaultRealization, bool, error)
	Observe(context.Context, Observation) error
	RecordOperation(context.Context, HistoryOperation) error
	RecordOracle(context.Context, OracleResult) error
	EnumerateCrashStates(context.Context, NodeHandle, VolumeID, VolumeCrashEnumerationLimits, *VolumeCrashFrontier) (VolumeCrashEnumeration, error)
}

type Scenario func(context.Context, Cluster) error
