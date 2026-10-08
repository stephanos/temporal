package testpilot

import "go.temporal.io/server/common/testing/testpilot/contract"

// The Driver-facing vocabulary lives in the contract leaf, which private execution shares; these
// aliases keep every Driver written against the facade compiling unchanged.
type (
	DriverIdentity           = contract.DriverIdentity
	Coordinate               = contract.Coordinate
	ReservationIdentity      = contract.ReservationIdentity
	ReservationRequest       = contract.ReservationRequest
	OpaqueHandle             = contract.OpaqueHandle
	PollPredicate            = contract.PollPredicate
	EffectResult             = contract.EffectResult
	HandleEffect             = contract.HandleEffect
	EffectHandle             = contract.EffectHandle
	ReservationHandle        = contract.ReservationHandle
	HandleBridge             = contract.HandleBridge
	Session                  = contract.Session
	PreparedRole             = contract.PreparedRole
	ReferenceKind            = contract.ReferenceKind
	ValueReference           = contract.ValueReference
	OutcomeSnapshot          = contract.OutcomeSnapshot
	ReservationCarrierPlan   = contract.ReservationCarrierPlan
	ReservationTopology      = contract.ReservationTopology
	ReservationRoute         = contract.ReservationRoute
	Opcode                   = contract.Opcode
	RolePolicy               = contract.RolePolicy
	ReservationCarrierPolicy = contract.ReservationCarrierPolicy
	ReservationCarrierShape  = contract.ReservationCarrierShape
	EnvironmentBinding       = contract.EnvironmentBinding
	InstructionDefaults      = contract.InstructionDefaults
	BoundScale               = contract.BoundScale
	EntrypointKind           = contract.EntrypointKind
)

const (
	SlotReference    = contract.SlotReference
	OutcomeReference = contract.OutcomeReference
)

const (
	InvokeRPC                   = contract.InvokeRPC
	AwaitSlot                   = contract.AwaitSlot
	Await                       = contract.Await
	Finish                      = contract.Finish
	InjectFault                 = contract.InjectFault
	WorkflowCommand             = contract.WorkflowCommand
	NexusHandlerReply           = contract.NexusHandlerReply
	NexusOperationCompletion    = contract.NexusOperationCompletion
	ReadEvidence                = contract.ReadEvidence
	ActivityAttemptFailure      = contract.ActivityAttemptFailure
	ActivityAttemptCancellation = contract.ActivityAttemptCancellation
	ActivityAttemptWithholding  = contract.ActivityAttemptWithholding
	ActivityHeartbeat           = contract.ActivityHeartbeat
	MaxOpcode                   = contract.MaxOpcode
)

const (
	ControllerEntrypoint   = contract.ControllerEntrypoint
	WorkflowEntrypoint     = contract.WorkflowEntrypoint
	ActivityEntrypoint     = contract.ActivityEntrypoint
	NexusHandlerEntrypoint = contract.NexusHandlerEntrypoint
	MaxEntrypointKind      = contract.MaxEntrypointKind
)

// EntrypointKindOf classifies an Entrypoint by its activation oneof, and returns zero when the
// entrypoint has no known activation.
var EntrypointKindOf = contract.EntrypointKindOf
