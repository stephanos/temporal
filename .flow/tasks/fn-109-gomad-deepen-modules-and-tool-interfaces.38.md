---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.38 Preserve target capability and cache digests while repairing lint

## Description
Repair the five target compatibility-import findings and three prepared-cache SHA-256 bookkeeping findings under R18/R19. This corrective owner does not widen tasks 9, 10, 11, 19 or 23.

**Size:** M
**Files:** the five capability import files, prepared_cache.go and a focused digest test file listed below.
**Touches:** [tools/gomad3/target/capability.go, tools/gomad3/target/capability_collection.go, tools/gomad3/target/capability_evaluation.go, tools/gomad3/target/capability_golden_test.go, tools/gomad3/target/capability_review_test.go, tools/gomad3/target/prepared_cache.go, tools/gomad3/target/prepared_cache_digest_test.go, tools/gomad3/target/internal/capabilitypolicy/policy.go, tools/gomad3/target/internal/capabilitypolicy/policy_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-38/import-alias-2026-10-05/**]

### Approach

- Admit source work only after the current predecessor candidate is integrated, independently source-reviewed and committed. Revalidate the target research bindings from .flow/tmp/fn109-target-lint-research.md against HEAD. Root owns lifecycle, review, evidence and commits; one shared-checkout source/cache writer runs at a time.
- Before production edits, retain actual unfiltered pinned target lint and add literal BASE digest controls. Five explicit compatibility aliases preserve all import uses and public projections. Replace exactly the three unchecked fmt.Fprintf calls using the existing infallible SHA-256 Write pattern at tools/gomad3/internal/sourceinventory/inventory.go:82; preserve each format and argument, without collecting the complete input stream or adding an impossible failure branch.
- Pin overlay empty/multiple replacement inputs, sorted original keys and map-order independence, replacement read failure, module-file present/absent/empty states, argument order and basename framing. Use fixed fictitious absolute original names with real temporary replacement files; originals need not exist and replacement paths stay outside the digest. Expected hashes are literal independent BASE values, never computed by the candidate under test. Preserve %s\x00%x\n and %s\x00absent\n, lowercase hexadecimal and sha256: exactly.
- Preserve all original production and fixture logic/comments/assertions outside the eight admitted statements and new controls. Retain module/source/profile identities, admission errors and go.mod/go.sum immutability. Source code, runtime overlays, pins, lint rules and compatibility grants outside Touches stay unchanged.

### Investigation targets

**Required:**
- tools/gomad3/target/prepared_cache.go:225-268
- tools/gomad3/target/capability_collection.go:188
- tools/gomad3/internal/sourceinventory/inventory.go:82
- tools/gomad3/target/capability_golden_test.go:24
- tools/gomad3/target/capability_projection_test.go:13-132

**Optional:** tools/gomad3/target/prepared_cache_test.go for native cache integration requirements.

### Quick commands

From tools/gomad3 with pinned stock Go1.27.1 first on PATH, GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= and both Gomad seed variables unset:

```sh
go test -count=1 -tags test_dep ./target -run 'TestPreparedCacheDigest|TestCapabilityReviewGoldenCanonicalBytes|TestCompatibilityPackProjectionPreserves'
go test -count=1 -tags test_dep . -run '^TestPackageArchitecture$'
/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./target
```

Run pinned errortype, formatting and generator validation where input ownership requires it. Capture the unfiltered lint RED before edits and its exact source-bound final diagnostic delta, including residuals. The expected eight repairs and nine residual cleanup findings are hypotheses until the actual command establishes them. No filtered gate, suppression, whole-scope subtraction or native qualification claim is allowed.

Root commits reviewed source progress before admitting the cleanup writer. Original R18/R19, task21/predecessor acceptance, matched first-baseline identities, complete/full/default/functional/affected-consumer/formal and native Darwin gates remain open wherever unproved. Native Linux execution remains with fn-128 and does not block this source owner.

### Source progress, 2026-10-05

The frozen candidate repairs exactly five compatibility import aliases and three infallible SHA-256 record writes. Six new literal digest tests pass on unchanged production and the final source. Formats, arguments, order, error wrapping and input bytes are preserved. Actual unfiltered target lint improves from 17 findings to nine unchanged task39 cleanup findings; it is still red.

The fresh independent same-family source review returned SOURCE_PROGRESS_COMMIT_ONLY with no actionable introduced defects. Its requested Sol6.1/high pin has no exposed executed-model attestation. Root independently reran the focused controls (10 top-level tests/20 leaves), architecture and errortype successfully and reproduced all nine unfiltered residuals. The final 1,039-entry manifest matches; exactly six existing inputs change, one test is added and 1,032 protected entries are unchanged. Pinned tools/configuration and all 12 retained raw-log bindings match.

Evidence: .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-38/handover.md, source-bindings.md, independent-source-review.md and conductor-verification.json. The handover/evidence.json retain the worker's pre-review snapshot; this later progress record supersedes their pending-review statement only. Root commits this source progress before admitting task39.

Acceptance remains open: task39 cleanup keeps target lint red, and original R18/R19, predecessor/task21, matched first-baseline, full/default/functional/affected-consumer/formal/native Darwin requirements remain unproved where recorded. No original acceptance is waived or narrowed. Linux execution remains transferred to fn128 and nonblocking.

stage: source-review - ran (same-family source-progress review; no formal verdict)
stage: impl-review - skipped(policy: actual unfiltered target lint remains red)
stage: plan-sync - skipped(config: disabled; no task completed)

### Later linked cleanup checkpoint, 2026-10-05

Task39's independently reviewed source candidate resolves the nine inherited cleanup findings. Fresh combined focused controls and actual unfiltered target lint pass with zero target issues on its 1,040-entry source closure. This supersedes the earlier checkpoint's lint blocker only; its source-bound red receipts remain historical. Original unproved qualification requirements still keep task38 open. See task39's independent-source-review.md and conductor-verification.json; no task completion or formal/native proof is claimed.

### Leaf import-alias admission, 2026-10-05

The integrated lint gate at source `3b15e3cab90c8d72ac5b7d298ebdb80537906145` also reports the same compatibility import-name mismatch in the private capabilitypolicy leaf. Admit exactly one explicit compatibility alias in each of policy.go and policy_test.go. The package identifier and every selector remain unchanged. This extends the earlier alias repair under R18/R19, not task11's R17 ownership.

This amendment supersedes the earlier eight-statement protection only for those two import declarations. Preserve all other source bytes, comments, assertions, existing digest controls, canonical output, module/profile/pin/generator inputs and lint rules. Previous manifests protect these files and remain immutable historical receipts; do not run their old protected-source verifier as current proof or rewrite them. Retain separate BASE/final bindings for the new candidate.

Reuse the integrated, independently reviewed and committed predecessor. Root owns Flow state, review and commits; one source/cache writer runs at a time. No dependency changes are required, and task21 already consumes task38. The original first five acceptance bullets remain in force for their admitted checkpoints. Do not reinterpret their historical counts as current findings.

Before editing, actual unfiltered pinned lint of ./target and ./target/internal/capabilitypolicy must reproduce the two goimports findings. Run the existing leaf policy suite and the target digest/golden/projection controls on BASE and final source. The BASE passed four top-level policy tests (16 including subtests) and ten target controls (23 including subtests); retain actual final counts and failures/skips. After editing, run the same unfiltered two-package lint, pinned errortype on both packages, gofmt checks and root TestPackageArchitecture, TestPureModulesHaveNoHostEffects and TestExactModuleEdges. Check generator input ownership and use check-only make validate at the concrete compatibility boundary; do not regenerate unchanged inputs.

Run the real root make lint-code-gomad3 gate on the frozen final source with original comparison revision 951c5516e9e7b3066e7e069adda9565cfd68844c, the pinned tools/configuration and FIX=false. Its BASE loaded 55 ordinary host packages and reported 327 findings; golangci-lint failed before errortype. Preserve the complete actual final output and package inventory; never infer its count by subtraction or claim the full errortype stage ran when fail-fast prevents it. Other findings require their own owners; do not broaden, suppress or change the baseline to clear this gate.

A fresh independent source-progress review must find no introduced defects before root commits this correction and its evidence. Keep task38 blocked for every unproved original R18/R19, predecessor, first-baseline, full/default/functional/affected-consumer/formal/native Darwin requirement. Scoped lint success is not qualification or task completion. Linux remains nonblocking under fn128.
## Acceptance
- [ ] Literal digest and error controls pass on unchanged BASE and final source, covering every named case with independent expectations; all five capability alias changes preserve public projection/canonical behavior.
- [ ] Exactly three hash-write statements change, with unchanged formats/arguments/order and no new possible failure, full-stream copy or module mutation; original logic/comments/assertions outside scope remain intact.
- [ ] Actual unfiltered pinned target lint reproduces the eight mapped findings before edits and removes exactly those afterward; every residual and unexpected finding is retained. Focused controls, architecture, errortype, formatting and applicable generator checks pass on frozen sources.
- [ ] A fresh independent source review finds no actionable introduced defect; root commits source, controls and owned Flow evidence before another writer. Source progress is distinguished from formal qualification.
- [ ] Required original R18/R19 and predecessor/full/default/functional/affected-consumer/formal/native Darwin acceptance is proved before task completion. Task21 consumes this evidence; task21 completion is not a prerequisite for starting this corrective source work. Missing or red source-owned gates keep acceptance open; transferred Linux proof is nonblocking.
- [ ] The two capabilitypolicy import aliases preserve every other source byte and the existing package namespace. Matched BASE/final policy and target controls, architecture/purity/edge checks, unfiltered two-package lint, errortype, formatting and check-only validation pass on frozen sources. Retain fresh bindings, immutable historical receipts, the actual integrated gate inventory/output and a fresh independent source-progress review; original qualification remains open where unproved.
## Done summary
Blocked:
Reviewed source progress repairs eight target lint findings; nine unchanged cleanup findings remain owned by fn109.39. Original R18/R19, predecessor/task21, matched first-baseline, full/default/functional/affected-consumer/formal/native Darwin gates remain required and unproved wherever recorded. Native Darwin and full qualification are unavailable on this developmental linux/arm64 host. Do not repeat unchanged host failures. Native Linux execution is deferred under fn128 and is not this task's blocker. Resume acceptance after task39 and the required source-bound supported-host evidence; source progress is not task completion.

Blocked:
Task39's independently reviewed linked source candidate repairs all nine inherited cleanup findings; combined focused controls and unfiltered target lint now pass with zero target issues. Source progress is committed without claiming task completion. Original R18/R19, predecessor/task21, matched first-baseline, full/default/functional/affected-consumer/formal/native Darwin requirements remain required and unproved wherever recorded. No original gate is waived. This developmental linux/arm64 host cannot provide supported native Darwin/full qualification; do not repeat unchanged host failures. Native Linux execution is deferred under fn128 and is not a blocker here.
## Evidence
- Commits:
- Tests:
- PRs:
