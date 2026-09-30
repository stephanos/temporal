// Package nexuscaller is the Nexus caller-side Model: one workflow-scheduled Nexus operation, as
// the caller sees it. The product machine says what an operation does, the protocol machine says
// how the server gets there and refines it, and the functional set runs one Query per side effect
// that settles the operation, once per value of the implementation switch. No cancellation (fn-79)
// and no concurrency-limit setup parameter.
//
// Ported from model/lean/Temporal/Feature/Nexus/Caller/Model.lean, in its order: vocabulary, the two
// machines, what they promise, what the set asks.
package nexuscaller

import (
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/go/worker"
)

// Family is the root of this package's Definition IDs.
const Family umpire.Family = "temporal.nexus.caller"

// Parties are names the feature declares by using them. The reserved party system is the server.
// A fault is an ordinary action of a declared party, and a timer is system behavior the machine
// owns, so neither is a separate kind.
const (
	Caller  umpire.Party = "caller"
	Handler umpire.Party = "handler"
	Network umpire.Party = "network"
)

// ### Entities
//
// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

var (
	Workflow  = &umpire.Entity{Name: "workflow"}
	Operation = &umpire.Entity{Name: "operation", Key: "scheduledEvent",
		Refer: map[string]*umpire.Entity{"caller": Workflow}}
)

// ### The input domains
//
// A class is one member of a domain, and a variant that carries finite fields contributes one class
// per assignment of them: HandlerError{Retryable} is one variant and two classes, which is the
// granularity an example is written at and what mirrors a protobuf oneof.

// Timeout is whether the schedule command sets a deadline.
type Timeout string

const (
	Unset   Timeout = "unset"
	Expires Timeout = "expires"
)

func (Timeout) Values() []Timeout { return []Timeout{Unset, Expires} }

// Reply is the handler's reply to the server's start request.
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

var _ = umpire.Sum[Reply](SyncSuccess{}, Async{}, OperationFailed{}, OperationCanceled{}, HandlerError{})

// Resolution is how an asynchronous completion settles the operation.
type Resolution string

const (
	ResolvedSucceeded Resolution = "succeeded"
	ResolvedFailed    Resolution = "failed"
	ResolvedCanceled  Resolution = "canceled"
)

func (Resolution) Values() []Resolution {
	return []Resolution{ResolvedSucceeded, ResolvedFailed, ResolvedCanceled}
}

// Delivery is what the completion's delivery reports.
type Delivery string

const (
	Delivered Delivery = "accepted"
	NotFound  Delivery = "notFound"
)

func (Delivery) Values() []Delivery { return []Delivery{Delivered, NotFound} }

// ### Actions

var (
	schedule = umpire.NewAction3[Timeout, Timeout, Timeout]("schedule", Caller,
		"scheduleToClose", "scheduleToStart", "startToClose",
		umpire.Creates(Operation),
		umpire.Schema("temporal.api.command.v1.ScheduleNexusOperationCommandAttributes"))

	handlerReply = umpire.NewAction1[Reply]("handlerReply", Handler, "reply",
		umpire.On(Operation),
		umpire.Schema("temporal.api.nexus.v1.StartOperationResponse", "temporal.api.nexus.v1.HandlerError"),
		umpire.Example(HandlerError{Retryable: false}, "BadRequest"),
		umpire.Example(HandlerError{Retryable: true}, "Internal"))

	// complete: the Nexus HTTP completion carries no protobuf message, so it declares no schema and
	// its classes are names the realization interprets.
	complete = umpire.NewAction1[Resolution]("complete", Handler, "resolution",
		umpire.On(Operation), umpire.Results("Delivery"))

	transportFault = umpire.NewAction0("transportFault", Network, umpire.On(Operation))

	// workerStop: the handler's worker stops polling. An action that names no entity is behavior no
	// entity records: the Run records the fault, but nothing recorded names the operation, so the
	// machines keep their state and record nothing at it.
	workerStop = worker.WorkerStop
)

// ### The derived observation
//
// A retryable attempt failure writes no history event, so the attempt count is read back through
// DescribeWorkflowExecution. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

var PendingAttempts = umpire.Observation{Name: "pendingAttempts", On: Operation, Read: "attempts"}

// Outcome is a step's outcome. The product and protocol machines share the two members, and an
// outcome reads as the refined machine's outcome of the same name.
type Outcome string

const (
	Accepted          Outcome = "accepted"
	CompletionMissing Outcome = "notFound"
)

func (Outcome) Values() []Outcome { return []Outcome{Accepted, CompletionMissing} }

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

// ProductPhase is the product operation's phase.
type ProductPhase string

const (
	ProductScheduled ProductPhase = "scheduled"
	ProductStarted   ProductPhase = "started"
	ProductSucceeded ProductPhase = "succeeded"
	ProductFailed    ProductPhase = "failed"
	ProductCanceled  ProductPhase = "canceled"
	ProductTimedOut  ProductPhase = "timedOut"
)

func (ProductPhase) Values() []ProductPhase {
	return []ProductPhase{ProductScheduled, ProductStarted, ProductSucceeded, ProductFailed,
		ProductCanceled, ProductTimedOut}
}

// ProductState is the product machine's state.
type ProductState struct {
	Phase ProductPhase
}

// ProductFact is what the product machine records.
type ProductFact string

const (
	ProductOperationScheduled ProductFact = "nexusOperationScheduled"
	ProductOperationStarted   ProductFact = "nexusOperationStarted"
	ProductOperationCompleted ProductFact = "nexusOperationCompleted"
	ProductOperationFailed    ProductFact = "nexusOperationFailed"
	ProductOperationCanceled  ProductFact = "nexusOperationCanceled"
	ProductOperationTimedOut  ProductFact = "nexusOperationTimedOut"
)

func (ProductFact) Values() []ProductFact {
	return []ProductFact{ProductOperationScheduled, ProductOperationStarted, ProductOperationCompleted,
		ProductOperationFailed, ProductOperationCanceled, ProductOperationTimedOut}
}

// ProductStep is one product step.
type ProductStep = umpire.Step[ProductState, Outcome, ProductFact]

func productStep(phase ProductPhase, recorded ProductFact) []ProductStep {
	return []ProductStep{{Outcome: Accepted, State: ProductState{phase}, Facts: []ProductFact{recorded}}}
}

// handlerReplyStep: the handler's reply to the server's start request. An operation that has not
// started yet is the only one a reply can move.
func handlerReplyStep(s ProductState, reply Reply) []ProductStep {
	if s.Phase != ProductScheduled {
		return nil
	}
	switch r := reply.(type) {
	case SyncSuccess:
		return productStep(ProductSucceeded, ProductOperationCompleted)
	case Async:
		return productStep(ProductStarted, ProductOperationStarted)
	case OperationFailed:
		return productStep(ProductFailed, ProductOperationFailed)
	case OperationCanceled:
		return productStep(ProductCanceled, ProductOperationCanceled)
	case HandlerError:
		// A retryable handler error leaves the operation where it is: the product machine does not
		// know about backing off, which is the whole of what the protocol machine adds.
		if r.Retryable {
			return nil
		}
		return productStep(ProductFailed, ProductOperationFailed)
	}
	// Unreachable: go-check-sumtype requires every Reply variant above.
	return nil
}

// ProductTerminal is the four phases the product machine ends on.
func ProductTerminal(s ProductState) bool {
	return s.Phase == ProductSucceeded || s.Phase == ProductFailed || s.Phase == ProductCanceled ||
		s.Phase == ProductTimedOut
}

// completeStep: an asynchronous completion. A completion that arrives after the operation is over
// is not found, and changes nothing.
func completeStep(s ProductState, resolution Resolution) []ProductStep {
	if ProductTerminal(s) {
		return []ProductStep{{Outcome: CompletionMissing, State: s}}
	}
	switch resolution {
	case ResolvedSucceeded:
		return productStep(ProductSucceeded, ProductOperationCompleted)
	case ResolvedFailed:
		return productStep(ProductFailed, ProductOperationFailed)
	case ResolvedCanceled:
		return productStep(ProductCanceled, ProductOperationCanceled)
	}
	// Unreachable: exhaustive requires every Resolution above.
	return nil
}

// transportFaultStep: a transport fault is an ordinary action of the network. The product machine
// cannot see one: whether a delivery was retried is the protocol's account of how, not what.
func transportFaultStep(ProductState) []ProductStep { return nil }

// workerStopStep: the handler's worker stopping is a fault the Run records and the operation does
// not feel. The product machine cannot see it, like the transport fault: a step that kept the state
// and recorded nothing would be indistinguishable from a stutter, and the refinement would read
// every stutter as this step.
func workerStopStep(ProductState) []ProductStep { return nil }

// timeoutStep: one of the operation's deadlines firing. Which deadline is the protocol's account of
// how, so the product machine has one timer, and it fires while the operation runs.
func timeoutStep(s ProductState) []ProductStep {
	if s.Phase == ProductScheduled || s.Phase == ProductStarted {
		return productStep(ProductTimedOut, ProductOperationTimedOut)
	}
	return nil
}

