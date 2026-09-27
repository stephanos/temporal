---
satisfies: [R3]
---
# fn-94-simplify-the-testpilot-go-runtime.7 Shared admission primitives exported from ir

## Description
Lane B for the core: `ir` exports `Invalid`, `ValidID`, `IsNil` and `CheckCeilings`, and `execution`, `verification` and the facade call them instead of their copies. The Driver call sites of `CheckCeilings` land in fn-94.9 and fn-94.10.

**Size:** M
**Files:** `common/testing/testpilot/internal/ir/catalog.go` (or a new `internal/ir/admission.go`), `internal/execution/prepare.go`, `internal/execution/contracts.go`, `internal/verification/prepare.go`, `internal/verification/correlated_prepare.go`, facade `prepare.go` and `profile.go`, tests
**Touches:** [common/testing/testpilot/internal/ir/catalog.go, common/testing/testpilot/internal/ir/admission.go, common/testing/testpilot/internal/ir/*_test.go, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/internal/execution/contracts.go, common/testing/testpilot/internal/verification/prepare.go, common/testing/testpilot/internal/verification/correlated_prepare.go, common/testing/testpilot/prepare.go, common/testing/testpilot/profile.go]
**Depends on (cross-spec):** fn-89-one-contract-rule-per-entity.5 (`verification/prepare.go`)

### Approach
- Copies to fold (from the scout): reflect nil check ×6 (`prepare.go:76`, `execution/contracts.go:60`, `ir/catalog.go:274` `missing`; the three Driver copies go in fn-94.8); `validID` ×3 (`execution/prepare.go:48`, `verification/prepare.go:117`, `profile.go:207`); `invalid` with 256-byte truncation ×3 (`ir/catalog.go:43`, `execution/prepare.go:42`, `verification/prepare.go:109`); ceiling loops ×3 in the core (`execution/prepare.go:121`, `verification/prepare.go:151`, `verification/correlated_prepare.go:106`).
- `CheckCeilings` takes the path and the fields to skip, and a way to put the field name in the path or the detail, so each caller's pinned path and detail are unchanged. Write a table test over the three call shapes before swapping callers.
- Preserve existing comments on moved helpers.

### Investigation targets
**Required:**
- `common/testing/testpilot/internal/ir/catalog.go:35-60,270-280`
- `common/testing/testpilot/internal/execution/prepare.go:40-130`
- `common/testing/testpilot/internal/verification/prepare.go:100-160`
- `common/testing/testpilot/internal/verification/correlated_prepare.go:100-120`

### Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/...
make umpire-check-case-runtime-conformance
make lint-code-fast
```

## Acceptance
- [ ] No core package keeps a private copy of the four primitives.
- [ ] A table test pins `CheckCeilings` for all three call shapes (path, detail, skipped fields).
- [ ] Corpus unchanged; tests and lint pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
