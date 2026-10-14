---
satisfies: [R3, R4, R5]
---
# fn-36-nexus-lifecycle-cleanup.2 Relocate AutoClose and caller closure behind Experimental opt-in

## Description
Move the detailed AutoClose/caller-closure Lean family under Experimental and establish explicit opt-in build/test consumers (R3, R4, R5). Artifact fixture/projection rebasing is left to task .3.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/AutoClose.lean`, `model/Temporal/Feature/Nexus/CallerClosure.lean`, `model/Temporal/Feature/Nexus/CallerClosureTests.lean`, `model/Temporal/Feature/Nexus/Experimental/**`, `model/Temporal/Tool/Inspect.lean`, `model/Temporal/Tool/InspectTests.lean`, `model/TemporalExperimentalTests.lean`, `model/lakefile.toml`, `Makefile`
**Touches:** [model/Temporal/Feature/Nexus/AutoClose.lean, model/Temporal/Feature/Nexus/CallerClosure.lean, model/Temporal/Feature/Nexus/CallerClosureTests.lean, model/Temporal/Feature/Nexus/Experimental/**, model/Temporal/Tool/Inspect.lean, model/Temporal/Tool/InspectTests.lean, model/TemporalExperimentalTests.lean, model/lakefile.toml, Makefile]

### Approach
- Move AutoClose, CallerClosure, and CallerClosureTests without rewriting their proofs or tutorial comments; update namespaces, imports, opens, and truthful SemanticSource paths.
- Preserve experimental semantic declaration IDs, semantic digest strings, and scenario behavior.
- Update inspector production/tests to opt into Experimental.CallerClosure explicitly.
- Add TemporalExperimentalTests as a separate Lake/default/full-regression target while keeping TemporalModelTests core-only.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/AutoClose.lean:1-1101` — detailed tutorial/proof to preserve.
- `model/Temporal/Feature/Nexus/CallerClosure.lean:1-712` — direct AutoClose consumer and source provenance.
- `model/Temporal/Feature/Nexus/CallerClosureTests.lean:1-256` — experimental tests and diagnostic provenance.
- `model/Temporal/Tool/Inspect.lean:1-75` — registered caller-closure opt-in.
- `model/lakefile.toml:1-33` — aggregate/default target declarations.

### Key context
- Do not move or regenerate the JSON fixture in this task; task .3 owns that atomic generated boundary.
- The ordinary Temporal.Feature and TemporalModelTests roots must remain free of Experimental imports after task .1.

### Acceptance
- [ ] Experimental namespaces compile and old root AutoClose/CallerClosure Lean modules no longer exist.
- [ ] Existing comments, proofs, semantic IDs, and digests remain intact.
- [ ] Inspector explicitly imports the experimental scenario.
- [ ] Separate experimental aggregate is built by default/full regression without leaking into ordinary facades.

## Acceptance
- [ ] R3 module relocation preserves model semantics and comments.
- [ ] R4 separate aggregate/import direction is established.
- [ ] R5 source provenance and build references use the new path.
- [ ] Focused moved-module and inspector targets compile before fixture regeneration.

## Done summary
Moved AutoClose, CallerClosure, and CallerClosureTests intact under Nexus/Experimental; updated namespaces and source provenance; made inspector opt in explicitly; added a separate TemporalExperimentalTests aggregate and default/full-regression build coverage. Mechanical preservation and core-import boundary checks passed. stage: plan-sync - skipped(config: planSync.enabled != true). No commit was created per repository instructions.
## Evidence
- Commits:
- Tests: cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.CallerClosureTests Temporal.Tool.InspectTests TemporalExperimentalTests temporal-model-inspect TemporalModelTests, cd model && mise exec -- lake build
- PRs: