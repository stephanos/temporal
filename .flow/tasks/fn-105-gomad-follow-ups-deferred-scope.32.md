---
satisfies: [R27]
---
# fn-105-gomad-follow-ups-deferred-scope.32 D27: state, pin, and remedy host-clock reporting escapes

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Contract/inventory/remedies, collector/assembly policy and native Darwin test-toolchain proof. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Origin: D21 investigation (`docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md`). Its proposals were left for a subsequent decision and had no owning task.

State the host-time escapes in the README contract. Pin the darwin `gettimeofday` path and `cputicks` in `toolchain/clock_inventory_test.go`, each with a fixture. Obtain and record the patch-policy owner's decision on overwriting the `LastGC` and `PauseEnd` stamps with stored virtual time from `runtime/proc.go`; implement the overwrite only with that approval.

### Current source acceptance (2026-10-08)

**Touches:** [tools/gomad3/README.md, tools/gomad3/toolchain/clock_inventory_test.go, .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task-32/source-acceptance-20261008/**]

Reconcile current R27 contract, inventory/fixtures and the recorded declined stamp-overwrite decision without reopening policy or native qualification. Retain current source checks, standards attribution, exact unchanged input bindings and fresh source review in the admitted evidence directory. The completed fn-112.5 source acceptance supplies exact retained both-source-set materialized inventory/preservation proof only where the current runtime, inventory, generator and archive inputs match; no historical whole-module, full/native or global lint pass follows from reuse. Review the complete current clock inventory and host-clock contract against bfb2bdb8ef136d3eb38cbd539735661d5d7c9af5 and the retained policy decision. All original requirements and historical evidence remain unchanged. Root owns Flow lifecycle, review, staging and local commits.

Current source handover and precise proof are in ../artifacts/fn-105-gomad-follow-ups-deferred-scope/task-32/source-acceptance-20261008/handover.md, evidence.json, source-binding.json and standards-attribution.json. Mandatory source review includes the complete current clock inventory, its platform/assembly/directive and activation-order checks, the six-count escape fixture, current README Contract host-clock passage, and durable declined-policy receipts. Compare original implementation range 70bb38e5ddec2d271c12f0e8c2489855f08eb0ef..bfb2bdb8ef136d3eb38cbd539735661d5d7c9af5 plus full current bodies; the 2819-byte host-clock passage and entire clock inventory remain exact original bytes. No artifact-only or empty HEAD..HEAD review substitutes for this source check. Changed-source configured lint and whole-clock-file attribution pass; unfiltered toolchain lint remains red on 16 unrelated findings. All native qualification remains transferred and unverified.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- The README contract names each host-time escape and what a target that reads it can observe.
- The clock inventory classifies the darwin `gettimeofday` path and `cputicks`; a fixture covers each.
- The stamp overwrite has a recorded patch-policy decision. If approved, it is implemented and the `LastGC` fixture qualifies; if declined, the limitation is stated in the contract.
- No prohibited collector or assembly file is edited without that approval.
- `make -C tools/gomad3 test-toolchain` passes on darwin/arm64; linux/amd64 status is recorded.


## Done summary
# fn-105.32 retained source acceptance

R27 source acceptance is complete without product changes. Three fresh-context Codex reviews returned actual SHIP, with zero findings and no unaddressed R-IDs, for source checkpoint 9081652cfd384d965391bede67a5b7d69f8056f3. Review included the complete clock inventory, original implementation range, current host-clock Contract and declined stamp-policy evidence.

The complete inventory and 2,819-byte Contract passage remain exact original implementation bytes. Current classification, six-count escape fixture, exposure/reader consequences and declined LastGC/PauseEnd overwrite are preserved. Two original policy receipts now have durable, lossless repository containers with their 3,660 original bytes verified; their historical skipped-review status supplies no new review credit.

The independent conductor rerun passed six selected portable tests with zero failures or skips. Generated validation and source/policy verification passed. Exact reuse verifies 87 runtime/inventory/generator/archive inputs and 1,223 referenced closure paths, while excluding five changed documents from historical reuse. Both supported materialized source sets retain 48 clock rows and activation guard checks. No native patched-runtime pass follows from this static source evidence.

Configured changed-source lint including errortype vet passes; the whole clock-inventory file has no lint findings and is gofmt clean. The unfiltered toolchain lint remains red on 16 unrelated findings, with no package/global waiver. The conductor verifier initially assumed a nonexistent bundle manifest; that failed artifact check is preserved, and the corrected actual-receipt-schema verification passes with no product changes.

Native Darwin clock/runtime/full qualification remains unverified under fn-149. Linux native qualification and CI remain deferred under fn-128. No collector or assembly edits, overwrite-policy revival, host spoofing, PR, push or CI action occurred.

stage: impl-review - ran (three actual SHIP draws; selected gpt-6.1-sol high, same GPT family; actual execution model metadata not independently verified)
stage: plan-sync - skipped(config: disabled)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: 9081652cfd384d965391bede67a5b7d69f8056f3
- Tests: /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go -C tools/gomad3 test -tags test_dep -count=1 -json -run ^(TestHostClockInventoryPinsPlatformSpecificEscapes|TestValidateAcceptsCurrentCheckedInputs|TestValidateRejectsPatchOutsideDescriptorAllowlist|TestValidateClassifiesProhibitedRuntimeAreas|TestValidateRejectsUnlistedAndBinaryOverlayEntries|TestBuildRejectsUnsupportedHostBeforePreparingInputs)$ ./toolchain, make -C tools/gomad3 validate, node .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task-32/source-acceptance-20261008/conductor-verify.mjs, timeout 600 make lint-code GOLANGCI_LINT_BASE_REV=70bb38e5ddec2d271c12f0e8c2489855f08eb0ef GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype LINT_CODE_DIR=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad3 LINT_CODE_TARGETS=./toolchain ALL_TEST_TAGS=test_dep, gofmt clock_inventory_test.go: no output; raw argv in task-owned-format.json, flowctl codex impl-review-fanout + finalize rid56ffa00e4de645ac99b2c81c2de8f995: three actual SHIP draws; R27 source met; native unverified
- PRs:
## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Contract/inventory/remedies, collector/assembly policy and native Darwin test-toolchain proof. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
