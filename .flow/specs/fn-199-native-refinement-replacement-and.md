# Native refinement, replacement and composed fairness

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user (2026-10-18): "what functionality do we have in temporal/tools/umpire that should actually be in temporal/model/framework? because it's simpler, or faster, or smaller (less code) or better integrated there."
> owner (2026-10-18): approved filing the ten-spec series that moves every Model-reading capability out of Go; this spec is the first because it closes the blockers the measurement note names.

## Goal & Context

The Scala checker is the oracle (fn-196), but the measurement decision keeps the Go `check` package alive because four obligations have no native owner: refinement simulation and cross-owner claims, the replacement-composition proof, composed fairness and composition ceilings, and the receipt kinds those consumers read. Today the native checker answers `unknown` with the detail "reads … through a refinement, which no native check reads" at every refinement site, and `MemberReplacement` is typing only. This spec gives each obligation a native implementation over the reachable graph, so that the later retirement series (the IR made internal, Cases produced in Scala, backends in Scala, the Go rename) can proceed without a second Model interpreter. It touches only the checker; it does not change what a Model author writes.

## Architecture & Data Models

- Refinement is checked over two reachable graphs, the refining Model's and the refined Model's, each explored under the caller's budget. The state map is the typed function the `Refinement` declaration already holds; there is no key parsing and no catalog. The refined graph supplies membership, starts and visible outcomes/facts.
- Simulation failures keep the Go vocabulary: catalog (mapped state not in the refined graph), initial (mapped start not a refined start), unmatched (a visible step the refined Model cannot take), visible stutter (a step that changes nothing visible is allowed only when the declaration permits it), and incomplete (either graph cut by budget, or an unread hole). Each failure carries a concrete path in the refining Model and, where one exists, the product witness.
- Queries and claims read through a refinement evaluate the refined Model's claim on the mapped state and step. The three native `unknown` sites become calls into the refinement runtime.
- A replacement composition is admitted only when the replacing member refines the replaced member under a declared or derived map and the replacing member's starts cover the replaced member's starts. Failure refuses the composition with the member name and the first uncovered start or simulation failure.
- Composed fairness accepts sync references and member-qualified action references in `fair` and resolves them to composed classes. Composition ceilings are the caller's budget applied to the product graph; `end` of a composition is the authored override or `true`.
- Verdict vocabulary gains the kinds the moved consumers need: resource limit (budget cut without a decisive answer), unsupported (a declaration the native checker refuses, with reason), and declaration error. Existing verdicts keep their meaning.

## API Contracts

- `Checker.verifyAll(instance)` includes refinement results for every refinement the instance declares, one result per refinement with verdict, failure kind, path and witness.
- `Checker.refines(instance, refinement, budget)` answers one refinement directly with the same `Result` shape.
- A composition built with `withMember` whose replacement does not refine or does not cover starts refuses at construction with a message naming the member and the failure kind; the failure kind is also reachable programmatically.
- `fair(...)` on a composition accepts sync vals and `member(action)` forms. An unbound or ambiguous reference refuses at declaration time.
- `Result.verdict` adds `resourceLimit`, `unsupported` and `declarationError`. Every existing consumer of `Verdict` compiles unchanged.

## Edge Cases & Constraints

- Either graph cut by budget yields `incomplete`, never `verified`. A counterexample found before the cut stands.
- A hole reached in either Model marks the affected pairs unread; a refinement whose only failures are unread pairs is `incomplete`, not `refuted`.
- Rule disjointness stays as it is: the native step function takes the first firing rule and no side checks pairwise guard overlap. This spec records the owner's disposition in Decision Context and implements nothing for it.
- Memory: the refined graph is explored once per refinement question and released with the Result (fn-196 R8 lifetimes apply).
- No IR, lifter, exporter or Go code changes. The Go `check` package keeps its implementation until the rename spec deletes it.

## Acceptance Criteria

- **R1:** Refinement simulation over two reachable graphs answers verified, refuted with a path and failure kind (catalog, initial, unmatched, visible stutter), or incomplete (budget cut or unread hole). Errors: a refinement whose map throws on a reachable state refutes with that state; a missing refined start set refuses at declaration.
- **R2:** Queries and claims that read through a refinement evaluate on the mapped state and step instead of answering unknown; a cross-owner claim over an unrelated owner refuses with the owner names (no error surface beyond that).
- **R3:** `withMember` replacement is admitted only when the replacing member refines the replaced one and covers its starts; otherwise construction refuses naming the member and the first failure. Errors: a replacement of a member that declares no refinement map refuses with "no map".
- **R4:** Composed fairness resolves sync vals and member-qualified actions to composed classes; progress claims over compositions answer deadlock, fair cycle and missed deadline with lasso witnesses as machines do. Errors: unbound or ambiguous fairness references refuse at declaration.
- **R5:** Composition exploration respects the caller's budget and reports resource limit when cut; `end` uses the authored override or `true`. No error surface beyond the budget rules of fn-196.
- **R6:** `Verdict` gains resource limit, unsupported and declaration error; `verifyAll` reports every refinement, replacement and composed progress result in its `Report`, and the samples' reports are unchanged for Models that declare none. No error surface beyond R1–R5.
- **R7:** The framework README and SEMANTICS state the native refinement, replacement and composed fairness contracts and remove the "no native check reads" wording; the measurement note's obligation table is updated to point here. No error surface.

## Boundaries

- No rule-disjointness check; recorded as the owner's decision only.
- No IR, exporter, lint, lowering or Go changes; the Go `check` package is deleted by the rename spec, not here.
- No new author-facing DSL vocabulary beyond accepting sync and member references in `fair`.
- No performance target; the measurement note's budgets apply.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

The alternative was to port Go's catalog-based `RefineTables` over serialized IR tables, which would keep a key world alive in Scala. Over two reachable graphs the state map is a typed function, membership is a set lookup and the budget replaces the catalog ceiling, so the native version is smaller and composes with the lifetime rules already in place. The replacement proof and composed fairness are small once refinement exists, so they share the spec rather than waiting. Verdict kinds are added here because every later consumer (lint, lowering, descriptor, backends) reads them; adding them later would touch each consumer twice. Rule disjointness is left untouched because deleting the Go check cannot settle it and the owner has not chosen between native enforcement and a semantic change; the question stays open in the measurement note. Filed 2026-10-18 as the first of the one-language series: fn-199 refinement (this), fn-200 native lint, fn-201 IR internal, fn-202 Testpilot schema in the framework, fn-203 Cases in Scala, fn-204 conformance descriptor, fn-205 Quint, fn-206 Apalache, fn-207 TLC, fn-208 the `tools/testpilot` rename.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.
