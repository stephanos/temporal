---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.44 Repair mechanical source-record conversion and import lint

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
Repair the four inherited mechanical compatibility-pack lint sites from reviewed source commit 13df4f16f90d49938ea123a29859da62ad2cab9f. Advances R18/R19 without a new policy or preservation exception. Task11 retains R17, task42 retains exhaustive correction ownership, and task43 retains R21. Source admission has no blocked completion prerequisite; root owns Flow, scope, review and commits.

**Size:** M verification, S implementation
**Files:** tools/gomad3/internal/compatibilitypack/schema.go, mutation_test.go, schema_timezone_test.go; task44 evidence.
**Touches:** [tools/gomad3/internal/compatibilitypack/schema.go, tools/gomad3/internal/compatibilitypack/mutation_test.go, tools/gomad3/internal/compatibilitypack/schema_timezone_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-44/**]

### Approach

Retain fresh unchanged BASE test/lint controls before editing. Use equivalent destination-typed conversions at schema.go:292 and mutation_test.go:98,102, and regroup only the imports at schema_timezone_test.go:3. The paired structs contain the same ordered string fields; JSON tags differ but the converted values retain the destination types. Preserve allocations, loops, indexes, iteration and activation order, empty-slice behavior and detached ownership. Keep schema.go:295's foreign-source Kind-colon-Name construction and DigestSources' NUL framing unchanged. Preserve all three ST1005 error strings, every existing comment, test body/assertion, policy and admission control. No mirror tests are needed for a mechanical conversion already covered by the existing source-inventory and canonical tests.

### Investigation targets

Required: schema.go:101,106,292,295; policy.go:48,53,215; mutation_test.go:15,98,102; schema_timezone_test.go; policy_test.go:42,72; tools/gomad3/Makefile:9,169; task43 root-integrated-lint.md. Reuse existing generated mutation, complete-inventory selection, DecodePackV2, governance canonical/error-priority, policy-copy and five-import controls.

### Verification

Reuse task44's actual BASE captures; match tools/config/offline environment and serialize all Go/cache work. Capture revision, before/after source/tool hashes, exact argv/cwd, elapsed time, exit, complete output and actual skips. Check-only generator validation precedes broader consumers because schema.go is a generator input; unexpected drift requires investigation, not regeneration or pin refresh. Compare exact admitted changes and protected tracked source/generated/pin hashes.

Run complete compatibilitypack, authoring and canonicaljson packages, affected target policy/digest consumers, architecture/purity controls, gofmt and standalone errortype. Existing package tests should pass both before and after; the actual configured analyzer is the meaningful RED. Run unfiltered scoped lint and actual original integrated lint with comparison base951c5516e9e7b3066e7e069adda9565cfd68844c and fix=false. Scoped7→3 and integrated323→319 are hypotheses, not results. Prove only the three S1016 and one gci blocks disappear, with all remaining diagnostic bodies/counts preserved except justified locations. Record actual Make exit and errortype stage reachability separately from standalone success.

Obtain fresh independent bounded source and evidence progress reviews, then commit separately. Formal implementation SHIP remains deferred while required lint is red. Task21 directly consumes this correction. Original matched-first-baseline, predecessor, fixed-identity preservation, full/default/functional/affected-consumer/native-Darwin/static-both-source-set qualification remain required and explicitly open wherever unproved under task11/21. Native Linux remains nonblocking and unverified under fn128. Developmental linux/arm64 checks establish no native qualification. Do not bypass guards, suppress analyzers, alter dependencies/pins/generated bytes/grants/CI or rewrite historical task42/43 evidence.

### Quick commands

- go test -count=1 -tags test_dep -json ./internal/compatibilitypack ./internal/compatibilitypack/authoring ./internal/canonicaljson
- make -C tools/gomad3 validate
- go test -count=1 -tags test_dep -run 'Test(PackageArchitecture|PureModulesHaveNoHostEffects|ExactModuleEdges|PublicPackagesDoNotExportTypeAliases|DomainModulesDoNotExportWireFraming|RunnerExecutionInjectionIsPrivate)$' .
- golangci-lint run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/compatibilitypack ./internal/compatibilitypack/authoring
- errortype -test=true ./internal/compatibilitypack ./internal/compatibilitypack/authoring
- make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
- [ ] Exact source proof confines changes to three equivalent destination-typed struct conversions and one import regrouping; errors, comments, assertions, source-inventory framing, allocation/copy behavior, policy, pins and generated bytes remain unchanged.
- [ ] Matched unchanged BASE/final package, mutation, selection, canonical/governance, admission and affected consumer controls retain actual passing results and skips; no native qualification is inferred.
- [ ] Check-only generator validation precedes broader consumers; architecture/purity, formatting and standalone errortype retain actual results. Protected-input manifests and candidate/tool hashes bind the evidence to the frozen source.
- [ ] Actual unfiltered scoped and original integrated lint remove exactly the four admitted findings with no introduced diagnostic; residuals, tool/config/base identities, Make exits and full integrated errortype reachability are retained. Fresh independent bounded source/evidence reviews precede the separate progress commit; formal SHIP is not inferred from them.
- [ ] Task21 directly consumes task44; task11/R17, task42 and task43 ownership and historical evidence remain unchanged. Original matched-first-baseline/predecessor/full/default/functional/affected-consumer/native-Darwin/static-both-source-set/formal requirements remain satisfied or explicitly open under their owner. Linux stays with fn128.


## Done summary
Blocked:
Blocked on original qualification, not Linux. The four-site R18/R19 source correction is verified progress: three per-element source-record conversions and test import regrouping, with exact reconstruction and no policy/error/comment/assertion/fixture change.

Actual scoped lint falls7→3, retaining byte-identical ST1005 blocks. Actual original integrated lint falls323→319, with exactly three S1016 and one gci block removed, no additions/shifts, and every319 residual block byte-identical. Top-level Make returns2; integrated errortype remains unreached. Standalone affected-package errortype passes separately.

BASE/final package controls retain469 pass/1 unsupported-profile skip and bounded target controls37 pass. Check-only generator validation, six architecture/purity tests and formatting pass. The initial broader BASE target selection retains38 pass/1 missing-patched-toolchain failure; narrower controls do not qualify it. The worker's1264 selected-file manifest retains1261 protected inputs,51 generated/pin paths and five executable hashes. Selected evidence establishes no full-repository/toolchain closure or native qualification.

[Worker handover](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-44/handover.md), [root actual gate](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-44/root-integrated-lint.md), [source review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-44/source-progress-review.md) and [evidence review](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-44/evidence-progress-review.md) retain the source-bound checks and original owner/command handback. Reviews are bounded progress checks, not formal SHIP. The worker's pending-root status is an earlier snapshot superseded by the actual root gate and authoritative blocked Flow status.

Task11 keeps R17, task42 its exhaustive correction and task43 R21; task21 directly consumes task44. Original matched-first-baseline, predecessor, fixed-identity preservation, full/default/functional/affected-consumer/native-Darwin/static-both-source-set/formal requirements remain required and open wherever unproved under task11/21. Required lint stays red. Native Linux remains nonblocking and unverified under fn128.

stage: impl-review - deferred(policy: required lint red; original qualification remains open)
stage: plan-sync - skipped(config: disabled; source-progress task remains blocked)
## Evidence
- Commits:
- Tests:
- PRs:


## Format-compatibility amendment (2026-10-09)

Per the spec's 2026-10-09 amendment, byte-for-byte and format compatibility is no longer required. "Generated bytes unchanged" is no longer required; generated output may change if regenerated and validated in the same change.
