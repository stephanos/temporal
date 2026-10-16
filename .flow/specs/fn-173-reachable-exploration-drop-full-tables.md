# Reachable exploration: drop full tables, native collections, backends that decide

> HTML render lens (local): open `.flow/artifacts/fn-173-reachable-exploration-drop-full-tables/spec.html` - regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Native DSL task order (owner override)

Execute .10, .11, then .12 after committed fn-165.9. Release .10's explicit task admission hold; unsupported cross-spec task edges must not become whole-spec closure gates. Task .10 separates native state data from finite inputs/outcomes/facts. Task .11 adds finite record-field domains and input-reading guards. Task .12 adds native reach/open/checker configuration and retires public Query.total. These are native contributions only; full lifting, engine, carrier retirement, six-sample graph-isomorphism and backend proofs remain open.

Original .2-.9 stay held until all native DSL work finishes. The earlier Go-first exception and IR/interpreter-before-DSL phase order are superseded. Preserve engine, numeric, lifting, graph and backend proofs for later. Scala Int admission alone does not prove checked-int64 or overflow-hole semantics; settle the numeric authoring contract before native DSL completion and retain cross-interpreter parity for later. Do not enumerate whole-state catalogs to admit collection state. Temporal remains excluded. After committed .12, release fn-123.9.

## Conversation Evidence

> user: "I'm considering changing Umpire to be more like Quint there; Umpire should only need to materialize the queries and they should be bounded. I like the simplicity of using scala Map, List and Set. And then relying on our simple model checker or TLC or Apalache to explore the sate."
> user: "wait; what are the full tables really buying us? should we drop this entirely? and then have checks in Go on the Umpire IR to account for some of the resulting gaps?"
> user: "yes rework on this bassis"
> user (decisions D1–D8, 2026-10-13): open graphs "Per Query, accepted"; fingerprint "IR structure only", "Own starts only"; backends "Decide always"; Scala checks "Confirm Go only"; Map "Native partial Map only"; channels "Keep Inbox now, library later"; keys "Structured lookups"; Int "Checked 64-bit"; reach bound "Per machine + default"; totals "Retire everywhere".
> user: "approved steps" (create this spec, mark superseded work in Flow, update MILESTONES.md)

## Goal & Context

Today the reader builds every machine's table over the **whole** state catalog before it answers anything (`tools/umpire/interp/machine.go:529-568`). Refinement, lint, fingerprints, totals, step tables and Case lowering all read that whole table. So every state type must be finite and small. That forces the samples' match-ladder encodings and Voting's invariant-as-guard, and it costs most of the Go check's time. For example, Bank evaluates 6,561 states, of which only 100 are reachable.

This spec drops full tables entirely. The only table is the graph reachable from declared roots, explored on demand within a per-machine reach bound. State may hold native Scala `Map`, `Set`, `List` and `Int`. Static Go checks on the Umpire IR cover the gaps that the catalog-wide checks used to close. Apalache and TLC may decide Query answers. The full design, its inventory, its risks and the before/after samples are in `model/design/reachable-exploration.md`. Its §10.1 records the owner's decisions.

## Architecture & Data Models

- **Roots:** machine starts, Scenario starts and Exploration candidates.
- **Reachable graph:** the breadth-first closure from the roots, with rows computed on demand (`RowsFrom`), within the machine's reach bound (`val reach`, defaulting from `temporal/Bounds.scala`).
- **Closed and open:** a graph is *closed* when exploration ran out of states before the bound, and *open* when the bound stopped it first. Receipts carry `Reached`, `Closed` and `Checker`.
- **Inputs stay finite:** through their types, or `oneOf(...)`.
- **Behavior Fingerprint:** an IR structural hash of behavior-affecting fields, Merkle-hashed with names erased and canonicalized against source order (design §4.13, §5.5).
- **Static Go checks** (design §5):
  - structural rule overlap, which needs per-rule guards in the IR;
  - exhaustive-match, refinement-map and role-closedness coverage;
  - unbounded-growth warnings and a growth report;
  - an optional invariant declared on the refinement;
  - root reporting.
- **Backends that decide** (design §7.3–§7.5): `checkedBy` and `confirm` on a Query; admission enforces export coverage; every backend trace is replayed through Go; a new orchestration module (for example `tools/umpire/answer`) sits above `check` and `export`.

## Acceptance Criteria

- **R1:** Governance amendments are approved under GOV-02 before the phases that need them:
  - the SPEC's "finite types" wording (`.plans/UMPIRE4_SPEC.md:42`);
  - the Behavior Fingerprint's definition (structural hash; every front end must lift identical IR);
  - VISION's "second opinion, never decide" (`.plans/UMPIRE4_VISION.md:101-102`);
  - the module map's new orchestration module.
- **R2:** The reachable graph is the only table. On-demand rows, roots, the reach bound, closed/open, the `open-graph` lint with `.open(because = ...)` acceptance (D1), and receipt fields `Reached`, `Closed` and `Checker`.
  - The IR structural fingerprint replaces the full-table hash (D2), with one regeneration of every IR, Case and fixture.
  - A parity test against the old eager tables proves answers unchanged on every existing Model, and Case bytes unchanged apart from fingerprints.
  - Voting drops `inv`.
- **R3:** Scala-side catalog machinery retires (D4):
  - the rule-overlap walk at construction;
  - `StepTables`, `Equivalence`, `StepLedger` and `InterpreterTables`;
  - the `Coverage.scala` baseline;
  - the equivalence stage of `make model-check-changed` (`ChangeCheck.scala:230-246`);
  - "Sealing a batch" while the IR is stale.

  Query totals retire everywhere (D8): `total` in the DSL and lifter, `ir/totals.go`, and the README's "Counting a Query's total".
- **R4:** The static Go checks of design §5 are implemented, each with tests, and SEMANTICS §6, §7, §9, §13 and §20 describe them.
- **R5:** The IR and interpreter carry `Set`, `Map` (partial; a missing key is a hole, D5a), `List`, and checked `int64` `Int` (overflow is a hole, D6). The four split-on-`-` key sites become structured lookups (D5c). Type order equals today's catalog order on finite types.
- **R6:** The DSL and lifter admit native collections and `Int` in state, `oneOf` input domains, input-reading guards `where((s, i) => ...)` (from bounded-collections §3.4), and the liftable operations of design §6.5. `Inbox` channels are kept (D5b), recognized by type, with no hand-written `given Finite`.
- **R7:** The six framework samples are rewritten natively. Their expectations are unchanged, and a reachable-graph isomorphism test maps each old encoding onto the new one.
- **R8:** Backends decide (D3):
  - export coverage first (refinement, progress, Queries, channels or the channel library, collections);
  - then `checkedBy`/`confirm`, admission of coverage, Go replay of traces, cross-checker agreement and the orchestration module;
  - then the TLA+/TLC route chosen under the design's open question.

## Edge Cases & Constraints

- **Nothing silently weaker:** every result names its scope (closed, or open at depth *d*), and open results need explicit acceptance.
- **Phase 1 changes no answer.** Its only intended diff is the fingerprint, in one reviewed regeneration.
- **D7:** the reach bound is per machine, with a feature-wide default.
- **D6:** Go and the backends differ past 2^63; Go reports `incomplete` there, and such results count as incomparable.
- Real features stay behind for the owner's sweep, under the mini-only contract in `MILESTONES.md`.

## Decision Context

The decisions settled by the owner on 2026-10-13 (D1–D8) are recorded in `model/design/reachable-exploration.md` §10.1. Related designs:
- `model/design/bounded-collections.md`: its input-reading guards and collection operations survive; `Total`, `ListUpTo` and the catalog ceilings do not.
- `model/design/ir-simplification.md`.
- `model/design/samples-vs-quint.md`: its findings #4 and #6 are resolved here.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Governance amendments are approved under GOV-02 before the phases that need them: | fn-173-reachable-exploration-drop-full-tables.1 | — |
| R2 | The reachable graph is the only table. On-demand rows, roots, the reach bound, closed/open, the `open-graph` lint with `.open(because = ...)` acceptance (D1), and receipt fields `Reached`, `Closed` and `Checker`. | fn-173-reachable-exploration-drop-full-tables.12, fn-173-reachable-exploration-drop-full-tables.2 | — |
| R3 | Scala-side catalog machinery retires (D4): | fn-173-reachable-exploration-drop-full-tables.12, fn-173-reachable-exploration-drop-full-tables.3 | — |
| R4 | The static Go checks of design §5 are implemented, each with tests, and SEMANTICS §6, §7, §9, §13 and §20 describe them. | fn-173-reachable-exploration-drop-full-tables.3 | — |
| R5 | The IR and interpreter carry `Set`, `Map` (partial; a missing key is a hole, D5a), `List`, and checked `int64` `Int` (overflow is a hole, D6). The four split-on-`-` key sites become structured lookups (D5c). Type order equals today's catalog order on finite types. | fn-173-reachable-exploration-drop-full-tables.4 | — |
| R6 | The DSL and lifter admit native collections and `Int` in state, `oneOf` input domains, input-reading guards `where((s, i) => ...)` (from bounded-collections §3.4), and the liftable operations of design §6.5. `Inbox` channels are kept (D5b), recognized by type, with no hand-written `given Finite`. | fn-173-reachable-exploration-drop-full-tables.10, fn-173-reachable-exploration-drop-full-tables.11, fn-173-reachable-exploration-drop-full-tables.5 | — |
| R7 | The six framework samples are rewritten natively. Their expectations are unchanged, and a reachable-graph isomorphism test maps each old encoding onto the new one. | fn-173-reachable-exploration-drop-full-tables.11, fn-173-reachable-exploration-drop-full-tables.6 | — |
| R8 | Backends decide (D3): | fn-173-reachable-exploration-drop-full-tables.12, fn-173-reachable-exploration-drop-full-tables.7, fn-173-reachable-exploration-drop-full-tables.8, fn-173-reachable-exploration-drop-full-tables.9 | — |

Native tasks contribute only their stated portions. Held tasks retain the complete IR/integration requirements; coverage is assignment, not completion.

## Supersedes

From the 2026-10-13 impact review:
- the `MILESTONES.md` "seal" row (the mini step-table equivalence);
- fn-157.1/.2/.3/.5 (full-table memory);
- the totals parts of fn-141.12/.13, fn-125.3 and fn-158 R5;
- fn-166's mini step-table baseline.

Specs whose evidence moves from step-table equivalence to fresh-lift IR identity, or which should wait for Phase 1, carry a "Redesign impact" note.
