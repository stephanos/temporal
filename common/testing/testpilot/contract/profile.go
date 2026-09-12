package contract

import testpilotspb "go.temporal.io/server/api/testpilot/v1"

type Opcode uint8

const (
	InvokeRPC Opcode = iota + 1
	AwaitSlot
	CompleteNexusOperation
	StartNexusOperation
	Await
	Finish
	RespondNexus
	InjectFault
)

// MaxOpcode is the highest declared capability. A Profile authorizes each capability at most
// once, so it is also the ceiling on an authorized capability list; Driver profile validation
// reuses it rather than restating a literal a new instruction would silently invalidate.
const MaxOpcode = InjectFault

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
	Kind         testpilotspb.EntrypointKind
	MaximumCount int64
}

type EnvironmentBinding struct {
	ID    string
	Value string
}
