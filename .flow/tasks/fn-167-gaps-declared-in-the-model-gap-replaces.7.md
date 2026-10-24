---
satisfies: [R9, R10]
---
# fn-167-gaps-declared-in-the-model-gap-replaces.7 Held sweep: runnable migration, mini proof and real-feature retirement

## Description
**Recut 2026-10-18 (one-language series):** The acceptance file is deleted by fn-200.6. This task's R9/R10 real-feature migration uses fn-200's native lint and `@gap`, not the IR plus `.lint.json`.

Held sweep: integrate the migration adapter and execute mini/real-feature migration with count evidence (R9, R10). This is the actual IR/acceptance-backed proof, not another hand-written acceptance update.

**Size:** M
**Files:** migration adapter/loading configuration, feature declarations and generated trees, model docs.
**Touches:** [model/migrate/**, model/.scalafix.conf, model/temporal/features/**, model/README.md, model/framework/SEMANTICS.md]

### Approach
- Admission prerequisite: conductor verifies fn-166.10's runnable migration infrastructure closure is committed in the current branch. No whole-fn-166 edge is added, because its held final loop consumes the gap migration this task supplies.
- Reuse the committed migration infrastructure supplied by fn-166.10 and task 4's pure transformation. Re-anchor current owner identities/layout before applying it; do not depend on stale capture names or merge source-feature acceptance files back in.
- Run the actual mini one-hole/one-waiver transformation first; compile, lift, lint and compare preserved semantic identities before running it on real features.
- The mini hole receives `@gap`; its existing `except`/`overriding` declaration remains behavior-bearing and supplies the automatically transported waiver reason. Verify that reason without adding a second annotation; mismatched input reasons abort migration.
- Before writes, resolve the exact source-feature paths and reject escapes/symlinks into other features or legacy trees. Stage changes without altering originals; publish only after complete validation with feature-scoped locking and preimage checks, so concurrent edits and failed placement leave the previous feature recoverable. Keep protected legacy inputs outside the writable selection.
- Migrate each source-backed feature transactionally. Remove its acceptance file only after all entries are placed or proven redundant and the replacement passes. Leave legacy payloads and their files to fn-160's retirement owner.
- Record exact before/after raw findings, reported findings and gap counts. Explain owner-rename inventory differences against the captured standalone baseline; never infer reduction from line count alone.

### Investigation targets
Required:
- `.flow/tasks/fn-166-mini-activity-a-small-complete-model-to.10.md` - runnable migration infrastructure closure, separate from final-loop task 5.
- `.flow/specs/fn-160-one-ir-file-per-feature-collected-from.md` — legacy retirement owner.
- `model/.scalafix.conf` — existing rule loading.
- `model/check/Gate.scala` — scoped generation/lint route.
- `model/README.md` — author commands and staged policy.
- `model/framework/SEMANTICS.md` — gap/derivation semantics.

### Verification
Run the real adapter with dry-run/refusal fixtures, then mini and all selected source-backed features. Compare behavior identities, table pins, Cases and acceptance coverage at the settled baseline; archive all count receipts and failed placements. Run parent Quick commands, full relevant Scala/Go suites and equivalence, with no false green credit for inherited failures. Record removed source-feature files and preserve every legacy file until its owner retires it.
## Acceptance
- [ ] Before admission, conductor verifies fn-166.10 done and its runnable migration-infrastructure closure committed in the current branch.
- [ ] Actual mini IR/acceptance-backed adapter proof passes before real-feature writes.
- [ ] Every source-backed feature migrates with preserved reasons, behavior identities and count receipts; unresolved entries abort publication.
- [ ] Source-feature acceptance files are removed only after successful replacement; legacy files/payloads remain untouched and the full authorized gate/equivalence passes.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: