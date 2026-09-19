package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
)

// The Driver-facing vocabulary lives in the contract leaf, which private execution shares; these
// aliases keep every Driver written against the facade compiling unchanged.
type (
	DriverIdentity           = contract.DriverIdentity
	Coordinate               = contract.Coordinate
	ReservationIdentity      = contract.ReservationIdentity
	ReservationRequest       = contract.ReservationRequest
	OpaqueCapability         = contract.OpaqueCapability
	EffectResult             = contract.EffectResult
	CapabilityEffect         = contract.CapabilityEffect
	EffectHandle             = contract.EffectHandle
	ReservationHandle        = contract.ReservationHandle
	CapabilityBridge         = contract.CapabilityBridge
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
	EntrypointKind           = contract.EntrypointKind
)

const (
	SlotReference    = contract.SlotReference
	OutcomeReference = contract.OutcomeReference
)

const (
	InvokeRPC                = contract.InvokeRPC
	AwaitSlot                = contract.AwaitSlot
	CompleteNexusOperation   = contract.CompleteNexusOperation
	StartNexusOperation      = contract.StartNexusOperation
	Await                    = contract.Await
	Finish                   = contract.Finish
	RespondNexus             = contract.RespondNexus
	InjectFault              = contract.InjectFault
	WorkflowCommand          = contract.WorkflowCommand
	NexusHandlerReply        = contract.NexusHandlerReply
	NexusOperationCompletion = contract.NexusOperationCompletion
	MaxOpcode                = contract.MaxOpcode
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
func EntrypointKindOf(entrypoint *testpilotspb.Entrypoint) EntrypointKind {
	return contract.EntrypointKindOf(entrypoint)
}
