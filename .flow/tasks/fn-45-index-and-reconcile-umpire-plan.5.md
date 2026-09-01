---
satisfies: [R1, R3, R5]
---
# fn-45-index-and-reconcile-umpire-plan.5 Reduce fn-17 and fn-33 to bounded serial scope

## Description
Rewrite fn-17 and fn-33 delegation payloads to the retained R5 prototype scope while preserving stable numeric IDs.

**Size:** M
**Files:** `.flow/specs/fn-17-bounded-semantic-exploration-and.{md,json}`, `.flow/tasks/fn-17-bounded-semantic-exploration-and.*.{md,json}`, `.flow/specs/fn-33-run-resumable-semantic-exploration.{md,json}`, `.flow/tasks/fn-33-run-resumable-semantic-exploration.*.{md,json}`, `.flow/specs/fn-33-run-serial-bounded-semantic-exploration.{md,json}`, `.flow/tasks/fn-33-run-serial-bounded-semantic-exploration.*.{md,json}`, `.plans/index.json`
**Touches:** [.flow/specs/fn-17-bounded-semantic-exploration-and.*, .flow/tasks/fn-17-bounded-semantic-exploration-and.*, .flow/specs/fn-33-run-resumable-semantic-exploration.*, .flow/tasks/fn-33-run-resumable-semantic-exploration.*, .flow/specs/fn-33-run-serial-bounded-semantic-exploration.*, .flow/tasks/fn-33-run-serial-bounded-semantic-exploration.*, .plans/index.json]

### Approach
- Resolve `$FLOWCTL` through the Flow-Next preamble and preserve numeric task identity/history.
- Require a quiescent checkout, run supported Flow setters serially under the conductor's one-writer invariant, and verify paired Markdown/JSON immediately afterward; concurrent external mutation is unsupported.
- Run after task .3 because both migrations update the manually authored plan registry.
- Retain the `coverageGuided` to `seeded` Query-strategy rename required by fn-40, bounded exhaustive enumeration, one uncovered-coordinate-guided policy, and pinned regressions outside the budget.
- Repurpose combinatorial/symmetry/resume/multiple-source tasks around retained universe, guidance, Nexus proof, integration, and documentation work.
- Reframe fn-33 as a serial bounded campaign and resume setters idempotently; no cross-file transaction is claimed.
- Use the supported title setter to rename fn-33's canonical slug to `fn-33-run-serial-bounded-semantic-exploration`, preserve numeric task suffixes/history, and update the registry row to the new canonical ID without changing its classification.
- Remove fn-17 and fn-33's obsolete fn-5 dependency after their retained contracts no longer require it, and add fn-33's real prerequisite on fn-40's canonical PlannerPolicy surface.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_ORDER.md:74-102` — exact fn-17/fn-33 retained/deferred boundary.
- `model/Umpire/Query/Language.lean:14` and `.flow/tasks/fn-40-centralize-plannerpolicy-constructors.1.md` — required `coverageGuided` to `seeded` handoff.
- `.flow/tasks/fn-17-bounded-semantic-exploration-and.3.md` — pairwise/t-wise/seeded scope to reduce.
- `.flow/tasks/fn-17-bounded-semantic-exploration-and.4.md` — symmetry/report/resume scope to reduce.
- `.flow/tasks/fn-17-bounded-semantic-exploration-and.8.md` — multiple-source/protocol scope to repurpose.
- `.flow/specs/fn-33-run-resumable-semantic-exploration.md` — source title/contract drift before the supported rename.
- `.plans/index.json` — desired-state Flow registry row that must follow the supported fn-33 rename.

### Quick commands
`$FLOWCTL validate --spec fn-17 --json && $FLOWCTL validate --spec fn-33 --json`
## Acceptance
- [ ] fn-17 contains only exhaustive and uncovered-coordinate-guided selection with pinned-regression precedence.
- [ ] Fn-17.1 owns the hard-cut Query strategy rename from `coverageGuided` to `seeded`, with repository-wide callers and canonical identities updated for fn-40's retained constructor work.
- [ ] Pairwise/t-wise, symmetry, multiple sources, generalized resume/reporting, and adaptive corpus work are explicit non-goals.
- [ ] fn-33 title and tasks consistently specify one serial bounded campaign with no concurrency/resume machinery.
- [ ] Fn-33's canonical spec/task IDs use the new serial-bounded slug with stable numeric ID 33 and task suffixes, no old registered row remains, and `.plans/index.json` preserves the row's intended scope/disposition/phase/state while recording the new ID/dependencies.
- [ ] Setters run serially in a quiescent checkout and verify Markdown/JSON afterward; interruption is fail-stop and idempotently resumable without claiming protection from unsupported concurrent writers.
- [ ] Existing numeric task IDs remain usable; fn-17/fn-33 no longer depend on fn-5, fn-33 depends on fn-40, no dependency is deferred-only, and Flow validation passes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
