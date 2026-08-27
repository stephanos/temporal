---
satisfies: [R4, R7]
---
# fn-5-umpire-discovery-promotion-and-artifact.4 Compile exact in-memory promotion proposals

## Description
Implement the pure reusable `Umpire.Promotion.compileExactProposal` boundary for R4/R7.

**Size:** M
**Files:** `model/Umpire/Promotion.lean`, `model/Umpire/Promotion/Language.lean`, `model/Umpire/Promotion/Compiler.lean`, `model/Umpire/Promotion/Tests/Fixtures.lean`, `model/Umpire/Promotion/Tests/Compilation.lean`, `model/Umpire.lean`
**Touches:** [model/Umpire/Promotion.lean, model/Umpire/Promotion/Language.lean, model/Umpire/Promotion/Compiler.lean, model/Umpire/Promotion/Tests/**, model/Umpire.lean]

### Approach

- Define an inert request/proposal carrying the canonical `CheckedCatalog`, `originalQuery : CheckedQuery LawStatement`, `originalRun : PlannerRun` with `.found trace reason` and an artifact, explicit fresh promoted Behavior and Query declaration IDs, and `kernel : IncrementalPlannerKernel originalQuery.target`. Never accept a loose caller-supplied trace or reason.
- Recompute `artifactOfSelection originalQuery trace reason originalRun.result.metadata.explored` from the found result and require whole-value equality with the sole `originalRun.artifact` before compiling anything; this binds Query, witness, reason, explored counts, plan, Behavior Fingerprints, properties, requirements, and provenance without treating `PlannerRun` as carrying checked Query semantics.
- Convert the witness into `traceExactly`, then reuse `checkBehavior` and `checkQuery`; prove `promotedQuery.target = originalQuery.target`, transport the kernel, and reuse the current planner and artifact construction.
- Validate the promoted declaration IDs differ from their source declarations and collide with neither canonical catalog identities nor aliases. Validate original declarations, Behavior Fingerprints, and references against the catalog. The typed reusable compiler has no missing/target-mismatched kernel state; outer Temporal candidate validation owns missing candidate data.
- Support all current QueryForms only for exact found semantics: exhaustive `verify`/`violatingCounterexample`, `witness`/`satisfyingWitness`, `counterexample`/`violatingCounterexample`, and `select`/`behaviorSelection`. Re-evaluate property truth, then require the promoted planner to return the same exact trace and reason; non-`.found` outcomes are not promotable.
- Compare the replanned witness/reason, properties, target, query form, policy, Limits, and Behavior Fingerprints against independent expected inputs. Return newly derived promoted declaration IDs/digests and plan/Artifact Checksums with ordinary recomputed promoted provenance plus original Query/Artifact Checksum and provenance lineage; never require promoted identity or provenance equality with the original artifact.
- Keep deterministic Lean source and accepted Temporal constants outside this reusable module.
- Keep filesystem writes, raw JSON, evidence, live replay, and target outcome invention outside the module.

### Investigation targets

**Required:**
- `model/Umpire/Behavior/Language.lean:112-179` — authored and checked exact-trace structures.
- `model/Umpire/Behavior/Language.lean:815-951` — checking and admission.
- `model/Umpire/Query/Language.lean:214-285` — query validation and replay constraints.
- `model/Umpire/Planning/Engine.lean:359-470` — pure planner/artifact seam.
- `model/Umpire/Artifact.lean:193-241` — canonical Behavior Fingerprint.
- `model/Umpire/Examples/Switch.lean:307-611` — independent complete proof fixture.

### Quick command

`cd model && mise exec -- lake build Umpire.Promotion.Tests.Compilation`

## Acceptance
- [ ] The canonical checked catalog, explicit fresh promoted IDs, an independently produced Switch checked Query plus exactly bound `PlannerRun.found`, and matching target-indexed kernel compile into one checked exact proposal with the same trace/reason and new checked Behavior/Query/plan/Artifact Checksums.
- [ ] Positive and negative fixtures cover every current QueryForm and its exact SelectionReason/property-truth mapping; every non-`.found` planner outcome is rejected.
- [ ] Reordered equivalent source collections produce identical promoted checked values and proposal identity.
- [ ] Original Query/artifact lineage identities and provenance are retained exactly and are provably distinct from the promoted Query/Artifact Checksums and ordinarily recomputed promoted provenance.
- [ ] Any whole-value mismatch between the stored original artifact and canonical recomputation from Query, found trace/reason, and explored counts returns no partial proposal, as do missing IDs, collisions with canonical identities or aliases, catalog/reference/digest mismatch, loose or non-`.found` selections, form/reason/property mismatch, unresolved roles, failed target equality, incompatible Limits, and trace/reason drift.
- [ ] Properties, target, query form, policy, Limits, witness, selection reason, and source provenance are retained exactly.
- [ ] Model outcomes come only from the selected target trace and the ordinary planner.
- [ ] The reusable module imports no Temporal or runtime code.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
