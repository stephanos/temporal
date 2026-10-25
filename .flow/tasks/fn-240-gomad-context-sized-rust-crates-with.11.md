---
satisfies: [R5, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.11 Extract gomad-campaign

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
crates/gomad-campaign: campaign/{mod, exploration (minus site_scope), counterfactual*, simulation*, choice, inspect, retention, failure_policy}. Depends on gomad-qualify, gomad-replay, gomad-execution. campaign->stress::compare_counterfactual (1) moves into counterfactual. Verb entry points explore/resume/merge/execute_shard are library functions. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); typed error kinds and documented refusal behavior remain tested; message text and goldens may change. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

Complexity amendment (owner, 2026-10-24): while extracting `gomad-campaign`, merge the two preparation layers (`campaign/prepare.rs` + `campaign/prepare/` and `campaign/preparation.rs`) into one module, or record why they are distinct.

Throughput amendment (owner, 2026-10-24): convert this crate's message-prefix dispatch (`runner_failure:`, `refused:`, `capacity:`) to the fn-240.5 kind enum in the same pass as R11's error enums.
## Acceptance
replay->campaign, qualify->campaign and minimize->campaign edges are gone (campaign sits above them); seed campaign fixtures and the scenario preservation tests pass unchanged; contract doc and tests/; measurements in .flow/artifacts/fn-240/gomad-campaign.md. No pub Result<_, String> in the crate; allow-list did not grow.
- One campaign preparation module, or a recorded reason for two.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
