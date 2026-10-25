---
satisfies: [R5, R10, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.12 Split gomad-measure and gomad-stress

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
crates/gomad-measure (measurement minus stress_driver and cluster_driver's stress calls; retention stays here) and crates/gomad-stress (stress, profile, measurement::stress_driver, the stress half of cluster_driver). Breaks the measurement->stress (12) / stress->measurement (8) cycle; stress depends on measure. Size check after fn-242's deduplication: each crate about 17k production lines or less, else record the next cut. gomad-measure and gomad-profile bins move to bins/gomad. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); typed error kinds and documented refusal behavior remain tested; message text and goldens may change. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

Throughput amendment (owner, 2026-10-24): convert this crate's message-prefix dispatch (`runner_failure:`, `refused:`, `capacity:`) to the fn-240.5 kind enum in the same pass as R11's error enums.

Scope note (owner, 2026-10-24): fn-242.10 folds the `gomad-measure` producers into the cycle verb. Extract `gomad-stress` and only whatever measurement code survives; there is no separate `gomad-measure` binary to split.
## Acceptance
no edge measure->stress; stress corpus, measurement_profile_corpus and profile_corpus tests pass unchanged; gomad-measure and gomad-profile bins produce authenticated current-schema receipts with deterministic regeneration; line counts per crate recorded; measurements in .flow/artifacts/fn-240/gomad-measure.md. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
