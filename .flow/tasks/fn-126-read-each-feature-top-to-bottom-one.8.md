---
satisfies: [R9, R10, R11, R18, R19, R20]
---
# fn-126-read-each-feature-top-to-bottom-one.8 Rename batch, level-name lint, Product and System docs; close

## Description
Also decision 28's task-8 items: Scenario names from their `val`, and the intent note in each feature header and `AGENTS.md`.

Task 6c of the split closing batch: the R18 rename batch with decision 19 and the task-queue rows, R20 (b) level names, R10 retired names, R19 and R9 docs, R11 evidence, and close the spec. Carry-forward constraints (`.flow/tmp/fn-126/carry-forward.md`): `callerSide.close`/`handlerSide.finish` (a recorded deviation), `reply` into `Inputs.reply` in both Nexus features, `result` stays top-level (fix the sketch). Prove with a name map applied to the before-IR and `projtool`. Scope the R10 retired-name check to `model/`, `tools/{umpire,canary}`, `tests/testcore/testpilot`, `common/testing/testpilot` and the R9 docs, word-bounded. Plan sections 3-4.

Plan: `.flow/tmp/fn-126/plan6.md` (read-only planning of 2026-10-05, built after task 4 against task 5 in progress; re-verify against the merged task 5). Host decisions on its open points: IDs follow decision 23's rule literally (section objects included, e.g. `…ActivityRecord.monitors.atMostOneActive`); the Kit's family comes from the lifter substituting the realization's package when it folds a Kit call (no macro); level files keep their subject's own types and signature; `Placebo`'s `inspect` moves into the feature's `object caller` (party unchanged), ending the shadowing; a `shared/` folder has at most one `object exports`. The golden harness is retired by fn-124.7 before these tasks: every proof is by projection (`projtool` + a before/after IR projection), never a golden re-capture.

## Acceptance
- [ ] Every item of this task's description is done, with the plan's verification list for its section passed.
- [ ] Equality proved by projection as described; any table, answer, verdict or fingerprint difference beyond the stated renames stops the task.
- [ ] All gates of the spec's Verification pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
