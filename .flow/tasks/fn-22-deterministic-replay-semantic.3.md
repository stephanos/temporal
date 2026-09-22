---
satisfies: [R3, R9]
---
# fn-22-deterministic-replay-semantic.3 Classify fresh concrete reruns of the subject

## Description
Implement two fresh isolated reruns of the admitted subject through `tools/umpire/binding` (`Open` once, `Bind` and `Run` per attempt, `Release` before the next) and classify: both closed and violated with the subject's key `reproduced`; a completed `satisfied` Verdict or another key `not-reproduced`; an incomplete Run, unclosed cleanup or `inconclusive` Verdict `indeterminate`. A Profile identity other than the subject's is `stale`, an admission failure before any Run; a preparation rejection is an admission failure with no Run. The classifier is a listener-free value like fn-33's `Outcome`; the bridge and the reducer of tasks .4 and .5 consume it unchanged for candidates.

### Approach
- Reuse `binding.Campaign.Bind` and `Bound.Run`; a scripted `Binder` as in `tools/umpire/campaign` tests drives every class.
- History replay has no type and no field; the test pins the report's field set.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/replay/`

**Size:** M
**Files:** `tools/umpire/replay/rerun.go`, `tools/umpire/replay/rerun_test.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK; see the spec's **Re-plan** section. Start only after the spec's fresh plan review.
## Acceptance
- [ ] Every attempt binds fresh state under the exact Profile identity; a stale identity or a preparation rejection creates no Run.
- [ ] Runtime, monitor, cleanup and Verdict outcomes keep fn-64 precedence and map to the three classes without a fourth.
- [ ] The report carries semantic replay, concrete rerun and no history-replay field.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
