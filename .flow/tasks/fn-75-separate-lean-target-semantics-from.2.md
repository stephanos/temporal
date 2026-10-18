---
satisfies: [R1, R2, R7]
---
# fn-75-separate-lean-target-semantics-from.2 Migrate semantic consumers and enforce complete import isolation

## Description
Move semantic consumers to the established narrow imports and enforce the resulting transitive boundary.

**Size:** M
**Files:** Property/Behavior/Query/Planning semantic modules, Artifact dependency modules, ModelLint graph policy/loader/tests and shared import graph machinery when necessary.
**Touches:** [model/Umpire/Property/**, model/Umpire/Behavior/**, model/Umpire/Query/**, model/Umpire/Planning/**, model/Umpire/Artifact/**, model/ModelLint.lean, model/ModelLint/**, model/Tools/LeanImportGraph.lean, model/Tools/LeanImportGraph/**, model/Temporal/Feature/Nexus/Lifecycle/Target.lean, model/Temporal/Feature/Nexus3/Authoring.lean, model/Temporal/Feature/Nexus/Operations/Internal.lean, model/Temporal/Feature/Nexus/Operations/AsyncStart.lean, model/Temporal/Feature/Nexus/Operations/Cancellation.lean, model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean]

### Approach
- Preserve the existing AsyncStart, Cancellation, and SuccessfulCompletion walkthroughs with import-only additions of their Behavior authoring owner, as required by whole-source compilation. This preserves existing models and does not implement deferred fn-79 cancellation.
- Whole-source compilation identified legitimate authoring consumers that depended on incidental exports through Planning: `model/Temporal/Feature/Nexus/Lifecycle/Target.lean` `model/Temporal/Feature/Nexus3/Authoring.lean`, and `model/Temporal/Feature/Nexus/Operations/Internal.lean` (the shared Property authoring constructor). Add only the existing Target/Property/Behavior/Query facade imports each consumer actually needs. Preserve their declarations, proofs, comments and expected outputs. Do not restore authoring dependencies to semantic Planning, weaken the import policy, or split additional construction types merely to retain incidental transitive imports.
- Enumerate the exact production semantic roots and close their full dependency paths, including Planning.Engine through Artifact.Planning/Codecs/Types and Planning.Types into Query. Import language/check/evaluation or semantic surfaces instead of broad authoring facades where sufficient. Preserve all public semantic types and computation.
- Extend ModelLint to reject Target frontend and Lean.Elab.Term in those roots' actual transitive closure. Current classification ignores external destinations and the graph loader omits external metadata; address both. Keep owned-source reconciliation and reachable external metadata inspection distinct, preserving missing-owned-metadata failures.
- Reuse deterministic qualified-path diagnostics and graph traversal. Add direct, first-party bridge, external-wrapper, cycle and allowed-import cases. Authoring facades/tests remain allowed; do not indiscriminately ban all authoring packages or traverse an unbounded filesystem inventory.
- Retain existing import policies and controlled-violation output; no fn76 outcome/Known Gap extraction or fn77 operation changes.

### Investigation targets
**Required:**
- model/Umpire/Property/Language.lean:1 — broad Target import
- model/Umpire/Behavior/Language.lean:2 — broad Target import
- model/Umpire/Query/Language.lean:1 — broad Property/Behavior closure
- model/Umpire/Planning/Types.lean:1 — Query facade dependency
- model/ModelLint/ImportGraph.lean:289 — external endpoint classification gap
- model/ModelLint.lean:30 — metadata loading
- model/Tools/LeanImportGraph.lean:36 — traversal of recorded modules

### Verification
Run serially: cd model && mise exec -- lake build Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Umpire.Planning.Tests modelLintTests; then cd model && mise exec -- lake exe modelLintTests. Run the real whole-source modelLint executable against compiled dependencies to prove the synthetic cases match actual closure enforcement. Use LEAN_NUM_THREADS=1.

## Acceptance
- [ ] Every selected semantic root satisfies R1/R2 through its actual external and first-party closure; authoring facades retain their intended imports.
- [ ] Direct, first-party bridge and external-wrapper violations report deterministic qualified paths; allowed imports, cycles and missing-owned-metadata cases are covered.
- [ ] Existing graph policies and controlled violation diagnostics remain valid; focused semantic consumer suites and actual modelLint check pass.
- [ ] No semantic behavior, public type, generated output or adjacent fn76/fn77 scope changes are introduced.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: