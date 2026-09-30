// authoring: header
//! # The Nexus caller-side Model
//!
//! One workflow-scheduled Nexus operation, as the caller sees it: the product machine says what an
//! operation does, the protocol machine says how the server gets there and refines it, and the
//! functional set runs one Query per side effect that settles the operation, once per value of the
//! implementation switch. Step functions and predicates rather than rows, no cancellation (fn-79)
//! and no concurrency-limit setup parameter.
//!
//! The regions `AUTHORING.md` quotes are marked `// authoring: <name>`; a region runs to the next
//! marker.
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
// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

entity! { workflow }

entity! { operation
    refer: { caller: workflow }
    key: scheduledEvent
}

// authoring: domains

// ### The input domains
//
// A class is one member of a domain, and a constructor that carries finite fields contributes one
// class per assignment of them: `HandlerError { retryable: bool }` is one variant and two classes,
// which is the granularity an example is written at and what mirrors a protobuf oneof.
// `#[derive(Finite)]` walks the payload, so the enum is the domain and nothing is listed twice.

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Timeout {
    Unset,
    Expires,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Reply {
    SyncSuccess,
    Async,
    OperationFailed,
    OperationCanceled,
    HandlerError { retryable: bool },
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Resolution {
    Succeeded,
    Failed,
    Canceled,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Delivery {
    Accepted,
    NotFound,
}

// The input domains are used bare below (`handlerReply(SyncSuccess)`, `complete(Succeeded)`); the
// phase enums are always qualified, so `Resolution::Failed` and `Phase::Failed` do not collide.
use Reply::*;
use Resolution::*;
use Timeout::*;

// authoring: actions

// ### Actions
//
// Parties are names the feature declares by using them: `caller`, `handler`, `network`, `worker`.
// The reserved party `system` is the server. A fault is an ordinary action of a declared party,
// and a timer is `system` behavior the machine owns, so neither is a separate kind.

action! { schedule
    party: caller
    creates: operation
    schema: "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes"
    input: { scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout }
}

action! { handlerReply
    party: handler
    on: operation
    schema: "temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError"
    input: { reply: Reply }
    examples: {
        HandlerError { retryable: false } => "BadRequest",
        HandlerError { retryable: true } => "Internal",
    }
}

/// The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
/// are names the realization interprets.
action! { complete
    party: handler
    on: operation
    input: { resolution: Resolution }
    results: Delivery
}

action! { transportFault
    party: network
    on: operation
}

/// The handler's worker stops polling. An action that names no entity is behavior no entity
/// records: the Run records the fault, but nothing recorded names the operation, so the machines
/// keep their state and record nothing at it. Declared by the worker module; both machines here
/// step on it.
pub use worker::workerStop;

// authoring: observation

// ### The derived observation
//
// A retryable attempt failure writes no history event, so the attempt count is read back through
// `DescribeWorkflowExecution`. Every other evidence name resolves against the realization's
// catalog, which is why only a derived observation is declared.

observation! { pendingAttempts
    on: operation
    read: attempts
}

// authoring: product

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum ProductPhase {
    Scheduled,
    Started,
    Succeeded,
    Failed,
    Canceled,
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
    NexusOperationScheduled,
    NexusOperationStarted,
    NexusOperationCompleted,
    NexusOperationFailed,
    NexusOperationCanceled,
    NexusOperationTimedOut,
}

pub type ProductStep = Step<ProductState, ProductOutcome, ProductFact>;

fn product_step(phase: ProductPhase, recorded: ProductFact) -> Vec<ProductStep> {
    vec![Step { outcome: ProductOutcome::Accepted, state: ProductState { phase }, facts: vec![recorded] }]
}

/// The handler's reply to the server's start request. An operation that has not started yet is the
/// only one a reply can move.
pub fn handler_reply_step(state: &ProductState, reply: Reply) -> Vec<ProductStep> {
    if state.phase != ProductPhase::Scheduled {
        return vec![];
    }
    match reply {
        SyncSuccess => product_step(ProductPhase::Succeeded, ProductFact::NexusOperationCompleted),
        Async => product_step(ProductPhase::Started, ProductFact::NexusOperationStarted),
        OperationFailed => product_step(ProductPhase::Failed, ProductFact::NexusOperationFailed),
        OperationCanceled => product_step(ProductPhase::Canceled, ProductFact::NexusOperationCanceled),
        // A retryable handler error leaves the operation where it is: the product machine does not
        // know about backing off, which is the whole of what the protocol machine adds.
        HandlerError { retryable: true } => vec![],
        HandlerError { retryable: false } => product_step(ProductPhase::Failed, ProductFact::NexusOperationFailed),
    }
}

/// The four phases the product machine ends on.
pub fn product_terminal(state: &ProductState) -> bool {
    matches!(state.phase, ProductPhase::Succeeded | ProductPhase::Failed | ProductPhase::Canceled | ProductPhase::TimedOut)
}

/// An asynchronous completion. A completion that arrives after the operation is over is not found,
/// and changes nothing.
pub fn complete_step(state: &ProductState, resolution: Resolution) -> Vec<ProductStep> {
    if product_terminal(state) {
        return vec![Step { outcome: ProductOutcome::NotFound, state: *state, facts: vec![] }];
    }
    match resolution {
        Succeeded => product_step(ProductPhase::Succeeded, ProductFact::NexusOperationCompleted),
        Failed => product_step(ProductPhase::Failed, ProductFact::NexusOperationFailed),
        Canceled => product_step(ProductPhase::Canceled, ProductFact::NexusOperationCanceled),
    }
}

/// A transport fault is an ordinary action of the network. The product machine cannot see one:
/// whether a delivery was retried is the protocol's account of how, not what.
pub fn transport_fault_step(_state: &ProductState) -> Vec<ProductStep> {
    vec![]
}

/// The handler's worker stopping is a fault the Run records and the operation does not feel. The
/// product machine cannot see it, like the transport fault: a step that kept the state and recorded
/// nothing would be indistinguishable from a stutter, and the refinement would read every stutter
/// as this step.
pub fn worker_stop_step(_state: &ProductState) -> Vec<ProductStep> {
    vec![]
}

/// One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
/// product machine has one timer, and it fires while the operation runs.
pub fn timeout_step(state: &ProductState) -> Vec<ProductStep> {
    match state.phase {
        ProductPhase::Scheduled | ProductPhase::Started => {
            product_step(ProductPhase::TimedOut, ProductFact::NexusOperationTimedOut)
        }
        _ => vec![],
    }
}

machine! { nexusProduct
    for: operation
    state: ProductState
    outcome: ProductOutcome
    facts: ProductFact
    starts: [Scheduled]
    ends: [Succeeded, Failed, Canceled, TimedOut]
    timers: [timeout]
    evidence: {
        NexusOperationStarted: nexusOperationStarted,
        NexusOperationCompleted: nexusOperationCompleted,
        NexusOperationFailed: nexusOperationFailed,
        NexusOperationCanceled: nexusOperationCanceled,
        NexusOperationTimedOut: nexusOperationTimedOut,
    }
    steps: {
        handlerReply: handler_reply_step,
        complete: complete_step,
        transportFault: transport_fault_step,
        workerStop: worker_stop_step,
        timeout: timeout_step,
    }
}

// authoring: protocol

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the three timers the
// schedule command sets, and the attempt count a retryable failure raises. Written against the
// same actions, so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state struct has no "no instance yet" member,
// so `Unscheduled` is that member, and it is what makes the three deadline fields reachable at
// anything but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the `cancel` field and its rows (fn-79), and
// the concurrency-limit rejection. The limit exists, but a step function does not read the setup,
// the key and value differ per switch value, and the rejection names no operation, so it is not
// modeled until a Query needs it.

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Phase {
    Unscheduled,
    Scheduled,
    BackingOff,
    Started,
    Succeeded,
    Failed,
    Canceled,
    TimedOut,
}

/// Which timer fired. The history event records it, so a Contract that did not check it would pass
/// a run that timed out on the wrong deadline.
#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum TimeoutType {
    ScheduleToClose,
    ScheduleToStart,
    StartToClose,
}

/// The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
/// machine's state, so the bound is written here and the saturating successor keeps a retry inside
/// it.
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

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum ProtocolFact {
    NexusOperationScheduled,
    NexusOperationStarted,
    NexusOperationCompleted,
    NexusOperationFailed,
    NexusOperationCanceled,
    NexusOperationTimedOut { timeout_type: TimeoutType },
    PendingAttempts,
}

pub type ProtocolStep = Step<ProtocolState, ProtocolOutcome, ProtocolFact>;

/// The four phases the design ends on. A completion that arrives after one of them is not found.
pub fn terminal_phase(phase: Phase) -> bool {
    matches!(phase, Phase::Succeeded | Phase::Failed | Phase::Canceled | Phase::TimedOut)
}

/// Scheduled and not yet over: the phases a completion resolves and a timer can fire in.
pub fn running(phase: Phase) -> bool {
    matches!(phase, Phase::Scheduled | Phase::BackingOff | Phase::Started)
}

fn moves(state: &ProtocolState, phase: Phase, recorded: Vec<ProtocolFact>) -> Vec<ProtocolStep> {
    vec![Step { outcome: ProtocolOutcome::Accepted, state: ProtocolState { phase, ..*state }, facts: recorded }]
}

/// The caller's schedule command. It names the operation's three deadlines, and every one of them
/// is a state field because whether a timer fires is a question about the operation and not about
/// the command that started it.
pub fn schedule_step(
    state: &ProtocolState,
    schedule_to_close: Timeout,
    schedule_to_start: Timeout,
    start_to_close: Timeout,
) -> Vec<ProtocolStep> {
    if state.phase != Phase::Unscheduled {
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
        facts: vec![ProtocolFact::NexusOperationScheduled],
    }]
}

/// The handler's reply to the server's start request. What the product machine cannot see is the
/// last arm: a retryable failure backs the operation off and raises its attempt count, and the
/// count is read back through the `pendingAttempts` observation because no history event records
/// it.
pub fn protocol_handler_reply_step(state: &ProtocolState, reply: Reply) -> Vec<ProtocolStep> {
    if state.phase != Phase::Scheduled {
        return vec![];
    }
    match reply {
        SyncSuccess => moves(state, Phase::Succeeded, vec![ProtocolFact::NexusOperationCompleted]),
        Async => moves(state, Phase::Started, vec![ProtocolFact::NexusOperationStarted]),
        OperationFailed => moves(state, Phase::Failed, vec![ProtocolFact::NexusOperationFailed]),
        OperationCanceled => moves(state, Phase::Canceled, vec![ProtocolFact::NexusOperationCanceled]),
        HandlerError { retryable: false } => moves(state, Phase::Failed, vec![ProtocolFact::NexusOperationFailed]),
        HandlerError { retryable: true } => vec![Step {
            outcome: ProtocolOutcome::Accepted,
            state: ProtocolState { phase: Phase::BackingOff, attempts: state.attempts.saturating_succ(), ..*state },
            facts: vec![ProtocolFact::PendingAttempts],
        }],
    }
}

/// A transport fault is the same failure arriving as a dropped delivery rather than as a reply.
pub fn protocol_transport_fault_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if state.phase != Phase::Scheduled {
        return vec![];
    }
    vec![Step {
        outcome: ProtocolOutcome::Accepted,
        state: ProtocolState { phase: Phase::BackingOff, attempts: state.attempts.saturating_succ(), ..*state },
        facts: vec![ProtocolFact::PendingAttempts],
    }]
}

/// The handler's worker stopping is a fault the Run records and the operation does not feel, so
/// the step keeps the state and records nothing. On a path it is confirmed by the evidence of the
/// step after it, and the Case says so in a Known Gap.
pub fn protocol_worker_stop_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    vec![Step { outcome: ProtocolOutcome::Accepted, state: *state, facts: vec![] }]
}

/// An asynchronous completion. Before a start, the server records a Started event first, which is
/// why the evidence is two facts and not one -- and why the product machine, which has no
/// `BackingOff` phase to have skipped, could write the completion alone.
pub fn protocol_complete_step(state: &ProtocolState, resolution: Resolution) -> Vec<ProtocolStep> {
    if terminal_phase(state.phase) {
        return vec![Step { outcome: ProtocolOutcome::NotFound, state: *state, facts: vec![] }];
    }
    if state.phase == Phase::Unscheduled {
        return vec![];
    }
    let started_first = || {
        if state.phase == Phase::Started { vec![] } else { vec![ProtocolFact::NexusOperationStarted] }
    };
    let with_start = |fact: ProtocolFact| { let mut facts = started_first(); facts.push(fact); facts };
    match resolution {
        Succeeded => moves(state, Phase::Succeeded, with_start(ProtocolFact::NexusOperationCompleted)),
        Failed => moves(state, Phase::Failed, with_start(ProtocolFact::NexusOperationFailed)),
        Canceled => moves(state, Phase::Canceled, with_start(ProtocolFact::NexusOperationCanceled)),
    }
}

/// The backoff timer. It is what makes `BackingOff` a phase the operation leaves rather than a
/// state it is stuck in, and it records nothing: a retry writes no history event.
pub fn backoff_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if state.phase != Phase::BackingOff {
        return vec![];
    }
    moves(state, Phase::Scheduled, vec![])
}

/// The schedule-to-close deadline covers the whole operation, so it fires in every running phase --
/// and only when the schedule command set it.
pub fn schedule_to_close_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if running(state.phase) && state.schedule_to_close == Expires {
        moves(state, Phase::TimedOut, vec![ProtocolFact::NexusOperationTimedOut { timeout_type: TimeoutType::ScheduleToClose }])
    } else {
        vec![]
    }
}

/// The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
/// start.
pub fn schedule_to_start_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if matches!(state.phase, Phase::Scheduled | Phase::BackingOff) && state.schedule_to_start == Expires {
        moves(state, Phase::TimedOut, vec![ProtocolFact::NexusOperationTimedOut { timeout_type: TimeoutType::ScheduleToStart }])
    } else {
        vec![]
    }
}

/// The start-to-close deadline covers the handler's own work, so it begins at the start.
pub fn start_to_close_step(state: &ProtocolState) -> Vec<ProtocolStep> {
    if state.phase == Phase::Started && state.start_to_close == Expires {
        moves(state, Phase::TimedOut, vec![ProtocolFact::NexusOperationTimedOut { timeout_type: TimeoutType::StartToClose }])
    } else {
        vec![]
    }
}

/// How a protocol state reads as a product state. A phase of the same name is that phase; backing
/// off is still scheduled, because the product machine cannot see a retry; and an operation not yet
/// scheduled reads as scheduled, because the product machine begins there. Every other field is
/// hidden, which is what a map that does not read it says.
pub fn product_of(state: &ProtocolState) -> ProductState {
    let phase = match state.phase {
        Phase::Unscheduled | Phase::Scheduled | Phase::BackingOff => ProductPhase::Scheduled,
        Phase::Started => ProductPhase::Started,
        Phase::Succeeded => ProductPhase::Succeeded,
        Phase::Failed => ProductPhase::Failed,
        Phase::Canceled => ProductPhase::Canceled,
        Phase::TimedOut => ProductPhase::TimedOut,
    };
    ProductState { phase }
}

machine! { nexusProtocol
    for: operation
    state: ProtocolState
    outcome: ProtocolOutcome
    facts: ProtocolFact
    refines: nexusProduct
    map: product_of
    starts: [Unscheduled]
    ends: [Succeeded, Failed, Canceled, TimedOut]
    timers: [backoff, scheduleToClose, scheduleToStart, startToClose]
    unobservable: [backoff]
    evidence: {
        NexusOperationScheduled: nexusOperationScheduled,
        NexusOperationStarted: nexusOperationStarted,
        NexusOperationCompleted: nexusOperationCompleted,
        NexusOperationFailed: nexusOperationFailed,
        NexusOperationCanceled: nexusOperationCanceled,
        NexusOperationTimedOut: nexusOperationTimedOut,
        PendingAttempts: pendingAttempts,
    }
    steps: {
        schedule: schedule_step,
        handlerReply: protocol_handler_reply_step,
        complete: protocol_complete_step,
        transportFault: protocol_transport_fault_step,
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
// realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
// action the Case performs; a transition claim is searched and verified, never realized.
//
// A `holds:` closure captures nothing, so it coerces to the `fn` pointer the framework stores.

// Once an operation is over, no step changes its phase. Declared on the product machine and read on
// the protocol machine through the map.
property! { terminalIsFinal
    machine: nexusProduct
    holds: |before, after| !product_terminal(&before.state) || after.state.phase == before.state.phase
}

// A synchronous reply settles the operation as succeeded, and the completed event records it.
property! { syncSucceeds
    machine: nexusProtocol
    when: handlerReply(SyncSuccess)
    holds: |step| step.state.phase == Phase::Succeeded && step.facts.contains(&ProtocolFact::NexusOperationCompleted)
}

// An asynchronous reply starts the operation, and the started event records it.
property! { asyncStarts
    machine: nexusProtocol
    when: handlerReply(Async)
    holds: |step| step.state.phase == Phase::Started && step.facts.contains(&ProtocolFact::NexusOperationStarted)
}

// A successful completion is recorded by the completed event. Neither the phase nor the outcome is
// fixed: a completion resolves any running phase, and `Accepted` is every earlier step's outcome
// too, so a clause fixing it would be answered before the completion.
property! { completionSucceeds
    machine: nexusProtocol
    when: complete(Succeeded)
    holds: |step| step.facts.contains(&ProtocolFact::NexusOperationCompleted)
}

// A failed completion is recorded by the failed event.
property! { completionFails
    machine: nexusProtocol
    when: complete(Failed)
    holds: |step| step.facts.contains(&ProtocolFact::NexusOperationFailed)
}

// A non-retryable handler error settles the operation as failed, and the failed event records it.
property! { handlerErrorFails
    machine: nexusProtocol
    when: handlerReply(HandlerError { retryable: false })
    holds: |step| step.state.phase == Phase::Failed && step.facts.contains(&ProtocolFact::NexusOperationFailed)
}

/// Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
/// so every field is named.
pub const SUCCEEDED_ON_RETRY: ProtocolState = ProtocolState {
    phase: Phase::Succeeded,
    attempts: match Attempts::new(1) { Some(n) => n, None => unreachable!() },
    schedule_to_close: Unset,
    schedule_to_start: Unset,
    start_to_close: Unset,
};

// A synchronous reply to the retried attempt settles the operation as succeeded on its second
// attempt: the count the retryable failure raised is still one, and the completed event records the
// reply.
property! { retrySucceeds
    machine: nexusProtocol
    when: handlerReply(SyncSuccess)
    holds: |step| step.state == SUCCEEDED_ON_RETRY && step.facts.contains(&ProtocolFact::NexusOperationCompleted)
}

// The schedule-to-start deadline settles an operation no handler started as timed out, and the
// timed-out event records which deadline it was.
property! { scheduleToStartFires
    machine: nexusProtocol
    when: scheduleToStart
    holds: |step| step.state.phase == Phase::TimedOut
        && step.facts.contains(&ProtocolFact::NexusOperationTimedOut { timeout_type: TimeoutType::ScheduleToStart })
}

// The start-to-close deadline settles a started operation no handler completed as timed out.
property! { startToCloseFires
    machine: nexusProtocol
    when: startToClose
    holds: |step| step.state.phase == Phase::TimedOut
        && step.facts.contains(&ProtocolFact::NexusOperationTimedOut { timeout_type: TimeoutType::StartToClose })
}

// authoring: scenarios

// ### The paths the Queries run
//
// A protocol Scenario names its classed actions with their inputs and its start by its phase. Each
// path below is one upstream functional test's shape: the schedule command with no deadline set,
// then the side effects that settle the operation.

scenario! { syncReplied
    model: nexusProtocol
    starts: Unscheduled
    actions: [schedule(Unset, Unset, Unset), handlerReply(SyncSuccess)]
}

scenario! { asyncThenSucceeded
    model: nexusProtocol
    starts: Unscheduled
    actions: [schedule(Unset, Unset, Unset), handlerReply(Async), complete(Succeeded)]
}

scenario! { asyncThenFailed
    model: nexusProtocol
    starts: Unscheduled
    actions: [schedule(Unset, Unset, Unset), handlerReply(Async), complete(Failed)]
}

scenario! { nonRetryableError
    model: nexusProtocol
    starts: Unscheduled
    actions: [schedule(Unset, Unset, Unset), handlerReply(HandlerError { retryable: false })]
}

// The retryable error backs the operation off; the backoff timer fires and records nothing; the
// retried attempt is answered synchronously.
scenario! { retriedThenSucceeded
    model: nexusProtocol
    starts: Unscheduled
    actions: [
        schedule(Unset, Unset, Unset),
        handlerReply(HandlerError { retryable: true }),
        backoff,
        handlerReply(SyncSuccess),
    ]
}

// The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
// answers the start request; the deadline fires. The worker stops after the schedule in the
// operation's order, where the stop changes nothing; the realization stops it before the workflow
// starts, where the stop cannot race the dispatch.
scenario! { scheduleToStartExpires
    model: nexusProtocol
    starts: Unscheduled
    actions: [schedule(Unset, Expires, Unset), workerStop, scheduleToStart]
}

// The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
// never completes; the deadline fires.
scenario! { startToCloseExpires
    model: nexusProtocol
    starts: Unscheduled
    actions: [schedule(Unset, Unset, Expires), handlerReply(Async), startToClose]
}

// Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
// sequence of two is found among ninety-nine candidates, one of three among about a thousand and
// one of four among about ten thousand.
limits! { two steps: 2 actions: 2 search: 512 }
limits! { three steps: 3 actions: 3 search: 4096 }
limits! { four steps: 4 actions: 4 search: 32768 }

// authoring: queries

// ### The Queries
//
// The design's seven: sync success, async reply then succeeded callback, async reply then failed
// callback, non-retryable handler error, retryable handler error then sync success after one
// backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
// after an asynchronous reply. Each finds its same-step claim on its path and is realized by the
// set below. The product claim is verified over every trace of one path, outside the set, because
// a `verify` Query realizes nothing.

query! { syncCompletion find: syncSucceeds in: syncReplied limits: two }
query! { asyncCompletion find: completionSucceeds in: asyncThenSucceeded limits: three }
query! { asyncFailure find: completionFails in: asyncThenFailed limits: three }
query! { handlerError find: handlerErrorFails in: nonRetryableError limits: two }
query! { retry find: retrySucceeds in: retriedThenSucceeded limits: four }
query! { scheduleToStartTimeout find: scheduleToStartFires in: scheduleToStartExpires limits: three }
query! { startToCloseTimeout find: startToCloseFires in: startToCloseExpires limits: three }
query! { terminalHolds verify: terminalIsFinal in: asyncThenSucceeded limits: three }

// authoring: set

// ### The functional set
//
// Every party but `system` is bound: the Case drives the caller, the handler and the worker, and
// observes the network. The set repeats over the implementation switch, so each Query's Case runs
// once under HSM and once under CHASM.

set! { nexusCallerTests
    purpose: functional
    bind: { caller: driven, handler: driven, network: observed, worker: driven }
    repeat: implementation
    queries: [
        syncCompletion, asyncCompletion, asyncFailure, handlerError, retry,
        scheduleToStartTimeout, startToCloseTimeout,
    ]
}

// ### The canary set
//
// A canary runs a Query against a deployment that performs the handler's part itself: the handler
// is `observed`, so the verifier reads which reply occurred and checks the machine allows it. What
// admits a canary is that a deployment can close every gap its Case carries, and every step of the
// sync and async completion paths records evidence; a path with a silent step -- the backoff, the
// worker stop -- is a capability gap no deployment closes, so a canary naming it is rejected.

set! { nexusCallerCanary
    purpose: canary
    bind: { caller: driven, handler: observed, network: observed, worker: driven }
    queries: [syncCompletion, asyncCompletion]
}

// ### The exploratory set
//
// An exploration covers the protocol machine rather than listing Queries. Its targets are the rows
// an exploration within the budget's steps of a start can take, the results those rows reach and
// the members of the classes their actions claim, each in the machine's catalog order and cut at
// the budget's search count, so the enumeration is the same on every reading.

set! { nexusCallerExploration
    purpose: exploratory
    bind: { caller: driven, handler: driven, network: observed, worker: driven }
    machine: nexusProtocol
    cover: [rows, results, classMembers]
    budget: four
}

// authoring: composition

// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No set
// names the composition; it is what the cross-entity claim is verified over.

/// The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
/// action no `sync:` line names would stay executable on its own and admit a stop, a resume and
/// then a reply; the operation's timers settle every state a stop leaves.
machine! { handlerWorker
    from: worker::polling
    restrict: [workerStop, worker::serve]
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub struct NexusCallerState {
    pub operation: ProtocolState,
    pub worker: WorkerState,
}

compose! { nexusCaller
    for: [operation, worker::worker]
    state: NexusCallerState
    members: { operation: nexusProtocol, worker: handlerWorker }
    sync: {
        workerStop: operation.workerStop || worker.workerStop,
        handlerReply: operation.handlerReply || worker.serve,
    }
    starts: { operation: Unscheduled, worker: Polling }
    ends: { operation: [Succeeded, Failed, Canceled, TimedOut] }
}

// Every reply, of any class, leaves the handler's worker polling: no handler replies while its
// worker is stopped.
property! { repliedByPollingWorker
    machine: nexusCaller
    when: handlerReply
    holds: |step| step.state.worker.phase == worker::Phase::Polling
}

// A retryable reply backs the operation off; the handler's worker then stops, so the retried
// attempt is never answered and the schedule-to-start deadline fires.
scenario! { repliedThenStopped
    model: nexusCaller
    starts: { operation: Unscheduled }
    actions: [
        operation.schedule(Unset, Expires, Unset),
        handlerReply(HandlerError { retryable: true }),
        workerStop,
        operation.scheduleToStart,
    ]
}

query! { stoppedWorkerRepliesNothing verify: repliedByPollingWorker in: repliedThenStopped limits: four }

// authoring: end
