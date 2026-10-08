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
- Coordinate implementation with fn-140's property/Query authoring changes and fn-141's declaration
  lifting changes. Re-anchor to the completed vocabulary and schema before implementation; this
  spec does not change the approved delivery order. [inferred]

- Planning confirmed R2-R6 against existing declaration admission, receipt subjects, Model pins
  and bounded-progress semantics. The planning pass removes their unconfirmed tags; it adds no
  stronger temporal guarantee.
- Only the designated safety/liveness groups participate in discovery. Aliases and factory results
  are checked by their resolved declaration kind. Monitor definitions and attachments retain their
  existing home; a grouped reference does not attach or register a monitor again.
- The first tasks may temporarily accept flat declarations. The Model migration activates the
  final layout refusal, so intermediate tasks remain buildable and the completed surface is uniform.
- No new hard dependency was found in either direction. Execution remains unscheduled and outside
  the approved activity batch; serialize overlapping source and regeneration work with the existing
  delivery chain. Task paths must follow completed package, schema and exporter moves.
- Retain the existing decision against broad generated-API drift verification and new CI coverage;
  use focused fixtures and existing gates.

## Quick commands

```bash
mise exec -- scala-cli test model/irgen --test-only Fixtures
go test -tags test_dep ./tools/umpire/check ./tools/umpire/internal/cli
```

The final task owns the integrated model, artifact and lint gates and the single full Go run.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Authored object properties have distinct safety and liveness groups, visible at the declaration site. No error surface beyond the declaration checks in R2. | fn-149-safety-and-liveness-groups-for-object.1, fn-149-safety-and-liveness-groups-for-object.2, fn-149-safety-and-liveness-groups-for-object.4 | — |
| R2 | The authoring gate rejects a progress declaration in safety and a safety assertion in liveness, naming the declaration and expected group. Valid groups and omitted empty groups pass; helpers and shared laws do not create duplicate claims. | fn-149-safety-and-liveness-groups-for-object.1, fn-149-safety-and-liveness-groups-for-object.2, fn-149-safety-and-liveness-groups-for-object.4 | — |
| R3 | Liveness declarations retain explicit progress bounds and assumptions. Negative fixtures cover a missing or invalid bound and a progress counterexample; incomplete checks remain incomplete. No unbounded eventuality claim is introduced. | fn-149-safety-and-liveness-groups-for-object.2, fn-149-safety-and-liveness-groups-for-object.5 | — |
| R4 | Existing claim diagnostics and reports distinguish safety from bounded liveness and retain source attribution. Progress receipts expose the declared `within` bound separately from exploration limits, including successful checks with unequal values. A missing progress prerequisite never suppresses an independently established safety violation. Unsupported checks remain explicitly unsupported. | fn-149-safety-and-liveness-groups-for-object.3, fn-149-safety-and-liveness-groups-for-object.5 | — |
| R5 | Existing Models, shared capability laws, references and authoring examples use the grouping consistently. Regression checks preserve behavior and verdicts, including negative controls; generated identity and provenance changes are explicitly accounted for. | fn-149-safety-and-liveness-groups-for-object.4, fn-149-safety-and-liveness-groups-for-object.5 | — |
| R6 | Author documentation explains the two groups with a safety postcondition and a bounded progress claim, including their different treatment of assumptions. It explicitly distinguishes finding a witness from proving progress. No error surface beyond R2 and R3. | fn-149-safety-and-liveness-groups-for-object.5 | — |
