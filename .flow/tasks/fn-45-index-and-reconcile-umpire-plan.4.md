---
satisfies: [R4]
---
# fn-45-index-and-reconcile-umpire-plan.4 Reduce fn-5 to retained discovery and promotion scope

## Description
Rewrite fn-5's spec/task delegation payloads to the retained R4 scope while preserving its history and IDs.

**Size:** M
**Files:** `.flow/specs/fn-5-umpire-discovery-promotion-and-artifact.{md,json}`, `.flow/tasks/fn-5-umpire-discovery-promotion-and-artifact.*.{md,json}`
**Touches:** [.flow/specs/fn-5-umpire-discovery-promotion-and-artifact.*, .flow/tasks/fn-5-umpire-discovery-promotion-and-artifact.*]

### Approach
- Resolve `$FLOWCTL` through the Flow-Next preamble; use its spec/task setters and never hand-edit Flow JSON.
- Require a quiescent checkout, run supported Flow setters serially under the conductor's one-writer invariant, and verify paired Markdown/JSON immediately afterward; concurrent external mutation is unsupported.
- Retain only coherent list/explain for current Nexus declarations and one checked review-only promotion of the minimized duplicate-delivery failure.
- Repurpose existing task IDs so no task remains stranded solely for generic graph/glossary/machine-index/broad-regression/artifact-evolution work.
- Resume granular setters idempotently after interruption; do not claim cross-file transactionality.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_ORDER.md:64-72` — exact retained/deferred fn-5 boundary.
- `.flow/specs/fn-5-umpire-discovery-promotion-and-artifact.md` — current broad contract.
- `.flow/tasks/fn-5-umpire-discovery-promotion-and-artifact.1.md` — generic catalog scope to reduce.
- `.flow/tasks/fn-5-umpire-discovery-promotion-and-artifact.3.md` — explicitly deferred glossary/index scope.
- `.flow/tasks/fn-5-umpire-discovery-promotion-and-artifact.6.md` — explicitly deferred broad regression scope.

### Quick commands
`$FLOWCTL validate --spec fn-5 --json`
## Acceptance
- [ ] Every fn-5 R-ID and task traces to retained list/explain or checked review-only promotion.
- [ ] Existing task IDs are coherently repurposed or narrowed; none depends on deferred machinery.
- [ ] Generic semantic graph, generated glossary, machine index, broad stable set, and general artifact evolution are explicit non-goals.
- [ ] Setters run serially in a quiescent checkout and verify Markdown/JSON afterward; interruption is fail-stop and idempotently resumable without claiming protection from unsupported concurrent writers.
- [ ] Flow validation passes and existing history/comments are preserved.
## Done summary
Reduced fn-5 to two retained requirements: deterministic list/explain for four current Nexus examples and one inert, checked review-only duplicate-delivery proposal whose runtime eligibility remains exclusively owned by fn-22. All seven stable task IDs now trace to those capabilities, deferred breadth is explicit under Non-goals, valid inventory permutations canonicalize, and the base expected-trace lineage is distinct from the fault-bearing ExperimentSpec lineage.

The serial Flow setters were semantically idempotent and preserved paired Markdown/JSON, history, the artifact-link comment, and scope comments. Codex review found three contract issues, all fixed in the same session before SHIP; memory capture was skipped because Flow memory is not initialized.

GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)

GATE_SKIPPED:smoke:docs-only - cumulative diff classified tier-B (no executable paths touched)

stage: impl-review - ran [NEEDS_WORK to SHIP at 2026-09-01T09:50:05Z; session 01a05c55-c001-7d71-ab2d-e260c61a1c1f]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 56fae915d5f0039e04f707060226a6bd646d2760, ab5a2bc6aef181c69e6ad46d817b01d9129deeea, 1f51dc319141a30c112c3ac0214fdd7e7648f9b6
- Tests: baseline: green (/Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/scripts/flowctl validate --spec fn-5 --json), TDD RED: deterministic fn5 contract-language audit (35 forbidden positive-scope violations before setters), /Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/scripts/flowctl validate --spec fn-5 --json (valid; 0 errors; 0 warnings), /Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/scripts/flowctl validate --all --json (valid; 0 errors; 203 inherited warnings), deterministic fn5 retained-contract audit (7 tasks; 2 R-IDs; fixed fn22 argv; canonical permutations; separate base/fault lineages; 0 forbidden positive-scope violations), git diff --check d62ae892478d856a0e3561c61779ef4292688d27..HEAD, GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:smoke:docs-only - cumulative diff classified tier-B (no executable paths touched)
- PRs:
