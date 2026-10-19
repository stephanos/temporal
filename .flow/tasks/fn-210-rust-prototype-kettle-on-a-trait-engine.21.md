---
satisfies: [R8, R23, R29]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.21 Remove fixture snapshot symlink aliases

## Description
Owner-approved simplification: consumers read Insta payload files directly; remove generated logical-filename symlinks, keep payload bytes and metadata unchanged, preserve consumer selection and pending snapshot acceptance. Touches: model/umpire/testing/**, model/umpire/**/tests/**, model/umpire/tests/fixtures/**/expected/**, model/umpire/hosts/go/host_test.go, model/README.md. Keep the ongoing claim and transition authoring changes separate.

## Acceptance
No tracked fixture symlinks remain outside the archive. Native runner updates and verifies payload-only fixtures including non-text payloads; aliases are not recreated. Native wasm and Go host parity use snapshot payloads. Existing payloads and metadata byte-identical; model-check passes.

## Done summary
Removed 169 tracked fixture symlink aliases, including reports, states, rows, paths, DSL values and macro expansions. Snapshot payload and metadata bytes are unchanged. Native runner checks payload-only fixture groups, does not recreate aliases during updates, and retains logical names for renderer registration. WASM native dispatch and Go host parity read payloads directly. README and the migration plan describe the payload-only layout.

Verification: regression test failed before implementation because the old runner recreated an alias. Updated consumers suite passed (3 passed, optional CLI test ignored); the optional real cargo-insta pending/accept test then passed separately. Integrated make model-check exited 0, including native fixtures, capability controls, wasm builds and Go-host parity. git diff --check passed; no snapshot metadata/payload changes.

Committed separately as 556ec3442a after independent gpt-6.1-sol high read-only review found no issues. Unrelated owner planning edits remain uncommitted and were not included in this checkpoint. This cleanup does not rename fixtures. The conductor also ran the combined post-.19 gate, which passed; one integrated warm timing was full4.005s and inner2.305s.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 556ec3442a
- Tests: Red: cargo test -p umpire-testing --test consumers consumers_select_generic_snapshot_files -- --exact exited 101 on the unwanted alias assertion, Green: cargo test -p umpire-testing --test consumers exited 0 (3 passed, 1 optional ignored), cargo test -p umpire-testing --test consumers accepts_generic_pending_snapshots_with_cli -- --ignored --exact exited 0, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check exited 0; log .flow/tmp/fn210-21-model-check.log, git diff --check exited 0; tracked snapshot payloads/metadata unchanged, Independent gpt-6.1-sol high scope review: no findings, Combined post19 make model-check and single model-time exited0; full4.005s inner2.305s
- PRs: