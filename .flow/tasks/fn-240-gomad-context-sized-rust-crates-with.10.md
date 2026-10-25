---
satisfies: [R5, R9, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.10 Extract gomad-replay, gomad-qualify, gomad-sweep, gomad-shim

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
Four sibling crates above gomad-execution: gomad-replay (replay, minimize), gomad-qualify (qualify minus spec/workloads/usage, runtime_qualification), gomad-sweep (sweep), gomad-shim (shim). They compile in parallel; none may depend on another or on campaign. minimize->live_backend hooks go through gomad-fixture's FaultHook (task .6). Keep each verb's public flags and JSON results unchanged. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); typed error kinds and documented refusal behavior remain tested; message text and goldens may change. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

Throughput amendment (owner, 2026-10-24): convert this crate's message-prefix dispatch (`runner_failure:`, `refused:`, `capacity:`) to the fn-240.5 kind enum in the same pass as R11's error enums.
## Acceptance
cargo tree shows no edge among the four crates and none to gomad-campaign; replay/minimize/qualify/sweep/shim fixture suites pass unchanged; each crate has contract doc and tests/; measurements in .flow/artifacts/fn-240/consumers.md. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
