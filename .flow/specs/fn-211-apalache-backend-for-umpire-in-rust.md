# Apalache backend for Umpire in Rust (model)

## Conversation Evidence

> user (2026-10-18): "should we also include the backends/apalache/ prototype already in fn-210?"
> assistant: keep it out of fn-210 (no verdict evidence, the TLC slice already proves lifting and emission, Apalache needs type annotations and bounded agreement rules); file it as its own spec on top of the shared emitter once the verdict is in and the TLC slice is green.
> user (2026-10-18): "let's file it now as a follow-up already"
> assistant (2026-10-21): each run binds one concrete World, which TLC requires; Apalache can do more: with ConstInit it checks all values of a constant in a range in one symbolic run, rather than once per World. fn-211 as written binds the selected World concretely too.
> user (2026-10-21): "well want that"

## Goal & Context

fn-210 builds Umpire in Rust (`temporal/model`) with one TLA+ backend, TLC, which gives an exhaustive reachable-set agreement with the native checker. Apalache adds a different kind of answer: a bounded symbolic verdict per invariant, with a counterexample when one exists within the bound, and it scales to state spaces TLC cannot enumerate. This spec adds Apalache as a second backend over the same lifted tree and TLA+ emitter that fn-212 delivered and fn-227 extends, run through the same fixture runner, with the same rule that every backend answer is confirmed by the native checker. It starts after fn-210, fn-212 and fn-227 are complete, with the recorded Rust decision; it is filed now so the design is recorded while fn-210's emitter is being shaped. The Scala-side equivalent is fn-206, deferred.

## Compatibility policy for remaining work

Owner, 2026-10-24: remaining work does not require backward compatibility or byte-for-byte equality with old outputs. Where needed for the planned change, internal and wire representations, generated code, diagnostics and goldens may change; update their in-tree producers/readers and regenerate affected guests and goldens together. Do not add legacy decoders, dual formats or old-guest support. Verify Model meaning, the current native contract, typed replay, reproducibility and safety instead. Deterministic ordering is a requirement of the new contract, not an obligation to retain old bytes. Pinned tool checksums, content-addressed integrity and actual proofs/negative controls remain required. This amendment covers only the remaining .4/.5 plans; fn-211 remains deferred and completed task bodies are unchanged.

## Architecture & Data Models

- Emitter: the TLA+ emitter in `umpire-lift` gains Snowcat type annotations (`\* @type: ...;`) for every variable, constant and operator it emits, derived from the Model's Rust types through the same mapping the TLC slice uses; TLC ignores them, with an explicitly selected Apalache safety relation as specified below.
- Runner: a crate `umpire-apalache`, placed next to `umpire-tlc` (same folder convention as fn-210.8 chooses), runs a pinned Apalache release on the mise-pinned JDK (Java 21 or newer) from `model/build/`, never from the user's home directory. One `apalache-mc check --inv=<invariant> --length=<N>` per invariant, where N is the native checker's BFS depth plus one; a server process, if used, is started and stopped by the runner with no process left behind.
- Counterexamples: Apalache's ITF JSON trace is decoded through the existing typed source Schema/Value contract into a path of states and actions (the ITF envelope through serde derive with `deny_unknown_fields`; the strict value visitor and its budgets stay; no `itf` crate), and replayed step by step through the Model's `checked_step` by the native checker; a path that is not a path of the Model is a witness-rejected result.
- Agreement: for every fixture selected for Apalache, each invariant's bounded verdict must agree with the native checker within the bound: no violation where the checker verifies, and a replayable violation where the checker refutes within that depth. The result is a golden `expected/apalache.txt` (invariant, bound, verdict, agreement), produced by an Apalache consumer in the fixture runner and selected by the presence of that file.
- Symbolic Worlds: for a fixture that opts in, declared World fields are emitted as TLA+ `CONSTANT`s constrained by a `ConstInit` predicate over each field's declared finite domain, so one Apalache run per invariant covers every World in that domain instead of one selected World. Counterexamples carry the chosen constant values; they are decoded into a concrete World and replayed natively under it.
- Postconditions and progress: postconditions are emitted as action properties checked as invariants over a one-step history; `boils`-style progress claims are out of scope for Apalache (TLC covers liveness).

## API Contracts

- `Apalache::check(module, invariant, length) -> Result<Bounded, Refusal>` where `Bounded` is `NoViolation { length }`, `Violation { path }` or `NotRun { reason }`.
- `apalache.txt` lists, per invariant in declaration order: name, bound, verdict, and `agrees` or the disagreement; a violation also writes `expected/apalache/paths/<invariant>.txt`.
- The fixture runner runs the Apalache consumer only for fixtures with `expected/apalache.txt`; `make model-update` rewrites it like any golden.
- The Apalache step runs only when `UMPIRE_BACKENDS=require` (or a dedicated make target); the default `make model-check` looks for no JVM tool beyond what fn-210 already requires.

## Edge Cases & Constraints

- Bounded checking says nothing beyond the bound; the golden states the bound, and agreement is only claimed within it.
- A module Apalache refuses (type errors, unsupported operators, inliner limits) is `NotRun` with the tool's message, never a pass.
- Integer semantics follow fn-210 R11: evaluated overflow and invalid division are failures, never disabled transitions. Use the total checked-arithmetic error monitor below; range constraints alone are insufficient.
- No Apalache-specific constructs leak into Models; the Model author writes nothing new.
- Determinism: the same fixture produces the same `apalache.txt`; timings never appear in goldens.

## Acceptance Criteria

- **R1:** The TLA+ emitter emits Snowcat type annotations for every variable, constant and operator of every fixture TLC already runs, and TLC still passes with the same Model semantics on the annotated modules; reviewed goldens may be regenerated. Errors: a Rust type with no TLA+ type mapping refuses emission naming the type.
- **R2:** `umpire-apalache` runs a pinned Apalache release on the mise JDK, one bounded check per invariant at the native depth plus one, with deterministic lifecycle and no files outside the workspace build directory. Errors: a missing or mis-versioned release fails before any test; a tool failure is `NotRun` with its message.
- **R3:** An Apalache violation is decoded from ITF through typed Schema/Value decoding into a path and replayed through the Model's `checked_step`; a non-replayable trace is witness-rejected and fails the fixture. No error surface beyond that.
- **R4:** Kettle's invariants and postconditions have an `apalache.txt` golden showing agreement within the bound, and Kettle's negative control (`!beeping`) shows a replayed violation within the bound. Errors: a disagreement is listed in the golden and fails.
- **R6:** Symbolic World constants. A fixture opting into symbolic mode emits its declared World fields as constants with a `ConstInit` over their declared finite domains and checks each invariant and postcondition in one bounded Apalache run per property across all Worlds in that domain. A violation decodes the constant values into a concrete World, constructs it natively, and replays the trace through `checked_step` under that World; `apalache.txt` records the World domain and, for violations, the decoded World. NoViolation is confirmed natively for the fixture's selected Worlds, and for every World when the domain product is within the catalog limit; beyond that it is reported as bounded and symbolic, never as native agreement. Errors: a World field without a declared finite domain refuses symbolic mode naming the field; a counterexample World outside the domain, or one the Model cannot construct, is witness-rejected and fails the fixture.
- **R5:** The Apalache consumer runs only for fixtures with `expected/apalache.txt` and only when backends are requested; the default gate is unchanged. No error surface.

## Boundaries

- No liveness through Apalache; TLC owns progress claims.
- No Quint; Apalache reads the emitted TLA+ directly.
- No new authoring surface or Model annotations, except that symbolic mode (R6) needs each World field's finite domain, declared with the existing one-range spelling from fn-227 (`Finite` types or `#[inputs(lo..=hi)]`), never a second bound table.
- Does not start before fn-210 is done and its verdict is to continue in Rust.

