# TLC backend in Scala

## Conversation Evidence

> user (2026-10-18): "we could code up the TLC and Apalache integration in Scala to make it a single full package that only requires Scala instead of Go too"
> user (2026-10-18): "let's do one spec per backend"
> user (2026-10-18): "let's mark [Quint] as deferred and start with TLC and Apalache individually actually"

## Goal & Context

TLC is the explicit-state checker: exhaustive over a finite space, with temporal properties and a deadlock check, and it can dump every reachable state it visits. No TLC route exists today; the design notes leave it as an open question behind a Quint-mediated path. This spec stands alone: it emits TLA+ from the lifted tree in `framework.export.tla` (shared with fn-206; whichever lands first owns it), runs `tla2tools` from the JVM in `framework.backends.tlc`, reads TLC's state dump and JSON trace, and holds the dump to the native graph. That gives the exhaustive transition agreement the Quint dump used to give, and adds the temporal verdicts (progress, fairness) that only TLC answers externally. `checkedBy(Checker.tlc)` and `confirm` select a run.

## Architecture & Data Models

- TLA+ emitter as in fn-206 R1, extended here with fairness: weak fairness on the classes a progress claim names, the liveness property per progress claim, and the deadlock predicate aligned with the Model's `end`.
- Finite observer state: monitors as finite automata; no history variable.
- Runner over `tla2tools.jar` on the framework classpath: `-config`, `-deadlock` kept on, `-dump` for the reachable states, `-dumpTrace json` for counterexamples, workers from the host. Spike whether repeated in-process runs are safe given TLC's static state; otherwise one JVM per run with the jar pinned.
- Agreement: the dumped reachable states decode through the lifted types and are compared as a set with the native graph's states; invariant verdicts per Property and monitor compare with the native verdicts; a TLC counterexample decodes to a `ReplayScript` and replays natively. Receipts: transition (reachable set), property, monitor, progress and coverage, plus witness-rejected and not-run.
- Native rows with a decoder-compatible value spelling come from fn-203 R1; this spec adds the TLA+ value spelling to the shared encoder.
- Claim selection: `checkedBy(Checker.tlc)` routes a claim here; `confirm` runs the named others; `Checker.umpire` is the default.

## API Contracts

- `Tla.emit(lifted): Either[Unsupported, Module]` with `Module.text`, `Module.config` and, new here, fairness and liveness sections.
- `Tlc.check(module, config): Verdict` with `ok(statesVisited, distinctStates)`, `violation(property, path)`, `deadlock(path)`, `notRun(reason)`; `Tlc.dump(module, config): Iterator[State]`.
- `TlcBackend.agree(dump, verdicts, graph, instance): Receipts` with the kinds above.
- The opt-in backend stage, `UMPIRE_BACKENDS=require`, runs TLC on the six samples; the default run looks for no tool.

## Edge Cases & Constraints

- TLC exits the JVM on some failures by default; the runner must set the flags that prevent it or run TLC in a child JVM.
- A reachable set that exceeds a configured ceiling stops the run with resource-limit, never a pass.
- Symmetry and state constraints are not written; the samples' worlds are small enough. A larger world is resource-limit.
- The numeric contract from fn-201 decides the integer translation.

## Acceptance Criteria

- **R1:** The TLA+ emitter (shared with fn-206) writes fairness, liveness and deadlock sections from progress claims and `end`; an unsupported Model returns the reason. Errors: a progress claim over an unbound class refuses naming it.
- **R2:** A design note records the spike on in-process TLC versus a child JVM, and the chosen route. No error surface.
- **R3:** The runner executes the pinned `tla2tools` with deadlock checking on, reads the state dump by streaming and the JSON trace on violation. Errors: a missing or mis-versioned jar fails before any test; a TLC failure is not-run with the error text.
- **R4:** The dumped reachable set equals the native graph's states for every sample, and invariant verdicts per Property and monitor agree; counterexamples replay through `Checker.replay`. Errors: a state the dump leaves out or adds is a difference; a refused replay is witness-rejected before any difference.
- **R5:** Progress claims get TLC liveness verdicts that agree with the native progress verdicts on the samples, including deadlock. Errors: a liveness verdict TLC cannot decide within the ceiling is resource-limit.
- **R6:** `checkedBy(Checker.tlc)` and `confirm` select backend runs; `Checker.umpire` remains the default. Errors: naming a backend the stage cannot run is unsupported with the reason.

## Boundaries

- No Apalache run (fn-206).
- No Quint emission (fn-205, deferred).
- No symmetry reduction, no Temporal slices.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

The design notes proposed reaching TLC through `quint compile --target tlaplus`, which itself invokes Apalache, and flagged two problems with Quint's TLC wrapper: it disables the deadlock check and returns no structured trace. A direct emitter and a direct `tla2tools` run avoid both and give the state dump, which replaces the Quint one-step dump as the exhaustive comparison. The emitter is shared with fn-206 rather than duplicated; this spec adds the temporal sections. Absorbs deferred fn-173.7 (export coverage) and the TLC half of fn-173.9. Depends on fn-201 (lift tree, numeric contract) and fn-203 R1 (native rows). Filed 2026-10-18 as part of the one-language series (fn-199 through fn-208).
