# Safety and liveness groups for object properties

> HTML render lens: open `.flow/artifacts/fn-149-safety-and-liveness-groups-for-object/spec.html` locally. Regenerable; markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 1): "> object properties:"
> user (turn 1): "in umpire; what if we separated clearly into safety and liveness properties on that level? I think it would be a great help"
> user (turn 2): "let's write a flow next spec (or just todo?) to address this; add it to MILESTONES.md"

## Goal & Context

<!-- Source: [paraphrase] -->
Model authors and reviewers should see safety and liveness separately at the object-properties
level. The distinction makes it easier to review both what a design must prevent and what it
must eventually accomplish.

## Architecture & Data Models

<!-- Source: [inferred] -->
Each machine or composition groups its authored claims under properties.safety and
properties.liveness. Safety contains same-step and transition assertions and references to
safety monitors. Liveness contains the existing bounded progress claims. Shared capability
laws follow the same classification without duplicating their definitions or registration.
Queries continue to select how and where a claim is checked.

## API Contracts

- The authoring checker enforces the group against the claim's declaration kind. It does not
  attempt to infer arbitrary temporal meaning from predicate bodies or declaration names. [inferred]
- Existing diagnostics and claim reports expose the classification and retain the claim's
  source location, bound and assumptions where applicable. Classification is derived from existing
  declaration kinds; no additional classification field is stored in the Model IR.

## Edge Cases & Constraints

- A postcondition saying that a successful worker response leaves an activity completed is
  safety. It does not require the worker to respond. An existential witness is likewise no
  liveness proof. [inferred]
- Liveness currently means bounded progress. Preserve explicit bounds and fairness or recovery
  assumptions; exhausted exploration or missing evidence cannot establish progress. [inferred]
- Helpers are not claims. Empty groups may be omitted, and regrouping must not register a shared
  law or monitor twice. Preserve monitor attachment and Query selection semantics. [inferred]
- Preserve modeled behavior, check results and generated Case meaning. Review and account for
  any source-derived identity or provenance changes caused by moving declarations. [inferred]

## Acceptance Criteria

- **R1:** Authored object properties have distinct safety and liveness groups, visible at the
  declaration site. No error surface beyond the declaration checks in R2. [paraphrase]
- **R2:** The authoring gate rejects a progress declaration in safety and a safety assertion in
  liveness, naming the declaration and expected group. Valid groups and omitted empty groups
  pass; helpers and shared laws do not create duplicate claims.
- **R3:** Liveness declarations retain explicit progress bounds and assumptions. Negative
  fixtures cover a missing or invalid bound and a progress counterexample; incomplete checks
  remain incomplete. No unbounded eventuality claim is introduced.
- **R4:** Existing claim diagnostics and reports distinguish safety from bounded liveness and
  retain source attribution. Progress receipts expose the declared `within` bound separately
  from exploration limits, including successful checks with unequal values. A missing progress prerequisite never suppresses an independently
  established safety violation. Unsupported checks remain explicitly unsupported.
- **R5:** Existing Models, shared capability laws, references and authoring examples use the
  grouping consistently. Regression checks preserve behavior and verdicts, including negative
  controls; generated identity and provenance changes are explicitly accounted for.
- **R6:** Author documentation explains the two groups with a safety postcondition and a bounded
  progress claim, including their different treatment of assumptions. It explicitly distinguishes
  finding a witness from proving progress. No error surface beyond R2 and R3.

## Early proof point

The first task proves that designated nested groups preserve declaration discovery, initialization
and registration, including progress without a Query consumer. If that pin fails, revise the
discovery/export approach before migrating Models.

## Boundaries

- Unbounded temporal operators, new fairness semantics and a new progress-checking algorithm are
  outside this change. [inferred]
- Classification does not change the machine transition relation or create new product promises.
  It does not turn witness Queries into universal claims. [inferred]
- No independent reporting application or new runtime verification protocol is required. [inferred]
- New progress-checking support for compositions belongs to
  [fn-150, bounded liveness across composed machines](fn-150-bounded-liveness-across-composed.md).
  Compositions use the safety grouping here; the liveness grouping accommodates that later extension
  without claiming that composition progress is already supported. [paraphrase]

## Decision Context

- Keep the two groups inside properties so a reviewer finds all promises in one place. Use a
  checked distinction so headings and evaluated claim kinds cannot disagree. [inferred]
- Capture this as one cohesive spec. Validation, reporting and migration make a bare reminder
  insufficient to preserve the intended distinction. [inferred]
- The conductor selected fn-149 after fn-140 and before fn-164 expectation/colon migration. Activation
  and readiness remain conductor-owned. Entry follows fn-140.10's committed witness source seal;
  fn-149.7's committed grouping source seal releases fn-164, whose final seal precedes fn-123's semantic fault changes. Fn-141's declaration
  lifting runs later in Batch 4 and supplies no prerequisite exporter here.

- Planning confirmed R2-R6 against existing declaration admission, receipt subjects, Model pins
  and bounded-progress semantics. The planning pass removes their unconfirmed tags; it adds no
  stronger temporal guarantee.
- Only the designated safety/liveness groups participate in discovery. Aliases and factory results
  are checked by their resolved declaration kind. Monitor definitions and attachments retain their
  existing home; a grouped reference does not attach or register a monitor again.
- The first tasks may temporarily accept flat declarations. The Model migration activates the
  final layout refusal, so intermediate tasks remain buildable and the completed surface is uniform.
- This wiring does not change scheduling metadata or readiness. Use committed source
  seals within Batch 2, with no fn-140 or fn-149 whole-spec dependency that would deadlock the shared
  close. Re-anchor to the actual fn-155 identity mapping, fn-156 enforcement and fn-140 witness,
  `when` and `.live` vocabulary at execution. Their future interfaces are not presumed complete.
- Retain the decision in `.flow/memory/declined/generated-api-drift-verification.md` against broad
  generated-API drift verification and new CI coverage; use focused fixtures and existing gates.

## Selected Batch 2 execution

The source order is committed fn-140.10 -> fn-149.1/.2/.6/.7 -> fn-164 expectation/colon migration
-> the explicit pre-fault step-table comparison -> fn-123. The conductor verifies committed source
seals before each admission. A failed or incomplete source seal holds downstream work.

Task .6 freezes the complete declaration-side baseline before grouping. Task .7 includes final
documentation and executable-layout changes in the independently compared source/scratch-lift
seal. Preserve native tables and predicates on every applicable row, Progress bounds/assumptions,
Query and expected-run metadata, realization membership, monitor attachment and capability origins/
waivers. Account separately for each qualified name, identity and source span. Step-table projection
does not prove provenance equality. Compose actual fn-140/fn-155 mappings and reject unexplained
differences. Whole-group omissions and shared laws omitted by both candidate scanners must fail
the independent frozen inventory.

The seven-task DAG preserves the original consumer tasks. The single-worker Scala lane is
.1 -> .2 -> .6 -> .7. Task .3 is the existing deferred Go reporting checkpoint; its in-progress
state and remaining obligations stay unchanged. Legacy .4 retains its original complete migration,
Go receipt, Case and assessment contract and adds .6 as a dependency. Legacy .5 retains integrated
consumer documentation/regeneration/gates/review/live closure and adds .7 as a dependency.
Neither task closes from native/source evidence.

Scala admission consumes committed fn-155/fn-156 source receipts and identity mappings. Their full
fn-155.6/fn-156.7 closures remain explicit prerequisites for deferred .4/.5 instead of whole-spec
dependencies blocking the owner-approved source lane. No source receipt counts as those closures.
Go, managed regeneration, full gates, independent review and live work stay held until the owner
lifts the mode. The shared fn-123.8 boundary supplies every retained consumer result and publishes
changed pins once. Publishing remains separately authorized.

The .1/.2 source proofs cover discovery, native initialization/registration, resolved declaration
kinds and compiler/lifter bound admission. Actual Query/progress check outcomes, finite-bound
counterexamples, incomplete/unsupported dispositions, unequal declared/search bounds, independent
safety failures despite missing progress prerequisites, canonical Cases/checksums and expected/
actual live/replay assessments stay in .3/.4/.5. Strict inherited Activity and resource failures
retain their original attribution and receive no green credit.

### Planning evidence

Current runtime discovery and TASTy discovery require independent unqueried-Progress pins.
Both capability scanners need the fixed seven-companion/thirteen-law inventory rather than
candidate-derived counts. The dedicated PropertyGroupsFixtures suite avoids the broad Fixtures
startup/golden campaign and must execute nonzero expected populations. Existing source StepTables
supply table equivalence, with a separate exact identity/provenance ledger.

Memory guidance retains exact authored coordinates and independent completeness universes
(`nested-admission-diagnostics-must-2026-09-05`, `a-checked-artifact-cannot-supply-its-2026-09-28`).
Source semantics must lift faithfully or refuse unsupported forms
(`channel-catalogs-and-visible-results-2026-09-30`). These entries guide the proof; they are not
completion evidence. The existing declined generated-API drift/CI scope remains excluded.

## Quick commands

```bash
mise exec -- scala-cli test model/irgen --test-only '*PropertyGroupsFixtures*'
go test -tags test_dep ./tools/umpire/check ./tools/umpire/internal/cli
```

Task .7 commits the Scala grouping seal and author documentation. Legacy .4/.5 retain integrated
model, artifact, lint, full Go and live obligations at the single fn-123.8 shared close. This planning-only wiring requires the conductor to verify the committed fn-140.10 source seal before admission; it neither executes these commands nor grants
passing-gate credit.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Authored object properties have distinct safety and liveness groups, visible at the declaration site. No error surface beyond the declaration checks in R2. | fn-149-safety-and-liveness-groups-for-object.1, fn-149-safety-and-liveness-groups-for-object.2, fn-149-safety-and-liveness-groups-for-object.4, fn-149-safety-and-liveness-groups-for-object.6 | — |
| R2 | The authoring gate rejects a progress declaration in safety and a safety assertion in liveness, naming the declaration and expected group. Valid groups and omitted empty groups pass; helpers and shared laws do not create duplicate claims. | fn-149-safety-and-liveness-groups-for-object.1, fn-149-safety-and-liveness-groups-for-object.2, fn-149-safety-and-liveness-groups-for-object.4, fn-149-safety-and-liveness-groups-for-object.6 | — |
| R3 | Liveness declarations retain explicit progress bounds and assumptions. Negative fixtures cover a missing or invalid bound and a progress counterexample; incomplete checks remain incomplete. No unbounded eventuality claim is introduced. | fn-149-safety-and-liveness-groups-for-object.2, fn-149-safety-and-liveness-groups-for-object.4, fn-149-safety-and-liveness-groups-for-object.5, fn-149-safety-and-liveness-groups-for-object.6, fn-149-safety-and-liveness-groups-for-object.7 | — |
| R4 | Existing claim diagnostics and reports distinguish safety from bounded liveness and retain source attribution. Progress receipts expose the declared `within` bound separately from exploration limits, including successful checks with unequal values. A missing progress prerequisite never suppresses an independently established safety violation. Unsupported checks remain explicitly unsupported. | fn-149-safety-and-liveness-groups-for-object.3, fn-149-safety-and-liveness-groups-for-object.4, fn-149-safety-and-liveness-groups-for-object.5 | — |
| R5 | Existing Models, shared capability laws, references and authoring examples use the grouping consistently. Regression checks preserve behavior and verdicts, including negative controls; generated identity and provenance changes are explicitly accounted for. | fn-149-safety-and-liveness-groups-for-object.4, fn-149-safety-and-liveness-groups-for-object.5, fn-149-safety-and-liveness-groups-for-object.6, fn-149-safety-and-liveness-groups-for-object.7 | — |
| R6 | Author documentation explains the two groups with a safety postcondition and a bounded progress claim, including their different treatment of assumptions. It explicitly distinguishes finding a witness from proving progress. No error surface beyond R2 and R3. | fn-149-safety-and-liveness-groups-for-object.5, fn-149-safety-and-liveness-groups-for-object.7 | — |
