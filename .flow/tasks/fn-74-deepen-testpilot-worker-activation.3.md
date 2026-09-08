---
satisfies: [R7, R8]
---
# fn-74-deepen-testpilot-worker-activation.3 Expose public preparation diagnostics with external coverage

## Description
Expose the existing preparation taxonomy at the public facade (R7, R8) independently of activation extraction.

**Size:** M
**Files:** new `common/testing/testpilot/preparation_error.go` and external diagnostic tests; `prepare.go`, `profile.go`, package README and `model/ARCHITECTURE.md`.
**Touches:** [common/testing/testpilot/preparation_error.go, common/testing/testpilot/preparation_error_test.go, common/testing/testpilot/prepare.go, common/testing/testpilot/profile.go, common/testing/testpilot/README.md, model/ARCHITECTURE.md]

### Approach
- Define the spec's exact public error fields and six categories; convert existing internal typed errors at NewCatalog/Prepare boundaries with errors.As-compatible wrapping. Preserve existing messages and bounded paths; no error-string classification.
- Attribute non-IR Profile/Catalog preconditions and BindingFingerprint errors to malformed Profile/Catalog paths. Preserve binding limit semantics without inferring categories by parsing text.
- External-package tests cover all six categories across applicable Catalog/Program/Contract failures, including fn78 scoped Contract rejection, wrapped errors and successful preparation. Cover typed-nil Profile implementations for applicable nil-capable kinds using current isNil behavior; NewCatalog takes a concrete pointer.
- Add public API comments and a README errors.As example; update the model architecture overview with links to the canonical package contracts. Exclude ProtoJSON decoding and runtime/Driver failures from this taxonomy.

### Investigation targets
**Required:**
- `common/testing/testpilot/prepare.go:23` — facade and nil preconditions.
- `common/testing/testpilot/profile.go:23` — Catalog and BindingFingerprint errors.
- `common/testing/testpilot/internal/ir/catalog.go:22` — existing six-category error type.
- `common/testing/testpilot/internal/verification/prepare.go:43` — Contract error boundary; scoped_prepare.go uses the same taxonomy.
- `common/testing/testpilot/facade_external_test.go:1` — public-only test pattern.
**Optional:**
- `model/ARCHITECTURE.md:156` — public/Driver ownership overview.

### Quick commands
`go test -tags test_dep ./common/testing/testpilot ./common/testing/testpilot/internal/ir ./common/testing/testpilot/internal/verification`
`go test -race -tags test_dep ./common/testing/testpilot`

## Acceptance
- [ ] Public NewCatalog/Prepare rejections expose the exact R7 PreparationError contract through errors.As, including ordinary wrapping, with retained messages and source paths.
- [ ] Public-only tests cover all categories, Profile/Catalog preconditions, binding errors, scoped Contract failures, typed nils and success; no internal imports or message parsing.
- [ ] Runtime/Driver/ProtoJSON errors and public prepared-plan/wire behavior remain unchanged under R8.
- [ ] Public comments, package guidance and architecture links describe the final classification and ownership; preserve existing comments.
- [ ] Focused unit/race checks pass. Whichever task completes last also runs combined facade/Driver suites and `make lint-code GOLANGCI_LINT_FIX=false`, reporting exact inherited failures separately.

## Done summary
# fn-74-deepen-testpilot-worker-activation.3 handover

Worker handover status was in_progress; conductor completed the mandatory review before recording done.

Implemented the exact public PreparationError category/Path/Detail contract with six categories, errors.As-compatible boundary translation, and retained original causes/messages. NewCatalog and Prepare translate internal typed diagnostics without string parsing. Profile/Catalog preconditions and BindingFingerprint failures use malformed Profile paths; binding rejection ceilings and fingerprints remain unchanged.

External-package tests (no internal imports) cover all six categories, ordinary wrapping, original messages and bounded paths, Catalog graph/intrinsic/unknown/limit failures, Case/Program/Contract rejection, fn78 scoped version/binding/evidence/limit failures, nil Case/Profile/Catalog and all applicable typed-nil Profile kinds, binding failures, successful preparation, and ProtoJSON/runtime exclusions. README includes an errors.As example; model architecture links to canonical facade and Driver ownership contracts. All existing comments and prior dirty source changes are preserved.

Baseline: green. Both exact pre-edit Quick unit/race commands passed. Behavioral red was observed before boundary translation; fixture compile typo and one validator-path expectation were corrected based on existing source.
Final verification: both task Quick commands and combined facade/Driver unit/race suites passed, including SDK replay and Stop regressions. Every executed suite has one exact command/log/terminal-exit record in /tmp/fn74-task3-evidence.json; none was rerun solely to observe a result.

Full make lint-code GOLANGCI_LINT_FIX=false remains inherited red: 1284 raw findings / 825 normalized distinct. Exact normalized (path, diagnostic text) sets equal /tmp/fn74-task2-conductor-lint.log and /tmp/fn74-task2-lint-verified.log, with no additions or removals. Three introduced NotErrorAs lint findings were fixed and final gates rerun. Makefile's separate go-vet phase was not reached because golangci-lint failed. Comparison: /tmp/fn74-task3-lint-comparison.json.

Task-only delta: /tmp/fn74-task3.patch
Approved changed paths: /tmp/fn74-task3-paths.txt
Final frozen hashes: /tmp/fn74-task3-frozen-hashes.json
The pre-task baseline hashes/copies/base files were not modified. Approved files absent from the Frozen86 copies were clean at baseline, so their task delta uses the pinned base commit. No prior source drift or unexpected source changes were found. Administrative .flow state is conductor-owned and excluded from the task delta.

Base commit: 375abfe180dba72da6dd357e6abe33fa75a292a7
Commits: [] (no staging, commits, push, or worktrees).
No review verdict, Flow lifecycle mutation, memory/gate/tracker writes, delegation, or next-task work was performed.

stage: impl-review - ran; SHIP, no findings; receipt /tmp/fn74-task3-impl-review.json (model: gpt-6-astra at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)

Limitations: inherited lint failure and unreached separate go-vet phase; no live-cluster or integration-tag tests requested or run. Runtime/ProtoJSON behavior, activation module, SDK code, IR/evaluator/protocol, and deferred fn79 operation cancellation remain outside this task's changes.

No further edits: source is frozen at the recorded hashes; this worker will make no further source edits after handover.
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot ./common/testing/testpilot/internal/ir ./common/testing/testpilot/internal/verification, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot -run '^TestPreparationError', TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot/temporal/... ./common/testing/testpilot, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false
- PRs: