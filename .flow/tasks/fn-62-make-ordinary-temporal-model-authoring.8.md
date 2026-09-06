---
satisfies: [R7, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.8 Compose authored Known Gaps through artifacts and Case inputs

## Description
Consume `.6`'s checked Query gaps and add checked phase composition before artifact publication, with exact downstream Case rows. Keep runtime execution unchanged.

**Size:** M
**Files:** Artifact planning, Planning types/engine/KnownGap tests, Case compiler tests, affected planning consumers
**Touches:** [model/Umpire/Artifact/Planning.lean, model/Umpire/Artifact/Types.lean, model/Umpire/Artifact/Tests/**, model/Umpire/Planning/Types.lean, model/Umpire/Planning/Engine.lean, model/Umpire/Planning/Tests/**, model/Umpire/Case/Compiler.lean, model/Umpire/Case/CompilerTests.lean, model/Umpire/Space/**, model/Umpire/Promotion.lean, model/Umpire/PromotionTests.lean, model/Umpire/Examples/Switch.lean, model/Umpire/Tests/MigrationCompatibility.lean, model/Umpire/Exploration/Tests/Validation.lean, model/Temporal/Feature/Nexus/Operations/**, model/Temporal/Feature/Nexus2/**, model/Temporal/Tool/**]

## Approach
- Replace phase-only artifact gap injection with deterministic checked union of Query-authored and phase-owned sets. Reuse `KnownGapSet.union`; preserve row schema and canonical order.
- Implement the exact parent API: composePlanningKnownGaps returns Except KnownGapError KnownGapSet; artifactOfSelection returns Except KnownGapError ExperimentSpec; plan returns Except KnownGapError PlannerRun. Compose once before traversal and pass the checked union into a total private finish/artifact builder, avoiding a second composition on successful selection.
- Add PlanningRequestError with knownGap and artifactIntent variants; planWithArtifactIntent preserves intent-first precedence. Space planPoint maps gap errors to SpaceCompilationErrorKind.knownGapCheckFailed and intent errors to existing intentCheckFailed. Promotion.compilePromotionSource maps outer gap errors to PromotionErrorKind.knownGapCheckFailed before anchored-run comparison. Both higher-level errors retain the complete typed payload, not only text.
- Leave ofFinite/ofCheckedQuery?/ofCheckedQuery, traverseBoundedCandidates, and analyzeCases unchanged. Adapt all direct plan consumers: Switch; Promotion; Space; established Nexus operations; Nexus2 Cancellation/Race; Planning Fixtures/Artifacts/Enumeration; MigrationCompatibility; Exploration Validation; Space intent tests; operation tests; Nexus2 AuthoringTests; artifact codec tests. Inventory additional call sites before editing. This is an atomic signature migration, not a change to their modeled behavior.
- Existing public example `run : PlannerRun` values may remain via adjacent explicit runResult : Except KnownGapError PlannerRun plus author-supplied success evidence/extraction. No hidden proof synthesis, empty fallback, silent artifact omission or error reclassification is permitted.
- Produce exact `Case.Compiler.Input.knownGaps` rows from checked composed gaps and test actual Case compilation. No Go runtime redesign or invented product gaps.
- Preserve default-empty artifact bytes; nonempty authored gaps may change only the named gap-bearing fields/checksum, never behavior fingerprints or selected plans.

## Investigation targets
**Required:**
- `model/Umpire/Artifact/Planning.lean:323` — phase injection.
- `model/Umpire/Planning/Types.lean` — checked gap operations/public results.
- `model/Umpire/Planning/Tests/KnownGaps.lean` — exact gap/error tests.
- `model/Umpire/Case/Compiler.lean:31` — downstream input.
- `model/Umpire/Case/CompilerTests.lean` — actual lowering regression.
- `model/Temporal/Feature/Nexus/Operations/PlanningTests.lean` — exact empty-baseline artifacts.

## Acceptance
- [ ] Empty, authored-only, phase-only, disjoint and exact-overlap sets retain exact rows in canonical order; overlap appears once and a conflicting detail returns visible `KnownGapError` before publication.
- [ ] Compile-time signature fixtures freeze the selected plan/artifact/request result types. Exact conflictingDetail payload returns before traversal even for unsatisfiable/no-selection requests; invalid Query evaluation remains PlanningOutcome.invalid inside .ok. Intent errors precede gap conflicts in planWithArtifactIntent, and Space/Promotion preserve typed gap payloads.
- [ ] Default-empty artifact bytes remain exact; authored additions change only named gap/checksum data and leave outcomes, selected plans and fingerprints unchanged.
- [ ] Malformed/noncanonical external sets and unknown wire categories fail at established boundaries; actual downstream Case input/compiled rows preserve all values without loss.
- [ ] `cd model && mise exec -- lake build Umpire.Planning.Tests UmpireTests Temporal.Feature.Nexus.OperationsTests` passes; every changed publication caller compiles and applicable lint/trust checks introduce no issues.
- [ ] Structural complexity audit identifies exactly one existing checked union per planning/direct-artifact request and one Case conversion pass over composed rows, with no union per row or search candidate. Report unchanged union complexity separately; 10× independent equal-sized requests adds at most 10× new orchestration work.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
