---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.42 Preserve exact-pack decisions while repairing exhaustive lint

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
Repair the single FactKind exhaustive finding in compatibilitypack Selection.Evaluate under R18/R19. Task11 retains R17 semantic ownership; task21 consumes this corrective evidence. Source admission starts from integrated, independently reviewed commit 7750f4f57b84289545f6e4972cf8b6fe85c92eb3, not completion of blocked tasks11/21/38/41. Root owns Flow, reviews, scope and commits; serialize source/cache writers.

### Touches

- tools/gomad3/internal/compatibilitypack/policy.go
- New tools/gomad3/internal/compatibilitypack/policy_exhaustive_test.go
- Task42 evidence artifacts only

### Approach

Characterize unchanged production with a validated, selected, genuinely matching pack before editing. Preserve the tagged switch and both existing capability/linkname grant selectors and bodies. Add only one terminal grouped case for FactMalformedLinkname and FactNoReviewedGoSource, continuing the inner candidate-rule loop. There are no post-switch statements in that loop, so this explicitly preserves the original no-grant fallthrough. Unknown and empty kinds retain final denial/remediation. No default, suppression, exclusion, cast or expressionless-switch workaround.

Use the existing validPackV2 fixture and exact package helper without editing either. Add a test-memory linkname with two ordered directives; LoadPack and SelectPacksForPlatform must prove the expected ID/digest, approved platform, HasPackage, exact capability and linkname grants. Assert all Decision fields and literal fixed-input canonical bytes. Exercise matching inventory with malformed/no-source/unknown/empty fact kinds and otherwise grantable fields; denied capability/linkname/source/module/inventory drifts, five forbidden host imports, nil/empty selections, rule/pack traversal, alias and mutation isolation. Existing failure/selection controls remain unchanged. BASE literal controls must run against unchanged production before the two-line correction.

Multi-pack controls must prove later-match traversal and first-grant PackID precedence; rule controls must prove later-match traversal. Special/unknown/empty facts must encounter a selected matching rule containing their otherwise grantable fields and retain empty PackID. Clone mutable test inputs so alias/mutation controls do not alter shared fixtures.

Preserve grants, error bytes/precedence, all other production bytes, existing tests, comments, modules/toolchain/profile/pack/source-inventory pins and generated output. The actual root BASE at /tmp/fn109-pack-policy-base.hc9osLjg has passing package/authoring tests and errortype, unchanged 996-input manifest, and eight unfiltered lint findings. This task owns only the exhaustive finding; retain the seven inherited staticcheck/gci findings. Expected eight-to-seven reduction is a hypothesis until actual unfiltered execution proves it, not a package-green promise. Measure exhaustive 1-to-0, gci 1-to-1, staticcheck 6-to-6 and no new findings. Distinguish the executed standalone errortype pass from the integrated gate's unreached later stage.

### Verification and evidence

Pin offline stock Go1.27.1, tools and repository config exactly as the root BASE. Retain commands, raw outputs, statuses, times, matched source/tool identities and skips. Execute additive BASE/final characterization, full compatibilitypack/authoring and canonical consumers, architecture/purity/exact-edge controls, scoped errortype/gofmt and check-only generator validation. Execute the original integrated lint gate with the frozen fn109 base 951c5516e9e7b3066e7e069adda9565cfd68844c and fix=false; retain residual diagnostics and actual stage reachability. Obtain fresh independent bounded source/evidence progress review. Do not invoke formal implementation SHIP on a red tree or claim narrow progress completes original acceptance.

Original matched-first-baseline, predecessor, R18/R19 preservation, full/default/functional/affected-consumer/formal/native Darwin and static both-source-set requirements remain open wherever unproved, with exact owner/command handback. Native Linux qualification stays nonblocking under fn128. Unsupported linux/arm64 controls are developmental only; do not bypass platform/runtime guards. Commit reviewed progress separately before another writer.

### Quick commands

- go test -count=1 -tags test_dep -json ./internal/compatibilitypack ./internal/compatibilitypack/authoring ./internal/canonicaljson
- golangci-lint run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/compatibilitypack
- errortype -test=true ./internal/compatibilitypack
- make -C tools/gomad3 validate
- make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false

Use absolute cached executables and the root BASE safe offline pins when running these commands; no downloads or generator mutation.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
- [ ] Literal matched-pack BASE/final controls preserve all Decision fields, fixed-input bytes, exact grants/denials, pin matching, remediation, traversal, alias isolation, nil/empty and unknown FactKind behavior; original tests remain unchanged.
- [ ] Exact reconstruction proves the terminal grouped case is the only production change; all other source, comments, errors, grants, modules, toolchain/profile/pack/inventory pins and generated outputs remain preserved.
- [ ] Matched executable package/consumer controls, architecture/purity checks, errortype, formatting and check-only generator validation retain actual outcomes. Unfiltered scoped lint removes exactly the admitted exhaustive diagnostic and preserves every inherited residual.
- [ ] Root retains actual frozen integrated-gate output, identities and stage reachability; fresh independent progress reviews precede a separate commit, without formal SHIP on a red tree.
- [ ] Task21 directly consumes this evidence and all original first-baseline, predecessor, preservation, full/default/functional/affected-consumer/formal/native-Darwin/static-both-source-set requirements remain satisfied or explicitly open with owner/commands; Linux stays transferred to fn128, never falsely qualified.


## Done summary
Blocked:
Blocked on original qualification, not on Linux. The exact-pack exhaustive source correction and additive characterization are verified progress; required integrated lint still has323 findings, and all original first-baseline, predecessor, preservation, full/default/functional/affected-consumer/formal/native-Darwin/static-both-source-set requirements remain open wherever unproved.

The candidate adds only the terminal two-line inner-loop continue case. The final literal test file ran on unchanged production before the source change and unchanged afterward:57 cases/five top-level tests pass. All995 protected inputs, existing tests, grants, errors, comments and pins remain unchanged. Scoped lint is8→7 with exact inherited residual blocks; the original integrated gate is324→323, with no added finding and all323 remaining blocks byte-identical. Its later full errortype stage is unreached; standalone scoped errortype passed separately.

[Root integrated evidence](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42/root-integrated-lint.md), [worker snapshot](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42/handover.md), [source proof](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42/source-proof.json) and [progress reviews](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-42/independent-progress-reviews.md) retain the evidence and original owner/command handback. The worker's in_progress/pending-root-gate state is an earlier snapshot; authoritative Flow is blocked after the actual root gate. Reviews are bounded progress checks, not formal SHIP.

The existing plugin/runtime-cgo pack-policy mismatch is characterized and separately documented; it is not repaired or endorsed here. Task11 keeps R17 and task21 directly consumes task42. Missing transferred Linux execution remains nonblocking under fn128; this checkpoint establishes no native support, qualification, plugin/cgo execution or host escape.

stage: impl-review - deferred(policy: required lint red; original qualification remains open)
stage: plan-sync - skipped(config: disabled; source-progress checkpoint remains blocked)
## Evidence
- Commits:
- Tests:
- PRs:
