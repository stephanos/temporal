---
satisfies: [R1, R4, R5]
---
# fn-9-umpire-reusable-dsl-package-split.2 Split Property Behavior and Search modules

## Description
Move the two independent authoring DSLs and extract the data-only Search contract (R1, R4, R5). Keep this separate from Query so the sibling import seam is visible and testable.

**Size:** M
**Files:** `model/Umpire.lean`, `model/UmpireTests.lean`, `model/Umpire/Property.lean`, `model/Umpire/PropertyTests.lean`, `model/Umpire/Behavior.lean`, `model/Umpire/BehaviorTests.lean`, `model/Umpire/Search.lean`, narrow-import test modules
**Touches:** [model/Umpire.lean, model/UmpireTests.lean, model/Umpire/Property.lean, model/Umpire/PropertyTests.lean, model/Umpire/Behavior.lean, model/Umpire/BehaviorTests.lean, model/Umpire/Search.lean, model/Umpire/*ImportTests.lean]

### Approach
- Move Property and Behavior intact so each imports only Core and retains its own declaration checker, evaluator/admission logic, canonical form, and structured errors.
- Extract search strategy, bounds, budgets, policy, and deterministic selection metadata from the current Query module into `Umpire.Search`.
- Add narrow-import compiler guards proving Property and Behavior do not expose one another or Query.
- Extend the Umpire roots only with modules that now compile; preserve the old modules until final cutover.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/Property.lean:1-218` — Property interface, denotation, evaluator, and errors
- `model/Temporal/Experiment/Behavior.lean:1-218` — Behavior interface, constraints, and errors
- `model/Temporal/Experiment/Query.lean:60-218` — search, bounds, policy, and completeness-adjacent types
- `model/Temporal/Experiment/PropertyTests.lean:213-483` — negative and canonical Property coverage
- `model/Temporal/Experiment/BehaviorTests.lean:169-511` — constraint and unsatisfiability coverage

**Optional** (reference as needed):
- `model/Temporal/Experiment/Query.lean:454-485` — deterministic selection metadata and planner protocol boundary

### Key context
Move only data-only Search contracts here. Planner pull/backend protocol belongs to Planning, and query completeness evidence remains with Query.

## Acceptance
- [ ] Property and Behavior import Core but not each other, Query, Temporal, or Nexus.
- [ ] Narrow-import guards fail if Property, Behavior, and Query become transitively exposed through the wrong root.
- [ ] Search owns the approved data-only strategy/bounds/budget/policy/selection surface without duplicating semantics.
- [ ] Existing positive, negative, canonical, and unsatisfiability tests move with unchanged strength and comments.
- [ ] Structured Property and Behavior errors retain their names, ordering, and canonical representation.
- [ ] `make umpire-check-regression` remains green.

## Done summary
Moved the independent Property and Behavior DSLs and their full positive, negative, canonical, and unsatisfiability suites onto `Umpire.Core`, preserving comments and structured error semantics. Added the data-only Search contract plus narrow-import compiler guards for the sibling module seams.

GATE_SKIPPED:smoke:green-receipt b7283ffc - baseline reused from prior post-gate pass

stage: impl-review - ran (SHIP; completed 2026-08-25T19:23:06.856965Z)
## Evidence
- Commits: 774f1c3d32d0b5950a6cd403f39fc5340c407620
- Tests: GATE_SKIPPED:smoke:green-receipt b7283ffc - baseline reused from prior post-gate pass, cd model && mise exec -- lake build UmpireTests, make umpire-check-regression
- PRs: