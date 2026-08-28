---
satisfies: [R2, R5, R7]
---
# fn-39-make-the-temporal-nexus-feature-model.1 Split Lifecycle behind its stable facade

## Description
Separate the Lifecycle semantic and target implementations while preserving the facade contract (R2, R5, R7). Complete the public authoritative-initial seam so downstream correspondence proofs no longer unfold the Feature representation.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Lifecycle.lean`, `model/Temporal/Feature/Nexus/Lifecycle/Semantics.lean`, `model/Temporal/Feature/Nexus/Lifecycle/Target.lean`, `model/Temporal/System/Nexus/ImplementationLink.lean`, `model/Temporal/ImplementationLinkTests/Nexus.lean`
**Touches:** [model/Temporal/Feature/Nexus/Lifecycle.lean, model/Temporal/Feature/Nexus/Lifecycle/**, model/Temporal/System/Nexus/ImplementationLink.lean, model/Temporal/ImplementationLinkTests/Nexus.lean]

### Approach
- Move the focused state/event vocabulary, transition relation, and semantic law into `Lifecycle/Semantics.lean`; keep `Temporal.Feature.Nexus.Lifecycle` as the declaration namespace.
- Move ModelValue encodings, target relation/proofs, kernel, provider, composition, and planning defaults into `Lifecycle/Target.lean`, imported by the existing facade.
- Retain the existing `Lifecycle.source` value and every public declaration, Definition ID, canonical behavior string, proof meaning, and comment.
- Add facade-level authoritative initial-state theorems parallel to the existing authoritative transition theorems and make the System witness consume them.
- Keep `Semantics` independent of `Target`; do not introduce imports from Feature to System.
- Add concise module and read-next documentation to the Lifecycle facade and both children while preserving every existing declaration comment.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Lifecycle.lean:31-68` — semantic vocabulary, transition, and law.
- `model/Temporal/Feature/Nexus/Lifecycle.lean:69-298` — value encoding and authoritative relation/proofs.
- `model/Temporal/Feature/Nexus/Lifecycle.lean:300-532` — target, provider, finite planning, and query support.
- `model/Temporal/System/Nexus/ImplementationLink.lean:312-365` — downstream witness and direct initial-state unfolding to replace.
- `.flow/tasks/fn-38-consolidate-layered-model-helpers.4.md:13-32` — predecessor source/provenance and facade constraints.

**Optional** (reference as needed):
- `model/Temporal/ImplementationLinkTests/Nexus.lean:18-40` — focused forward-correspondence assertions.

### Acceptance
- [ ] Lifecycle facade imports the focused children without an import cycle and every existing qualified declaration still elaborates.
- [ ] Lifecycle semantics, target values, IDs, source, fingerprints, comments, and canonical outputs are unchanged.
- [ ] Implementation Link initial-forward proof uses public authoritative-initial lemmas rather than unfolding Lifecycle initial-state internals.
- [ ] Facade and child module docs explain the semantic-to-target reading order without removing or rewriting existing declaration comments.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus` passes.

## Acceptance
- [ ] R2, R5, and R7 task-scoped checks pass.
- [ ] No unrelated worktree file is modified.

## Done summary
Split the ordinary Nexus Lifecycle behind stable Semantics and Target children while preserving its facade, public identities, provenance, behavior, canonical outputs, and existing comments. Added public scheduled/started initial-authority theorems so the System correspondence witness no longer unfolds Feature initial-state representation.

stage: impl-review - ran (SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7160d89ad33eb479809378742a74f9b98180c585
- Tests: baseline qualification: parent aggregate target Temporal.Feature.NexusTests is expected fn39.5 work; conductor authorized fn39.1 task-focused baseline, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus, make umpire-check-regression, make lint-model, git diff --check bee5442ff6c67c3980e8f807d33330dc458180fa..HEAD, Lifecycle import-direction, protected-scope, comment-preservation, and downstream representation-unfolding audits
- PRs:
