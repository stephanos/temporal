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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: