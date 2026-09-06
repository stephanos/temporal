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
Implemented checked authored-plus-phase Known Gap publication end to end: planning composes once before traversal, artifact and request APIs return typed errors, Space/Promotion/Nexus2/discovery/inspection consumers preserve full typed failures, and Case lowering performs one exact order-preserving conversion pass. Exact composition, conflict precedence, payload, Case-row, byte-stability, selected-plan, fingerprint, and caller migration regressions are included.

Verification: baseline and final exact Quick builds passed 203/203; the final public/caller build passed 110 jobs; `make lint-model` passed all 260 targets. `make lint-code GOLANGCI_LINT_FIX=false` retained the inherited rc2 with exactly 1,316 sorted diagnostic headers and SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`, byte-identical to task6. Gate classification was full; the green-receipt write was nonblocking/unwarrantable because cumulative source work is dirty.

Structural cost: `composePlanningKnownGaps` contains the only new production `KnownGapSet.union`, called once per direct artifact or planning request before traversal; no row/candidate loop contains a union. `KnownGapSet.union` retains its existing merge/sort/deduplicate complexity. Ten independent equal-sized requests therefore perform exactly ten independent unions and at most 10x the new orchestration work. Case publication contains one `KnownGapSet.toList.map KnownGap.toCaseKnownGap` pass and no secondary conversion.

Trust: no task8 production addition contains `native_decide`, `axiom`, `implemented_by`, `sorry`, or `admit`. Switch retains its pre-existing `artifact_isSome` native evidence and derives the adjacent `exactActionRunResult_isSome` by cases without new native evidence; established Nexus operations remain typed `Except` and introduce no total fallback. The operation trust printout retains the task-start transitive baseline: `IncrementalPlannerKernel.ofCheckedQuery_isSome` uses only `propext`, `Classical.choice`, and `Quot.sound`; checked operation declarations/runs retain only their historical target/step-result and property/behavior/query native witnesses. Test-only fixture extractions remain confined to tests.

Review: official codex:gpt-5.6-sol:medium staged-overlay review returned SHIP with zero findings. Reviewed tree `62249a5c21b61277d8712786bd50c5ef04773d1d` equals the final pre-receipt staged tree, all 33 owned paths have no unstaged changes, and the reviewed task diff SHA-256 is `bec65df4e0b5e859360a98ea7227d716e8947a08b2e3dcd8f2939a0bc354ea55`. The reviewer independently reran the already-green 203-job Quick build despite the no-build instruction; it made no tracked or staged change.

stage: impl-review - ran [2026-09-06T05:29:00Z..2026-09-06T05:31:50Z] codex:gpt-5.6-sol:medium SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: tracker-sync - skipped(config: tracker.enabled != true)
## Evidence
- Commits:
- Tests: baseline: cd model && mise exec -- lake build Umpire.Planning.Tests UmpireTests Temporal.Feature.Nexus.OperationsTests (green; 203/203; /tmp/fn62-task8-baseline.log), TDD red: cd model && mise exec -- lake build Umpire.Planning.Tests (expected missing task8 API failures; /tmp/fn62-task8-red.log), cd model && mise exec -- lake build Umpire.Case.CompilerTests (green; exact all-value Case conversion), cd model && mise exec -- lake build Umpire.Space.Tests.Compilation (green; full typed gap payload; /tmp/fn62-task8-space-payload-2.log), cd model && mise exec -- lake build Umpire.Planning.Tests.Artifacts (green; exact composition/conflict/stability; /tmp/fn62-task8-authored-stability.log), cd model && mise exec -- lake build Umpire.ImportTests Umpire.CoreImportTests Temporal.Tool.NexusDiscoveryTests Temporal.Tool.InspectTests Temporal.Feature.Nexus2.Tests Temporal.Feature.Nexus2.AuthoringTests (green; 110 jobs; /tmp/fn62-task8-public-callers.log), cd model && mise exec -- lake build Umpire.Planning.Tests UmpireTests Temporal.Feature.Nexus.OperationsTests (green; 203/203; /tmp/fn62-task8-quick-final.log), make lint-model (green; 260/260; /tmp/fn62-task8-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited rc2; exactly 1316 sorted diagnostic headers; SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; byte-identical task6 baseline; /tmp/fn62-task8-lint-code.log), flowctl gate classify --base 8f40772ebb8e56fac4d068c9990b3c1827f8a05f (FULL); gate receipt nonblocking unavailable because cumulative source work makes receipt unwarrantable, structural/trust audit: one production union, one Case conversion map, zero new production native/compiler/assumption patterns; exact transitive trust baseline retained (/tmp/fn62-task8-structural-trust-audit.txt), impl-review codex:gpt-5.6-sol:medium SHIP; zero findings; receipt /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.8.json, reviewed staged tree 62249a5c21b61277d8712786bd50c5ef04773d1d equals final pre-receipt staged tree; owned unstaged clean; task diff SHA-256 bec65df4e0b5e859360a98ea7227d716e8947a08b2e3dcd8f2939a0bc354ea55
- PRs: