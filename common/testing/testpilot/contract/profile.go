package contract

import (
	"math"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	pbduration "go.temporal.io/server/common/testing/testpilot/duration"
	"google.golang.org/protobuf/proto"
)

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
	ActivityAttemptFailure
	ActivityAttemptCancellation
	ActivityAttemptWithholding
	ActivityHeartbeat
)

// MaxOpcode is the highest declared Opcode. A Profile authorizes each Opcode at most
// once, so it is also the ceiling on an authorized Opcode list; Driver profile validation
// reuses it rather than restating a literal a new instruction would silently invalidate.
const MaxOpcode = ActivityHeartbeat

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
func (d InstructionDefaults) Resolve(limits *testpilotspb.InstructionLimits) (timeoutMilliseconds, maxAttempts int64, err error) {
	timeoutMilliseconds, maxAttempts = d.TimeoutMilliseconds, d.MaxAttempts
	if limits.GetTimeout() != nil {
		timeoutMilliseconds, err = pbduration.Milliseconds("instruction.limits.timeout", limits.GetTimeout())
		if err != nil {
			return 0, 0, err
		}
	}
	if limits != nil && limits.MaxAttempts != nil {
		maxAttempts = limits.GetMaxAttempts()
	}
	return timeoutMilliseconds, maxAttempts, nil
}

// BoundScale is how much more time than declared an environment needs, in percent. It scales every
// wait hint's bound and the Profile's duration ceilings together, so a scaled bound fits its scaled
// ceiling exactly when the declared bound fits the declared one. Zero applies them as declared, as
// 100 does. Preparation refuses a scale below 100: a bound is an at-most, and shrinking it only
// makes a wait fail sooner.
type BoundScale int64

// Percent is the scale in percent, 100 when unset.
func (s BoundScale) Percent() int64 {
	if s == 0 {
		return 100
	}
	return int64(s)
}

// Scaled reports whether the scale changes a bound.
func (s BoundScale) Scaled() bool { return s.Percent() != 100 }

// Apply scales a bound in milliseconds, rounding up and saturating, so a scaled bound is never
// shorter than the declared one asks for.
func (s BoundScale) Apply(milliseconds int64) int64 {
	percent := s.Percent()
	if percent == 100 || milliseconds <= 0 {
		return milliseconds
	}
	if milliseconds > (math.MaxInt64-99)/percent {
		return math.MaxInt64
	}
	return (milliseconds*percent + 99) / 100
}

// Ceilings returns limits with its total and cleanup duration ceilings scaled: limits itself when
// the scale changes nothing, and a copy otherwise.
func (s BoundScale) Ceilings(limits *testpilotspb.ProgramLimits) *testpilotspb.ProgramLimits {
	if !s.Scaled() || limits == nil {
		return limits
	}
	scaled := proto.CloneOf(limits)
	maxDuration, maxErr := pbduration.Milliseconds("max_duration", limits.MaxDuration)
	cleanupDuration, cleanupErr := pbduration.Milliseconds("cleanup_duration", limits.CleanupDuration)
	if maxErr != nil || cleanupErr != nil {
		return scaled
	}
	scaled.MaxDuration = pbduration.FromMilliseconds(s.Apply(maxDuration))
	scaled.CleanupDuration = pbduration.FromMilliseconds(s.Apply(cleanupDuration))
	return scaled
}

type EnvironmentBinding struct {
	ID    string
	Value string
}
