# Symbolic back end: Models as BTOR2 transition systems checked by rIC3, witnesses replayed natively (model)

## Conversation Evidence

> user (2026-10-23): "for model/ umpire, is apalache actually th best apprpach for a SMT? or can we use a native rust option"
> assistant: Apalache is the best SMT route for TLA+, not for Umpire; a direct word-level encoding from the shared admitted tree skips the TLA+ detour; recommended a spike.
> user (2026-10-23): "run a one-task spike that emits Kettle from the shared admitted structure as BTOR2 (and optionally SMT-LIB BMC), checks it with rIC3 or a pinned bitwuzla, and replays the counterexample through checked_step. approved spike"
> user (2026-10-23): "if they work, is Apalache redundant?"
> assistant: for the word-level subset the front end admits today, yes; Apalache keeps one asset, symbolic sets, functions and unbounded sequences, which no Model uses yet.
> user (2026-10-23): "let's file a flow spec to make the prototype real ie as a real backend"

## Goal & Context

The 2026-10-23 spike (`model/design/native-symbolic-backend.md`, probe kept at `model/design/native-symbolic/`) emitted Kettle from the embedded admitted tree (`umpire_ir::Lifted`, what the shared front end of fn-238 produces) as a BTOR2 word-level transition system, checked it with rIC3, and replayed every witness through `checked_step`. All five Kettle properties agreed with the native checker in 0.10 s each: unbounded IC3 proofs for the three safe claims and the arithmetic monitor, and a 62-step `never_beeps` counterexample replayed and natively confirmed. Two controls that mutate only the admitted tree were flagged. On a scaled Kettle (boiling at 1 000 000, two million states) the native checker took 19 s and IC3 0.10 s per proof, so proof time did not move with the state space. The owner's standing concern is scaling (the Model expands dramatically soon) and eventual Quint parity.

This spec turns the probe into a real back end beside `umpire/backends/tlc` and `umpire/backends/apalache`: same shared front end, same fixture runner, same rule that every back-end answer is confirmed by the native checker. It gives what fn-211 was filed for (bounded and unbounded symbolic verdicts over large state spaces, symbolic Worlds) without TLA+ emission, Snowcat annotations, ITF decoding or a JVM. fn-211 stays deferred until set-valued state (symbolic sets, functions, unbounded sequences) is admitted; this spec does not close it.

## Architecture & Data Models

- **Crate** `umpire-symbolic` at `model/umpire/backends/symbolic/`, workspace member, same manifest conventions as `umpire-tlc`. It depends on `umpire-ir` (and `umpire-dsl` for `Lifts`/`Machine`), never on the lifter: the emitter reads the embedded admitted tree of the compiled World, not source.
- **Emitter** (ported from the probe's `btor2.rs`): one BTOR2 file per Model and property. State fields become states with widths from the field's Rust type; enums are small unsigned words with a domain constraint; World fields are free states with `next = self` (symbolic constants) or `init` to the selected World's values (concrete mode). Branches keep first-match semantics; an action is enabled when its taken branch changes state or preserves a self-loop; a free `act` input with an extra "no step" value stutters. Postconditions use one-step history (`prev_*`, `prev_act`, `has_step`) and their `after =` filters. Checked arithmetic (fn-210 R11: overflow, division and remainder by zero, signed MIN/-1) latches a `fault` state, with overflow detected by widening because rIC3's logic library has no `saddo`-family ops; the fault is its own property, distinct from any authored claim. Exactly one `bad` per property; property metadata distinguishes invariant, postcondition, arithmetic and negative control.
- **Engines**: rIC3 pinned as a built binary (it is a Rust crate with C++ SAT solvers; the build needs cmake and a clang or a shim), admitted into `model/build/ric3` by a script shaped like `model/scripts/fetch-apalache.py` (pinned version and source checksum, private staging, atomic publish under a lock, symlink refusal). IC3 (`ric3 check <file> --cex --ui=false ic3`) for proofs; BMC to the native BFS depth plus one for violations when IC3 times out or is asked to; `kind` optional. Verdicts: `Safe` (unbounded proof), `Unsafe { witness }`, `Bounded { depth }` (no violation to the bound, never a proof), `Timeout`, `NotRun { reason }`.
- **Witness decoding and replay**: BTOR2 witnesses key assignments by state and input declaration index, so the emitter records a layout (field name, index, width, signedness, `Type`). Frames decode into typed `Value`s through the existing `Values`/`Value` contract; the start is found in `world.starts()`, each step through `checked_step`, the successor matched by typed values, and the violated claim re-judged natively (invariant predicate, `StepClaim::holds`, arithmetic refusal). Reuse fn-211.3's `Replay`/`Confirmation`/`Replayed` types from `umpire-apalache` by moving them to a shared place rather than copying. A non-replayable witness is witness-rejected and fails the fixture.
- **Fixture consumer**: `expected/symbolic.txt` per fixture (property, kind, engine, verdict, bound where bounded, `agrees` or the disagreement); violations also write `expected/symbolic/paths/<property>.txt`. Selected by the presence of the golden, run only under `UMPIRE_BACKENDS=require`, same ownership registration as the TLC consumer. Timings never appear in goldens.
- **Symbolic Worlds** (fn-211 R6, moved here): a fixture opting in leaves World fields free within their declared finite domains, so one run per property covers every World; a violation decodes the World values, constructs the World natively and replays under it. Computed starts (`#[starts]` bodies) need either the front end to admit the start bodies or a native start table over the World domain; the task plan chooses and records why.
- **Optional SMT-LIB path**: the same node graph unrolled as QF_BV SMT-LIB piped to a solver (the probe's `smt.rs`) stays a development tool for timing comparisons and is not a consumer.

## API Contracts

- `Symbolic::emit(world: &W, property, mode) -> Result<System, Refusal>` where `System` carries the BTOR2 text and the `Layout`; `W: Machine + Lifts`.
- `Ric3::check(system, engine, bound, timeout) -> Result<Verdict, Refusal>` with the verdict enum above; `Refusal` is a tool admission, execution or decoding failure, never a property verdict, as in `umpire-tlc`.
- `replay(world, layout, frames, property) -> Replayed` with a `Confirmation` of the violated claim.
- `symbolic.txt` lists properties in declaration order; `make model-update` rewrites it like any golden; the default `make model-check` neither downloads nor builds rIC3.

## Edge Cases & Constraints

- Unbounded `Safe` is a proof of the encoding, not of the Model: agreement with the native checker on every selected fixture is what makes it evidence. `Bounded` says nothing beyond the bound and the golden states the bound.
- Integer widths come from the Rust type carried on `Type::Int`, not from the declared range; a `Finite` narrowing is a domain constraint on top. The front end gains the width if it does not carry it yet.
- Unsupported admitted forms (sequences beyond a bounded length, text, non-unit outcomes before R7) refuse emission naming the construct; a refusal is `NotRun`, never a pass.
- Determinism: the same fixture produces the same `symbolic.txt`; engine timeouts are not blessed by golden update under required mode.
- rIC3 witnesses may omit states the engine did not assign at later frames; replay takes the first successor matching the assigned fields and records the choice; frame 0 must be complete.
- No Temporal detail in `model/umpire` (owner rule); the back end is system-agnostic.

## Acceptance Criteria

- **R1:** `umpire-symbolic` emits every Kettle-shaped fixture (bools, ranged ints, fieldless enums, unit outcomes, no action inputs, concrete starts) as BTOR2 from the embedded admitted tree with one `bad` per property, first-match branches, enabledness and stutter, one-step history for postconditions with `after =` filters, and checked arithmetic latched as its own property. Errors: an unsupported admitted form refuses emission naming the construct and the Model.
- **R2:** A pinned rIC3 is built and admitted into `model/build/ric3` with checksum, lock and symlink refusal; the runner runs IC3 for proofs and BMC to the native depth plus one for violations, with a bounded timeout, output drained, children reaped, and no files outside the build root. Errors: a missing or mis-versioned tool fails before any fixture; a tool failure is `NotRun` with its message.
- **R3:** Witnesses are decoded through typed `Value`s by declaration index and replayed through `checked_step`; the violated claim is re-judged natively, and an arithmetic witness is validated by the native arithmetic refusal. The replay types are shared with `umpire-apalache`, not copied. Errors: a non-replayable or non-confirming witness is witness-rejected and fails the fixture.
- **R4:** Kettle has `expected/symbolic.txt` showing agreement on every invariant, postcondition and the arithmetic monitor (unbounded `Safe`) and a replayed, natively confirmed `never_beeps` violation; two mutation controls (a changed postcondition literal, an overflow mutant) are tests that must disagree or witness-reject. Errors: a disagreement is listed in the golden and fails.
- **R5:** The consumer runs only for fixtures with `expected/symbolic.txt` and only under `UMPIRE_BACKENDS=require`; ownership of `symbolic.txt` and `symbolic/paths/*.txt` is registered even when skipped; the default gate is unchanged. No error surface.
- **R6:** Symbolic Worlds: a fixture opting in checks each property once across every World in its declared finite domain; a violation decodes the World, constructs it natively and replays under it; the golden records the domain and, for violations, the decoded World. `parameterized_world` is the first such fixture. Errors: a World field without a finite domain refuses symbolic mode naming the field; a decoded World the Model cannot construct is witness-rejected.
- **R7:** The rest of the admitted subset: action inputs over finite domains, non-unit outcomes, bounded sequences, and Rust widths carried on `Type::Int`; `action_inputs` and `shared_returns` gain goldens. Errors: constructs still outside the subset refuse as in R1.
- **R8:** DESIGN.md and MILESTONES describe the back end, its verdict kinds and the rule that proofs are evidence only with native agreement; `model/design/native-symbolic-backend.md` gains a status line pointing at the crate; fn-211 is recorded as deferred until set-valued state. No error surface.

## Boundaries

- No liveness: TLC owns progress claims. No change to TLC or its goldens.
- Apalache (fn-211) is neither closed nor resumed here; its .4 and .5 stay deferred.
- No new Model authoring surface; symbolic mode uses the existing finite-domain declarations.
- No third-party crate beyond the admitted list; rIC3 is a pinned built binary, not a dependency. Its license is checked in R2 before admission.
- Not an SMT-LIB consumer; the SMT-LIB unrolling stays a development tool.

## Decision Context

2026-10-23, owner: "let's mark fn-240 as deferred for now". Deferred before any task started; the probe at `model/design/native-symbolic/` stays runnable. Revive when a Model needs proofs beyond TLC's reach or when set-valued state arrives and the arrays-versus-Apalache choice is made.

2026-10-23, owner: approved the spike and, on its positive verdict, asked to file this spec ("make the prototype real ie as a real backend"). The probe moved from `.flow/tmp` to `model/design/native-symbolic/` so it is preserved beside its write-up; tasks port from it rather than rewrite.

2026-10-23, naming: the crate and golden are called `symbolic` rather than `btor2`, since BTOR2 is the current wire format and a later SMT-LIB-with-arrays route for set-valued state would sit in the same back end. The owner may rename.

2026-10-23, Apalache: redundant for the word-level subset admitted today; its remaining value is symbolic sets, functions and unbounded sequences. Deferred until set-valued state exists, per the write-up; fn-211's R6 (symbolic Worlds) moves here as R6.

2026-10-24, paths (fn-244): the back ends sit under `model/umpire/backends/{native,tla,tlc,apalache}` and this crate joins them at `backends/symbolic`; the fixture runner is `model/umpire/dev/testing`, fixtures are `model/umpire/dev/fixtures/fixtures/<name>`, and the shared replay types to lift out of `umpire-apalache` live in `backends/apalache`.

Tool recipe from the spike: rIC3 v1.5.2 from source needs cmake and a `clang` on PATH; the binary is `ric3`; z3 (for the SMT-LIB tool) came from the `z3-solver` wheel.
