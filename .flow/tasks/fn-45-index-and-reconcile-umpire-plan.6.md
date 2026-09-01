---
satisfies: [R3, R4, R5, R6]
---
# fn-45-index-and-reconcile-umpire-plan.6 Reconcile Flow dispositions and prototype gates

## Description
Apply and verify the explicit one-time Flow reconciliation for R3-R6 after all plan content is current.

**Size:** S
**Files:** `.flow/specs/fn-{5,14,15,17,22,23,24,25,26,28,29,30,33,40,42,43,44,48,49,50,51}-*.json` and matching Markdown where flowctl records state
**Touches:** [.flow/specs/fn-5-*, .flow/specs/fn-14-*, .flow/specs/fn-15-*, .flow/specs/fn-17-*, .flow/specs/fn-22-*, .flow/specs/fn-23-*, .flow/specs/fn-24-*, .flow/specs/fn-25-*, .flow/specs/fn-26-*, .flow/specs/fn-28-*, .flow/specs/fn-29-*, .flow/specs/fn-30-*, .flow/specs/fn-33-*, .flow/specs/fn-40-*, .flow/specs/fn-42-*, .flow/specs/fn-43-*, .flow/specs/fn-44-*, .flow/specs/fn-48-*, .flow/specs/fn-49-*, .flow/specs/fn-50-*, .flow/specs/fn-51-*]

### Approach
- Resolve `$FLOWCTL` through the Flow-Next preamble; require a quiescent checkout, read exact IDs, statuses, readiness, and dependencies, run setters serially, and verify each result before continuing.
- Keep fn-14 open and unready as the supported superseded tombstone; never mark its unfinished tasks done.
- Through flowctl only: keep fn-14 and fn-15 unready; mark fn-23..26 and fn-29..30 unready; keep retained fn-43/fn-48/fn-49/fn-51 unready; and mark decision-gated fn-5/fn-17/fn-22/fn-33/fn-40 unready until fn-28 evidence is reviewed.
- Preserve completed-prerequisite fn-42/fn-44/fn-50 as open-SHIP at their actual unready state. Do not add retroactive fn-27 -> fn-21 or fn-5/fn-17 -> fn-28 dependency edges.
- Verify fn-17/fn-33 no longer depend on fn-5 and fn-33 has the retained fn-40 prerequisite established by task .5.
- Verify Markdown/JSON after each operation; use the checker to report and idempotently resume only remaining drift after interruption.
- Run full Flow validation and the plan-index check; do not suppress warnings.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_ORDER.md:150-177` — verification gate and dispositions.
- `.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md:5-10` — supported superseded tombstone representation.
- `.flow/specs/fn-21-nexus-duplicate-observation-control.json` — completed prerequisite that must not become a retroactive prerequisite of completed fn-27.
- `.flow/specs/fn-28-authorized-remote-staging-black-box.json` — existing fn-27 sequencing and current prototype decision point.
- `.flow/specs/fn-48-canonicalize-known-gaps-as-a-checked-set.json` — retained support dependency on retained fn-43/fn-47.

### Quick commands
`$FLOWCTL validate --all --json && make umpire-check-plan-index`
## Acceptance
- [ ] fn-14 remains open/unready and is classified superseded without falsifying task completion.
- [ ] Exact deferred and decision-gated execution states match R3; retained non-gating support remains distinct from deferred work.
- [ ] fn-28 keeps its existing fn-27 dependency, while the fn-28 evidence decision gate is represented without retroactive or priority-only hard edges.
- [ ] fn-42/fn-44/fn-50 remain open-SHIP completed prerequisites and fn-43/fn-48/fn-49/fn-51 remain retained unready support.
- [ ] Fn-17/fn-33 have no obsolete fn-5 edge, fn-33 depends on fn-40, and the full graph remains acyclic.
- [ ] No retained task depends directly or transitively on deferred-only scope.
- [ ] Setters run serially in a quiescent checkout and are post-verified; interrupted reconciliation is checker-visible/idempotently resumable, full Flow/index validation exits zero, and the inherited warning baseline is surfaced without increase or suppression.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
