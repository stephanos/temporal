# Quint backend in Scala

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user (2026-10-18): "we could code up the TLC and Apalache integration in Scala to make it a single full package that only requires Scala instead of Go too"
> user (2026-10-18): "let's do one spec per backend" (Quint, Apalache, TLC)
> user (2026-10-18): "let's mark that as deferred and start with TLC and Apalache individually actually"

## Goal & Context

The Quint export is the second opinion: the Model is translated declaration for declaration into one Quint module whose single state holds a dump of every reachable state, every class result, the monitor products and the Property readings, and that dump is compared key by key with the reader's tables. Today both the translation and the reference side are Go over the IR. This spec moves the translation onto the lifted tree and the reference side onto the native graph, in `framework.export.quint` and `framework.backends.quint`. **Deferred (owner, 2026-10-18):** Apalache (fn-206) and TLC (fn-207) run on a direct TLA+ emission and do not depend on this spec. Quint remains a candidate for its simulator and its REPL-style authoring comparisons; it is filed so the design is on record and is not in any phase.

## Architecture & Data Models

- Emitter over `Lifted`: types, functions and expressions one to one; per machine a class variant type, a step function, starts and ends, and a reachable fixed point whose round count is the native BFS depth plus one; the product with monitors; Property claims; compositions over the member modules with the composed step as the product of member results. Refusals: channels and inbox operations, holes anywhere a machine reaches, anonymous functions outside ends, choice names Quint cannot write as they are.
- Reference side: the native graph exported as rows (fn-203 R1) with a value encoder that spells states, classes, outcomes and facts exactly as Quint's dump decoder expects; `Step` carries its choice name.
- Runner: `quint run --backend typescript --max-samples 1 --max-steps 1 --seed 0x1 --out-itf`, the pinned Quint package through a launcher the framework owns; ITF decoded by the lifted types with the streaming span walker for large dumps.
- Agreement receipts: transition, monitor, property and coverage, plus unsupported for Queries, progress and refinements, and the resource-limit and witness-rejected standings.
- Counterexample replay: a Quint path becomes a typed `ReplayScript` and runs through `Checker.replay`; a rejected replay stands before any difference.

## API Contracts

- `Quint.emit(lifted): Either[Unsupported, Module]`; `Module.text` is the `.qnt` source.
- `QuintBackend.run(module, tool): Dump`; `QuintBackend.agree(dump, graph): Receipts`.
- `framework check` gains an opt-in backend stage, `UMPIRE_BACKENDS=require`, that fails on a missing or mis-versioned tool, a tool failure or any disagreement; the default run looks for no tool.
- Receipts are the existing kinds with the existing names so the design notes' tables stay readable.

## Edge Cases & Constraints

- Quint's simulator proves nothing by sampling; the module computes the fixed point itself and reports whether a step leaves the set (`closed: false` fails).
- A value no `match` accepts and a call outside a precondition are written as expressions the evaluator stops on, so the run fails instead of producing a row.
- The numeric contract from fn-201 decides the integer translation; the module header states the bound.
- Dumps can reach gigabytes on large slices; decoding streams and never holds the whole JSON.

## Acceptance Criteria

- **R1:** `Quint.emit` translates every sample Model into one module with the refusals listed; an unsupported Model returns the reason and writes nothing. Errors: each refusal names the declaration and position.
- **R2:** The native graph exports rows whose spellings decode to the same keys as the Quint dump; `Step` carries its choice name. Errors: an unencodable value refuses naming the type.
- **R3:** The runner executes the pinned Quint through the framework's launcher and decodes the ITF dump by streaming. Errors: missing or mis-versioned tool fails before any test; a dump that cannot be decoded is an error of the call.
- **R4:** Agreement produces transition, monitor, property and coverage receipts over the six samples with no disagreement; Queries, progress and refinements are unsupported receipts. Errors: a pair the dump leaves out is a difference, never disabled.
- **R5:** Quint counterexamples replay through `Checker.replay`; a rejected replay yields witness-rejected before any difference. No error surface beyond that.
- **R6:** The Go `export` package's Quint translation, agreement, ITF, dump, raw and checked parts are no longer run by any gate (deletion is fn-208). No error surface.

## Boundaries

- No Apalache verdict (fn-206), no TLA text or TLC (fn-207).
- No Temporal slices; the gate runs over the samples until Temporal sources return.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

Keeping the translation in Go would keep the Go reader as the reference side, which is being retired. The translation is tree-walking over the IR shape the lift already holds, so it moves nearly one to one; the new work is the reference side, where the native graph must spell values as the dump decoder expects, and that encoder is shared with fn-204's descriptor and fn-203's rows. The receipt vocabulary is preserved so the historical comparisons remain comparable. Deferred by the owner on 2026-10-18 in favour of direct TLA+ routes: Quint would add a translation hop and a second tool pin in front of both checkers, and its exhaustive comparison trick (one dump step) is replaced by TLC's state dump in fn-207. Would depend on fn-201 and the fn-192 recut if revived. fn-173.7/.8 are absorbed by fn-207 instead. Filed 2026-10-18 as part of the one-language series (fn-199 through fn-208).

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.
