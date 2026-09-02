---
satisfies: [R6, R7]
---

# fn-28-portable-evaluation-contract-and.9 Run the no-Lean end-to-end disposable-cluster mini-demo
## Description

Add the sole tagged Go integration test using `testcore.NewEnv`; keep one disposable cluster and resident HTTP executor alive while running the pre-generated normal and duplicate-delivery contracts.

**Size:** L
**Files:** `tests/umpire4_portable_executor_test.go`
**Touches:** [`tests/umpire4_portable_executor_test.go`]

### Approach
- Construct the concrete attached-authority adapter from `testcore.NewEnv` without exposing `testing.T` or testcore types through production interfaces.
- Send both contracts through the same live HTTP handler/executor/cluster with fresh run identities and isolated run-owned workers/resources.
- Require normal detailed statuses plus local `pass`, and negative-control detailed statuses plus trustworthy local `fail`; assert the process invokes no Lean/Make/shell checker.

### Investigation targets

**Required** (read before coding):
- `tests/testcore.NewEnv`, its SDK client/namespace/cleanup interfaces, and existing tagged functional test conventions.
- Existing generated caller-closure path, live negative control, and parent executor/HTTP adapter.
- Project test-tag and `require` conventions.

## Acceptance
- [ ] `go test -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$'` runs the exact two live contracts through one resident process and one disposable cluster.
- [ ] The normal run passes locally; duplicate delivery fails locally for the expected semantic clause; all detailed stages and cleanup remain inspectable.
- [ ] No Lean, Make, `mise`, `lake`, nested `go test`, or generated per-verification Go process is invoked at test runtime.

## Done summary
Added the sole tagged portable canary integration test. One `testcore.NewEnv` disposable cluster, attached authority, resident executor, and HTTP server execute the pre-generated normal and duplicate-delivery contracts sequentially with fresh run, task-queue, endpoint, workflow, operation, and correlation identities. The normal contract produces detailed accepted/applied/satisfied/complete statuses and local `pass`; the duplicate-delivery contract produces accepted/applied/violated/complete statuses and a trustworthy local `fail` at the uniqueness clause. Cleanup removes every Nexus endpoint after each run, and the runtime executes with an empty `PATH`, proving no Lean, Make, `mise`, `lake`, shell, nested `go test`, or generated Go process is available.

Aligned the live evidence boundary required by that portable path: environment receipts use the cleanup source/kind; successful participant lifecycle receipts retain only semantic realization evidence; synthetic duplicate delivery has a dedicated raw kind and cancellation coordinate; the runtime engine, legacy Lean projection, and canonical fixtures preserve exact raw identities while remaining compatible with the existing semantic model. The stale parity-generator normalization found by review was removed and all affected fixtures regenerated.

Fresh verification passed for the exact tagged acceptance test, hermetic portability, affected runtime/checker packages, portable-evaluation parity, and canonical fixture drift. The task-scoped Codex review reached SHIP after its sole P2 fixture-generation finding was fixed. Existing comments remain intact, and the unrelated user-owned config/schema modifications remain unstaged and uncommitted.

stage: impl-review - ran (Codex SHIP after one fixed P2 parity-fixture finding; receipt `.flow/tmp/handovers/fn-28.9-review.json`)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 079dced3a2fbbf7eb651d7c59c54bcfe9bddd7a9, abc5df5036a49600f9d017b75171dfcd4a185fde
- Tests: go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$' (pass), go test -count=1 ./tools/umpire/internal/runtimeengine (pass within affected-package run), go test -count=1 ./tools/umpire/runevaluation (pass), go test -count=1 ./tools/umpire/temporal/local (pass within affected-package run), go test -count=1 ./tools/umpire/temporal/nexus (pass within affected-package run), go test -count=1 ./tools/umpire/temporal/nexus -run '^TestHermeticCIPortability$' (pass), go test -count=1 -tags test_dep ./tools/umpire/portableevaluation/... (pass), make umpire-check-portable-evaluation-fixtures (pass), Codex implementation review c95feb08dc36ba7a6ee1167abeabf621b4af5ea3..abc5df5036a49600f9d017b75171dfcd4a185fde (SHIP)
- PRs:
