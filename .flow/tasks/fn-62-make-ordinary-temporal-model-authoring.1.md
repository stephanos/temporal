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
Added `FiniteMachine.targetDefinition` and `FiniteMachine.authoredTarget` as the ordinary proof-carrying finite assembly seam. The existing `FiniteMachine` record remains unchanged: authors still supply all five domains, five encoders, both enumerators, seven closure proofs, and `actionExecutable`; the new constructors derive only the repeated setup/kernel/planning assembly and keep `checkTarget` separate.

Tests establish definitional equality between the old and new complete authored values and checker results with nonempty capabilities, provider/connector composition, and occurrences. Actual record constructions missing `initialStateCoverage` or `actionExecutable` fail elaboration under `#guard_msgs`; the aggregate Target suite retains collision, raw incomplete-kernel, missing-provider, and conflicting-provider diagnostics. Public import checks expose both constructors, and both new declarations report no axioms.

Structural complexity audit: `targetDefinition` calls `kernelAvailability`, transitively `kernel`, and projects `setups`; `authoredTarget` calls `targetDefinition`, `AuthoredTarget.make`, and `authoredPlanning`, which transitively calls `kernelAvailability`, `kernel`, and `planning`. These calls only project or assemble records. They add zero list traversals, normalization, validation, nested scans, or checker calls. The 1×/10× fixture produces exactly one authored assembly per independent declaration; unchanged `checkTarget` work is excluded from that claim.

Verification: focused baseline and final `Umpire.TargetTests` builds passed (26 targets); `make lint-model` passed (257 targets). `make lint-code GOLANGCI_LINT_FIX=false` retained the inherited exit 2 with exactly 1,316 diagnostic headers, byte-identical to the established baseline (SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`). The official staged-overlay review returned SHIP with no findings, and reviewed source blobs equal the final staged source blobs.

No commit was created under the user's standing commit policy; HEAD remains `7774fdc7ac751ac959816c9829516ce54af57194`.

stage: impl-review - ran (codex:gpt-5.6-sol:medium; SHIP; receipt timestamp 2026-09-06T02:44:37.419547Z)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: cd model && mise exec -- lake build Umpire.TargetTests (pass; 26 targets; /tmp/fn62-task1-baseline-targettests.log), TDD red: cd model && mise exec -- lake build Umpire.TargetTests (expected failure: FiniteMachine.targetDefinition and FiniteMachine.authoredTarget absent; /tmp/fn62-task1-red-targettests.log), cd model && mise exec -- lake build Umpire.TargetTests (pass; 26 targets; /tmp/fn62-task1-final-targettests.log), make lint-model (pass; 257 targets; /tmp/fn62-task1-final-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2; exactly 1316 diagnostic headers; normalized SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; byte-identical baseline; /tmp/fn62-task1-final-lint-code.log), #guard_msgs actual FiniteMachine constructions missing initialStateCoverage and actionExecutable (pass in Umpire.TargetTests), #print axioms Umpire.FiniteMachine.targetDefinition and Umpire.FiniteMachine.authoredTarget (no axioms), structural complexity: 1x/10x independent authoredTarget declarations produce lengths 1/10 without checker work, impl-review codex:gpt-5.6-sol:medium (SHIP; /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.1.json), reviewed source equality (review tree 0aef85714909708deb0095251365e78d3872463e equals final staged source blobs)
- PRs: