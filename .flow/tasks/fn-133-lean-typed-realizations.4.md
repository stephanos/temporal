---
satisfies: [R6, R13]
---
# fn-133-lean-typed-realizations.4 Coverage report, class-pattern rule, deadlines bound once, action-level onPath

## Description
Part B, R6, R13.

1. **Settle the class-pattern rule first.** Read the lifter and the Go lowering to find whether `caller.start(scheduleToStart := expires)` names one exact class or every class with that input. Record the answer in the spec (closing its Parked unknown) and in README.
2. **Coverage report.** The gate or lint reports a `perform`/`onPath` binding of a class no realizable path takes (an unreachable binding, e.g. the activity's `deadline.scheduleToClose` await), and a class of a performed action that a Query's path takes but no binding covers. One fixture each.
3. **`deadlines(input -> field, …)`.** One declaration per realization generates the binding of every class of the start or schedule action, combined classes included, and marks classes the server refuses as unrealizable (they show in the report). The activity's and the Nexus caller's per-class variants go.
4. **Action-level `onPath`.** The Nexus caller's three-class list becomes `onPath(caller.schedule)`.

Combined classes may become realizable and add Cases. List each one; no existing Case may change.

## Acceptance
- [ ] The class-pattern rule is recorded in the spec and README.
- [ ] The report's two fixtures fire; the three realizations have no unreachable binding and no uncovered class that is not marked unrealizable.
- [ ] `deadlines(…)` replaces the per-class variants; a declaration naming a non-`Timeout` input is refused at its line.
- [ ] A before/after projection: existing Cases are identical; new Cases are listed.
- [ ] The spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
