---
satisfies: [R3, R5]
---
# fn-227-parameterized-models-declared-worlds.4 After .3: timeouts, Heartbeat, Tick and asynchronous cancel on the activity Model with progress and bound claims

## Description
Add the four Option<Ticks> timeouts and asynchronous_cancel to the activity World, per-deadline budgets and cancel_requested to the state, Worker::Heartbeat and Deadline::Tick (decrements live budgets, expires the first exhausted one with the expired deadline as a fact; disabled with no live budget). Schedule-to-close spans attempts; start-to-close and heartbeat budgets reset on Poll. Asynchronous cancel sets cancel_requested in Started and the next worker response or heartbeat closes as Canceled. Claims: a started activity under a fair worker eventually closes; schedule-to-close bounds total ticks. Verified natively on the default world and by TLC on the larger worlds under UMPIRE_BACKENDS=require; native default world stays inside R30.

**Admission gate:** R4's literal compatibility witness and R5's cancellation contract require the pending owner decisions. The original acceptance remains in force. Do not implement this task or alter activity expectations until those decisions are recorded and the plan review passes. The engineering support below does not authorize a cancellation correction or compatibility projection.

**Size:** L until the conductor splits the bounded timeout/fact support checkpoint from the consuming activity rewrite. The support must pass native and independent TLC controls before activity consumption. This task is not execution-ready merely because the support contract is now specified.
**Files:** `model/umpire/dsl/macros/src/machine.rs`; `model/umpire/lift/src/{lib,tree,shape,expr,emit,build,names}.rs` and `body/{typer,expr,effects,claim,progress,vector}.rs` as typed-expression consumers; `model/umpire/backends/tlc/src/{module,decode}.rs`; `model/umpire/testing/src/backend.rs`; new macro/source/shared fixture controls; `model/temporal/feature/activity/{model.rs,tests.rs,expected/**}`; `model/README.md`.
**Touches:** [model/umpire/dsl/**, model/umpire/lift/**, model/umpire/backends/tlc/**, model/umpire/testing/**, model/umpire/tests/**, model/temporal/feature/activity/**, model/README.md, model/temporal/hosts/go/umpire_temporal_guest.wasm]

## Approach

### Bounded zero-or-one facts

Admit `Option<F>` as an opt-in handler result, where F is one local nongeneric nonempty fieldless enum with derived Debug. Unit handlers retain their exact expansion and rejection/stay behavior. No Fact derive, fact catalog, TLS recorder or new envelope is needed. The additional declaration shape is `#[on(Deadline)] fn deadline(&mut self, action: Deadline, w: &World) -> Option<ExpiredDeadline>`.

Resolve one fact carrier consistently across the module. Refuse payload/discriminant carriers, unsupported aliases, nested options, vectors, arbitrary return types and inconsistent carriers. The bounded source result grammar is None, Some(F::Variant), or exhaustive tail if/else over those leaves, after admitted effects. Arbitrary return statements, result-producing helpers and stored optional fact results remain located source refusals. Unit effect helpers remain unit. A direct reject! in the optional-result handler fails return typing; existing Client/Worker rejection paths stay unit-returning.

Capture the result inside the existing unit outcome/stay evaluation closure, then construct zero or one fact on an emitted edge. Preserve enabledness: returning Some without state change/rejection/stay does not enable a step; None does not suppress a changed-state step. Every ordinary activity edge writes an empty fact list. A timeout transition writes exactly its authored expired-deadline enum value.

Extend private source Branch metadata with source-derived facts and select the existing Seq<Enum> observer type for factful machines. Lower fact selection in the same conditional SSA environment as its state update. Preserve authored first-matching priority when deadlines expire together; the activity defines and tests that order explicitly. Empty ordinary edges must not retain prior facts. Preserve the exact no-fact source/TLA path and Seq<Unit> metadata for existing fixtures. Reuse typed enum decoding and Debug rendering for replay; native rows never supply emitted fact meaning.

### Bounded optional ticks

Reuse .2's single authored range-newtype declaration and .7's typed selected-World binding. Configured Ticks use the small positive domain 1..=3; Remaining uses 0..=3 for countdown state. None means absent/unarmed. Some(Ticks(0)) is invalid, never absent or an implicit immediate timeout. Some(Remaining(0)) is exhausted and remains distinct from None. A valid one-tick budget expires on the Tick that decrements it to zero, without unsigned underflow or an armed nonterminal zero deadlock. Use explicit checked native construction/admission; Finite enumeration alone does not prevent forged out-of-domain Rust values.

Keep nominal newtype identity and bounds in source/backend metadata. Encode optional bounded numbers as a scalar ordinal: None is 0 and Some(n) is n+1. Configured Ticks admit exactly {0,2,3,4}; code 1 must refuse. Remaining admits {0,1,2,3,4}. Validate backing widths, range bounds and encoding overflow. Do not erase the type to an ordinary Int range, admit arbitrary ADTs, or let encoded integers replace authored None/Some/newtype spelling in reports.

Admit canonical prelude/core/std Option paths for these source-local bounded newtypes. Support typed None/Some, copying, equality/inequality, is_none/is_some, eager unwrap_or and numeric payload projection. Preserve eager fallback errors even when Some is present. Conditional assignments reuse existing SSA lowering and distinct World/state/before/source-constant identities. Refuse shadowing, unsupported aliases/inner types, nested options, arbitrary Option match/if-let/combinators, unwrap/expect and mutable reference operations with located diagnostics. Do not weaken existing Vector get/unwrap_or controls. If the final Model needs a wider form, name and test that bounded admission before using it.

The same validated binding and ordinal assignments reach every main/invariant/postcondition/negative/progress CFG. Typed decoding validates exact code sets before reconstructing standard Debug spelling and replaying under the selected native World. A legal code from another selected World cannot certify agreement. No native reached graph may define Next or claims.

### Activity consumption after support and owner gates

Author four deadline facts, explicit simultaneous-expiry priority, reset/spanning laws, heartbeat/Tick enabledness and the approved cancellation contract in the existing activity Model. Preserve every law and independently source-lift the actual authored behavior. State the fair-worker assumption and larger/default selected Worlds. A tick-count safety claim must not become true merely by clamping a counter or cutting a domain. Retain counterexample controls that detect an actual reset/expiry/bound defect. Keep the original R4 evidence unchanged until its owner-approved compatibility contract is executable.

## Investigation targets

**Required**:
- `model/umpire/dsl/macros/src/machine.rs:320` - handler return admission, selected Fact and reject/stay adapters.
- `model/umpire/lift/src/body/effects.rs:160` - conditional effects and branch SSA for fact results.
- `model/umpire/lift/src/body/typer.rs:10` - bounded newtype/option admission; follow expression and Vector method consumers.
- `model/umpire/lift/src/emit.rs:123` - facts, typed defaults, priming and all CFG paths.
- `model/umpire/backends/tlc/src/decode.rs:105` - exact typed value admission, wrapper rendering and replay consumers.
**Optional**:
- `model/umpire/lift/tests/vector.rs:20` - real native/TLC and eager-fallback controls.
- `model/umpire/testing/src/backend.rs:170` - existing fact identity comparison and observer controls.

## Verification

First reproduce RED for unsupported fact-return handlers and Option<Ticks> source; then retain actual native/source/TLC GREEN evidence. Fact controls cover all four values, empty ordinary facts, disabled Some, changed None, stay/reject compatibility, unsupported return shapes, SSA guards and source-only fact swaps. Required TLC traces must cross each expiry and replay exact facts. Mutating a fact to another valid enum, omitting/duplicating it, or adding one to an ordinary edge must fail replay; unknown enums must fail decoding.

Option controls distinguish None/Some(1)/Some(3), forbidden Some(0)/Some(4), countdown zero, mismatched newtypes, invalid updates and eager fallback failure. Exercise before/World/source name collisions, complete correlated starts, every CFG and false-claim replay. Forged codes -1/1/5, wrong value types and missing/extra/duplicate fields must refuse under the relevant exact domain before lookup. Preserve old unit expansion/golden bytes in the support checkpoint, including metadata.

Quick from model/: scoped `mise exec -- cargo test -p umpire-dsl-macros -p umpire-lift -p umpire-tlc -p umpire-testing -p umpire-tests`, with actual admitted-fixture controls repeated under UMPIRE_BACKENDS=require, plus formatting and strict clippy. Full from repository root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. Retain actual counts, replay traces and timings, including soft-target misses. Rebuild changed linked guest bytes with the existing builder and rerun embedded slice/gap/privacy controls; no ABI, panic, dependency or core protocol change.

## Acceptance
- [ ] Timeouts, Heartbeat, Tick and asynchronous cancel modeled; world with no timeouts has no Tick edge
- [ ] Progress and bound claims verified natively and by TLC
- [ ] Goldens, cover test and controls updated; R30 budgets hold
- [ ] The prerequisite support checkpoint proves typed zero-or-one facts and bounded option/newtype admission, all-CFG emission, native Debug identity, exact decode/refusal and actual counterexample replay. All negative controls above remain discriminating; no native graph supplies source behavior. Split/admission and pending owner gates are satisfied before the activity rewrite.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
