---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.56 Check builder cleanup while preserving publication outcomes

## Description
Own exactly the 15 unchecked production cleanup results retained in toolchain/build.go: snapshotInputs six, publishStable two and temporaryFile seven. Tasks47/48 supply established cleanup conventions but do not admit this file. Root admits this serial correction at reviewed commit60ae2649db11f637b851817cd77c3fc814e33d1b, using task55/reconciled-evidence.json, reconciled-source-proof.json and independent-review.md. The actual original-base RED95 is the retained defect baseline; measure RED95 to RED80 with exactly 15 removed and zero added. Root owns lifecycle/review/commit and the fresh worker owns code/tests/evidence. Task21 consumes it. Do not require a predecessor's Done status where the source gate consumes this correction.

**Touches:** [tools/gomad3/toolchain/build.go, tools/gomad3/toolchain/build_cleanup_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-56/**]

### Explicit cleanup correction

Check the existing cleanup calls in place. Preserve deferred registration, argument/receiver evaluation, LIFO order, nil-success behavior and original primary error object/bytes when cleanup succeeds. Return a sole cleanup failure directly; join a primary and genuine cleanup errors primary-first in operation order only when cleanup fails. Admit only the newly surfaced cleanup-error behavior. Add no retry/rollback/helper/framework/callback/seam/library/policy/pin/native guard or API change. Preserve all old tests and unrelated comments. The competing-build test sleep is excluded and requires its own owner.

snapshotInputs keeps patch Close before patch Remove and before original primary formatting; its overlay failure keeps patch Remove before overlay RemoveAll and before formatting. Return the unchanged zero snapshot on error. temporaryFile keeps Close then Remove on Chmod/Write/Sync failures, and Remove after final Close failure. Close exactly once in each path; retain empty returned pathname and raw primary identity when cleanup succeeds.

publishStable may use a named error result with two inline defers at their existing registration points. The stamp cleanup runs before launcher cleanup. Track each successful Rename independently; ignore os.ErrNotExist only for the corresponding already-renamed temporary, never for missing unpublished paths or other errors. Keep create/sync/close launcher then stamp, stamp Rename and flag before root sync/after-stamp-publish hook, launcher Rename and flag before bin sync/after-launcher-publish hook. Preserve completed publications and InjectedFailure discoverability through the existing buildFailure boundary. Leave buildWith's already-checked snapshot/lock/work cleanup and wrapping unchanged.

### Controls and source gates

Read AGENTS.md/README/MILESTONES, current task/spec and nearby source.go/patch_regenerate.go/task47/48 evidence before implementation. Retain the actual source-identical unfiltered analyzer RED as defect evidence. Baseline/final additive controls cover literal patch/overlay bytes/modes and independent snapshots; missing patch/overlay and unsupported entry failures with exact primary diagnostics/no new debris; temporary bytes/0644/0755 modes/closed-file usability and invalid destination with empty pathname; exact launcher/stamp bytes/modes/no temporaries/repeated publication; existing FailurePhase after stamp and launcher publication and genuine destination-directory rename failures with typed phase, precise surviving files and old launcher behavior. Keep all existing controls/assertions.

Direct helpers and existing fake-dependency builder tests are ordinary source coverage on Linux/arm64. Real Build rejects this unsupported host before reaching these helpers. No deterministic helper hook induces genuine private Close/Remove failures after creation; do not manufacture descriptor theft, races, callbacks or a cleanup framework. Retain source proof and honest genuine cleanup/multiple-failure runtime gaps. Native qualification stays under deferred fn149/fn128.

Use apply_patch, established libraries, pinned stock Go1.27.1, every Go test -tags test_dep -count=1 and the established local cache/module/file-proxy recipe with fresh private /tmp. Serialize gates and freeze source. Run focused baseline/final, full ordinary affected toolchain tests with honest failures, vet/standalone errortype, required architecture/public/purity and both supported static source sets, fresh check-only validation, format, actual unfiltered affected configured lint, actual FIX=false make lint-code-fast against the admitted base and original-base make --trace lint-code-gomad3 against951c5516e9e7b3066e7e069adda9565cfd68844c. Measure exactly15 removed/0introduced; retain residuals and integrated errortype reachability. Reuse valid exact-source receipts rather than rerunning unchanged red only to reconfirm. Bind commands/exits/elapsed/raw/source/tools and keep handover small.

Fresh independent source-progress review may license a separate progress commit. Formal acceptance stays open while required affected/integrated/ordinary source gates remain red; original preservation/first-baseline/R18/R19 requirements and native transfers are unchanged. No native revival, PR, push or CI authority.

## Acceptance
- [ ] Exactly 15 cleanup results are checked with preserved call order/evaluation/LIFO, nil-success and original primary identity/bytes. Genuine cleanup failures surface sole/direct or primary-first joined errors; only the corresponding renamed temporary permits ENOENT. Existing publication/phase semantics, old tests and unrelated source remain unchanged.
- [ ] Baseline/final ordinary helper and fake-dependency controls cover healthy bytes/modes, primary errors, injected publication boundaries and genuine rename failures without native spoofing. Every unexecuted Close/Remove/multiple-failure branch has precise source proof and disclosed runtime ownership.
- [ ] Required frozen-source focused/ordinary tests, vet/errortype, boundaries/both static sets, validation/format and actual unfiltered configured/fast/original-base lint evidence show exactly 15 removed and zero added. All required source-gate gaps and integrated errortype reachability stay explicit.
- [ ] Fresh independent review, root verification and separate commit feed task21. Red required gates keep formal acceptance open; original preservation/first-baseline/native requirements and publication authority remain unchanged.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
