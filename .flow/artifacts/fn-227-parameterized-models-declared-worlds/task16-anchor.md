# Worker anchor bundle - fn-227-parameterized-models-declared-worlds.16 (spec fn-227-parameterized-models-declared-worlds)

Each section is the verbatim output of the command it is labeled with, in fixed order, untruncated. The bundle is a floor, not a ceiling - memory keyword-search and every further read remain available.

===== [1/11] task_show: `flowctl show fn-227-parameterized-models-declared-worlds.16 --json` =====
{
  "success": true,
  "assignee": "stephanos@users.noreply.github.com",
  "claim_note": "",
  "claimed_at": "2026-10-22T20:17:56.347843Z",
  "created_at": "2026-10-22T06:08:11.479000Z",
  "depends_on": [
    "fn-227-parameterized-models-declared-worlds.15"
  ],
  "id": "fn-227-parameterized-models-declared-worlds.16",
  "priority": null,
  "spec": "fn-227-parameterized-models-declared-worlds",
  "spec_path": ".flow/tasks/fn-227-parameterized-models-declared-worlds.16.md",
  "status": "in_progress",
  "title": "Named intermediates in the IR so admitted trees stay linear in conditional returns",
  "updated_at": "2026-10-22T20:17:56.347822Z",
  "impl": null,
  "review": null,
  "sync": null,
  "status_source": "flow-state"
}

===== [2/11] task_md: `flowctl cat fn-227-parameterized-models-declared-worlds.16` =====
---
satisfies: [R3]
---
# fn-227-parameterized-models-declared-worlds.16 Named intermediates in the IR so admitted trees stay linear in conditional returns

## Description
From fn-227.15's reviews (`.flow/tmp/fn22715-review.md` L1, `fn22715-review2.md` N2). Emitted TLA+ is linear in conditional returns since fn-227.15, but the front end's admitted tree and the macro-embedded `__lifted` data still grow about 7x per return when a later condition or update reads a field changed after a conditional return (and a run of updates after one early return doubles the tree per update). fn-227.15 added a 10,000-node budget so such steps refuse at the statement instead of exhausting rustc memory (measured: about 24k-64k nodes OOM), which caps the pattern at about four sequential returns.

Introduce named intermediates in the IR (`ExprKind::Local` plus per-branch definitions, emitted as TLA+ `LET`) so the tree, the embedded data and compile time stay linear. Then raise or remove the budget. Run before fn-227.3, whose retries add conditional rejections to the activity.

### Verification
Size tests over the admitted tree and embedded data (not only emitted TLA+) for sequential returns and repeated updates; TLC verdicts, state counts and replay unchanged on every fixture; listed TLA+ text goldens may change, justified. Quick then Full gates; record compile time for a pathological probe before/after.


**Size:** L
**Touches:** `model/umpire/ir/**`, `model/umpire/front/**`, `model/umpire/lift/**`, `model/umpire/backends/tlc/**`, `model/umpire/tests/**`, `model/umpire/dsl/macros/**`, `model/README.md`, `model/DESIGN.md`, `model/temporal/feature/activity/expected/**` (only justified emitted TLA+ text), `model/temporal/hosts/go/umpire_temporal_guest.wasm` (generated refresh).

### Execution and gates

Consume fn-238.9's derived constructor printer before adding the new IR fields and variant. Keep generic IR/front-end/lifter ownership; use the existing emitted LET/name-collision machinery where it fits. Preserve first-return semantics, conditional execution, checked arithmetic, payloads, selected World substitution, phase/rejection and terminal-law behavior. An authored name must not capture a generated intermediate. Source/TLC replay must resolve the same typed values. This changes the admitted data representation, not the authored Model semantics.

Before changing expectations, retain baseline semantic/golden hashes and per-fixture TLC counts/verdicts, and measure the existing pathological source probe. Add discriminating linear-growth tests for both sequential returns and repeated updates over admitted tree size and embedded constructor data, plus actual compilation past the old refusal/OOM range. Emitted-text size alone is insufficient. Raise or remove the obsolete node cap only with measured safe compilation. Preserve canonical Model source, native reports/states/rows, verdicts and replay outputs. List each necessary emitted TLA+ text golden change with its mechanism; do not globally refresh unrelated goldens. A needed path outside Touches must be reported to the conductor before editing.

Quick from model/: `mise exec -- cargo test -p umpire-front -p umpire-ir`; `mise exec -- cargo test -p umpire-lift --test phased`; relevant source/replay controls under required TLC. Include the new growth/collision/compilation controls in focused runs. Full from root: one `UMPIRE_BACKENDS=require make model-time`, whose nested `make model-check` must explicitly show required TLC execution and all stages passing. This satisfies the complete full gate and timing together; preserve the actual command. Use an outer bound of at least 1800s for this full/timing run, and at least 1300s for the gap target whose Go test has a 20m bound. Do not repeat a completed unchanged full gate merely for a second command spelling. If changed closure bytes require the embedded guest refresh, generate through `make umpire-guest` in a separate commit once source settles, then actual `make umpire-check-model-slice` and `make umpire-check-model-gaps`; don't fit expected semantics to the server. Preserve literal timing output, probe before/after, TLC counts/verdicts/replays and golden comparisons in task-owned artifacts. Record soft-target overruns honestly.
## Acceptance
- [ ] Admitted tree and embedded data grow linearly with sequential conditional returns and repeated updates (size tests).
- [ ] Node budget raised or removed with a probe showing no rustc blow-up.
- [ ] TLC verdicts, state counts and replay unchanged; changed goldens justified.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

===== [3/11] spec_show: `flowctl show fn-227-parameterized-models-declared-worlds --json` =====
{
  "success": true,
  "branch_name": "fn-227-parameterized-models-declared-worlds",
  "completion_review_status": "unknown",
  "completion_reviewed_at": null,
  "created_at": "2026-10-20T18:58:19.229711Z",
  "default_impl": null,
  "default_review": null,
  "default_sync": null,
  "depends_on_epics": [
    "fn-218-test-space-patterns-cover-and-the-full",
    "fn-217-annotation-catalog-over-the-temporal"
  ],
  "id": "fn-227-parameterized-models-declared-worlds",
  "impl_review_rounds": {
    "fn-227-parameterized-models-declared-worlds.1": 0
  },
  "next_task": 1,
  "plan_review_rounds": 0,
  "plan_review_status": "ship",
  "plan_reviewed_at": "2026-10-21T16:24:48.439136Z",
  "review_hash_epoch": {
    "impl:fn-227-parameterized-models-declared-worlds.1": 1,
    "plan": 1,
    "plan#completion": 0
  },
  "review_pending_rounds": {},
  "review_reservations": {},
  "review_transport_failures": {
    "impl:fn-227-parameterized-models-declared-worlds.1": 0,
    "plan": 0
  },
  "spec_path": ".flow/specs/fn-227-parameterized-models-declared-worlds.md",
  "status": "open",
  "title": "Parameterized Models: declared worlds, action inputs, and the activity Model with retries (model, temporal)",
  "updated_at": "2026-10-21T21:39:19.727582Z",
  "tasks": [
    {
      "id": "fn-227-parameterized-models-declared-worlds.1",
      "title": "Declared World, typed world adapters and computed starts with owned native suite binding",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": []
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.2",
      "title": "After native World: Finite, typed World-dependent action domains and checked native catalog admission",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.1",
        "fn-227-parameterized-models-declared-worlds.7"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.3",
      "title": "Activity retries and retained-lifecycle refinement checkpoint",
      "status": "todo",
      "status_source": "committed",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.2",
        "fn-227-parameterized-models-declared-worlds.11",
        "fn-227-parameterized-models-declared-worlds.15",
        "fn-227-parameterized-models-declared-worlds.16"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.5",
      "title": "Transactional live World admission and SDK retry proof",
      "status": "todo",
      "status_source": "committed",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.3",
        "fn-227-parameterized-models-declared-worlds.12"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.6",
      "title": "After .5: DESIGN.md parameterized worlds write-up and readiness table closure",
      "status": "todo",
      "status_source": "committed",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.5"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.7",
      "title": "After native World: independent typed source binding, whole-state Init and parameterized_world TLC proof",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.1"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.8",
      "title": "Independent finite action-input binding, exact-payload source fairness and action_inputs TLC proof",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.2",
        "fn-227-parameterized-models-declared-worlds.7"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.11",
      "title": "Independent phase, rejection and closure-law source admission",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.8"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.12",
      "title": "Actual-request Observation transport and owned same-invocation capture",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": []
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.13",
      "title": "Simplify: one #[machine] generator, no module generics, method-level #[starts]",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": []
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.14",
      "title": "Simplify: derived value handoff replaces the augmented-copy binding route",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.13"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.15",
      "title": "Compact emitted TLA+ for conditional returns before activity retries",
      "status": "done",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.11"
      ]
    },
    {
      "id": "fn-227-parameterized-models-declared-worlds.16",
      "title": "Named intermediates in the IR so admitted trees stay linear in conditional returns",
      "status": "in_progress",
      "status_source": "flow-state",
      "implicit_owner": false,
      "priority": null,
      "depends_on": [
        "fn-227-parameterized-models-declared-worlds.15"
      ]
    }
  ],
  "ready": false,
  "no_plan": false
}

===== [4/11] spec_md: `flowctl cat fn-227-parameterized-models-declared-worlds` =====
# Parameterized Models: declared worlds, action inputs, and the activity Model with retries (model, temporal)

> HTML render lens: local `.flow/artifacts/fn-227-parameterized-models-declared-worlds/spec.html` - regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (2026-10-20): "for model/ ie umpire; after the filed specs what should we do next?"
> assistant: parameterized Models delivered through the activity Model, since every downstream item (fn-217 `fill`/`validate`, TLC and Apalache scaling, Bank and Voting) waits on the one DSL gap DESIGN.md names with a settled shape and leaves unbuilt.
> user: "why second model and not extend existing?"
> assistant: extend the existing activity Model in place; "second Model" was the wrong word. Vertical rule, parameters belong to it, goldens and the live proof are the regression control, refinement later wants one Model at two levels. Cost: TLC refuses the Model today, so the spec carries a TLC-admission task.
> user: "what would we parameterize there on the tempporal model?"
> assistant: world parameters `maximum_attempts`, the four timeouts present or absent with small values, asynchronous cancel; dynamic `attempt`, deadline budgets, `cancel_requested`; action inputs `retryable` on RespondFailed, Heartbeat, Poll's expected attempt; computed starts over small domains; leave out task queues, versioning and local activities.
> user: "okay file spec for parameterized model"
> user (2026-10-21), approving the three compatibility, cancellation and request-evidence amendments described in Decision Context: "yes Approve all three"
> user (2026-10-21), owner decision: move the activity timeout semantics (old R5, tasks .9, .10 and .4, the cooperative-cancellation live part of R6 and .5, and the final one-tick compatibility World re-run) into a new spec, fn-237-activity-timeouts-heartbeat-and. Also: "we need TLC on the real, activity Model still", so R3 stays here in full, including .11.

## Goal & Context

Before this spec, attribute-declared Models used unit worlds: `#[machine]` emitted `struct World;`, generated handlers and claims discarded the world, and `#[starts(range)]` enumerated static field ranges. `derive(Action)` already admitted tuple/struct inputs through explicit `#[inputs(range)]` or `#[inputs([values])]` domains. This spec adds `Finite`, world-dependent domains and world-aware machine adapters on top of that support. DESIGN.md's "Parameterized Models: readiness" section records the settled future shape and defers each piece to "the first sample needing it". That sample is the activity Model: a retry policy, timeouts and heartbeat are what make its lifecycle more than seven edges, and they are what the real server already does on the slice fn-216 proved. This spec delivers the retry policy and TLC on the real activity Model; timeouts, heartbeat and cooperative cancellation moved to fn-237-activity-timeouts-heartbeat-and on 2026-10-21.

Why now. The DSL gap blocks three filed lines of work: fn-217's `fill`/`validate` need request fields with domains; fn-212's TLC and fn-211's Apalache were justified by scaling but have no Model whose state space exceeds the native checker; Bank and Voting are parked on `Finite` and world access. Extending the activity Model in place keeps the owner's vertical rule (one construct at a time through checker, TLA+ and the live slice) and keeps its goldens, the fn-218 cover tests and the SDK run as the regression control.

## Architecture & Data Models

- **Declared world.** Inside a `#[machine]` module, an optional `#[derive(World)] pub struct World { .. }` carries the run's fixed parameters. Absent, the macro keeps `struct World;` and every existing Model expands exactly as today. Present, `world()` is replaced by `World::new(..)` / the derived constructor; the checker explores one world value per call as it does now.
- **World access.** Handlers, claims, postconditions, scopes and progress methods may take an optional explicit trailing `w: &World` parameter. Adapters first recognize the trailing immutable World type, then interpret the remaining old positional arity, so before-only and World-only postconditions remain distinct regardless of parameter names. Values cross to the lifter through hidden derived `values()` exports on `derive(World)`/`derive(State)`/`derive(Action)` (owner decision 2026-10-21: the no-serialization rule is dropped for values only); meaning is still translated from source independently. The generated adapters pass the world value the `Machine` impl already holds.
- **Computed starts.** A `#[starts]` method in the main `impl Model` (any name, `fn(w: &World) -> impl IntoIterator<Item = Model>`) supplies world-dependent starts (method form by owner decision 2026-10-21, replacing the separate impl block). Field-level `#[starts(range)]` stays for literal/static domains; a Model declaring both is refused.
- **Action inputs.** `derive(Action)` admits tuple and struct variants whose field types implement `Finite` (`fn domain() -> impl Iterator<Item = Self>`, with derives for `bool`, small integer ranges, and fieldless enums). The action catalog enumerates each variant's domain product in declaration order, so structural determinism holds. A world-dependent domain comes through a `#[domain(path)]` attribute on an individual variant field, naming a function that receives `&World` and returns an iterator of that field's type. Multiple fields form the same declaration-order product; variant-level placement is refused.
- **Generics.** Removed by owner decision 2026-10-21: `#[machine]` modules refuse generic `Model`/`World`/actor declarations with a located error (no Model used them and the lifter refused them); standalone derives keep their generics. `Spell` output stays the author's Rust names.
- **Lifting.** `umpire-lift` admits the declared world as TLA+ `CONSTANT`s with the explored world's values written into the `.cfg`, computed starts as the `Init` disjunction over the enumerated start set, and action inputs as existential choice over their finite domains. The activity Model's `impl Phased` and `reject!` arms are admitted by the lifter so the Model is no longer refused.
- **Activity Model, extended in place.** World: `maximum_attempts: 1..=3`. State: `phase`, `attempt`. Actions include retryable failure, Poll's expected attempt, the existing direct deadline actions and the existing atomic `Client::Cancel`. The retained old lifecycle graph is the independent reference for an explicit semantic-refinement check. This spec's compatibility witness is the checkpoint World with one attempt, the retained direct deadline actions and atomic cancellation. The four optional timeouts, heartbeat, `Deadline::Tick`, cooperative cancellation with its acknowledgement action, and the final one-tick compatibility World moved to fn-237-activity-timeouts-heartbeat-and.
- **Realization.** `Realize<activity::World>` uses the actual filled `StartActivityExecutionRequest` and successful result captured from the same invocation. Request retry options supply `maximum_attempts`; request timeout fields are recorded with the Start receipt but not admitted as World parameters until fn-237. Answer maps actual worker completion and failure evidence. The SDK retry worker fails retryably once and completes on its second attempt. The explicit pinned cooperative-cancellation protocol binding moved to fn-237.

## API Contracts

- `#[derive(World)]` on exactly one struct per `#[machine]` module; its fields are `Clone + Eq + Hash + Debug + Spell`.
- Optional trailing `&World` parameter on `#[on]`, `#[invariant]`, `#[postcondition]`, `#[assumption]`, `#[scope]` and `#[progress]` methods; typed trailing-World detection precedes positional arity, names do not decide. Per-method `#[scope]` is a compatibility spelling feeding the existing assumption registration and order; grouped `#[scopes]` stays refused.
- `#[starts]` on an associated fn in the main `impl Model { }` (any name, optional `&World`) or field `#[starts(range)]`, never both. Refusal: "declare starts once".
- `derive(Action)` variant fields: `Finite` types, or `#[domain(path)]` naming `fn(&World) -> impl Iterator<Item = T>`. Refusal: a field type that is neither.
- `trait Finite: Sized { fn domain() -> impl Iterator<Item = Self>; }` in `umpire-dsl`, with `#[derive(Finite)]` for fieldless enums and `Finite` impls for `bool` and `RangeInclusive`-bounded newtypes.
- A bounded integer newtype declares its one backing-field range through `#[inputs(lo..=hi)]` under `derive(Finite)`. Enumeration and generated `try_new(raw) -> Option<Self>` / `contains(raw) -> bool` use that one declaration. Raw tuple construction remains possible. Explicit fixture/live admission validates configuration; authored native invariants detect invalid state updates, while source bindings and typed decoding independently refuse invalid values. No generic Machine/state validity hook or fallible World constructor is added.
- `Machine::actions(&self)` returns the expanded catalog (every variant times its domain product) in declaration order.
- Preserve the legacy Vec-returning action API and static Actor implementations. Add defaulted fallible catalog admission with exact count/overflow refusals, limit 4096 by default, and optional zero-variant metadata. Native consumers propagate it without turning errors into empty actions or panics. World-dependent actors expose a typed explicit bound-domain route and value/classification capability rather than inventing a static catalog. Existing static builders stay compatible; an unbound World-dependent wildcard refuses.
- Source action relations choose finite inputs existentially, but concrete action identity survives typed binding, decoding and replay. Existing only/without/fair lists retain exact-value meaning. Fairness for one payload specializes that action relation; stale or different payloads cannot discharge it through a variant-wide existential union.
- No change to `Steps`, `Outcome`, `Verdict`, `Spell`'s rendering rules or the core oracle protocol. The sole approved envelope amendment is optional request `Any` at fresh `Observation` tag 4. Existing field numbers, absent-request bytes, Callback, Reply and the three-envelope count remain unchanged.
- The deadline-fact support (opt-in `Option<F>` handler results) and optional bounded timeouts moved with old R5 to fn-237-activity-timeouts-heartbeat-and on 2026-10-21.

## Edge Cases & Constraints

- Every Model without a declared world, inputs or computed starts keeps byte-identical goldens and expansion snapshots: both Kettles, every shared and checker-only fixture, and the activity Model until its own task changes it.
- An input domain product above a configurable catalog limit (default 4096 actions) is refused with the count, not explored.
- A `#[domain]` function that returns an empty iterator makes the variant never enabled; the report records the variant with zero actions rather than silently dropping it.
- Computed starts that yield duplicates are deduplicated with the existing shared-starts behaviour.
- Rust type inference: generated adapters supply the world type explicitly so authors never write `Claim::<Self>` at the call site in attribute Models.
- Activity: a retryable failure with `attempt == maximum_attempts` is final `Failed`; a non-retryable failure is final on any attempt. Cancellation stays atomic in this spec. Deadline budgets (schedule-to-close spanning attempts, start-to-close and heartbeat reset on `Poll`, disabled `Tick` with no live budget) and cooperative cancellation (pending request reported by heartbeat, completion/failure races, explicit acknowledgement, live mode from the pinned protocol binding) moved to fn-237-activity-timeouts-heartbeat-and.
- R30 budgets hold: inner loop under 10 s and gate under 30 s warm; the activity's default fixture world is bounded so the native checker finishes well inside them, and the larger worlds are exercised through TLC under `UMPIRE_BACKENDS=require`.
- No new crates beyond what the workspace already admits; the fn-210 R18 dependency policy applies.

## Acceptance Criteria

- **R1:** A `#[machine]` module may declare `#[derive(World)]`, take `&World` in handlers and claims, and supply computed starts through a `#[starts]` impl block; a shared fixture with a two-parameter world and computed starts has report/rows/states/paths goldens and agrees with TLC. Errors: both start forms declared, a second `World` struct, or a world parameter in a wrong position are compile errors pinned by trybuild; every existing golden and expansion snapshot is byte-identical.
- **R2:** `derive(Action)` admits variants with `Finite` fields and `#[domain]` world-dependent fields; `actions()` enumerates the domain product in declaration order; a shared fixture with a `bool` input and a world-sized range input has goldens and agrees with TLC. Errors: a non-`Finite` field without `#[domain]` is a compile error pinned by trybuild; a catalog above the limit is a `Refusal` naming the count.
- **R3:** `umpire-lift` emits declared worlds as `CONSTANT`s with `.cfg` values, computed starts as the enumerated `Init` set and inputs as finite existential choice, and admits `impl Phased` and `reject!` arms, so the activity Model is no longer refused. Errors: an unsupported world field type is a located refusal; TLC disagreement on any fixture fails `UMPIRE_BACKENDS=require`.
- **R4:** The activity Model carries `maximum_attempts` and `RespondFailed { retryable }`: a retryable failure below the cap returns to `Scheduled` with `attempt + 1`, otherwise `Failed`. Preserve the old six-phase evidence and prove explicit lifecycle refinement against its graph using a stated state/action/fact projection and exact comparison of the projected behavior. This spec's compatibility witness is the checkpoint World with one attempt, the retained direct deadline actions and atomic cancellation; the final one-tick compatibility World moved to fn-237-activity-timeouts-heartbeat-and (2026-10-21). The default three-attempt World has new goldens; fn-218's activity cover test states its `Cases(n)`. Errors: a stale Poll is Rejected; terminal laws hold at every terminal phase/attempt; a changed old lifecycle edge must fail the refinement control. Old-byte equality is replaced only for this activity extension.
- R5 (retired ID, not reused): moved on 2026-10-21 to fn-237-activity-timeouts-heartbeat-and as its R2 — the four optional timeouts, Heartbeat, Tick, asynchronous cancellation, the fair-worker progress claim, the schedule-to-close total-tick bound and TimedOut with the exact deadline fact.
- **R6:** The realization binds a World from the actual filled Start request and successful same-invocation result, transported with optional `Observation.request` Any at tag 4. Admission covers `maximum_attempts`; request timeout fields are recorded with the Start receipt but not admitted as World parameters, and present values are not refused, until fn-237 admits them. Answer uses actual completion/failure evidence. The live SDK run proves retryable failure then completion through real Describe attempt observations; existing Complete, Fail, HostRefusal and context-Cancellation controls keep passing. Cooperative cancellation (the explicit pinned protocol binding and live held-attempt cancellation evidence) moved to fn-237-activity-timeouts-heartbeat-and (2026-10-21). Errors: out-of-domain options return field-named Unsupported and freeze admission; failed/crossed/missing evidence cannot install a World. Existing absent-request bytes, field numbers, Callback, Reply and core oracle protocol remain unchanged.
- **R7:** DESIGN.md gains a "Parameterized worlds and inputs" section recording the chosen spellings, the activity world and state table, the native and TLC state counts per fixture world, and `make model-time` output; the readiness table rows become "delivered" with their final shape. No error surface beyond the gate.

## Boundaries

- No task queue, worker versioning, local activity, eager dispatch or Nexus parameters; they parameterize routing, not the lifecycle.
- No real clock: `Tick` is an abstract unit; wall-clock timeout values in requests map to tick budgets through a declared domain, not to durations.
- No Bank or Voting port; `Finite` and world access are built for the activity Model and the two shared fixtures only.
- No nondeterministic choice within one action and no typed outcomes beyond what the activity needs; no facts (the expired-deadline fact moved with R5 to fn-237).
- No timeouts as World parameters, Heartbeat, Tick or cooperative cancellation; they moved to fn-237-activity-timeouts-heartbeat-and on 2026-10-21.
- No Apalache work; fn-211 consumes the same lifted tree afterwards.
- No Testpilot Case plan work (R33 to R40); this spec supplies the inputs that plan will need.
- No extraction per MIGRATE.md until this spec closes, since it changes the `umpire/` surface.

## Decision Context

2026-10-20, owner: file the spec. The root agent recommended it as the next piece after the filed specs fn-218, fn-217 and fn-211, and the owner asked why a second Model rather than extending the existing one; the answer, accepted, is to extend the activity Model in place so its goldens, cover tests and live proof show each change.

Placement, owner decision 2026-10-20: after fn-218 (its activity cover test must be updated here) and fn-217 (its catalog over `StartActivityExecutionRequest` supplies the fields the world reads), and **before fn-211**, so a parameterized Model exists when the Apalache backend arrives. The owner confirmed: "Moving parameterization ahead of it would give Apalache something worth checking on arrival." fn-211 therefore depends on this spec; the priority order becomes fn-218, fn-217, this spec, fn-211.

2026-10-21, owner approval: "yes Approve all three" explicitly approves the assistant's three named amendments. R4 uses lifecycle refinement against retained old evidence in place of impossible byte equality. R5 uses cooperative cancellation with explicit protocol evidence, including heartbeat signaling and completion/failure races. R6 adds optional actual-request evidence to Observation at tag 4 while preserving absent encodings. These approvals do not approve other protocol changes, remove laws, complete tasks or turn the existing NEEDS_WORK plan review into SHIP.

2026-10-21, owner decision: move the activity timeout semantics out of this spec into fn-237-activity-timeouts-heartbeat-and ("Activity timeouts, heartbeat and cooperative cancellation"). Moved: old R5 entirely; tasks .9 (F), .10 (O) and .4; the cooperative-cancellation live part of R6 and .5 (pinned protocol binding, held-attempt server cancellation with pending heartbeat and acknowledgement, completion-after-cancel race, timeout-to-tick admission); the final one-tick compatibility World refinement re-run; and a timeout/cancellation write-up. The owner said explicitly: "we need TLC on the real, activity Model still". This spec therefore keeps R3 in full, including .11 (source `impl Phased`/`reject!` admission, so the lifter no longer refuses the activity Model), and keeps .3, .12, .5 (retry part), .6 and the finished or committed tasks .1, .7, .2 and .8. The R5 cooperative-cancellation approval above now binds fn-237.

Rejected: a sibling activity Model beside the six-phase one (breadth over depth, and refinement later wants one Model at two levels); a real-time clock in the Model (TLC and native exploration need an abstract tick, and the realization maps durations to budgets); `Option<()>`-style world access through a global (not wasm-safe and hides the parameter from `Spell` output). Keep the old lifecycle evidence as the independent reference; the approved compatibility World and projection must prove its retained behavior within the one extended Model.

2026-10-21, owner decision after the simplification scout (`.flow/artifacts/fn-227-parameterized-models-declared-worlds/simplify-report.md`): add .13 (one `#[machine]` generator, no module generics, method-level `#[starts]`) and .14 (derived value handoff replacing the augmented-copy binding route) before .8's review resumes. Both keep checker, TLA+ and cfg goldens and TLC agreement byte-identical. The owner also confirmed TLC and Apalache scaling is crucial ("we'll expand the model dramatically very soon"), so independent source lifting stays required.

2026-10-21, owner: "the only diff should be that the native checker acts like an interpreter and the lifter generates TLA+ (and later Apalache)", accepting the shared-misreading risk. fn-238 builds that shared front end after .14; for remaining and future work it supersedes this spec's "independent source admission" and "no shared interpretation" wording. Typed counterexample replay stays.

## Resolved via Codebase

- World delivery uses two cohesive checkpoints. The existing first task owns native authoring, typed association and explicit suite ownership; a new dependent source task owns typed fixture binding, all configurations and complete native/TLC fixture evidence. Action-input work waits for both. Existing task identities, all R-IDs and the activity compatibility decision remain intact.
- Declared World is the existing Machine identity. Construction uses a field-order constructor with the struct's visibility; Model generic arguments determine its World type and propagate authored bounds. Type association is separate from implicit unit-world construction, preserving manual registrations and state-only shorthand. Explicit suite operations retain one owned selected World through searches and Runs, including empty starts; declared parameters never acquire fabricated defaults.
- Bound source emission validates ordered World fields and complete starts against the authored schema using the existing source type/value representation. A compiler-checked fixture projection supplies the selected World and authored starts to emission and the same-World native replay. Unit-source APIs retain their old path. World/source/state/before names remain distinct; all five configuration families receive identical bindings. The source transition relation and claims remain independently translated from Rust source.
- Native World is complete with its committed closure, SHIP implementation review and final gates. Compiler-checked source projection remains a separate checkpoint: generated helpers live inside the compiled machine module, independent admission reads original authored source, selection stays in its fixture lexical scope, and one owned World/complete-start enumeration feeds binding, checking and replay.
- Input delivery has native and independent source checkpoints. Support aliases in task briefs are I (input source, .8), P (phase/rejection source admission, .11) and T (request capture, .12); F (facts) and O (optional values), formerly .9 and .10, moved to fn-237.1 and fn-237.2 on 2026-10-21. Native inputs retain .2; retry/refinement, live admission and final documentation retain .3/.5/.6; activity deadlines (formerly .4) moved to fn-237.3. Each support slice closes its own vertical controls before its activity consumer. Overlapping source/generated ownership is scheduled serially; task dependencies express consumption, not permission to share a cache or checkout.
- Research at `9438ccbf6d` found existing tuple/struct Action domain products and generic derive bounds in `model/umpire/dsl/macros/src/action.rs`. The `action_inputs_values`, `action_inputs_limits` and `action_inputs_builders` controls already cover explicit domains, declaration order and bounded lazy path enumeration. R2 still requires Finite, world-dependent domains, the machine catalog limit and backend agreement. Reuse these controls and mechanisms rather than replacing them.
- The Rust workspace renders through Display/Debug; it has no Spell trait. Spell in this spec denotes the existing author-name rendering contract. Preserve it rather than adding a parallel rendering abstraction. The completed native checkpoint supports generic declared machines and adapters; unsupported generic source forms remain explicit located refusals, not a substituted backend meaning.
- The completed native association separates selected World identity from implicit unit-world construction and retains it through suite queries/Runs, including empty starts. World-dependent action domains and contextual path patterns consume that same explicit binding in the native input checkpoint. Existing static Actor/manual registration and unit-world shorthand remain compatible.
- Source lifting currently takes source without live world values and represents starts as independent field products. R1/R3 require explicit world binding in every main/property/progress cfg and correlated whole-state computed starts. Backend counterexamples must retain action-input identity. Existing closure-law helpers and rejecting self-loops need independent admission; removing their laws would not make the activity Model conform.
- Runtime currently owns a private machine created by Runtime::new, and Realize::observe/answer receive no mutable world. R6 requires transactional live world admission from actual Start evidence, invalidation of old exploration evidence and publication rollback. Its out-of-domain Unsupported-plus-freeze behavior is specific to this admission; existing malformed/unsupported observation refusals preserve state.

## Resolved via Project Docs

- `model/README.md` already documents explicit `#[inputs]` support. DESIGN.md's unit-only readiness row is stale and remains a documentation obligation in R7. All seven R-IDs, the in-place activity requirement, prior evidence and dependencies remain. Only the three owner-approved R4/R5/R6 contracts above are amended.
- The owner delegated recommendations through the active milestone goal. This refinement asked zero questions and changed only Goal & Context plus these evidence/open-contract sections. Both ready state and all task statuses remain unchanged. The report `.flow/tmp/fn227-prep.md` supplies source locations and a proposed implementation split for later planning; it records no test execution.

## Engineering decomposition and review boundary

The three owner contract decisions are resolved. The intentional task set now has nine checkpoints. On 2026-10-21 the owner moved R5 and tasks .4, .9 and .10, with the cooperative-cancellation part of R6/.5, to fn-237-activity-timeouts-heartbeat-and; every remaining R-ID keeps an owner here. Independent input-source/fairness, phase/rejection admission and request capture are separately assigned before their consumers. The source/native bounds distinction is explicit: checked scalar construction and consumer-owned admission do not imply automatic native range validation. Current task briefs supersede earlier engineering notes that proposed such validation or still called the approved amendments pending.

Native World is complete, and the revised parent plan passed formal review. Independent source World is next under its task-scoped gates. Later input/activity/live tasks require their recorded dependencies and review/admission checks. No implementation may reconstruct request evidence, fabricate a Start cancellation field (that constraint now binds fn-237), weaken old laws, substitute native exploration for independent source semantics or count union fairness as exact-payload fairness.

