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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: