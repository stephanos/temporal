---
satisfies: [R5, R6, R7]
---
# fn-166-mini-activity-a-small-complete-model-to.5 Loop under 90 s, migration as code, the mini feature as the guide's example

## Description
Finish the proving ground: the measured loop, the convention by which a DSL change moves into the real features, and the guide written against the mini feature (R5, R6, R7).

**Size:** M
**Files:** `Makefile` (`model` target stages and timing output; the `model-mini` alias from task 1), `model/.scalafix.conf` and a `model/migrations/` (or the scalafix rules folder the build already uses) for rewrite rules, `model/README.md`, `model/SEMANTICS.md`, `.flow/specs/fn-159-*.md` through `fn-165-*.md` (one sentence each)
**Touches:** [Makefile, model/.scalafix.conf, model/migrations/**, model/README.md, model/SEMANTICS.md, .flow/specs/*.md]

### Approach
- Loop: `make model FEATURES=activity/mini` runs compile, lift `activity-mini.json`, the Go checks on it, the syntax lint, then `git diff --stat model/ir/activity-mini.json`, stopping at the first failure and printing stage times. Measure three warm runs and record them; the target is under 90 seconds. If the floor is the framework and lifter compile, record that against fn-141 rather than trimming the mini feature.
- Migration as code: a location and naming for scalafix rewrite rules under the model build (one rule per DSL task that renames or reshapes a form), applied with the existing `make lint-model-*` fix route. Worked example: a throwaway rename proven on the mini feature (byte-identical IR after the rule) and applied to one real feature, then reverted. One sentence in the README and in each open DSL spec of the batch: a change is proven on the mini feature and ships its rule; a real feature is migrated by running the rule; a spec closes when the full set is current.
- The guide: replace the hypothetical `job` in `model/SEMANTICS.md` with the mini feature, citing its files and lines for every snippet, so the gate compiles what the guide shows. The README's "Writing a Model" names `features/activity/mini` as the smallest complete feature, with the hard-instance table.
- Run the full set once at the end so every real feature is reported current at this spec's close.

## Acceptance
- [ ] `make model-mini` runs compile, lift, Go checks, lint and the IR diff in order, stops at the first failure, prints stage times, and completes in under 90 seconds with warm caches (three runs recorded on the task).
- [ ] A scalafix rule location and naming exist under the model build with one worked example applied to the mini feature and a real feature; the README and each open DSL spec of the batch carry the one-sentence convention.
- [ ] `model/SEMANTICS.md` uses the mini feature as its running example with file citations, and the README names it as the smallest complete feature.
- [ ] A full-set `make model` reports every feature current.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