## Backend implementation contracts

**Emitter semantics (R1/R2).** Task .1 owns an explicit Apalache safety projection over the same independently lifted source, not merely annotation comments. Emit a typed Init and Next without ENABLED or imperative TLC Assert. The safety Next is the union of lifted action transitions plus an explicitly tagged stutter that preserves all state and monitor fields; unconditional stutter adds no state-safety violation and carries no authored action. TLC retains its existing completion/liveness relation and results. Do not feed the TLC Complete operator into Apalache. A real pinned-tool Kettle check must pass during .1 before the runner integration is accepted; the tool admission helper may be introduced here and finalized in .2. Source admission remains located and explicit; do not substitute native explored edges for source translation.

**Checked arithmetic (R1/R2/R3).** Emit each evaluated expression with a typed value and validity predicate. Respect Rust evaluation order, short-circuit guards and untaken branches. Use total fallback values only when invalid, and latch an arithmetic-error monitor on an evaluated overflow, division/remainder by zero, or signed MIN/-1 division/remainder overflow. An error successor remains reachable and violates a dedicated safety invariant; it must not disappear through a range restriction. Include start/claim evaluation as well as action expressions. Decode the offending action and replay its valid prefix through checked_step; the expected native arithmetic refusal validates an arithmetic-failure result, never an ordinary accepted Model edge. Pin successor overflow, intermediate overflow with an in-range final mathematical result, zero division, MIN/-1, and a guarded unevaluated error. Keep arithmetic failure distinct from an authored invariant violation and from tool NotRun. No success is reported after an error.

**Postconditions (R1/R4).** Task .1 also owns one-step history: previous complete state, concrete action identity and outcome/facts where the predicate uses them, plus a has_step flag. Init sets has_step=false with typed defaults; each real action records its pre-state and exact action/outcome, then sets has_step=true. Check each postcondition as an invariant guarded by has_step and its exact after-action selector. Stutter preserves history and cannot fabricate a new authored event. Property metadata distinguishes invariants, postconditions and arithmetic checks. Add a failing postcondition and an after-filtered negative control, and prove unrelated actions do not trigger that filter. Replay strips only declared monitor fields/stutter, validates initial membership and every concrete action/choice, and rechecks the specific violated claim on the reconstructed transition.

**Typed trace decoding (R3).** Reuse the current source type schema and TLC typed-value conventions, with explicit ITF conversion. Display/Debug are rendering only, never a parsing or identity contract; Spell has been removed. Preserve selected World, complete starts and concrete action inputs from fn-227. Refuse malformed/unknown/missing/duplicate fields, out-of-range values, crossed World bindings and non-replayable edges. Do not look up states by display strings or guess a payload from the action's variant name.

**Bound and agreement (R2/R4).** N counts Model transitions from an initial state at depth zero; the selected bound is the completed native BFS maximum discovery depth plus one, so outgoing postcondition failures are covered. Pin length-zero/one controls against the selected release's CLI length convention. A truncated native exploration does not establish exhaustive agreement; report incomplete instead. Bounded NoViolation is never an unbounded proof. Required-mode NotRun, timeout, unsupported source, malformed trace or tool failure fails the selected fixture and cannot be blessed by golden update.

**Fixture ownership (R5).** The Apalache consumer exclusively owns apalache.txt and apalache/paths/*.txt. The checker keeps paths/*.txt. Register both ownership patterns before execution, including when the backend is skipped. Default mode preserves/skips committed Apalache payloads without launching/downloading a tool; required mode compares every selected output. Tests cover a committed violating trace in both modes and reject duplicate ownership or stale/unowned backend output.

**Runner lifecycle (R2).** Reuse TLC's process and canonical-root containment patterns rather than copy its orchestration. Resolve tool/work paths under the approved build root, reject symlink escapes, download/check a pinned checksum into a private staging directory, and atomically publish the validated installation under a lock. Concurrent installers must converge without partial visibility. Each invocation owns a separate work directory and bounded timeout, drains output while running and terminates/reaps children on timeout/cancellation/error. Tests use a fake tool for wrong version, failed launch/exit, timeout, child cleanup, symlink escape and concurrent installation, plus actual Kettle checks. Emit fixture/property/bound/start/finish/failure progress to stderr; deterministic goldens exclude timestamps, absolute paths and timing.

The supported-feature boundary was checked against the official Apalache feature table on 2026-10-21: https://apalache-mc.org/docs/apalache/features.html. The unsupported ENABLED form requires this explicit safety projection; the implementation spike must verify the pinned release rather than assume annotations alone are sufficient.

## Decision Context

2026-10-24, owner: "can we drop byte-for-byte compatability from open tasks to speed things up? we don't need backwards compatabiltiy". This supersedes old-output/backward-format requirements for remaining work. Current-contract integrity and semantic correctness remain required. Fresh plan review is required before implementation of this amended scope.


2026-10-22, owner: deferred after .3; the unfinished .4 consumer is preserved on `work/fn2114-deferred` at `fee5fc56ae`.

2026-10-23: Apalache left the required gate (`UMPIRE_APALACHE=require` runs its real-tool controls); resuming this spec restores it. The native symbolic spike (BTOR2 + rIC3) was positive and is filed as fn-240, which takes over R6 (symbolic Worlds) and the large-state-space goal for the word-level subset without TLA+, Snowcat or a JVM. This spec resumes only when set-valued state (sets, functions, unbounded sequences) is admitted and the arrays-versus-Apalache choice is made; on resumption R6 and task .5 are dropped here in favour of fn-240 R6 unless the owner decides otherwise, and .4 is re-anchored.

2026-10-24, paths (fn-244): `umpire-lift` is `umpire-tla` at `model/umpire/backends/tla` and owns the subprocess supervision TLC and Apalache share; `umpire-apalache` no longer depends on `umpire-tlc`; the fixture runner is `model/umpire/dev/testing` and the fixtures are `model/umpire/dev/fixtures/fixtures/<name>`.

2026-10-23, task .3 JSON dependency reuse (R18): admit direct host-backend use of the already resolved `serde` 1.0.229 and `serde_json` 1.0.154 for strict bounded ITF decoding with duplicate-key detection. Their official releases were published [2026-07-18](https://github.com/serde-rs/serde/releases/tag/v1.0.229) and [2026-10-11](https://github.com/serde-rs/json/releases/tag/v1.0.154), both within twelve months. All 135 lockfile packages and versions remain unchanged; only existing dependency edges are promoted. These dependencies belong to the optional development backend, outside the guest and runtime dependency closure; retain the closure check in task evidence. No new third-party package or version upgrade is authorized.

2026-10-24, owner (fn-248 crate review): ITF envelope decoding moves to serde derive with `deny_unknown_fields`; the strict value visitor (duplicate keys, typed values) and its size/depth budgets stay. The `itf` crate is rejected: stale under fn-210 R18 and less strict than the current decoder. The change to the delivered .3 code is owned by fn-248 (R1), not by a task here; .4 and .5 consume the decoder as it then stands and do not rework it. New Cargo dependencies are allowed when admitted under R18, and byte-for-byte golden compatibility is no longer required (goldens may be re-accepted after review).

Apalache was kept out of fn-210 because R24 decides fn-210 on the wasm slice alone, the TLC slice already proves lifting and emission, and Apalache adds type annotations and bounded agreement rules that TLC does not need. Filed on 2026-10-18 at the owner's request so the emitter is shaped with annotations in mind. It reuses fn-210's lifted tree, emitter, typed decoding and fixture runner, and selects fixtures by the presence of `apalache.txt`, so no Model is defined twice. The Scala-side Apalache spec (fn-206, deferred) explored an in-process route through Apalache's Scala libraries; from Rust the CLI or JSON-RPC server is the route, and a pinned release in the build directory replaces the earlier `~/.quint` download and port-kill lifecycle.

2026-10-18, placement (owner): `umpire-apalache` lives at `model/umpire/backends/apalache/`, beside `umpire/backends/tlc/`.

2026-10-21, owner: add symbolic World constants (R6) so Apalache checks every World in a declared domain in one run, as part of the goal to eventually meet Quint's capabilities fully. Open design point for planning: computed starts depend on the World, so Init over symbolic constants needs either source-lifted `#[starts]` bodies or a per-World start table built natively from values; choose in the task plan, keeping source meaning independent. Native replay needs a World built from decoded values, the inverse of fn-227.14's derived value export.

2026-10-21, owner: fn-211 consumes fn-238's shared front end; the shared reading of a Model replaces independent source admission, with typed replay as the cross-check.

2026-10-18: fn-210 R30's feedback-loop budget applies; R5 keeps Apalache out of the default gate for that reason.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The TLA+ emitter emits Snowcat type annotations for every variable, constant and operator of every fixture TLC already runs, and TLC still passes with the same Model semantics on the annotated modules; reviewed goldens may be regenerated. Errors: a Rust type with no TLA+ type mapping refuses emission naming the type. | fn-211-apalache-backend-for-umpire-in-rust.1 | — |
| R2 | `umpire-apalache` runs a pinned Apalache release on the mise JDK, one bounded check per invariant at the native depth plus one, with deterministic lifecycle and no files outside the workspace build directory. Errors: a missing or mis-versioned release fails before any test; a tool failure is `NotRun` with its message. | fn-211-apalache-backend-for-umpire-in-rust.2 | — |
| R3 | An Apalache violation is decoded from ITF through typed Schema/Value decoding into a path and replayed through the Model's `checked_step`; a non-replayable trace is witness-rejected and fails the fixture. No error surface beyond that. | fn-211-apalache-backend-for-umpire-in-rust.3 | — |
| R4 | Kettle's invariants and postconditions have an `apalache.txt` golden showing agreement within the bound, and Kettle's negative control (`!beeping`) shows a replayed violation within the bound. Errors: a disagreement is listed in the golden and fails. | fn-211-apalache-backend-for-umpire-in-rust.4 | — |
| R6 | Symbolic World constants. A fixture opting into symbolic mode emits its declared World fields as constants with a `ConstInit` over their declared finite domains and checks each invariant and postcondition in one bounded Apalache run per property across all Worlds in that domain. A violation decodes the constant values into a concrete World, constructs it natively, and replays the trace through `checked_step` under that World; `apalache.txt` records the World domain and, for violations, the decoded World. NoViolation is confirmed natively for the fixture's selected Worlds, and for every World when the domain product is within the catalog limit; beyond that it is reported as bounded and symbolic, never as native agreement. Errors: a World field without a declared finite domain refuses symbolic mode naming the field; a counterexample World outside the domain, or one the Model cannot construct, is witness-rejected and fails the fixture. | fn-211-apalache-backend-for-umpire-in-rust.5 | — |
| R5 | The Apalache consumer runs only for fixtures with `expected/apalache.txt` and only when backends are requested; the default gate is unchanged. No error surface. | fn-211-apalache-backend-for-umpire-in-rust.4 | — |

2026-10-24, owner (throughput ordering): consolidation lands before the feature work that would otherwise repeat each change in every copy. Order: fn-247 → fn-248 → fn-250, fn-251 and fn-252 as parallel lanes → fn-246 → fn-237.4 → fn-243.5 → fn-237.5 → fn-245; fn-240 and fn-211 stay deferred behind fn-250. This spec now depends on fn-250: the deferred consumer resumes on the shared Literal, Schema and replay core rather than the per-crate copies (rebase `work/fn2114-deferred` then).
