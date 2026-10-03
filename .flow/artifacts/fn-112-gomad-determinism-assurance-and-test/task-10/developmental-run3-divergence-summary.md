## Determinism soak developmental-core-soak on linux/arm64: fail

- N = 4 fresh repetitions per workload and seed (1 batches of 4; minimum 1, maximum 1, stopped at maximum_batches), seeds 11, choice tracing and diagnostics on
- Load: 2 busy host threads on 12 CPUs
- Toolchain: go1.27.1-stub, build key stub
- This run: 1 batches, 4 repetitions, 1 divergences, 0 overflows, 0 target failures, 0 infrastructure failures
- Ledger: 3 retained runs

| Cohort | Workload | Seed | Run repetitions | Cumulative repetitions | Cumulative divergences | Runs | Bound |
| --- | --- | --- | --- | --- | --- | --- | --- |
| a9f9779fb69e64f0 | concurrency-state-invariant | 11 | 4 | 28 | 1 | 3 | 0 |

- concurrency-state-invariant seed 11 batch 1: divergence. fresh repetitions disagree at diagnostics. Retained: divergences/concurrency-state-invariant-seed-11-batch-1.
