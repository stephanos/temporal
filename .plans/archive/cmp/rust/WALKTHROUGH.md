# Walkthrough: the standalone activity Model in Rust

## 1. What this is

Umpire is model-based testing with three parts. A Model is the rulebook: a finite description of
what a Temporal feature may do. A playthrough is one path through the rulebook, chosen by a search.
A referee runs that path against a real server and checks that what the server recorded is what
the rulebook allowed.

A standalone activity is a Temporal activity started directly with `StartActivityExecution`, with
no workflow around it. It writes no history events, so the only way to observe it is to read its
status through `DescribeActivityExecution` or its result through `PollActivityExecution`.

`standalone_activity.rs` is the rulebook for one such activity: the vocabulary, two machines at
two levels of detail, the claims they make, the paths a search should try, and the sets of tests
that would run them. `pins.rs` freezes what the rulebook says.

## 2. The language in five minutes

Rust is a compiled language with a strong static type system. The walkthrough leans on these
features.

**Enums with payloads.** An `enum` is a closed set of variants, and a variant may carry fields.
`Failed { retryable: bool }` is one variant with one field.

```rust
pub enum AttemptResult {
    Completed,
    Failed { retryable: bool },
    Canceled,
}
```

**Exhaustive `match`.** A `match` must cover every possible value or the program does not compile.
`A | B` in a pattern means either; `_` matches anything; `matches!(x, A | B)` returns `true` when
`x` fits the pattern.

**Derives.** `#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]` above a type asks the
compiler, or a plugin called a derive macro, to write standard trait implementations. `Copy` means
values duplicate on assignment; `Finite` is Umpire's own derive, explained in section 4.

**Struct update syntax.** `ProtocolState { phase, ..*state }` builds a struct with `phase` set and
every other field copied from `state`. The `*` reads the value behind a reference `&ProtocolState`.
The helper `moves` at `standalone_activity.rs:377-379` uses it to build every "move to this phase
and record these facts" step.

**`Vec` and `vec![]`.** `Vec<T>` is a growable list; `vec![]` is the empty one.

**Macros.** `name! { ... }` invokes a macro, for example `query! { completion find: completes in:
completed limits: three }`. A procedural macro is a compiler plugin that receives the tokens
between the braces and returns Rust code. Every declarative block in the Model is one.

**Closures and statics.** `|step| step.state.phase == Phase::Completed` is an anonymous function
of one argument. A `static` is a global value; `LazyLock` computes it on first read, and the macros
expand each machine, property, scenario and query to one.

## 3. Vocabulary: entities, parties, actions, inputs

An **entity** is what a machine is about. The activity is named by the id its caller chose.

`standalone_activity.rs:28`
```rust
entity! { activity key: activityId }
```

A **party** is who performs an action. This Model uses `caller` and `worker`; `system` is reserved
for the timers a machine owns. An **action** is a named side effect of a party with typed finite
inputs. `start` (caller, creates the activity, three `Timeout` inputs), `attemptStart` (worker, no
input) and `control` (caller, one `Control` input) are declared the same way as this one:

`standalone_activity.rs:92-101`
```rust
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
```

`action! { attemptResult .. }` expands to a unit struct named `attemptResult` implementing the
framework's `Action` trait, plus `attemptResult::class(result)`, which builds one **action
class**, `attemptResult::any()`, which matches every class, and `attemptResult::bind(f)`, which
turns a step function into one bound step per class. Lowercase type names are unusual in Rust; the
Diesel database library does the same for its column types.

An action class is one action with one assignment of its inputs. `AttemptResult` has four values:
`Completed`, `Failed { retryable: false }`, `Failed { retryable: true }`, `Canceled`, so
`attemptResult` contributes four classes. `Control` has four variants, so `control` contributes four.
`start` has three two-valued inputs, eight classes. `attemptStart` has no input, one class.

`AttemptResult` (section 2, declared at `standalone_activity.rs:43-48`) is how Rust represents a
finite input domain with a payload: the payload is a field of the variant, and `#[derive(Finite)]`
enumerates the field's values to list every class. The Lean writes `failed(retryable : Bool)`.

A fault is just an action. The worker stopping is an action of the `worker` party that names no
entity, so the machines keep their state and record nothing at it. It is declared once in
`worker.rs` and re-exported at `standalone_activity.rs:113` with `pub use worker::workerStop;`.

## 4. State

There are two state types. `ProductState` (`standalone_activity.rs:149-152`) is what the caller
can read through Describe: one field, a phase with nine values (`Scheduled`, `Started`, `Paused`,
`CancelRequested`, `Completed`, `Failed`, `Canceled`, `Terminated`, `TimedOut`). `ProtocolState` is how the server gets there: a twelve-value phase that adds `Unstarted`,
`BackingOff` and `PauseRequested`, an attempt counter, and the three deadlines the start request
set.

`standalone_activity.rs:329-339`
```rust
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
```

Every field is finite because the framework builds a table by listing every state and running
every step on it. An unbounded integer would make that impossible, so the attempt count is a
`Bounded<2>`: an integer in `0..=2` whose `saturating_succ` (`umpire/lib.rs:119-122`) returns the
bound itself once it is reached. `Bounded<const MAX: u8>` uses a const generic, a type parameter
that is a number.

The enumeration mechanism is the `Finite` trait. A trait is an interface; this one has an
associated constant, the number of values, and a function listing them.

`umpire/lib.rs:49-53`
```rust
pub trait Finite: Sized + Clone + Eq + Hash + Debug + Send + Sync + 'static {
    const CARDINALITY: usize;

    /// Every member, in canonical order. `all().len() == CARDINALITY`.
    fn all() -> Vec<Self>;
```

`#[derive(Finite)]` writes the implementation. For an enum, the cardinality is the sum over
variants of the product of the fields' cardinalities. For a struct, it is the product of the
fields. `ProductState` has 9 values. `ProtocolState` has 12 phases times 3 attempt counts times 2
times 2 times 2 deadlines, 288 values. In this sketch the derive emits the `CARDINALITY` formula
but leaves the struct's `all()` body as `todo!()`; a full version takes the cartesian product of
each field's `all()`. Because `CARDINALITY` is a constant, the compiler can check it before any
test runs.

## 5. Step functions

A step function has the shape `(state, inputs) -> list of steps`, and a step is an outcome, a next
state, and the facts the step records.

`umpire/lib.rs:243-247`
```rust
pub struct Step<S, O, F> {
    pub outcome: O,
    pub state: S,
    pub facts: Vec<F>,
}
```

An empty list means the action is not enabled in that state. Two entries would mean the action is
nondeterministic. `ProductStep` is a type alias so the signature stays short.

The product's `attempt_result_step`, arm by arm:

`standalone_activity.rs:199-223`
```rust
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
```

The `match` is on a pair, phase and result. Arm one: from started or cancel-requested, a completed
response settles the activity as completed and Describe reads COMPLETED. Arm two: a non-retryable
failure settles it as failed. Arm three: a retryable failure from started reads as SCHEDULED again;
Describe shows the retry, unlike the Nexus Model where a retry is invisible. Arm four: under a
requested cancel, both a retryable failure and a cancel response resolve the activity as canceled.
Arm five: a cancel response with no cancel requested is not enabled. The last arm lists every other
phase and disables everything; the `_` on the right covers any result.

The protocol version dispatches on phase as well as result because three source phases react
differently to the same retryable failure.

`standalone_activity.rs:429-459`
```rust
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
```

From started, a retryable failure backs the activity off and records only the attempt count, the
CHASM `TransitionRescheduled`. From cancel-requested, it resolves the activity as canceled. From
pause-requested, it parks the activity as paused, the CHASM
`TransitionAttemptFailedWhilePauseRequested`. Same class, three destinations, so the phase must be
part of the pattern.

Exhaustiveness is enforced by the compiler. Delete the `(Phase::Started, Canceled)` arm and the
build fails with `error[E0004]: non-exhaustive patterns`, naming the missing pattern.

## 6. The machine and its table

The `machine!` block wires the pieces together.

`standalone_activity.rs:552-586`
```rust
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
```

`starts` names the one start state by its phase; the other fields take their first value, so the
machine begins unstarted with zero attempts and no deadline set. `ends` names the five phases a run
may stop in; every state with one of those phases is an end. `timers` declares four system actions
this machine owns; the macro generates an action type for each, so `backoff` can be named in a
scenario like any other action. `unobservable` says the backoff timer records nothing a verifier
could read. `outcome:` and `facts:` name types the Lean infers; a Rust `static` must have its type
spelled out.

`evidence` maps each fact to the name of the recorded thing that confirms it. For a workflow that
would be a history event type. A standalone activity writes no history, so every name here is a
status value read from `DescribeActivityExecution`, plus the derived `attemptCount` observation for
the one fact no status change records. A misspelled fact is a compile error on that token.

Each `steps:` line becomes `attemptResult::bind(protocol_attempt_result_step)`, emitted at the
author's token so that an undeclared action or a mismatched signature is a compiler error under
that line. `bind` calls the step function once per action class, four bound steps for
`attemptResult`. `Machine::table` (`umpire/lib.rs:300-317`) then builds the finite table: it
takes `S::all()`, the 288 protocol states, runs every bound step on every state, and keeps one row
per returned step with the source state, the action class and the step.

This runs at test time, on first use, and is cached. Rust cannot run it at compile time: `const`
evaluation has no heap and cannot call boxed closures. The Lean builds the same table during
elaboration.

## 7. Two levels and the refinement

The product machine says what an activity does, as the caller observes it. The protocol machine
says how the server gets there: the backoff between a failure and the next poll, the pause a
started attempt can only request, which of three deadlines fired. Properties are easiest to state
on the product; the protocol is where the interesting paths live. The refinement is what lets a
product Property be trusted on protocol paths.

`product_of` is the abstraction function from protocol state to product state.

`standalone_activity.rs:537-550`
```rust
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
```

Backing off reads as scheduled because the caller sees SCHEDULED. A requested pause reads as
started, not paused: the worker still holds the attempt, so every answer it can give is a product
row from started, and the pause request itself becomes a stutter. The spec's first draft mapped it
to paused and the check rejected three rows; the revision note in `SPEC.md` records the fix.

The rule: for every protocol row from `s` to `s'`, either `product_of(s) == product_of(s')` (a
stutter) or the product table has some row from `product_of(s)` to `product_of(s')` under any
action class. The check is implemented in the framework and runs at test time.

`umpire/lib.rs:402-406`
```rust
        for row in rows {
            let (before, after) = ((self.map)(&row.from), (self.map)(&row.to));
            let mapped = match product.get(&(before.key(), after.key())) {
                _ if before == after => None,
                Some(step) => Some(step.clone()),
```

`product` is a map from every product row's (from, to) pair of state keys to the first product
class in table order between them, built just above these lines. A protocol row whose mapped ends
are equal is a stutter, `None`. One with a product pair is that class. Anything else is recorded
as the rejection, naming the protocol row.

One row by hand. Take the protocol state started, attempts 1, no deadlines set, and the class
`attemptResult(Failed { retryable: true })`. The step function's third arm moves it to backing-off
with attempts still 1 and the fact `AttemptCount`. Map both ends: started reads as `Started`,
backing-off reads as `Scheduled`. They differ, so it is not a stutter. Does the product have a row
from started to scheduled? Yes: `attempt_result_step`'s third arm, `(ProductPhase::Started, Failed
{ retryable: true })`, goes to scheduled. The row is accepted and recorded as that product class.
The pin at `pins.rs:216-219` freezes exactly this.

The real Lean checker is stricter than the spec's stated rule: the matching product row must also
have the same outcome, and its facts must appear among the protocol row's facts by evidence name.
Under that rule this row would need the protocol step to record `StatusScheduled` as well as
`AttemptCount`. The Rust check implements the mapped-states rule, so the sample is unaffected, but
a reader porting it to the Lean rule should add that fact.

## 8. Properties

A same-step claim names the action it is about under `when:` and holds of the step that action
produces.

`standalone_activity.rs:604-608`
```rust
property! { completes
    machine: activityProtocol
    when: attemptResult(Completed)
    holds: |step| step.state.phase == Phase::Completed && step.facts.contains(&ProtocolFact::StatusCompleted)
}
```

`retryCompletes` fixes the whole state, so it can only hold after exactly two attempts. The state
is a `const`, a compile-time constant, with every field named.

`standalone_activity.rs:619-633`
```rust
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
```

A transition claim has no `when:` and takes two steps, the one before and the one after. Both of
this Model's transition claims are on the product machine and are read on the protocol through the
map.

`standalone_activity.rs:658-661`
```rust
property! { pausedIsNotDispatched
    machine: activityProduct
    holds: |before, after| before.state.phase != ProductPhase::Paused || after.state.phase != ProductPhase::Started
}
```

`terminalIsFinal` (`standalone_activity.rs:598-601`) has the same shape: once `product_terminal`
holds of the state before, the phase after must be unchanged. The macro checks the closure's arity
against the claim kind: a `when:` with a two-argument closure, or no `when:` with a one-argument
closure, is a compile error on the closure.

## 9. Scenarios and limits

A scenario is a path: a start state by its phase and a list of classed actions in order. A classed
action is spelled as a call, `start(Unset, Unset, Unset)`, or a bare name when it has no input.
The variant names are bare because the file glob-imports the input enums with `use
AttemptResult::*;`. The macro rewrites `start(Unset, Unset, Unset)` to `start::class(Unset, Unset,
Unset)`, so a misspelled action or input is a compile error.

`standalone_activity.rs:701-712`
```rust
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
```

`standalone_activity.rs:728-738`
```rust
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
```

Limits bound the search: how many steps a path may take, how many actions, and how many candidates
to examine. `limits! { six steps: 6 actions: 6 search: 262144 }` at `standalone_activity.rs:758`
is the widest of the three this Model declares.

## 10. Queries

A query joins a property, a scenario and limits. `find` searches the scenario for a path on which
the same-step claim holds and stops at the first; a set can later turn it into a test. `verify`
checks the claim on every trace of the scenario within the limits and is never turned into a test.

`standalone_activity.rs:769` and `776`
```rust
query! { retry find: retryCompletes in: retriedThenCompleted limits: six }
query! { pauseHolds verify: pausedIsNotDispatched in: pausedThenCompleted limits: six }
```

`pauseHolds` reads a product claim on a protocol scenario. The framework admits that when the query
runs: the property's state type differs from the scenario's, so it looks for a `refines:` whose
target has that type and reads the claim through `product_of`. In the Lean a bad pairing is an
error at the block; here it is the `Err` of `Query::run`.

The search is sketched: `Query::run` (`umpire/lib.rs:596-605`) admits the property, rejects a
`find` of a transition claim, then hits `todo!()`. A full version would walk the table from the
scenario's start, following its actions in order and interleaving other enabled actions up to the
limits, checking the claim at the triggering step for `find` and at every step for `verify`, under
`cargo test`.

When a query fails, the author sees a failed test: the pin compares the outcome name, so the
output is a two-line diff with `left: Ok("not-found")` against `right: Ok("found")`.

## 11. Sets

A set groups queries into tests and says which parties the test drives and which it only observes.
The functional set drives both parties. There is no `repeat: implementation` line: standalone
activities exist only in CHASM, so there is no HSM run to repeat under.

`standalone_activity.rs:784-791`
```rust
set! { standaloneActivityTests
    purpose: functional
    bind: { caller: driven, worker: driven }
    queries: [
        completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
        scheduleToStartTimeout, startToCloseTimeout,
    ]
}
```

The canary set (`standalone_activity.rs:798-802`) runs `completion` and `cancel` against a
deployment whose own worker answers, so its `bind:` line reads `worker: observed`: the test reads
what the worker did and checks the machine allows it. The exploratory set
(`standalone_activity.rs:806-812`) names no queries; it has `machine: activityProtocol`,
`cover: [rows, results, classMembers]` and `budget: four`, and covers the rows within the budget's
steps of a start, the results those rows reach, and the members of the input classes.

The macro checks the shape: a functional set with a `machine:` line, or an exploratory set with
`queries:`, is a compile error on the offending keyword.

## 12. Composition with the worker

The protocol machine's `workerStop` is a stutter: the activity cannot see its worker. To state a
claim about the worker, the Model composes the activity with the `polling` machine from
`worker.rs:93-106`: a `WorkerState` with a two-value phase, `Polling` or `Stopped`, and three
actions. `workerStop` moves polling to stopped, `workerResume` moves it back, and `serve` keeps a
polling worker polling and is not enabled for a stopped one. Both phases are ends.

The composition keeps only the worker's stop and serve. Without a resume, a stop is final, and the
activity's timers settle whatever state a stop leaves.

`standalone_activity.rs:824-845`
```rust
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
```

Each `sync:` line makes two member actions fire as one. `workerStop` is the activity's stutter and
the worker's phase change at once. `attemptStart` is the activity's poll and the worker serving,
so the composite has an `attemptStart` row only where the worker has a `serve` row, and a stopped
worker has none.

`standalone_activity.rs:848-852` and `858-871`
```rust
property! { startedByPollingWorker
    machine: standaloneActivity
    when: attemptStart
    holds: |step| step.state.worker.phase == worker::Phase::Polling
}

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
```

The claim: every attempt starts with the worker polling. The path starts one attempt while the
worker polls, fails it retryably, backs off, stops the worker, and lets the schedule-to-start
deadline settle the activity. Verified over the scenario's traces, it proves that once the worker
stops, no poll can happen. The one `attemptStart` on the path is what makes the claim fire, so the
verify exercises it rather than passing vacuously, which the spec's earlier `stoppedBeforeDispatch`
path did. `compose!` and `Compose::machine` are sketched in this sample.

## 13. Pins

Pins freeze what the Model says. The first tier is decided by the compiler when the test file is
built. `const _: () = assert!(..)` is an unnamed constant whose value is an assertion; if it fails,
compilation fails. These guard the state-space size and the class count against a phase or variant
nobody meant to add.

`pins.rs:39` and `43`
```rust
const _: () = assert!(activity::ProtocolState::CARDINALITY == 12 * (activity::ATTEMPT_BOUND as usize + 1) * 2 * 2 * 2);
const _: () = assert!(activity::AttemptResult::CARDINALITY == 4);
```

The second tier is ordinary tests. `cancel_without_request_is_not_enabled` (`pins.rs:202-205`)
calls the protocol step function directly and asserts the empty list. This one guards the
refinement and freezes the row walked through in section 7 and the
pause-request stutter.

`pins.rs:214-220`
```rust
        assert_eq!(refinement.rejected, None);
        assert_eq!(refinement.rows.len(), activityProtocol.table().transitions.len());
        assert_eq!(
            refinement.lookup("started-1-unset-unset-unset-attemptResult-failed-true"),
            Some(Some("attemptResult-failed-true"))
        );
        assert_eq!(refinement.lookup("started-1-unset-unset-unset-control-pause"), Some(None));
```

## 14. From model to running test

None of what follows is in this sample. In the Lean system, a `case` block lowers each functional
query into a Case: a protobuf program that says which party performs which class in which order,
which evidence to read, and a Contract stating the claim's clause. A realization tells the lowering
how each class is performed against a real server. No realization exists yet for standalone
activities; the Nexus Model has one, and a standalone activity one would need to issue
`StartActivityExecution`, drive a worker, and read Describe.

The Go Testpilot runtime executes a Case against a server, collects the evidence, and returns a
verdict: held, violated with the recorded evidence, or a Known Gap left a step unconfirmed. The
Rust sample stops at the checked Model and its pins.

## 15. Gaps and gradual growth

The Model says "not modeled" in several places, and each is a seam for growth. Reset is not modeled; the file header says it is deferred like cancellation in the Nexus Model.
Heartbeat timeout is not modeled. Adding either means one new `action!` or timer, one arm per phase
in the step functions the compiler will insist on, and a `product_of` case if a new phase appears.

Stutters are how the Model stays honest about what it cannot see. `protocol_worker_stop_step`
keeps the state and records nothing, so a path through it is confirmed by the step after it. The
product's `worker_stop_step` returns the empty list instead: a product stutter would be
indistinguishable from every other stutter under the refinement.

Empty steps are the third tool. Every `=> vec![]` arm states that the action is not enabled there.
To model a new behavior, an author replaces an empty arm with a real step, and the refinement check
says whether the product needs a matching row.

## 16. Mental model recap

- A Model is a finite rulebook. Every state type and input type is `Finite`, so the framework can
  list every state and every action class and build a complete table.
- A step function maps a state and inputs to a list of steps. Empty means not enabled. The `match`
  must be exhaustive, so adding a variant forces every step function to say what it does with it.
- Two machines describe one activity. The product is what Describe shows. The protocol is how the
  server gets there. `product_of` maps protocol states to product states.
- The refinement check walks every protocol row through the map and demands a stutter or a product
  row between the mapped states. It is a test, not a compile error.
- A same-step property is about the step one action produces. A transition property is about any
  two consecutive steps. Product properties are read on protocol scenarios through the map.
- A scenario is a path of classed actions. A query searches it for a property, either to find a
  witness or to verify every trace.
- Every name in a declarative block is a Rust path, so a typo is a compile error under that token.

## 17. Where this implementation is weak

- **Nothing here has been compiled or run.** The framework's bounded search, reachability,
  `restrict`, `Compose::machine`, the struct and tuple `Finite::all()` bodies, and the `scenario!`,
  `compose!`, `observation!` and `limits!` parsers are `todo!()`. Every `#[test]` in `pins.rs` would
  panic at the first `todo!()` today. The README estimates six hundred more lines of parser.
- **Names are Rust-cased, not spec-cased.** Variants are `Completed` and `Unset`, fields are
  `schedule_to_close`. The spec's spelling comes back only at run time through `Finite::key()`. A
  reader comparing with the Lean or the spec side by side must translate.
- **A product claim over facts cannot be read on the protocol.** `Refines::map_erased` maps state
  only and `PropertyOn::holds_step` is `todo!()`. Both product properties here are state-only
  transition claims, so nothing triggers it, but a product same-step claim about a fact would.
- **Compile-time claims need care.** The `const _: () = assert!(..)` pins live in a test file, so
  they run under `cargo check --tests` or `cargo test`, not a plain library `cargo check`. The
  `trybuild` fixtures freeze rustc's own wording and need refreshing when the toolchain changes it.
  The review also found a nonexistent `PhaseField` trait in the derive and a wrong error code in the
  README; both are fixed in the current files.
