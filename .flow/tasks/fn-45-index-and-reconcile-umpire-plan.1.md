---
satisfies: [R1, R2]
---
# fn-45-index-and-reconcile-umpire-plan.1 Build the strict read-only plan index checker

## Description
Implement the reusable parser and pure validation core for R1/R2. Keep repository discovery and diagnostics at a thin command boundary.

**Size:** M
**Files:** `tools/planindex/main.go`, `tools/planindex/index.go`, `tools/planindex/check.go`, `tools/planindex/index_test.go`, `tools/planindex/check_test.go`
**Touches:** [tools/planindex/**]

### Approach
- Follow small command/package separation used under `tools/umpire/cmd/` while keeping the validation core independently testable.
- Decode the closed v1 schema token-by-token so duplicate keys and unknown fields fail; use sorted slices for all rendered output.
- Validate confined normalized paths, complete document coverage, authority references/cycles, Markdown links/anchors, and Flow JSON against fixture roots.
- Never call a mutating command or infer lifecycle/disposition from prose.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md:3-14` — normative authority.
- `.plans/UMPIRE4_ORDER.md:28-31,150-177` — reduced scope, gate, and dispositions.
- `.flow/specs/fn-23-veil-toolchain-compatibility-and.json` — tracked readiness shape.
- `tools/umpire/cmd/umpire-check-legacy-vocabulary/main.go` — focused checker command pattern.

**Optional** (reference as needed):
- `tools/common/artifactio/` — repository confinement/error test patterns; reuse only if it fits without expanding scope.

### Quick commands
`go test -count=1 -tags test_dep ./tools/planindex/...`
## Acceptance
- [ ] Strict parser rejects malformed/duplicate-key/unknown-field/version/enum/type/nullability input with deterministic diagnostics.
- [ ] Pure checks cover complete document and Flow-spec registration, graph integrity, links/anchors, confined paths, exact Flow state/dependencies, and cross-field invariants, including completed-prerequisite specs that remain open under Flow lifecycle policy.
- [ ] Success and multi-error output are byte-stable across reordered fixture input.
- [ ] All checks are read-only and use no new third-party library.
- [ ] Go tests use `require` and whole-value comparisons.
## Done summary
Implemented the strict read-only Umpire plan-index command: closed token-level JSON parsing, deterministic complete registry checks, authority and supersession graph validation, repository-confined Markdown link/anchor checking, and exact committed Flow state/dependency comparison. Tests cover R1/R2 success, parser failures, coverage drift, graph/link/path/Flow failures, open-SHIP completed prerequisites, read-only behavior, and reordered multi-error stability.

The persisted run base is `5e4def302`; `0000c9fc2` is the conductor-owned approved fn45 contract reconciliation, followed by implementation `8b23bc5a7`, review fixes `9a4137a3b`, and review metadata `5971cf936`. The expected greenfield baseline was red because `tools/planindex` did not exist. Final focused tests, package vet, and diff check pass; task-scoped golangci reports zero introduced issues, while the make wrapper remains red only for the inherited `tools/umpire/runtime/errors.go:60` et/unw warning. Flow memory capture was skipped because memory is not initialized.

stage: impl-review - ran [2026-09-01T08:08:35Z..2026-09-01T08:14:04Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0000c9fc223a4e5bb27a26ed7a843d9f3ef5d7c6, 8b23bc5a72e2061142633aad8680b865c9c2db74, 9a4137a3bb47c456c07486c21055a1d8fd161ecd, 5971cf936ce6d5c0cddc7e6b6c115dd033d611ea
- Tests: baseline: red (go test -count=1 -tags test_dep ./tools/planindex/... failed pre-edit because the greenfield package did not exist), go test -count=1 -tags test_dep ./tools/planindex/..., go vet -tags test_dep ./tools/planindex/..., git diff --check, make lint-code GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=0000c9fc2 (golangci: 0 introduced issues; inherited tools/umpire/runtime/errors.go:60 et/unw warning keeps wrapper red)
- PRs: