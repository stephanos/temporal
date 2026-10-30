# Retained candidate ledger

Fixed source: `c88fdc550694020eb1c6a2e09577813d906c9ffe`. Observation: `2026-10-26T15:27:54.064443+00:00`.

This ledger covers **81 local branch refs and 36 observed worktrees**. It records Git/task evidence only; no builds, tests, remote operations, transfers or cleanup occurred. Every recovery ref and checkout remains in place.

The JSON companion contains every committed path/status against each actual merge-base, provisional destination path (via the shared `path_mappings` table), separate dirty observations, commit subjects, tracker/evidence paths, task scope and required final disposition. `candidate-observation.json` remains unchanged.

Only current `model/umpire/` paths map directly to the standalone root. Historical Rust paths require reviewed correspondence; historical Scala and its recovery refs remain in Temporal. Shared files stay in Temporal until reviewed framework content is separately adapted.

Task names below describe corroborated lane intent. Assembly-derived refs also contain inherited work; they do not imply exclusive task ownership. A missing owner stays unresolved.

| Ref | Head | Merge-base | Intent | Committed paths by ownership | Dirty paths | Worktree |
| --- | --- | --- | --- | --- | --- | --- |
| `agent/attempt-fencing-prototype` | `bac2d44038` | `b5736f7cc3` | unresolved | retained-legacy-scala: 19 | none | none |
| `agent/batch2-anchor` | `b6638e9d2e` | `dfbc5e6895` | unresolved | tracker-evidence: 44 | none | none |
| `agent/fn1292-source` | `ee6921caa4` | `dede10e56a` | unresolved | retained-legacy-scala: 15, retained-temporal: 39, shared-or-unresolved: 1, shared-support-or-documentation: 1 | none | none |
| `agent/fn142` | `5676998ad3` | `ee32b5fa60` | unresolved | retained-legacy-scala: 48, retained-temporal: 6, shared-support-or-documentation: 7, tracker-evidence: 4 | none | none |
| `agent/fn1451-current` | `f674900e81` | `d14df5655e` | 145.1 | retained-legacy-scala: 2, retained-temporal: 4, shared-support-or-documentation: 1, tracker-evidence: 3 | none | none |
| `agent/fn1452-current` | `18033cf125` | `1fc67726c9` | 145.2 | retained-temporal: 28, tracker-evidence: 3 | none | none |
| `agent/fn1453-current` | `30872af040` | `5596d43bcc` | 145.3 | retained-legacy-scala: 4, retained-temporal: 33, tracker-evidence: 3 | none | none |
| `agent/fn1454-current` | `ac261f72aa` | `dfbc5e6895` | unresolved | retained-legacy-scala: 48, retained-temporal: 60, shared-or-unresolved: 1, shared-support-or-documentation: 3, tracker-evidence: 2 | none | none |
| `agent/fn146-boundary-review` | `fcb608ecf4` | `55eda3440e` | unresolved | retained-temporal: 3, tracker-evidence: 1 | none | none |
| `agent/fn155-task6` | `2250d1f9ae` | `893ee2f191` | unresolved | retained-legacy-scala: 2, retained-temporal: 1, shared-support-or-documentation: 1 | none | none |
| `checklocks-matcher-data` | `a325f98d35` | `951c5516e9` | unresolved | retained-temporal: 3, shared-support-or-documentation: 2 | none | none |
| `fn-146-adopt-cel-for-runtime-predicates-and` | `10be643d3b` | `84b1f26b30` | unresolved | retained-temporal: 245, shared-support-or-documentation: 5 | none | none |
| `fn-149-safety-and-liveness-groups-for-object` | `2b449cb7b5` | `aee9093f77` | unresolved | retained-temporal: 3 | none | none |
| `fn-156-let-the-scala-compiler-and-lint-enforce` | `dc482b82f8` | `84b1f26b30` | unresolved | retained-legacy-scala: 194, shared-or-unresolved: 1, shared-support-or-documentation: 2 | none | none |
| `main` | `6875191ef7` | `6875191ef7` | unresolved | none | none | none |
| `umpire` | `c88fdc5506` | `c88fdc5506` | fixed source | none | shared-support-or-documentation: 2, tracker-evidence: 21 | temporal |
| `w-1d-core` | `243f9dd525` | `e1f833520c` | unresolved | retained-legacy-scala: 2 | none | none |
| `w-1d-shared` | `86e4ee2505` | `0d534a0d34` | unresolved | retained-temporal: 1 | none | none |
| `w-perf` | `f4fba65eb6` | `a3f3a545bc` | unresolved | retained-legacy-scala: 1, retained-temporal: 31, shared-support-or-documentation: 2 | none | none |
| `w-perf-dev` | `f4fba65eb6` | `a3f3a545bc` | unresolved | retained-legacy-scala: 1, retained-temporal: 31, shared-support-or-documentation: 2 | none | none |
| `work/fn1643-bridges` | `0299b17d72` | `4ff42b9d1c` | 164.3 | retained-legacy-scala: 16 | none | none |
| `work/fn1742-driver` | `4b4fa23c11` | `18262189c3` | unresolved | retained-legacy-scala: 10 | none | none |
| `work/fn1742-temporal` | `ff5d179061` | `18262189c3` | unresolved | retained-legacy-scala: 16 | none | none |
| `work/fn1742-verification` | `6936fab02e` | `18262189c3` | unresolved | retained-legacy-scala: 2 | none | none |
| `work/fn19015-runtime` | `8359d2818b` | `37e268cf28` | unresolved | retained-legacy-scala: 70, shared-support-or-documentation: 1 | none | none |
| `work/fn19522-conformance` | `989d17c61b` | `c172f01ef9` | unresolved | retained-legacy-scala: 4 | none | none |
| `work/fn2092-graal` | `13251fe12b` | `a577d763d9` | unresolved | retained-legacy-scala: 1 | none | none |
| `work/fn2095-wasm` | `6ceb466cf1` | `a577d763d9` | unresolved | retained-legacy-scala: 1 | none | none |
| `work/fn2103-fixtures` | `280fc46eab` | `4458089159` | unresolved | historical-framework: 48 | none | none |
| `work/fn2103-oracle` | `62cfa7fcf3` | `4458089159` | unresolved | historical-framework: 1 | none | none |
| `work/fn2114-deferred` | `fee5fc56ae` | `1e15ffba6c` | 211.4 | framework: 18, retained-temporal: 1, shared-or-unresolved: 1, shared-support-or-documentation: 1, tracker-evidence: 4 | none | none |
| `work/fn2386-interpreted-switch` | `6deb7dcb9d` | `39cdae2691` | 238.6 | framework: 6, retained-temporal: 1, shared-support-or-documentation: 3 | none | none |
| `work/fn24613-protocol` | `5d8c856521` | `354b9735f6` | 246.13 | retained-temporal: 82 | none | fn24613-protocol |
| `work/fn2462-once-entity-lens` | `7de877e5de` | `1df6f6b75d` | 246.2 | framework: 311, retained-temporal: 39, shared-or-unresolved: 3, shared-support-or-documentation: 3 | none | fn2462-once-entity-lens |
| `work/fn2469-baseline` | `d0469357c1` | `d0469357c1` | 246.9 | none | retained-temporal: 1 | fn2469-baseline |
| `work/fn2469-prefix` | `eac683e603` | `d0469357c1` | 246.9 | framework: 8, retained-temporal: 1 | none | fn2469-prefix |
| `work/fn2532-prep-countdown` | `817904c3d0` | `1df6f6b75d` | 253.2 | shared-support-or-documentation: 1 | none | prep2532-countdown |
| `work/fn2533-once-countdown-library` | `fa1feeaff7` | `1df6f6b75d` | 253.3 | framework: 313, retained-temporal: 22, shared-or-unresolved: 3, shared-support-or-documentation: 3 | none | fn2533-once-countdown-library |
| `work/fn254-receipt-counts` | `097e800d72` | `1df6f6b75d` | 254.12, 254.17 | framework: 303, retained-temporal: 13, shared-or-unresolved: 2, shared-support-or-documentation: 3 | none | prep254-receipt-counts |
| `work/fn254-source-assembly` | `7cf05d1c53` | `1df6f6b75d` | 254.5, 254.12, 254.15, 254.16, 254.17 | framework: 345, retained-temporal: 35, shared-or-unresolved: 3, shared-support-or-documentation: 7 | framework: 9, retained-temporal: 1 | fn254-source-assembly |
| `work/fn25410-controls` | `ee01177359` | `208610049c` | 254.10 | framework: 5, retained-temporal: 6 | none | none |
| `work/fn25410-fix-boundary` | `210a333719` | `79111d7fd5` | 254.10 | framework: 2 | none | none |
| `work/fn25410-fix-paths` | `642eb51b63` | `79111d7fd5` | 254.10 | retained-temporal: 6, shared-support-or-documentation: 1 | none | none |
| `work/fn25410-fix-runner` | `6373099ed8` | `79111d7fd5` | 254.10 | framework: 5 | none | none |
| `work/fn25410-native` | `3e389df1df` | `208610049c` | 254.10 | framework: 5 | none | none |
| `work/fn25410-outer` | `f95d524307` | `208610049c` | 254.10 | retained-temporal: 6, shared-support-or-documentation: 2 | none | none |
| `work/fn25411-standalone` | `6660073f42` | `7ec6525f6f` | 254.11 | framework: 1, tracker-evidence: 47 | none | fn25411-standalone |
| `work/fn25412-budget-prep` | `4835a4734a` | `1df6f6b75d` | 254.12 | framework: 4, retained-temporal: 1 | none | prep25412-budgets |
| `work/fn25412-budget-wiring` | `f7ad9c4e58` | `1df6f6b75d` | 254.12 | framework: 340, retained-temporal: 34, shared-or-unresolved: 3, shared-support-or-documentation: 7 | none | fn25412-budget-wiring |
| `work/fn25413-driver` | `86f8126927` | `abd5107b8e` | 254.13 | framework: 52, retained-temporal: 30, shared-or-unresolved: 4, shared-support-or-documentation: 4 | none | none |
| `work/fn25413-paths` | `23c3d48819` | `abd5107b8e` | 254.13 | framework: 36, retained-temporal: 3 | none | none |
| `work/fn25413-scripts` | `0bd07ad88d` | `abd5107b8e` | 254.13 | framework: 14, retained-temporal: 8 | none | none |
| `work/fn25414-probes` | `5ba482b5b7` | `234dcd5c6f` | 254.14 | framework: 11, retained-temporal: 2, shared-or-unresolved: 1, shared-support-or-documentation: 1 | none | fn25414-probes |
| `work/fn25415-native-controls-v2` | `022d614b6f` | `1df6f6b75d` | 254.15 | framework: 307, retained-temporal: 19, shared-or-unresolved: 2, shared-support-or-documentation: 3 | none | fn25415-native-controls-v2 |
| `work/fn25415-prep-proto` | `6c726af351` | `eb881b03ac` | 254.15 | framework: 11, retained-temporal: 1 | none | prep25415-proto |
| `work/fn25415-prep-scripts` | `4d913e06d0` | `eb881b03ac` | 254.15 | framework: 19, retained-temporal: 11, shared-or-unresolved: 1 | none | prep25415-scripts |
| `work/fn25415-shell-finish` | `a5c556ca2b` | `1df6f6b75d` | 254.15 | framework: 308, retained-temporal: 18, shared-or-unresolved: 2, shared-support-or-documentation: 5 | none | fn25415-shell-finish |
| `work/fn25416-prep-tools` | `31e989d809` | `eb881b03ac` | 254.16 | framework: 18, shared-or-unresolved: 1, shared-support-or-documentation: 1 | none | prep25416-tools |
| `work/fn25417-mise-adapter-v2` | `ce5996f4d8` | `1df6f6b75d` | 254.17 | framework: 335, retained-temporal: 33, shared-or-unresolved: 3, shared-support-or-documentation: 5 | none | fn25417-mise-adapter-v2 |
| `work/fn25417-prep-controls` | `22b3a6b3a5` | `eb881b03ac` | 254.17 | framework: 5, retained-temporal: 3 | none | prep25417-controls |
| `work/fn25417-prep-gate` | `64d2a62e86` | `eb881b03ac` | 254.17 | framework: 4, retained-temporal: 2 | none | prep25417-gate |
| `work/fn25417-prep-mise` | `c9e91a4750` | `eb881b03ac` | 254.17 | framework: 4, retained-temporal: 1, shared-or-unresolved: 1, shared-support-or-documentation: 3 | none | prep25417-mise |
| `work/fn25417-prep-nextest` | `42d06a0c6d` | `eb881b03ac` | 254.17 | framework: 7, retained-temporal: 3, shared-or-unresolved: 1 | none | prep25417-nextest |
| `work/fn2543-nextest` | `d49485e91f` | `6d7cbebe21` | 254.3 | framework: 7, retained-temporal: 2, shared-or-unresolved: 1, shared-support-or-documentation: 2, tracker-evidence: 584 | none | none |
| `work/fn2544-author` | `a9de3aad7d` | `76408c5585` | 254.4 | framework: 49 | none | none |
| `work/fn2544-backends` | `ad1d8f8847` | `76408c5585` | 254.4 | framework: 46 | none | none |
| `work/fn2544-catalogs` | `845eba5065` | `76408c5585` | 254.4 | framework: 41 | none | none |
| `work/fn2544-runtime` | `02f4279a61` | `76408c5585` | 254.4 | framework: 38 | none | none |
| `work/fn2545-prep-compiler` | `b4b9b05cde` | `eb881b03ac` | 254.5 | framework: 1, retained-temporal: 2 | none | prep2545-compiler |
| `work/fn2545-prep-fixtures` | `088e98ecae` | `eb881b03ac` | 254.5 | framework: 5, retained-temporal: 2 | none | prep2545-fixtures |
| `work/fn2545-prep-parser` | `77af035124` | `eb881b03ac` | 254.5 | framework: 262, retained-temporal: 4 | none | prep2545-parser |
| `work/fn2546-quick` | `0c41b85a49` | `46e96cfa5c` | 254.6 | framework: 2, retained-temporal: 1, shared-support-or-documentation: 1, tracker-evidence: 28 | none | fn2546-quick |
| `work/fn2551-prep-generator` | `d9a378401b` | `1df6f6b75d` | 255.1 | framework-experiment-in-tracker-storage: 4, shared-support-or-documentation: 1 | none | prep2551-generator |
| `work/fn2552-once-differential` | `71d33377ee` | `1df6f6b75d` | 255.2 | framework: 321, framework-experiment-in-tracker-storage: 4, retained-temporal: 22, shared-or-unresolved: 3, shared-support-or-documentation: 4 | none | once2552-differential |
| `work/fn2553-prep-fuzz` | `1df6f6b75d` | `1df6f6b75d` | 255.3 | none | framework: 45 | prep2553-fuzz |
| `work/fn2554-prep-mutants` | `a573caade1` | `1df6f6b75d` | 255.4 | framework: 4 | none | prep2554-mutants |
| `work/fn2555-once-miri` | `9583f7eef7` | `1df6f6b75d` | 255.5 | framework: 319, retained-temporal: 22, shared-or-unresolved: 3, shared-support-or-documentation: 3 | none | fn2555-once-miri |
| `work/fn2556-coverage` | `daee451d5c` | `1df6f6b75d` | 255.6 | framework: 316, retained-temporal: 22, shared-or-unresolved: 3, shared-support-or-documentation: 3 | none | once2556-coverage |
| `work/fn2556-coverage-fixes` | `91847b54dc` | `1df6f6b75d` | 255.6 | framework: 351, retained-temporal: 35, shared-or-unresolved: 3, shared-support-or-documentation: 7 | none | fn2556-coverage-fixes |
| `work/fn2561-retirement` | `d9a58c62dd` | `71ff747ebb` | 256.1 | retained-temporal: 167, shared-support-or-documentation: 5, tracker-evidence: 26 | none | fn2561-retirement |
| `work/fn2562-once-retirement` | `f6e39420c2` | `1df6f6b75d` | 256.2 | framework: 311, retained-temporal: 23, shared-or-unresolved: 3, shared-support-or-documentation: 5 | none | fn2562-once-retirement |

## Required final disposition

Every entry remains **not transferred**. Its existing ref and exact head are the recovery anchor; the JSON deliberately leaves destination recovery refs and tracker records empty. Before source cutover, record the receiving repository, mapped commit/hunks, destination recovery ref, destination tracker record, retained source evidence links and cross-repository prerequisites. Keep unfinished obligations open. Mixed commits and inherited assemblies require selective integration rather than wholesale replay.

Preserve frozen `work/fn2469-baseline` at its recorded head with its dirty regression control. Preserve `work/fn2114-deferred`, `work/fn2386-interpreted-switch`, historical Rust/Scala recovery refs and unknown-owner branches. Apparent integration, a closed spec or numeric similarity is not cleanup authority.

Dirty and untracked bytes have only been observed, not backed up. The source-assembly worktree contains an unresolved index; fuzz contains uncommitted framework source and untracked fuzz files. Unique ignored source and evidence have not been inventoried. Preserve all of them in place until exact recoverability has been established.

## Prerequisites retained

- **validation-hold:** No new builds/tests; current-source verification remains required before final cutover. Evidence: `MILESTONES.md; fn-257 spec R5/R6`.
- **fn254-temporal-consolidation:** fn-254.7 requires committed fn-246 spec closure; fn-254.8 consumes .7 and remaining gate producers. This does not block source export or every fn-254 lane. Evidence: `.flow/specs/fn-254-fast-validation-gates-and-a-real-inner.md:136-139`.
- **fn255-tier-wiring:** fn-255.7 requires committed fn-254.6, .10 and .12; new stages preserve budgets and standalone operation. Existing individual source lanes remain separate. Evidence: `.flow/tasks/fn-255-thorough-framework-testing-generated.7.md:11`.
- **fn256-live-consumer:** fn-256.1 remains gated on fn-246.9 whole-prefix consumer repair, current Temporal guest/policy renewal and actual full live proof; .2 consumes .1. Shared Makefile/CI/pin/guidance changes land serially with fn-254. Evidence: `.flow/tasks/fn-256-retire-the-scala-era-build-and-go.1.md:42-44; .2.md`.
- **fn253-activity-seam:** Generic countdown .2-.5 may proceed independently; activity changes require committed fn-246.13/.15 corrections and retain their Temporal ordering. Preserve the unresolved compatibility-control/ownership decisions. Evidence: `MILESTONES.md:187-209`.
- **fn246-live-followups:** Keep fn-246 refinement, fn-237.4 timeout/cancellation, fn-243.5 R40 and fn-237.5 documentation dependencies open; current prefix/protocol checkpoints are not full live umbrella closure. Evidence: `MILESTONES.md:122-169`.
- **deferred-backends:** fn-211 remains deferred; preserve fn-211.4 consumer and fn-211.5 ownership/set-valued-state obligations. Deferred work is not a source-export prerequisite. Evidence: `MILESTONES.md:228-239`.

## Material gaps

- No destination candidate recovery refs or tracker records have been created; all transfer dispositions remain incomplete.
- Dirty/untracked/ignored source is preserved in place; observation is not a backup. Expand unique ignored evidence/source and verify recoverability before any later retirement.
- Old branch owners not established by inspected task/commit evidence remain unresolved; preserve them even where a spec is closed or a patch appears integrated.
- Net merge-base deltas on assembly-derived branches include inherited candidates; avoid duplicate replay and establish commit/hunk ownership before integration.
- Final current-source standalone/paired validation is deferred under the owner hold; existing historical proofs do not close migration or other specs.

Other existing remote/tag/stash/backup/original refs observed locally: 217. Their exact names/objects are retained in the JSON; no remote contents or permissions were queried.

Structural checks confirmed observation hash, all 81 local branch heads, 36-worktree coverage and open transfer dispositions. These are artifact/Git checks, not build or test evidence. Tracked `.flow/tmp/fn255-spike` files include framework experiment source; preserve them with provenance and resolve any later destination explicitly.
