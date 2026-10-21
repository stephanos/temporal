# Cases are produced in Scala

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user (2026-10-18): "have a temporal-specific adapter in model/temporal/ that converts the model tests/queries into TestPilot IR; then we could rename tools/umpire to tools/testpilot and have that Go code only deal with that part?"
> owner (2026-10-18): approved filing; Scala emitting the Testpilot Case proto was already the selected future producer in the measurement decision.

## Goal & Context

Case lowering turns a find Query's witness, through the realization its machine declares, into a Testpilot Case: a Program of instructions, a Contract of correlated clauses, identities and fingerprints, inside a managed Case tree. Today this is Go over the IR with a second Model interpreter underneath. The measurement decision selected Scala emitting the Case proto as the producer. This spec builds it in two halves along the existing framework/Temporal line: the generic producer in the framework (witness to Program, Property to Contract, fingerprints, trees, candidate enumeration, bridge server) and the Temporal adapter under `model/temporal` (descriptors, attempt and reset semantics, settlements, waits, rejection codes). Byte equality against the checked-in Cases gates the switch before the one-time recut that fn-201's fingerprints require.

## Architecture & Data Models

- The checker exposes the graph publicly: rows from a state (every result of every class, in order, with choice name), and path values that carry Definition IDs from the lift. `Checker.find` witnesses are the lowering input; the pinned schedule is the `Space` selector.
- Generic producer: Program placement of path actions and entrypoints; Contract clauses from Properties (a trigger plus state, outcome and fact requirements); correlated clauses and projection plans; localized Definition IDs; fingerprints over `Lifted.canonical`; managed Case trees (model, functional, canary) with manifest, selection, sync and pin checks; candidate enumeration over typed `Variation`s and drop-prefix reduction; the NDJSON bridge server.
- Property to clauses needs an inspectable claim: either a finite enumeration of the claim over `Finite` domains for state, outcome and fact, or a claim-section seam that exposes the requirement structure. The spec picks enumeration as the default and records the seam as the fallback.
- Temporal adapter: evidence messages and field paths from ScalaPB descriptors (the `FieldSource` macro already spells `oneof<field>`), activity attempt and reset settlement, external settlements, read waits from behavior hints, rejection codes and exhaustive evidence rules. These come from the realization kits that are currently disabled.
- Output is the Case proto through the ScalaPB API jar; nothing writes IR.

## API Contracts

- `Checker.rows(instance, state)` returns ordered results per class with choice names; `PathStep` carries the action's, state's and outcome's Definition IDs.
- `Cases.lower(suite, query): Either[Standing, Case]` where `Standing` is realizable, no realization, unsupported with reason, or resource limit; the Case is the proto message.
- `Cases.tree(kind).generate / check / sync` mirror the Go tree commands; byte-for-byte equality is the check.
- `Explore.candidates(query)` and `Explore.reduce(path)`; `Bridge.serve(in, out)` speaks the campaign and replay protocol frames defined by the Go coordinator.
- Fixture gate: every managed Case produced by Scala equals the Go-produced one byte for byte before fn-201's fingerprint recut; after the recut, the trees are regenerated once and the Go producer is no longer run.

## Edge Cases & Constraints

- Orders, identities and spellings are compared with checked-in fixtures; any difference before the recut is a defect, not a formatting choice.
- A witness found before a budget cut lowers; a Query answered unknown has standing resource limit.
- The bridge must honour the coordinator's frame limits and the outstanding-candidate rule.
- The Temporal adapter compiles only when Temporal sources are re-enabled; the generic producer is proven on the samples' realizations first.

## Acceptance Criteria

- **R1:** The checker exposes ordered rows per state and class with choice names, and path values carry Definition IDs. Errors: rows of a state outside the explored graph refuse naming the state.
- **R2:** The generic producer lowers a found witness with a realization into a Case proto: Program placement, Contract clauses, correlated clauses, localized IDs, fingerprints over `Lifted.canonical`. Errors: no realization, unsupported realization and resource limit are standings, never exceptions.
- **R3:** Property to Contract clauses works for every Property the samples' realizations declare, by finite enumeration; a Property outside the enumerable form is unsupported with the reason. No error surface beyond that.
- **R4:** Managed Case trees generate, check and sync for the three kinds with manifest and pins; before the recut every Case equals the Go output byte for byte; after it the trees regenerate once. Errors: a tree with a stale pin fails naming the Case.
- **R5:** The Temporal adapter admits descriptors, attempt and reset settlement, external settlements, waits, rejection codes and exhaustive evidence with the refusals the Go admission has today. Errors: each refusal names the realization and the rule.
- **R6:** Candidate enumeration, drop-prefix reduction and the bridge server pass the existing exploration and replay bridge checks with the Go coordinator unchanged. Errors: protocol violations are reported as the coordinator's error frames.
- **R7:** The Go `lower`, `lower/internal/producer`, `explore` and `realization` packages and `umpire-gen-cases`, `umpire-ir-bridge` are no longer run by any gate (deletion is fn-208). No error surface.

## Boundaries

- No change to the Testpilot runtime or Driver.
- No conformance descriptor (fn-204).
- No Temporal Model migration beyond what the adapter needs to compile; Temporal sources stay disabled until the owner's sweep.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

Keeping the Go lowerer would keep the Go checker, because `NewProducer` runs `check.Check` and reads tables, witnesses and identities from it; the measurement note says it cannot lose one call. Producing in Scala removes the second interpreter and lets types replace about a hundred protoreflect sites. The hard parts are the byte-frozen fingerprints, now owned by fn-201, and Property clause lowering, which has no inspectable predicate in the native claim form; finite enumeration is chosen because `Finite` domains exist for every lowerable value, with a claim seam as the recorded fallback. Depends on fn-201 (lift and fingerprints), fn-202 (schema and ScalaPB jar) and fn-195 (ModelSuite owns conformance tests). Absorbs the remainder of fn-164.3 (expectation bridges). Filed 2026-10-18 as part of the one-language series (fn-199 through fn-208).

2026-10-18, one-language series: absorbs by reference: fn-160 R15 (drop `scala.` from Case identities, in the one-time identity recut), fn-166.6's Case-tree layout under `generated/cases/`, fn-148.7's format-4 artifact migration (merged into .4), fn-123.6 and fn-129.3-.5 lowering refusals (Temporal adapter, .5), and the lowering halves fn-146.6, fn-147.2, fn-148.2/.3/.5/.6 (Temporal adapter, .5). fn-195.11's producer-facing native selector is .1's input.

2026-10-18, target boundary (owner): on the fn-209 go branch the producer emits the Program as data for the Model side to replay and the Case identity; R3 (Property to Contract clauses) closes as unnecessary and the generic Go runtime interprets nothing. On no-go the spec stands as filed.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.
