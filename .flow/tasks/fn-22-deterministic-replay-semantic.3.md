---
satisfies: [R3, R9]
---
# fn-22-deterministic-replay-semantic.3 Classify fresh concrete reruns of the subject

## Description
Implement two fresh isolated reruns of the admitted subject through `tools/umpire/binding` (`Open` once, `Bind` and `Run` per attempt, `Release` before the next) and classify: both in the admissible violated form with the subject's key `reproduced`; a `COMPLETED` `satisfied` Verdict or another key `not-reproduced`; an `INCOMPLETE` Run, unclosed cleanup or `inconclusive` Verdict `indeterminate`. Stale identity and preparation rejection were decided at admission (task .1) and never reach a Run. The classifier is a listener-free value like fn-33's `Outcome`; the reducer of task .5 consumes it unchanged for candidates.

### Approach
- Reuse `binding.Campaign.Bind` and `Bound.Run`; a scripted `Binder` as in `tools/umpire/campaign` tests drives every class.
- History replay has no type and no field; the test pins the report's field set.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/replay/`

**Size:** M
**Files:** `tools/umpire/replay/rerun.go`, `tools/umpire/replay/rerun_test.go`
**Touches:** `tools/umpire/replay/rerun*.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; revised after plan review round one; see the spec's **Re-plan** and **Plan review** sections. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Every attempt binds fresh state under the exact Profile identity; nothing stale or rejected at admission reaches a Run.
- [ ] Runtime, monitor, cleanup and Verdict outcomes keep fn-64 precedence and map to the three classes without a fourth.
- [ ] The report carries semantic replay, concrete rerun and no history-replay field.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
