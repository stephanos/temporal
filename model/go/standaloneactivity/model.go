// Package standaloneactivity is the standalone activity Model: one activity started directly
// through StartActivityExecution, with no workflow around it. The product machine says what an
// activity does as DescribeActivityExecution reports it, the protocol machine says how the server
// gets there and refines it, and the functional set runs one Query per side effect that settles the
// activity. Grounded in chasm/lib/activity/statemachine.go; reset is deferred, like cancellation in
// the Nexus caller Model, and the heartbeat timeout is not modeled.
//
// A standalone activity writes no history event. Every fact below is a status read through
// DescribeActivityExecution, or a result read through PollActivityExecution, so the evidence lines
// of the machines name observations rather than events.
//
// Ported from .plans/cmp/lean/StandaloneActivity.lean, in its order: vocabulary, the two machines,
// what they promise, what the set asks.
package standaloneactivity

import (
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/go/worker"
)

// Family is the root of this package's Definition IDs.
const Family umpire.Family = "temporal.activity.standalone"

// Caller starts and controls the activity; the worker party runs its attempts; system owns the
// timers. The worker's stop is an ordinary action of the worker party, as in every other Model.
const Caller umpire.Party = "caller"

// ### Entities

// Activity is named by the id the caller chose for it: every status read and every result read
// carries it, and no run id or event id is needed to tell two apart.
var Activity = &umpire.Entity{Name: "activity", Key: "activityId"}

// ### The input domains
//
// As in the caller Model, a variant with a finite field contributes one class per assignment:
// Failed{Retryable} is two classes, which mirrors the retryable flag of an ApplicationFailure.

// Timeout is whether the start request sets a deadline.
type Timeout string

const (
	Unset   Timeout = "unset"
	Expires Timeout = "expires"
)

func (Timeout) Values() []Timeout { return []Timeout{Unset, Expires} }

// AttemptResult is the worker's answer to an attempt.
//
//sumtype:decl
type AttemptResult interface{ isAttemptResult() }

type (
	Completed     struct{}
	Failed        struct{ Retryable bool }
	CanceledByRun struct{}
)

func (Completed) isAttemptResult()     {}
func (Failed) isAttemptResult()        {}
func (CanceledByRun) isAttemptResult() {}

// Key spells the answer the way the Lean constructor is named.
func (CanceledByRun) Key() string { return "canceled" }

var _ = umpire.Sum[AttemptResult](Completed{}, Failed{}, CanceledByRun{})

// Delivery is what a control reports.
type Delivery string

const (
	Delivered Delivery = "accepted"
	NotFound  Delivery = "notFound"
)

func (Delivery) Values() []Delivery { return []Delivery{Delivered, NotFound} }

// Control is one of the caller's four controls.
type Control string

const (
	Pause         Control = "pause"
	Unpause       Control = "unpause"
	RequestCancel Control = "requestCancel"
	Terminate     Control = "terminate"
)

func (Control) Values() []Control { return []Control{Pause, Unpause, RequestCancel, Terminate} }

// ### Actions

var (
	start = umpire.NewAction3[Timeout, Timeout, Timeout]("start", Caller,
		"scheduleToClose", "scheduleToStart", "startToClose",
		umpire.Creates(Activity),
		umpire.Schema("temporal.api.workflowservice.v1.StartActivityExecutionRequest"))

	// attemptStart: the worker's poll receives the task for the current attempt.
	attemptStart = umpire.NewAction0("attemptStart", worker.Party, umpire.On(Activity),
		umpire.Schema("temporal.api.workflowservice.v1.PollActivityTaskQueueResponse"))

	attemptResult = umpire.NewAction1[AttemptResult]("attemptResult", worker.Party, "result",
		umpire.On(Activity),
		umpire.Schema("temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest",
			"temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest",
			"temporal.api.workflowservice.v1.RespondActivityTaskCanceledRequest"),
		umpire.Example(Failed{Retryable: false}, "ApplicationFailureNonRetryable"),
		umpire.Example(Failed{Retryable: true}, "ApplicationFailureRetryable"))

	// control: the four caller-side controls are one action with a finite input, because they share
	// a result: a control on an activity that is over is not found.
	control = umpire.NewAction1[Control]("control", Caller, "control",
		umpire.On(Activity),
		umpire.Schema("temporal.api.workflowservice.v1.PauseActivityExecutionRequest",
			"temporal.api.workflowservice.v1.UnpauseActivityExecutionRequest",
			"temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest",
			"temporal.api.workflowservice.v1.TerminateActivityExecutionRequest"),
		umpire.Results("Delivery"))

	// workerStop: the worker stops polling. Nothing recorded names the activity, so the machines
	// keep their state and record nothing at it.
	workerStop = worker.WorkerStop
)

// ### The derived observation
//
// A retried attempt writes nothing the caller can see except the attempt count that
// DescribeActivityExecution reports, so it is the one derived observation. Each status a machine
// records is an observation of the status field; the Lean sample declares nine observations over
// that one field and notes the evidence catalog may not accept them. Go declares them as data and
// leaves the question to the realization.

var AttemptCount = umpire.Observation{Name: "attemptCount", On: Activity, Read: "attempt"}

// Outcome is a step's outcome, shared by both machines by name.
type Outcome string

const (
	Accepted    Outcome = "accepted"
	ControlLost Outcome = "notFound"
)

func (Outcome) Values() []Outcome { return []Outcome{Accepted, ControlLost} }

// ### The product machine
//
// What the caller sees through DescribeActivityExecution, with no account of how: a retry reads as
// scheduled again, and a pause requested of a running attempt reads as started until the worker
// yields.

// ProductPhase is the product activity's phase.
type ProductPhase string

const (
	ProductScheduled       ProductPhase = "scheduled"
	ProductStarted         ProductPhase = "started"
	ProductPaused          ProductPhase = "paused"
	ProductCancelRequested ProductPhase = "cancelRequested"
	ProductCompleted       ProductPhase = "completed"
	ProductFailed          ProductPhase = "failed"
	ProductCanceled        ProductPhase = "canceled"
	ProductTerminated      ProductPhase = "terminated"
	ProductTimedOut        ProductPhase = "timedOut"
)

func (ProductPhase) Values() []ProductPhase {
	return []ProductPhase{ProductScheduled, ProductStarted, ProductPaused, ProductCancelRequested,
		ProductCompleted, ProductFailed, ProductCanceled, ProductTerminated, ProductTimedOut}
}

// ProductState is the product machine's state.
type ProductState struct {
	Phase ProductPhase
}

// ProductFact is a status the caller reads.
type ProductFact string

const (
	ProductStatusScheduled       ProductFact = "statusScheduled"
	ProductStatusStarted         ProductFact = "statusStarted"
	ProductStatusPaused          ProductFact = "statusPaused"
	ProductStatusCancelRequested ProductFact = "statusCancelRequested"
	ProductStatusCompleted       ProductFact = "statusCompleted"
	ProductStatusFailed          ProductFact = "statusFailed"
	ProductStatusCanceled        ProductFact = "statusCanceled"
	ProductStatusTerminated      ProductFact = "statusTerminated"
	ProductStatusTimedOut        ProductFact = "statusTimedOut"
)

func (ProductFact) Values() []ProductFact {
	return []ProductFact{ProductStatusScheduled, ProductStatusStarted, ProductStatusPaused,
		ProductStatusCancelRequested, ProductStatusCompleted, ProductStatusFailed, ProductStatusCanceled,
		ProductStatusTerminated, ProductStatusTimedOut}
}

// ProductStep is one product step.
type ProductStep = umpire.Step[ProductState, Outcome, ProductFact]

func productStep(phase ProductPhase, recorded ProductFact) []ProductStep {
	return []ProductStep{{Outcome: Accepted, State: ProductState{phase}, Facts: []ProductFact{recorded}}}
}

// ProductTerminal is the phases the product machine ends on.
func ProductTerminal(s ProductState) bool {
	switch s.Phase {
	case ProductCompleted, ProductFailed, ProductCanceled, ProductTerminated, ProductTimedOut:
		return true
	case ProductScheduled, ProductStarted, ProductPaused, ProductCancelRequested:
		return false
	}
	return false
}

// attemptStartStep: a worker takes the attempt of a scheduled activity.
func attemptStartStep(s ProductState) []ProductStep {
	if s.Phase != ProductScheduled {
		return nil
	}
	return productStep(ProductStarted, ProductStatusStarted)
}

// attemptResultStep: the worker's answer to the attempt. Unlike the Nexus caller, a retryable
// failure is visible here: DescribeActivityExecution reads SCHEDULED again with a higher attempt
// count (TransitionRescheduled), and under a cancel request it settles the activity as canceled. The
// backoff between the two is what the protocol machine adds. A canceled answer settles only an
// activity whose cancellation was requested.
func attemptResultStep(s ProductState, result AttemptResult) []ProductStep {
	if s.Phase != ProductStarted && s.Phase != ProductCancelRequested {
		return nil
	}
	switch r := result.(type) {
	case Completed:
		return productStep(ProductCompleted, ProductStatusCompleted)
	case Failed:
		switch {
		case !r.Retryable:
			return productStep(ProductFailed, ProductStatusFailed)
		case s.Phase == ProductCancelRequested:
			return productStep(ProductCanceled, ProductStatusCanceled)
		default:
			return productStep(ProductScheduled, ProductStatusScheduled)
		}
	case CanceledByRun:
		if s.Phase == ProductCancelRequested {
			return productStep(ProductCanceled, ProductStatusCanceled)
		}
		return nil
	default:
		// Unreachable: go-check-sumtype requires every AttemptResult variant above.
		return nil
	}
}

// controlStep: a control on an activity that is over is not found and changes nothing.
func controlStep(s ProductState, c Control) []ProductStep {
	if ProductTerminal(s) {
		return []ProductStep{{Outcome: ControlLost, State: s}}
	}
	switch c {
	case Pause:
		if s.Phase == ProductScheduled || s.Phase == ProductStarted {
			return productStep(ProductPaused, ProductStatusPaused)
		}
		return nil
	case Unpause:
		if s.Phase == ProductPaused {
			return productStep(ProductScheduled, ProductStatusScheduled)
		}
		return nil
	case RequestCancel:
		if s.Phase == ProductScheduled || s.Phase == ProductStarted || s.Phase == ProductPaused ||
			s.Phase == ProductCancelRequested {
			return productStep(ProductCancelRequested, ProductStatusCancelRequested)
		}
		return nil
	case Terminate:
		return productStep(ProductTerminated, ProductStatusTerminated)
	}
	// Unreachable: exhaustive requires every Control above.
	return nil
}

// workerStopStep: the worker stopping is a fault the Run records and the activity does not feel.
func workerStopStep(ProductState) []ProductStep { return nil }

// timeoutStep: one of the activity's deadlines firing. Which deadline is the protocol's account of
// how.
func timeoutStep(s ProductState) []ProductStep {
	if s.Phase == ProductScheduled || s.Phase == ProductStarted || s.Phase == ProductCancelRequested ||
		s.Phase == ProductPaused {
		return productStep(ProductTimedOut, ProductStatusTimedOut)
	}
	return nil
}

var timeout = umpire.Timer("timeout")

// ActivityProduct is the product machine.
var ActivityProduct = evidenceForStatuses(
	umpire.NewMachine[ProductState, Outcome, ProductFact](Family, "activityProduct").
		For(Activity).
		Starts(ProductState{ProductScheduled}).
		Ends(ProductTerminal)).
	Step0(attemptStart, attemptStartStep).
	Step1(attemptResult, attemptResultStep).
	Step1(control, controlStep).
	Step0(workerStop, workerStopStep).
	Step0(timeout, timeoutStep)

// statuses is every status an activity machine records, each confirmed by the status observation
// of its name.
var statuses = []string{"statusScheduled", "statusStarted", "statusPaused", "statusCancelRequested",
	"statusCompleted", "statusFailed", "statusCanceled", "statusTerminated", "statusTimedOut"}

func evidenceForStatuses[S, O, F any](m *umpire.Machine[S, O, F]) *umpire.Machine[S, O, F] {
	for _, s := range statuses {
		m.Evidence(s, s)
	}
	return m
}

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the pause a running attempt
// turns into a pause request, the three timers the start request sets, and the attempt count. The
// machine begins before the activity exists, so unstarted is a phase and the start request is what
// sets the deadlines.

// Phase is the protocol activity's phase.
type Phase string

const (
	Unstarted       Phase = "unstarted"
	Scheduled       Phase = "scheduled"
	BackingOff      Phase = "backingOff"
	Started         Phase = "started"
	Paused          Phase = "paused"
	PauseRequested  Phase = "pauseRequested"
	CancelRequested Phase = "cancelRequested"
	PhaseCompleted  Phase = "completed"
	PhaseFailed     Phase = "failed"
	PhaseCanceled   Phase = "canceled"
	Terminated      Phase = "terminated"
	TimedOut        Phase = "timedOut"
)

func (Phase) Values() []Phase {
	return []Phase{Unstarted, Scheduled, BackingOff, Started, Paused, PauseRequested, CancelRequested,
		PhaseCompleted, PhaseFailed, PhaseCanceled, Terminated, TimedOut}
}

// TimeoutType is which deadline fired.
type TimeoutType string

const (
	TypeScheduleToClose TimeoutType = "scheduleToClose"
	TypeScheduleToStart TimeoutType = "scheduleToStart"
	TypeStartToClose    TimeoutType = "startToClose"
)

func (TimeoutType) Values() []TimeoutType {
	return []TimeoutType{TypeScheduleToClose, TypeScheduleToStart, TypeStartToClose}
}

// AttemptBound bounds the attempt count.
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

// ProtocolState is the protocol machine's state: 12 phases, 3 attempt counts and 3 deadline flags,
// 288 states, which is past the Lean elaborator's bound of 256.
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
	StatusScheduled       struct{}
	StatusStarted         struct{}
	StatusPaused          struct{}
	StatusCancelRequested struct{}
	StatusCompleted       struct{}
	StatusFailed          struct{}
	StatusCanceled        struct{}
	StatusTerminated      struct{}
	StatusTimedOut        struct{ TimeoutType TimeoutType }
	AttemptCountRead      struct{}
)

func (StatusScheduled) isProtocolFact()       {}
func (StatusStarted) isProtocolFact()         {}
func (StatusPaused) isProtocolFact()          {}
func (StatusCancelRequested) isProtocolFact() {}
func (StatusCompleted) isProtocolFact()       {}
func (StatusFailed) isProtocolFact()          {}
func (StatusCanceled) isProtocolFact()        {}
func (StatusTerminated) isProtocolFact()      {}
func (StatusTimedOut) isProtocolFact()        {}
func (AttemptCountRead) isProtocolFact()      {}

// Key spells the fact as the observation it is read through.
func (AttemptCountRead) Key() string { return "attemptCount" }

var _ = umpire.Sum[ProtocolFact](StatusScheduled{}, StatusStarted{}, StatusPaused{},
	StatusCancelRequested{}, StatusCompleted{}, StatusFailed{}, StatusCanceled{}, StatusTerminated{},
	StatusTimedOut{}, AttemptCountRead{})

// ProtocolStep is one protocol step.
type ProtocolStep = umpire.Step[ProtocolState, Outcome, ProtocolFact]

func terminalPhase(p Phase) bool {
	return p == PhaseCompleted || p == PhaseFailed || p == PhaseCanceled || p == Terminated || p == TimedOut
}

// running is started and not over: the phases a deadline can fire in.
func running(p Phase) bool {
	return p == Scheduled || p == BackingOff || p == Started || p == Paused || p == PauseRequested ||
		p == CancelRequested
}

// attemptHeld is where a worker holds the attempt: the phases a start-to-close deadline covers and
// a worker's answer settles.
func attemptHeld(p Phase) bool { return p == Started || p == PauseRequested || p == CancelRequested }

func moves(s ProtocolState, phase Phase, recorded ...ProtocolFact) []ProtocolStep {
	s.Phase = phase
	return []ProtocolStep{{Outcome: Accepted, State: s, Facts: recorded}}
}

func startStep(s ProtocolState, scheduleToClose, scheduleToStart, startToClose Timeout) []ProtocolStep {
	if s.Phase != Unstarted {
		return nil
	}
	return []ProtocolStep{{Outcome: Accepted,
		State: ProtocolState{Phase: Scheduled, Attempts: 0, ScheduleToClose: scheduleToClose,
			ScheduleToStart: scheduleToStart, StartToClose: startToClose},
		Facts: []ProtocolFact{StatusScheduled{}}}}
}

// protocolAttemptStartStep: the worker's poll takes the attempt and raises the count the caller
// reads back.
func protocolAttemptStartStep(s ProtocolState) []ProtocolStep {
	if s.Phase != Scheduled {
		return nil
	}
	s.Attempts = s.Attempts.saturatingSucc()
	return moves(s, Started, StatusStarted{}, AttemptCountRead{})
}

// protocolAttemptResultStep: the worker's answer, by the phase it lands in. A retryable failure
// backs a started attempt off, which the caller reads as scheduled again with a higher attempt
// count, settles a cancel-requested one as canceled, and lands a pause-requested one in paused
// (TransitionAttemptFailedWhilePauseRequested). A canceled answer is honored only under a cancel
// request.
func protocolAttemptResultStep(s ProtocolState, result AttemptResult) []ProtocolStep {
	if !attemptHeld(s.Phase) {
		return nil
	}
	switch r := result.(type) {
	case Completed:
		return moves(s, PhaseCompleted, StatusCompleted{})
	case Failed:
		switch {
		case !r.Retryable:
			return moves(s, PhaseFailed, StatusFailed{})
		case s.Phase == CancelRequested:
			return moves(s, PhaseCanceled, StatusCanceled{})
		case s.Phase == PauseRequested:
			return moves(s, Paused, StatusPaused{})
		default:
			steps := moves(s, BackingOff, StatusScheduled{}, AttemptCountRead{})
			steps[0].Because = "a retryable failure backs off; the caller reads scheduled again"
			return steps
		}
	case CanceledByRun:
		if s.Phase == CancelRequested {
			return moves(s, PhaseCanceled, StatusCanceled{})
		}
		return nil
	default:
		// Unreachable: go-check-sumtype requires every AttemptResult variant above.
		return nil
	}
}

// protocolControlStep: the caller's controls. A pause of a held attempt is a request the worker
// learns of on its next heartbeat, so it is its own phase; the caller reads it as paused either way.
func protocolControlStep(s ProtocolState, c Control) []ProtocolStep {
	if terminalPhase(s.Phase) {
		return []ProtocolStep{{Outcome: ControlLost, State: s}}
	}
	if s.Phase == Unstarted {
		return nil
	}
	switch c {
	case Pause:
		switch s.Phase {
		case Scheduled, BackingOff:
			return moves(s, Paused, StatusPaused{})
		case Started:
			steps := moves(s, PauseRequested, StatusPaused{})
			steps[0].Because = "the worker learns of the pause on its next heartbeat"
			return steps
		case Unstarted, Paused, PauseRequested, CancelRequested, PhaseCompleted, PhaseFailed, PhaseCanceled,
			Terminated, TimedOut:
			return nil
		}
	case Unpause:
		switch s.Phase {
		case Paused:
			return moves(s, Scheduled, StatusScheduled{})
		case PauseRequested:
			return moves(s, Started, StatusStarted{})
		case Unstarted, Scheduled, BackingOff, Started, CancelRequested, PhaseCompleted, PhaseFailed,
			PhaseCanceled, Terminated, TimedOut:
			return nil
		}
	case RequestCancel:
		return moves(s, CancelRequested, StatusCancelRequested{})
	case Terminate:
		return moves(s, Terminated, StatusTerminated{})
	default:
	}
	// Unreachable: exhaustive requires every Control and Phase above.
	return nil
}

// protocolWorkerStopStep: the worker stopping keeps the state and records nothing; on a path it is
// confirmed by the evidence of the step after it, and the Case says so in a Known Gap.
func protocolWorkerStopStep(s ProtocolState) []ProtocolStep {
	return []ProtocolStep{{Outcome: Accepted, State: s}}
}

// backoffStep: the backoff timer. A retry writes nothing the caller can read.
func backoffStep(s ProtocolState) []ProtocolStep {
	if s.Phase != BackingOff {
		return nil
	}
	return moves(s, Scheduled)
}

func scheduleToCloseStep(s ProtocolState) []ProtocolStep {
	if running(s.Phase) && s.ScheduleToClose == Expires {
		return moves(s, TimedOut, StatusTimedOut{TypeScheduleToClose})
	}
	return nil
}

func scheduleToStartStep(s ProtocolState) []ProtocolStep {
	if (s.Phase == Scheduled || s.Phase == BackingOff) && s.ScheduleToStart == Expires {
		return moves(s, TimedOut, StatusTimedOut{TypeScheduleToStart})
	}
	return nil
}

func startToCloseStep(s ProtocolState) []ProtocolStep {
	if attemptHeld(s.Phase) && s.StartToClose == Expires {
		return moves(s, TimedOut, StatusTimedOut{TypeStartToClose})
	}
	return nil
}

// productOf is how a protocol state reads as a product state: not yet started and backing off read
// as scheduled; a pause request reads as started, because the worker still holds the attempt and
// every answer it can give is a row the product has from started, while the request itself is a
// stutter; every other phase is its namesake.
func productOf(s ProtocolState) ProductState {
	switch s.Phase {
	case Unstarted, Scheduled, BackingOff:
		return ProductState{ProductScheduled}
	case Started, PauseRequested:
		return ProductState{ProductStarted}
	case Paused:
		return ProductState{ProductPaused}
	case CancelRequested:
		return ProductState{ProductCancelRequested}
	case PhaseCompleted:
		return ProductState{ProductCompleted}
	case PhaseFailed:
		return ProductState{ProductFailed}
	case PhaseCanceled:
		return ProductState{ProductCanceled}
	case Terminated:
		return ProductState{ProductTerminated}
	case TimedOut:
		return ProductState{ProductTimedOut}
	}
	// Unreachable: exhaustive requires every Phase above.
	return ProductState{}
}

var (
	backoff         = umpire.Timer("backoff")
	scheduleToClose = umpire.Timer("scheduleToClose")
	scheduleToStart = umpire.Timer("scheduleToStart")
	startToClose    = umpire.Timer("startToClose")
)

// ActivityProtocol is the protocol machine.
var ActivityProtocol = evidenceForStatuses(
	umpire.NewMachine[ProtocolState, Outcome, ProtocolFact](Family, "activityProtocol").
		For(Activity).
		Refines(ActivityProduct, productOf).
		Starts(ProtocolState{Phase: Unstarted, ScheduleToClose: Unset, ScheduleToStart: Unset, StartToClose: Unset}).
		Ends(func(s ProtocolState) bool { return terminalPhase(s.Phase) }).
		Unobservable(backoff).
		Evidence("attemptCount", "attemptCount")).
	Step3(start, startStep).
	Step0(attemptStart, protocolAttemptStartStep).
	Step1(attemptResult, protocolAttemptResultStep).
	Step1(control, protocolControlStep).
	Step0(workerStop, protocolWorkerStopStep).
	Step0(backoff, backoffStep).
	Step0(scheduleToClose, scheduleToCloseStep).
	Step0(scheduleToStart, scheduleToStartStep).
	Step0(startToClose, startToCloseStep)

// ActivityWorker is the activity's view of its worker: it stops and it serves.
var ActivityWorker = worker.PollingMachine.Restrict(Family, "activityWorker",
	worker.WorkerStop.ActionDecl, worker.Serve.ActionDecl)

// ### The activity and its worker
//
// Composed with the worker of the activity's task queue, the stop is the worker's own phase change
// and every attempt start is the worker serving, so an attempt has a row only while the worker
// polls.

// StandaloneActivityState is the composed state.
type StandaloneActivityState struct {
	Activity ProtocolState `umpire:"activity"`
	Worker   worker.State  `umpire:"worker"`
}

// StandaloneActivity composes the protocol machine with the activity's worker.
var StandaloneActivity = umpire.Compose[StandaloneActivityState](Family, "standaloneActivity").
	Member("activity", ActivityProtocol).
	Member("worker", ActivityWorker).
	Sync("workerStop", "activity.workerStop", "worker.workerStop").
	Sync("attemptStart", "activity.attemptStart", "worker.serve").
	Ends(func(s StandaloneActivityState) bool { return terminalPhase(s.Activity.Phase) })

// ProductWithoutControls is the product machine as it stood before the caller's controls were
// added: the revision the generated behavior diff compares against.
func ProductWithoutControls() *umpire.Machine[ProductState, Outcome, ProductFact] {
	return ActivityProduct.Restrict(Family, "activityProduct", attemptStart.ActionDecl, attemptResult.ActionDecl,
		workerStop.ActionDecl, timeout.ActionDecl)
}
