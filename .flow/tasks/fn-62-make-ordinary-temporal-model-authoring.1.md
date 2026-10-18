---
satisfies: [R2]
---
# fn-62-make-ordinary-temporal-model-authoring.1 Deepen explicit finite author evidence assembly

## Description
Group the proof-carrying finite inputs behind a smaller constructor seam (R2). This task establishes the reusable interface; `.2` proves the established consumer migration.

**Size:** M
**Files:** `model/Umpire/Target/FiniteMachine.lean`, `model/Umpire/Target/Tests/FiniteMachine.lean`, `model/Umpire/Target/ImportTests.lean`
**Touches:** [model/Umpire/Target/FiniteMachine.lean, model/Umpire/Target/Tests/FiniteMachine.lean, model/Umpire/Target/ImportTests.lean]

## Approach
- Keep the existing `FiniteMachine` record as the complete author-evidence input; do not add another wrapper around its fields. Add `FiniteMachine.targetDefinition(machine, id, source, definitions, requiredCapabilities)` returning the existing `TargetDefinition`, and `FiniteMachine.authoredTarget(machine, id, source, definitions, requiredCapabilities, composition, occurrences)` returning the existing `AuthoredTarget`. All metadata, composition and occurrences are explicit inputs; use the existing types and parameter order conventions.
- The first constructor derives only `resolvedSetups := machine.setups` and `kernel := machine.kernelAvailability`; the second packages that definition, explicit composition/occurrences, and `machine.authoredPlanning` through `AuthoredTarget.make`. Authors still supply all five domains/encoders, both enumerators, and all eight existing proof fields: setupCoverage, initialStateCoverage, transitionSourceCoverage, actionCoverage, resultingStateCoverage, outcomeCoverage, observationCoverage, and actionExecutable. Final `checkTarget` remains separate.
- Keep old public Nexus aliases supported, but ordinary authoring must no longer assemble `TargetDefinition`, dependent `.available kernel rfl planning`, or duplicate machine setup/kernel selections. Test exact equivalence of old assembly and these constructors on the same finite fixture; this removes a complete representation-dependent assembly layer rather than repackaging the FiniteMachine record.
- Preserve `ValidatedFiniteTable.machine` as the separate prototype route. Do not derive missing author proofs or add a second finite semantic representation.
- Compile positive assembly and actual missing-proof constructions using `#guard_msgs`; negated propositions about a relation do not prove missing arguments fail elaboration.
- Audit transitive trust of new public constructors and witnesses. Keep existing comments and add public API documentation describing the evidence boundary.

## Investigation targets
**Required:**
- `model/Umpire/Target/FiniteMachine.lean:13` — explicit fields and existing adapters.
- `model/Umpire/Target/Language.lean:202` — `AuthoredTarget.make`.
- `model/Umpire/Target/Tests/FiniteMachine.lean:113` — proof and collision fixtures.
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean` — concrete repeated assembly to eliminate.

## Acceptance
- [ ] Positive old/new assembly fixture has identical complete authored/checkTarget results. The new author-facing path has zero hand-built TargetDefinition records, zero `.available ... rfl ...` transport expressions, and zero repeated setup/kernel selection fields after the one FiniteMachine input; all eight proof arguments remain explicit.
- [ ] Missing domain closure and Action-executability arguments fail elaboration; collision, invalid raw incomplete kernel, and provider failures retain exact typed diagnostics.
- [ ] Public import checks expose the intended seam without implementation imports or additional trust.
- [ ] Record a structural complexity audit of the two constructors: only record projection/assembly and existing AuthoredTarget.make work; zero added list traversal, normalization, validation or nested scan. The audit identifies every called function and shows 1×/10× declaration sets add only one assembly per declaration, excluding existing checker work from claims about added cost.
- [ ] `cd model && mise exec -- lake build Umpire.TargetTests` passes; run model lint and required Go lint with inherited output compared to baseline.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: