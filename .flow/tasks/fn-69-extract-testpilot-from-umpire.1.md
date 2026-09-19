---
satisfies: [R6, R7]
---
# fn-69-extract-testpilot-from-umpire.1 Freeze Testpilot migration compatibility and ownership

## Description
Create the migration ledger and freeze the complete fn-64/fn-66 compatibility baseline before any owner or namespace moves (R6, R7). This task authorizes later deletions; it does not move code or create a new scanner.

**Size:** M
**Files:** `.flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md`
**Touches:** [.flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md]

### Approach
- Inventory the five proto sources, ten generated Go files, all active Go/Lean/fixture type-name consumers, generic runtime files, functional adapter files, tests, Make targets, workflow selectors, and active documentation.
- Freeze descriptor shapes/numbers, canonical corpus hashes, generated identities, public API/error text, exact live failure identities, package/test counts, and current focused/full terminal results.
- Record the only permitted namespace-derived substitutions separately from bytes and behavior that must remain exact.
- Classify every old owner and consumer with a destination and task number; zero ambiguous rows is the gate for task 2.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/prepare.go:15-45` — immutable public preparation baseline
- `tools/umpire/internal/execution/runtime.go:12-95` — Run and cleanup precedence
- `proto/internal/temporal/server/api/umpire/v1/case.proto:1-30` — protocol identity and numbering pattern
- `tools/umpire/regression/ci_workflow_test.go:21-193` — path, gate, and documentation enforcement
- `.flow/artifacts/fn-64-umpire-case-runtime/task8-migration-ledger.md` — historical baseline to preserve

### Acceptance

## Acceptance
- [ ] Ledger classifies every protocol, generated file, runtime/facade, adapter, importer, fixture, test, command, selector, and active-document reference with a concrete final owner and migration task; no row is ambiguous.
- [ ] Descriptor numbers/shapes, ordinary wire bytes, namespace-bearing values, generated identities, API/error behavior, live identities, package/test counts, and artifact hashes are frozen separately.
- [ ] Existing focused Case Runtime, Temporal adapter, generator/conformance, model, and aggregate commands record actual terminal results and inherited failures before migration.
- [ ] Fn-64/fn-66 ledgers and retained Umpire authoring/generator sources remain unchanged; no implementation or broad drift gate is added.

## Done summary
Created the Testpilot migration ledger with a closed ownership inventory, compatibility-class baselines, protocol/generated/corpus hashes, public behavior, package/test counts, selectors, active documentation, and exact inherited command failures. The artifact adds no implementation, scanner, alias, converter, or broad drift gate; the Codex review returned SHIP after directly reviewing the user-owned untracked file.

baseline: red (`go test -count=1 -tags test_dep ./common/testing/testpilot/...` and `./tests/testcore/testpilot/...` fail pre-migration because their task-2/task-6 destinations do not exist; focused Temporal packages also inherit the local missing `stddef.h` toolchain failure); canonical generation, conformance, model, generator, and aggregate regression commands are green.

stage: impl-review - ran [2026-09-06T17:41Z..2026-09-06T17:45:07Z]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: git diff --check -- .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md, go test -count=1 -tags test_dep ./common/testing/testpilot/... (exit 1 inherited: pre-migration destination absent), go test -count=1 -tags test_dep ./tests/testcore/testpilot/... (exit 1 inherited: pre-migration destination absent), make proto, make umpire-gen-lean-api, make umpire-check-case-runtime-conformance, cd model && mise exec -- lake build Temporal.CaseRuntimeTests, TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tools/umpire/cmd/umpire-gen-lean-api, TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/temporal/... (exit 1 inherited local toolchain: runtime/cgo cannot find stddef.h; delivery and worker passed), make umpire-check-regression
- PRs:
