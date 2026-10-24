---
satisfies: [R5, R9, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.9 Extract gomad-install and gomad-execution; remove cli back-edges

## Description
crates/gomad-install: install, config, backstop, slots, execute, the option parser from cli/explore.rs+cli/exploration.rs, qualify/spec* strict codec; invert install->campaign::prepare::process (3). crates/gomad-execution: campaign/{prepare*, preparation, execution, scenario*, ledger, soft_assertions, launcher, seeds, shards}, qualify/workloads, qualify/usage. Move cli::{install,replay}::run and retain_observed bodies into their owning library modules so campaign->cli (16), replay->cli (4), measurement->cli (4), runtime_qualification->cli (2), qualify->cli (1) disappear. Single base64 helper in gomad-evidence replaces campaign::exploration::base64 and qualify::cells::base64. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); pinned messages stay byte-identical. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

## Acceptance
no crate below the bin references a cli module; cargo test -p gomad-install and -p gomad-execution run in isolation; operator_cli e2e tests pass unchanged; contract docs and tests/ present; measurements in .flow/artifacts/fn-240/gomad-execution.md. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
