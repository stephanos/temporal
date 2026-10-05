---
satisfies: [R1, R10, R20]
---
# fn-126-read-each-feature-top-to-bottom-one.6 Product and system folders, zoom-ins flattened, structure lint (a)(c)

## Description
Task 6a of the split closing batch (decisions 16, 22; R20 (a) and (c); R10 folders). Give `standaloneactivity/`, `nexuscaller/` and `shared/taskqueue/` a `product/` and a `system/` folder and flatten `record/`, `withTaskQueue/` and `closepolicy/` into `system/` files, under today's pins so every ID is frozen: the reader projection (`projtool`) must be byte-identical and the IR diff only paths, lines, Function symbols and source roots. Land the R20 structure lint (a) and (c) in a sibling pass (`model/irgen/Structure.scala`), the template fixture `model/irgen/testdata/layout/` and one refusal fixture per rule; teach `Order.scala` the level-file role and the R10 layout test the new and retired folders. Re-record pinned Runs if Case identities move. Plan section 2.

Plan: `.flow/tmp/fn-126/plan6.md` (read-only planning of 2026-10-05, built after task 4 against task 5 in progress; re-verify against the merged task 5). Host decisions on its open points: IDs follow decision 23's rule literally (section objects included, e.g. `…ActivityRecord.monitors.atMostOneActive`); the Kit's family comes from the lifter substituting the realization's package when it folds a Kit call (no macro); level files keep their subject's own types and signature; `TrustingCaller`'s `inspect` moves into the feature's `object caller` (party unchanged), ending the shadowing; a `shared/` folder has at most one `object exports`. The golden harness is retired by fn-124.7 before these tasks: every proof is by projection (`projtool` + a before/after IR projection), never a golden re-capture.
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
