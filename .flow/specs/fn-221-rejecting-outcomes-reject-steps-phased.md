## Conversation Evidence

> user (2026-10-19): on `Client::Cancel => {}` in the mini activity Model: "should we actually reject this somehow as an invalid request? and match that in the realization to an actual error type". Then "outline the desired design", "should we also have a trait like `Phased` that abstracts some of this away for machines that have phases (most of them likely)?", "is there a design that doesn't require such an elaborate type annotation?" (on `-> Result<(), Rejection>`), and "yes write flow spec".

The owner wants a legal request that the system answers with an error to be a first-class step of the Model, distinguished from a disabled action, judged against the real error in the realization, written as a statement in the handler, and shared through a phase abstraction with the two closure laws the archive proved.

## Goal & Context

The attribute Model cannot say "this request is answered with an error". `#[machine]` fixes `Outcome = ()` and emits an edge only when a handler changes the state, so a client cancelling a closed activity collapses into the same thing as a worker polling an already started one: no edge. Those are different facts. The real server answers `RequestCancelActivityExecution` on a closed standalone activity with NotFound ("no running activity execution: it closed with status X", `chasm/lib/activity/activity.go:105`), and the archived mini activity product modeled exactly that (`when[Closed] ~> rejects(Rejection.notFound)` in `Completion.scala`, with `Closable.closedIsRejectedUniformly`). fn-216 scoped DSL expansion out and recorded the gap; this spec closes it as the first DSL expansion driven by Temporal evidence.

The author writes `reject!(Rejection::NotFound)` as a statement inside an ordinary unit-returning handler, the way `check!` is written inside a claim. The Model gains a phase abstraction with two shared closure laws. The realization projects the client's RPC result, success or gRPC status, to the Model's outcome and judges it by replay. One live scenario proves it against the real server.

## Architecture & Data Models

**Rejections and outcomes (`model/umpire/dsl`).** `Rejection` is a generic enum after the canonical status codes a system can answer a request with: NotFound, AlreadyExists, FailedPrecondition, InvalidArgument, PermissionDenied, Unavailable. `Outcome` is `Accepted` or `Rejected(Rejection)`. Both derive `Clone, Copy, PartialEq, Eq, Hash` and implement `Spell` (`Accepted`, `Rejected(NotFound)`). No reason string.

**`reject!` mirrors `check!`.** `umpire_dsl::outcome` keeps a thread-local frame like `claim`. `reject!(r)` records the first rejection in the active frame and then `return`s from the enclosing function. The generated `step` runs each `#[on]` handler inside `outcome::evaluate(|| next.handler(value))`, which opens the frame, runs the handler and yields the outcome. Edge emission:

| Handler | State | Edge |
| --- | --- | --- |
| no rejection | unchanged | none; the action is disabled |
| no rejection | changed | edge with `Accepted` |
| rejection | unchanged | self-loop edge with `Rejected(r)` |
| rejection | changed | Model error; the checker refuses the run naming the step |

`#[machine]` sets `type Outcome = ::umpire_dsl::Outcome` only when the module body contains a `reject!` invocation; otherwise it stays `()`, so Kettle and every existing fixture keep their goldens. A `reject!` in a helper records the rejection but only leaves the helper; the unchanged-state rule catches a helper that rejects and then mutates. A `reject!` outside an active frame has no effect, as `check!` outside an evaluation has none.

**Claims see the outcome.** A `#[postcondition]` may take a third parameter `outcome: &Outcome`; the macro passes the emitted edge's outcome, which `StepClaim` already carries.

**Phases (`model/umpire/dsl`).**

```rust
pub trait Phase: Copy + Eq + Hash + Spell { fn closed(&self) -> bool; }
pub trait Phased { type Phase: Phase; fn phase(&self) -> Self::Phase; fn closed(&self) -> bool { self.phase().closed() } }
```

A Model implements both by hand inside the module; no attribute. `umpire_dsl::closable` holds two functions over `S: Phased + Eq`, each registered by a one-line delegating `#[postcondition]`: `terminal_remains(after, before)` (no step leaves a closed phase or moves between closed phases) and `closed_rejects(after, before, outcome, expected: Rejection)` (every step from a closed phase keeps the state and answers `Rejected(expected)`). The archive's caveats carry over as doc comments: neither law promises anything about repeated request IDs, and the rejection is entity-specific.

**Mini activity Model.** `Phase::closed` names the four terminal variants once; the test file's `TERMINALS` array reads it. `Client::Cancel if self.closed() => reject!(Rejection::NotFound)`. `terminal_remains` delegates to `closable`; `closed_rejects` is added. Cancel in Scheduled or Started still enters Canceled atomically; the six-phase restriction is unchanged.

**Core protocol (`umpire.model.v1`).** `Explanation` gains `repeated string outcomes = 2`, parallel to `steps`, spelled from each replayed edge. `Realize::matches` takes the replayed outcome: `fn matches(&self, observation: &Any, state: &M::State, outcome: Option<&M::Outcome>)`, `None` for the zero-step current check. The runtime's replay stops discarding the outcome it computes.

**Temporal envelope (`umpire.temporal.v1`).** `Observation` becomes `{ string method = 1; oneof result { google.protobuf.Any response = 2; google.rpc.Status error = 3; } }`. `response` keeps tag 2. If `google.rpc.Status` is not in the generated descriptor closure, a two-field `umpire.temporal.v1.Error { int32 code = 1; string message = 2; }` takes its place; the task records which.

**Realization (`features/activity_mini/temporal.rs`).** A second recognized method, `/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution`: an empty `RequestCancelActivityExecutionResponse` projects to candidates `[[Client::Cancel]]` and matches only an `Accepted` replay; a status with code NotFound projects to the same candidate and matches only `Rejected(NotFound)`; any other code is Unexplained. The code table is one `match` in `temporal.rs`: NotFound 5, AlreadyExists 6, FailedPrecondition 9, InvalidArgument 3, PermissionDenied 7, Unavailable 14. Describe projection is untouched.

## API Contracts

- `umpire_dsl::{Rejection, Outcome, Phase, Phased, closable, reject!}` as above. `reject!` takes one expression of type `Rejection`.
- `Explanation { repeated string steps = 1; repeated string outcomes = 2; }`; `outcomes.len() == steps.len()`; a unit outcome spells `()`.
- `Realize::matches(&self, &Any, &M::State, Option<&M::Outcome>) -> Result<bool, Refusal>`.
- `Observation` as above. Judge of an error observation whose method is unknown is Unsupported; a status whose code is outside the table is Unexplained, not a refusal, because the request was recognized.
- Rows spell the outcome of a rejecting self-loop as `outcome=Rejected(NotFound)`; the report counts it as an edge.

## Edge Cases & Constraints

- A handler that rejects and changes the state is a Model error: `Checker::explore` returns a refusal naming the action and the before state; the guest surfaces it as the existing checker refusal.
- `reject!` in a claim, helper outside a step, or test has no effect; a debug assertion on the frame makes it loud under `cfg(debug_assertions)`.
- Existing pinned wire bytes stay identical: `steps` keeps tag 1, `response` keeps tag 2, the three ABI exports and method numbers do not change.
- Cancel in Started is not exercised live: the real server passes through CancelRequested, which the six-phase Model deliberately omits. The restriction stays stated in `model.rs` and the realization README.
- Repeated cancel with the same request ID (server dedup) and with a different one (FailedPrecondition) are not modeled; `closed_rejects` documents the first.
- The boundary gate must still pass: `Rejection` and the code names are generic status vocabulary, not Temporal; the numeric gRPC table lives in `model/temporal`.
- Goldens for the mini activity change once (new self-loop rows, one new claim); Kettle and umpire fixtures must not change.

## Acceptance Criteria

- **R1:** `reject!(rejection)` inside a unit-returning `#[on]` handler emits a self-loop edge with `Rejected(rejection)`; handlers without it behave as today and `#[machine]` keeps `Outcome = ()` for modules without `reject!`. A `#[postcondition]` may take `outcome: &Outcome`. Fixtures cover accepted, disabled, rejected, and a rejecting helper; rows and report spell outcomes. Errors: a rejecting step that changed the state is a checker refusal naming the step; `reject!` outside a frame is inert; the macro snapshot and every existing golden are unchanged.
- **R2:** `Phase`, `Phased` and `closable::{terminal_remains, closed_rejects}` exist in `umpire_dsl`, registered from a Model by delegating postconditions. The mini activity Model implements them, names its terminal set once, and `Client::Cancel` in a closed phase rejects NotFound; cancel in Scheduled and Started still enters Canceled. Errors: a hypothetical reopening edge fails `terminal_remains`; a hypothetical accepted or wrong-rejection step from a closed phase fails `closed_rejects`; the fixture controls assert both.
- **R3:** `Explanation.outcomes` is parallel to `steps`; `Realize::matches` receives the replayed outcome; `Observation` carries either a response or an error. Judge explains a NotFound cancel observation after closure with one `Client::Cancel` step and outcome `Rejected(NotFound)` and leaves the phase unchanged; a successful cancel response in Scheduled is explained with `Accepted` and reaches Canceled. Errors: an error observation on an unknown method is Unsupported; a recognized method with a code outside the table or contradicting the replay is Unexplained and freezes the run; malformed envelopes are MalformedRequest; old pinned bytes remain pinned.
- **R4:** The real in-process server and SDK worker run two new scenarios: cancel after completion judged NotFound, and cancel before the worker starts judged Accepted then Canceled by Describe. The Go host carries the error observation without reading Temporal types. Errors: a mutated status code is Unexplained with expected actions; the slice gate, model gate and boundary gate pass; DESIGN records the first evidence-driven DSL expansion and the realization README documents the cancel projection and the Started restriction.

## Boundaries

- No reason strings on rejections, no FailedPrecondition for repeated cancel, no request-ID dedup, no late worker replies after closure (the archive omitted those too).
- No change to `Projection`, `AnswerPlan`, `Answer`, the ABI exports or method numbers.
- No other archived capability (Terminable, Pausable, Pollable, Describable, Retryable, Expirable) and no capability registry; `Phased` is the foundation only.
- No `#[phase]`, `#[closable]` or any new attribute; no `Result`-returning handler form.
- Kettle stays unphased and untouched.
- Starts after fn-216.4, which owns the `temporal.rs` cleanup this spec builds on.

## Decision Context

2026-10-19, owner. A request answered with an error is a Model step, not a disabled action; the realization judges it against the actual gRPC status. Design agreed in conversation: statement-form `reject!` mirroring `check!` instead of `-> Result<(), Rejection>` handlers; a hand-implemented `Phased` trait with the archive's two `Closable` laws as shared functions rather than attributes or a capability registry; `Outcome` type switched on by the presence of `reject!` so existing goldens stand. Rejected alternatives, one line each: a Model-declared outcome enum per Model (noisier, and the archive converged on shared outcomes); carrying the outcome inside the spelled step string (would change pinned `steps` bytes); a fourth envelope for RPC errors (a `oneof` on `Observation` is wire-compatible and smaller).

## Quick commands

```bash
make model-check
make umpire-check-model-slice
CGO_ENABLED=0 mise exec -- go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice$' -count=1
```
