---
satisfies: [R3, R5]
---
# fn-112-gomad-determinism-assurance-and-test.5 Inventory seeded-stream draw sites and check host-timed paths at runtime

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Draw inventory/collector-contract reconciliation, current-source Darwin runtime/core/smoke and full/review gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stream isolation (R5): a checked-in, classified inventory of every seeded draw site, rerouting of any host-timed site still on the seeded stream, and a diagnostic-mode runtime check.

**Size:** M
**Files:** new `tools/gomad3/toolchain/draw_inventory_test.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch`, `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, a new toolchain test and fixture directory, generated protocol consumers, seeded conformance helpers, and the nested/root Makefile seeded launchers
**Touches:** [tools/gomad3/toolchain/draw_inventory_test.go, tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/runner/internal/execution/draw_check_toolchain_test.go, tools/gomad3/internal/gomadtool/conformance/testdata/draw_check/**, tools/gomad3/internal/gomadtool/conformance/runtime*.go, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go, tools/gomad3/Makefile, Makefile, tools/gomad3/runner/testdata/diagnostic-identity-choices.json, .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/host-draw-field-compact-20261005/**]

### Approach
- First step (R3): re-anchor assessment findings Q2 and Q3 against the current patch.
- Mirror the clock inventory test: enumerate references to the runtime rand helpers in the patched source, require each to carry a reviewed classification (target-ordered or host-timed), and fail on an unclassified reference.
- Known host-timed sites already on the M-local stream: lock-profile sample, anti-starvation wake, steal order, symtab cache. Known seeded sites: runnext flip, run-queue shuffle and pick, select, timer rand.
- Reroute any host-timed site still on the seeded stream. A site inside a collector file is not edited; record it and raise spec Open Question 3.
- Runtime check, diagnostics only: fail the process when a path classified host-timed draws from the seeded stream. Trigger it with the task 3 fault switch in a negative fixture run through the toolchain-test launcher in `runner/internal/execution`.
- Regenerate the patch with the governed `patch-regenerate` command; do not hand-edit hunks.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/toolchain/clock_inventory_test.go:86-212` — inventory pattern to mirror
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:89-99`, `:274-275`, `:745-746` — host-timed reroutes
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:556-594`, `:693-694`, `:780-806` — seeded sites
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:614-644` — rand helpers

**Optional** (reference as needed):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:369-477` — overlay rand helpers
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:121-127` — the one-shot syscall wait in `suspendG` (Q3, owned by fn-128.2 (transferred D12))

### Key context
- Rerouting a draw shifts every seed's schedule. Seed-specific expectations (seeds 11 and 17 in the manifests, D16 seed lists) must be requalified, so batch this with the other runtime edits named in spec Open Questions 2.
- fn-105 D26 moves forward-clock draws; if it has landed, the inventory reflects it.

### Re-anchor (2026-10-02)
Q2 is changed: purpose streams already exist, but runqputbatch still routes target/global admission and host netpoll/no-P admission through the same seeded shuffle helper. Preserve target-purpose diversification and route only host-timed batches to the M-local stream. Q3's one-shot wait remains confirmed by source and belongs to fn-128.2 (transferred D12); this task does not change suspendG.

Go 1.27.1 enables Green Tea by default. Runner preparation explicitly selects the qualified classic collector, while the current raw seeded conformance builders and Make launchers omit that compile profile. Inventory classification must reflect the qualified profile; seeded Green Tea activation is refused outside it without editing collector files. Seeded harness/helper builds and the supported seeded launchers must establish the classic profile themselves. Keep a raw Green Tea rejection and unseeded control; a one-off environment override is not qualification evidence. These launcher and generated-protocol updates are directly implied by the runtime acceptance boundary.

Extend the existing trusted diagnostic fault switch with an explicit host:<ordinal> mode for the host-path negative control. The numeric ordinal mode continues to perturb a target draw and must retain task 3's successful diagnostic localization. Per-M host-path scope must be checked before seeded counters or state change.

### Source-progress revival (2026-10-05). Compact private host-draw field

Shorten the private per-M diagnostic field from gomadHostDrawScope to gomadHostDraw at its producer. Preserve its bool type and exact struct position, every read/write/check, previousHostDrawScope, fault modes, counters, existing comments and runtime behavior. The current long field name makes gofmt realign 36 unrelated m fields in the upstream patch. This bounded correction contributes to fn-110.2's still-open R8 size gap without moving protected scheduler machinery or removing diagnostics.

Use the verified cached Go1.27.1 archive, materialize the current canonical patch into unique gitignored scratch, edit only runtime2.go's field and proc.go's three references with apply_patch, then regenerate through patch-regenerate. Update the three overlay references with apply_patch. Neither generated patch hunks nor protocol identities may be hand-edited. Run make generate validate only after recording the generator input/output closure; root admits any additional derived output paths before publication. Keep both exact allowlists unchanged.

Baseline and rerun both platform draw inventories, canonical/pinned U1/U3 regeneration and zero-fuzz materialization controls, and the existing diagnostic numeric perturbation/host-timed refusal tests where executable. Retain frozen source, literal command/exit/test receipts, actual patch/overlay measurements and alpha-renaming/layout/comment preservation. A test-first measurement of real canonical output must expose the avoidable alignment overhead before production edits; use retained task-local evidence rather than a brittle permanent private-name assertion. Preserve diagnostics and actual pinned-rejection tests.

The pre-implementation scout estimates U3 38,362 to 34,148 bytes and U1 29,015 to 24,894 bytes. These are estimates until governed regeneration proves them. The original baseline U3 32,652 bytes and R8 requirement remain unchanged, so a predicted 1,496-byte residual still prevents size acceptance. No native Darwin runtime/core/smoke/full/affected or formal qualification is claimed from developmental linux/arm64 checks. Linux execution stays transferred and nonblocking under fn-128.

This source checkpoint resumes the reviewed integrated predecessor candidate while keeping dependency .3 and every original Acceptance item open where unproved. Root owns Flow lifecycle, review, staging and commit. One worker owns source, canonical generators and Go/cache execution; parallel agents are read-only. Preserve unrelated user files, historic reports and workload dispositions. No push, stash, worktree, history rewrite, network download, collector edit, feature deletion or host impersonation is authorized.

### Source-derived diagnostic fixture refresh

The retained choices fixture was already stale at BASE 1147416b2e. Its supplied fake BuildKey and unchanged Darwin metadata yield an implementation identity different from the retained golden even before this rename. Root admits only tools/gomad3/runner/testdata/diagnostic-identity-choices.json for a source-derived refresh after baseline and RED checks terminate and generation freezes. The task-local independent Node/schema calculator must first reproduce the retained canonical bytes, trace/tape digests, failure signature, record hash and portable-plan hash, and verify the current fake preparer/executor contracts. Its final derivation changes exactly seven JSON pointers: three choice implementation identities, tape digest, failure signature, record hash and portable-plan digest. Publish its emitted canonical bytes through apply_patch. Preserve every other field, supplied BuildKey, Darwin metadata, trace bytes, schemas, counts and the plain fixture. Keep diagnostic_identity_test.go, its native host guard and complete-byte assertion unchanged. This source calculation supplies no native test or replay pass; all original Darwin/full gates remain required.

### Verified source checkpoint (2026-10-05)

The seven-site private-field compaction and its four generated protocol mirrors are integrated with the independently derived choices golden. Actual canonical U1 is 24,894 bytes / 706 lines; U3 is 34,148 bytes / 1,040 lines. The same task-local canonical-output check went RED for 36 unchanged-field alignment edits and GREEN for zero. Both supported source-set inventories pass at 273 draw rows and 86 seeded rows. Original pinned regeneration, checksum/rejection and U1/U3 zero-fuzz equivalence checks pass without skips; fresh final materialization and 21-file alpha-renaming/pinned-gofmt preservation pass. The bool field stays between spinning and blocked, with previousHostDrawScope and every comment/check/fault/counter unchanged. Allowlists remain 20 patched / 79 overlay paths.

Root independently verified the 5,076-file source closure, exact seven product paths, emitted 13,271 canonical golden bytes and all six retained self-checks/seven identity pointers, original owner contracts, tools/archive/user-file hashes and receipt stream digests. Fresh root generator/patch/pack validation and TestPackageArchitecture pass. Scoped vet passes. Documented changed-line lint covers 55 host packages and reports zero new findings with retained native pinned executables and fixes disabled. The initial Darwin-only executable failure is preserved. Its actual diff filter removes 317 existing findings, so full lint remains unresolved.

The final broad developmental selection remains red at 972 pass / 106 fail / 13 skip events. Root and the fresh reviewer independently paired all 1,091 verdicts and all raw failed-test error arrays with the pre-edit baseline; none changed. Three original diagnostic controls still fail before their runtime assertions because the patched driver is absent; the guarded native identity test skips. The full developmental baseline's 600.082-second unsupported-host timeout remains inconclusive and was not retried unchanged. Separate trace/simulation residuals remain unresolved.

The fresh source reviewer approved SOURCE_PROGRESS_COMMIT_APPROVED with no introduced findings. This supplies no formal SHIP or native acceptance. AGENTS requested Sol/high for writer/reviewer, a same-family pairing, and Astra/high for research; actual runtime model identities were not independently evidenced. Original Acceptance and dependency .3 are unchanged. Current-source Darwin runtime/diagnostic/core/smoke/full/affected and formal gates remain open. Linux execution remains nonblocking under fn-128; fn-110 R8 still has a 1,496-byte gap to its unchanged original U3 comparator.

See `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/host-draw-field-compact-20261005/` for handover, checkpoint-report, root-verification, raw receipts and source-review. Root commits this verified source progress under MILESTONES instruction 5 and keeps this task blocked for the unproved acceptance. No push or qualification waiver occurs.

stage: source-progress-review - ran
stage: impl-review - skipped(policy: full/native acceptance remains unproved and the developmental full tree is red)
stage: plan-sync - skipped(config: planSync.enabled is false; no task completed)
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Findings Q2 and Q3 re-anchored and marked confirmed, changed, or refuted
- [ ] The inventory test lists every seeded-stream reference with a classification and fails on an unclassified one; a deliberate unclassified reference demonstrates the failure
- [ ] Every host-timed site draws from the M-local stream, or is recorded as blocked by the collector prohibition
- [ ] The diagnostic-mode check stops the process in the negative fixture
- [ ] If any site was rerouted: core and smoke qualification sets pass on the new toolchain identity
- [ ] The inventory's location and the reroutes made are in the done summary for task 10 to document
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime` pass on darwin/arm64; linux status recorded
## Done summary
Delivered R3/R5 on Darwin runtime key `2ecdbd330bb5928f360739fdd80cb3eb0f0764e513a5cc208cd83714fff0a457`. Q2 changed: purpose streams already existed, but target admission and host netpoll/no-P batch admission shared a seeded shuffle. Explicit batch origins retain target seeded diversification and route host batches to M-local entropy; the Linux CPU-profiler timer sample is also rerouted. Q3 confirmed: suspendG still performs its one-shot syscall wait before its retry loop; D12 owns that correction.

The checked-in inventory is `tools/gomad3/toolchain/draw_inventory_test.go`: 269 platform references, 135 classified rows, both qualified platforms, declarations/calls/function values and package-scoped linkname aliases. Unclassified references, count drift, and disappeared references fail closed. Scratch controls demonstrated an unclassified direct draw and alias uses in another file; the cross-file control failed before the fix and passes afterwards. The classic collector's `runtime/mgcpacer.go:enlistWorker` remains an explicitly recorded host-timed seeded draw blocked by the collector-file prohibition. No collector file was edited. Green Tea's worker draw is outside the qualified profile: seeded activation is rejected before user code, with a passing unseeded control.

Diagnostics use a per-M host scope and check each seeded entry before counters or state change. Existing numeric perturbation still localizes ordinal 5; `host:5` fails with the host-timed seeded-draw diagnostic. Seeded raw builders and supported Make launchers own `GOEXPERIMENT=nogreenteagc`. The full gate exposed another simulation integration builder, fixed at its compile boundary without changing any assertion or simulation selection. Additional existing test owners used instead of the initially proposed new fixture file: diagnostics_toolchain_test.go, simulation_root_integration_test.go, and target/internal/livecap/toolchain_test.go; generated protocol/descriptor consumers follow the changed runtime identity. Parent diagnostics choices golden changed in seven identity-derived fields only; the plain golden and complete byte comparison remain.

Verification: required validate/test-toolchain/test-runtime passed. The final census fix passed its focused and full toolchain suites and scoped vet; the parent retained a final patched-driver toolchain run. All components of the nested full test gate passed on stable inputs, with already green components reused after the integration-only helper correction; `fn-114/qualification/darwin-2ecdbd33/combined-gates.md` lists each component and log. Host gate: 45 packages. Core: 7/7 qualified and replayed. Smoke: 4/4 qualified and replayed. Scoped vet, integration-tagged execution vet, gofmt, diff check, and 16/16 source-hash verification passed. Linux/amd64 classic runtime cross-build passed, but native Linux qualification remains unavailable. The existing root lint cannot analyse nested-module paths; its prior environment-failure log is retained, rather than reporting lint green.

Independent working-tree implementation review: SHIP, codex gpt-6-sol at high, same-family reviewer with fresh context, no introduced findings. The reviewer inspected the uncommitted patch, 16 bound sources, generated/test changes, and gate evidence. `working-tree-review.json` is the acceptance receipt. The earlier task-scoped empty HEAD..HEAD review explicitly excluded uncommitted work; its receipt is preserved as `review-empty-committed-scope.json` and is not acceptance evidence. The recorded collector blocker and a possible census extraction were nonblocking; no speculative refactor was added.

Task-10 documentation facts: inventory path/counts; host batch and Linux profiler reroutes; diagnostic host mode; classic-only seeded activation and raw disabled control; remaining classic collector blocker; Q3/D12 ownership. Shared candidate qualification stays with fn-114.14. Its first representative run stopped at the unchanged 2 GiB space bound after 1/28; caches were cleared and the unchanged manifest retry is running. Representative and native Linux acceptance remain unproven. No disposition was weakened; no staging, commit, or push was performed.

Evidence: task-5/handover.md, source-hashes.txt, working-tree-review.json, and the shared fn-114 qualification/darwin-2ecdbd33 directory. HEAD is d635e23f00d926a43b942f25a9d05bd0ccb72025 with uncommitted sources.

stage: wave-dispatch - ran (model: gpt-6-sol at high)
stage: impl-review - ran (model: gpt-6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)

Blocked:
Verified private host-draw compaction, generated identities and independent canonical golden refresh are ready for a source progress commit, with a fresh source-only approval. Original Acceptance and dependency .3 remain unchanged. Current-source native Darwin runtime/diagnostic/core/smoke/full/affected qualification and formal review remain unproved on this Linux/arm64 developmental host. The full developmental baseline timed out after 600.082 seconds; the final broad selection retains 106 baseline-paired failed events and 13 skips, and the three diagnostic controls cannot reach runtime assertions without the patched driver. Changed-line lint passes across 55 packages; 317 original full-lint findings remain unresolved. Retain raw receipts and draw-inventory/collector exclusions. Native Linux execution stays transferred to fn-128 and does not block this source task. Evidence is in task-5/host-draw-field-compact-20261005, including root-verification.md and source-review.md. No source checkpoint result constitutes formal SHIP or native task completion.
## Evidence
- Commits:
- Tests: make -C tools/gomad3 validate test-toolchain test-runtime (Darwin, exit 0; required-gates.log), Final cross-file alias census: focused and full toolchain tests, stock Go1.27.1, -tags test_dep -count=1, exit0; scoped toolchain vet exit0, make -C tools/gomad3 -o toolchain test-toolchain (candidate patched driver, -tags test_dep -count=1; census-final.log), Focused numeric diagnostic perturb, host:5 rejection, GreenTea seeded rejection/unseeded control, collector-profile unit control: exit0, make -C tools/gomad3 -o test-toolchain -o test-runtime test: harness/interception/45-package host/nine-overlay packages passed; simulation integration exposed a raw builder collector-profile gap, make -C tools/gomad3 -o test-harness -o test-toolchain -o intercept-test -o test-host -o overlay-test -o test-runtime test: exit0, 252.304s; simulation/World race/builder/live capability/upstream passed; unchanged green components reused explicitly, make -C tools/gomad3 core-qualification: exit0,192.040s,7/7 qualified and replayed on2ecdbd33, make gomad3-smoke-qualification: exit0,402.453s,4/4 qualified and replayed on2ecdbd33, Patched Go vet -tags test_dep ./toolchain ./internal/gomadtool/conformance ./runner/internal/execution ./runner ./target/internal/livecap: exit0, Patched Go vet -tags test_dep,integration ./runner/internal/execution: exit0, GOOS=linux GOARCH=amd64 GOEXPERIMENT=nogreenteagc CGO_ENABLED=0 patched Go build runtime: exit0; cross-compilation only, Pinned gofmt: no output; git diff --check: exit0; 16/16 source SHA256 bindings match, flowctl codex impl-review standalone working tree --base HEAD --spec codex:gpt-6-sol:high: SHIP; working-tree-review.json, NOT VERIFIED: native linux/amd64; representative qualification is still running under fn114.14, ROOT LINT UNAVAILABLE: root module cannot load nested module paths; reuse fn114/task-13/integrated-root-lint.log environment failure
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Draw inventory/collector-contract reconciliation, current-source Darwin runtime/core/smoke and full/review gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
