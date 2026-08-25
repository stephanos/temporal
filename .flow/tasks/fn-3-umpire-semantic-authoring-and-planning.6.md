---
satisfies: [R1, R4, R5, R6, R7, R8]
---
# fn-3-umpire-semantic-authoring-and-planning.6 Cut over the caller-closure scenario and public Experiment API

## Description
Replace the fn-1 combined authoring/compiler seam with the new public modules and port the bounded Workflow–Nexus caller-closure scenario (R1/R6/R7). This is one coordinated cutover so no callback-bearing or compatibility authoring path remains reachable.

**Size:** M
**Files:** `model/Temporal/Experiment/DSL.lean`, `model/Temporal/Experiment/Compiler.lean`, `model/Temporal/Experiment/Json.lean`, `model/Temporal/Experiment/NexusCallerClosure.lean`, `model/Temporal/Experiment/Inspect.lean`, `model/Temporal/ExperimentTests.lean`, `model/Temporal.lean`
**Touches:** [model/Temporal/Experiment/DSL.lean, model/Temporal/Experiment/Compiler.lean, model/Temporal/Experiment/Json.lean, model/Temporal/Experiment/NexusCallerClosure.lean, model/Temporal/Experiment/Inspect.lean, model/Temporal/ExperimentTests.lean, model/Temporal.lean]

### Approach
- Turn the existing DSL/compiler/JSON files into the single new public facade/entry points or remove redundant modules; do not retain legacy structures behind aliases.
- Implement the Workflow/Nexus target transition kernel with finite initial/step enumerators and soundness/completeness proofs against the authoritative checked relation, then express cancellation uniqueness and caller-closure behavior through its declared lifecycle, cancellation, and ownership-connector capabilities.
- Author exploratory, exact-action, and model-only query fixtures through the same Property/Behavior/Query interface.
- Make missing ownership connector, missing law, and ambiguous provider failures occur during checked composition before planner enumeration.
- Update the pure inspector registry/output and compile-time fixtures to the replacement artifact contract while preserving deterministic success and structured negative diagnostics.
- Preserve all existing comments in modified files.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/NexusCallerClosure.lean:8-75` — current pilot and model bridge to replace
- `model/NexusAutoClose.lean:508-627` — authoritative configuration, transition, and reachability semantics
- `model/NexusAutoClose.lean:740-755` — cancellation-honoring and clash semantics
- `model/NexusAutoClose.lean:870-932` — cancellation uniqueness theorem/counterexample
- `model/Temporal/Experiment/Inspect.lean:6-78` — closed pilot registry and structured inspector boundary

**Optional** (reference as needed):
- `.plans/UMPIRE_DSL.md:833-861` — required clean-replacement relationship to fn-1

### Quick commands
```bash
cd model && mise exec -- lake build ExperimentTests temporal-experiment-inspect
```
## Acceptance
- [ ] The caller-closure property declares only its required capabilities and is evaluated over the authoritative Workflow/Nexus composition.
- [ ] Every caller-closure trace step comes from the proved Workflow/Nexus target kernel, and exhaustive completeness relies on that kernel's finite-enumeration proofs.
- [ ] Removing the ownership connector or a required law rejects the query before enumeration with the expected structured diagnostic.
- [ ] Exploratory and exact-action caller-closure declarations compile through the same Query/planner path and retain model-owned outcome variability where applicable.
- [ ] Legacy combined regression, expected-property bag, callback projection, and old compile signature are no longer publicly importable or used by fixtures.
- [ ] Inspector success is canonical and repeatable, while unknown scenario and invalid composition paths emit one stable diagnostic and no artifact JSON.
- [ ] The focused Lean build passes and the R8 dependency/import/reference audit is clean.
## Done summary
Replaced callback-based experiment authoring with proof-backed semantic declarations for targets, properties, behaviors, queries, planning, and canonical artifacts. The Nexus caller-closure integration now proves an explicit owner-to-operation relation, reconciles conflicting internal claims into one public meaning, and exposes deterministic inspector output through the canonical query identity.

baseline: green via receipts
GATE_SKIPPED:unittest:green-receipt 101d2400 - baseline reused from prior post-gate pass
GATE_SKIPPED:smoke:green-receipt 101d2400 - baseline reused from prior post-gate pass
stage: impl-review - ran [2026-08-25T04:46:15Z..2026-08-25T05:03:55Z]
stage: plan-sync - skipped(config: planSync.enabled != true)

Memory capture was attempted after the non-trivial review fix and failed non-blockingly because the memory store is uninitialized; it was not initialized.
## Evidence
- Commits: d3e4b8d7db8f6a6d2cae19416c1bb959df776391, 62e8b514208c4a0a61a52628f0b07a5ebd14d6ba, cdc37b93cd75a922698f7019bab3f34d0ed203f5, 782e861820f1a44474cd4055001239b5b4c267f7
- Tests: GATE_SKIPPED:unittest:green-receipt 101d2400 - baseline reused from prior post-gate pass, GATE_SKIPPED:smoke:green-receipt 101d2400 - baseline reused from prior post-gate pass, cd model && mise exec -- lake build Temporal.Experiment.NexusCallerClosure, cd model && mise exec -- lake build ExperimentTests temporal-experiment-inspect, make umpire-check-regression
- PRs:
