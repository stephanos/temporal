// authoring: header

// Package standaloneactivity is the standalone activity Model.
//
// A Temporal activity started directly through `StartActivityExecution`, with no workflow. Grounded in
// `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is deferred (like
// cancellation in the Nexus Model) and not modeled; the heartbeat timeout is not modeled. Standalone
// activities write no history events, so every fact is a status read through
// `DescribeActivityExecution` or a result read through `PollActivityExecution`.
//
// Read from top to bottom: vocabulary → the two machines → what they promise → what the set asks.
package standaloneactivity

import (
	"fmt"

	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/umpire/model/common"
	"go.temporal.io/umpire/model/umpire"
	"go.temporal.io/umpire/model/worker"
)

//go:generate go run ../umpire/cmd/finite -type=Control,ProductPhase,ProductOutcome,ProductFact,Phase,TimeoutType,ProtocolOutcome

// authoring: entities

// ### Entities
//
// An activity started on its own is named by the id the caller gave it; every status and result read
// carries that id.

var activity = &umpire.Entity{Name: "activity", Key: "activityId"}

// authoring: domains

// ### The input domains
//
// Timeout and Delivery are common.Timeout and common.Delivery, shared with the Nexus Model.
// `AttemptFailed{Retryable bool}` is one constructor and two classes, the granularity an example is
// written at.

//sumtype:decl
type AttemptResult interface{ isAttemptResult() }

type (
	AttemptCompleted struct{}
	AttemptFailed    struct{ Retryable bool }
	AttemptCanceled  struct{}
)

func (AttemptCompleted) isAttemptResult() {}
func (AttemptFailed) isAttemptResult()    {}
func (AttemptCanceled) isAttemptResult()  {}

var attemptResults = umpire.Sum[AttemptResult](AttemptCompleted{}, AttemptFailed{}, AttemptCanceled{})

type Control uint8

const (
	Pause Control = iota
	Unpause
	RequestCancel
	Terminate
)

// authoring: actions

// ### Actions
//
// The caller starts and controls the activity; the worker starts and answers attempts. A fault is an
// ordinary action of a declared party, and a timer is `system` behavior the machine owns.

var start = &umpire.Action3[common.Timeout, common.Timeout, common.Timeout]{
	Name:    "start",
	Party:   umpire.Caller,
	Creates: activity,
	Schema:  umpire.Schema(&workflowservice.StartActivityExecutionRequest{}),
	Inputs:  [3]string{"scheduleToClose", "scheduleToStart", "startToClose"},
}

// The worker's poll receives the task.
var attemptStart = &umpire.Action0{
	Name:   "attemptStart",
	Party:  umpire.Worker,
	On:     activity,
	Schema: umpire.Schema(&workflowservice.PollActivityTaskQueueResponse{}),
}

var attemptResult = &umpire.Action1[AttemptResult]{
	Name:  "attemptResult",
	Party: umpire.Worker,
	On:    activity,
	Schema: umpire.Schema(
		&workflowservice.RespondActivityTaskCompletedRequest{},
		&workflowservice.RespondActivityTaskFailedRequest{},
		&workflowservice.RespondActivityTaskCanceledRequest{},
	),
	Input:   "result",
	Classes: attemptResults,
	Examples: map[AttemptResult]string{
		AttemptFailed{Retryable: false}: "ApplicationFailure nonRetryable",
		AttemptFailed{Retryable: true}:  "ApplicationFailure retryable",
	},
}

var control = &umpire.Action1[Control]{
	Name:  "control",
	Party: umpire.Caller,
	On:    activity,
	Schema: umpire.Schema(
		&workflowservice.PauseActivityExecutionRequest{},
		&workflowservice.UnpauseActivityExecutionRequest{},
		&workflowservice.RequestCancelActivityExecutionRequest{},
		&workflowservice.TerminateActivityExecutionRequest{},
	),
	Input:   "control",
	Results: umpire.Results[common.Delivery](),
}

// The worker stops polling. It names no entity, so the machines keep their state and record nothing.
var workerStop = &umpire.Action0{Name: "workerStop", Party: umpire.Worker}

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
// The attempt count is a field of `DescribeActivityExecution`, not a status, so it is a derived read.

var attemptCount = &umpire.Observation{Name: "attemptCount", On: activity, Read: "attempt"}

// authoring: product

// ### The product machine
//
// What the caller sees through Describe, with no account of how the server gets there. Unlike the
// Nexus product, a retry is visible here: a retryable failure puts the status back to SCHEDULED
// (TransitionRescheduled), and Describe reads it, so the product machine has a row for it. What it
// still cannot see is the backoff between the failure and the next dispatch.

type ProductPhase uint8

const (
	ProductScheduled ProductPhase = iota
	ProductStarted
	ProductPaused
	ProductCancelRequested
	ProductCompleted
	ProductFailed
	ProductCanceled
	ProductTerminated
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

type ProductFact uint8

const (
	ProductStatusScheduled ProductFact = iota
	ProductStatusStarted
	ProductStatusPaused
	ProductStatusCancelRequested
	ProductStatusCompleted
	ProductStatusFailed
	ProductStatusCanceled
	ProductStatusTerminated
	ProductStatusTimedOut
)

type (
	ProductStep     = umpire.Step[ProductState, ProductOutcome, ProductFact]
	productProperty = umpire.Property[ProductState, ProductOutcome, ProductFact]
)

func productStep(phase ProductPhase, recorded ProductFact) []ProductStep {
	return []ProductStep{{Outcome: ProductAccepted, State: ProductState{Phase: phase}, Facts: []ProductFact{recorded}}}
}

// The five phases the product machine ends on.
func productTerminal(state ProductState) bool {
	switch state.Phase {
	case ProductCompleted, ProductFailed, ProductCanceled, ProductTerminated, ProductTimedOut:
		return true
	case ProductScheduled, ProductStarted, ProductPaused, ProductCancelRequested:
		return false
	default:
		panic(fmt.Sprintf("unhandled ProductPhase %v", state.Phase))
	}
}

// A worker picks the task up. Only a scheduled activity is dispatched.
func attemptStartStep(state ProductState) []ProductStep {
	if state.Phase != ProductScheduled {
		return nil
	}
	return productStep(ProductStarted, ProductStatusStarted)
}

// The worker answers the attempt it holds: one it started, or one whose cancel was requested while it
// held it. A retryable failure of a started attempt reads as scheduled again; one of a cancel-requested
// attempt honors the cancel.
func attemptResultStep(state ProductState, result AttemptResult) []ProductStep {
	if state.Phase != ProductStarted && state.Phase != ProductCancelRequested {
		return nil
	}
	switch result := result.(type) {
	case AttemptCompleted:
		return productStep(ProductCompleted, ProductStatusCompleted)
	case AttemptFailed:
		if !result.Retryable {
			return productStep(ProductFailed, ProductStatusFailed)
		}
		if state.Phase == ProductCancelRequested {
			return productStep(ProductCanceled, ProductStatusCanceled)
		}
		return productStep(ProductScheduled, ProductStatusScheduled)
	case AttemptCanceled:
		// A worker cancels only an attempt whose cancel was requested.
		if state.Phase != ProductCancelRequested {
			return nil
		}
		return productStep(ProductCanceled, ProductStatusCanceled)
	default:
		panic(fmt.Sprintf("unhandled AttemptResult %T", result))
	}
}

// The caller's control calls. A call after the activity is over is not found and changes nothing. A
// cancel request of an activity already cancel-requested is idempotent and records the status again.
func controlStep(state ProductState, ctl Control) []ProductStep {
	if productTerminal(state) {
		return []ProductStep{{Outcome: ProductNotFound, State: state}}
	}
	switch ctl {
	case Pause:
		if state.Phase == ProductScheduled || state.Phase == ProductStarted {
			return productStep(ProductPaused, ProductStatusPaused)
		}
		return nil
	case Unpause:
		if state.Phase == ProductPaused {
			return productStep(ProductScheduled, ProductStatusScheduled)
		}
		return nil
	case RequestCancel:
		return productStep(ProductCancelRequested, ProductStatusCancelRequested)
	case Terminate:
		return productStep(ProductTerminated, ProductStatusTerminated)
	default:
		panic(fmt.Sprintf("unhandled Control %v", ctl))
	}
}

// The worker stopping is invisible to the product machine, for the reason the Nexus Model gives: a
// step that kept the state and recorded nothing would be read as every stutter by the refinement.
func workerStopStep(ProductState) []ProductStep { return nil }

// One of the activity's deadlines firing. Which one is the protocol's account of how, so the product
// machine has one timer, and it fires while the activity is not over.
func timeoutStep(state ProductState) []ProductStep {
	if productTerminal(state) {
		return nil
	}
	return productStep(ProductTimedOut, ProductStatusTimedOut)
}

var ActivityProduct = &umpire.Machine[ProductState, ProductOutcome, ProductFact]{
	Name:   "activityProduct",
	For:    activity,
	States: umpire.Fields[ProductState](),
	Starts: []ProductState{{Phase: ProductScheduled}},
	Ends:   productTerminal,
	Timers: []umpire.Action{timeout},
	Evidence: map[ProductFact]string{
		ProductStatusScheduled:       "statusScheduled",
		ProductStatusStarted:         "statusStarted",
		ProductStatusPaused:          "statusPaused",
		ProductStatusCancelRequested: "statusCancelRequested",
		ProductStatusCompleted:       "statusCompleted",
		ProductStatusFailed:          "statusFailed",
		ProductStatusCanceled:        "statusCanceled",
		ProductStatusTerminated:      "statusTerminated",
		ProductStatusTimedOut:        "statusTimedOut",
	},
	Steps: umpire.Steps(
		umpire.Bind0(attemptStart, attemptStartStep),
		umpire.Bind1(attemptResult, attemptResultStep),
		umpire.Bind1(control, controlStep),
		umpire.Bind0(workerStop, workerStopStep),
		umpire.Bind0(timeout, timeoutStep),
	),
}

// authoring: protocol

// ### The protocol machine
//
// How the server gets there: the retry and its backoff, the pause a worker has to yield to, the three
// timers the start request sets, and the attempt count. Written against the same actions, so a
// Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the activity exists: `Unstarted` is the "no instance yet" member, and it is
// what makes the deadline fields reachable at anything but their first value.

type Phase uint8

const (
	Unstarted Phase = iota
	Scheduled
	BackingOff
	Started
	Paused
	PauseRequested
	CancelRequested
	Completed
	Failed
	Canceled
	Terminated
	TimedOut
)

// TimeoutType is which timer fired; the Describe status carries the failure that names it.
type TimeoutType uint8

const (
	ScheduleToClose TimeoutType = iota
	ScheduleToStart
	StartToClose
)

// The attempt count is bounded here, as in the Nexus Model, and the saturating successor keeps a retry
// inside the bound.
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

// ProtocolFact is the product's status reads, with the timed-out status carrying which deadline
// fired, plus the derived attempt count.
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
	AttemptCount          struct{}
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
func (AttemptCount) isProtocolFact()          {}

type (
	ProtocolStep     = umpire.Step[ProtocolState, ProtocolOutcome, ProtocolFact]
	protocolProperty = umpire.Property[ProtocolState, ProtocolOutcome, ProtocolFact]
	protocolScenario = umpire.Scenario[ProtocolState, ProtocolOutcome, ProtocolFact]
)

// The five phases the design ends on. A control call after one of them is not found.
func terminalPhase(phase Phase) bool {
	return phase == Completed || phase == Failed || phase == Canceled || phase == Terminated || phase == TimedOut
}

// Started and not yet over: every phase the schedule-to-close deadline covers.
func running(phase Phase) bool {
	return phase != Unstarted && !terminalPhase(phase)
}

// A worker holds the attempt: it was started, and the caller may have asked for a pause or a cancel
// of it meanwhile.
func held(phase Phase) bool {
	return phase == Started || phase == PauseRequested || phase == CancelRequested
}

func moves(state ProtocolState, phase Phase, recorded ...ProtocolFact) []ProtocolStep {
	state.Phase = phase
	return []ProtocolStep{{Outcome: Accepted, State: state, Facts: recorded}}
}

// The caller's start request. It names the three deadlines, and each is a state field because whether
// a timer fires is a question about the activity, not about the request that started it.
func startStep(state ProtocolState, scheduleToClose, scheduleToStart, startToClose common.Timeout) []ProtocolStep {
	if state.Phase != Unstarted {
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
		Facts: []ProtocolFact{StatusScheduled{}},
	}}
}

// A worker picks the task up: the attempt count rises, and Describe reads both the status and the
// count.
func protocolAttemptStartStep(state ProtocolState) []ProtocolStep {
	if state.Phase != Scheduled {
		return nil
	}
	state.Attempts = state.Attempts.saturatingSucc()
	return moves(state, Started, StatusStarted{}, AttemptCount{})
}

// The worker answers the attempt it holds. What the product machine cannot see is where a retryable
// failure yields to, which is whatever the caller asked for meanwhile: nothing, and the attempt backs
// off; a cancel, and the cancel is honored rather than a retry (in statemachine.go CANCEL_REQUESTED is
// a source of Canceled, not of Rescheduled); a pause, and the activity pauses
// (TransitionAttemptFailedWhilePauseRequested). A worker cancels only an attempt whose cancel was
// requested.
func protocolAttemptResultStep(state ProtocolState, result AttemptResult) []ProtocolStep {
	if !held(state.Phase) {
		return nil
	}
	switch result := result.(type) {
	case AttemptCompleted:
		return moves(state, Completed, StatusCompleted{})
	case AttemptFailed:
		if !result.Retryable {
			return moves(state, Failed, StatusFailed{})
		}
		if state.Phase == CancelRequested {
			return moves(state, Canceled, StatusCanceled{})
		}
		if state.Phase == PauseRequested {
			return moves(state, Paused, StatusPaused{})
		}
		return moves(state, BackingOff, AttemptCount{})
	case AttemptCanceled:
		if state.Phase != CancelRequested {
			return nil
		}
		return moves(state, Canceled, StatusCanceled{})
	default:
		panic(fmt.Sprintf("unhandled AttemptResult %T", result))
	}
}

// The caller's control calls. A pause of an activity no worker holds pauses it (TransitionPaused); a
// pause of one a worker holds asks the worker to yield (TransitionPauseRequested), and Describe reads
// PAUSE_REQUESTED, which the product cannot tell from PAUSED, so both record statusPaused. An unpause
// returns the activity to where the pause caught it. A cancel request of any running activity is
// accepted, once more if already requested; a termination ends any running activity.
func protocolControlStep(state ProtocolState, ctl Control) []ProtocolStep {
	if terminalPhase(state.Phase) {
		return []ProtocolStep{{Outcome: NotFound, State: state}}
	}
	if state.Phase == Unstarted {
		return nil
	}
	switch ctl {
	case Pause:
		if state.Phase == Scheduled || state.Phase == BackingOff {
			return moves(state, Paused, StatusPaused{})
		}
		if state.Phase == Started {
			return moves(state, PauseRequested, StatusPaused{})
		}
		return nil
	case Unpause:
		if state.Phase == Paused {
			return moves(state, Scheduled, StatusScheduled{})
		}
		if state.Phase == PauseRequested {
			return moves(state, Started, StatusStarted{})
		}
		return nil
	case RequestCancel:
		return moves(state, CancelRequested, StatusCancelRequested{})
	case Terminate:
		return moves(state, Terminated, StatusTerminated{})
	default:
		panic(fmt.Sprintf("unhandled Control %v", ctl))
	}
}

// The worker stopping keeps the state and records nothing; on a path it is confirmed by the evidence
// of the step after it.
func protocolWorkerStopStep(state ProtocolState) []ProtocolStep {
	return []ProtocolStep{{Outcome: Accepted, State: state}}
}

// The backoff timer returns a backed-off attempt to the queue and records nothing.
func backoffStep(state ProtocolState) []ProtocolStep {
	if state.Phase != BackingOff {
		return nil
	}
	return moves(state, Scheduled)
}

// The schedule-to-close deadline covers the whole activity, so it fires in every running phase -- and
// only when the start request set it.
func scheduleToCloseStep(state ProtocolState) []ProtocolStep {
	if running(state.Phase) && state.ScheduleToClose == common.Expires {
		return moves(state, TimedOut, StatusTimedOut{TimeoutType: ScheduleToClose})
	}
	return nil
}

// The schedule-to-start deadline covers the wait for a worker, so it stops at the start.
func scheduleToStartStep(state ProtocolState) []ProtocolStep {
	if (state.Phase == Scheduled || state.Phase == BackingOff) && state.ScheduleToStart == common.Expires {
		return moves(state, TimedOut, StatusTimedOut{TimeoutType: ScheduleToStart})
	}
	return nil
}

// The start-to-close deadline covers the worker's own work, so it runs while a worker holds the attempt.
func startToCloseStep(state ProtocolState) []ProtocolStep {
	if held(state.Phase) && state.StartToClose == common.Expires {
		return moves(state, TimedOut, StatusTimedOut{TimeoutType: StartToClose})
	}
	return nil
}

// How a protocol state reads as a product state. Backing off and not yet started both read as
// scheduled; a requested pause reads as started, because the worker still holds the attempt and the
// product sees the pause only once the worker yields; every other phase by name. The other fields are
// hidden, which is what a map that does not read them says. The refinement is by mapped states: a
// protocol row is fine if its mapped states are equal or the product has any row between them.
func productOf(state ProtocolState) ProductState {
	switch state.Phase {
	case Unstarted, Scheduled, BackingOff:
		return ProductState{Phase: ProductScheduled}
	case Started, PauseRequested:
		return ProductState{Phase: ProductStarted}
	case Paused:
		return ProductState{Phase: ProductPaused}
	case CancelRequested:
		return ProductState{Phase: ProductCancelRequested}
	case Completed:
		return ProductState{Phase: ProductCompleted}
	case Failed:
		return ProductState{Phase: ProductFailed}
	case Canceled:
		return ProductState{Phase: ProductCanceled}
	case Terminated:
		return ProductState{Phase: ProductTerminated}
	case TimedOut:
		return ProductState{Phase: ProductTimedOut}
	default:
		panic(fmt.Sprintf("unhandled Phase %v", state.Phase))
	}
}

var ActivityProtocol = &umpire.Machine[ProtocolState, ProtocolOutcome, ProtocolFact]{
	Name:         "activityProtocol",
	For:          activity,
	States:       umpire.Fields[ProtocolState](),
	Refines:      umpire.Refines(ActivityProduct, productOf),
	Starts:       []ProtocolState{{Phase: Unstarted}},
	Ends:         func(s ProtocolState) bool { return terminalPhase(s.Phase) },
	Timers:       []umpire.Action{backoff, scheduleToClose, scheduleToStart, startToClose},
	Unobservable: []umpire.Action{backoff},
	Evidence: map[ProtocolFact]string{
		StatusScheduled{}:       "statusScheduled",
		StatusStarted{}:         "statusStarted",
		StatusPaused{}:          "statusPaused",
		StatusCancelRequested{}: "statusCancelRequested",
		StatusCompleted{}:       "statusCompleted",
		StatusFailed{}:          "statusFailed",
		StatusCanceled{}:        "statusCanceled",
		StatusTerminated{}:      "statusTerminated",
		StatusTimedOut{}:        "statusTimedOut",
		AttemptCount{}:          attemptCount.Name,
	},
	Steps: umpire.Steps(
		umpire.Bind3(start, startStep),
		umpire.Bind0(attemptStart, protocolAttemptStartStep),
		umpire.Bind1(attemptResult, protocolAttemptResultStep),
		umpire.Bind1(control, protocolControlStep),
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
// Same-step claims name their action under When; transition claims hold of consecutive steps and are
// verified, never realized.

// Once an activity is over, no step changes its phase.
var terminalIsFinal = &productProperty{
	Name:    "terminalIsFinal",
	Machine: ActivityProduct,
	Transition: func(before, after ProductStep) bool {
		return !productTerminal(before.State) || after.State.Phase == before.State.Phase
	},
}

// A completed attempt completes the activity, and the status reads it.
var completes = &protocolProperty{
	Name:    "completes",
	Machine: ActivityProtocol,
	When:    attemptResult.With(AttemptCompleted{}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Completed && step.Records(StatusCompleted{})
	},
}

// A non-retryable failure fails the activity.
var nonRetryableFails = &protocolProperty{
	Name:    "nonRetryableFails",
	Machine: ActivityProtocol,
	When:    attemptResult.With(AttemptFailed{Retryable: false}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Failed && step.Records(StatusFailed{})
	},
}

// completedOnRetry is completed on the second attempt of an activity with no deadline set.
var completedOnRetry = ProtocolState{
	Phase:           Completed,
	Attempts:        2,
	ScheduleToClose: common.Unset,
	ScheduleToStart: common.Unset,
	StartToClose:    common.Unset,
}

// The retried attempt completes the activity on its second attempt.
var retryCompletes = &protocolProperty{
	Name:    "retryCompletes",
	Machine: ActivityProtocol,
	When:    attemptResult.With(AttemptCompleted{}),
	Holds: func(step ProtocolStep) bool {
		return step.State == completedOnRetry && step.Records(StatusCompleted{})
	},
}

// A cancel request is accepted and the status reads it.
var cancelRequestedWhileStarted = &protocolProperty{
	Name:    "cancelRequestedWhileStarted",
	Machine: ActivityProtocol,
	When:    control.With(RequestCancel),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == CancelRequested && step.Records(StatusCancelRequested{})
	},
}

// The worker's cancellation of a cancel-requested attempt cancels the activity.
var canceledByWorker = &protocolProperty{
	Name:    "canceledByWorker",
	Machine: ActivityProtocol,
	When:    attemptResult.With(AttemptCanceled{}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Canceled && step.Records(StatusCanceled{})
	},
}

// A termination ends the activity.
var terminated = &protocolProperty{
	Name:    "terminated",
	Machine: ActivityProtocol,
	When:    control.With(Terminate),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Terminated && step.Records(StatusTerminated{})
	},
}

// A paused activity is never dispatched: no step takes it from paused to started.
var pausedIsNotDispatched = &productProperty{
	Name:    "pausedIsNotDispatched",
	Machine: ActivityProduct,
	Transition: func(before, after ProductStep) bool {
		return before.State.Phase != ProductPaused || after.State.Phase != ProductStarted
	},
}

// The schedule-to-start deadline settles an activity no worker started as timed out, and the status
// records which deadline it was.
var scheduleToStartFires = &protocolProperty{
	Name:    "scheduleToStartFires",
	Machine: ActivityProtocol,
	When:    scheduleToStart,
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == TimedOut && step.Records(StatusTimedOut{TimeoutType: ScheduleToStart})
	},
}

// The start-to-close deadline settles a started activity no worker answered as timed out.
var startToCloseFires = &protocolProperty{
	Name:    "startToCloseFires",
	Machine: ActivityProtocol,
	When:    startToClose,
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == TimedOut && step.Records(StatusTimedOut{TimeoutType: StartToClose})
	},
}

// authoring: scenarios

// ### The paths the Queries run
//
// Each path starts unstarted and, unless said, starts the activity with no deadline set.

var unstarted = ProtocolState{Phase: Unstarted}

var noDeadlines = start.With(common.Unset, common.Unset, common.Unset)

var completed = &protocolScenario{
	Name:   "completed",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		noDeadlines, attemptStart, attemptResult.With(AttemptCompleted{}),
	},
}

var nonRetryable = &protocolScenario{
	Name:   "nonRetryable",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		noDeadlines, attemptStart, attemptResult.With(AttemptFailed{Retryable: false}),
	},
}

// The retryable failure backs the attempt off; the backoff timer fires and records nothing; the
// retried attempt completes.
var retriedThenCompleted = &protocolScenario{
	Name:   "retriedThenCompleted",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		noDeadlines, attemptStart, attemptResult.With(AttemptFailed{Retryable: true}),
		backoff, attemptStart, attemptResult.With(AttemptCompleted{}),
	},
}

var cancelRequestedThenCanceled = &protocolScenario{
	Name:   "cancelRequestedThenCanceled",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		noDeadlines, attemptStart, control.With(RequestCancel), attemptResult.With(AttemptCanceled{}),
	},
}

// The worker stops, so nothing picks the task up; the caller terminates the scheduled activity.
var terminatedWhileScheduled = &protocolScenario{
	Name:   "terminatedWhileScheduled",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		noDeadlines, workerStop, control.With(Terminate),
	},
}

var pausedThenCompleted = &protocolScenario{
	Name:   "pausedThenCompleted",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		noDeadlines, control.With(Pause), control.With(Unpause), attemptStart,
		attemptResult.With(AttemptCompleted{}),
	},
}

// The start request sets the schedule-to-start deadline; the worker stops, so nothing picks the task
// up; the deadline fires.
var scheduleToStartExpires = &protocolScenario{
	Name:   "scheduleToStartExpires",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		start.With(common.Unset, common.Expires, common.Unset), workerStop, scheduleToStart,
	},
}

// The start request sets the start-to-close deadline; a worker starts the attempt and never answers;
// the deadline fires.
var startToCloseExpires = &protocolScenario{
	Name:   "startToCloseExpires",
	Model:  ActivityProtocol,
	Starts: unstarted,
	Actions: []umpire.Class{
		start.With(common.Unset, common.Unset, common.Expires), attemptStart, startToClose,
	},
}

var (
	three = &umpire.Limits{Name: "three", Steps: 3, Actions: 3, Search: 4096}
	four  = &umpire.Limits{Name: "four", Steps: 4, Actions: 4, Search: 32768}
	six   = &umpire.Limits{Name: "six", Steps: 6, Actions: 6, Search: 262144}
)

// authoring: queries

// ### The Queries
//
// Eight find their same-step claim on a path and are realized by the functional set; two verify a
// product claim over every trace of a path.

var (
	completion             = &umpire.Query{Name: "completion", Find: completes, In: completed, Limits: three}
	nonRetryableFailure    = &umpire.Query{Name: "nonRetryableFailure", Find: nonRetryableFails, In: nonRetryable, Limits: three}
	retry                  = &umpire.Query{Name: "retry", Find: retryCompletes, In: retriedThenCompleted, Limits: six}
	cancel                 = &umpire.Query{Name: "cancel", Find: canceledByWorker, In: cancelRequestedThenCanceled, Limits: four}
	terminate              = &umpire.Query{Name: "terminate", Find: terminated, In: terminatedWhileScheduled, Limits: three}
	pauseResume            = &umpire.Query{Name: "pauseResume", Find: completes, In: pausedThenCompleted, Limits: six}
	scheduleToStartTimeout = &umpire.Query{Name: "scheduleToStartTimeout", Find: scheduleToStartFires, In: scheduleToStartExpires, Limits: three}
	startToCloseTimeout    = &umpire.Query{Name: "startToCloseTimeout", Find: startToCloseFires, In: startToCloseExpires, Limits: three}
	terminalHolds          = &umpire.Query{Name: "terminalHolds", Verify: terminalIsFinal, In: completed, Limits: three}
	pauseHolds             = &umpire.Query{Name: "pauseHolds", Verify: pausedIsNotDispatched, In: pausedThenCompleted, Limits: six}
)

// authoring: set

// ### The functional set
//
// The Case drives the caller and the worker. No repeat: standalone activities are CHASM only.

var StandaloneActivityTests = &umpire.Set{
	Name:    "standaloneActivityTests",
	Purpose: umpire.Functional,
	Bind: map[umpire.Party]umpire.Role{
		umpire.Caller: umpire.Driven,
		umpire.Worker: umpire.Driven,
	},
	Queries: []*umpire.Query{
		completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
		scheduleToStartTimeout, startToCloseTimeout,
	},
}

// ### The canary set
//
// The deployment performs the worker's part itself, so the worker is observed.

var StandaloneActivityCanary = &umpire.Set{
	Name:    "standaloneActivityCanary",
	Purpose: umpire.Canary,
	Bind: map[umpire.Party]umpire.Role{
		umpire.Caller: umpire.Driven,
		umpire.Worker: umpire.Observed,
	},
	Queries: []*umpire.Query{completion, cancel},
}

// ### The exploratory set

var StandaloneActivityExploration = &umpire.Set{
	Name:    "standaloneActivityExploration",
	Purpose: umpire.Exploratory,
	Bind: map[umpire.Party]umpire.Role{
		umpire.Caller: umpire.Driven,
		umpire.Worker: umpire.Driven,
	},
	Machine: ActivityProtocol,
	Cover:   umpire.Rows | umpire.Results | umpire.ClassMembers,
	Budget:  four,
}

// authoring: case

// ### The Cases
//
// Out of scope for this sample; Testpilot realizes the Sets above.

// authoring: composition

// ### The activity and its worker
//
// Composed with the worker of the activity's task queue, the worker stop is the worker's own phase
// change and every attempt start is the worker serving, so a start has a row only while the worker
// polls.

// ActivityWorker stops and serves; it never resumes, for the reason the Nexus Model gives.
var ActivityWorker = umpire.Restrict("activityWorker", worker.Polling, worker.WorkerStop, worker.Serve)

type StandaloneActivityState struct {
	Activity ProtocolState      `umpire:"activity"`
	Worker   worker.WorkerState `umpire:"worker"`
}

var StandaloneActivity = &umpire.Compose[StandaloneActivityState]{
	Name:    "standaloneActivity",
	For:     []*umpire.Entity{activity, worker.Worker},
	Members: umpire.Members{"activity": ActivityProtocol, "worker": ActivityWorker},
	Sync: umpire.Sync{
		workerStop:   {"activity": workerStop, "worker": worker.WorkerStop},
		attemptStart: {"activity": attemptStart, "worker": worker.Serve},
	},
	Starts: []StandaloneActivityState{{Activity: unstarted, Worker: worker.WorkerState{Phase: worker.PhasePolling}}},
	Ends:   func(s StandaloneActivityState) bool { return terminalPhase(s.Activity.Phase) },
}

// Every attempt start leaves the worker polling: no stopped worker starts an attempt.
var startedByPollingWorker = &umpire.Property[StandaloneActivityState, umpire.Joint, umpire.Joint]{
	Name:    "startedByPollingWorker",
	Machine: StandaloneActivity,
	When:    attemptStart,
	Holds: func(step umpire.Composed[StandaloneActivityState]) bool {
		return step.State.Worker.Phase == worker.PhasePolling
	},
}

// An attempt starts while the worker polls and fails retryably; the worker then stops before the
// retry, so the retried attempt is never started and the schedule-to-start deadline fires. The path
// performs attemptStart once, so the claim is exercised rather than vacuously true.
var stoppedBeforeRetry = &umpire.Scenario[StandaloneActivityState, umpire.Joint, umpire.Joint]{
	Name:   "stoppedBeforeRetry",
	Model:  StandaloneActivity,
	Starts: StandaloneActivityState{Activity: unstarted, Worker: worker.WorkerState{Phase: worker.PhasePolling}},
	Actions: []umpire.Class{
		umpire.At("activity", start.With(common.Unset, common.Expires, common.Unset)),
		attemptStart,
		umpire.At("activity", attemptResult.With(AttemptFailed{Retryable: true})),
		umpire.At("activity", backoff),
		workerStop,
		umpire.At("activity", scheduleToStart),
	},
}

var stoppedWorkerStartsNothing = &umpire.Query{
	Name:   "stoppedWorkerStartsNothing",
	Verify: startedByPollingWorker,
	In:     stoppedBeforeRetry,
	Limits: six,
}

// authoring: end