var timeout = umpire.Timer("timeout")

// NexusProduct is the product machine.
var NexusProduct = umpire.NewMachine[ProductState, Outcome, ProductFact](Family, "nexusProduct").
	For(Operation).
	Starts(ProductState{ProductScheduled}).
	Ends(ProductTerminal).
	Evidence("nexusOperationStarted", "nexusOperationStarted").
	Evidence("nexusOperationCompleted", "nexusOperationCompleted").
	Evidence("nexusOperationFailed", "nexusOperationFailed").
	Evidence("nexusOperationCanceled", "nexusOperationCanceled").
	Evidence("nexusOperationTimedOut", "nexusOperationTimedOut").
	Step1(handlerReply, handlerReplyStep).
	Step1(complete, completeStep).
	Step0(transportFault, transportFaultStep).
	Step0(workerStop, workerStopStep).
	Step0(timeout, timeoutStep)

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the three timers the
// schedule command sets, and the attempt count a retryable failure raises. Written against the same
// actions, so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state structure has no "no instance yet"
// member, so unscheduled is that member, and it is what makes the three deadline fields reachable at
// anything but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the cancel field and its rows (fn-79), and the
// concurrency-limit rejection, which names no operation and is not modeled until a Query needs it.

// Phase is the protocol operation's phase.
type Phase string

const (
	Unscheduled Phase = "unscheduled"
	Scheduled   Phase = "scheduled"
	BackingOff  Phase = "backingOff"
	Started     Phase = "started"
	Succeeded   Phase = "succeeded"
	Failed      Phase = "failed"
	Canceled    Phase = "canceled"
	TimedOut    Phase = "timedOut"
)

func (Phase) Values() []Phase {
	return []Phase{Unscheduled, Scheduled, BackingOff, Started, Succeeded, Failed, Canceled, TimedOut}
}

// TimeoutType is which timer fired. The history event records it, so a Contract that did not check
// it would pass a run that timed out on the wrong deadline.
type TimeoutType string

const (
	TypeScheduleToClose TimeoutType = "scheduleToClose"
	TypeScheduleToStart TimeoutType = "scheduleToStart"
	TypeStartToClose    TimeoutType = "startToClose"
)

func (TimeoutType) Values() []TimeoutType {
	return []TimeoutType{TypeScheduleToClose, TypeScheduleToStart, TypeStartToClose}
}

// AttemptBound bounds the attempt count. Nothing wires the Limits into a machine's state, so the
// bound is written here and the saturating successor keeps a retry inside it.
const AttemptBound = 2

// Attempts is the attempt count, 0 to AttemptBound.
type Attempts uint8

func (Attempts) Values() []Attempts {
	out := make([]Attempts, AttemptBound+1)
	for i := range out {
		out[i] = Attempts(i)
	}
	return out
}

func (a Attempts) saturatingSucc() Attempts { return min(a+1, AttemptBound) }

// ProtocolState is the protocol machine's state.
type ProtocolState struct {
	Phase           Phase
	Attempts        Attempts
	ScheduleToClose Timeout
	ScheduleToStart Timeout
	StartToClose    Timeout
}

// ProtocolFact is what the protocol machine records.
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
	PendingAttemptsRead     struct{}
)

func (NexusOperationScheduled) isProtocolFact() {}
func (NexusOperationStarted) isProtocolFact()   {}
func (NexusOperationCompleted) isProtocolFact() {}
func (NexusOperationFailed) isProtocolFact()    {}
func (NexusOperationCanceled) isProtocolFact()  {}
func (NexusOperationTimedOut) isProtocolFact()  {}
func (PendingAttemptsRead) isProtocolFact()     {}

// Key spells the fact as the observation it is read through.
func (PendingAttemptsRead) Key() string { return "pendingAttempts" }

var _ = umpire.Sum[ProtocolFact](NexusOperationScheduled{}, NexusOperationStarted{},
	NexusOperationCompleted{}, NexusOperationFailed{}, NexusOperationCanceled{},
	NexusOperationTimedOut{}, PendingAttemptsRead{})

// ProtocolStep is one protocol step.
type ProtocolStep = umpire.Step[ProtocolState, Outcome, ProtocolFact]

// terminalPhase is the four phases the design ends on. A completion that arrives after one of them
// is not found.
func terminalPhase(p Phase) bool {
	return p == Succeeded || p == Failed || p == Canceled || p == TimedOut
}

// running is scheduled and not yet over: the phases a completion resolves and a timer can fire in.
func running(p Phase) bool { return p == Scheduled || p == BackingOff || p == Started }

func moves(s ProtocolState, phase Phase, recorded ...ProtocolFact) []ProtocolStep {
	s.Phase = phase
	return []ProtocolStep{{Outcome: Accepted, State: s, Facts: recorded}}
}

// scheduleStep: the caller's schedule command. It names the operation's three deadlines, and every
// one of them is a state field because whether a timer fires is a question about the operation and
// not about the command that started it.
func scheduleStep(s ProtocolState, scheduleToClose, scheduleToStart, startToClose Timeout) []ProtocolStep {
	if s.Phase != Unscheduled {
		return nil
	}
	return []ProtocolStep{{Outcome: Accepted,
		State: ProtocolState{Phase: Scheduled, Attempts: 0, ScheduleToClose: scheduleToClose,
			ScheduleToStart: scheduleToStart, StartToClose: startToClose},
		Facts: []ProtocolFact{NexusOperationScheduled{}}}}
}

// protocolHandlerReplyStep: the handler's reply to the server's start request. What the product
// machine cannot see is the last arm: a retryable failure backs the operation off and raises its
// attempt count, and the count is read back through the pendingAttempts observation because no
// history event records it.
func protocolHandlerReplyStep(s ProtocolState, reply Reply) []ProtocolStep {
	if s.Phase != Scheduled {
		return nil
	}
	switch r := reply.(type) {
	case SyncSuccess:
		return moves(s, Succeeded, NexusOperationCompleted{})
	case Async:
		return moves(s, Started, NexusOperationStarted{})
	case OperationFailed:
		return moves(s, Failed, NexusOperationFailed{})
	case OperationCanceled:
		return moves(s, Canceled, NexusOperationCanceled{})
	case HandlerError:
		if !r.Retryable {
			return moves(s, Failed, NexusOperationFailed{})
		}
		s.Attempts = s.Attempts.saturatingSucc()
		return moves(s, BackingOff, PendingAttemptsRead{})
	}
	// Unreachable: go-check-sumtype requires every Reply variant above.
	return nil
}

// protocolTransportFaultStep: a transport fault is the same failure arriving as a dropped delivery
// rather than as a reply.
func protocolTransportFaultStep(s ProtocolState) []ProtocolStep {
	if s.Phase != Scheduled {
		return nil
	}
	s.Attempts = s.Attempts.saturatingSucc()
	return moves(s, BackingOff, PendingAttemptsRead{})
}

// protocolWorkerStopStep: the handler's worker stopping is a fault the Run records and the
// operation does not feel, so the step keeps the state and records nothing. On a path it is
// confirmed by the evidence of the step after it, and the Case says so in a Known Gap.
func protocolWorkerStopStep(s ProtocolState) []ProtocolStep {
	return []ProtocolStep{{Outcome: Accepted, State: s}}
}

// protocolCompleteStep: an asynchronous completion. Before a start, the server records a Started
// event first, which is why the evidence is two facts and not one -- and why the product machine,
// which has no backingOff phase to have skipped, could write the completion alone.
func protocolCompleteStep(s ProtocolState, resolution Resolution) []ProtocolStep {
	if terminalPhase(s.Phase) {
		return []ProtocolStep{{Outcome: CompletionMissing, State: s}}
	}
	if s.Phase == Unscheduled {
		return nil
	}
	var startedFirst []ProtocolFact
	if s.Phase != Started {
		startedFirst = []ProtocolFact{NexusOperationStarted{}}
	}
	switch resolution {
	case ResolvedSucceeded:
		return moves(s, Succeeded, append(startedFirst, NexusOperationCompleted{})...)
	case ResolvedFailed:
		return moves(s, Failed, append(startedFirst, NexusOperationFailed{})...)
	case ResolvedCanceled:
		return moves(s, Canceled, append(startedFirst, NexusOperationCanceled{})...)
	}
	// Unreachable: exhaustive requires every Resolution above.
	return nil
}

// backoffStep: the backoff timer. It is what makes backingOff a phase the operation leaves rather
// than a state it is stuck in, and it records nothing: a retry writes no history event.
func backoffStep(s ProtocolState) []ProtocolStep {
	if s.Phase != BackingOff {
		return nil
	}
	return moves(s, Scheduled)
}

// scheduleToCloseStep: the schedule-to-close deadline covers the whole operation, so it fires in
// every running phase -- and only when the schedule command set it.
func scheduleToCloseStep(s ProtocolState) []ProtocolStep {
	if running(s.Phase) && s.ScheduleToClose == Expires {
		return moves(s, TimedOut, NexusOperationTimedOut{TypeScheduleToClose})
	}
	return nil
}

// scheduleToStartStep: the schedule-to-start deadline covers the wait for the handler to accept,
// so it stops at the start.
func scheduleToStartStep(s ProtocolState) []ProtocolStep {
	if (s.Phase == Scheduled || s.Phase == BackingOff) && s.ScheduleToStart == Expires {
		return moves(s, TimedOut, NexusOperationTimedOut{TypeScheduleToStart})
	}
	return nil
}

// startToCloseStep: the start-to-close deadline covers the handler's own work, so it begins at the
// start.
func startToCloseStep(s ProtocolState) []ProtocolStep {
	if s.Phase == Started && s.StartToClose == Expires {
		return moves(s, TimedOut, NexusOperationTimedOut{TypeStartToClose})
	}
	return nil
}

// productOf is how a protocol state reads as a product state. A phase of the same name is that
// phase; backing off is still scheduled, because the product machine cannot see a retry; and an
// operation not yet scheduled reads as scheduled, because the product machine begins there. Every
// other field is hidden, which is what a map that does not read it says.
func productOf(s ProtocolState) ProductState {
	switch s.Phase {
	case Unscheduled, Scheduled, BackingOff:
		return ProductState{ProductScheduled}
	case Started:
		return ProductState{ProductStarted}
	case Succeeded:
		return ProductState{ProductSucceeded}
	case Failed:
		return ProductState{ProductFailed}
	case Canceled:
		return ProductState{ProductCanceled}
	case TimedOut:
		return ProductState{ProductTimedOut}
	}
	// Unreachable: exhaustive requires every Phase above. The zero state is outside the product's
	// domain, so the refinement check would reject a row that reached it.
	return ProductState{}
}

var (
	backoff         = umpire.Timer("backoff")
	scheduleToClose = umpire.Timer("scheduleToClose")
	scheduleToStart = umpire.Timer("scheduleToStart")
	startToClose    = umpire.Timer("startToClose")
)

// NexusProtocol is the protocol machine.
var NexusProtocol = umpire.NewMachine[ProtocolState, Outcome, ProtocolFact](Family, "nexusProtocol").
	For(Operation).
	Refines(NexusProduct, productOf).
	Starts(ProtocolState{Phase: Unscheduled, ScheduleToClose: Unset, ScheduleToStart: Unset, StartToClose: Unset}).
	Ends(func(s ProtocolState) bool { return terminalPhase(s.Phase) }).
	Unobservable(backoff).
	Evidence("nexusOperationScheduled", "nexusOperationScheduled").
	Evidence("nexusOperationStarted", "nexusOperationStarted").
	Evidence("nexusOperationCompleted", "nexusOperationCompleted").
	Evidence("nexusOperationFailed", "nexusOperationFailed").
	Evidence("nexusOperationCanceled", "nexusOperationCanceled").
	Evidence("nexusOperationTimedOut", "nexusOperationTimedOut").
	Evidence("pendingAttempts", "pendingAttempts").
	Step3(schedule, scheduleStep).
	Step1(handlerReply, protocolHandlerReplyStep).
	Step1(complete, protocolCompleteStep).
	Step0(transportFault, protocolTransportFaultStep).
	Step0(workerStop, protocolWorkerStopStep).
	Step0(backoff, backoffStep).
	Step0(scheduleToClose, scheduleToCloseStep).
	Step0(scheduleToStart, scheduleToStartStep).
	Step0(startToClose, startToCloseStep)

// HandlerWorker is the caller's view of the handler's worker: it stops and it serves. It never
// resumes, because an action no sync line names would stay executable on its own and admit a stop,
// a resume and then a reply; the operation's timers settle every state a stop leaves.
var HandlerWorker = worker.PollingMachine.Restrict(Family, "handlerWorker",
	worker.WorkerStop.ActionDecl, worker.Serve.ActionDecl)

// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No set
// names the composition; it is what the cross-entity claim is verified over.

// NexusCallerState is the composed state: the operation and the handler's worker.
type NexusCallerState struct {
	Operation ProtocolState `umpire:"operation"`
	Worker    worker.State  `umpire:"worker"`
}

// NexusCaller composes the protocol machine with the handler's worker.
var NexusCaller = umpire.Compose[NexusCallerState](Family, "nexusCaller").
	Member("operation", NexusProtocol).
	Member("worker", HandlerWorker).
	Sync("workerStop", "operation.workerStop", "worker.workerStop").
	Sync("handlerReply", "operation.handlerReply", "worker.serve").
	Ends(func(s NexusCallerState) bool { return terminalPhase(s.Operation.Phase) })
