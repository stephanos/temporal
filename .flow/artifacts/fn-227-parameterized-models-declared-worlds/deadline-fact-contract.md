# fn227 expired-deadline fact contract

Recommend an opt-in `Option<ExpiredDeadline>` result on the deadline handler. Each enabled expiration edge carries one fieldless enum value; every other activity edge carries an empty fact vector. The source lifter translates that authored result independently to a zero-or-one sequence. Existing native rendering, typed TLC decoding and replay already support that representation.

This is a planning recommendation against the supplied source baseline `af0ebafc57`. No source, task, requirement or golden was changed, and no build, test, TLC or live run was executed. The sole output is this report. The supplied judge result is `no_key`; retain the requested thinking-scout tier `gpt-6.1-sol/high`, with no retry. The owner decisions about R4's impossible literal byte witness and R5's stronger cancellation semantics remain pending. The checkpoint states both decisions exceed implementation authority and admits no implementation (`.flow/artifacts/fn-227-parameterized-models-declared-worlds/planning-checkpoint.md:24`).

## Existing support and missing contract

`Machine` already associates a model-owned `Fact: Debug`, and `Step<S,O,F>` already contains `Vec<F>` (`model/umpire/dsl/src/machine.rs:18`, `model/umpire/dsl/src/machine.rs:103`). The generated adapter is the gap. It refuses every explicit handler return type, fixes `Fact = ()`, fixes the opaque world return constraint to unit facts, and creates empty fact vectors in all four reject/stay dispatch combinations (`model/umpire/dsl/macros/src/machine.rs:320`, `model/umpire/dsl/macros/src/machine.rs:717`, `model/umpire/dsl/macros/src/machine.rs:826`, `model/umpire/dsl/macros/src/machine.rs:905`).

The source tree's `Branch` has outcome and enabled-self-loop information but no facts (`model/umpire/lift/src/tree.rs:41`). Its effect pass already carries SSA state through conditionals and inlines parameterless unit `#[effect]` helpers (`model/umpire/lift/src/body/effects.rs:160`, `model/umpire/lift/src/body/effects.rs:188`). The emitter assigns an empty fact sequence to every real edge and declares `Seq<Unit>` observer metadata (`model/umpire/lift/src/emit.rs:123`, `model/umpire/lift/src/emit.rs:324`).

The lower layers need no new fact serialization. The lift type system already has fieldless enums and sequences (`model/umpire/lift/src/lib.rs:35`, `model/umpire/lift/src/lib.rs:48`); expression IR already has conditional and sequence expressions (`model/umpire/lift/src/expr.rs:18`). TLC schema conversion already accepts `Seq<Enum>` and validates enum declarations (`model/umpire/backends/tlc/src/module.rs:51`). Decoder admission rejects undeclared enum values (`model/umpire/backends/tlc/src/decode.rs:105`). Replay formats enum variants using the native derived-Debug spelling and compares the complete fact list (`model/umpire/testing/src/backend.rs:16`, `model/umpire/testing/src/backend.rs:170`). Native row rendering already prints `facts={:?}` (`model/umpire/testing/src/render.rs:30`).

## Alternatives

| Alternative | Exact authoring shape | Tradeoff |
| --- | --- | --- |
| A. Optional handler result, recommended | `#[on(Deadline)] fn deadline(&mut self, action: Deadline, w: &World) -> Option<ExpiredDeadline>;` | Uses Rust's return typing, fixes multiplicity at zero or one and needs no new recording API. Unit-returning handlers and their rejection macros retain their path. The lifter needs a narrow tail-result grammar. |
| B. Explicit optional output | `#[on(Deadline)] fn deadline(&mut self, action: Deadline, fact: &mut Option<ExpiredDeadline>, w: &World);` | Keeps every handler unit-returning and works with existing direct `reject!` returns. Adds parameter-position rules beside trailing World recognition, mutable output assignment/escape rules and last-assignment semantics. These costs serve no activity requirement. |
| C. Lexical fact marker | `#[fact] enum ExpiredDeadline { ... }` plus `fact!(ExpiredDeadline::Heartbeat)` in a unit handler | Keeps inline statement style. Requires a new marker, macro context validation, a generated lexical sink and explicit helper propagation. A TLS-backed generic collector would widen the existing outcome frame into a new recording framework. Reject this alternative for this slice. |

Alternative A does not require `#[fact]`, a `Fact` derive, a separate fact catalog or a new result wrapper. The return type selects the one local enum used as the machine's fact type. Framework code recognizes a fieldless enum and an optional result; only the activity Model names deadlines.

## Proposed bounded authoring contract

These are declaration shapes, not implementation bodies. The optional World parameter remains trailing as the epic already specifies (`.flow/specs/fn-227-parameterized-models-declared-worlds.md:22`).

```rust
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum ExpiredDeadline {
    ScheduleToStart,
    StartToClose,
    ScheduleToClose,
    Heartbeat,
}

#[on(Client)]
fn client(&mut self, action: Client, w: &World);

#[on(Worker)]
fn worker(&mut self, action: Worker, w: &World);

#[on(Deadline)]
fn deadline(&mut self, action: Deadline, w: &World) -> Option<ExpiredDeadline>;
```

1. Preserve the existing default-unit handler form. Admit exactly `Option<F>` as the additional result form, where `F` resolves to one local, nongeneric, nonempty fieldless enum with derived `Debug`. Reject payload variants, explicit discriminants, type aliases, references, `Vec<F>`, `Result`, arbitrary return types and inconsistent fact enum types across fact-returning handlers. Native Rust may support wider enum declarations elsewhere; these restrictions apply to the new fact carrier. The current source enum admission already has the fieldless/discriminant boundary (`model/umpire/lift/src/shape.rs:103`).
2. A fact-returning handler still matches its declared action. Each arm ends in `None`, `Some(F::Variant)`, or an exhaustive tail `if/else` tree with those leaves. Effects and typed local/guard expressions can precede the tail. Source admission initially refuses arbitrary `return`, result-producing helper calls and stored optional fact values. It reports the offending source location. This is a dedicated zero-or-one step-fact result grammar and requires no general Option type, methods or serialization support. The separate optional-tick contract remains necessary for World/state budgets.
3. Only the deadline handler needs the result form. Unit `#[effect]` helpers retain their current surface and cannot return or record a fact. Keep `reject!`'s existing unit-return behavior (`model/umpire/dsl/src/outcome.rs:125`). A direct `reject!` in an optional-result handler fails Rust return typing; Client/Worker rejection handling stays unit-returning. Existing rejection outcome and reject-after-mutation rules remain authoritative, including when a helper records rejection. Add no new outcome type or fact-specific rejection precedence.
4. The generated dispatch captures the optional result in an ordinary local while it calls the handler through the existing outcome/stay evaluation closure. The closure itself still returns unit. Convert `None` to `Vec::new()` and `Some(f)` to one element on an emitted edge. This requires no change to the signatures of `outcome::evaluate` or `evaluate_staying` (`model/umpire/dsl/src/outcome.rs:95`). No facts enter the existing TLS frame.
5. Keep enabledness unchanged. State change, existing rejection, or `stay!` enables an edge; returning `Some` alone does not. A disabled handler produces no Step. `None` does not suppress a changed-state edge. A supported explicit stay edge may carry a result, although the activity Model must emit its fact only when it transitions to `TimedOut`. Ordinary activity transitions, nonexpiring Tick edges, retries, heartbeat resets, cancellation and rejecting self-loops have `[]`.
6. All activity expiry leaves change phase to `TimedOut` and return the corresponding `Some(ExpiredDeadline::...)`. The authored Tick's complete guard order decides which fact wins when two budgets expire together. Record that order in the Model and exercise it with an applicable simultaneous-expiry control. Neither this report nor the framework chooses the priority. The source pass must preserve the authored first-matching guard order, as its current branch pass already does (`model/umpire/lift/src/body/effects.rs:37`).

## Independent source and identity

Add `fact_type: Type` to the private `Lifted` tree and a typed sequence expression to each `Branch`. The absent-fact path uses unit and an empty sequence. A factful module uses the existing `Type::Enum` and `Type::Seq`; its branch result is either the empty sequence or a singleton enum expression. Lower a tail conditional with the same SSA environment and condition used for the arm's state update. Reuse conditional/sequence expression nodes and qualify variants with the authored enum name. This keeps state update and fact selection on the same authored branch without duplicating timeout policy in an adapter.

Emission initializes `__umpire_facts` to `<<>>`, assigns the branch's source-derived fact expression on every real edge, and declares `__umpire_facts` as `Seq<Enum(ExpiredDeadline)>`. Ordinary edges explicitly write `<<>>`, so an earlier expiry fact cannot leak into a later edge. Observer-only progress activation continues preserving the last edge's facts exactly, as it does now (`model/umpire/lift/src/emit.rs:239`); it is not an ordinary Model edge.

The enum's source identity is, for example, `ExpiredDeadline::Heartbeat`. Native derived Debug and native rows spell the element `Heartbeat`; TLC carries the qualified string in `<<"ExpiredDeadline::Heartbeat">>`. The existing typed replay conversion produces `[Heartbeat]` and compares it against native `Vec<ExpiredDeadline>` Debug. Keep that established enum bridge; do not parse Debug to construct TLA behavior. No native exploration, fact vector or recorded row may supply the emitter's branch fact expression. Backend replay remains a comparison consumer.

When no handler returns a fact, generate the exact old macro tokens and retain the old `Seq<Unit>` observer metadata and TLA strings. Select a separate opt-in expansion branch. Changing `Machine::Fact` in a new factful activity is intentional; changing old Kettle or rejecting/staying expansion snapshots is a regression. The opaque machine return constraint must use the selected `F` only for the opt-in path.

## Files and Touches

| Surface | Files or owned tree | Change |
| --- | --- | --- |
| Native macro | `model/umpire/dsl/macros/src/machine.rs` | Classify optional result, validate carrier, select Fact and opaque bound, capture handler result in all existing reject/stay combinations. Preserve old expansion path. |
| Source admission | `model/umpire/lift/src/shape.rs` | Resolve the fact-return signature and its enum independently from source; preserve trailing World handling. |
| Private IR | `model/umpire/lift/src/tree.rs` | Add fact type and branch fact expression. |
| Source effects | `model/umpire/lift/src/body/effects.rs` | Lower narrow tail results together with effects and existing conditional SSA state. |
| TLA | `model/umpire/lift/src/emit.rs` | Emit branch facts and typed observer metadata. |
| Compile/expansion controls | `model/umpire/dsl/macros/tests/{ui,pass,fixtures}/` | New optional-result success/refusal and expansion controls. Existing fixture runner already discovers these (`model/umpire/dsl/macros/tests/fixtures.rs:29`). |
| Located refusals | `model/umpire/lift/tests/source.rs` and owned expected fixtures | Unsupported result grammar, wrong carrier and source-only mutation controls. |
| Native/TLC fixture | `model/umpire/tests/fixtures/<new-generic-fact-fixture>/` | Model, test hook, native rows/paths and emitted module/config/TLC agreement goldens. Framework fixture names use generic A/B/C/D fact values. |
| Replay controls | `model/umpire/testing/src/backend.rs` tests | Matching typed facts and forged/missing/duplicate facts on real-edge replay; retain observer-only projection controls. |
| Activity | `model/temporal/feature/activity/{model.rs,tests.rs,expected/}` | Four fact values, authored priority, expiry and empty-fact assertions. Activity changes still wait for owner decisions. |
| Documentation | `model/README.md`, later `model/DESIGN.md` | State the bounded handler result surface and measured final evidence. |

`Machine`, `Steps`, `StepError`, `Outcome`, protocol schemas, envelopes, Go hosts, TLC parser production code and rendering production code need no interface change for this recommendation. If a control demonstrates a gap there, name it before widening this contract.

## Required controls and completion evidence

- Compile success covers a public and private machine, declared and unit World forms, unit handlers mixed with one fact handler, and a module containing ordinary rejection and stay handlers. Compile failures cover a non-Option handler result, wrong/nested/payload fact carrier, mixed fact types and direct `reject!` in the optional-result handler. Pin located source refusals for arbitrary return statements, result-producing helpers, unrecognized constructors and unsupported fact aliases.
- Native fixture controls cover all four singleton values, ordinary changed edges with empty facts, `None` on a changed edge, disabled unchanged handlers, an unchanged `Some` without stay remaining disabled, a stay edge, and rejection/invalid-mutation behavior. Retain all existing reject/stay controls. Check the no-fact expansion snapshots byte-for-byte.
- Source controls inspect branch facts and enum observer schema. A source-only swap of A and B at one expiry must change emitted TLA facts even when native exploration/rows are untouched. Ordinary branches after a factful branch must still assign an empty sequence. A guarded helper/state update before the result must not make the fact guard use a different pre/post SSA value.
- Actual required TLC execution must replay a deliberately refuted claim whose path crosses each expiry. Four forced expiry traces make all four values observable to replay; passing safety claims and complete projected state sets alone do not exercise this comparison. Add ordinary-edge and simultaneous-expiry traces. On the same exact action/choice/state edge, replace a valid fact with another declared fact, omit it, duplicate it, or add a fact to an ordinary edge; each mutated replay must fail. Unknown enum strings must fail typed decoding. Retain the existing observer-only edge-field refusal (`model/umpire/testing/src/backend.rs:390`).
- Red/green means the new authored fixture is first refused by the current handler-return check and source emitter, then passes with the new surface; a wrong fact remains red in replay. Record actual native projected state/row counts, raw TLC state counts where observers distinguish states, replayed traces and R30 timing after implementation. This report supplies none of those measured results.
- Focused proposed commands are `cd model && mise exec -- cargo test -p umpire-dsl-macros`, `cd model && mise exec -- cargo test -p umpire-lift`, `cd model && mise exec -- cargo test -p umpire-testing`, and `cd model && UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-tests --test fixtures`. Finish with `UMPIRE_BACKENDS=require make model-check` and the already-required `make model-time` from the repository root. These are future checks, not executed evidence.

## Task size

Fact support alone spans five existing production files across the macro and independent lifter, plus four test owners covering compile/expansion, source refusals, shared native/TLC fixtures and replay. It adds no checker or wire format. Current affected production files range from 78 to 938 lines, and the existing replay test owner is 426 lines; these are inspected file sizes, not an implementation line estimate.

Task .4 also owns the activity rewrite, four optional budgets, budget resets across attempts, heartbeat, Tick, cancellation, fairness/progress and schedule-to-close bounds (`.flow/tasks/fn-227-parameterized-models-declared-worlds.4.md:7`). Finding 5 independently adds optional-tick source admission. Keeping all of that plus this five-file support seam in one task makes its acceptance boundary difficult to isolate. Recommend a bounded prerequisite support task delivering this generic fact fixture through actual TLC replay before activity .4 consumes it. Retain .4's R5 ownership and all owner gates; the conductor can assign the new prerequisite after the engineering contracts are accepted. This report creates or renumbers no task.
