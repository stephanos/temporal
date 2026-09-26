package contract

import testpilotspb "go.temporal.io/server/api/testpilot/v1"

type Opcode uint8

const (
	InvokeRPC Opcode = iota + 1
	AwaitSlot
	Await
	Finish
	InjectFault
	WorkflowCommand
	NexusHandlerReply
	NexusOperationCompletion
	ReadEvidence
)

// MaxOpcode is the highest declared Opcode. A Profile authorizes each Opcode at most
// once, so it is also the ceiling on an authorized Opcode list; Driver profile validation
// reuses it rather than restating a literal a new instruction would silently invalidate.
const MaxOpcode = ReadEvidence

// EntrypointKind classifies an Entrypoint by its activation. The protocol carries no kind: the
// activation oneof is the one source, read by EntrypointKindOf.
type EntrypointKind uint8

const (
	ControllerEntrypoint EntrypointKind = iota + 1
	WorkflowEntrypoint
	ActivityEntrypoint
	NexusHandlerEntrypoint
)

// MaxEntrypointKind is the highest declared EntrypointKind, so a caller ranging over every kind
// needs no literal a new activation would silently invalidate.
const MaxEntrypointKind = NexusHandlerEntrypoint

// EntrypointKindOf classifies an Entrypoint by its activation oneof, and returns zero when the
// entrypoint has no known activation.
func EntrypointKindOf(entrypoint *testpilotspb.Entrypoint) EntrypointKind {
	switch entrypoint.GetActivation().(type) {
	case *testpilotspb.Entrypoint_Controller:
		return ControllerEntrypoint
	case *testpilotspb.Entrypoint_Workflow:
		return WorkflowEntrypoint
	case *testpilotspb.Entrypoint_Activity:
		return ActivityEntrypoint
	case *testpilotspb.Entrypoint_NexusHandler:
		return NexusHandlerEntrypoint
	default:
		return 0
	}
}

type RolePolicy struct {
	ID                  string
	Kind                testpilotspb.RoleKind
	Methods             []string
	ReservationCarriers []ReservationCarrierPolicy
}

type ReservationCarrierPolicy struct {
	Method string
	Shapes []ReservationCarrierShape
}

type ReservationCarrierShape struct {
	Kind         EntrypointKind
	MaximumCount int64
}

// InstructionDefaults are the limits an instruction takes where its Case writes none, each within the
// Profile's ceilings. A zero default supplies nothing, so an instruction that omits that limit is
// refused.
type InstructionDefaults struct {
	TimeoutMilliseconds int64
	MaxAttempts         int64
}

// Resolve returns an instruction's limits: each one limits writes, or the default where it writes
// none. A zero result is a limit neither supplies.
func (d InstructionDefaults) Resolve(limits *testpilotspb.InstructionLimits) (timeoutMilliseconds, maxAttempts int64) {
	timeoutMilliseconds, maxAttempts = d.TimeoutMilliseconds, d.MaxAttempts
	if limits.GetTimeout() != nil {
		timeoutMilliseconds = limits.GetTimeoutMilliseconds()
	}
	if limits.GetAttempts() != nil {
		maxAttempts = limits.GetMaxAttempts()
	}
	return timeoutMilliseconds, maxAttempts
}

type EnvironmentBinding struct {
	ID    string
	Value string
}
