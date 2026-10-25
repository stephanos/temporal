---
satisfies: [R1]
---
# fn-245-gomad-typed-boundaries-for-the-rust-host.3 Replace message-prefix dispatch with typed outcome kinds

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
Production code branches on error text: execute.rs:13 (message.starts_with('compile cache:')), measurement/mod.rs:236 and sweep/mod.rs:64 (starts_with('refused:')), wasmhost engine.rs:345 (to_string().starts_with('instrumentation divergence:')), engine.rs:468 and :656 (contains('capacity:')), plus the 'runner_failure: ' (51 sites), 'refused:' (31), 'WASI capacity:' (24) prefix conventions that callers parse. Make the classification a typed field (an outcome Kind enum carried by HostError / the host-side error enums from fn-240 R11) and dispatch on it; keep the prefixed messages byte-identical in output and goldens. Sequence after fn-240.5 (HostError) and alongside each crate's R11 conversion.

Throughput amendment (owner, 2026-10-24): fn-240 R11 extractions convert prefix dispatch crate by crate (fn-240.5, .7–.13). This task is the final sweep: remove leftover prefix matching and prove none remains.
## Acceptance
grep for starts_with/contains on error text in production code is empty; every kind literal is produced by one as_str; CLI exit statuses and message goldens unchanged; cargo test --workspace green.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
