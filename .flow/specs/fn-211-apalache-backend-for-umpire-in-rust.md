# Apalache backend for Umpire in Rust (model2)

## Conversation Evidence

> user (2026-10-18): "should we also include the backends/apalache/ prototype already in fn-210?"
> assistant: keep it out of fn-210 (no verdict evidence, the TLC slice already proves lifting and emission, Apalache needs type annotations and bounded agreement rules); file it as its own spec on top of the shared emitter once the verdict is in and the TLC slice is green.
> user (2026-10-18): "let's file it now as a follow-up already"

## Goal & Context

fn-210 builds Umpire in Rust (`temporal/model`) with one TLA+ backend, TLC, which gives an exhaustive reachable-set agreement with the native checker. Apalache adds a different kind of answer: a bounded symbolic verdict per invariant, with a counterexample when one exists within the bound, and it scales to state spaces TLC cannot enumerate. This spec adds Apalache as a second backend over the same lifted tree and TLA+ emitter that fn-210.8 builds, run through the same fixture runner, with the same rule that every backend answer is confirmed by the native checker. It starts after fn-210 is done and its verdict (fn-210 R24) is to continue in Rust; it is filed now so the design is recorded while fn-210's emitter is being shaped. The Scala-side equivalent is fn-206, deferred.

## Architecture & Data Models

- Emitter: the TLA+ emitter in `umpire-lift` gains Snowcat type annotations (`\* @type: ...;`) for every variable, constant and operator it emits, derived from the Model's Rust types through the same mapping the TLC slice uses; TLC ignores them, so one module serves both backends.
- Runner: a crate `umpire-apalache`, placed next to `umpire-tlc` (same folder convention as fn-210.8 chooses), runs a pinned Apalache release on the mise-pinned JDK (Java 21 or newer) from `model/build/`, never from the user's home directory. One `apalache-mc check --inv=<invariant> --length=<N>` per invariant, where N is the native checker's BFS depth plus one; a server process, if used, is started and stopped by the runner with no process left behind.
- Counterexamples: Apalache's ITF JSON trace is decoded through the canonical `Spell` form into a path of states and actions, and replayed step by step through the Model's `step` by the native checker; a path that is not a path of the Model is a witness-rejected result.
- Agreement: for every fixture selected for Apalache, each invariant's bounded verdict must agree with the native checker within the bound: no violation where the checker verifies, and a replayable violation where the checker refutes within that depth. The result is a golden `expected/apalache.txt` (invariant, bound, verdict, agreement), produced by an Apalache consumer in the fixture runner and selected by the presence of that file.
- Postconditions and progress: postconditions are emitted as action properties checked as invariants over a one-step history; `boils`-style progress claims are out of scope for Apalache (TLC covers liveness).

## API Contracts

- `Apalache::check(module, invariant, length) -> Result<Bounded, Refusal>` where `Bounded` is `NoViolation { length }`, `Violation { path }` or `NotRun { reason }`.
- `apalache.txt` lists, per invariant in declaration order: name, bound, verdict, and `agrees` or the disagreement; a violation also writes `expected/paths/apalache_<invariant>.txt`.
- The fixture runner runs the Apalache consumer only for fixtures with `expected/apalache.txt`; `make model-update` rewrites it like any golden.
- The Apalache step runs only when `UMPIRE_BACKENDS=require` (or a dedicated make target); the default `make model-check` looks for no JVM tool beyond what fn-210 already requires.

## Edge Cases & Constraints

- Bounded checking says nothing beyond the bound; the golden states the bound, and agreement is only claimed within it.
- A module Apalache refuses (type errors, unsupported operators, inliner limits) is `NotRun` with the tool's message, never a pass.
- Integer semantics follow fn-210 R11: overflow is a refuted step in the checker; the emitter states the integer bound so Apalache's unbounded integers are restricted to the checker's range.
- No Apalache-specific constructs leak into Models; the Model author writes nothing new.
- Determinism: the same fixture produces the same `apalache.txt`; timings never appear in goldens.

## Acceptance Criteria

- **R1:** The TLA+ emitter emits Snowcat type annotations for every variable, constant and operator of every fixture TLC already runs, and TLC still passes on the annotated modules unchanged. Errors: a Rust type with no TLA+ type mapping refuses emission naming the type.
- **R2:** `umpire-apalache` runs a pinned Apalache release on the mise JDK, one bounded check per invariant at the native depth plus one, with deterministic lifecycle and no files outside the workspace build directory. Errors: a missing or mis-versioned release fails before any test; a tool failure is `NotRun` with its message.
- **R3:** An Apalache violation is decoded from ITF through `Spell` into a path and replayed through the Model's `step`; a non-replayable trace is witness-rejected and fails the fixture. No error surface beyond that.
- **R4:** Kettle's invariants and postconditions have an `apalache.txt` golden showing agreement within the bound, and Kettle's negative control (`!beeping`) shows a replayed violation within the bound. Errors: a disagreement is listed in the golden and fails.
- **R5:** The Apalache consumer runs only for fixtures with `expected/apalache.txt` and only when backends are requested; the default gate is unchanged. No error surface.

## Boundaries

- No liveness through Apalache; TLC owns progress claims.
- No Quint; Apalache reads the emitted TLA+ directly.
- No new authoring surface or Model annotations.
- Does not start before fn-210 is done and its verdict is to continue in Rust.

## Decision Context

Apalache was kept out of fn-210 because R24 decides fn-210 on the wasm slice alone, the TLC slice already proves lifting and emission, and Apalache adds type annotations and bounded agreement rules that TLC does not need. Filed on 2026-10-18 at the owner's request so the emitter is shaped with annotations in mind. It reuses fn-210's lifted tree, emitter, `Spell` and fixture runner, and selects fixtures by the presence of `apalache.txt`, so no Model is defined twice. The Scala-side Apalache spec (fn-206, deferred) explored an in-process route through Apalache's Scala libraries; from Rust the CLI or JSON-RPC server is the route, and a pinned release in the build directory replaces the earlier `~/.quint` download and port-kill lifecycle.

2026-10-18, placement (owner): `umpire-apalache` lives at `model/umpire/backends/apalache/`, beside `umpire/backends/tlc/`.

2026-10-18: fn-210 R30's feedback-loop budget applies; R5 keeps Apalache out of the default gate for that reason.
