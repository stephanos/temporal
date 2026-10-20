# Realization ergonomics: shared Temporal helpers, runtime constructors, and the activity Realization rewrite (model, temporal)

## Conversation Evidence

> user (2026-10-20): "can we find ways to make temporal/model/temporal/features/activity/temporal.rs nicer/shorter/beautiful/elegant/clean/nice/easy to read/easy to maintian?"
> assistant: 230 lines, about 150 of them plumbing every future Realization repeats. Three groups: shared Temporal helpers (observation decoding, gRPC code to Rejection, method names, payload reading, failure building); runtime constructors (`Transition::from`, `Projection::step`, `AnswerPlan::reply`, consider folding `matches` into `observe`); local shape (`TryFrom<ActivityExecutionInfo> for Phase` beside the Model, `payload.clone()`, drop the `#[path]` prelude if the fixture runner allows, import the actor enums). Target: a sixty-line Realization reading as "Describe shows a phase, Cancel shows an outcome, a Poll callback answers Complete or Fail". Land before fn-227.5 and fn-228.2.
> user: "approved"

## Goal & Context

The activity Realization is the only Model-to-system binding in Rust and the template the workflow Model will copy. Today most of it is protobuf plumbing: envelope unpacking, result oneof splitting, a gRPC status table, hand-typed method paths, payload metadata checks and nested failure literals. fn-227.5 (world from the start request, retryable failures) and fn-228.2 (Propose) both extend this file, so its shape must be settled first. The goal is a Realization that reads as Model vocabulary, with the Temporal mechanics in helpers the next Realization reuses unchanged.

## Architecture & Data Models

- **Runtime constructors in `umpire-wasm`** (renamed by fn-228.1 later; no dependency): `impl<A> From<A> for Transition<A>` with choice zero; `Projection::step(action)` for a single one-step candidate and `Projection::steps(paths)` for several; `AnswerPlan::new(delivery, response_action, reply)` taking the system reply `Any` so the envelope packing stays in the Temporal helper. `Projection::Candidates` gains an optional expected outcome per candidate so `Realize::matches` is removed from the trait: the Runtime compares the final edge outcome itself. A candidate with no expected outcome matches any outcome, which is today's Describe behaviour.
- **Shared Temporal helpers** in the realization crate's `support` module (generic across Temporal RPCs, no activity names): `Observed` (decoded `umpire.temporal.v1.Observation` with the method and either a typed response via `response::<T>()` or a gRPC `code`); `Rejection::from_grpc(code) -> Option<Rejection>` as a free function `rejection(code)`; `method::<Service>("Name")` or generated constants for full RPC paths; `plain_text(&Payload) -> Result<&[u8], Refusal>` enforcing the single `encoding = binary/plain` metadata entry; `Failure::application(kind, message, non_retryable)`; `reply(request: &impl Name) -> Any` packing the `Reply` envelope; `callback::<T>(any) -> Result<T, Refusal>` unpacking `Callback` and its task.
- **Activity Realization** rewritten against both: `impl TryFrom<ActivityExecutionInfo> for Phase` beside the Model (six valid rows, one error arm per cause); the Realization's `observe` matches on `(method, Observed)` and returns `Projection::step(..)` with expected outcomes for Cancel; `answer` reads the policy through `plain_text` and builds the reply through the helpers; actor enums imported, no `activity::` qualification; the `#[path = "model.rs"]` prelude replaced by a normal module if the fixture runner permits, otherwise kept with a one-line reason.

## API Contracts

- `umpire-wasm`: `From<A> for Transition<A>`, `Projection::{step, steps}`, `Projection::Candidates(Vec<Candidate<A>>)` where `Candidate { path: Vec<Transition<A>>, outcome: Option<Outcome> }`, `AnswerPlan::new`; `Realize` loses `matches`. The oracle protocol and wire bytes are unchanged.
- Realization crate `support` module with the helpers listed above; every helper returns `Refusal` with the same messages the activity file uses today where a message is pinned by a test.
- The activity Realization keeps `runtime()`, `Realization` and `activity` as its public items.

## Edge Cases & Constraints

- Every golden, wire golden, the Go host parity tests, the dedicated slice (Complete, Fail, HostRefusal, Cancellation) and the fn-218 cover tests pass unchanged; the activity report, rows, states and lift goldens are byte-identical.
- Refusal messages that the live or protocol tests assert on keep their text; others may be reworded only if no test pins them.
- Removing `matches` is a trait change: the only implementor is the activity Realization; the trait Kettle guest uses no Realization and is unaffected. If removal turns out to change any judgment on the slice, keep `matches` and record why in DESIGN.md.
- Helpers contain no activity names (R5 of fn-216's boundary applies to `umpire-wasm`; the `support` module is Temporal-specific by design and lives in `model/temporal`).
- R30 budgets hold; no new crates.

## Acceptance Criteria

- **R1:** `umpire-wasm` provides `From<A> for Transition<A>`, `Projection::step`/`steps`, `Candidate` with optional expected outcome, and `AnswerPlan::new`; `Realize::matches` is removed and the Runtime compares the final edge outcome itself. Errors: a candidate whose expected outcome differs from the replayed outcome is Unexplained exactly as the old `matches` false was; protocol wire goldens and Go parity unchanged.
- **R2:** The realization crate's `support` module provides `Observed`, `rejection(code)`, RPC method naming, `plain_text`, `Failure::application`, `reply` and `callback`, each with a unit control for its refusal cases, and contains no activity-specific name. Errors: missing method, missing result, missing payload, wrong metadata, unsupported encoding and unknown Any type each yield the same refusal code as today.
- **R3:** The activity Realization is rewritten against R1 and R2 with `TryFrom<ActivityExecutionInfo> for Phase` beside the Model, actor enums imported, and the body at most eighty lines excluding imports and docs; all activity goldens byte-identical; the dedicated live slice and cover tests pass. Errors: no error surface beyond the gate.

## Boundaries

- No change to the activity Model, its goldens or its lifted form.
- No Propose, no world, no retryable input; those arrive with fn-228.2 and fn-227.5 on top of this shape.
- No rename of crates (fn-228.1).
- No generalization of `support` beyond what the activity file uses today; the workflow Model adds what it needs.

## Decision Context

2026-10-20, owner: approved the assessment as a small spec. Placement: before fn-227.3 touches the activity crate and before fn-227.5 and fn-228.2 extend the Realization; it has no spec dependency since fn-216 is closed, and it should run while fn-218.3 and fn-217 proceed in other crates. Rejected: leaving the plumbing in place until the workflow Model forces the duplication (two copies to clean instead of one); moving the Temporal helpers into `umpire-wasm` (they name Temporal types, which the boundary forbids).