## Plan review follow-up

The first formal plan review returned NEEDS_WORK on 2026-10-21, session 01a15ea6-c45d-7d01-b74c-ea7c7829f57e, actual gpt-6.1-sol/high. It reported six findings across R2-R6 and no maintainability findings. Field-level domain placement is consistent. Follow-up parallel scouts supplied source-grounded contracts for zero-or-one deadline facts, bounded optional ticks and whole-Runtime admission through Temporal Dispatch. The owner approved the three amendments, and the revised twelve-task decomposition assigns their support and consumer controls. Native task .1 completed independently; its SHIP implementation review is separate from the parent plan review.

Round two returned SHIP at 2026-10-21T16:24:48.428225Z on the same session, actual gpt-6.1-sol/high. The reviewer marked all six prior findings fixed, found no new blockers and reported no maintainability findings. Task prerequisites, scope and serial scheduling remain binding. Both review rounds are retained; the CLI's normal SHIP transition owns the cumulative counter reset, with no manual reset or force override.

The reviewer independently confirmed that the original literal witness was impossible because the old graph has enabled deadline actions and the new Model adds state/input spelling. The approved replacement is an executable semantic-refinement witness with the compatibility World specified in R4, a stated state/action/fact projection and exact comparison against the retained old lifecycle graph. Preserve the original graph/goldens as evidence. Hiding fields, keeping files without a proof, or removing old edges does not discharge the amended requirement. The retry checkpoint establishes its projection with the retained deadline behavior; the timeout checkpoint re-runs the proof with the final one-tick compatibility World. Since 2026-10-21 that timeout checkpoint is fn-237.3; this spec's R4 witness is the retry checkpoint's.

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
| R2 | `derive(Action)` admits variants with `Finite` fields and `#[domain]` world-dependent fields; `actions()` enumerates the domain product in declaration order; a shared fixture with a `bool` input and a world-sized range input has goldens and agrees with TLC. Errors: a non-`Finite` field without `#[domain]` is a compile error pinned by trybuild; a catalog above the limit is a `Refusal` naming the count. | fn-227-parameterized-models-declared-worlds.2, fn-227-parameterized-models-declared-worlds.8 | — |
| R3 | `umpire-lift` emits declared worlds as `CONSTANT`s with `.cfg` values, computed starts as the enumerated `Init` set and inputs as finite existential choice, and admits `impl Phased` and `reject!` arms, so the activity Model is no longer refused. Errors: an unsupported world field type is a located refusal; TLC disagreement on any fixture fails `UMPIRE_BACKENDS=require`. | fn-227-parameterized-models-declared-worlds.1, fn-227-parameterized-models-declared-worlds.11, fn-227-parameterized-models-declared-worlds.3, fn-227-parameterized-models-declared-worlds.7, fn-227-parameterized-models-declared-worlds.8 | — |
| R4 | The activity Model carries `maximum_attempts` and `RespondFailed { retryable }`: a retryable failure below the cap returns to `Scheduled` with `attempt + 1`, otherwise `Failed`. Preserve the old six-phase evidence and prove explicit lifecycle refinement against its graph using a stated state/action/fact projection and exact comparison of the projected behavior. This spec's compatibility witness is the checkpoint World with one attempt, the retained direct deadline actions and atomic cancellation; the final one-tick compatibility World moved to fn-237-activity-timeouts-heartbeat-and (2026-10-21). The default three-attempt World has new goldens; fn-218's activity cover test states its `Cases(n)`. Errors: a stale Poll is Rejected; terminal laws hold at every terminal phase/attempt; a changed old lifecycle edge must fail the refinement control. Old-byte equality is replaced only for this activity extension. | fn-227-parameterized-models-declared-worlds.3 | — |
| R6 | The realization binds a World from the actual filled Start request and successful same-invocation result, transported with optional `Observation.request` Any at tag 4. Admission covers `maximum_attempts`; request timeout fields are recorded with the Start receipt but not admitted as World parameters, and present values are not refused, until fn-237 admits them. Answer uses actual completion/failure evidence. The live SDK run proves retryable failure then completion through real Describe attempt observations; existing Complete, Fail, HostRefusal and context-Cancellation controls keep passing. Cooperative cancellation (the explicit pinned protocol binding and live held-attempt cancellation evidence) moved to fn-237-activity-timeouts-heartbeat-and (2026-10-21). Errors: out-of-domain options return field-named Unsupported and freeze admission; failed/crossed/missing evidence cannot install a World. Existing absent-request bytes, field numbers, Callback, Reply and core oracle protocol remain unchanged. | fn-227-parameterized-models-declared-worlds.12, fn-227-parameterized-models-declared-worlds.5 | — |
| R7 | DESIGN.md gains a "Parameterized worlds and inputs" section recording the chosen spellings, the activity world and state table, the native and TLC state counts per fixture world, and `make model-time` output; the readiness table rows become "delivered" with their final shape. No error surface beyond the gate. | fn-227-parameterized-models-declared-worlds.6 | — |

R5 is retired here: it moved to fn-237-activity-timeouts-heartbeat-and (R2 there) on 2026-10-21 and is covered by fn-237-activity-timeouts-heartbeat-and.1, .2 and .3.

===== [5/11] git_status: `git status --short --branch` =====
## umpire...origin/umpire [ahead 15]
?? .flow/artifacts/fn-227-parameterized-models-declared-worlds/task16-anchor.md

===== [6/11] git_log: `git log -5 --oneline` =====
d70684bca2 chore(flow): resume fn-227 at linear IR task
4c6b464b8b chore(flow): verify fn-238.9 and update milestone status
1e125ce95e chore(flow): task receipt fn-238-shared-front-end-one-model-reading-for.9
4348bade02 chore(flow): retain fn-238.9 implementation review receipt
11436f2f2b test(ir): retain constructor equivalence and required TLC gate evidence

===== [7/11] git_branch: `git rev-parse --abbrev-ref HEAD` =====
umpire

===== [8/11] memory_enabled: `flowctl config get memory.enabled --json` =====
{
  "success": true,
  "key": "memory.enabled",
  "value": true
}

===== [9/11] glossary: `flowctl glossary list --json --match "<task title + description>"` =====
(no glossary entry matches the task title or description - skipped)

===== [10/11] memory_index: `flowctl memory list` =====
bug/build-errors/
  track-schema-inputs-before-reusing-2026-09-07 — "Track schema inputs before reusing generated Lean modules" (module: model/lakefile.lean)
  glossary-renames-can-reintroduce-names-2026-09-13 — "Glossary renames can reintroduce names the vocabulary gate retired" (module: tools/umpire/internal/retiredvocabulary/check.go)
  ci-job-timeout-must-exceed-a-gates-go-2026-09-26 — "CI job timeout must exceed a gate's Go -timeout or output is lost" (module: .github/workflows/umpire.yml)
  slicescollectmapskeys-turns-empty-into-2026-09-27 — "slices.Collect(maps.Keys) turns empty into nil; cmp.Or[error] trips SA4023" (module: common/testing/testpilot/temporal/worker)
  package-only-model-headers-need-normal-2026-10-06 — "Package-only Model headers need normal compilation in every caller" (module: Makefile)
  isolate-scala-cli-author-builds-after-2026-10-16 — "Isolate Scala CLI author builds after local macro jar changes" (module: model/framework/README.md)
  match-enum-values-rather-than-2026-10-18 — "Match enum values rather than references in empty derives" (module: model2/umpire/dsl/macros/src/action.rs)
  resolve-model-impl-identity-in-its-2026-10-19 — "Resolve Model impl identity in its original lexical and trait context" (module: model/umpire/dsl/macros/src/machine.rs)
  preserve-generic-declaration-context-in-2026-10-21 — "Preserve generic declaration context in generated adapters" (module: model/umpire/dsl/macros/src/machine/declared.rs)

bug/data/
  joined-composition-keys-must-be-checked-2026-09-28 — "Joined composition keys must be checked injective before keying rows" (module: model/Umpire/Command/Compose.lean)

bug/integration/
  portable-execution-boundaries-must-2026-09-03 — "Portable execution boundaries must preserve runtime context" (module: tools/umpire/runner/runner.go)
  portable-model-plans-need-exact-2026-09-03 — "Portable model plans need exact artifacts and checked obligations" (module: model/Temporal/Tool/PortableEvaluationContract.lean)
  portable-schemas-must-preserve-source-2026-09-03 — "Portable schemas must preserve source semantic kinds and cardinalities" (module: proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto)
  behavior-neutral-refactors-must-not-2026-09-04 — "Behavior-neutral refactors must not strengthen validation" (module: model/Umpire/Observation/Evaluation/Admission.lean)
  contract-work-bounds-must-follow-typed-2026-09-04 — "Contract work bounds must follow typed values" (module: tools/umpire/verification)
  full-integration-gates-must-select-the-2026-09-04 — "Full integration gates must select the complete migrated suite" (module: Makefile)
  program-admission-must-validate-2026-09-04 — "Program admission must validate producer kinds and bound repeated lookups" (module: tools/umpire/internal/execution)
  coverage-findings-must-retain-2026-09-05 — "Coverage findings must retain independent obligations and evidence" (module: model/Umpire/Planning/CaseAnalysis.lean)
  default-empty-extensions-must-preserve-2026-09-05 — "Default-empty extensions must preserve canonical fingerprints" (module: model/Umpire/Property/Check.lean)
  joint-conflict-scopes-require-realized-2026-09-05 — "Joint conflict scopes require realized trigger identity" (module: model/Umpire/Planning/CaseAnalysis.lean)
  keep-raw-semantics-behind-checked-input-2026-09-05 — "Keep raw semantics behind checked input boundaries" (module: model/Umpire/Property/Evaluation.lean)
  nested-admission-diagnostics-must-2026-09-05 — "Nested admission diagnostics must retain exact source coordinates" (module: model/Umpire/Property/Check.lean)
  reconciliation-must-preserve-authority-2026-09-05 — "Reconciliation must preserve authority, IDs, and references" (module: .plans/UMPIRE4_SPEC.md)
  reusable-cases-need-run-coordinates-and-2026-09-05 — "Reusable Cases need run coordinates and event-aligned captures" (module: model/Temporal/CaseRuntime.lean)
  validate-protobuf-descriptor-structure-2026-09-05 — "Validate protobuf descriptor structure before accepting same-name responses" (module: tools/umpire/internal/ir/runtime_value.go)
  moved-conformance-tests-must-not-import-2026-09-06 — "Moved conformance tests must not import functional adapters" (module: common/testing/testpilot/conformance_test.go)
  check-unbounded-lean-numbers-before-2026-09-07 — "Check unbounded Lean numbers before protobuf narrowing" (module: model/Umpire/Case/Compiler.lean)
  dual-purpose-coverage-maps-need-per-2026-09-09 — "Dual-purpose coverage maps need per-entry admissibility" (module: model/Umpire/Observation/Projection/Coverage.lean)
  homogeneous-owner-index-blocked-mixed-2026-09-09 — "Homogeneous owner index blocked mixed-schema field Properties" (module: model/Umpire/Property/Evaluation.lean)
  keyed-capture-operands-must-be-checked-2026-09-09 — "Keyed capture operands must be checked against their declaration" (module: model/Umpire/Property/Check.lean)
  portable-carriers-must-keep-the-checked-2026-09-09 — "Portable carriers must keep the checked namespace and type rules" (module: model/Testpilot/Scoped.lean)
  consolidated-extractor-dropped-a-2026-09-27 — "Consolidated extractor dropped a caller's nonempty-field rejection" (module: common/testing/testpilot/temporal/driver.go)
  every-scenario-command-scenario-carries-2026-09-27 — "Every scenario-command Scenario carries ordering; pinned schedules decide it" (module: model/Umpire/Search/Product/Scenario.lean)
  test-output-parsers-must-accept-tlog-2026-09-27 — "Test output parsers must accept t.Log file:line decoration" (module: tools/umpire/cmd/umpire-repeat/signature.go)
  a-checked-artifact-cannot-supply-its-2026-09-28 — "A checked artifact cannot supply its own completeness universe" (module: model/Umpire/Command/ComposeProofs.lean)
  feature-modules-must-import-temporal-2026-09-28 — "Feature modules must import Temporal conventions or IDs change family" (module: model/Temporal/Case/Conventions.lean)
  channel-catalogs-and-visible-results-2026-09-30 — "Channel catalogs and visible results must survive source lifting" (module: model/scalav2/lifter/Lift.scala)
  realized-controls-constrain-admission-2026-09-30 — "Realized controls constrain admission evidence compatibility" (module: model/scalav2/specimens/activity.md)
  paired-level-validators-must-check-each-2026-10-06 — "Paired level validators must check each artifact independently" (module: model/irgen/src/main/scala/modelir/lint/Structure.scala)
  transition-table-reads-must-omit-2026-10-06 — "Transition-table reads must omit realization-only refinement metadata" (module: tools/umpire/model/compose.go)
  retired-protobuf-names-must-guard-every-2026-10-09 — "Retired protobuf names must guard every top-level declaration kind" (module: tools/umpire/ir/schema_test.go)
  schema-closure-tests-must-compile-2026-10-09 — "Schema closure tests must compile imported generated declarations" (module: model/check/test/Gate.test.scala)
  output-relocation-must-scope-shared-2026-10-11 — "Output relocation must scope shared readers and publication" (module: tools/umpire/cmd/umpire-gen-cases)
  preserve-explicit-replay-bindings-when-2026-10-13 — "Preserve explicit replay bindings when sharing a Model" (module: tools/umpire/check/checking.go)
  encode-fixture-folder-names-before-2026-10-18 — "Encode fixture folder names before generating Rust identifiers" (module: model2/umpire/testing/src/discovery.rs)
  separate-archived-scan-roots-from-2026-10-18 — "Separate archived scan roots from historical source paths" (module: tools/umpire/ir)
  use-file-snapshots-for-reviewable-2026-10-18 — "Use file snapshots for reviewable external insta goldens" (module: model2/framework/samples/tests/kettle.rs)
  match-complete-macro-invocations-when-2026-10-19 — "Match complete macro invocations when selecting machine outcomes" (module: model/umpire/dsl/macros/src/machine.rs)
  preserve-rust-expression-and-inference-2026-10-20 — "Preserve Rust expression and inference contracts in path macros" (module: model/umpire/dsl/macros/src/path.rs)
  prove-result-shape-before-propagating-2026-10-21 — "Prove result shape before propagating generated-enum coverage" (module: model/temporal/annotate/syntax/src/realize.rs)
  required-numeric-generation-must-sample-2026-10-21 — "Required numeric generation must sample present candidates" (module: model/umpire/annotations/src/value.rs)

bug/performance/
  authoring-comparisons-must-force-equal-2026-09-06 — "Authoring comparisons must force equal checker work" (module: model/Temporal/Feature/Nexus2/AuthoringTests.lean)
  canonical-identity-embeds-whole-2026-09-09 — "Canonical identity embeds whole generated schema (27MB per payload)" (module: model/Umpire/Operation/Canonical.lean)
  build-counterexample-paths-only-for-2026-10-18 — "Build counterexample paths only for failing predicates" (module: model2/framework/checker/src/explore.rs)

bug/runtime-errors/
  interface-nil-checks-must-cover-every-2026-09-04 — "Interface nil checks must cover every nil-capable kind" (module: tools/umpire/temporal/nexus/runner.go)
  monitor-closure-must-honor-cancellation-2026-09-04 — "Monitor closure must honor cancellation without rebuilding proofs" (module: tools/umpire/verification)
  freeze-contract-transitions-when-2026-09-05 — "Freeze Contract transitions when execution becomes incomplete" (module: tools/umpire/verification)
  retain-opaque-completion-authority-2026-09-05 — "Retain opaque completion authority across rejected acceptance" (module: tools/umpire/temporal/server)
  workflow-replay-must-complete-2026-09-05 — "Workflow replay must complete unfinished admissions once" (module: tools/umpire/temporal/worker)
  replay-rejection-lost-to-2026-09-27 — "Replay rejection lost to finalizePlanning's unsatisfiable-Scenario priority" (module: model/Umpire/Search.lean)
  field-alterations-must-range-over-2026-09-28 — "Field alterations must range over structure fields, not the reachable catalog" (module: model/Umpire/Command/Predicate.lean)
  finite-ir-admission-must-count-work-2026-09-30 — "Finite IR admission must count work before listing catalogs" (module: model/scalav2/goir)
  finite-search-requires-injective-2026-09-30 — "Finite search requires injective identity and bounded admission" (module: model/go/umpire)
  reconstruct-crash-resets-and-budget-2026-10-16 — "Reconstruct crash resets and budget updates atomically" (module: model/framework/dsl/StateField.scala)
  reject-async-methods-before-discarding-2026-10-19 — "Reject async methods before discarding generated call results" (module: model/umpire/dsl/macros/src/machine.rs)

bug/test-failures/
  differentials-must-enumerate-every-2026-09-28 — "Differentials must enumerate every search caller, not only declarations" (module: model/Umpire/Search/Tests/Differential.lean)

knowledge/decisions/
  host-verified-model-provenance-without-2026-09-03 — "Host-verified model provenance without protocol signatures" (module: Umpire PortableTestPlan provenance)

===== [11/11] dependencies: ids, titles, statuses, done summaries =====
- fn-227-parameterized-models-declared-worlds.15 [done] - Compact emitted TLA+ for conditional returns before activity retries
    Emitted TLA+ grows linearly with sequential conditional returns: outcomes are one chain of the rejections a step reaches, updates after a return are guarded where recorded (`running /\ cond` for late conditions), equal arms fold only when the condition has no checked arithmetic or indexing, and subexpressions over 400 characters repeated within a step are emitted once as a `LET` with allocated `__umpire_s` names. Sizes: `phased_rejection` Job.tla 18,040 -> 12,799 bytes (its Drop arm 7,024 -> 1,593); probe 1-6 returns 2,356 -> 706,669,127 bytes before, 1,582 -> 5,789 after. TLC verdicts, state counts and replay unchanged; new `shared_returns` fixture runs shared LETs through TLC for unit and payload actions (18 states agree, counterexample replays through shared edges) and pins the late-condition guard; `phased_rejection` gains a payload counterexample. L1/L3/L4/L5/L6 resolved. A front-end node budget of 10,000 refuses steps whose admitted tree would exhaust rustc memory (measured about 24k-64k nodes OOM); the admitted tree still grows per return, filed as fn-227.16 (named IR intermediates) before .3.
    
    Reviews: NEEDS_WORK (LET path untested in TLC; unguarded late conditions), then NEEDS_WORK (budget too high), fixed by the conductor in 3cb0412d8f (.flow/tmp/fn22715-review.md, fn22715-review2.md). Gates: Full at bf7936c4b0 (`UMPIRE_BACKENDS=require make model-check` 318s, model-time 52.955s/7.827s, slice 7s, gaps 366s); at c3dc7d72e0 model-check 273s, model-time 46.298s/6.618s; at 3cb0412d8f Quick default/required, model-check 2,868s (cold, shared machine), model-time 45.288s gate / 6.080s inner loop. Guest unchanged throughout.
    
    stage: plan-sync - skipped(config: planSync.enabled != true)

