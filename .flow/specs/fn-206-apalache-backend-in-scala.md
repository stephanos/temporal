# Apalache backend in Scala

## Conversation Evidence

> user (2026-10-18): "we could code up the TLC and Apalache integration in Scala to make it a single full package that only requires Scala instead of Go too"
> user (2026-10-18): "let's do one spec per backend"
> user (2026-10-18): "let's mark [Quint] as deferred and start with TLC and Apalache individually actually"

## Goal & Context

Apalache gives a bounded model-checking verdict on a monitor: a violation within the bound, or none within it. Today it is reached only through `quint verify`, which downloads it into the user's home, starts a server on a fixed port, and is stopped by `pkill`; the verdict is read from the tool's text. Apalache is a Scala program on the JVM the framework already runs, and it takes TLA+ directly. This spec stands alone: it emits TLA+ from the lifted tree in `framework.export.tla`, runs Apalache from Scala in `framework.backends.apalache`, in-process where its published artifacts allow or through its JSON-RPC server otherwise, decodes a counterexample into a typed path and confirms it through the native checker. `checkedBy(Checker.apalache)` and `confirm` on claims start to select a run. fn-207 (TLC) reuses the TLA+ emitter; whichever spec lands first owns it and the other extends it.

## Architecture & Data Models

- TLA+ emitter over `Lifted`: one module per Model with its types as TLA+ values (records, sets, functions, sequences), functions and expressions declaration for declaration, per machine the state variables, `Init` from the starts, `Next` as the disjunction of class actions, and the end predicate; monitors as additional variables with their `next`, read and violated predicates; Properties as invariants or action properties; compositions as the product of member modules with syncs as conjoined actions. Refusals: channels and inbox operations, holes anywhere a machine reaches, anonymous functions outside ends, names TLA+ cannot spell.
- Finite observer state: monitors are encoded as finite automata; no growing history variable is written. This is the prerequisite TLC needs and Apalache benefits from.
- Spike first: whether the full Apalache checker is published to Maven (the `tla-ir` and `tla-io` libraries are, from 0.59) and whether `Tool.run` under `OutputManager.withScope` is usable in-process; recorded as a design note. The spec implements whichever route the spike proves.
- Runner: in-process invocation, or a server the framework starts on an ephemeral port and stops deterministically; Java 21+ from the pinned toolchain; the artifact pinned in the framework's dependency list or fetched to the framework build directory.
- Verdict: no violation within the bound, violation with an ITF trace, or not-run with the tool's error. The bound is the native product depth plus one. The trace decodes through the lifted types to a `ReplayScript` and runs through `Checker.replay`.
- Claim selection: `checkedBy(Checker.apalache)` routes a claim here; `confirm` runs the named others; `Checker.umpire` is the default.

## API Contracts

- `Tla.emit(lifted): Either[Unsupported, Module]`; `Module.text` is the `.tla` source and `Module.config` the `.cfg`.
- `Apalache.check(module, invariant, steps): Verdict` with `noViolation(steps)`, `violation(path)`, `notRun(reason)`.
- `ApalacheBackend.confirm(verdict, instance): Receipt`: agreed on a confirmed violation or on no violation when the native checker also finds none within the bound; witness-rejected on a refused replay; disagreement when native finds a violation within the bound that Apalache did not.
- The framework check gains an opt-in backend stage, `UMPIRE_BACKENDS=require`, that fails on a missing or mis-versioned artifact, a tool failure or any disagreement; the default run looks for no tool.

## Edge Cases & Constraints

- Bounded checking says nothing about longer runs; the receipt carries the bound.
- A module the inliner cannot take is not-run, never a pass.
- The numeric contract from fn-201 decides the integer translation; the module states the bound.
- Nothing is downloaded into the user's home; no process is left behind.

## Acceptance Criteria

- **R1:** `Tla.emit` translates every sample Model into a TLA+ module and config with finite monitor automata and the listed refusals; an unsupported Model returns the reason and writes nothing. Errors: each refusal names the declaration and position.
- **R2:** A design note records the spike: Maven availability of the full checker, in-process viability, and the chosen route. No error surface.
- **R3:** The runner executes Apalache from the JVM by the chosen route with a pinned artifact and deterministic lifecycle. Errors: a missing artifact fails before any test; an inliner or tool failure is not-run with the error text.
- **R4:** Each monitored sample gets a bounded verdict per monitor at the native product depth plus one; violations decode to a typed path and replay through `Checker.replay`; receipts are agreed, witness-rejected or disagreement as contracted. Errors: a trace that cannot be decoded is an error of the call.
- **R5:** `checkedBy(Checker.apalache)` and `confirm` select backend runs in the backend stage; `Checker.umpire` remains the default. Errors: naming a backend the stage cannot run is unsupported with the reason.
- **R6:** The Go `export` package's Apalache paths are no longer run by any gate (deletion is fn-208). No error surface.

## Boundaries

- No TLC run (fn-207); the emitter is shared.
- No Quint emission (fn-205, deferred).
- No Temporal slices until Temporal sources return.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

The Quint route was deferred by the owner on 2026-10-18: it put a translation hop and a second tool pin in front of both checkers. Apalache accepts TLA+ directly and is a JVM program, so a direct emitter plus an in-process or managed-server runner is the shorter path, and the emitter is the same one TLC needs. Finite monitor automata are chosen now rather than a history variable because TLC cannot take an infinite space and Apalache's inliner struggled with the Nexus module. The in-process route is preferred if the spike proves it. Depends on fn-201 (lift tree, numeric contract). Shares deferred fn-173.9 (the TLC route) with fn-207. Filed 2026-10-18 as part of the one-language series (fn-199 through fn-208).
