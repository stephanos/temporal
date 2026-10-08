---
satisfies: [R13, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.33 Preserve artifact reflection helper coverage and clone isolation

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Repair the two exhaustive reflection-helper findings in artifact/opened_test.go while preserving the existing manifest-clone and opened-handle assertions. The reviewed task 31 and task 32 candidates are integrated at 1ee85b004cfaac41e178784701decbf0dd277968; their original acceptance remains open. This owner advances R13/R18/R19 test-support qualification, not a production clone redesign.

**Touches:** tools/gomad3/artifact/opened_test.go; .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-33/**

Follow the source-bound strategy in .flow/tmp/artifact-remaining-diagnostics-owner-plan.md (SHA256 911049ae81ae792740d70b67d7ad93e8cea586525249f4b161b5de09406efbe9). Add testing.T to populate and its recursion; retain existing branch operations, scalar initialization values, traversal order and assertions. Explicitly cover all 27 reflect.Kind values in both helpers. Scalar kinds are explicit no-ops where previously untouched; unsupported/invalid shapes fail through t.Fatalf, including typed nils. Add assertNoSharedMemory array-element recursion with useful paths. Keep the original equality precondition and fixture-specific traversal.

Add supported synthetic array/reference copy-isolation, nonnil-empty/nil containers and nonzero scalar controls, and direct characterization of the unchanged production unsupported-kind guard, including typed nils. Preserve production manifest_copy.go and publication.go byte-for-byte. The two residual invariant-panic and uppercase-World diagnostics are not waived. Passing positive tests do not prove the helper's Fatalf rejection or rejection of intentionally shared arrays; disclose source-review-only coverage rather than introduce a failure-capture framework.

Root owns Flow, spec/MILESTONES updates, review, staging and commits. The worker owns only this test file and task-unique proof files, excluding root-admission/review/checkpoint files. Preserve all earlier evidence and unrelated untracked files. No production, schema, public API, config, suppression, pin, dependency, runtime, worktree, bridge, download, push or history changes.

**Quick commands:** cached pinned Go 1.27.1 linux/arm64: full artifact package and focused clone/opened/copy-isolation/cleanup controls with -count=1 -tags test_dep; actual nested-root architecture/public-signature/external-consumer boundaries; pinned unfiltered configured artifact lint; errortype; source diff/gofmt checks; make validate after inspecting generator inputs. Retain command, actual exit, time, source bindings and meaningful RED evidence. Use offline GOWORK=off, GOTOOLCHAIN=local, GOPROXY=off with Go first on PATH. Native/full/formal failures already tied to unchanged environment are not retried.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- Both Kind switches explicitly account for all 27 pinned reflect.Kind values without a passing default or analyzer bypass. Existing population branch operations and initialized values remain unchanged apart from testing.T propagation. Existing no-sharing checks and record/opened-handle assertions remain intact; arrays recursively inspect every element.
- Supported synthetic array/reference controls exercise actual deepCopy, equality, no-sharing and clone mutation/original isolation. Literal nil/nonnil-empty containers and nonzero uintptr/float/complex controls retain values and container distinctions. Direct unchanged production-guard controls cover valid Interface/Func/Chan/UnsafePointer values including typed nils and exact invariant messages; supported controls distinguish indiscriminate rejection.
- Retain a real baseline and final full artifact package, focused tests, actual architecture/public/external boundaries, errortype, source/gofmt and generator validation on the available pinned stock toolchain. Run the actual unfiltered pinned artifact analyzer before and after. Prove the two helper findings resolved with no introduced diagnostics; retain actual residual findings without claiming clean whole lint. Disclose helper fatal/shared-array rejection proof limits.
- The only product diff is opened_test.go. All production sources, other old tests, public APIs, schemas, generator inputs, config, dependencies/pins, older artifacts and original acceptance criteria are preserved. Independent source review permits a source-progress checkpoint before the next writer. Root commits the reviewed implementation, proof and Flow/docs together.
- Original task12/predecessor and task21 acceptance, R13/R18/R19, matched first-baseline fixed identities, full/completion/formal/affected-consumer and native darwin/arm64 qualification remain required. Developmental stock linux/arm64 checks do not satisfy these. Complete this task only when its corresponding source-owned gates actually pass; otherwise record reviewed source progress and retain blocked acceptance. Linux native execution, pack/report/replay and qualification documentation belong to fn-128.1, fn-128.4 and fn-128.7; missing transferred Linux evidence does not block this task.


## Done summary
Reviewed source progress only; acceptance remains blocked. Both helpers cover
27 Kinds, preserve all existing branches/values/assertions, and recurse arrays.
Three characterization controls retain copy/scalar/nil-empty and eight production
panic cases. Serial independent artifact 49/49, focused 23/23, five boundaries,
errortype, source/static and generator checks pass. Actual unfiltered artifact
lint falls from four to two unchanged production findings, with none introduced.
Production and 1,044 protected inputs remain unchanged. Helper Fatalf and
intentionally shared-array rejection execution remain unproved.

Fresh source review: SOURCE_PROGRESS_COMMIT_ONLY, no introduced findings.
[Review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-33/independent-source-review.md),
[checks](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-33/independent-source-review.json),
[acceptance](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-33/acceptance-open.md).
Root independently reaudited preserved writer and serialized reviewer receipts.
The generated review-output file is separately hash-bound and excluded only
from command-receipt interpretation; archived audit bytes remain unchanged.
Reviewer/root preflight failures and superseded concurrent checks are retained.

Original R13/R18/R19, task12/predecessors/task21, matched first-baseline fixed
identities and complete/full/completion/formal/both-native/affected-consumer
qualification remain required and open. No native or formal SHIP is claimed.
Root's reviewed source-progress checkpoint is e521cbd2e39e8521bff013de19e569eff8a1392c.

Tier: session (jev-unavailable(no_key))
Requested writer/reviewer: gpt-6.1-sol at high, same family; actual models unknown.
stage: impl-review - skipped(policy: conductor-deferred; fresh source review passed, full/native qualification remains red)
stage: plan-sync - skipped(config: disabled; task remains blocked rather than done)

## Evidence
- Commits: e521cbd2e39e8521bff013de19e569eff8a1392c
- Tests: serial artifact49/49, focused23/23, boundaries5/5; errortype/static/generator exit0; lint exit1 two unchanged findings.
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
