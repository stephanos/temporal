---
satisfies: [R5]
---
# fn-45-index-and-reconcile-umpire-plan.5 Reduce fn-17 and fn-33 to bounded serial scope

## Description
Rewrite fn-17 and fn-33 delegation payloads to the retained R5 prototype scope while preserving stable numeric IDs.

**Size:** M
**Files:** `.flow/specs/fn-17-bounded-semantic-exploration-and.{md,json}`, `.flow/tasks/fn-17-bounded-semantic-exploration-and.*.{md,json}`, `.flow/specs/fn-33-run-resumable-semantic-exploration.{md,json}`, `.flow/tasks/fn-33-run-resumable-semantic-exploration.*.{md,json}`
**Touches:** [.flow/specs/fn-17-bounded-semantic-exploration-and.*, .flow/tasks/fn-17-bounded-semantic-exploration-and.*, .flow/specs/fn-33-run-resumable-semantic-exploration.*, .flow/tasks/fn-33-run-resumable-semantic-exploration.*]

### Approach
- Resolve `$FLOWCTL` through the Flow-Next preamble and preserve numeric task identity/history.
- Require a quiescent checkout, run supported Flow setters serially under the conductor's one-writer invariant, and verify paired Markdown/JSON immediately afterward; concurrent external mutation is unsupported.
- Retain bounded exhaustive enumeration, one uncovered-coordinate-guided policy, and pinned regressions outside the budget.
- Repurpose combinatorial/symmetry/resume/multiple-source tasks around retained universe, guidance, Nexus proof, integration, and documentation work.
- Reframe fn-33 as a serial bounded campaign and resume setters idempotently; no cross-file transaction is claimed.
- Remove fn-17 and fn-33's obsolete fn-5 dependency after their retained contracts no longer require it, and add fn-33's real prerequisite on fn-40's canonical PlannerPolicy surface.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_ORDER.md:74-102` — exact fn-17/fn-33 retained/deferred boundary.
- `.flow/tasks/fn-17-bounded-semantic-exploration-and.3.md` — pairwise/t-wise/seeded scope to reduce.
- `.flow/tasks/fn-17-bounded-semantic-exploration-and.4.md` — symmetry/report/resume scope to reduce.
- `.flow/tasks/fn-17-bounded-semantic-exploration-and.8.md` — multiple-source/protocol scope to repurpose.
- `.flow/specs/fn-33-run-resumable-semantic-exploration.md` — title/contract drift.

### Quick commands
`$FLOWCTL validate --spec fn-17 --json && $FLOWCTL validate --spec fn-33 --json`
## Acceptance
- [ ] fn-17 contains only exhaustive and uncovered-coordinate-guided selection with pinned-regression precedence.
- [ ] Pairwise/t-wise, symmetry, multiple sources, generalized resume/reporting, and adaptive corpus work are explicit non-goals.
- [ ] fn-33 title and tasks consistently specify one serial bounded campaign with no concurrency/resume machinery.
- [ ] Setters run serially in a quiescent checkout and verify Markdown/JSON afterward; interruption is fail-stop and idempotently resumable without claiming protection from unsupported concurrent writers.
- [ ] Existing task IDs remain usable; fn-17/fn-33 no longer depend on fn-5, fn-33 depends on fn-40, no dependency is deferred-only, and Flow validation passes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
