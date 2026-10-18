---
satisfies: [R1, R2, R3, R6]
---
# fn-69-extract-testpilot-from-umpire.3 Extract the private Testpilot execution core

## Description
Relocate the generic IR, admission, scheduling, recording, and Contract evaluation implementation under Testpilot-private packages while the old runtime remains temporarily for existing callers (R1-R3, R6). Preserve the complete white-box corpus with its owner.

**Size:** L — cohesive internal-package move; splitting would require exported internals or a cross-root internal import
**Files:** `common/testing/testpilot/internal/{ir,execution,verification}/**`, migration ledger
**Touches:** [common/testing/testpilot/internal/**, .flow/artifacts/fn-69-extract-testpilot-from-umpire/migration-ledger.md]

### Approach
- Move the fn-64 implementations and white-box tests together, changing only package/import/proto identities required by the new owner.
- Preserve internal dependency direction: execution may use IR, verification may use execution/IR, and none may import Umpire, adapters, SDK, or canary code.
- Retain cancellation, Observe/Close atomicity, fresh Run state, quarantine, cleanup, proven-violation precedence, nil handling, copying, bounds, and error text exactly.
- Keep the old core unchanged and classified as temporary until consumers move; do not share state or translate between cores.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/execution/contracts.go:53-120` — Driver/Session capability seam
- `tools/umpire/internal/execution/prepare.go:57-85` — static admission
- `tools/umpire/internal/execution/recorder.go` — immutable event ownership
- `tools/umpire/verification/prepare.go:14-100` — Contract preparation
- `tools/umpire/internal/execution/dependencies_test.go:11-20` — import-boundary test pattern
- `.flow/memory/bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04.md` — closure invariant

### Acceptance

## Acceptance
- [ ] Testpilot-private IR/execution/verification packages and their tests preserve the frozen production/test corpus and expose no new public constructors or evaluator substitution seam.
- [ ] Dependency tests prove the private core imports neither Umpire tooling nor functional/canary adapters or SDK packages; no cross-root Go `internal` import is introduced.
- [ ] Focused tests preserve admission errors, scheduler/dataflow behavior, event ordering, cancellation, cleanup/quarantine, violation precedence, concurrent independent Runs, and static/runtime bounds.
- [ ] The ledger accounts for every moved and temporarily duplicated file/symbol; old and new cores do not interoperate, translate, or share mutable state.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: