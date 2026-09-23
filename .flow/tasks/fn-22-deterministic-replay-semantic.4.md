---
satisfies: [R3, R9]
---
# fn-22-deterministic-replay-semantic.4 Classify fresh concrete reruns of the subject

## Description
Implement two fresh isolated reruns of the admitted subject through `tools/umpire/binding` (`Open` once, after admission; `Bind` and `Run` per attempt; `Release` before the next) and classify: both in the admissible violated form (`STOPPED_BY_MONITOR`, cleanup `SUCCEEDED`, `VIOLATED`) with the subject's key `reproduced`; a `COMPLETED` `satisfied` Verdict or another key `not-reproduced`; an `INCOMPLETE` Run, unclosed cleanup or `inconclusive` Verdict `indeterminate`. Each Run is classed alone by the admissible-form function task .1 defines, and the pair by precedence: any `not-reproduced` makes the pair `not-reproduced`, otherwise any `indeterminate` makes it `indeterminate`, otherwise `reproduced`. Stale identity and preparation rejection were decided at admission (task .1) and never reach a Run. The classifier is a listener-free value like fn-33's `Outcome`; the reducer of task .6 consumes it unchanged for candidates.

### Approach
- Reuse `binding.Campaign.Bind` and `Bound.Run`; a scripted `Binder` as in `tools/umpire/campaign` tests drives every class.
- History replay has no type and no field; the test pins the classifier's value type alone, and task .8 pins the report's field set.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/replay/`

**Size:** M
**Files:** `tools/umpire/replay/rerun.go`, `tools/umpire/replay/rerun_test.go`
**Touches:** `tools/umpire/replay/rerun*.go`

### Re-plan note (2026-09-22)
Rewritten on fn-85, fn-86, fn-87 and fn-33 after the first plan's MAJOR_RETHINK and revised through the six plan review rounds the spec's **Plan review** section records (SHIP on round six, 2026-09-22).
## Acceptance
- [x] Every attempt binds fresh state under the exact Profile identity; nothing stale or rejected at admission reaches a Run; `Open` runs only after admission.
- [x] Runtime, monitor, cleanup and Verdict outcomes keep fn-64 precedence and map to the three classes without a fourth.
- [x] The classifier's value carries the Run's class, the pair's class and the key it compared, and nothing about history replay.
## Done summary
`replay.Rerun(ctx, binder, target)` binds a `Target` (Case, prepared Case, Profile identity, key; `Subject.Target()` gives the subject's own, and task .6 hands a candidate's Case and identity with the subject's key) fresh through a `campaign.Binder` the caller opened after admission, once per attempt (`Attempts` = 2), runs it and releases it before the next attempt binds. Each attempt is classed alone by `ViolatedForm` and `Classify`: in the admissible violated form its key is derived offline through the target's prepared Case (`PreparedCase.Evaluate`, then `KeyOf`) and compared, a Run the offline replay does not reproduce is `indeterminate`, a Run that errs is `indeterminate`; the pair is classed by `ClassifyPair`. A binding under another identity, a failed bind and a failed release are errors, never a class. `Reruns{Key, Attempts, Class}` and `Attempt{Class, Detail, Key, Run, Verdict, Identity}` are listener-free values with no history-replay field, pinned by reflection; a scripted Binder over the conformance corpus drives every class, the precedence, the bind/run/release order, and each refusal.
## Evidence
- Commits: 5d67de32bfa62e242879ea8468724e08088fff70
- Tests: go test -count=1 -tags test_dep ./tools/umpire/replay/, GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: