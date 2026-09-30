// authoring: header

// Package nexuscaller is the Nexus caller-side Model.
//
// One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
// operation does, the protocol machine says how the server gets there and refines it, and the
// functional set runs one Query per side effect that settles the operation, once per value of the
// implementation switch. `DESIGN.md` section 3 is this file's specimen, written in the landed grammar:
// step functions and predicates rather than rows, no cancellation (fn-79) and no concurrency-limit
// setup parameter (`UMPIRE4_RESEARCH_NEXUS_MODEL.md` section 1).
//
// The regions `AUTHORING.md` quotes are marked `// authoring: <name>`; a region runs to the next
// marker. The drift test reads the markers, so a quoted block and the Model cannot part.
//
// Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
package nexuscaller

import (
	"fmt"

	commandpb "go.temporal.io/api/command/v1"
	nexuspb "go.temporal.io/api/nexus/v1"

	"go.temporal.io/umpire/model/common"
	"go.temporal.io/umpire/model/umpire"
	"go.temporal.io/umpire/model/worker"
)

// finite emits Values and String for each integer enum, with the type prefix trimmed and the first
// letter lowered, so row keys read `scheduled` and `nexusOperationCompleted` as the Lean tables do.
//go:generate go run ../umpire/cmd/finite -type=Resolution,ProductPhase,ProductOutcome,ProductFact,Phase,TimeoutType,ProtocolOutcome

// authoring: entities

// ### Entities
//
// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled event:
// every history event of the operation carries that event's id.

var workflow = &umpire.Entity{Name: "workflow"}

var operation = &umpire.Entity{
	Name:  "operation",
	Refer: map[string]*umpire.Entity{"caller": workflow},
	Key:   "scheduledEvent",
}

// authoring: domains

// ### The input domains
//
// A class is one member of a domain, and a constructor that carries finite fields contributes one
// class per assignment of them: `HandlerError{Retryable bool}` is one constructor and two classes,
// which is the granularity an example is written at and what mirrors a protobuf oneof.
//
// Timeout and Delivery are common.Timeout and common.Delivery: both Models use them.

// Reply is a sealed interface: the marker method keeps other packages from adding variants, and
// go-check-sumtype reads the `sumtype:decl` line to check every type switch over it.
//
//sumtype:decl
type Reply interface{ isReply() }

type (
	SyncSuccess       struct{}
	Async             struct{}
	OperationFailed   struct{}
	OperationCanceled struct{}
	HandlerError      struct{ Retryable bool }
)

func (SyncSuccess) isReply()       {}
func (Async) isReply()             {}
func (OperationFailed) isReply()   {}
func (OperationCanceled) isReply() {}
func (HandlerError) isReply()      {}

// replies lists the variants by hand, since Go cannot enumerate an interface's implementers.
// HandlerError{} expands to both of its classes.
var replies = umpire.Sum[Reply](SyncSuccess{}, Async{}, OperationFailed{}, OperationCanceled{}, HandlerError{})

type Resolution uint8

const (
	ResolutionSucceeded Resolution = iota
	ResolutionFailed
	ResolutionCanceled
)

// authoring: actions

// ### Actions
//
// Parties are names the feature declares by using them: `caller`, `handler`, `network`, `worker`.
// The reserved party `system` is the server. A fault is an ordinary action of a declared party, and a
// timer is `system` behavior the machine owns, so neither is a separate kind.

var schedule = &umpire.Action3[common.Timeout, common.Timeout, common.Timeout]{
	Name:    "schedule",
	Party:   umpire.Caller,
	Creates: operation,
	Schema:  umpire.Schema(&commandpb.ScheduleNexusOperationCommandAttributes{}),
	Inputs:  [3]string{"scheduleToClose", "scheduleToStart", "startToClose"},
}

var handlerReply = &umpire.Action1[Reply]{
	Name:    "handlerReply",
	Party:   umpire.Handler,
	On:      operation,
	Schema:  umpire.Schema(&nexuspb.StartOperationResponse{}, &nexuspb.HandlerError{}),
	Input:   "reply",
	Classes: replies,
	Examples: map[Reply]string{
		HandlerError{Retryable: false}: "BadRequest",
		HandlerError{Retryable: true}:  "Internal",
	},
}

// The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes are
// names the realization interprets.
var complete = &umpire.Action1[Resolution]{
	Name:    "complete",
	Party:   umpire.Handler,
	On:      operation,
	Input:   "resolution",
	Results: umpire.Results[common.Delivery](),
}

var transportFault = &umpire.Action0{Name: "transportFault", Party: umpire.Network, On: operation}

// The handler's worker stops polling. An action that names no entity is behavior no entity records:
// the Run records the fault, but nothing recorded names the operation, so the machines keep their
// state and record nothing at it.
var workerStop = &umpire.Action0{Name: "workerStop", Party: umpire.Worker}

// The timers. A machine owns them under Timers, which is what makes them `system` actions.
var (
	timeout         = &umpire.Action0{Name: "timeout", Party: umpire.System}
	backoff         = &umpire.Action0{Name: "backoff", Party: umpire.System}
	scheduleToClose = &umpire.Action0{Name: "scheduleToClose", Party: umpire.System}
	scheduleToStart = &umpire.Action0{Name: "scheduleToStart", Party: umpire.System}
	startToClose    = &umpire.Action0{Name: "startToClose", Party: umpire.System}
)

// authoring: observation

// ### The derived observation
//
// A retryable attempt failure writes no history event, so the attempt count is read back through
// `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

var pendingAttempts = &umpire.Observation{Name: "pendingAttempts", On: operation, Read: "attempts"}

// authoring: product

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

type ProductPhase uint8

const (
	ProductScheduled ProductPhase = iota
	ProductStarted
	ProductSucceeded
	ProductFailed
	ProductCanceled
	ProductTimedOut
)

type ProductState struct {
	Phase ProductPhase
}

type ProductOutcome uint8

const (
	ProductAccepted ProductOutcome = iota
	ProductNotFound
)

// ProductFact is a plain enum, since no product fact carries a field. The identifiers carry a
// Product prefix because Go scopes constants to the package, not to the type.
type ProductFact uint8

const (
	ProductNexusOperationScheduled ProductFact = iota
	ProductNexusOperationStarted
	ProductNexusOperationCompleted
	ProductNexusOperationFailed
	ProductNexusOperationCanceled
	ProductNexusOperationTimedOut
)

// ProductStep and the aliases below are how a Go author avoids writing three type arguments on every
// signature; the Lean file spells `Step ProductState ProductOutcome ProductFact` each time.
type (
	ProductStep     = umpire.Step[ProductState, ProductOutcome, ProductFact]
	productProperty = umpire.Property[ProductState, ProductOutcome, ProductFact]
)

func productStep(phase ProductPhase, recorded ProductFact) []ProductStep {
	return []ProductStep{{Outcome: ProductAccepted, State: ProductState{Phase: phase}, Facts: []ProductFact{recorded}}}
}

// The handler's reply to the server's start request. An operation that has not started yet is the
// only one a reply can move.
func handlerReplyStep(state ProductState, reply Reply) []ProductStep {
	if state.Phase != ProductScheduled {
		return nil
	}
	switch reply := reply.(type) {
	case SyncSuccess:
		return productStep(ProductSucceeded, ProductNexusOperationCompleted)
	case Async:
		return productStep(ProductStarted, ProductNexusOperationStarted)
	case OperationFailed:
		return productStep(ProductFailed, ProductNexusOperationFailed)
	case OperationCanceled:
		return productStep(ProductCanceled, ProductNexusOperationCanceled)
	case HandlerError:
		// A retryable handler error leaves the operation where it is: the product machine does not know
		// about backing off, which is the whole of what the protocol machine adds.
		if reply.Retryable {
			return nil
		}
		return productStep(ProductFailed, ProductNexusOperationFailed)
	default:
		// go-check-sumtype proves this arm dead; the compiler still wants a terminating statement.
		panic(fmt.Sprintf("unhandled Reply %T", reply))
	}
}

// The four phases the product machine ends on.
func productTerminal(state ProductState) bool {
	return state.Phase == ProductSucceeded || state.Phase == ProductFailed || state.Phase == ProductCanceled ||
		state.Phase == ProductTimedOut
}

// An asynchronous completion. A completion that arrives after the operation is over is not found, and
// changes nothing.
func completeStep(state ProductState, resolution Resolution) []ProductStep {
	if productTerminal(state) {
		return []ProductStep{{Outcome: ProductNotFound, State: state}}
	}
	switch resolution {
	case ResolutionSucceeded:
		return productStep(ProductSucceeded, ProductNexusOperationCompleted)
	case ResolutionFailed:
		return productStep(ProductFailed, ProductNexusOperationFailed)
	case ResolutionCanceled:
		return productStep(ProductCanceled, ProductNexusOperationCanceled)
	default:
		panic(fmt.Sprintf("unhandled Resolution %v", resolution))
	}
}

// A transport fault is an ordinary action of the network. The product machine cannot see one: whether
// a delivery was retried is the protocol's account of how, not what.
func transportFaultStep(ProductState) []ProductStep { return nil }

// The handler's worker stopping is a fault the Run records and the operation does not feel. The
// product machine cannot see it, like the transport fault: a step that kept the state and recorded
// nothing would be indistinguishable from a stutter, and the refinement would read every stutter as
// this step.
func workerStopStep(ProductState) []ProductStep { return nil }

// One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
// product machine has one timer, and it fires while the operation runs.
func timeoutStep(state ProductState) []ProductStep {
	if state.Phase == ProductScheduled || state.Phase == ProductStarted {
		return productStep(ProductTimedOut, ProductNexusOperationTimedOut)
	}
	return nil
}

var NexusProduct = &umpire.Machine[ProductState, ProductOutcome, ProductFact]{
	Name:   "nexusProduct",
	For:    operation,
	States: umpire.Fields[ProductState](),
	Starts: []ProductState{{Phase: ProductScheduled}},
	Ends:   productTerminal,
	Timers: []umpire.Action{timeout},
	Evidence: map[ProductFact]string{
		ProductNexusOperationStarted:   "nexusOperationStarted",
		ProductNexusOperationCompleted: "nexusOperationCompleted",
		ProductNexusOperationFailed:    "nexusOperationFailed",
		ProductNexusOperationCanceled:  "nexusOperationCanceled",
		ProductNexusOperationTimedOut:  "nexusOperationTimedOut",
	},
	Steps: umpire.Steps(
		umpire.Bind1(handlerReply, handlerReplyStep),
		umpire.Bind1(complete, completeStep),
		umpire.Bind0(transportFault, transportFaultStep),
		umpire.Bind0(workerStop, workerStopStep),
		umpire.Bind0(timeout, timeoutStep),
	),
}

// authoring: protocol

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the three timers the schedule
// command sets, and the attempt count a retryable failure raises. Written against the same actions,
// so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state struct has no "no instance yet" member, so
// `Unscheduled` is that member, and it is what makes the three deadline fields reachable at anything
// but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79), and the
// concurrency-limit rejection. The limit exists -- one dynamic-config key per implementation, and the
// schedule command fails the workflow task at it without writing a `NexusOperationScheduled` event --
// but a step function does not read the setup, the key and value differ per switch value, and the
// rejection names no operation, so it is not modeled until a Query needs it.

type Phase uint8

const (
	Unscheduled Phase = iota
	Scheduled
	BackingOff
	Started
	Succeeded
	Failed
	Canceled
	TimedOut
)

// TimeoutType is which timer fired. The history event records it, so a Contract that did not check it
// would pass a run that timed out on the wrong deadline.
type TimeoutType uint8

const (
	ScheduleToClose TimeoutType = iota
	ScheduleToStart
	StartToClose
)

// The attempt count is bounded by the Limits in the design; nothing wires the Limits into a machine's
// state, so the bound is written here and the saturating successor keeps a retry inside it. Lean's
// `Fin (attemptBound + 1)` makes leaving the bound a type error; Attempts is a uint8 whose Values stop
// at the bound, so Check rejects a row that leaves it when it builds the table.
const attemptBound = 2

type Attempts uint8

func (Attempts) Values() []Attempts {
	values := make([]Attempts, 0, attemptBound+1)
	for i := Attempts(0); i <= attemptBound; i++ {
		values = append(values, i)
	}
	return values
}

func (a Attempts) saturatingSucc() Attempts {
	if a < attemptBound {
		return a + 1
	}
	return a
}

type ProtocolState struct {
	Phase           Phase
	Attempts        Attempts
	ScheduleToClose common.Timeout
	ScheduleToStart common.Timeout
	StartToClose    common.Timeout
}

type ProtocolOutcome uint8

const (
	Accepted ProtocolOutcome = iota
	NotFound
)

// ProtocolFact is sealed like Reply, because one fact carries a field.
//
//sumtype:decl
type ProtocolFact interface{ isProtocolFact() }

type (
	NexusOperationScheduled struct{}
	NexusOperationStarted   struct{}
	NexusOperationCompleted struct{}
	NexusOperationFailed    struct{}
	NexusOperationCanceled  struct{}
	NexusOperationTimedOut  struct{ TimeoutType TimeoutType }
	PendingAttempts         struct{}
)

func (NexusOperationScheduled) isProtocolFact() {}
func (NexusOperationStarted) isProtocolFact()   {}
func (NexusOperationCompleted) isProtocolFact() {}
func (NexusOperationFailed) isProtocolFact()    {}
func (NexusOperationCanceled) isProtocolFact()  {}
func (NexusOperationTimedOut) isProtocolFact()  {}
func (PendingAttempts) isProtocolFact()         {}

type (
	ProtocolStep     = umpire.Step[ProtocolState, ProtocolOutcome, ProtocolFact]
	protocolProperty = umpire.Property[ProtocolState, ProtocolOutcome, ProtocolFact]
	protocolScenario = umpire.Scenario[ProtocolState, ProtocolOutcome, ProtocolFact]
)

// The four phases the design ends on. A completion that arrives after one of them is not found.
func terminalPhase(phase Phase) bool {
	return phase == Succeeded || phase == Failed || phase == Canceled || phase == TimedOut
}

// Scheduled and not yet over: the phases a completion resolves and a timer can fire in.
func running(phase Phase) bool {
	return phase == Scheduled || phase == BackingOff || phase == Started
}

// moves is Lean's `{ state with phase }`: Go copies the struct by value, so the argument is the copy.
func moves(state ProtocolState, phase Phase, recorded ...ProtocolFact) []ProtocolStep {
	state.Phase = phase
	return []ProtocolStep{{Outcome: Accepted, State: state, Facts: recorded}}
}

// The caller's schedule command. It names the operation's three deadlines, and every one of them is
// a state field because whether a timer fires is a question about the operation and not about the
// command that started it.
func scheduleStep(state ProtocolState, scheduleToClose, scheduleToStart, startToClose common.Timeout) []ProtocolStep {
	if state.Phase != Unscheduled {
		return nil
	}
	return []ProtocolStep{{
		Outcome: Accepted,
		State: ProtocolState{
			Phase:           Scheduled,
			Attempts:        0,
			ScheduleToClose: scheduleToClose,
			ScheduleToStart: scheduleToStart,
			StartToClose:    startToClose,
		},
		Facts: []ProtocolFact{NexusOperationScheduled{}},
	}}
}

// The handler's reply to the server's start request. What the product machine cannot see is the last
// arm: a retryable failure backs the operation off and raises its attempt count, and the count is read
// back through the `pendingAttempts` observation because no history event records it.
func protocolHandlerReplyStep(state ProtocolState, reply Reply) []ProtocolStep {
	if state.Phase != Scheduled {
		return nil
	}
	switch reply := reply.(type) {
	case SyncSuccess:
		return moves(state, Succeeded, NexusOperationCompleted{})
	case Async:
		return moves(state, Started, NexusOperationStarted{})
	case OperationFailed:
		return moves(state, Failed, NexusOperationFailed{})
	case OperationCanceled:
		return moves(state, Canceled, NexusOperationCanceled{})
	case HandlerError:
		if !reply.Retryable {
			return moves(state, Failed, NexusOperationFailed{})
		}
		state.Attempts = state.Attempts.saturatingSucc()
		return moves(state, BackingOff, PendingAttempts{})
	default:
		panic(fmt.Sprintf("unhandled Reply %T", reply))
	}
}

// A transport fault is the same failure arriving as a dropped delivery rather than as a reply.
func protocolTransportFaultStep(state ProtocolState) []ProtocolStep {
	if state.Phase != Scheduled {
		return nil
	}
	state.Attempts = state.Attempts.saturatingSucc()
	return moves(state, BackingOff, PendingAttempts{})
}

// The handler's worker stopping is a fault the Run records and the operation does not feel, so the
// step keeps the state and records nothing. On a path it is confirmed by the evidence of the step
// after it, and the Case says so in a Known Gap.
func protocolWorkerStopStep(state ProtocolState) []ProtocolStep {
	return []ProtocolStep{{Outcome: Accepted, State: state}}
}

// An asynchronous completion. Before a start, the server records a Started event first, which is why
// the evidence is two facts and not one -- and why the product machine, which has no `BackingOff`
// phase to have skipped, could write the completion alone.
func protocolCompleteStep(state ProtocolState, resolution Resolution) []ProtocolStep {
	if terminalPhase(state.Phase) {
		return []ProtocolStep{{Outcome: NotFound, State: state}}
	}
	if state.Phase == Unscheduled {
		return nil
	}
	var startedFirst []ProtocolFact
	if state.Phase != Started {
		startedFirst = []ProtocolFact{NexusOperationStarted{}}
	}
	switch resolution {
	case ResolutionSucceeded:
		return moves(state, Succeeded, append(startedFirst, NexusOperationCompleted{})...)
	case ResolutionFailed:
		return moves(state, Failed, append(startedFirst, NexusOperationFailed{})...)
	case ResolutionCanceled:
		return moves(state, Canceled, append(startedFirst, NexusOperationCanceled{})...)
	default:
		panic(fmt.Sprintf("unhandled Resolution %v", resolution))
	}
}

// The backoff timer. It is what makes `BackingOff` a phase the operation leaves rather than a state
// it is stuck in, and it records nothing: a retry writes no history event.
func backoffStep(state ProtocolState) []ProtocolStep {
	if state.Phase != BackingOff {
		return nil
	}
	return moves(state, Scheduled)
}

// The schedule-to-close deadline covers the whole operation, so it fires in every running phase --
// and only when the schedule command set it.
func scheduleToCloseStep(state ProtocolState) []ProtocolStep {
	if running(state.Phase) && state.ScheduleToClose == common.Expires {
		return moves(state, TimedOut, NexusOperationTimedOut{TimeoutType: ScheduleToClose})
	}
	return nil
}

// The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the start.
func scheduleToStartStep(state ProtocolState) []ProtocolStep {
	if (state.Phase == Scheduled || state.Phase == BackingOff) && state.ScheduleToStart == common.Expires {
		return moves(state, TimedOut, NexusOperationTimedOut{TimeoutType: ScheduleToStart})
	}
	return nil
}

// The start-to-close deadline covers the handler's own work, so it begins at the start.
func startToCloseStep(state ProtocolState) []ProtocolStep {
	if state.Phase == Started && state.StartToClose == common.Expires {
		return moves(state, TimedOut, NexusOperationTimedOut{TimeoutType: StartToClose})
	}
	return nil
}

// How a protocol state reads as a product state. A phase of the same name is that phase; backing off
// is still scheduled, because the product machine cannot see a retry; and an operation not yet
// scheduled reads as scheduled, because the product machine begins there. Every other field is
// hidden, which is what a map that does not read it says.
func productOf(state ProtocolState) ProductState {
	switch state.Phase {
	case Unscheduled, Scheduled, BackingOff:
		return ProductState{Phase: ProductScheduled}
	case Started:
		return ProductState{Phase: ProductStarted}
	case Succeeded:
		return ProductState{Phase: ProductSucceeded}
	case Failed:
		return ProductState{Phase: ProductFailed}
	case Canceled:
		return ProductState{Phase: ProductCanceled}
	case TimedOut:
		return ProductState{Phase: ProductTimedOut}
	default:
		panic(fmt.Sprintf("unhandled Phase %v", state.Phase))
	}
}

var NexusProtocol = &umpire.Machine[ProtocolState, ProtocolOutcome, ProtocolFact]{
	Name:         "nexusProtocol",
	For:          operation,
	States:       umpire.Fields[ProtocolState](),
	Refines:      umpire.Refines(NexusProduct, productOf),
	Starts:       []ProtocolState{{Phase: Unscheduled}},
	Ends:         func(s ProtocolState) bool { return terminalPhase(s.Phase) },
	Timers:       []umpire.Action{backoff, scheduleToClose, scheduleToStart, startToClose},
	Unobservable: []umpire.Action{backoff},
	Evidence: map[ProtocolFact]string{
		NexusOperationScheduled{}: "nexusOperationScheduled",
		NexusOperationStarted{}:   "nexusOperationStarted",
		NexusOperationCompleted{}: "nexusOperationCompleted",
		NexusOperationFailed{}:    "nexusOperationFailed",
		NexusOperationCanceled{}:  "nexusOperationCanceled",
		NexusOperationTimedOut{}:  "nexusOperationTimedOut",
		PendingAttempts{}:         pendingAttempts.Name,
	},
	Steps: umpire.Steps(
		umpire.Bind3(schedule, scheduleStep),
		umpire.Bind1(handlerReply, protocolHandlerReplyStep),
		umpire.Bind1(complete, protocolCompleteStep),
		umpire.Bind0(transportFault, protocolTransportFaultStep),
		umpire.Bind0(workerStop, protocolWorkerStopStep),
		umpire.Bind0(backoff, backoffStep),
		umpire.Bind0(scheduleToClose, scheduleToCloseStep),
		umpire.Bind0(scheduleToStart, scheduleToStartStep),
		umpire.Bind0(startToClose, startToCloseStep),
	),
}

// authoring: properties

// ### What the machines promise
//
// A same-step claim names the action it is about under When and holds of the step that action
// produces; a transition claim holds of the step before and the step after. A functional Query
// realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
// action the Case performs; a transition claim is searched and verified, never realized.

// Once an operation is over, no step changes its phase. Declared on the product machine and read on
// the protocol machine through the map.
var terminalIsFinal = &productProperty{
	Name:    "terminalIsFinal",
	Machine: NexusProduct,
	Transition: func(before, after ProductStep) bool {
		return !productTerminal(before.State) || after.State.Phase == before.State.Phase
	},
}

// A synchronous reply settles the operation as succeeded, and the completed event records it.
var syncSucceeds = &protocolProperty{
	Name:    "syncSucceeds",
	Machine: NexusProtocol,
	When:    handlerReply.With(SyncSuccess{}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Succeeded && step.Records(NexusOperationCompleted{})
	},
}

// An asynchronous reply starts the operation, and the started event records it.
var asyncStarts = &protocolProperty{
	Name:    "asyncStarts",
	Machine: NexusProtocol,
	When:    handlerReply.With(Async{}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Started && step.Records(NexusOperationStarted{})
	},
}

// A successful completion is recorded by the completed event. Neither the phase nor the outcome is
// fixed: a completion resolves any running phase, and `Accepted` is every earlier step's outcome too,
// so a clause fixing it would be answered before the completion.
var completionSucceeds = &protocolProperty{
	Name:    "completionSucceeds",
	Machine: NexusProtocol,
	When:    complete.With(ResolutionSucceeded),
	Holds:   func(step ProtocolStep) bool { return step.Records(NexusOperationCompleted{}) },
}

// A failed completion is recorded by the failed event.
var completionFails = &protocolProperty{
	Name:    "completionFails",
	Machine: NexusProtocol,
	When:    complete.With(ResolutionFailed),
	Holds:   func(step ProtocolStep) bool { return step.Records(NexusOperationFailed{}) },
}

// A non-retryable handler error settles the operation as failed, and the failed event records it.
var handlerErrorFails = &protocolProperty{
	Name:    "handlerErrorFails",
	Machine: NexusProtocol,
	When:    handlerReply.With(HandlerError{Retryable: false}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Failed && step.Records(NexusOperationFailed{})
	},
}

// succeededOnRetry is succeeded on the second attempt of an operation with no deadline set. A claim
// fixes one state, so every field is named.
var succeededOnRetry = ProtocolState{
	Phase:           Succeeded,
	Attempts:        1,
	ScheduleToClose: common.Unset,
	ScheduleToStart: common.Unset,
	StartToClose:    common.Unset,
}

// A synchronous reply to the retried attempt settles the operation as succeeded on its second attempt:
// the count the retryable failure raised is still one, and the completed event records the reply.
var retrySucceeds = &protocolProperty{
	Name:    "retrySucceeds",
	Machine: NexusProtocol,
	When:    handlerReply.With(SyncSuccess{}),
	Holds: func(step ProtocolStep) bool {
		return step.State == succeededOnRetry && step.Records(NexusOperationCompleted{})
	},
}

// The schedule-to-start deadline settles an operation no handler started as timed out, and the
// timed-out event records which deadline it was.
var scheduleToStartFires = &protocolProperty{
	Name:    "scheduleToStartFires",
	Machine: NexusProtocol,
	When:    scheduleToStart,
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == TimedOut && step.Records(NexusOperationTimedOut{TimeoutType: ScheduleToStart})
	},
}

// The start-to-close deadline settles a started operation no handler completed as timed out.
var startToCloseFires = &protocolProperty{
	Name:    "startToCloseFires",
	Machine: NexusProtocol,
	When:    startToClose,
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == TimedOut && step.Records(NexusOperationTimedOut{TimeoutType: StartToClose})
	},
}

// authoring: scenarios

// ### The paths the Queries run
//
// A protocol Scenario names its classed actions with their inputs and its start by its phase. Each
// path below is one upstream functional test's shape: the schedule command with no deadline set, then
// the side effects that settle the operation.

// unscheduled is the start every protocol Scenario shares: the zero value of every other field is the
// value the operation begins with, which is what Lean's `starts: unscheduled` defaults to.
var unscheduled = ProtocolState{Phase: Unscheduled}

var syncReplied = &protocolScenario{
	Name:   "syncReplied",
	Model:  NexusProtocol,
	Starts: unscheduled,
	Actions: []umpire.Class{
		schedule.With(common.Unset, common.Unset, common.Unset),
		handlerReply.With(SyncSuccess{}),
	},
}

var asyncThenSucceeded = &protocolScenario{
	Name:   "asyncThenSucceeded",
	Model:  NexusProtocol,
	Starts: unscheduled,
	Actions: []umpire.Class{
		schedule.With(common.Unset, common.Unset, common.Unset),
		handlerReply.With(Async{}),
		complete.With(ResolutionSucceeded),
	},
}

var asyncThenFailed = &protocolScenario{
	Name:   "asyncThenFailed",
	Model:  NexusProtocol,
	Starts: unscheduled,
	Actions: []umpire.Class{
		schedule.With(common.Unset, common.Unset, common.Unset),
		handlerReply.With(Async{}),
		complete.With(ResolutionFailed),
	},
}

var nonRetryableError = &protocolScenario{
	Name:   "nonRetryableError",
	Model:  NexusProtocol,
	Starts: unscheduled,
	Actions: []umpire.Class{
		schedule.With(common.Unset, common.Unset, common.Unset),
		handlerReply.With(HandlerError{Retryable: false}),
	},
}

// The retryable error backs the operation off; the backoff timer fires and records nothing; the
// retried attempt is answered synchronously.
var retriedThenSucceeded = &protocolScenario{
	Name:   "retriedThenSucceeded",
	Model:  NexusProtocol,
	Starts: unscheduled,
	Actions: []umpire.Class{
		schedule.With(common.Unset, common.Unset, common.Unset),
		handlerReply.With(HandlerError{Retryable: true}),
		backoff,
		handlerReply.With(SyncSuccess{}),
	},
}

// The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
// answers the start request; the deadline fires. The worker stops after the schedule in the
// operation's order, where the stop changes nothing; the realization stops it before the workflow
// starts, where the stop cannot race the dispatch.
var scheduleToStartExpires = &protocolScenario{
	Name:   "scheduleToStartExpires",
	Model:  NexusProtocol,
	Starts: unscheduled,
	Actions: []umpire.Class{
		schedule.With(common.Unset, common.Expires, common.Unset),
		workerStop,
		scheduleToStart,
	},
}

// The schedule command sets the start-to-close deadline; the handler accepts asynchronously and never
// completes; the deadline fires.
var startToCloseExpires = &protocolScenario{
	Name:   "startToCloseExpires",
	Model:  NexusProtocol,
	Starts: unscheduled,
	Actions: []umpire.Class{
		schedule.With(common.Unset, common.Unset, common.Expires),
		handlerReply.With(Async{}),
		startToClose,
	},
}

// Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
// sequence of two is found among ninety-nine candidates, one of three among about a thousand and one
// of four among about ten thousand.
var (
	two   = &umpire.Limits{Name: "two", Steps: 2, Actions: 2, Search: 512}
	three = &umpire.Limits{Name: "three", Steps: 3, Actions: 3, Search: 4096}
	four  = &umpire.Limits{Name: "four", Steps: 4, Actions: 4, Search: 32768}
)

// authoring: queries

// ### The Queries
//
// The design's seven: sync success, async reply then succeeded callback, async reply then failed
// callback, non-retryable handler error, retryable handler error then sync success after one backoff,
// schedule-to-start timeout with the handler's worker stopped, start-to-close timeout after an
// asynchronous reply. Each finds its same-step claim on its path and is realized by the set below. The
// product claim is verified over every trace of one path, outside the set, because a Verify Query
// realizes nothing.

var (
	syncCompletion         = &umpire.Query{Name: "syncCompletion", Find: syncSucceeds, In: syncReplied, Limits: two}
	asyncCompletion        = &umpire.Query{Name: "asyncCompletion", Find: completionSucceeds, In: asyncThenSucceeded, Limits: three}
	asyncFailure           = &umpire.Query{Name: "asyncFailure", Find: completionFails, In: asyncThenFailed, Limits: three}
	handlerError           = &umpire.Query{Name: "handlerError", Find: handlerErrorFails, In: nonRetryableError, Limits: two}
	retry                  = &umpire.Query{Name: "retry", Find: retrySucceeds, In: retriedThenSucceeded, Limits: four}
	scheduleToStartTimeout = &umpire.Query{Name: "scheduleToStartTimeout", Find: scheduleToStartFires, In: scheduleToStartExpires, Limits: three}
	startToCloseTimeout    = &umpire.Query{Name: "startToCloseTimeout", Find: startToCloseFires, In: startToCloseExpires, Limits: three}
	terminalHolds          = &umpire.Query{Name: "terminalHolds", Verify: terminalIsFinal, In: asyncThenSucceeded, Limits: three}
)

// authoring: set

// ### The functional set
//
// Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
// observes the network. The set repeats over the implementation switch, so each Query's Case runs
// once under HSM and once under CHASM.

var NexusCallerTests = &umpire.Set{
	Name:    "nexusCallerTests",
	Purpose: umpire.Functional,
	Bind: map[umpire.Party]umpire.Role{
		umpire.Caller:  umpire.Driven,
		umpire.Handler: umpire.Driven,
		umpire.Network: umpire.Observed,
		umpire.Worker:  umpire.Driven,
	},
	Repeat: umpire.Implementation,
	Queries: []*umpire.Query{
		syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
		scheduleToStartTimeout, startToCloseTimeout,
	},
}

// ### The canary set
//
// A canary runs a Query against a deployment that performs the handler's part itself: the handler is
// observed, so the verifier reads which reply occurred and checks the machine allows it. What admits a
// canary is that a deployment can close every gap its Case carries, and every step of the sync and
// async completion paths records evidence; a path with a silent step -- the backoff, the worker stop
// -- is a capability gap no deployment closes, so a canary naming it is rejected.

var NexusCallerCanary = &umpire.Set{
	Name:    "nexusCallerCanary",
	Purpose: umpire.Canary,
	Bind: map[umpire.Party]umpire.Role{
		umpire.Caller:  umpire.Driven,
		umpire.Handler: umpire.Observed,
		umpire.Network: umpire.Observed,
		umpire.Worker:  umpire.Driven,
	},
	Queries: []*umpire.Query{syncCompletion, asyncCompletion},
}

// ### The exploratory set
//
// An exploration covers the protocol machine rather than listing Queries. Its targets are the rows an
// exploration within the budget's steps of a start can take, the results those rows reach and the
// members of the classes their actions claim, each in the machine's catalog order and cut at the
// budget's search count, so the enumeration is the same on every reading;
// `Fixtures/CallerExploratoryCoverage.json` pins it.

var NexusCallerExploration = &umpire.Set{
	Name:    "nexusCallerExploration",
	Purpose: umpire.Exploratory,
	Bind: map[umpire.Party]umpire.Role{
		umpire.Caller:  umpire.Driven,
		umpire.Handler: umpire.Driven,
		umpire.Network: umpire.Observed,
		umpire.Worker:  umpire.Driven,
	},
	Machine: NexusProtocol,
	Cover:   umpire.Rows | umpire.Results | umpire.ClassMembers,
	Budget:  four,
}

// authoring: case

// ### The Cases
//
// Out of scope for this sample: the Testpilot runtime realizes each Set above into Cases from the
// machines' Evidence lines along the witness, so nothing is written twice. The Lean file declares
// them here; the marker keeps the section order.

// authoring: composition

// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's worker,
// so the schedule-to-start Scenario orders the stop before the request by convention. Composed with
// the worker of the handler's task queue, the stop is the worker's own phase change and every reply is
// the worker serving, so a reply has a row only while the worker polls. No set names the composition;
// it is what the cross-entity claim is verified over.

// HandlerWorker is the caller's view of the handler's worker: it stops and it serves. It never
// resumes, because an action no Sync line names would stay executable on its own and admit a stop, a
// resume and then a reply; the operation's timers settle every state a stop leaves.
var HandlerWorker = umpire.Restrict("handlerWorker", worker.Polling, worker.WorkerStop, worker.Serve)

// NexusCallerState is the composed state. The tags name the members, the way encoding/json names
// fields; Check matches them against Members.
type NexusCallerState struct {
	Operation ProtocolState      `umpire:"operation"`
	Worker    worker.WorkerState `umpire:"worker"`
}

var NexusCaller = &umpire.Compose[NexusCallerState]{
	Name:    "nexusCaller",
	For:     []*umpire.Entity{operation, worker.Worker},
	Members: umpire.Members{"operation": NexusProtocol, "worker": HandlerWorker},
	Sync: umpire.Sync{
		workerStop:   {"operation": workerStop, "worker": worker.WorkerStop},
		handlerReply: {"operation": handlerReply, "worker": worker.Serve},
	},
	Starts: []NexusCallerState{{Operation: unscheduled, Worker: worker.WorkerState{Phase: worker.PhasePolling}}},
	Ends:   func(s NexusCallerState) bool { return terminalPhase(s.Operation.Phase) },
}

// Every reply, of any class, leaves the handler's worker polling: no handler replies while its worker
// is stopped.
var repliedByPollingWorker = &umpire.Property[NexusCallerState, umpire.Joint, umpire.Joint]{
	Name:    "repliedByPollingWorker",
	Machine: NexusCaller,
	When:    handlerReply,
	Holds: func(step umpire.Composed[NexusCallerState]) bool {
		return step.State.Worker.Phase == worker.PhasePolling
	},
}

// A retryable reply backs the operation off; the handler's worker then stops, so the retried attempt
// is never answered and the schedule-to-start deadline fires.
var repliedThenStopped = &umpire.Scenario[NexusCallerState, umpire.Joint, umpire.Joint]{
	Name:   "repliedThenStopped",
	Model:  NexusCaller,
	Starts: NexusCallerState{Operation: unscheduled, Worker: worker.WorkerState{Phase: worker.PhasePolling}},
	Actions: []umpire.Class{
		umpire.At("operation", schedule.With(common.Unset, common.Expires, common.Unset)),
		handlerReply.With(HandlerError{Retryable: true}),
		workerStop,
		umpire.At("operation", scheduleToStart),
	},
}

var stoppedWorkerRepliesNothing = &umpire.Query{
	Name:   "stoppedWorkerRepliesNothing",
	Verify: repliedByPollingWorker,
	In:     repliedThenStopped,
	Limits: four,
}

// authoring: end
