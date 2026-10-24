## Memory findings

| Track | Category | Entry | Why relevant |
|---|---|---|---|
| bug | build-errors | `ci-job-timeout-must-exceed-a-gates-go-2026-09-26` | Direct timing-gate lesson: outer job timeout must exceed the inner gate timeout, especially when output is buffered until exit, or a hang loses diagnostics. |
| bug | build-errors | `package-only-model-headers-need-normal-2026-10-06` | A zero exit can hide compiler diagnostics; inspect emitted diagnostics and retain execution evidence. It specifically says inherited warnings remain separate and cannot waive a new compiler error. |
| bug | integration | `behavior-neutral-refactors-must-not-2026-09-04` | Keep behavior-neutral savings/refactor scope from strengthening validation; isolate newly discovered hardening as separate work. |

### Relevance

- Searches did not find a direct entry defining this timing gate's fail-fast contract, exact stderr/stdout capture, or soft-warning versus required/default-mode semantics. The first two entries support timeout hierarchy and diagnostic inspection only.
- No relevant declined-ledger concept was found. The only declined file, `generated-api-drift-verification.md`, concerns broad generated Lean API drift checks and CI coverage and is unrelated here.

### Query details

- Cached `.flow/tmp/fn247-savings-memory-query.json` used `model timing validation required backend gate exit diagnostics soft target`; retrieval reports `bm25`, `rerank_reason: no_key`, `stage_line: memory: bm25 (jev-unavailable(no_key))`.
- Ran three active-only searches without `--rerank`: `timing gate fail fast timeout required default mode validation`; `stderr stdout exact status exit diagnostics backend gate`; `soft warnings required mode optional target only default behavior`. No judge retry.
