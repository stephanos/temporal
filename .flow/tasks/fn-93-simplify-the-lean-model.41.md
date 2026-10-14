---
satisfies: [R15]
---
# fn-93-simplify-the-lean-model.41 Strip history narrative from Lean comments and trim Nexus test prose (G3, G4)

## Description
Lanes G3 and G4. Remove the 29 `fn-NN` comment lines (21 files), task `.N` references, dated decisions, the commit pin and the timing log; keep current deferral pointers (e.g. "cancellation deferred to fn-79"). Fix the three stale claims (`Nexus/Tests/Commands.lean:14-15`, `Case/Schema.lean:26`, `TemporalModelTests.lean:26-28`). Trim the block prose in `Success/Tests` (1,889), `Tests/Machines` (847), `Tests/Commands` (493), `Caller/Tests` (415) to one line per pinned behavior.

**Size:** M
**Files:** comment-bearing Lean files (`grep -rn 'fn-[0-9]' model --include='*.lean'` minus `.lake`), `model/Temporal/Feature/Nexus/Success/Tests.lean`, `model/Temporal/Feature/Nexus/Tests/{Machines,Commands}.lean`, `model/Temporal/Feature/Nexus/Caller/Tests.lean`, `model/Temporal/Case/Schema.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/**/*.lean]

### Approach
- Comments only: `git diff --stat` must show comment/blank-line changes only (check with a diff filtered to non-comment lines = empty). Nothing pins source positions in these files (commands use `sourceLocation path 1 1`), but re-check that no `#guard_msgs` output contains a line number before trimming.
- The spec's rule: keep comments that say why.

### Quick commands
```sh
grep -rn 'fn-[0-9]' model --include='*.lean' | grep -v .lake
cd model && lake build
```

## Acceptance
- [ ] No Lean comment carries a spec/task history reference other than a current deferral pointer
- [ ] Three stale claims corrected; Nexus test prose one line per pinned behavior
- [ ] Only comments changed; every root builds


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
