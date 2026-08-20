package gomadv3sim

import (
	"context"
	"errors"
	"fmt"
)

const SpecSchema = "gomadv3.simulation-spec/v6"
const ClusterRecordSchema = "gomadv3.cluster-record/v6"
const ClusterReplaySchema = "gomadv3.cluster-replay/v6"

const MaximumClusterRecordBytes = 128 << 20
const MaximumTerminalReasonBytes = 4096

const (
	BackendInProcess Backend = "in_process"
	BackendProcess   Backend = "process"
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
	ReplayDimensionTransition ReplayDimension = "transition"
	ReplayDimensionNetwork    ReplayDimension = "network"
	ReplayDimensionVolume     ReplayDimension = "volume"
	ReplayDimensionFault      ReplayDimension = "fault"
	ReplayDimensionScenario   ReplayDimension = "scenario"
	ReplayDimensionEvidence   ReplayDimension = "evidence"
	ReplayDimensionTerminal   ReplayDimension = "terminal"
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
type BootID string
type NodeID string
type VolumeID string
type NodeState string
type LifecycleAction string
type ReplayDimension string
type Outcome string
type OutputStream string
type Limitation string
type LeakKind string
type NetworkTransitionKind string
type NetworkOutcome string
type VolumeTransitionKind string
type VolumeOutcome string
type VolumeCrashCapacity string

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
	NetworkDeliver          NetworkTransitionKind = "deliver"
	NetworkClose            NetworkTransitionKind = "close"
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
	NetworkOutcomeOK            NetworkOutcome = "ok"
	NetworkOutcomeClosed        NetworkOutcome = "closed"
	NetworkOutcomeRefused       NetworkOutcome = "refused"
	NetworkOutcomeDeadline      NetworkOutcome = "deadline"
	NetworkOutcomePartitionDrop NetworkOutcome = "partition_drop"
	NetworkOutcomeStaleDrop     NetworkOutcome = "stale_drop"
	NetworkOutcomeReset         NetworkOutcome = "reset"
	NetworkOutcomeCapacity      NetworkOutcome = "capacity"
	NetworkOutcomeUnsupported   NetworkOutcome = "unsupported"
)

type NodeHandle struct {
	Node        NodeID `json:"node"`
	Incarnation uint64 `json:"incarnation"`
}

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

type OutputObservation struct {
	Handle         NodeHandle   `json:"handle"`
	Stream         OutputStream `json:"stream"`
	Bytes          []byte       `json:"bytes"`
	FullSHA256     string       `json:"full_sha256"`
	TotalBytes     uint64       `json:"total_bytes"`
	RetainedBytes  uint64       `json:"retained_bytes"`
	DiscardedBytes uint64       `json:"discarded_bytes"`
	Truncated      bool         `json:"truncated"`
}

type LeakDiagnostic struct {
	Handle NodeHandle `json:"handle"`
	Kind   LeakKind   `json:"kind"`
}

type NetworkEndpoint struct {
	Node        NodeID `json:"node,omitempty"`
	Incarnation uint64 `json:"incarnation,omitempty"`
	Address     string `json:"address,omitempty"`
	Port        uint64 `json:"port,omitempty"`
}

type NetworkTransition struct {
	Ordinal       uint64                `json:"ordinal"`
	Kind          NetworkTransitionKind `json:"kind"`
	Source        NetworkEndpoint       `json:"source,omitempty"`
	Destination   NetworkEndpoint       `json:"destination,omitempty"`
	Connection    uint64                `json:"connection,omitempty"`
	Delivery      uint64                `json:"delivery,omitempty"`
	Bytes         uint64                `json:"bytes,omitempty"`
	Count         uint64                `json:"count,omitempty"`
	DelayNanos    uint64                `json:"delay_nanos,omitempty"`
	Outcome       NetworkOutcome        `json:"outcome"`
	PayloadSHA256 string                `json:"payload_sha256,omitempty"`
}

type NetworkNodeSnapshot struct {
	Node             NodeID `json:"node"`
	Address          string `json:"address"`
	LastIncarnation  uint64 `json:"last_incarnation"`
	NextListenerPort uint64 `json:"next_listener_port"`
	NextClientPort   uint64 `json:"next_client_port"`
}

type NetworkLinkSnapshot struct {
	From       NodeID `json:"from"`
	To         NodeID `json:"to"`
	Enabled    bool   `json:"enabled"`
	DelayNanos uint64 `json:"delay_nanos"`
}

type NetworkListenerSnapshot struct {
	Endpoint NetworkEndpoint `json:"endpoint"`
	Closed   bool            `json:"closed"`
}

type NetworkConnectionSnapshot struct {
	Identity uint64          `json:"identity"`
	Client   NetworkEndpoint `json:"client"`
	Server   NetworkEndpoint `json:"server"`
	Closed   bool            `json:"closed"`
	Reset    bool            `json:"reset"`
}

type NetworkDeliverySnapshot struct {
	Identity    uint64          `json:"identity"`
	Connection  uint64          `json:"connection"`
	Source      NetworkEndpoint `json:"source"`
	Destination NetworkEndpoint `json:"destination"`
	Bytes       uint64          `json:"bytes"`
	DelayNanos  uint64          `json:"delay_nanos"`
}

type NetworkSnapshot struct {
	Nodes            []NetworkNodeSnapshot       `json:"nodes"`
	Links            []NetworkLinkSnapshot       `json:"links"`
	Listeners        []NetworkListenerSnapshot   `json:"listeners"`
	Connections      []NetworkConnectionSnapshot `json:"connections"`
	Deliveries       []NetworkDeliverySnapshot   `json:"deliveries"`
	NextConnection   uint64                      `json:"next_connection"`
	NextDelivery     uint64                      `json:"next_delivery"`
	TransitionSHA256 string                      `json:"transition_sha256"`
	Identity         string                      `json:"identity"`
}

type NetworkRecord struct {
	Transitions []NetworkTransition `json:"transitions"`
	Snapshot    NetworkSnapshot     `json:"snapshot"`
}

type VolumeTransition struct {
	Ordinal            uint64               `json:"ordinal"`
	Kind               VolumeTransitionKind `json:"kind"`
	Handle             NodeHandle           `json:"handle"`
	Volume             VolumeID             `json:"volume"`
	Operation          uint64               `json:"operation,omitempty"`
	Dependencies       []uint64             `json:"dependencies,omitempty"`
	SelectedOperations []uint64             `json:"selected_operations,omitempty"`
	Path               string               `json:"path,omitempty"`
	Destination        string               `json:"destination,omitempty"`
	Inode              uint64               `json:"inode,omitempty"`
	Offset             uint64               `json:"offset,omitempty"`
	Bytes              uint64               `json:"bytes,omitempty"`
	PayloadSHA256      string               `json:"payload_sha256,omitempty"`
	EffectSHA256       string               `json:"effect_sha256,omitempty"`
	Outcome            VolumeOutcome        `json:"outcome"`
}

type VolumeEntrySnapshot struct {
	Path       string `json:"path"`
	Mode       uint32 `json:"mode"`
	Kind       string `json:"kind"`
	ModTime    int64  `json:"mod_time"`
	Size       uint64 `json:"size"`
	DataSHA256 string `json:"data_sha256,omitempty"`
}

type VolumeStateSnapshot struct {
	Volume            VolumeID              `json:"volume"`
	Node              NodeID                `json:"node"`
	Mount             string                `json:"mount"`
	CapacityBytes     uint64                `json:"capacity_bytes"`
	Persisted         []VolumeEntrySnapshot `json:"persisted"`
	Volatile          []VolumeEntrySnapshot `json:"volatile"`
	PendingOperations uint64                `json:"pending_operations"`
	PendingSHA256     string                `json:"pending_sha256"`
	NextOperation     uint64                `json:"next_operation"`
	Identity          string                `json:"identity"`
}

type VolumeSnapshot struct {
	Volumes          []VolumeStateSnapshot `json:"volumes"`
	TransitionSHA256 string                `json:"transition_sha256"`
	Identity         string                `json:"identity"`
}

type VolumeRecord struct {
	Transitions []VolumeTransition `json:"transitions"`
	Snapshot    VolumeSnapshot     `json:"snapshot"`
}

type VolumeCrashEnumerationLimits struct {
	States     uint64 `json:"states"`
	Operations uint64 `json:"operations"`
	Depth      uint64 `json:"depth"`
	Bytes      uint64 `json:"bytes"`
	WallNanos  uint64 `json:"wall_nanos"`
}

type VolumeCrashFrontier struct {
	Volume        VolumeID `json:"volume"`
	PendingSHA256 string   `json:"pending_sha256"`
	Cursor        []byte   `json:"cursor"`
	Seen          []string `json:"seen"`
	Identity      string   `json:"identity"`
}

type VolumeCrashEntry struct {
	Path    string `json:"path"`
	Mode    uint32 `json:"mode"`
	Kind    string `json:"kind"`
	ModTime int64  `json:"mod_time"`
	Data    []byte `json:"data"`
}

type VolumeCrashState struct {
	Volume             VolumeID           `json:"volume"`
	PendingSHA256      string             `json:"pending_sha256"`
	SelectedOperations []uint64           `json:"selected_operations"`
	Entries            []VolumeCrashEntry `json:"entries"`
	Identity           string             `json:"identity"`
}

type VolumeCrashEnumeration struct {
	States   []VolumeCrashState   `json:"states"`
	Frontier *VolumeCrashFrontier `json:"frontier,omitempty"`
	Complete bool                 `json:"complete"`
	Capacity VolumeCrashCapacity  `json:"capacity,omitempty"`
}

type ClusterStaticIdentities struct {
	TargetSHA256         string `json:"target_sha256"`
	PlatformBundleSHA256 string `json:"platform_bundle_sha256"`
}

type ClusterModelIdentities struct {
	RuntimeDomainSHA256 string `json:"runtime_domain_sha256"`
	ProcessSHA256       string `json:"process_sha256"`
	NetworkSHA256       string `json:"network_sha256"`
	VolumeSHA256        string `json:"volume_sha256"`
	FaultSHA256         string `json:"fault_sha256"`
	ScenarioSHA256      string `json:"scenario_sha256"`
	OracleSHA256        string `json:"oracle_sha256"`
}

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
	Dimension        ReplayDimension     `json:"dimension"`
	Ordinal          uint64              `json:"ordinal"`
	Expected         LifecycleTransition `json:"expected"`
	Actual           LifecycleTransition `json:"actual"`
	ExpectedSHA256   string              `json:"expected_sha256,omitempty"`
	ActualSHA256     string              `json:"actual_sha256,omitempty"`
	ExpectedNetwork  *NetworkTransition  `json:"expected_network,omitempty"`
	ActualNetwork    *NetworkTransition  `json:"actual_network,omitempty"`
	ExpectedVolume   *VolumeTransition   `json:"expected_volume,omitempty"`
	ActualVolume     *VolumeTransition   `json:"actual_volume,omitempty"`
	ExpectedFault    *FaultAction        `json:"expected_fault,omitempty"`
	ActualFault      *FaultAction        `json:"actual_fault,omitempty"`
	ExpectedScenario *ScenarioDecision   `json:"expected_scenario,omitempty"`
	ActualScenario   *ScenarioDecision   `json:"actual_scenario,omitempty"`
}

type ReplayDivergenceError struct {
	Divergence ReplayDivergence
}

func (err *ReplayDivergenceError) Error() string {
	return fmt.Sprintf("%v: dimension=%s ordinal=%d expected_sha256=%s actual_sha256=%s expected_network=%+v actual_network=%+v expected_volume=%+v actual_volume=%+v expected_fault=%+v actual_fault=%+v expected_scenario=%+v actual_scenario=%+v", ErrReplayDiverged, err.Divergence.Dimension, err.Divergence.Ordinal, err.Divergence.ExpectedSHA256, err.Divergence.ActualSHA256, err.Divergence.ExpectedNetwork, err.Divergence.ActualNetwork, err.Divergence.ExpectedVolume, err.Divergence.ActualVolume, err.Divergence.ExpectedFault, err.Divergence.ActualFault, err.Divergence.ExpectedScenario, err.Divergence.ActualScenario)
}

func (err *ReplayDivergenceError) Unwrap() error {
	return ErrReplayDiverged
}

type ClusterRecord struct {
	Schema          string                  `json:"schema"`
	Backend         Backend                 `json:"backend"`
	Fidelity        Fidelity                `json:"fidelity"`
	Seed            uint64                  `json:"seed"`
	Limits          Limits                  `json:"limits"`
	Static          ClusterStaticIdentities `json:"static_identities"`
	Models          ClusterModelIdentities  `json:"model_identities"`
	NodeSpecs       []NodeSpec              `json:"node_specs"`
	LinkSpecs       []LinkSpec              `json:"link_specs"`
	VolumeSpecs     []VolumeSpec            `json:"volume_specs"`
	SpecSHA256      string                  `json:"spec_sha256"`
	Outcome         Outcome                 `json:"outcome"`
	Reason          string                  `json:"reason,omitempty"`
	FailureIdentity string                  `json:"failure_identity,omitempty"`
	Nodes           []NodeResult            `json:"nodes"`
	Transitions     []LifecycleTransition   `json:"transitions"`
	FaultPlan       FaultPlan               `json:"fault_plan"`
	Faults          []FaultRealization      `json:"faults"`
	ScenarioChoices ScenarioChoicePlan      `json:"scenario_choice_plan"`
	Scenarios       []ScenarioDecision      `json:"scenario_tape"`
	History         []HistoryOperation      `json:"history"`
	Observations    []Observation           `json:"observations"`
	Oracles         []OracleResult          `json:"oracles"`
	Network         NetworkRecord           `json:"network"`
	Volumes         VolumeRecord            `json:"volumes"`
	Outputs         []OutputObservation     `json:"outputs"`
	Limitations     []Limitation            `json:"limitations"`
	Leaks           []LeakDiagnostic        `json:"leaks"`
	Divergence      *ReplayDivergence       `json:"divergence,omitempty"`
	Identity        string                  `json:"identity"`
}

type ReplayPlan struct {
	Schema          string                  `json:"schema"`
	SpecSHA256      string                  `json:"spec_sha256"`
	Static          ClusterStaticIdentities `json:"static_identities"`
	Models          ClusterModelIdentities  `json:"model_identities"`
	Outcome         Outcome                 `json:"outcome"`
	Reason          string                  `json:"reason,omitempty"`
	FailureIdentity string                  `json:"failure_identity,omitempty"`
	Nodes           []NodeResult            `json:"nodes"`
	Transitions     []LifecycleTransition   `json:"transitions"`
	FaultPlan       FaultPlan               `json:"fault_plan"`
	Faults          []FaultRealization      `json:"faults"`
	ScenarioChoices ScenarioChoicePlan      `json:"scenario_choice_plan"`
	Scenarios       []ScenarioDecision      `json:"scenario_tape"`
	History         []HistoryOperation      `json:"history"`
	Observations    []Observation           `json:"observations"`
	Oracles         []OracleResult          `json:"oracles"`
	Network         NetworkRecord           `json:"network"`
	Volumes         VolumeRecord            `json:"volumes"`
	Outputs         []OutputObservation     `json:"outputs"`
	Leaks           []LeakDiagnostic        `json:"leaks"`
	Identity        string                  `json:"identity"`
}

type Result struct {
	Outcome         Outcome             `json:"outcome"`
	Reason          string              `json:"reason,omitempty"`
	FailureIdentity string              `json:"failure_identity,omitempty"`
	Nodes           []NodeResult        `json:"nodes"`
	Outputs         []OutputObservation `json:"outputs"`
	Leaks           []LeakDiagnostic    `json:"leaks"`
	Network         NetworkRecord       `json:"network"`
	Volumes         VolumeRecord        `json:"volumes"`
	Faults          []FaultRealization  `json:"faults"`
	Scenarios       []ScenarioDecision  `json:"scenario_tape"`
	History         []HistoryOperation  `json:"history"`
	Observations    []Observation       `json:"observations"`
	Oracles         []OracleResult      `json:"oracles"`
	Divergence      *ReplayDivergence   `json:"divergence,omitempty"`
	Record          ClusterRecord       `json:"record"`
}

type NodeContext struct {
	NodeHandle
	Address string `json:"address"`
	Config  []byte `json:"config,omitempty"`
}

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
