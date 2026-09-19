# Authored variation spaces and deterministic batch compilation

> HTML render lens (local): open `.flow/artifacts/fn-16-authored-variation-spaces-and/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 architecture reconciliation

`Umpire.Space` follows the deep `Umpire.Target` authoring boundary. It produces checked variation/fault/coverage data and deterministic selected semantic intent; it does not freeze the current partial `umpire-experiment/v1` as the final executable format. The current v1 bytes remain compatibility fixtures while `Umpire.Artifact` owns compilation of selected intent into the complete executable `ExperimentSpec` required by fn-18.

The public generation handoff is `umpire-gen-tests`: a named regression, test set, or model-selected batch becomes canonical artifacts. Space itself exposes no competing public command and no runtime semantics.

## Overview

Add one reusable `Umpire.Space` composition layer for finite variation axes, named choices, requested fault intents, and explicit semantic coverage goals. A checked space lowers one canonical assignment or its complete bounded Cartesian product through the existing Behavior and Query checkers and the existing target-owned planner, producing ordinary `umpire-experiment/v1` values with the already-reserved choice, variant, and fault fields populated.

The layer exposes canonical checked metadata for fn-5's semantic catalog. It does not create a second catalog, persisted registry, reader, coverage engine, runtime, or outcome language.

## Goal & Context
<!-- scope: business -->

Model authors need to describe “vary these inputs,” “request these fault attempts,” and “seek these semantic cases” without copying a Query or asserting what the target will do. Success means a small authored space is finite, reviewable, deterministic, and reusable by later exploration while Property stays pure and every result still comes from the authoritative target kernel.

## Architecture & Data Models
<!-- scope: technical -->

```text
Property + Behavior + Query + checked target
                 |
                 v
         Umpire.Space.Language
         axes / choices / faults / goals
                 |
                 v
          CheckedExperimentSpace
             /          \
            v            v
  CheckedSpaceMetadata   lowerSpacePoint
       (fn-5 input)            |
                              v
                 existing Behavior/Query checks
                              |
                              v
                    existing pure planner
                              |
                              v
                   ExperimentSpec / batch
```

`Umpire.Space` is a terminal, Temporal-independent composition package above Behavior, Query, Artifact, and Planning. `DeclarationKind` gains `experiment-space`, `variation-axis`, `choice`, `fault`, and `coverage-goal` so these authored concepts have stable checked vocabulary and can enter fn-5's catalog without a parallel kind system.

An `ExperimentSpaceDeclaration` references one existing checked Query and declares one to eight axes, zero to twelve fault intents, one to sixty-four coverage goals, and the fixed maximum of 256 Cartesian points. The checked value retains the exact base Query/Behavior/target/property identities and semantic digests; it copies no clauses, traces, transition functions, or outcome semantics.

Each axis has two to sixteen canonically ordered choices. A choice has a stable ID, an optional binding of that axis's one existing Behavior role to a checked semantic value, and a canonical set of selected fault IDs. At most one baseline choice with no binding or faults is allowed per axis, and every other choice must have a distinct non-empty effect. Different axes cannot bind the same role. Role bindings to outcome or observation kinds are prohibited. Every referenced value must already be valid for the target and base Behavior.

A fault intent references one named required occurrence already present in the base Behavior, the occurrence's action, and one required target capability. It describes only a requested attempt. It contains no expected outcome, resulting state, observation, receipt, or success assertion. Explicit incompatibility edges are symmetric and checked; duplicate or explicitly incompatible co-selected faults fail before lowering. Fault selection never changes the target kernel or planner result.

A coverage goal has a stable identity, a positive minimum, and one closed subject: an axis/choice pair, a fault, an existing state/action/outcome/observation/relation declaration, or an existing checked Property. Goals are seek-only metadata. Checking proves reference validity and static cardinality feasibility; it does not score traces, filter the batch, accumulate coverage state, or claim a goal was achieved. Later C8 exploration consumes these goals.

`CheckedSpaceMetadata` is the canonical, source-backed semantic projection consumed by fn-5. It exposes the space, axes, choices, faults, goals, exact references, limits, base digests, and one semantic digest. It is an in-memory checked value, not a persisted registry or list/explain implementation.

The first Temporal space is pinned rather than invented during implementation. It is `temporal.nexus.basic-lifecycle.space.fault-matrix`, built on checked Behavior `temporal.nexus.basic-lifecycle.behavior.two-action-lifecycle` and Query `temporal.nexus.basic-lifecycle.query.two-action-lifecycle`. Its exact metadata identities are:

```text
temporal.nexus.basic-lifecycle.axis.start-fault
  temporal.nexus.basic-lifecycle.choice.start-baseline
  temporal.nexus.basic-lifecycle.choice.start-delay
temporal.nexus.basic-lifecycle.axis.completion-fault
  temporal.nexus.basic-lifecycle.choice.completion-baseline
  temporal.nexus.basic-lifecycle.choice.completion-handler-failure
temporal.nexus.basic-lifecycle.fault.start-delay
temporal.nexus.basic-lifecycle.fault.completion-handler-failure
temporal.nexus.basic-lifecycle.coverage.start-baseline
temporal.nexus.basic-lifecycle.coverage.start-delay
temporal.nexus.basic-lifecycle.coverage.completion-baseline
temporal.nexus.basic-lifecycle.coverage.completion-handler-failure
```

The two-action Behavior names `temporal.nexus.basic-lifecycle.occurrence.two-action.start` and `temporal.nexus.basic-lifecycle.occurrence.two-action.succeed`, in that order. Each fault targets the corresponding occurrence and requires the existing lifecycle capability. The four canonical assignments are baseline/baseline, baseline/completion-handler-failure, start-delay/baseline, and start-delay/completion-handler-failure.

## API Contracts
<!-- scope: technical -->

- Checking returns one complete `CheckedExperimentSpace` or one structured first error in canonical identity order. Unchecked or partially checked spaces cannot lower, compile, or supply catalog metadata.
- Fixed v1 bounds are 1–8 axes, 2–16 choices per axis, at most 256 Cartesian points, 0–12 fault intents, 1–64 coverage goals, and coverage minimums 1–256 not exceeding the total point bound. Multiplication detects overflow before materialization.
- Axis, choice, fault, goal, and point ordering is lexicographic by fully qualified canonical identity, independent of authoring order.
- `lowerSpacePoint` accepts one checked space and one complete exact assignment. It derives fresh point Behavior/Query identities from the space identity plus canonical assignment digest, applies role restrictions through existing Behavior checking, and rechecks the derived Query against the unchanged target. It returns a dependent `LoweredSpacePoint` containing that checked Query, a proof `query.target = space.baseQuery.target`, and a checked `ArtifactIntent`. It does not plan or select an outcome.
- `compileBatch` accepts `kernel : IncrementalPlannerKernel space.baseQuery.target`, enumerates every canonical assignment, and calls `lowerSpacePoint`. For each point it transports that same proof-carrying kernel across the returned target equality before invoking the existing planner; it never constructs or copies another kernel. A point that is invalid, unsatisfiable, duplicate, budget-exhausted, verified-without-artifact, or otherwise produces no selected artifact rejects the entire batch at the first canonical point; no partial list is returned.
- The ordinary `plan` API remains unchanged and byte-identical. An intent-aware artifact seam is usable only with checked intent and recomputes DrivePlan and ExperimentSpec semantic identities after projection.
- Artifact format v1 remains unchanged:
  - `selectedChoices` is exactly `{ identity := axis-id, value := choice-id }` for every axis;
  - `selectedVariants` is the canonical list of selected role-binding semantic values;
  - `requestedFaults` is exactly `{ identity := fault-id, value := planned-occurrence-id }` for selected faults; and
  - selected fault capabilities are unioned into `capabilityRequirements`.
- A missing selected occurrence, stale fault/action relation, duplicate derived point/spec identity, or intent inconsistent with the selected trace is a compile failure, never an omission.
- `CheckedSpaceMetadata` is produced from the same checked space and digest as lowering. Fn-5 owns aggregation, glossary/index generation, list/explain, dispositions, aliases, and persistence projections.

## Edge Cases & Constraints
<!-- scope: technical -->

- Empty/singleton axes, empty non-baseline effects, multiple baseline choices, duplicate/case-colliding IDs, equal choice effects, duplicate axis-controlled roles, and incomplete assignments fail checking.
- Unknown or wrong-kind roles, values, faults, capabilities, occurrences, properties, or semantic coverage subjects fail checking.
- Axis bindings that conflict with base setup/target setup, bind target-owned outcome/observation roles, or leave a derived Behavior unsatisfiable fail before planning.
- Faults that name an absent/optional action rather than a required named occurrence, mismatch its action, lack a target capability, or are explicitly incompatible when co-selected fail.
- An axis-choice/fault goal whose minimum exceeds the number of matching Cartesian points is statically impossible. Other semantic subjects are validated and bounded but achievement remains unknown until C8 evaluates selected model traces.
- Query forms or exact traces that cannot be safely role-restricted and rechecked fail with an explicit unsupported-lowering error; the compiler never mutates a checked object in place.
- Equivalent authoring order yields identical checked space metadata, assignments, point identities, planner inputs, artifacts, and output order.
- No IO occurs in checking, lowering, metadata projection, or batch compilation.

## Quick commands
<!-- scope: technical -->

```bash
cd model && mise exec -- lake build Umpire.Space.Tests.Validation
cd model && mise exec -- lake build Umpire.Space.Tests.Compilation
cd model && mise exec -- lake build Umpire.Space.Tests.Metadata
cd model && mise exec -- lake build Umpire.Examples.SwitchTests
cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.VariationSpaceTests
cd model && mise exec -- lake build UmpireTests TemporalModelTests
make umpire-check-regression
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Authors can check finite domain-neutral axes and named choices whose only effects are binding one existing Behavior role to a checked value and selecting declared fault intents. The checker enforces the exact v1 bounds, canonical order, valid role/value kinds, unique controlled roles, at most one baseline choice, and distinct effects. Errors return one canonical typed failure and no checked space. [user]
- **R2:** Faults are first-class request-only declarations that reference one required named action occurrence and a target capability, with explicit symmetric incompatibilities. They never encode outcome, state, observation, receipt, realization, or success. Errors: absent/optional/mismatched occurrences, missing capabilities, duplicate/incompatible selections, or semantic-result fields fail checking or point lowering. [user]
- **R3:** Every space carries 1–64 checked seek-only semantic coverage goals with positive bounded minimums and exact subjects. Static cardinality feasibility is checked for axis-choice and fault goals; other semantic subjects are validated but not claimed achieved. Property remains pure. Errors: duplicate, unknown, wrong-kind, zero/out-of-range, or statically impossible goals fail checking. [user]
- **R4:** One canonical `CheckedSpaceMetadata` projection exposes all space/axis/choice/fault/goal identities, source/version data, base semantic references, exact limits, and a deterministic digest for fn-5. It creates no persisted registry or competing list/explain authority. Errors: stale/mismatched references, source-order sensitivity, or metadata/space digest disagreement fail before projection. [paraphrase]
- **R5:** `lowerSpacePoint` derives one fresh checked Behavior/Query and checked artifact intent through existing checkers and returns proof that the derived Query retains the base target. `compileBatch` accepts the base Query's `IncrementalPlannerKernel`, transports that exact proof-carrying value across each target equality, expands at most 256 canonical points, and atomically returns ordinary `ExperimentSpec` values through the existing planner. Any point error or non-artifact planning outcome rejects the whole batch with the first canonical point identity and no partial output. [user]
- **R6:** Artifact v1 keeps its exact existing wire fields and formats while populating choices, role variants, requested faults, and fault capabilities from checked intent. Ordinary planning remains byte-identical with empty reserved arrays, and every intended artifact identity is recomputed canonically. [paraphrase]
- **R7:** A reusable synthetic two-by-two proof and the exact named `temporal.nexus.basic-lifecycle.space.fault-matrix` declaration produce exactly four ordered specs without authoring an outcome. The Temporal space uses the named two-action Behavior/Query, two exact request-only fault axes, two named start/success occurrences, and four exact coverage goals above. Reordering inputs cannot change metadata or artifact bytes. [user]
- **R8:** The public package, tests, architecture, model walkthrough, and roadmap preserve vertical package boundaries and existing comments. No runtime, evidence, conformance, persisted reader/migration, coverage scoring/state/report, replay, promotion, Go facade, API/config catalog, separate glossary, model-local Makefile, or Umpire3 use is introduced. [user]
- **R9:** Space lowering returns checked selected intent that the Artifact module can compile into a complete executable ExperimentSpec, and named batch generation is exposed through `umpire-gen-tests` rather than a Space-specific command. This criterion supersedes any final-format reading of R6: existing v1 bytes remain compatibility fixtures, not a promise that an incomplete executable schema is permanent. Errors: Space emitting a final persisted schema independently, omitting participant/setup/ordering/termination/cleanup handoff data required downstream, broadening model bounds, or adding a second generation command fails completion.

## Early proof point
<!-- scope: both -->

Tasks `.1`, `.2`, and `.4` must first prove one synthetic two-by-two space yields exactly four canonical `ExperimentSpec`s, populates all three reserved intent arrays across the batch, preserves ordinary-plan bytes, and receives every outcome only from the unchanged target kernel. The proof must reject one bounds overflow, duplicate effect, stale occurrence/capability, impossible goal, incompatible selection, and non-artifact planner point. Failure blocks the Temporal example, catalog metadata handoff, and documentation.

## Boundaries
<!-- scope: business -->

- No second Behavior, Query, Property, target, planner, semantic IR, or outcome language.
- No persisted catalog/registry reader, artifact decoder, compatibility/migration path, list/explain command, glossary generator, or UI; fn-5 owns discoverability and persisted projections.
- No coverage state, scoring, campaign selection, pairwise/t-wise/random algorithm, or report; later C8 work consumes checked goals and `lowerSpacePoint`.
- No runtime, fault injection, receipts, raw/semantic evidence, verdict, replay, minimization, promotion, qualification, or generated Go authoring facade.
- No model-local Makefile and no CI/workflow change.
- No Umpire3 dependency, inspection, invocation, artifact, schema, compatibility, or implementation reuse.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

Finite authored variation closes the remaining C3/C4 language gap and populates already-reserved artifact intent without broadening Property or target semantics. It also gives later exploration one checked input rather than forcing C8 to invent another DSL.

### Implementation Tradeoffs
<!-- scope: technical -->

Fixed small limits favor complete deterministic validation over an open generator language. Baseline choices make fault/no-fault comparison expressible while keeping every other choice effectful. Static feasibility catches goals knowable from the authored Cartesian product; semantic achievement remains later exploration work. Canonical metadata is exposed for fn-5 instead of adding a second registry or discoverability surface.

## References
<!-- scope: technical -->

- `fn-3-umpire-semantic-authoring-and-planning` — checked Behavior/Query, target-owned planner, and ExperimentSpec substrate.
- `fn-9-umpire-reusable-dsl-package-split` and `fn-10-temporal-semantic-model-layout-and` — package purity and Temporal adapter ownership.
- `fn-5-umpire-discovery-promotion-and-artifact` — reverse consumer of canonical checked space metadata.
- `fn-4-umpire-observation-and-semantic-verdicts` — separate future interpreter of fault receipts/evidence.
- `fn-14-milestone-a-pilot-baseline-and-lean` — independent frozen usability/mutation measurement.

## Requirement coverage
<!-- scope: both -->

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Checked axes and choices | `.1`, `.4`, `.5` | — |
| R2 | Request-only fault intents | `.1`, `.2`, `.4`, `.5` | — |
| R3 | Seek-only coverage goals | `.1`, `.3`, `.5` | — |
| R4 | Canonical metadata for fn-5 | `.3`, `.5`, `.6` | — |
| R5 | Point lowering and atomic batch | `.2`, `.4`, `.5` | — |
| R6 | Existing artifact v1 intent fields | `.2`, `.4`, `.5` | — |
| R7 | Synthetic and Temporal examples | `.5`, `.6` | — |
| R8 | Verification, docs, boundaries | `.1`–`.6` | — |
| R9 | Checked intent and complete-artifact handoff | `.1`, `.2`, `.4`, `.6` | — |
