---
satisfies: [R18, R19, R21]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.43 Enforce the documented five-import compatibility-pack admission boundary

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
Implements R21 through the existing shared admission predicate. Task11 retains R17; task21 consumes the correction. Root owns scope, Flow, reviews and commits. Start from reviewed integrated commit 43d264e24cae659ed57b81b9028e14fccb5a1fa3; no completion dependency on blocked predecessor acceptance.

**Size:** M
**Files:** tools/gomad3/internal/compatibilitypack/schema.go, policy_exhaustive_test.go, new schema_admission_test.go, external_test.go, authoring/request_test.go, authoring/generate_test.go; task43 artifacts.
**Touches:** [tools/gomad3/internal/compatibilitypack/schema.go, tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go, tools/gomad3/internal/compatibilitypack/schema_admission_test.go, tools/gomad3/internal/compatibilitypack/external_test.go, tools/gomad3/internal/compatibilitypack/authoring/request_test.go, tools/gomad3/internal/compatibilitypack/authoring/generate_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-43/**]

### Approach

Add independently enumerated five-import regressions first. Retain meaningful RED for the two previously accepted imports on unchanged production. Extend only the shared unadmittableCapabilities list and its adjacent rationale. Keep ValidatePack's two-pass structural-then-admission ordering and existing error construction. Do not change Evaluate, remediation, external-loader or authoring production.

Explicitly supersede only task42's plugin/cgo expected loader results. Keep its other assertions and literal decision/digest controls unchanged, including future/empty kinds, traversal, matching syscall/linkname grants, drift and copy isolation. Preserve historical receipts rather than replacing their BASE snapshots.

Cover ValidatePack, DecodePack and LoadPack with literal errors for all five imports. Test a prohibited grant in rule0 plus an invalid inventory in rule1, preserving pack structural error precedence; sorted multi-rule and two-capability cases pin admission ordering. Forge same-package ValidatedPack tokens to prove selection and identity verification revalidate even otherwise-unselected packs. Preserve ValidateRequest's existing fact-validation priority and cross-pack earlier digest/duplicate error priority; do not normalize these to the within-pack two-pass rule. No new public construction or bypass seam.

Exercise external directory and environment loaders, with a valid file ordered before the invalid pack. Assert exact wrappers and no partial result. Preserve duplicate-ID/path/entry errors. Authoring allowed facts reject; denied prohibited facts remain valid. Generate must reject before publication even with supplied approval, leaving a populated generated root byte-identical. A request with a denied prohibited fact plus an allowed syscall fact can generate; retain denial evidence and exclude its capability from emitted grants. Avoid a denied-only rule, which cannot form a valid emitted pack.

### Investigation targets

Required: schema.go:36,131,386; v2_selection.go:15,24,138,187,221; authoring/request.go:269; authoring/generate.go:236,314; policy_exhaustive_test.go:80; external_test.go; authoring/request_test.go and generate_test.go. Use validPackV2/validRequest and existing external/generation helpers. Read the task42 external-pack-policy-gap.md and generator input list in tools/gomad3/Makefile before coding.

### Verification

Use the existing pinned offline Go1.27.1 and analyzer executables/config; serialize all Go/cache commands. Retain source revision and candidate hashes, commands, complete meaningful RED/final outputs, exit codes, elapsed times and actual skips in one compact handover. Check-only make -C tools/gomad3 validate precedes broader consumers because schema.go and all authoring files are generator inputs. Output digests hash generated bytes, not source files; no regeneration or pin refresh is admitted.

Run compatibilitypack/authoring and affected canonical, target policy/digest consumers, architecture/purity checks, gofmt and standalone errortype. Run actual unfiltered scoped lint and original integrated lint with base951c5516e9e7b3066e7e069adda9565cfd68844c, fix=false; retain inherited residuals and stage reachability. A policy change does not promise a diagnostic-count reduction. Obtain fresh independent bounded source/evidence progress review and commit separately. Formal implementation SHIP remains deferred on required lint red.

Task21/11 retain original matched-first-baseline, predecessor, preservation, full/default/functional/affected-consumer/formal/native-Darwin/static-both-source-set qualification wherever unproved, with owner/command handback. Missing native Linux proof remains nonblocking under fn128. Developmental linux/arm64 checks establish no native/plugin/cgo execution or host-access claim. Do not bypass platform/runtime guards, suppress analyzers, modify dependencies/pins/grants/generated outputs/CI, or weaken unrelated tests.

### Quick commands

- go test -count=1 -tags test_dep -json ./internal/compatibilitypack ./internal/compatibilitypack/authoring ./internal/canonicaljson
- make -C tools/gomad3 validate
- go test -count=1 -tags test_dep -run TestPackageArchitecture .
- golangci-lint run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/compatibilitypack ./internal/compatibilitypack/authoring
- errortype -test=true ./internal/compatibilitypack ./internal/compatibilitypack/authoring
- make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
- [ ] The two newly banned imports produce a meaningful failing-before/passing-after regression; all five literal loader/request errors, selection/identity revalidation, structural/admission precedence and external wrappers/no-partial-result controls pass.
- [ ] Denied prohibited request facts remain valid and can render/generate with an allowed exact syscall fact; refusal precedes publication and preserves an existing generated root. Emitted packs contain no prohibited grant; valid exact grants, unknown-capability behavior and fixed-input decisions/digests remain unchanged.
- [ ] Production scope contains only the shared two-entry admission extension and owning comment. Task42's only changed expectations are the two R21-authorized loader errors; historical evidence, other tests, source, pins, grants, generated bytes and CI dispositions remain preserved.
- [ ] Check-only generator validation, affected package/consumer and architecture/purity tests, formatting and standalone errortype retain actual results. Unfiltered scoped and original integrated lint retain complete inherited residuals and stage reachability; fresh independent progress reviews precede a separate commit.
- [ ] Task21 consumes this correction through a direct dependency. Original first-baseline, predecessor, preservation, full/default/functional/affected-consumer/formal/native-Darwin/static-both-source-set acceptance remains satisfied or explicitly open under its owner; Linux remains transferred and unverified under fn128.
## Done summary
Blocked:
Blocked on original qualification, not Linux. R21's five-import admission correction is verified source progress. Required integrated lint still has323 findings; formal implementation SHIP and original matched-first-baseline/predecessor/preservation/full/default/functional/affected-consumer/native-Darwin/static-both-source-set acceptance remain open wherever unproved under task11/21.

The shared list now rejects plugin and runtime/cgo through the existing loader, external-loader, token-revalidation and authoring seams. Final literal-approval RED proves both previously published into populated temporary roots; the same five test files pass after the minimal correction. The two authorized task42 loader expectations and owning comment are the only existing behavior/expectation changes. Denied facts and exact syscall/linkname grants remain supported; all1214 protected tracked inputs and51 generated/pin paths retain their hashes.

Actual scoped lint remains7 with no added finding. Actual integrated lint remains323, with319 byte-identical residual blocks and only four one-line schema location shifts from the owning comment. The full integrated errortype stage is unreached; standalone affected-package errortype passes separately. Focused and consumer controls, architecture/purity, formatting and check-only generator validation retain passing outcomes and the one unsupported-host profile skip.

[Worker handover](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-43/handover.md), [root gate and verification](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-43/root-integrated-lint.md), [source review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-43/source-progress-review.md) and [evidence review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-43/evidence-progress-review.md) retain the source-bound evidence and original owner/command handback. Reviews are independent bounded progress checks, not formal SHIP. The worker's pending-root status is an earlier snapshot superseded by the actual root gate and authoritative blocked Flow status.

Task11 keeps sole R17 extraction ownership, and task21 directly consumes task43. Native Linux remains nonblocking and unverified under fn128. No native qualification, plugin/cgo execution or host escape is claimed.

stage: impl-review - deferred(policy: required lint red; original qualification remains open)
stage: plan-sync - skipped(config: disabled; source-progress task remains blocked)
## Evidence
- Commits:
- Tests:
- PRs:
