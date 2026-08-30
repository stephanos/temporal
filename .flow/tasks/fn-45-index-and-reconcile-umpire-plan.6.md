---
satisfies: [R3, R4, R5, R6]
---
# fn-45-index-and-reconcile-umpire-plan.6 Reconcile Flow dispositions and prototype gates

## Description
Apply and verify the explicit one-time Flow reconciliation for R3-R6 after all plan content is current.

**Size:** S
**Files:** `.flow/specs/fn-{5,14,15,17,23,24,25,26,27,28,29,30,33,43,48,49,51}-*.json` and matching Markdown where flowctl records state
**Touches:** [.flow/specs/fn-5-*, .flow/specs/fn-14-*, .flow/specs/fn-15-*, .flow/specs/fn-17-*, .flow/specs/fn-23-*, .flow/specs/fn-24-*, .flow/specs/fn-25-*, .flow/specs/fn-26-*, .flow/specs/fn-27-*, .flow/specs/fn-28-*, .flow/specs/fn-29-*, .flow/specs/fn-30-*, .flow/specs/fn-33-*, .flow/specs/fn-43-*, .flow/specs/fn-48-*, .flow/specs/fn-49-*, .flow/specs/fn-51-*]

### Approach
- Resolve `$FLOWCTL` through the Flow-Next preamble; preflight accepted hashes, exact IDs, statuses, readiness, and dependencies before each setter.
- Keep fn-14 open and unready as the supported superseded tombstone; never mark its unfinished tasks done.
- Through flowctl only: mark fn-15, fn-23..26, fn-29..30, fn-43, fn-48, fn-49, and fn-51 unready; add fn-27 -> fn-21 and retained P3 roots fn-5/fn-17 -> fn-28.
- Verify Markdown/JSON after each operation; use the checker to report and idempotently resume only remaining drift after interruption.
- Run full Flow validation and the plan-index check; do not suppress warnings.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_ORDER.md:223-261` — verification gate and dispositions.
- `.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md:5-10` — supported superseded tombstone representation.
- `.flow/specs/fn-27-hermetic-ci-execution-and-qualification.json` — missing fn-21 dependency.
- `.flow/specs/fn-28-authorized-remote-staging-black-box.json` — existing fn-27 sequencing.
- `.flow/specs/fn-48-canonicalize-known-gaps-as-a-checked-set.json` — support dependency on deferred fn-43.

### Quick commands
`$FLOWCTL validate --all --json && make umpire-check-plan-index`
## Acceptance
- [ ] fn-14 remains open/unready and is classified superseded without falsifying task completion.
- [ ] Exact deferred execution states match R3, including support specs that depend on fn-43.
- [ ] fn-27 waits for fn-21, fn-28 waits for fn-27, and fn-5/fn-17 wait for fn-28 without a cycle.
- [ ] No retained task depends directly or transitively on deferred-only scope.
- [ ] Every setter enforces its accepted baseline; interrupted reconciliation is checker-visible/idempotently resumable, and full Flow/index validation passes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
