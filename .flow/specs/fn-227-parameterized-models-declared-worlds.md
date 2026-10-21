# Parameterized Models: declared worlds, action inputs, and the activity Model with retries and timeouts (model, temporal)

## Conversation Evidence

> user (2026-10-20): "for model/ ie umpire; after the filed specs what should we do next?"
> assistant: parameterized Models delivered through the activity Model, since every downstream item (fn-217 `fill`/`validate`, TLC and Apalache scaling, Bank and Voting) waits on the one DSL gap DESIGN.md names with a settled shape and leaves unbuilt.
> user: "why second model and not extend existing?"
> assistant: extend the existing activity Model in place; "second Model" was the wrong word. Vertical rule, parameters belong to it, goldens and the live proof are the regression control, refinement later wants one Model at two levels. Cost: TLC refuses the Model today, so the spec carries a TLC-admission task.
> user: "what would we parameterize there on the tempporal model?"
> assistant: world parameters `maximum_attempts`, the four timeouts present or absent with small values, asynchronous cancel; dynamic `attempt`, deadline budgets, `cancel_requested`; action inputs `retryable` on RespondFailed, Heartbeat, Poll's expected attempt; computed starts over small domains; leave out task queues, versioning and local activities.
> user: "okay file spec for parameterized model"

## Goal & Context

Attribute-declared Models currently use unit worlds: `#[machine]` emits `struct World;`, generated handlers and claims discard the world, and `#[starts(range)]` enumerates static field ranges. `derive(Action)` already admits tuple/struct inputs through explicit `#[inputs(range)]` or `#[inputs([values])]` domains. This spec adds `Finite`, world-dependent domains and world-aware machine adapters on top of that existing support. DESIGN.md's "Parameterized Models: readiness" section records the settled future shape and defers each piece to "the first sample needing it". That sample is the activity Model: a retry policy, timeouts and heartbeat are what make its lifecycle more than seven edges, and they are what the real server already does on the slice fn-216 proved.

Why now. The DSL gap blocks three filed lines of work: fn-217's `fill`/`validate` need request fields with domains; fn-212's TLC and fn-211's Apalache were justified by scaling but have no Model whose state space exceeds the native checker; Bank and Voting are parked on `Finite` and world access. Extending the activity Model in place keeps the owner's vertical rule (one construct at a time through checker, TLA+ and the live slice) and keeps its goldens, the fn-218 cover tests and the SDK run as the regression control.

## Architecture & Data Models

- **Declared world.** Inside a `#[machine]` module, an optional `#[derive(World)] pub struct World { .. }` carries the run's fixed parameters. Absent, the macro keeps `struct World;` and every existing Model expands exactly as today. Present, `world()` is replaced by `World::new(..)` / the derived constructor; the checker explores one world value per call as it does now.
- **World access.** Handlers, claims, postconditions, scopes and progress methods may take an optional explicit trailing `w: &World` parameter. Adapters first recognize the trailing immutable World type, then interpret the remaining old positional arity, so before-only and World-only postconditions remain distinct regardless of parameter names. The generated adapters pass the world value the `Machine` impl already holds.
- **Computed starts.** A `#[starts]` impl block `fn starts(w: &World) -> impl IntoIterator<Item = Model>` supplies world-dependent starts. Field-level `#[starts(range)]` stays for literal/static domains; a Model declaring both is refused.
- **Action inputs.** `derive(Action)` admits tuple and struct variants whose field types implement `Finite` (`fn domain() -> impl Iterator<Item = Self>`, with derives for `bool`, small integer ranges, and fieldless enums). The action catalog enumerates each variant's domain product in declaration order, so structural determinism holds. A world-dependent domain comes through a `#[domain(path)]` attribute on an individual variant field, naming a function that receives `&World` and returns an iterator of that field's type. Multiple fields form the same declaration-order product; variant-level placement is refused.
- **Generics.** `#[machine]` accepts generics on `Model`, `World` and action enums and propagates their bounds to the generated adapters. `Spell` output stays the author's Rust names.
- **Lifting.** `umpire-lift` admits the declared world as TLA+ `CONSTANT`s with the explored world's values written into the `.cfg`, computed starts as the `Init` disjunction over the enumerated start set, and action inputs as existential choice over their finite domains. The activity Model's `impl Phased` and `reject!` arms are admitted by the lifter so the Model is no longer refused.
- **Activity Model, extended in place.** World: `maximum_attempts: 1..=3`, `schedule_to_start`, `start_to_close`, `schedule_to_close`, `heartbeat_timeout`, each `Option<Ticks>` over a small domain, `asynchronous_cancel: bool`. State: `phase`, `attempt`, per-deadline remaining budget in abstract ticks, `cancel_requested`. Actions: `Worker::RespondFailed { retryable: bool }`, `Worker::Heartbeat`, `Worker::Poll` carrying the expected attempt, `Deadline::Tick` as the one internal clock event that decrements live budgets and expires the first exhausted one, plus the existing `Client::Cancel`. `maximum_attempts = 1`, no timeouts and atomic cancel reproduce today's six-phase graph exactly.
- **Realization.** `Realize<activity::World>` builds the world from the activity options of the `StartActivityExecutionRequest` the run observed; Answer maps the callback's `Complete`/`Fail` payload to `RespondCompleted` or `RespondFailed { retryable }`; the SDK worker returns a retryable failure on the first attempt and completes on the second.

## API Contracts

- `#[derive(World)]` on exactly one struct per `#[machine]` module; its fields are `Clone + Eq + Hash + Debug + Spell`.
- Optional trailing `&World` parameter on `#[on]`, `#[invariant]`, `#[postcondition]`, `#[assumption]`, `#[scope]` and `#[progress]` methods; typed trailing-World detection precedes positional arity, names do not decide. Per-method `#[scope]` is a compatibility spelling feeding the existing assumption registration and order; grouped `#[scopes]` stays refused.
- `#[starts] impl Model { fn starts(w: &World) -> .. }` or field `#[starts(range)]`, never both. Refusal: "declare starts once".
- `derive(Action)` variant fields: `Finite` types, or `#[domain(path)]` naming `fn(&World) -> impl Iterator<Item = T>`. Refusal: a field type that is neither.
- `trait Finite: Sized { fn domain() -> impl Iterator<Item = Self>; }` in `umpire-dsl`, with `#[derive(Finite)]` for fieldless enums and `Finite` impls for `bool` and `RangeInclusive`-bounded newtypes.
- `Machine::actions(&self)` returns the expanded catalog (every variant times its domain product) in declaration order.
- No change to `Steps`, `Outcome`, `Verdict`, `Spell`'s rendering rules, the oracle protocol or the three Temporal envelopes.

## Edge Cases & Constraints

- Every Model without a declared world, inputs or computed starts keeps byte-identical goldens and expansion snapshots: both Kettles, every shared and checker-only fixture, and the activity Model until its own task changes it.
- An input domain product above a configurable catalog limit (default 4096 actions) is refused with the count, not explored.
- A `#[domain]` function that returns an empty iterator makes the variant never enabled; the report records the variant with zero actions rather than silently dropping it.
- Computed starts that yield duplicates are deduplicated with the existing shared-starts behaviour.
- Rust type inference: generated adapters supply the world type explicitly so authors never write `Claim::<Self>` at the call site in attribute Models.
- Activity: the schedule-to-close budget spans attempts; start-to-close and heartbeat budgets reset on each `Poll`; a retryable failure with `attempt == maximum_attempts` is final `Failed`; a non-retryable failure is final on any attempt; `Tick` with no live budget is disabled; cancel with `asynchronous_cancel` sets `cancel_requested` in `Started` and the next worker response or heartbeat closes as `Canceled`.
- R30 budgets hold: inner loop under 10 s and gate under 30 s warm; the activity's default fixture world is bounded so the native checker finishes well inside them, and the larger worlds are exercised through TLC under `UMPIRE_BACKENDS=require`.
- No new crates beyond what the workspace already admits; the fn-210 R18 dependency policy applies.

## Acceptance Criteria

- **R1:** A `#[machine]` module may declare `#[derive(World)]`, take `&World` in handlers and claims, and supply computed starts through a `#[starts]` impl block; a shared fixture with a two-parameter world and computed starts has report/rows/states/paths goldens and agrees with TLC. Errors: both start forms declared, a second `World` struct, or a world parameter in a wrong position are compile errors pinned by trybuild; every existing golden and expansion snapshot is byte-identical.
- **R2:** `derive(Action)` admits variants with `Finite` fields and `#[domain]` world-dependent fields; `actions()` enumerates the domain product in declaration order; a shared fixture with a `bool` input and a world-sized range input has goldens and agrees with TLC. Errors: a non-`Finite` field without `#[domain]` is a compile error pinned by trybuild; a catalog above the limit is a `Refusal` naming the count.
- **R3:** `umpire-lift` emits declared worlds as `CONSTANT`s with `.cfg` values, computed starts as the enumerated `Init` set and inputs as finite existential choice, and admits `impl Phased` and `reject!` arms, so the activity Model is no longer refused. Errors: an unsupported world field type is a located refusal; TLC disagreement on any fixture fails `UMPIRE_BACKENDS=require`.
- **R4:** The activity Model carries `maximum_attempts` and `RespondFailed { retryable }`: a retryable failure below the cap returns to `Scheduled` with `attempt + 1`, otherwise `Failed`; the fixture world `maximum_attempts = 1` with no timeouts reproduces the six-phase goldens exactly, and the default fixture world `maximum_attempts = 3` has new goldens; fn-218's activity cover test is updated to the new space with its `Cases(n)` stated. Errors: a stale `Poll` naming the wrong attempt is `Rejected`; terminal laws keep holding at every terminal phase and attempt.
- **R5:** The activity Model carries the four optional timeouts, `Heartbeat`, `Tick` and `asynchronous_cancel`, with a progress claim that a started activity under a fair worker eventually closes and a safety claim that schedule-to-close bounds the total ticks; both verified natively on the default world and by TLC on the larger worlds. Errors: a world with every timeout absent has no `Tick` edge; expiring budgets produce `TimedOut` with the expired deadline recorded as a fact.
- **R6:** The realization builds the world from the observed start request's activity options, Answer selects `RespondFailed { retryable }` from the callback payload, and the live SDK run proves one retryable failure followed by completion with `attempt` advancing through real Describe observations; cancellation and the existing Complete/Fail/HostRefusal controls keep passing. Errors: a start request whose options exceed the admitted domains is `Unsupported` and freezes the run with the offending field named.
- **R7:** DESIGN.md gains a "Parameterized worlds and inputs" section recording the chosen spellings, the activity world and state table, the native and TLC state counts per fixture world, and `make model-time` output; the readiness table rows become "delivered" with their final shape. No error surface beyond the gate.

## Boundaries

- No task queue, worker versioning, local activity, eager dispatch or Nexus parameters; they parameterize routing, not the lifecycle.
- No real clock: `Tick` is an abstract unit; wall-clock timeout values in requests map to tick budgets through a declared domain, not to durations.
- No Bank or Voting port; `Finite` and world access are built for the activity Model and the two shared fixtures only.
- No nondeterministic choice within one action and no typed outcomes beyond what the activity needs; facts are limited to the expired-deadline fact in R5.
- No Apalache work; fn-211 consumes the same lifted tree afterwards.
- No Testpilot Case plan work (R33 to R40); this spec supplies the inputs that plan will need.
- No extraction per MIGRATE.md until this spec closes, since it changes the `umpire/` surface.

## Decision Context

2026-10-20, owner: file the spec. The root agent recommended it as the next piece after the filed specs fn-218, fn-217 and fn-211, and the owner asked why a second Model rather than extending the existing one; the answer, accepted, is to extend the activity Model in place so its goldens, cover tests and live proof show each change.

Placement, owner decision 2026-10-20: after fn-218 (its activity cover test must be updated here) and fn-217 (its catalog over `StartActivityExecutionRequest` supplies the fields the world reads), and **before fn-211**, so a parameterized Model exists when the Apalache backend arrives. The owner confirmed: "Moving parameterization ahead of it would give Apalache something worth checking on arrival." fn-211 therefore depends on this spec; the priority order becomes fn-218, fn-217, this spec, fn-211.

Rejected: a sibling activity Model beside the six-phase one (breadth over depth, and refinement later wants one Model at two levels); a real-time clock in the Model (TLC and native exploration need an abstract tick, and the realization maps durations to budgets); `Option<()>`-style world access through a global (not wasm-safe and hides the parameter from `Spell` output). The six-phase graph is kept reachable as the `maximum_attempts = 1`, no-timeout world and pinned as a fixture rather than preserved as a separate Model.

## Resolved via Codebase

- World delivery uses two cohesive checkpoints. The existing first task owns native authoring, typed association and explicit suite ownership; a new dependent source task owns typed fixture binding, all configurations and complete native/TLC fixture evidence. Action-input work waits for both. Existing task identities, all R-IDs and the activity compatibility decision remain intact.
- Declared World is the existing Machine identity. Construction uses a field-order constructor with the struct's visibility; Model generic arguments determine its World type and propagate authored bounds. Type association is separate from implicit unit-world construction, preserving manual registrations and state-only shorthand. Explicit suite operations retain one owned selected World through searches and Runs, including empty starts; declared parameters never acquire fabricated defaults.
- Bound source emission validates ordered World fields and complete starts against the authored schema using the existing source type/value representation. A compiler-checked fixture projection supplies the selected World and authored starts to emission and the same-World native replay. Unit-source APIs retain their old path. World/source/state/before names remain distinct; all five configuration families receive identical bindings. The source transition relation and claims remain independently translated from Rust source.
- Research at `9438ccbf6d` found existing tuple/struct Action domain products and generic derive bounds in `model/umpire/dsl/macros/src/action.rs`. The `action_inputs_values`, `action_inputs_limits` and `action_inputs_builders` controls already cover explicit domains, declaration order and bounded lazy path enumeration. R2 still requires Finite, world-dependent domains, the machine catalog limit and backend agreement. Reuse these controls and mechanisms rather than replacing them.
- The Rust workspace renders through Display/Debug; it has no current Spell trait. Spell in this spec denotes the existing author-name rendering contract. The implementation must preserve that contract rather than adding a parallel rendering abstraction solely to match stale prose. Generic derives exist; generic `#[machine]` registration and adapters remain unsupported.
- Native Machine/Claim interfaces receive the world, while generated adapters and static Actor domains discard it. Registered::world() also constructs unit worlds for Space/Run. World-dependent actions, paths, claims and covered-pattern queries need the same explicit world binding, with the existing unit-world shorthand preserved.
- Source lifting currently takes source without live world values and represents starts as independent field products. R1/R3 require explicit world binding in every main/property/progress cfg and correlated whole-state computed starts. Backend counterexamples must retain action-input identity. Existing closure-law helpers and rejecting self-loops need independent admission; removing their laws would not make the activity Model conform.
- Runtime currently owns a private machine created by Runtime::new, and Realize::observe/answer receive no mutable world. R6 requires transactional live world admission from actual Start evidence, invalidation of old exploration evidence and publication rollback. Its out-of-domain Unsupported-plus-freeze behavior is specific to this admission; existing malformed/unsupported observation refusals preserve state.

## Resolved via Project Docs

- `model/README.md` already documents explicit `#[inputs]` support. DESIGN.md's unit-only readiness row is stale and remains a documentation obligation in R7. All seven acceptance criteria, the in-place activity requirement, prior evidence and dependencies remain unchanged.
- The owner delegated recommendations through the active milestone goal. This refinement asked zero questions and changed only Goal & Context plus these evidence/open-contract sections. Both ready state and all task statuses remain unchanged. The report `.flow/tmp/fn227-prep.md` supplies source locations and a proposed implementation split for later planning; it records no test execution.

## Open Questions

- R4's literal old-golden compatibility conflicts with new attempt/action-input spelling and R5's no-Tick behavior when every timeout is absent. The current six-phase Model has directly enabled ScheduleToStart/StartToClose deadline actions. Research and formal review found no exact witness under the combined contracts. Preserve the independent old evidence and obtain owner approval of a consistent compatibility contract before changing those goldens. No normalization, waiver or removal of existing edges is approved. Owner: human, before the activity rewrite.
- The current StartActivityExecutionRequest has retry policy and the four timeout fields but no cancellation-type field; standalone SDK Start options do not supply it either. The pinned API documents cooperative cancellation as a protocol property. Source inspection also shows that heartbeat reports a pending cancellation without closing, while completion/failure remain permitted after cancellation. This contradicts R5's stronger next-response/heartbeat closure rule. Owner approval of the independent cancellation contract and its evidence source is required before .4/.5; no Model transition has been fitted to server behavior. Bind the actual filled Start request plus successful new-run result to admission, and do not invent a protobuf field or silently choose a World value.

## Plan review follow-up

The first formal plan review returned NEEDS_WORK on 2026-10-21, session 01a15ea6-c45d-7d01-b74c-ea7c7829f57e, actual gpt-6.1-sol/high. It reported six findings across R2-R6 and no maintainability findings. Field-level domain placement is now resolved consistently. Remaining engineering work includes explicit Start evidence transport and transactional admission, expired-deadline fact authoring/lowering/replay, and bounded optional-tick source admission/decoding. These are still open and must be made executable before the affected tasks. The review has not passed and no implementation task has been admitted.

The reviewer independently confirmed that R4's literal old-byte witness is impossible under its no-timeout World and R5's Tick rule. The old deadline edges and new state/input rendering are separate contradictions. Preserving old files alone, hiding new fields or projecting graphs cannot discharge the unchanged literal requirement. The owner must authorize a different compatibility contract before the activity rewrite. A proposed semantic-refinement witness uses one attempt, atomic cancellation and one-tick schedule-to-start/start-to-close budgets, with an explicit state/action/fact projection and exact comparison against the retained old lifecycle graph. This is a proposal only; R4 and the old goldens remain unchanged.

## Quick commands

```bash
make model-check
UMPIRE_BACKENDS=require make model-check
make model-time
```

## Early proof point

Task fn-227-parameterized-models-declared-worlds.1 proves native declared World ownership and computed starts through the suite. Task fn-227-parameterized-models-declared-worlds.7 proves independent source binding and actual TLC agreement before action-input work begins. Reconsider the binding seam if either fails; do not weaken the existing Model or its goldens.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | A `#[machine]` module may declare `#[derive(World)]`, take `&World` in handlers and claims, and supply computed starts through a `#[starts]` impl block; a shared fixture with a two-parameter world and computed starts has report/rows/states/paths goldens and agrees with TLC. Errors: both start forms declared, a second `World` struct, or a world parameter in a wrong position are compile errors pinned by trybuild; every existing golden and expansion snapshot is byte-identical. | fn-227-parameterized-models-declared-worlds.1, fn-227-parameterized-models-declared-worlds.7 | — |
| R2 | `derive(Action)` admits variants with `Finite` fields and `#[domain]` world-dependent fields; `actions()` enumerates the domain product in declaration order; a shared fixture with a `bool` input and a world-sized range input has goldens and agrees with TLC. Errors: a non-`Finite` field without `#[domain]` is a compile error pinned by trybuild; a catalog above the limit is a `Refusal` naming the count. | fn-227-parameterized-models-declared-worlds.2 | — |
| R3 | `umpire-lift` emits declared worlds as `CONSTANT`s with `.cfg` values, computed starts as the enumerated `Init` set and inputs as finite existential choice, and admits `impl Phased` and `reject!` arms, so the activity Model is no longer refused. Errors: an unsupported world field type is a located refusal; TLC disagreement on any fixture fails `UMPIRE_BACKENDS=require`. | fn-227-parameterized-models-declared-worlds.1, fn-227-parameterized-models-declared-worlds.2, fn-227-parameterized-models-declared-worlds.3, fn-227-parameterized-models-declared-worlds.7 | — |
| R4 | The activity Model carries `maximum_attempts` and `RespondFailed { retryable }`: a retryable failure below the cap returns to `Scheduled` with `attempt + 1`, otherwise `Failed`; the fixture world `maximum_attempts = 1` with no timeouts reproduces the six-phase goldens exactly, and the default fixture world `maximum_attempts = 3` has new goldens; fn-218's activity cover test is updated to the new space with its `Cases(n)` stated. Errors: a stale `Poll` naming the wrong attempt is `Rejected`; terminal laws keep holding at every terminal phase and attempt. | fn-227-parameterized-models-declared-worlds.3 | — |
| R5 | The activity Model carries the four optional timeouts, `Heartbeat`, `Tick` and `asynchronous_cancel`, with a progress claim that a started activity under a fair worker eventually closes and a safety claim that schedule-to-close bounds the total ticks; both verified natively on the default world and by TLC on the larger worlds. Errors: a world with every timeout absent has no `Tick` edge; expiring budgets produce `TimedOut` with the expired deadline recorded as a fact. | fn-227-parameterized-models-declared-worlds.4 | — |
| R6 | The realization builds the world from the observed start request's activity options, Answer selects `RespondFailed { retryable }` from the callback payload, and the live SDK run proves one retryable failure followed by completion with `attempt` advancing through real Describe observations; cancellation and the existing Complete/Fail/HostRefusal controls keep passing. Errors: a start request whose options exceed the admitted domains is `Unsupported` and freezes the run with the offending field named. | fn-227-parameterized-models-declared-worlds.5 | — |
| R7 | DESIGN.md gains a "Parameterized worlds and inputs" section recording the chosen spellings, the activity world and state table, the native and TLC state counts per fixture world, and `make model-time` output; the readiness table rows become "delivered" with their final shape. No error surface beyond the gate. | fn-227-parameterized-models-declared-worlds.6 | — |
