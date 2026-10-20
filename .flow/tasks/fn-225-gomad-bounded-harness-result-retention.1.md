---
satisfies: [R1, R2, R3]
---
# fn-225-gomad-bounded-harness-result-retention.1 Bound harness outputs and replace completed results safely

## Description
Implement the approved bounded retention design at the existing harness output lifecycle. Stage complete runs, safely publish latest results, release disposable superseded outputs, and retain bounded failures plus pinned inputs. Keep authenticated receipt paths valid, preserve concurrent runs and unrelated files, and document defaults. Implement and test in this isolated checkout; integrate after fn-189 qualification.

## Acceptance
Repeated real harness runs have bounded storage; failed publication preserves prior results; active and pinned inputs survive; concurrency and path/symlink safety are covered by meaningful filesystem tests. Affected tests and lint pass, and independent review confirms receipt/replay compatibility.

## Done summary
Bounded retention in the shared measurement runner: each producer output root stages runs under .gomad-measure/<producer>/run-* (paths never move, so receipts stay valid), latest.json atomically selects the completed run, and the default keeps the latest success plus one recent failure; active, pinned and --keep runs are exempt and unmanaged evidence is untouched. Real wasmmeasure CLI repeat: 5 successes stay flat at ~897 B with 1 run, 3 failures stay flat with 1 failed run, keep+2 successes stay flat, historical file preserved. Review: two rounds; round 1 found 1 major (store-lock wait vs GB deletes) and 5 minor/nit issues, all fixed in 560b6b72b4 with failing-first tests (--keep in rerun records declined); round 2 approve-with-nits, 4 residual nits accepted (delayed reclaim on trash races or partial deletion, rare dir-fsync misreport, strict .gomad-measure path refusal). Merged as 76bc9b75f5; on the merged tree the host/fs, wasmmeasure, measurement and measure tests pass (Retention x3), build and vet pass, and make lint-code-gomad (base 325eba4425) reports 0 issues.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: bd3c56b484, 3b0fdcee12, 5333cc92e1, 560b6b72b4, 76bc9b75f5
- Tests: go test -tags test_dep ./internal/host/fs/... ./internal/backends/wasm/cmd/wasmmeasure/... ./internal/backends/wasm/internal/measurement/... ./internal/backends/wasm/internal/measure/..., go test -tags test_dep -count=3 -run Retention ./internal/backends/wasm/internal/measure/, make lint-code-gomad GOLANGCI_LINT_BASE_REV=325eba4425
- PRs: