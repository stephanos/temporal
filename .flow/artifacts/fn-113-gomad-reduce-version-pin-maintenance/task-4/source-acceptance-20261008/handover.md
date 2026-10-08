# R5/R6 retained source handover

The bump procedure now states that pack refresh reevaluates every mapped target directory before validating and merging a saved impact report. The executed original CLI scratch bump publishes all six adapter outputs through its unchanged default pipeline and passes scratch validation. The conductor still must commit and independently review the candidate before source acceptance or completion.

Task `fn-113-gomad-reduce-version-pin-maintenance.4` remains `in_progress`. Admission base and HEAD are `da1e726eab2d7211ec854df2d20fc2625c0c1695`; the source acceptance base is `9663e4c1bae2c101d452f9ef97e2009143a45e36`. The worker made no commits, review verdict, lifecycle mutation or milestone status change. [evidence.json](evidence.json) indexes the 33 command receipts and five deduplicated source manifests. Final source identity is `7550610348c8f837d0874639ccf8173509b9b2a0437fc7a477897c8e6f8068cb`.

Tier: session (jev-unavailable(no_key))

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

## Source evidence

Counts below are observed named Go test events, including subtests. Static and text-mode command receipts have unavailable test-count parser fields; their zero parser counts and empty failure arrays supply no assertion count.

| Receipt | Exit | Observed scope |
| --- | --- | --- |
| [final-version](final-version-receipt.json) | 0 | 14 pass, zero fail/skip; existing guide assertion extended, with intended [red reproduction](guide-contract-red-receipt.json) |
| [final-authoring](final-authoring-receipt.json) | 1 | 666 pass, one fail, one skip; raw broad command remains RED |
| [resolved-refresh-cache](resolved-refresh-cache-receipt.json) | 0 | Four pass, zero fail/skip; only the exact failed original stock-source test |
| [final-pinimpact-portable](final-pinimpact-portable-receipt.json) | 0 | 121 pass, zero fail/skip; three unchanged qualified wrappers explicitly excluded |
| [final-architecture](final-architecture-receipt.json) | 0 | 78 pass, zero fail/skip |
| [final-validate](final-validate-receipt.json) | 0 | Exact check-only `make -C tools/gomad3 validate` |
| [final-lint-fast](final-lint-fast-receipt.json), [final-version-lint](final-version-lint-receipt.json) | 0 | Mandatory diff-filtered make gate and complete version-package lint, both fixes disabled |
| [final-vet](final-vet-receipt.json), [final-errortype](final-errortype-receipt.json) | 0 | Full affected source scope; errortype uses `-style-check=false` |
| [final-darwin-static](final-darwin-static-receipt.json), [final-linux-static](final-linux-static-receipt.json) | 0 | Both supported source sets, list and vet; no native execution |
| [final-format](final-format-receipt.json) | 0 | Both Go files formatted; `git diff --check` |
| [documentation-audit](documentation-audit-receipt.json) | 0 | Seven documents, 195 resolved relative links, 33 registered verbs and 38 actual help invocations |

[assertion-mapping.json](assertion-mapping.json) names every source test outcome, the authorized one-case rerun and native exclusions. Its scoped authoring union contains 670 passing identities, zero unresolved failing identities and one original qualified-host skip. That union supplies no broad command pass. The original root upgrade publication controls and `TestGenerateRequiresExactApprovalAndCheckDetectsDrift` actually pass in the current authoring log, with unchanged assertions and ordinary workspace TMPDIR.

The one current portable failure occurred while `TestRunCompatibilityPackRefreshResolvesTwoModulesAndKeepsPartialApproval` collected its initial stock capability closure. Its child opened `/home/agent/.cache/go-build/19/...-a` and returned ENOSPC. Root filesystem availability was zero; the assigned workspace had 67GiB available. `target/internal/build/context.go` strips GOCACHE and retains XDG_CACHE_HOME. The conductor admitted only this unchanged test with command-local XDG_CACHE_HOME under the assigned private module cache. [resolved-effective-cache](resolved-effective-cache-receipt.json) removes reserved GOCACHE with the same pinned Go and deterministic settings, then observes the actual `<private-XDG>/go-build` result. The original test passes four events with that cache input. No source fix, shared-cache cleanup, HOME mutation, temp relocation or native exception follows from this admission.

[predecessor-source-reuse.json](predecessor-source-reuse.json) binds the accepted task2 deterministicio workspace complement and filesystem-separated cache-control packet to the current actual test dependency listing. All 120 ordinary dependency/module/test/embed paths are byte-exact. The remaining descriptor file differs only in its unexported `renderUpgradeGuide` function; every byte outside that function is exact, generated identities are exact, and reused sources contain no guide-generator calls. Fresh version, CLI publication and scratch validation cover the changed generator. Reuse preserves the original 269-pass/four-skip complement and 17-pass cache packet as predecessor evidence, not new execution. Original qualified wrappers, skipped external checkout portions and their native owners retain their prior meaning. No closed cleanup investigation was rerun or widened.

## Matched measurement and preservation

[measurement.md](measurement.md), [matched-first-baseline.json](matched-first-baseline.json) and [invocation-accounting.json](invocation-accounting.json) retain the accepted first baseline at `d635e23f00d926a43b942f25a9d05bd0ccb72025`, all four original repair paths and the exact old Sprig v3.3.0 module/sum identity. Only the historical native consumer is removed from its seven units, leaving six source repair units. The current observed repair uses five invocations; normalized repair uses four by omitting the extra JSON dry run. Common candidate preparation is two invocations on each side.

Every actual walk invocation and retry remains counted. The workflow executes nine commands and one recovery hand edit, or ten units. Including all six measurement-driver invocations yields 15 invocations and one edit, or 16 instrumentation-inclusive units. Both failed pre-apply driver setup attempts are explicit. Neither total supplies a savings claim. The historical one-request pack comparison and unexecuted two-request projection remain historical; no fresh pack reduction or qualification was measured.

[walkthrough.json](walkthrough.json) binds the original CLI binary, candidate module inputs, exact scratch approval and six published hashes. [preservation.json](preservation.json) binds the byte-exact scratch surrounding-source manifest, all 1,017 unaffected Gomad paths, root/nested module pins and generated identity, all 12 historical task4 artifacts, the accepted first-baseline bytes and both user-owned untracked files. The worker inspected and approved only this controlled scratch source fixture. This demonstrates the governed source pipeline and supplies no literal human or operational approval of a production pin.

The misplaced first module edit inherited BASH_ENV, which reset the requested child cwd. The harness stopped immediately after observing changed root source; the worker restored only its own go.mod line. Root go.mod/go.sum are byte-exact. Later children remove BASH_ENV and assert physical cwd. This diagnosed shell redirect has no asserted relationship to the baseline Go directory-sync error. The JSON dry run retains its existing approval-omission contract, so the extra human-rendered dry run is counted. Driver output-buffer and gitlink-copy setup errors occurred before apply and are recorded as instrumentation failures.

The baseline Quick test remains RED, including unavailable qualified driver/host wrappers and the original authoring report-parent Sync EACCES. Its text parser cannot count named tests. The exact authoring test passed isolated before edits and in the current full authoring package, without relocation or changed assertions. Its original EACCES cause remains unknown; neither pass supplies a broad-green or repaired-cause claim.

The production Go diff changes one guide-rendering sentence and extends one existing expected-string list. Established generation changed only the upgrade-guide paragraph. README, CLI, SPEC and architecture describe mandatory live discovery plus saved-report merge; architecture now describes the already-required core corpus. SPEC gains the missing row for an already-registered qualification-manifest command. The audit preserves every prior semantic identifier and binds the one intentional documentation-only addition. Grammar, public APIs, runtime patch/overlay, adapter source/token/report/pin/approval behavior and production dependency identities remain unchanged. Roadmap and MILE link the fresh source measurement, with every status and orchestration policy preserved.

## Conductor and native owners

The conductor owns staging/commits, the fresh independent `codex:gpt-6.1-sol:high` review, Flow completion and milestone statuses. No worker SHIP or Done verdict exists. The diff includes executable template source, so classification correctly rejects a docs-only shortcut. No green full-suite receipt was written.

Native Darwin consumer, pack/core/replay, patched-runtime, full-host and qualification measurements remain with fn149.2/.4. Corresponding Linux obligations remain with fn128.4/.7 under the [native transfer manifest](../../../native-scope-transfer-2026-10-07.md). Historical Darwin SHIP, full-test and qualification bytes remain unchanged and establish no current-candidate qualification. The actual SDK checkout prerequisite, missing supplied-checkout workload and fn105 runtime work remain user/native owned. No PR, push or CI action ran.

All worker-started command handles and the read-only documentation researcher are terminal. The shared Go lane is released. The worker will address conductor review findings without changing task state.
