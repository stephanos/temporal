# Orchestration recommendation: key-level query and composition seams before task 3

Status: proposed. No Flow ID is minted, no dependency is edited, and no test was run.

## Decision

**Not parallel. Consolidate into one L serial prerequisite with two phases**:

- **Phase A**: key-level query surface and unknown-pair contract (`key-level-query-proposal.md`).
- **Phase B**: key-level bounded composition (`key-level-composition-proposal.md`).

Both phases touch `model/go/umpire/**`. Phase B consumes types and semantics that Phase A defines.

## Why the two proposals cannot run in parallel

Four couplings rule it out, all grounded in reviewed source:

1. **Shared `Table`/`TableSpec` type.** Phase A adds `TableSpec.Unknown` and `Table.Unknown` with an `UnknownFrom` index. It also adds a cached table-backed `Model`, because `Query.check` compares models by interface identity (`claims.go:260`). Phase B adds `TableSpec.RefinedField`, since `composedFields` reads the private `refinedField` (`compose.go:541-552`, `table.go:59`). It also emits composed `Unknown` pairs and needs the same `Model` handle. Both phases edit `table.go` (`:41-72`, `:246-279`), so disjoint Touches cannot be named.
2. **Shared unknown-pair semantics.** Phase B must propagate member unknown pairs into composed pairs, with disabled dominating unknown and unknown dominating enabled. That propagation only means something under Phase A's search and progress reading of `Unknown`: explored versus unexplored, and never downgrading a violation. Defining either phase alone would fix half a contract.
3. **Shared error channel.** Phase B's `Ends` callback and replacement errors must survive with their type (`*goir.Hole`, `*RefinementError`). That relies on Phase A's `Unwrap` on `umpire.Error`, which has none today (`table.go:228-237`), and on the change that stops the `%v` flattening at `search.go:245-246`.
4. **Shared consumer path.** A key-level Query over a composed table reads `Result.Step` as `ComposedStep`, which Phase B produces, through Phase A's key-level `observe` dispatch (`search.go:258-282`). The composition acceptance "a verify becomes incomplete only when the composed unknown pair is explored" needs both phases.

The phases can still be implemented and reviewed as internal checkpoints. Phase A goes first, with its tests green, and then Phase B.

## Consolidated L task spec (draft)

**Title:** Key-level claim, query and composition seams over IR-built tables

**Touches:** `[model/go/umpire/**]`

**Size:** L. Phase A is about 8 additive surfaces across 7 files plus the progress unknown rules. Phase B extracts a non-generic composition core from a 577-line file behind byte-identical typed behavior.

**Satisfies (proposed):** R3, R9. It supplies generic support only; task 3 carries R4, R5 and R9 binding.

**depends_on:** task 14.

**Description:** Add additive key-level constructors so IR-built `*Table`s reach the reviewed search, monitor, through-refinement, replay, progress and composition algorithms. Carry holes as explicit unknown pairs that are never read as disabled. Preserve error types across callbacks. Bound composition by a mandatory ceiling. Change no typed constructor, Lean-parity table, Definition ID or fingerprint. Add no feature policy and no IR reading.

**Approach:**
- **Phase A.** Items 1-10 of `key-level-query-proposal.md`: the table `Model`, `Unknown` pairs, `Error.Unwrap`, key-level Property, Scenario, Query and through-refinement, `KeyMonitor`/`AfterKey`, the classifier, explored-unknown accounting and `Incomplete()`, and progress unknown rules. Go test-first per item. Key-level twins of the existing typed tests are the parity oracle.
- **Phase B.** Everything in `key-level-composition-proposal.md`: the shared non-generic composition core, `ComposeTables`, `ComposedStep`, `RefinedField`, unknown propagation, ceilings before allocation, and replacement via `RefineTables(…, CoverStarts:true)`. `compose_test.go` stays unmodified and parity twins are added.
- Preserve existing comments. Do not edit typed tests. Keep every existing message byte-identical.

**Acceptance:** the union of both proposals' acceptance lists, plus:
- [ ] `go test -tags test_dep ./model/go/...` passes, with every pre-existing test unmodified (to be run by the implementer).
- [ ] No exported identifier changes signature or meaning, and no file outside `model/go/umpire/**` changes.

**Quick commands:** `mise exec -- go test -tags test_dep ./model/go/umpire/...` and `make lint-code-fast`.

## Dependency placement

```
task14 (done) ─────────────> NEW-L (Phase A → Phase B) ─┐
task14, task15 (done) ─────> task16 (live) ─────────────┼─> task3 ─> task4, task5
task2, task14, task15 ──────────────────────────────────┘
```

- **New prerequisite.** `depends_on: [task14]`. It extends the reviewed task-14 APIs and needs neither task 15 (goir) nor task 16 source.
- **Task 3.** `depends_on` gains the new prerequisite on top of the existing `[task2, task14, task15, task16]` (`.flow/tasks/…3.json`).
- **Tasks 14 and 15.** Unchanged: both are done and reviewed (R2). Reopening task 14 is unnecessary, because every addition is additive and task 14's acceptance stays true.
- **Task 16.** Its Touches (`model/scalav2/**`, `Makefile`, `model/scala/umpire/*`; `.flow/tasks/…16.md`) are file-disjoint from `model/go/umpire/**`, so the prerequisite may be **implemented** concurrently in an isolated clone. However, task 16's R11 isolation gate runs Go checks over the snapshot, and the spec's quick command includes `./model/go/...`. Join the prerequisite into the root tree only after task 16's gate evidence is recorded and its review resolves. The alternative is `depends_on: [task14, task16]` when no isolated clone is used. Either way, the parent reconciles proposals after the live task-16 wave, as instructed.
- **The `ends`/`visible` Kind guard** (`end_visible_fix.md`) lives in `model/scalav2/goir/**`, inside task 16's Touches. It therefore belongs to task 3, which already runs after task 16. It must not go into the new prerequisite.

## Rejected alternatives

- **Two parallel tasks** (query and composition). Rejected for shared `table.go` Touches and shared unknown and error semantics.
- **Widen task 3's Touches to `model/go/umpire/**`.** This would turn task 3, the IR binding, into generic engine work. It would contradict the parent spec's split, where tasks 14 and 15 own the generic surfaces, and make task 3 roughly XL.
- **Only a conservative interim, no prerequisite.** Task 3 would report explicit `unsupported` for every Query, monitor and composition. That is honest, but it leaves AC1 and AC4 of task 3 and the R3 composition coverage unmet. It is acceptable only as task 3's interim state (`task3-revised-draft.md`).

## Unresolved decisions (normal spec or SEMANTICS decisions; no GOV-02 exception identified)

- **S2.** Do a composition member's monitors watch a composed Query? Interim: located `unsupported` in task 3. Machine-monitor semantics are untouched.
- **S3.** The composed step record is typed as model/go and Scala already read it: a composed state record, with String `<field>_<key>` outcome and facts. Task 3 records this in SEMANTICS.
- **S5.** Claim Definition ID scope. Receipts key claims by (family, machine, local name) inside task 3's adapter. `PropertyID`/`ScenarioID` and their Go/Lean pins stay unchanged.
- **S6.** A hole inside a claim function is unknown evidence: an unknown edge through the Phase A classifier, never `false` or disabled. Task 3 records this in SEMANTICS.
- **Scala `starts.head`** versus Go's all-starts product. Task 16 preserves Scala as-is. Correction is explicit future front-end work.
