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
- Capture accepted hashes and Flow state for every target, compare immediately before each setter, abort that setter on change, and verify paired Markdown/JSON afterward.
- Retain only coherent list/explain for current Nexus declarations and one checked review-only promotion of the minimized duplicate-delivery failure.
- Repurpose existing task IDs so no task remains stranded solely for generic graph/glossary/machine-index/broad-regression/artifact-evolution work.
- Resume granular setters idempotently after interruption; do not claim cross-file transactionality.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_ORDER.md:170-178` — exact retained/deferred fn-5 boundary.
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
- [ ] Every setter enforces its accepted baseline and verifies Markdown/JSON; interruption is idempotently resumable.
- [ ] Flow validation passes and existing history/comments are preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
