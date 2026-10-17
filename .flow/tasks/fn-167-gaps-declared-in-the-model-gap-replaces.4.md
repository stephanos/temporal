---
satisfies: [R9]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.4 Pure Scala gap migration transformation and copied-mini source proof

## Description
Build the deterministic Scala migration transformation and copied-mini source proof (groundwork for R9). The executable adapter and full IR-backed proof remain task 7.

**Size:** M
**Files:** new migration core/tests/fixtures; mini-only source proof; docs.
**Touches:** [model/migrate/**, model/temporal/features/activity_mini/**, model/README.md, model/framework/SEMANTICS.md]

### Approach
- Admission prerequisite: conductor verifies fn-166.3 is done with its closure committed in the current branch. The CLI does not support cross-spec task edges; this prerequisite is not replaced with a whole-spec dependency that would create a sweep cycle.
- Consume existing IR/acceptance shapes plus resolved placement/provenance from tasks 2–3. Keep input parsing separate from the pure transformation; do not create a public sidecar format.
- Group by concrete declaration and exact reason, preserving reason text and existing source formatting outside insertion spans. Resolve old owner names explicitly; do not guess nearest declarations.
- Remove redundancy only when the supplied provenance proves the corresponding inherited finding equivalent. Return every unresolved acceptance or unsafe reduction as a diagnostic, with no partial source publication.
- Prove the annotated mini hole and preserve the existing behavior-bearing waiver with its exact reason using copied inputs and source fixtures, then compile/re-lift the result. Reconcile capability-waiver acceptance entries with `except`/`overriding`, report mismatched reasons, and never insert a second waiver annotation. Keep real sources/artifacts and all existing acceptance payloads untouched. Coordinate the future adapter location with fn-166's migration-as-code convention without depending on its whole held closure here.

### Investigation targets
Required:
- `.flow/tasks/fn-166-mini-activity-a-small-complete-model-to.3.md` — committed mini protocol producer.
- `.flow/tasks/fn-166-mini-activity-a-small-complete-model-to.5.md` — migration convention owner.
- `model/irgen/Context.scala` — resolved declarations and positions.
- `model/check/Gate.scala:1402` — current acceptance representation.
- `tools/umpire/lint/accept.go:88` — exact grouping identity.

### Verification
Add proposed `umpire.migrate.GapsSuite` with reason preservation, multiple reasons, inherited duplicates, unmappable owners/subjects, conflicting insertions and idempotent second application. Pin original inputs and non-insertion source bytes. Compile/lift copied mini output and pin its step tables. Do not claim full R9 from fixtures alone; task 7 must run the actual IR/acceptance-backed adapter.
## Acceptance
- [ ] Before admission, conductor verifies fn-166.3 done and its closure committed in the current branch.
- [ ] Transformation fixtures preserve reason text, source bytes outside insertions and original inputs; repeat application is idempotent.
- [ ] Unplaceable or unsafe reduction inputs fail without partial publication.
- [ ] Copied mini one-hole/one-waiver output compiles/lifts with equivalent tables; actual adapter/IR-backed R9 remains task 7.

## Done summary
Built the deterministic Scala gap migration (`model/migrate/rules/Fn167Task4Gaps.scala`) with its input adapter for the existing `<file>.lint.json` and `<file>.waivers.json` shapes kept apart (`Fn167Task4GapInputs.scala`); no sidecar format. Admission: fn-166.3 verified done with its closure in history.

baseline: green — fn-167.3 receipt state; mini selected lift bytes in .flow/tmp/fn167/task2/before.
stage: impl-review - skipped(config: REVIEW_MODE=none; owner gates independent review once at Scala batch end)

Requirement evidence (R9 groundwork):
- Transformation: one `@gap` per declaration and exact reason in acceptance order, reason text escaped and preserved, inserted above the declaration (before existing annotations) with every other byte unchanged; old owners resolve only through the caller's explicit placement map (no nearest-declaration guessing). Proven-inherited findings (fn-167.3 base keys) are dropped only when the base is itself accepted. Capability-waiver acceptances are reconciled with the lifted except/overriding waivers and never inserted; mismatched or missing reasons are reported. Duplicates, unresolved keys, missing/ambiguous/mid-line declarations and conflicting existing gaps refuse the whole run with no source returned. Second application converges.
- `umpire.migrate.GapsSuite` 7/7 (suite-final.log): reason preservation incl. quotes/backslash/unicode/tab, multiple reasons, grouping, byte preservation and untouched inputs, idempotence, inherited drop and unsafe reduction, unmappable owner/undeclared/duplicated declaration/duplicate acceptance, conflicting insertions, waiver reconcile/mismatch/missing, input parsing including the mini fixture.
- Copied-mini proof `prove-mini.sh umpire.migrate.RewriteGapsMini compare` exit 0 (proof-final.log): inserts `@gap(unproduced(Fact.statusScheduled))` with its reason on `ActivityProduct`, drops both mini capability waivers as lowered (exact reasons from the copy's own lift), converges, refuses a reworded waiver; the rewritten copy compiles and lifts (so the gap passes source admission and resolution), complete IR equals the original but for 116 positions below Product.scala:28 moved by 3 lines, waivers byte-identical. Real sources and all acceptance payloads untouched.
- `prove-mini.sh` now lifts each copy from the copy's own directory: previously the lifter read the authored sources by repo-relative path, which broke any line-inserting proof (R4.dotted-in at shifted lines, proof-2.log). The unchanged `before` lift is byte-identical to the earlier repo-cwd lift (before-cmp.log).

Verification: GapsSuite 7/7; copied-mini proof PASS; before-lift cmp; scalafmt --check on new Scala files; git diff --check.

Failures retained: proof-1.log (scratch-path depth), proof-2.log (authored-source reads by the lifter, fixed in prove-mini.sh), proof-3.log (sandbox filesystem went read-only mid-run; rerun after restart).

Open obligations (task 7): the runnable IR/acceptance-backed adapter that resolves finding keys to declarations and typed subjects, real-feature migration and acceptance-file retirement, and the before/after finding and gap counts (R10). Full R9 is not claimed from fixtures.

Deferred gates: Go/proto/regeneration, lint acceptance, real features, broad validation and independent review at batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 050000fb5e64bf99163473de9c6137147be9392c
- Tests: timeout 1800 mise exec -- scala-cli test --server=false model/migrate --test-only umpire.migrate.GapsSuite (7/7), timeout 1800 /bin/bash model/migrate/prove-mini.sh umpire.migrate.RewriteGapsMini compare (PASS), cmp copied before lift vs .flow/tmp/fn167/task2/before/lift (identical), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check <new Scala files>, git diff --check
- PRs: