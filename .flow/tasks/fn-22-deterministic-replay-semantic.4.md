---
satisfies: [R4, R5]
---

# fn-22-deterministic-replay-semantic.4 Implement bounded monotonic minimization
## Description
Implement the deep deterministic reducer over checked candidate Cases. Try every applicable edit in Lean order, retain an edit only after two fresh Runs reproduce the original semantic violation key, never reintroduce a removed coordinate, and distinguish minimized, irreducible, and bounded-incomplete completion.

**Size:** L
**Touches:** `tools/umpire/replay/minimize.go`, `tools/umpire/replay/minimize_test.go`

## Acceptance
- [ ] Fixed edit, Run, wall-time, Case-byte, event, and report limits are enforced before N+1 work.
- [ ] Compile/preparation rejection, not-reproduced, indeterminate, cancellation, and limit exhaustion remain distinct.
- [ ] Deterministic inputs and semantic outcomes produce the same reduction decisions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
