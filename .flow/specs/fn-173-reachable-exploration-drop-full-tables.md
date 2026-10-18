# Reachable exploration: drop full tables, native collections, backends that decide

## Current admission order (owner override, 2026-10-16)

Native checkpoints .10-.17 are committed; fn-188.3 is complete with formal native-batch review SHIP. Native Duration coverage/guidance follow-up fn-147.5 is also complete (SHIP). The audit of all open specs and remaining milestone tasks found no additional currently authorized native DSL gap; do not invent work or revive deferred proposals. The native sequence below records completed implementation order, not instructions to reopen those tasks. Build/export, IR/capture, schema, Go and backend work remain separately held; neither native closure nor fn-147.5 completion releases them automatically. Temporal remains entirely excluded, including disabled source contents. Native results do not close full parent requirements or deferred integration proofs.

> HTML render lens (local): open `.flow/artifacts/fn-173-reachable-exploration-drop-full-tables/spec.html` - regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Native DSL task order (owner override)

Execute .10, .11, then .12 after committed fn-165.9. Release .10's explicit task admission hold; unsupported cross-spec task edges must not become whole-spec closure gates. Task .10 separates native state data from finite inputs/outcomes/facts. Task .11 adds finite record-field domains and input-reading guards. Task .12 adds native reach/open/checker configuration and retires public Query.total. These are native contributions only; full lifting, engine, carrier retirement, six-sample graph-isomorphism and backend proofs remain open.

Original .2-.9 stay held until all native DSL work finishes. The earlier Go-first exception and IR/interpreter-before-DSL phase order are superseded. Preserve engine, numeric, lifting, graph and backend proofs for later. The approved design keeps native Scala `Int` as authoring syntax; checked-int64 and overflow-hole semantics belong to the interpreted IR, not to direct Scala execution. Native tests must document their actual host boundaries and are authoring/runtime smoke checks, not a semantic checking oracle. No numeric wrapper, alias or new arithmetic surface is implied. Do not enumerate whole-state catalogs to admit collection state. Temporal remains excluded. After committed .12, release fn-123.9.

The numeric boundary follows design §§6.4, 6.5, 9.6 and decision D6: Scala no longer checks Models by executing their callbacks or walking their state space. Admitting `Data[Int]` earns no checked-int64 proof. The held lifting/interpreter work must establish checked arithmetic, located overflow holes and preservation or explicit refusal of host-folded numeric expressions and initialized constants; native wrapping values must never silently stand in for a checked interpretation. Cross-interpreter boundary parity remains open with R5/R6.

### Native source-tail contract (conductor, 2026-10-16)

After committed fn-123.9/.10 and fn-153.7, finish the remaining native source contributions before IR: .13 Bank/TwoPhaseCommit, .14 Voting/ReliableBroadcast, .15 TwoLayeredCache/Kettle, then .16 native catalog-walk retirement. Task .13 has an explicit admission hold on committed fn-153.7, not whole-spec closure. Later checkpoints consume the shared retained fixture/map/expectation bundle and updated native cases from their predecessor. Original .2-.9 remain held with their existing integration edges.

These are existing R7/R3 source obligations, not new DSL features. Retain all six old sample sources as durable non-production text fixtures at one committed baseline before the first rewrite. Preserve old positive/negative machines, starts, input domains/order, Property/monitor/progress associations, Query names/forms/bounds and intended answers. Maintain an explicit state-map and intentional-delta inventory through the three pairs. Native construction, dispatch and handwritten Property/history cases prove only those examples; the lift-and-Go sample harness, full Query answers, checked-int64, operation liftability and complete graph proof remain held. Fixtures/maps are future proof inputs, not proof results.

Voting removes the actual inv guards from increaseMaxBal and normal voteFor while preserving its invariant Property. The unsafe vote already lacks that guard; the negative control also inherits the normal max-ballot path. Record each real change and the intended reached-invariant premise, not arbitrary old/new catalog-row equality. Cache history distinguishes states the old summary merges, so no blanket product-graph bijection is asserted. Original .6 must reconcile the exact underlying/product graph scope, finite Query bounds, erased history information and intended answers before any R7 graph credit; documenting a projection does not discharge isomorphism.

Cache history policy: keep existing protocol guards/bounds, actions, facts and end condition. Each passive monitor uses immutable ordered observations carrying the original cache-event kind and observed Option[Int] value. Append one write observation for a wrote step, otherwise one read observation for a read/miss step, using after-state l1 as the old monitors do; preserve the old write-first event priority. Silent expiry adds no event. Preserve repeated observations and introduce no history cap or mutable global log. Both monitors may share the pure history-update function, while their own verdicts express the original readAfterWrite/writeAfterWrite rules over history and retain after-every-step observation. Pin repeated reads/writes, expiry and the stale-read negative control independently. This is a representation contract, not a new checking interface.

Native retirement removes the Scala StepTables producer/CLI and independent refinement/law catalog walks, preserving genuine concrete binding-order, discovery, Hole and callback controls without a MachineLaws.check semantic facade. Disabled Temporal consumers do not constrain this source surface and are neither inspected nor repaired. Retain Refinement/declaredBy, pure textual comparison/RenameMap/lift comparison and the private total None bridge where held integration still consumes them. Original .3 retains remaining total transport, orchestration/consumer reconciliation and Go static replacements. Historical ChangeCheck/Makefile equivalence stages become unavailable source gates after producer retirement, not passing seals. Removed universal checks remain explicit proof debt.

The first broad native batch after committed .16 compiled and executed 451 tests: 449 passed and two existing fixtures failed. Native follow-up .17 reconciles those demonstrated mismatches before the remaining batch gates: derived initialization stays lazy during metadata work and memoizes only when explicitly read; an inferred assumption's actual declaration spelling must agree with retained metadata, without inventing a source identity or relaxing refusals. Preserve the original ID/name/fairness assertions and strengthen callback/init nonexecution controls. This bounded fixture correction contributes only native R6 compatibility; no new public interface, IR proof or whole-spec closure follows.

Alternatives: keeping these source changes behind mixed IR tasks leaves approved authoring work unfinished; pulling the mixed proofs forward changes the native-first phase boundary. Separate source checkpoints preserve both obligations. Research is grounded in committed source/canonical specs; no new external library or API is introduced, so SHORT planning skips web scouts. Source, dependency and gap scouts plus active memory retrieval inform task boundaries. Relevant memory separates native admission from lifting, diagnostic metadata from claim evaluation, and bounded-work checks from catalog allocation; none supplies a native graph proof.

### Quick commands for native source checkpoints

```bash
mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only 'framework.dsl.SampleExpectTest'
mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'
```

Concrete new-suite selectors are task-local and must be re-anchored after the committed predecessor. Never substitute Samples' lift-and-Go harness, old tables or a new Scala explorer for these native gates.

### Native ordering clarification (conductor, 2026-10-16)

Structural `Data.compare` is independent of `Finite.values` and must not enumerate a catalog. Explicit finite domains retain their authored enumeration order for action classes, defaults and domain iteration; do not sort them when introducing Data. The existing `Finite.of(2, 0)` control in `InputRecords.test.scala` demonstrates that an arbitrary custom catalog need not equal numeric type order. Design §6.3's catalog/type-order equivalence applies to the listed canonical derived catalogs, not every explicitly authored finite sequence.

Unordered Inbox canonicalization follows the approved structural type order, not the selected finite message catalog's indices. For a custom message catalog `[2, 0]`, canonical contents therefore order those values `[0, 2]` while action enumeration remains `[2, 0]`. Record this explicit native representation delta with independent pins; do not claim unchanged arbitrary custom-channel keys or IR parity. FIFO order, duplicates, capacity and channel ownership remain unchanged. The later interpreter/backend work must reconcile this same boundary; none of its key/parity proofs closes through native tests.

### Native query configuration policy (conductor, 2026-10-16)

The following resolves unspecified native declaration details under the owner's instruction to follow conductor recommendations. These are current policy choices, not historical owner-approved numeric bounds or evidence of exploration adequacy.

- Reach reuses immutable `Limits`. The framework-owned default is named `frameworkReach`, with `steps = 256`, `actions = 0`, `search = 65536`. Machine and Composition use the same default/override contract. Native reach consumption accepts `steps >= 0`, `search > 0`, `actions >= 0`; zero steps means initial-state-only exploration. Only steps/search govern reach. Actions is retained unchanged for Limits compatibility. Ordinary Query Limits keep their existing validation rules. The later engine must establish adequacy or report open regions.
- Derived/built machines and compositions inherit source reach unless intentionally overridden. Configuration follows the Scenario owner, including a Property from a refined owner. Attaching or inspecting metadata does not evaluate model callbacks, search, or read an overridden reach val before owner initialization.
- Native checker choices are Umpire, Apalache and TLC, independent of backend availability. Defaults are Umpire primary, no confirmations and no open acceptance. Open requires a nonblank reason retained exactly. Open and checkedBy are each single-assignment; repeated calls reject even when equal.
- Confirm is one nonempty, ordered, duplicate-free assignment excluding the effective primary. It fixes that primary, including default Umpire. Select a nondefault primary with checkedBy before confirm; checkedBy after confirm rejects with an ordering diagnostic. Repeated confirm rejects rather than appending. Unrelated open/live/explore and Witness refinements preserve configuration; checker setters are not claimed to commute universally.
- Every Query/Witness copy preserves unrelated declaration metadata and Witness-specific fluent return types. Public total writers, getters and compatibility aliases disappear. Any private carrier bridge needed for compilation remains explicitly held integration debt, not total retirement everywhere or a checked answer.

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

- **R1:** *(2026-10-18, one-language series: governance: the SPEC/Vision amendments move to fn-208.4)* Governance amendments are approved under GOV-02 before the phases that need them:
  - the SPEC's "finite types" wording (`.plans/UMPIRE4_SPEC.md:42`);
  - the Behavior Fingerprint's definition (structural hash; every front end must lift identical IR);
  - VISION's "second opinion, never decide" (`.plans/UMPIRE4_VISION.md:101-102`);
  - the module map's new orchestration module.
- **R2:** *(2026-10-18, one-language series: superseded by reference: the reachable graph is the Scala checker's (fn-196); the open-graph lint is fn-200; the fingerprint is fn-201.4/fn-203)* The reachable graph is the only table. On-demand rows, roots, the reach bound, closed/open, the `open-graph` lint with `.open(because = ...)` acceptance (D1), and receipt fields `Reached`, `Closed` and `Checker`.
  - The IR structural fingerprint replaces the full-table hash (D2), with one regeneration of every IR, Case and fixture.
  - A parity test against the old eager tables proves answers unchanged on every existing Model, and Case bytes unchanged apart from fingerprints.
  - Voting drops `inv`.
- **R3:** *(2026-10-18, one-language series: retained; done natively under fn-173.16)* Scala-side catalog machinery retires (D4):
  - the rule-overlap walk at construction;
  - `StepTables`, `Equivalence`, `StepLedger` and `InterpreterTables`;
  - the `Coverage.scala` baseline;
  - the equivalence stage of `make model-check-changed` (`ChangeCheck.scala:230-246`);
  - "Sealing a batch" while the IR is stale.

  Query totals retire everywhere (D8): `total` in the DSL and lifter, `ir/totals.go`, and the README's "Counting a Query's total".
- **R4:** *(2026-10-18, one-language series: superseded by reference: refinement invariant to fn-199, overlap/coverage/growth lint to fn-200, totals deleted by fn-208)* The static Go checks of design §5 are implemented, each with tests, and SEMANTICS §6, §7, §9, §13 and §20 describe them.
- **R5:** *(2026-10-18, one-language series: superseded by reference: collections are native (fn-173.14/.15); the checked int64 contract is decided in fn-201.5)* The IR and interpreter carry `Set`, `Map` (partial; a missing key is a hole, D5a), `List`, and checked `int64` `Int` (overflow is a hole, D6). The four split-on-`-` key sites become structured lookups (D5c). Type order equals today's catalog order on finite types.
- **R6:** The DSL and lifter admit native collections and `Int` in state, `oneOf` input domains, input-reading guards `where((s, i) => ...)` (from bounded-collections §3.4), and the liftable operations of design §6.5. `Inbox` channels are kept (D5b), recognized by type, with no hand-written `given Finite`.
- **R7:** The six framework samples are rewritten natively. Their expectations are unchanged, and a reachable-graph isomorphism test maps each old encoding onto the new one.
- **R8:** *(2026-10-18, one-language series: superseded by reference: backends decide in fn-206 (Apalache) and fn-207 (TLC) over a direct TLA+ emission, replay through `Checker.replay`; no `tools/umpire/answer` module; fn-205 (Quint) is deferred)* Backends decide (D3):
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

2026-10-18, one-language series: R1's governance amendments go to fn-208.4; the structural fingerprint definition goes to fn-201.4 (hashing `Lifted.canonical`) and fn-203; the orchestration module `tools/umpire/answer` is dropped; R2, R4, R5 and R8 are superseded by reference to fn-199/fn-200/fn-201/fn-206/fn-207. Tasks .2/.3/.4/.7/.8/.9 are closed as obsolete (lane A); .5's lifter part goes to fn-190.9/fn-201 and .6 proves isomorphism on the Scala checker or closes citing fn-195.4 (lane C).

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Governance amendments are approved under GOV-02 before the phases that need them: | fn-173-reachable-exploration-drop-full-tables.1 | — |
| R2 | The reachable graph is the only table. On-demand rows, roots, the reach bound, closed/open, the `open-graph` lint with `.open(because = ...)` acceptance (D1), and receipt fields `Reached`, `Closed` and `Checker`. | fn-173-reachable-exploration-drop-full-tables.12, fn-173-reachable-exploration-drop-full-tables.14, fn-173-reachable-exploration-drop-full-tables.2 | — |
| R3 | Scala-side catalog machinery retires (D4): | fn-173-reachable-exploration-drop-full-tables.12, fn-173-reachable-exploration-drop-full-tables.16, fn-173-reachable-exploration-drop-full-tables.3 | — |
| R4 | The static Go checks of design §5 are implemented, each with tests, and SEMANTICS §6, §7, §9, §13 and §20 describe them. | fn-173-reachable-exploration-drop-full-tables.3 | — |
| R5 | The IR and interpreter carry `Set`, `Map` (partial; a missing key is a hole, D5a), `List`, and checked `int64` `Int` (overflow is a hole, D6). The four split-on-`-` key sites become structured lookups (D5c). Type order equals today's catalog order on finite types. | fn-173-reachable-exploration-drop-full-tables.4 | — |
| R6 | The DSL and lifter admit native collections and `Int` in state, `oneOf` input domains, input-reading guards `where((s, i) => ...)` (from bounded-collections §3.4), and the liftable operations of design §6.5. `Inbox` channels are kept (D5b), recognized by type, with no hand-written `given Finite`. | fn-173-reachable-exploration-drop-full-tables.10, fn-173-reachable-exploration-drop-full-tables.11, fn-173-reachable-exploration-drop-full-tables.17, fn-173-reachable-exploration-drop-full-tables.5 | — |
| R7 | The six framework samples are rewritten natively. Their expectations are unchanged, and a reachable-graph isomorphism test maps each old encoding onto the new one. | fn-173-reachable-exploration-drop-full-tables.11, fn-173-reachable-exploration-drop-full-tables.13, fn-173-reachable-exploration-drop-full-tables.14, fn-173-reachable-exploration-drop-full-tables.15, fn-173-reachable-exploration-drop-full-tables.6 | — |
| R8 | Backends decide (D3): | fn-173-reachable-exploration-drop-full-tables.12, fn-173-reachable-exploration-drop-full-tables.7, fn-173-reachable-exploration-drop-full-tables.8, fn-173-reachable-exploration-drop-full-tables.9 | — |

Native tasks contribute only their stated portions. Held tasks retain the complete IR/integration requirements; coverage is assignment, not completion.

## Supersedes

From the 2026-10-13 impact review:
- the `MILESTONES.md` "seal" row (the mini step-table equivalence);
- fn-157.1/.2/.3/.5 (full-table memory);
- the totals parts of fn-141.12/.13, fn-125.3 and fn-158 R5;
- fn-166's mini step-table baseline.

Specs whose evidence moves from step-table equivalence to fresh-lift IR identity, or which should wait for Phase 1, carry a "Redesign impact" note.
