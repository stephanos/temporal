---
satisfies: [R5]
---
# fn-126-read-each-feature-top-to-bottom-one.7 Definition IDs are fully qualified Scala names; pins and families removed

## Description
Task 6b of the split closing batch (decisions 23, 24, 25). Also: the capabilities section is the declaration and `exports` may name a `queries` section (decision 24, no `val all`); machine-level sections drop `extends Section` (decision 25). Make every Definition ID the declaration's fully qualified Scala name and the IR family the declaring package: rewrite `Context.definitionId`, remove `DefinitionScope`, the pin lookup, section transparency, `Family` givens and `…Family` objects, the family arguments of derivations and compositions; keep section placement refusals; add a whole-index check of derived IDs (machine and Query names unique per package). Invert and regenerate the fixtures. Prove only IDs changed with an ID map (`project6.py`): a bijection applied to the before-IR, then `projtool` on mapped-before vs after (fingerprints recomputed, never mapped), and Cases regenerated from the mapped IR in a scratch worktree. Re-record pinned Runs. The lifter, `model/umpire` and fixture work may be written alongside 6a in its own worktree; its Model edits and regeneration wait for 6a to merge. Plan section 1.

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
