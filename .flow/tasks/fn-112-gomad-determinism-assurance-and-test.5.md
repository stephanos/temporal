---
satisfies: [R3, R5]
---
# fn-112-gomad-determinism-assurance-and-test.5 Inventory seeded-stream draw sites and check host-timed paths at runtime

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Draw inventory/collector-contract reconciliation, current-source Darwin runtime/core/smoke and full/review gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stream isolation (R5): a checked-in, classified inventory of every seeded draw site, rerouting of any host-timed site still on the seeded stream, and a diagnostic-mode runtime check.

**Size:** M
**Files:** new `tools/gomad3/toolchain/draw_inventory_test.go`, `tools/gomad3/toolchain/runtime/go1.27.1.patch`, `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, a new toolchain test and fixture directory, generated protocol consumers, seeded conformance helpers, and the nested/root Makefile seeded launchers
**Touches:** [tools/gomad3/toolchain/draw_inventory_test.go, tools/gomad3/toolchain/runtime/go1.27.1.patch, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/runner/internal/execution/draw_check_toolchain_test.go, tools/gomad3/internal/gomadtool/conformance/testdata/draw_check/**, tools/gomad3/internal/gomadtool/conformance/runtime*.go, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go, tools/gomad3/Makefile, Makefile, tools/gomad3/runner/testdata/diagnostic-identity-choices.json, .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/host-draw-field-compact-20261005/**, .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008/**]

### Current source acceptance (2026-10-08)

Retain current-source acceptance and review evidence in the admitted source-acceptance-20261008 directory. Bind historical positive inventories and preservation evidence only through the exact unchanged runtime, inventory, generator and manifest input closure; changed shared documents are not covered by older whole-module snapshots. Run fresh negative inventory controls, generated-source validation and applicable portable standards. Review the complete current host-draw guards, seeded helper bodies, inventory classifications and collector/launcher boundary, not only an alpha-renaming diff. The classic collector prohibition remains an explicit source acceptance alternative; no collector edit or broader isolation claim is admitted. Transferred runtime/native execution remains deferred, and partial portable passes or skipped native tests supply no native qualification. Root retains lifecycle, staging, local commits and review ownership. All original acceptance and historical evidence remain unchanged.

Current source handover and review context are in ../artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008/handover.md, evidence.json, source-binding.json and standards-attribution.json. Mandatory review includes the complete current host-scope/seeded guard bodies, fault fixtures, inventory classifications, batch producers/consumers and collector/launcher boundary. Use committed compact base 1b0bc277589d141aca8b534b03135ab3e57fc050 plus original source base d5330bb779c55f1b9af57c5845ec195275df0276 and implementation c2fd6d2f0f/review-fix 5df49456e6 for the full historical-to-current body context. Artifact-only or tiny alpha diff review is insufficient. The broader lint command remains red on two unrelated descriptor-test closes; the unchanged overlay separator-format defect is handed back to fn-109.13. Neither observation is suppressed or upgraded to a full standards pass.

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


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Findings Q2 and Q3 re-anchored and marked confirmed, changed, or refuted
- [ ] The inventory test lists every seeded-stream reference with a classification and fails on an unclassified one; a deliberate unclassified reference demonstrates the failure
- [ ] Every host-timed site draws from the M-local stream, or is recorded as blocked by the collector prohibition
- [ ] The diagnostic-mode check stops the process in the negative fixture
- [ ] If any site was rerouted: core and smoke qualification sets pass on the new toolchain identity
- [ ] The inventory's location and the reroutes made are in the done summary for task 10 to document
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime` pass on darwin/arm64; linux status recorded
## Done summary
# fn-112.5 retained source acceptance

R3/R5 source acceptance is complete on the unchanged runtime candidate. Three fresh-context Codex reviews returned actual SHIP, with zero findings and no unaddressed R-IDs, on compact base 1b0bc277589d141aca8b534b03135ab3e57fc050 through source checkpoint 806d2d462aa22d4d059e2f6967ac50fbc98547b1. Full guard, seeded helper, batch, inventory, collector and launcher bodies were reviewed.

The independent conductor run passed 21 selected portable tests with zero failures or skips, and generated validation passed. Exact reuse verifies 87 source bindings, 1,223 source/generator paths with five changed documents excluded, and 72 retained raw files containing 521,317 original bytes. Both supported materialized source-set inventories retain 273 draw rows and 86 seeded rows. No product code, generated bytes, native guard, collector policy or user-owned file changed.

Q2 remains changed, target/host batch routing is distinct, and Q3's confirmed one-shot wait remains fn-128.2/D12 work. The classic collector enlistWorker blocker is the original explicit acceptance alternative; seeded Green Tea remains refused. Task 10's inventory/reroute/fault-mode documentation handoff is retained in handover.md.

Broader lint remains red on two unrelated Linux descriptor-test closes, and the unchanged overlay separator formatting is explicitly handed back to fn-109.13. No aggregate lint, full-format or native pass is claimed. The contracts reviewer noted one stale evidence-helper manifest hash; it was corrected and all 41 listed entries verified, without altering source or recorded commands.

Darwin native diagnostic/runtime/core/smoke/full qualification remains unverified under fn-149. Linux native qualification and CI remain deferred under fn-128. Partial portable coverage and source inventory materialization do not qualify either native platform.

stage: impl-review - ran (three actual SHIP draws; gpt-6.1-sol high, same GPT family, actual execution metadata not independently verified)
stage: plan-sync - skipped(config: disabled)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: 806d2d462aa22d4d059e2f6967ac50fbc98547b1
- Tests: conductor: /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -count=1 -json -run ^(TestDrawInventoryRejectsUnclassifiedReference|TestSeededDrawInventoryRejectsUnclassifiedReference|TestRuntimeCampaignCollectorProfile|TestPackageArchitecture|TestChoiceWire.*|Test(ProjectFindings|LookupBoundarySymbol|IsGuardSymbol|Extract(MachO|ELF)Record|Decode).*)$ ./toolchain ./internal/gomadtool/conformance ./choice/internal/wire ./target/internal/livecap ., conductor: make -C tools/gomad3 validate, node .flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008/conductor-verify.mjs, configured errortype vet over toolchain/conformance/wire/livecap/execution: exit 0; exact argv in configured-vet.json, 41 bundle-manifest entries verified after one metadata hash correction, flowctl codex impl-review-fanout + finalize rid05e1be3dc45d4654a50d7753c3807fce: three actual SHIP draws; no native qualification
- PRs:
## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Draw inventory/collector-contract reconciliation, current-source Darwin runtime/core/smoke and full/review gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
