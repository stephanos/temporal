---
satisfies: [R3]
---
# fn-113-gomad-reduce-version-pin-maintenance.2 Regenerate adapter anchors for a new module version behind an approval digest

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Platform-aware pin/pack behavior, unavailable-platform refusal/unknown handling, measured steps, source reconciliation, Darwin gates and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

One governed `gomadtool` command that re-derives an adapter's rewrite and digest anchors for a new exact module version (R3).

**Size:** M
**Files:** new subcommand file under `tools/gomad3/cmd/gomadtool/`, `tools/gomad3/deterministicio/adapter_rewrite.go`, `adapter_copy.go`, the regenerated `*_adapter.go` and `toolchain/version/version.json`, fixtures
**Touches:** [tools/gomad3/cmd/gomadtool/**, tools/gomad3/deterministicio/**, tools/gomad3/toolchain/version/**, tools/gomad3/internal/compatibilitypack/**]

### Approach
- Extend the existing rewrite helpers; do not add a second rewrite engine.
- Dry run: fetch the new exact version into a private cache, apply each rewrite by its existing exact-occurrence anchor, and print the changed upstream source for the rewritten files plus the proposed new anchors and an approval digest over both.
- Fail without writing when an anchor matches zero or more than one time, or a rewritten file no longer exists upstream.
- Apply: with the matching approval digest, build the complete output set in a scratch copy first: adapter constants, the version descriptor entry, pinned-digest test data, and every file `make generate` derives from them. Verify the staged set, then publish it.
- The transaction boundary includes the generated outputs. Publication takes an exclusive lock, revalidates that the checkout files it read are unchanged since staging, and records a marker so an interrupted publication is completed or rolled back by the next run. A generation failure in the scratch copy publishes nothing.
- Report libc-bound packs the change leaves stale; the pack refresh in task 3 repairs them.
- Regenerate one real adapter: the first adapted module the root `go.mod` has moved past when the task starts. If none has moved, state that and rely on the fixture.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/deterministicio/adapter_rewrite.go:17-118` — `sourceRewrite`, `anchorRewrite`, `prepareRewrittenModule`, `rewriteAdapterSource`
- `tools/gomad3/deterministicio/adapter_copy.go:144` — source inventory digest
- `tools/gomad3/deterministicio/grpc_adapter.go` — the adapter with the most anchors
- `tools/gomad3/internal/compatibilitypack/authoring/generate.go` — approval-digest pattern to mirror

**Optional** (reference as needed):
- `.flow/memory/bug/integration/profile-adapter-changes-leave-libc-2026-10-01.md`
- `tools/gomad3/deterministicio/adapter_rewrite_test.go:239-279` — module download in tests

### Key context
- fn-109 tasks 10 and 11 change the registry and the source-inventory owner (spec Open Questions 1). Check their state and build on whichever landed.
- fn-112 task 9 consolidates the adapter test family; coordinate edits to those tests.
- Changing an adapter changes target identity; `.bin/gomad` must be rebuilt before qualification.
### Source correction admitted after the task1 report checkpoint

The reviewed task1 source correction is integrated at ce126fe8d308da268fecf14e57dc7832a46c5084. Task1's original native/full/comparator acceptance and this task's dependency remain open. Admit only the independent cleanup correction in the existing task2 verifier implementation. This is source progress, not completion of task1 or task2, and retains the original delivery and qualification requirements.

**Touches:** [tools/gomad3/deterministicio/adapter_regenerate.go, tools/gomad3/deterministicio/libc_regenerate.go, tools/gomad3/deterministicio/adapter_verify_cleanup_test.go, .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/verifier-cleanup-progress/**]

VerifyRegisteredAdapter and verifyLibcAdapter each ignore their deferred scratch RemoveAll error. Check those two returns exactly once at their existing release boundary, following the conditional composition used by target/adapter_source_set.go and artifact/open.go. Successful cleanup preserves the exact primary error object and unwrap shape. A sole cleanup failure returns directly; simultaneous failures join primary first. Checking those formerly ignored failures introduces a bounded additional error surface. Preserve public signatures, existing comments, verification/validation ordering, selected module/source pins, preparation and source-set logic, CLI/schema semantics, existing native guards and generated/toolchain inputs. Do not change transaction publication, Lock.Release, removeWritable or adapter regeneration policy.

Use TDD through the public VerifyRegisteredAdapter function for both a registered rewritten adapter and libc. Existing cached pinned Sentry and libc trees and the current pinned Go release provide portable source-listing controls without executing a patched target. Keep tests separate from pinnedReleaseGo/newRegenerationFixture and never spoof .toolchain/bin/go. A supplied fixture goCommand may emit malformed listing JSON and create a nonempty mode-000 directory within the verifier's owned scratch tree. This must produce a real RemoveAll permission failure on an unprivileged host. Register checked permission restoration and cleanup before invoking verification. Probe actual permissions and explicitly skip unsupported/root permission cases rather than claiming the fault was exercised.

Retain unchanged-source RED for simultaneous primary/listing and real cleanup failure. Verify the primary-only malformed-listing error retains its existing message and single-error unwrap shape when cleanup succeeds. Final controls must include successful verification and complete removal, primary-only failure and removal, simultaneous primary-first failure with errors.As reaching json.SyntaxError and os.PathError plus errors.Is(fs.ErrPermission), and, where the real listing wrapper supports it, successful verification followed by a sole cleanup failure returned directly. Both qualified source sets are statically listed with the pinned release; this supplies no native workload qualification.

Retain frozen source/tool/config bindings, command exits, timings and compact RED/GREEN receipts. Compare unfiltered scoped lint diagnostics on unchanged and corrected source and remove exactly the two owned errcheck findings without new diagnostics or suppressions. Run relevant portable tests, scoped vet and errortype, formatting, check-only make -C tools/gomad3 validate, architecture boundaries and mandatory make lint-code-fast with fixes disabled. Keep every inherited native/patched-driver, original full/default/functional/affected-consumer and formal gate explicitly open. Do not repeat unchanged unavailable-driver checks.

A fresh independent source-progress review must inspect conditional error composition, exactly-once cleanup, real fault execution and permission restoration, unchanged pins/guards and evidence scope. The conductor retains lifecycle, review and a separate source-progress commit. fn-109 R19 consumes this improved lint evidence; fn-109.39 retains only its existing target-owned cleanup scope. No duplicate owner is added. Linux qualification remains deferred and unverified under fn128.

### Publication-aware scratch cleanup source admission

The verifier cleanup checkpoint is integrated and independently reviewed at cefbe17351. Admit a separate source correction under this existing R3 owner for the two removeWritable calls in upgrade/adapterregen. Task1's dependency and all original native/full/formal acceptance remain open.

**Touches:** [tools/gomad3/upgrade/adapterregen/adapterregen.go, tools/gomad3/upgrade/adapterregen/transaction.go, tools/gomad3/upgrade/adapterregen/*cleanup*_test.go, .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/publication-cleanup-progress/**]

Run and apply must check scratch cleanup once at their existing deferred boundaries. removeWritable must handle traversal and chmod errors while continuing its existing traversal and always attempting RemoveAll. Nil cleanup preserves the exact primary error and returned value. Before completeJournal succeeds, a sole cleanup failure returns directly and simultaneous failures join primary first, preserving InputError and BlockedError classification. After completeJournal succeeds, stage cleanup appends publication warnings and outer download cleanup appends Result.Warnings. Preserve Applied, Published, Staged, previous warnings, and a nil operation error. Keep stage cleanup, existing lock release, and download cleanup in their current release order. Leave both Lock.Release calls, recovery decisions, journal retirement branches, pins, comments, public signatures, schemas, generators, native guards and toolchain inputs unchanged.

Use public Run tests with fixture checkout and file proxy outside a private TMPDIR. Existing afterStage and residualScan seams may make that private parent non-writable after staging or publication. This must cause actual scratch-root unlink EACCES on an unprivileged host. Register checked restoration before Run, verify permission denial and retained scratch roots, and skip unsupported/root permission cases explicitly. Test primary-first failure with both cleanup causes and unchanged checkout, stage-only sole cleanup failure, completed publication with ordered stage/download warnings, and preservation of an earlier scan warning. Include no-fault controls for exact primary identity/result preservation, removal and lock reacquisition. Do not add a production fault seam.

Retain unchanged-source RED, final GREEN, frozen source/tool/config bindings and unfiltered scoped lint delta without suppressions. Run portable package tests, scoped vet and errortype, formatting, check-only validate, architecture tests and mandatory fast lint with fixes disabled. A fresh independent source-progress review checks actual fault execution, publication preservation, error composition, release ordering and evidence scope. The conductor owns lifecycle and a separate source-progress commit. Original native/full/default/functional/affected-consumer and formal gates remain open. Linux qualification remains deferred under fn128. No task completion or formal SHIP follows from this checkpoint.

### Checkout type drift source admission

The reviewed source checkpoints are integrated through 717a2de678. Admit only a pre-publication checkout revalidation correction under the existing R3 owner. Task1's dependency and all original native/full/formal acceptance remain open.

**Touches:** [tools/gomad3/upgrade/adapterregen/transaction.go, tools/gomad3/upgrade/adapterregen/checkout_revalidation_test.go, .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/checkout-revalidation-progress/**]

Investigate whether revalidate follows a same-byte symlink introduced after staging and allows publication to overwrite the user's replacement. Reuse the existing public Run fixture, file proxy, generator/verifier and afterStage seam. Retain an unchanged-production RED before fixing. Apply the existing copyCheckout regular-file policy in revalidate after its appeared check and before fileDigest. Preserve appeared/changed error precedence, BlockedError classification, byte-based regular-file identity, publication ordering, locks, recovery, journal retirement, pins, generators, native guards, public signatures, comments and schemas. Do not broaden fileDigest or Recover semantics, add production fault seams, or claim closure of the existing revalidation/publication race.

Controls must prove unchanged regular publication and same-byte regular replacement remain allowed; changed regular contents remain blocked; same-byte symlink replacement remains intact and unpublished, with external bytes and other checkout content preserved; newly added symlinks retain appeared precedence; initial symlinks remain rejected. Require no publication marker or generated-output writes on refusal. Exercise real filesystem symlinks; explicitly skip only if the host cannot create them.

Retain frozen source/tool/config bindings, RED/GREEN command exits and timings. Run all portable adapterregen tests, scoped vet/errortype, formatting, check-only validate, architecture boundaries, unfiltered scoped lint comparison and mandatory changed-line fast lint with fixes disabled. Report inherited full lint failures separately. A fresh independent source-progress review must check the public fault, regular-file compatibility, refusal before publication, unchanged recovery and evidence scope. The conductor retains lifecycle and a separate source-progress commit. Source progress supplies no formal SHIP or task completion; native Darwin/full/default/functional/affected-consumer requirements remain open where unproved. Linux qualification remains deferred and unverified under fn128.

## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] A dry run prints changed upstream source, proposed anchors, and an approval digest, and writes nothing
- [ ] Apply with the matching digest updates constants, descriptor, and pinned test data together; a wrong digest writes nothing
- [ ] Negative fixtures: moved anchor, anchor matching twice, rewritten file deleted upstream, generation failure in staging, interrupted publication, checkout changed between staging and publication, and two competing apply operations
- [ ] After any failed or interrupted apply, the checkout holds either the old complete set or the new complete set, including generated outputs
- [ ] One real adapter is regenerated across a version bump and its workload qualifies, or the done summary states that no adapted module had moved
- [ ] Stale libc-bound packs are reported
- [ ] `make -C tools/gomad3 validate` and the `deterministicio` tests pass
## Done summary
Delivered `gomadtool adapter-regenerate` for all 15 adapter families using the existing exact-occurrence rewrite engine. Dry run downloads into a private cache and prints readable upstream changes, proposed anchors, and their approval digest without checkout writes. Approved apply stages constants, descriptor, pinned fixtures and every generated output, including the root qualification manifest, and verifies them before publication.

Publication uses an exclusive lock, exact old-byte checks and re-enumeration of the three copied input trees with the same traversal exclusions as staging. A durable old/new transaction marker recovers interrupted source and generated-output publication. Round-one review found newly added input files escaped drift validation; the regression reproduced that failure, and additions in all three trees now reject before marker/output writes. Cache-only additions still allow valid publication. Anchor zero/two matches, missing files, wrong approval, generation failure, interruptions, drift, competing applies and stale libc pack reporting have retained test evidence in `handover.json` and `fix-round-1-evidence.json`.

No adapted root dependency had moved: nine selected versions/sums match, six adapters are absent (`root-adapter-versions.json`). The authorized controlled Sprig v3.3.0 to v3.2.3 fixture exercised the real command, complete staged apply and patched-runtime qualification with two exact replays (`sprig-fixture-evidence.json`, `fixture-qualification-evidence.json`). Production adapter pins were not changed.

Focused command, deterministicio and compatibility-pack packages, validate, scoped vet, formatting and diff checks passed. Root lint exit2 is retained: its checker cannot load nested-module packages from the root. Nested lint reports pre-existing findings outside the changed adapter files and zero findings in those files; global lint is not claimed green. Independent same-family codex:gpt-6-sol:high re-review returned SHIP with R3 met and no findings; current nine source hashes and patch digest match the receipt scope. R4/R5/R6 and both-platform full qualification remain later tasks.

No files staged or committed: the user owns commits.
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (sync active=false).

Blocked:
The two verifier scratch-cleanup errors are checked at their existing lifetime boundaries. Nil cleanup preserves the primary error and unwrap shape, sole cleanup failures return directly, and simultaneous failures retain primary-first causes. Public Sentry and libc controls reproduce four genuinely discarded EACCES failures on the unchanged source and pass after correction. Successful and primary-only behavior and both pinned prepared source sets remain intact.

Final portable verification has fourteen leaf controls, four real permission-failure cases and zero skips. Root and the fresh independent reviewer each reran the eight public verifier controls and observed all four faults. Unfiltered deterministicio lint improves from four findings to two, with adapter_registry:390 and profile:211 unchanged. Scoped vet, errortype, check-only generator validation, four architecture checks and changed-line fast lint pass. Full lint still has 308 configured residual findings and remains red. Initial setup, overly broad unavailable-driver selection and deprecated resolver observations are retained rather than counted as successful gates.

Evidence is retained under .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/verifier-cleanup-progress/. Only the two verifiers and their additive public test change. Production pins, native guards, comments, preparation/selection order, generated/runtime/toolchain inputs and transaction publication remain unchanged. Source progress supplies no formal SHIP or task completion.

Task1 remains an open dependency. Original task2 regeneration acceptance, current native-Darwin/full/default/functional/affected-consumer and formal requirements remain open wherever unproved. This Linux ARM host and absent patched driver cannot supply native acceptance. Do not repeat unchanged unavailable-driver checks. Linux qualification remains deferred and unverified under fn128. The distinct publication-aware scratch cleanup correction remains outside this verifier checkpoint and needs its own bounded source admission before implementation.

### Historical implementation summary retained from the pre-correction task

Delivered `gomadtool adapter-regenerate` for all 15 adapter families using the existing exact-occurrence rewrite engine. Dry run downloads into a private cache and prints readable upstream changes, proposed anchors, and their approval digest without checkout writes. Approved apply stages constants, descriptor, pinned fixtures and every generated output, including the root qualification manifest, and verifies them before publication.

Publication uses an exclusive lock, exact old-byte checks and re-enumeration of the three copied input trees with the same traversal exclusions as staging. A durable old/new transaction marker recovers interrupted source and generated-output publication. Round-one review found newly added input files escaped drift validation; the regression reproduced that failure, and additions in all three trees now reject before marker/output writes. Cache-only additions still allow valid publication. Anchor zero/two matches, missing files, wrong approval, generation failure, interruptions, drift, competing applies and stale libc pack reporting have retained test evidence in `handover.json` and `fix-round-1-evidence.json`.

No adapted root dependency had moved: nine selected versions/sums match, six adapters are absent (`root-adapter-versions.json`). The authorized controlled Sprig v3.3.0 to v3.2.3 fixture exercised the real command, complete staged apply and patched-runtime qualification with two exact replays (`sprig-fixture-evidence.json`, `fixture-qualification-evidence.json`). Production adapter pins were not changed.

Focused command, deterministicio and compatibility-pack packages, validate, scoped vet, formatting and diff checks passed. Root lint exit2 is retained: its checker cannot load nested-module packages from the root. Nested lint reports pre-existing findings outside the changed adapter files and zero findings in those files; global lint is not claimed green. Independent same-family codex:gpt-6-sol:high re-review returned SHIP with R3 met and no findings; current nine source hashes and patch digest match the receipt scope. R4/R5/R6 and both-platform full qualification remain later tasks.

No files staged or committed: the user owns commits.
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (sync active=false).

Blocked:
Adapter regeneration now reports scratch cleanup errors before publication and adds ordered warnings after completeJournal succeeds. Nil cleanup preserves exact primary errors and results. Sole causes return directly, multiple causes retain primary first, and completed publication preserves Applied, Published, Staged, Residual and earlier scan warnings with nil operation error. Newly exposed stage-only cleanup failures retain the staged result. Both lock releases and existing recovery and journal-retirement branches remain unchanged.

The unchanged-source RED passes six controls and fails seven fault leaves. Final portable tests pass 35 leaves with zero skips. The conductor and fresh same-family reviewer each reran all thirteen public cleanup leaves and observed twelve actual scratch-root EACCES failures with no skips. Scoped vet, errortype, generator validation, four architecture checks and mandatory changed-line fast lint pass. Scoped unfiltered lint retains two lock-release findings. Configured full lint remains red with 308 inherited findings. Setup and intermediate failures are disclosed in the evidence.

The source checkpoint and exact retained regression logs are under .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/publication-cleanup-progress/. The preceding verifier cleanup checkpoint remains integrated at cefbe173514a4dde8b263462d3fc5504df340bf1, with its separate evidence under verifier-cleanup-progress/. Source progress changes no original acceptance or historical evidence and supplies no formal SHIP or task completion.

Task1 remains an open dependency. Original task2 regeneration acceptance, native Darwin, full-host/default/functional/affected-consumer and formal requirements remain open wherever unproved. The Linux ARM host and absent patched driver cannot supply native acceptance. Linux qualification remains deferred and unverified under fn128. Do not repeat unchanged unavailable-driver checks. Lock-release error handling remains separate from this bounded scratch correction.

Blocked:
Pre-publication checkout revalidation now rejects an existing input replaced by a same-byte symlink, using copyCheckout's regular-file policy after appeared precedence and before fileDigest. The user's symlink, external target bytes and other checkout content remain intact, with no journal or publication writer entry. Same-byte different-inode regular replacement still publishes; changed regular content, new symlinks and initial symlinks retain their refusal behavior.

Unchanged-production RED passes five controls and fails the same-byte symlink control with successful publication and Readlink EINVAL. Final portable adapterregen tests pass 40 leaves, 45 records and 25 top-level tests with zero skips. The conductor and fresh independent same-family reviewer each reran all six public controls. The reviewer also reproduced the single failing control using exact baseline production with the frozen final test. No findings were raised; the verdict is source-progress-acceptable only.

Scoped vet, errortype, check-only generation validation, four architecture tests and mandatory changed-line fast lint pass. Baseline and final scoped lint retain the identical two unchecked lock.Release findings. Configured lint has 302 residual findings before diff filtering and is not clean. Exact source/tool/config/log bindings, commands, timings, setup failures and review are retained under .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/checkout-revalidation-progress/.

Task1 remains an open dependency. Original R3 regeneration acceptance, native Darwin/full/default/functional/affected-consumer and formal requirements remain open wherever unproved. This Linux ARM host and absent patched driver do not supply native acceptance. Linux qualification remains deferred and unverified under fn128. The independent Recover behavior and existing revalidation/publication race are unchanged. Original acceptance and historical summary/evidence are preserved. This source checkpoint supplies no formal SHIP or task completion.
## Evidence
- Commits:
- Tests: make -C tools/gomad3 validate, go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./deterministicio ./internal/compatibilitypack/..., go -C tools/gomad3 vet -tags test_dep ./cmd/gomadtool ./deterministicio, go -C tools/gomad3 test -tags test_dep -count=1 -run ^TestAdapterPublication ./cmd/gomadtool, Sprig controlled fixture: adapter-regenerate dryrun/apply, make validate, Gomad qualification seed7 repeat2 choices and two exact success replays, codex impl-review round2: SHIP, working-tree patch/source binding verified, git diff --check; gofmt changed adapter files: clean, root lint-code-fast: exit2 nested-module loading limitation; nested lint: pre-existing findings, zero changed adapter findings
- PRs:
