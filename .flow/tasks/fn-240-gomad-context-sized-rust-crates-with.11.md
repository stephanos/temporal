---
satisfies: [R5, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.11 Extract gomad-campaign

## Description
crates/gomad-campaign: campaign/{mod, exploration (minus site_scope), counterfactual*, simulation*, choice, inspect, retention, failure_policy}. Depends on gomad-qualify, gomad-replay, gomad-execution. campaign->stress::compare_counterfactual (1) moves into counterfactual. Verb entry points explore/resume/merge/execute_shard are library functions. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); pinned messages stay byte-identical. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

## Acceptance
replay->campaign, qualify->campaign and minimize->campaign edges are gone (campaign sits above them); seed campaign fixtures and the scenario preservation tests pass unchanged; contract doc and tests/; measurements in .flow/artifacts/fn-240/gomad-campaign.md. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
