---
satisfies: [R4, R5]
---
# fn-166-mini-activity-a-small-complete-model-to.5 make model-mini loop target and the proving-ground rules

## Description
The loop target and the rules of the proving ground (R4, R5).

**Size:** S
**Files:** `Makefile` (`model-mini` target), `MILESTONES.md` (the rules under "Open: Scala DSL work"), `model/README.md` ("Writing a Model": the mini Model as the smallest complete example), `.flow/specs/fn-159-*.md` through `fn-165-*.md` (one line each: iterate on the mini Model; migration note per task; parked Models migrated before close)
**Touches:** [Makefile, MILESTONES.md, model/README.md, .flow/specs/*.md]

### Approach
- `make model-mini`: `MODEL_FEATURES=activity/mini`, then compile, lift `activity-mini.json`, the Go checks on that file, the syntax lint, `git diff --stat model/ir/activity-mini.json`; stop at the first failure; print stage times. Measure three runs with warm caches and record them; the target is under 90 seconds.
- Rules in `MILESTONES.md`: the DSL batch iterates on the mini Model with the parked Models untouched; every DSL task carries a one-line migration note (what the parked Models must change, whether identities move); a DSL spec closes only after the parked Models are migrated per its notes and the existing seal passes on them (steps-only per spec, complete once per batch, as already scheduled for fn-141); the mini Model is never parked.
- README: one paragraph pointing a new author at `features/activity/mini` as the smallest complete Model, with the hard-instance table.

## Acceptance
- [ ] `make model-mini` runs compile, lift, Go checks, lint and the IR diff in order, stops at the first failure, prints stage times, and completes in under 90 seconds on the measured machine with warm caches (three runs recorded).
- [ ] `MILESTONES.md` states the three rules; each DSL spec of the batch carries the one-line pointer; the README names the mini Model as the smallest complete example.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
