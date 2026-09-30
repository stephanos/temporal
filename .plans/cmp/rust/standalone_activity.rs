// authoring: header
//! # The standalone activity Model
//!
//! A Temporal activity started directly through `StartActivityExecution`, with no workflow: the
//! product machine says what the caller sees through `DescribeActivityExecution`, the protocol
//! machine says how the server gets there and refines it. Grounded in
//! `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is deferred
//! (like cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not modeled.
//!
//! Standalone activities write no history events, so every fact is a status read through
//! `DescribeActivityExecution` or a result read through `PollActivityExecution`.
//!
//! Read from top to bottom: vocabulary, the two machines, what they promise, what the set asks.

use umpire::{
    action, compose, entity, limits, machine, observation, property, query, scenario, set, Bounded,
    Finite, Step,
};

use crate::worker::{self, WorkerState};

// authoring: entities

// ### Entities
//
// An activity is named by the id the caller chose for it.

entity! { activity key: activityId }

// authoring: domains

// ### The input domains
//
// `Failed { retryable: bool }` is one variant and two classes, the granularity of an example and
// of a protobuf oneof.

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Timeout {
    Unset,
    Expires,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum AttemptResult {
    Completed,
    Failed { retryable: bool },
    Canceled,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Delivery {
    Accepted,
    NotFound,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Control {
    Pause,
    Unpause,
    RequestCancel,
    Terminate,
}

// Input domains bare, phase enums qualified: `AttemptResult::Completed` is written `Completed`,
// the phase is always `Phase::Completed`.
use AttemptResult::*;
use Control::*;
use Timeout::*;

// authoring: actions

// ### Actions
//
// The caller starts and controls the activity; the worker's poll receives the task and its
// response settles the attempt. A fault is an ordinary action of a declared party, and a timer is
// `system` behavior the machine owns, so neither is a separate kind.

action! { start
    party: caller
    creates: activity
    schema: "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    input: { scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout }
}

/// The worker's poll receives the task (`PollActivityTaskQueue`).
action! { attemptStart
    party: worker
    on: activity
    schema: "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse"
}

action! { attemptResult
    party: worker
    on: activity
    schema: "RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | RespondActivityTaskCanceledRequest"
    input: { result: AttemptResult }
    examples: {
        Failed { retryable: false } => "ApplicationFailure nonRetryable",
        Failed { retryable: true } => "ApplicationFailure retryable",
    }
}

action! { control
    party: caller
    on: activity
    schema: "PauseActivityExecutionRequest | UnpauseActivityExecutionRequest | RequestCancelActivityExecutionRequest | TerminateActivityExecutionRequest"
    input: { control: Control }
    results: Delivery
}

/// The worker stops polling. An action that names no entity is behavior no entity records, so the
/// machines keep their state and record nothing at it. Declared by the worker module.
pub use worker::workerStop;

// authoring: observation

// ### The derived observation
//
// A retryable attempt failure is not a status change, so the attempt count is read back through
// `DescribeActivityExecution`.

observation! { attemptCount
    on: activity
    read: attempt
}

// authoring: product

// ### The product machine
//
// What the caller sees through Describe, with no account of how. Every Property written against it
// is carried to the protocol machine by the refinement declared there. Describe reads a retry, so
// unlike the Nexus product this one sees a retryable failure; what it cannot see is the backoff
// between the failure and the next poll, and the difference between a pause and a requested one.

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum ProductPhase {
    Scheduled,
    Started,
    Paused,
    CancelRequested,
    Completed,
    Failed,
    Canceled,
    Terminated,
    TimedOut,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub struct ProductState {
    pub phase: ProductPhase,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum ProductOutcome {
    Accepted,
    NotFound,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum ProductFact {
    StatusScheduled,
    StatusStarted,
    StatusPaused,
    StatusCancelRequested,
    StatusCompleted,
    StatusFailed,
    StatusCanceled,
    StatusTerminated,
    StatusTimedOut,
}

pub type ProductStep = Step<ProductState, ProductOutcome, ProductFact>;

fn product_step(phase: ProductPhase, recorded: ProductFact) -> Vec<ProductStep> {
    vec![Step { outcome: ProductOutcome::Accepted, state: ProductState { phase }, facts: vec![recorded] }]
}

/// The five phases the product machine ends on.
pub fn product_terminal(state: &ProductState) -> bool {
    matches!(
        state.phase,
        ProductPhase::Completed | ProductPhase::Failed | ProductPhase::Canceled | ProductPhase::Terminated | ProductPhase::TimedOut
    )
}

/// The worker's poll receives the task. Only a scheduled activity has one to receive.
pub fn attempt_start_step(state: &ProductState) -> Vec<ProductStep> {
    match state.phase {
        ProductPhase::Scheduled => product_step(ProductPhase::Started, ProductFact::StatusStarted),
        _ => vec![],
    }
}

/// The worker's response to a started attempt. Unlike the Nexus product, the retry is visible: a
/// retryable failure reads as SCHEDULED again through Describe, with a higher attempt count
/// (statemachine.go's `TransitionRescheduled`), and under a requested cancel it resolves the
/// activity as canceled. A cancel response counts only where a cancel was requested.
pub fn attempt_result_step(state: &ProductState, result: AttemptResult) -> Vec<ProductStep> {
    match (state.phase, result) {
        (ProductPhase::Started | ProductPhase::CancelRequested, Completed) => {
            product_step(ProductPhase::Completed, ProductFact::StatusCompleted)
        }
        (ProductPhase::Started | ProductPhase::CancelRequested, Failed { retryable: false }) => {
            product_step(ProductPhase::Failed, ProductFact::StatusFailed)
        }
        (ProductPhase::Started, Failed { retryable: true }) => product_step(ProductPhase::Scheduled, ProductFact::StatusScheduled),
        (ProductPhase::CancelRequested, Failed { retryable: true } | Canceled) => {
            product_step(ProductPhase::Canceled, ProductFact::StatusCanceled)
        }
        (ProductPhase::Started, Canceled) => vec![],
        (
            ProductPhase::Scheduled
            | ProductPhase::Paused
            | ProductPhase::Completed
            | ProductPhase::Failed
            | ProductPhase::Canceled
            | ProductPhase::Terminated
            | ProductPhase::TimedOut,
            _,
        ) => vec![],
    }
}

/// The caller's control requests. After the activity is over every one is not found and changes
/// nothing; a repeated cancel request is accepted again and reads the same status.
pub fn control_step(state: &ProductState, control: Control) -> Vec<ProductStep> {
    if product_terminal(state) {
        return vec![Step { outcome: ProductOutcome::NotFound, state: *state, facts: vec![] }];
    }
    match (control, state.phase) {
        (Pause, ProductPhase::Scheduled | ProductPhase::Started) => product_step(ProductPhase::Paused, ProductFact::StatusPaused),
        (Pause, _) => vec![],
        (Unpause, ProductPhase::Paused) => product_step(ProductPhase::Scheduled, ProductFact::StatusScheduled),
        (Unpause, _) => vec![],
        (RequestCancel, ProductPhase::Scheduled | ProductPhase::Started | ProductPhase::Paused | ProductPhase::CancelRequested) => {
            product_step(ProductPhase::CancelRequested, ProductFact::StatusCancelRequested)
        }
        (RequestCancel, _) => vec![],
        // Every non-terminal phase: the terminal ones returned above.
        (Terminate, _) => product_step(ProductPhase::Terminated, ProductFact::StatusTerminated),
    }
}

/// The worker stopping is a fault the Run records and the activity does not feel. The product
/// machine cannot see it: a step that kept the state and recorded nothing would be read by the
/// refinement as every stutter.
pub fn worker_stop_step(_state: &ProductState) -> Vec<ProductStep> {
    vec![]
}

/// One of the activity's deadlines firing. Which deadline is the protocol's account of how, so the
/// product machine has one timer, and it fires while the activity is open.
pub fn timeout_step(state: &ProductState) -> Vec<ProductStep> {
    match state.phase {
        ProductPhase::Scheduled | ProductPhase::Started | ProductPhase::CancelRequested | ProductPhase::Paused => {
            product_step(ProductPhase::TimedOut, ProductFact::StatusTimedOut)
        }
        _ => vec![],
    }
}

machine! { activityProduct
    for: activity
    state: ProductState
    outcome: ProductOutcome
    facts: ProductFact
    starts: [Scheduled]
    ends: [Completed, Failed, Canceled, Terminated, TimedOut]
    timers: [timeout]
    evidence: {
        StatusScheduled: statusScheduled,
        StatusStarted: statusStarted,
        StatusPaused: statusPaused,
        StatusCancelRequested: statusCancelRequested,
        StatusCompleted: statusCompleted,
        StatusFailed: statusFailed,
        StatusCanceled: statusCanceled,
        StatusTerminated: statusTerminated,
        StatusTimedOut: statusTimedOut,
    }
    steps: {
        attemptStart: attempt_start_step,
        attemptResult: attempt_result_step,
        control: control_step,
        workerStop: worker_stop_step,
        timeout: timeout_step,
    }
}

// authoring: protocol

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the pause a started attempt
// only requests, the three timers the start request sets, and the attempt count a poll raises.
// Written against the same actions, so a Property proved on the product machine is carried here by
// the refinement.
//
// The machine begins before the activity exists: `Unstarted` is the "no instance yet" member, and
// it is what makes the deadline fields reachable at anything but their first value.

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Phase {
    Unstarted,
    Scheduled,
    BackingOff,
    Started,
    Paused,
    PauseRequested,
    CancelRequested,
    Completed,
    Failed,
    Canceled,
    Terminated,
    TimedOut,
}

/// Which timer fired. The Describe status does not say, so the fact carries it.
#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum TimeoutType {
    ScheduleToClose,
    ScheduleToStart,
    StartToClose,
}

/// The attempt count is bounded by the Limits in the design; the saturating successor keeps a retry
/// inside it.
pub const ATTEMPT_BOUND: u8 = 2;
pub type Attempts = Bounded<ATTEMPT_BOUND>;

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub struct ProtocolState {
    pub phase: Phase,
    pub attempts: Attempts,
    pub schedule_to_close: Timeout,
    pub schedule_to_start: Timeout,
    pub start_to_close: Timeout,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum ProtocolOutcome {
    Accepted,
    NotFound,
}

/// As the product's, with the timed-out status carrying its type and the derived attempt count.
#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum ProtocolFact {
    StatusScheduled,
    StatusStarted,
    StatusPaused,
    StatusCancelRequested,
    StatusCompleted,
    StatusFailed,
    StatusCanceled,
    StatusTerminated,
    StatusTimedOut { timeout_type: TimeoutType },
    AttemptCount,
}

pub type ProtocolStep = Step<ProtocolState, ProtocolOutcome, ProtocolFact>;

/// The five phases the design ends on. A control that arrives after one of them is not found.
pub fn terminal_phase(phase: Phase) -> bool {
    matches!(phase, Phase::Completed | Phase::Failed | Phase::Canceled | Phase::Terminated | Phase::TimedOut)
}

/// Started and not yet over: the phases the schedule-to-close deadline covers.
pub fn running(phase: Phase) -> bool {
    matches!(
        phase,
        Phase::Scheduled | Phase::BackingOff | Phase::Started | Phase::Paused | Phase::PauseRequested | Phase::CancelRequested
    )
}

fn moves(state: &ProtocolState, phase: Phase, recorded: Vec<ProtocolFact>) -> Vec<ProtocolStep> {
    vec![Step { outcome: ProtocolOutcome::Accepted, state: ProtocolState { phase, ..*state }, facts: recorded }]
}

fn not_found(state: &ProtocolState) -> Vec<ProtocolStep> {
    vec![Step { outcome: ProtocolOutcome::NotFound, state: *state, facts: vec![] }]
}

/// The caller's start request. It names the activity's three deadlines, each a state field because
/// whether a timer fires is a question about the activity and not about the request that started
/// it.
pub fn start_step(
    state: &ProtocolState,
    schedule_to_close: Timeout,
    schedule_to_start: Timeout,
    start_to_close: Timeout,
) -> Vec<ProtocolStep> {
    if state.phase != Phase::Unstarted {
        return vec![];
    }
    vec![Step {
        outcome: ProtocolOutcome::Accepted,
        state: ProtocolState {
            phase: Phase::Scheduled,
            attempts: Attempts::ZERO,
            schedule_to_close,
            schedule_to_start,
            start_to_close,
        },
        facts: vec![ProtocolFact::StatusScheduled],
    }]
}

/// The worker's poll receives the task and the attempt count rises. No status change records the
/// count, so it is read back through the `attemptCount` observation.
pub fn protocol_attempt_start_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if state.phase != Phase::Scheduled {
        return vec![];
    }
    vec![Step {
        outcome: ProtocolOutcome::Accepted,
        state: ProtocolState { phase: Phase::Started, attempts: state.attempts.saturating_succ(), ..*state },
        facts: vec![ProtocolFact::StatusStarted, ProtocolFact::AttemptCount],
    }]
}

/// The worker's response to a started attempt. What the product machine cannot see: a retryable
/// failure backs the activity off, and under a requested pause it parks it (statemachine.go's
/// `TransitionAttemptFailedWhilePauseRequested`). Under a requested cancel, statemachine.go makes
/// CANCEL_REQUESTED a source of Completed, Failed, Canceled, TimedOut and Terminated, and a
/// retryable failure there resolves the activity as canceled. A cancel response with no cancel
/// requested is not enabled.
pub fn protocol_attempt_result_step(state: &ProtocolState, result: AttemptResult) -> Vec<ProtocolStep> {
    match (state.phase, result) {
        (Phase::Started, Completed) => moves(state, Phase::Completed, vec![ProtocolFact::StatusCompleted]),
        (Phase::Started, Failed { retryable: false }) => moves(state, Phase::Failed, vec![ProtocolFact::StatusFailed]),
        (Phase::Started, Failed { retryable: true }) => moves(state, Phase::BackingOff, vec![ProtocolFact::AttemptCount]),
        (Phase::Started, Canceled) => vec![],

        (Phase::CancelRequested, Completed) => moves(state, Phase::Completed, vec![ProtocolFact::StatusCompleted]),
        (Phase::CancelRequested, Failed { retryable: false }) => moves(state, Phase::Failed, vec![ProtocolFact::StatusFailed]),
        (Phase::CancelRequested, Failed { retryable: true }) => moves(state, Phase::Canceled, vec![ProtocolFact::StatusCanceled]),
        (Phase::CancelRequested, Canceled) => moves(state, Phase::Canceled, vec![ProtocolFact::StatusCanceled]),

        (Phase::PauseRequested, Completed) => moves(state, Phase::Completed, vec![ProtocolFact::StatusCompleted]),
        (Phase::PauseRequested, Failed { retryable: false }) => moves(state, Phase::Failed, vec![ProtocolFact::StatusFailed]),
        (Phase::PauseRequested, Failed { retryable: true }) => moves(state, Phase::Paused, vec![ProtocolFact::StatusPaused]),
        (Phase::PauseRequested, Canceled) => vec![],

        (
            Phase::Unstarted
            | Phase::Scheduled
            | Phase::BackingOff
            | Phase::Paused
            | Phase::Completed
            | Phase::Failed
            | Phase::Canceled
            | Phase::Terminated
            | Phase::TimedOut,
            _,
        ) => vec![],
    }
}

/// The caller's control requests. A pause of a started attempt is only requested, and the Describe
/// status reads PAUSE_REQUESTED; the fact is `StatusPaused` for both, as the product cannot tell
/// them apart. A cancel request and a termination after the activity is over are not found; a
/// control before it exists is not enabled.
pub fn protocol_control_step(state: &ProtocolState, control: Control) -> Vec<ProtocolStep> {
    match (control, state.phase) {
        (Pause, Phase::Scheduled | Phase::BackingOff) => moves(state, Phase::Paused, vec![ProtocolFact::StatusPaused]),
        (Pause, Phase::Started) => moves(state, Phase::PauseRequested, vec![ProtocolFact::StatusPaused]),
        (Pause, _) => vec![],

        (Unpause, Phase::Paused) => moves(state, Phase::Scheduled, vec![ProtocolFact::StatusScheduled]),
        (Unpause, Phase::PauseRequested) => moves(state, Phase::Started, vec![ProtocolFact::StatusStarted]),
        (Unpause, _) => vec![],

        (RequestCancel, Phase::Unstarted) => vec![],
        (RequestCancel, phase) if terminal_phase(phase) => not_found(state),
        (RequestCancel, _) => moves(state, Phase::CancelRequested, vec![ProtocolFact::StatusCancelRequested]),

        (Terminate, Phase::Unstarted) => vec![],
        (Terminate, phase) if terminal_phase(phase) => not_found(state),
        (Terminate, _) => moves(state, Phase::Terminated, vec![ProtocolFact::StatusTerminated]),
    }
}

/// The worker stopping is a fault the Run records and the activity does not feel, so the step keeps
/// the state and records nothing. On a path it is confirmed by the evidence of the step after it.
pub fn protocol_worker_stop_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    vec![Step { outcome: ProtocolOutcome::Accepted, state: *state, facts: vec![] }]
}

/// The backoff timer. It is what makes `BackingOff` a phase the activity leaves rather than a state
/// it is stuck in, and it records nothing: a retry is not a status change.
pub fn backoff_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if state.phase != Phase::BackingOff {
        return vec![];
    }
    moves(state, Phase::Scheduled, vec![])
}

fn timed_out(state: &ProtocolState, timeout_type: TimeoutType) -> Vec<ProtocolStep> {
    moves(state, Phase::TimedOut, vec![ProtocolFact::StatusTimedOut { timeout_type }])
}

/// The schedule-to-close deadline covers the whole activity, so it fires in every running phase --
/// and only when the start request set it.
pub fn schedule_to_close_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if running(state.phase) && state.schedule_to_close == Expires {
        timed_out(state, TimeoutType::ScheduleToClose)
    } else {
        vec![]
    }
}

/// The schedule-to-start deadline covers the wait for a worker to poll, so it stops at the start.
pub fn schedule_to_start_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if matches!(state.phase, Phase::Scheduled | Phase::BackingOff) && state.schedule_to_start == Expires {
        timed_out(state, TimeoutType::ScheduleToStart)
    } else {
        vec![]
    }
}

/// The start-to-close deadline covers the attempt, requested pause or cancel included, so it begins
/// at the start.
pub fn start_to_close_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if matches!(state.phase, Phase::Started | Phase::PauseRequested | Phase::CancelRequested) && state.start_to_close == Expires {
        timed_out(state, TimeoutType::StartToClose)
    } else {
        vec![]
    }
}

/// How a protocol state reads as a product state. Backing off is still scheduled and an activity
/// not yet started reads as scheduled, because the product machine begins there. A requested pause
/// reads as started: the worker still holds the attempt, so every answer it can give is a product
/// row from started, and the request itself is a stutter. Every other field is hidden.
pub fn product_of(state: &ProtocolState) -> ProductState {
    let phase = match state.phase {
        Phase::Unstarted | Phase::Scheduled | Phase::BackingOff => ProductPhase::Scheduled,
        Phase::Started | Phase::PauseRequested => ProductPhase::Started,
        Phase::Paused => ProductPhase::Paused,
        Phase::CancelRequested => ProductPhase::CancelRequested,
        Phase::Completed => ProductPhase::Completed,
        Phase::Failed => ProductPhase::Failed,
        Phase::Canceled => ProductPhase::Canceled,
        Phase::Terminated => ProductPhase::Terminated,
        Phase::TimedOut => ProductPhase::TimedOut,
    };
    ProductState { phase }
}

machine! { activityProtocol
    for: activity
    state: ProtocolState
    outcome: ProtocolOutcome
    facts: ProtocolFact
    refines: activityProduct
    map: product_of
    starts: [Unstarted]
    ends: [Completed, Failed, Canceled, Terminated, TimedOut]
    timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
    unobservable: [backoff]
    evidence: {
        StatusScheduled: statusScheduled,
        StatusStarted: statusStarted,
        StatusPaused: statusPaused,
        StatusCancelRequested: statusCancelRequested,
        StatusCompleted: statusCompleted,
        StatusFailed: statusFailed,
        StatusCanceled: statusCanceled,
        StatusTerminated: statusTerminated,
        StatusTimedOut: statusTimedOut,
        AttemptCount: attemptCount,
    }
    steps: {
        start: start_step,
        attemptStart: protocol_attempt_start_step,
        attemptResult: protocol_attempt_result_step,
        control: protocol_control_step,
        workerStop: protocol_worker_stop_step,
        backoff: backoff_step,
        scheduleToClose: schedule_to_close_step,
        scheduleToStart: schedule_to_start_step,
        startToClose: start_to_close_step,
    }
}

// authoring: properties

// ### What the machines promise
//
// A same-step claim names the action it is about under `when:` and holds of the step that action
// produces; a transition claim holds of the step before and the step after. A functional Query
// realizes a same-step claim; a transition claim is searched and verified, never realized.

// Once an activity is over, no step changes its phase. Declared on the product machine and read on
// the protocol machine through the map.
property! { terminalIsFinal
    machine: activityProduct
    holds: |before, after| !product_terminal(&before.state) || after.state.phase == before.state.phase
}

// A completed attempt settles the activity as completed, and the status records it.
property! { completes
    machine: activityProtocol
    when: attemptResult(Completed)
    holds: |step| step.state.phase == Phase::Completed && step.facts.contains(&ProtocolFact::StatusCompleted)
}

// A non-retryable failure settles the activity as failed, and the status records it.
property! { nonRetryableFails
    machine: activityProtocol
    when: attemptResult(Failed { retryable: false })
    holds: |step| step.state.phase == Phase::Failed && step.facts.contains(&ProtocolFact::StatusFailed)
}

/// Completed on the second attempt of an activity with no deadline set. A claim fixes one state, so
/// every field is named.
pub const COMPLETED_ON_RETRY: ProtocolState = ProtocolState {
    phase: Phase::Completed,
    attempts: Attempts::LAST,
    schedule_to_close: Unset,
    schedule_to_start: Unset,
    start_to_close: Unset,
};

// A completed retried attempt settles the activity as completed on its second attempt: the count
// the second poll raised is two, and the status records the completion.
property! { retryCompletes
    machine: activityProtocol
    when: attemptResult(Completed)
    holds: |step| step.state == COMPLETED_ON_RETRY && step.facts.contains(&ProtocolFact::StatusCompleted)
}

// A cancel request of a started attempt is recorded as requested; the attempt keeps running.
property! { cancelRequestedWhileStarted
    machine: activityProtocol
    when: control(RequestCancel)
    holds: |step| step.state.phase == Phase::CancelRequested && step.facts.contains(&ProtocolFact::StatusCancelRequested)
}

// The worker honors the cancel request and the status records it.
property! { canceledByWorker
    machine: activityProtocol
    when: attemptResult(Canceled)
    holds: |step| step.state.phase == Phase::Canceled && step.facts.contains(&ProtocolFact::StatusCanceled)
}

// A termination settles the activity, and the status records it.
property! { terminated
    machine: activityProtocol
    when: control(Terminate)
    holds: |step| step.state.phase == Phase::Terminated && step.facts.contains(&ProtocolFact::StatusTerminated)
}

// A paused activity is never dispatched: no step moves it straight to started. Declared on the
// product machine and read on the protocol machine through the map.
property! { pausedIsNotDispatched
    machine: activityProduct
    holds: |before, after| before.state.phase != ProductPhase::Paused || after.state.phase != ProductPhase::Started
}

// The schedule-to-start deadline settles an activity no worker polled as timed out, and the status
// records which deadline it was.
property! { scheduleToStartFires
    machine: activityProtocol
    when: scheduleToStart
    holds: |step| step.state.phase == Phase::TimedOut
        && step.facts.contains(&ProtocolFact::StatusTimedOut { timeout_type: TimeoutType::ScheduleToStart })
}

// The start-to-close deadline settles a started attempt no worker answered as timed out.
property! { startToCloseFires
    machine: activityProtocol
    when: startToClose
    holds: |step| step.state.phase == Phase::TimedOut
        && step.facts.contains(&ProtocolFact::StatusTimedOut { timeout_type: TimeoutType::StartToClose })
}

// authoring: scenarios

// ### The paths the Queries run
//
// A protocol Scenario names its classed actions with their inputs and its start by its phase. The
// start request sets no deadline unless the path is about one.

scenario! { completed
    model: activityProtocol
    starts: Unstarted
    actions: [start(Unset, Unset, Unset), attemptStart, attemptResult(Completed)]
}

scenario! { nonRetryable
    model: activityProtocol
    starts: Unstarted
    actions: [start(Unset, Unset, Unset), attemptStart, attemptResult(Failed { retryable: false })]
}

// The retryable failure backs the activity off; the backoff timer fires and records nothing; the
// second poll starts the retried attempt, which completes.
scenario! { retriedThenCompleted
    model: activityProtocol
    starts: Unstarted
    actions: [
        start(Unset, Unset, Unset),
        attemptStart,
        attemptResult(Failed { retryable: true }),
        backoff,
        attemptStart,
        attemptResult(Completed),
    ]
}

scenario! { cancelRequestedThenCanceled
    model: activityProtocol
    starts: Unstarted
    actions: [start(Unset, Unset, Unset), attemptStart, control(RequestCancel), attemptResult(Canceled)]
}

// The worker stops, so nothing polls; the caller terminates the scheduled activity.
scenario! { terminatedWhileScheduled
    model: activityProtocol
    starts: Unstarted
    actions: [start(Unset, Unset, Unset), workerStop, control(Terminate)]
}

// Paused before any poll, resumed, then polled and completed.
scenario! { pausedThenCompleted
    model: activityProtocol
    starts: Unstarted
    actions: [
        start(Unset, Unset, Unset),
        control(Pause),
        control(Unpause),
        attemptStart,
        attemptResult(Completed),
    ]
}

// The start request sets the schedule-to-start deadline; the worker stops, so nothing polls; the
// deadline fires.
scenario! { scheduleToStartExpires
    model: activityProtocol
    starts: Unstarted
    actions: [start(Unset, Expires, Unset), workerStop, scheduleToStart]
}

// The start request sets the start-to-close deadline; the worker polls and never answers; the
// deadline fires.
scenario! { startToCloseExpires
    model: activityProtocol
    starts: Unstarted
    actions: [start(Unset, Unset, Expires), attemptStart, startToClose]
}

limits! { three steps: 3 actions: 3 search: 4096 }
limits! { four steps: 4 actions: 4 search: 32768 }
limits! { six steps: 6 actions: 6 search: 262144 }

// authoring: queries

// ### The Queries
//
// Eight find their same-step claim on their path and are realized by the set below. The two
// product claims are verified over every trace of one path each, outside the set.

query! { completion find: completes in: completed limits: three }
query! { nonRetryableFailure find: nonRetryableFails in: nonRetryable limits: three }
query! { retry find: retryCompletes in: retriedThenCompleted limits: six }
query! { cancel find: canceledByWorker in: cancelRequestedThenCanceled limits: four }
query! { terminate find: terminated in: terminatedWhileScheduled limits: three }
query! { pauseResume find: completes in: pausedThenCompleted limits: six }
query! { scheduleToStartTimeout find: scheduleToStartFires in: scheduleToStartExpires limits: three }
query! { startToCloseTimeout find: startToCloseFires in: startToCloseExpires limits: three }
query! { terminalHolds verify: terminalIsFinal in: completed limits: three }
query! { pauseHolds verify: pausedIsNotDispatched in: pausedThenCompleted limits: six }

// authoring: set

// ### The functional set
//
// The Case drives the caller and the worker. No repeat: standalone activities are CHASM only.

set! { standaloneActivityTests
    purpose: functional
    bind: { caller: driven, worker: driven }
    queries: [
        completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
        scheduleToStartTimeout, startToCloseTimeout,
    ]
}

// ### The canary set
//
// The worker is `observed`: the deployment's own worker answers, and the verifier checks the
// machine allows what it did. Both paths record evidence at every step.

set! { standaloneActivityCanary
    purpose: canary
    bind: { caller: driven, worker: observed }
    queries: [completion, cancel]
}

// ### The exploratory set

set! { standaloneActivityExploration
    purpose: exploratory
    bind: { caller: driven, worker: driven }
    machine: activityProtocol
    cover: [rows, results, classMembers]
    budget: four
}

// authoring: composition

// ### The activity and its worker
//
// The protocol machine's worker stop is a stutter row. Composed with the worker of the task queue,
// the stop is the worker's own phase change and every poll is the worker serving, so an attempt
// starts only while the worker polls.

/// The caller's view of the worker: it stops and it serves, and never resumes, so the activity's
/// timers settle every state a stop leaves.
machine! { activityWorker
    from: worker::polling
    restrict: [workerStop, worker::serve]
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub struct StandaloneActivityState {
    pub activity: ProtocolState,
    pub worker: WorkerState,
}

compose! { standaloneActivity
    for: [activity, worker::worker]
    state: StandaloneActivityState
    members: { activity: activityProtocol, worker: activityWorker }
    sync: {
        workerStop: activity.workerStop || worker.workerStop,
        attemptStart: activity.attemptStart || worker.serve,
    }
    starts: { activity: Unstarted, worker: Polling }
    ends: { activity: [Completed, Failed, Canceled, Terminated, TimedOut] }
}

// Every attempt starts with the worker polling: a stopped worker starts nothing.
property! { startedByPollingWorker
    machine: standaloneActivity
    when: attemptStart
    holds: |step| step.state.worker.phase == worker::Phase::Polling
}

// An attempt starts while the worker polls and fails retryably; the worker stops before the retry,
// so the retried attempt never starts and the schedule-to-start deadline fires. The one
// `attemptStart` on the path is what makes the claim fire, so the verify exercises it rather than
// passing vacuously, as the earlier `stoppedBeforeDispatch` path did.
scenario! { stoppedBeforeRetry
    model: standaloneActivity
    starts: { activity: Unstarted }
    actions: [
        activity.start(Unset, Expires, Unset),
        attemptStart,
        activity.attemptResult(Failed { retryable: true }),
        activity.backoff,
        workerStop,
        activity.scheduleToStart,
    ]
}

query! { stoppedWorkerStartsNothing verify: startedByPollingWorker in: stoppedBeforeRetry limits: six }

// authoring: end
