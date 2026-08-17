package gomadv3sim

import "context"

const SpecSchema = "gomadv3.simulation-spec/v1"

const (
	BackendInProcess Backend = "in_process"
	BackendProcess   Backend = "process"
)

const (
	FidelitySimulationModel Fidelity = "simulation_model"
	FidelityHardIsolation   Fidelity = "hard_isolation"
)

const (
	NodeStateRunning NodeState = "running"
	NodeStateExited  NodeState = "exited"
	NodeStateStopped NodeState = "stopped"
	NodeStateCrashed NodeState = "crashed"
)

type Backend string
type Fidelity string
type BootID string
type NodeID string
type VolumeID string
type NodeState string

type NodeHandle struct {
	Node        NodeID `json:"node"`
	Incarnation uint64 `json:"incarnation"`
}

type NodeResult struct {
	Handle NodeHandle `json:"handle"`
	State  NodeState  `json:"state"`
	Reason string     `json:"reason,omitempty"`
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
}

type Scenario func(context.Context, Cluster) error
