---
satisfies: [R4]
---
# fn-196-the-scala-checker-is-the-oracle.3 DSL completion: measure the Scala checker and decide the Go deletion and the lowering location

## Description
**DSL completion phase (owner, 2026-10-17).** Write `model/design/scala-checker-measurement.md`: wall time and live heap of `verifyAll` on the six samples at their intended budgets and on the largest graphs the parked Temporal sources in the equivalence testdata allow (replayed natively where they compile against the current DSL, otherwise the largest sample scaled by its parameters, fn-194). Compare with the recorded Go numbers where any exist. If memory asks for it, key the visited set by a canonical state hash and re-measure. End with a go or no-go on deleting the Go check, and a decision on where Case lowering runs (Go over the IR, or Scala emitting the Case proto) with the reasons. List the IR-only checks (refinement, rule disjointness, totals) as kept, moved or dropped.

**Size:** M

**Files:** `model/design/scala-checker-measurement.md`, optionally `model/framework/checker/Visited.scala` for the hash-keyed set, a measurement script kept beside the note.

### Verification
- The note carries a table per graph (states, steps, seconds, live heap) and the two decisions.
- The measurement script reruns from the note's instructions.

## Acceptance
- [ ] Measurement note with per-graph numbers and the Go-deletion go/no-go
- [ ] Lowering location decided with reasons
- [ ] IR-only checks listed as kept, moved or dropped

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
