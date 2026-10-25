---
satisfies: [R6, R7, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.5 Pre-cut foundations: HostError, identity closure, inventory scanner, workspace lints

## Description
Precondition for every extraction (Amendment 4). (a) Add HostError in wasmhost and replace the 28 wasmtime::{Error, Result} sites below engine (model, namespace, protocol, transcript, storage_transport, world, volume, network); engine converts at its boundary. HostError is the first R11 error enum: it carries the record classification kinds as an enum with as_str, defined once. (b) Extend gomad/build.rs and wasmhost/build.rs so each implementation SHA hashes every workspace crate the consumer depends on (path-dependency walk over cargo metadata), plus measurement-source-inputs.tsv closure; add a test that flips one byte in a dependency crate and asserts the identity changes. (c) Make tests/inventory/host.rs walk every workspace crate root. (d) Move clippy lints and pinned versions to [workspace.lints]/[workspace.dependencies]; add clippy::too_many_lines and clippy::cognitive_complexity at warn, put explicit #[allow] on every current offender and record the list in .flow/artifacts/fn-240/lint-allows.md (R12). No module moves in this task. Ordering: after fn-242 (house convention: cross-spec order recorded here, spec-level dep already present); before .2 and .3.

Crate amendment (owner, 2026-10-24): `HostError` and every later per-crate error enum (R11) derive with `thiserror` (pinned `=`). Message text is not pinned; goldens and CLI tests that pin message text are updated in the same task, exit statuses stay.

Throughput amendment (owner, 2026-10-24): the classification-kind enum R11 introduces also replaces message-prefix dispatch (`runner_failure:`, `refused:`, `capacity:`). Each later extraction converts that crate's prefix dispatch in the same pass as its `Result<_, String>` signatures, so fn-245.3 only sweeps leftovers.

Umpire-parity amendment (owner, 2026-10-25; compared against umpire/temporal/model):
- (e) Every workspace package opts into `[workspace.lints]` with `[lints] workspace = true`, including `tools/source_size`, which has no lint table today and which `lint-rust` and the driver `rust-lint` stage never lint (both list packages by name). The workspace table keeps exactly today's per-crate clippy set; R12 adds the two size lints and nothing else. Extend `lint-rust` and the driver stage to every workspace package here rather than waiting for .13.
- (f) Add `clippy.toml` with umpire's thresholds: `too-many-arguments-threshold = 6`, `type-complexity-threshold = 300`, `cognitive-complexity-threshold = 25`. New offenders join the R12 allow-list under the same rules.
- (g) Add `[workspace.package]` (`edition`, `rust-version`, `publish = false`, `license`); crates inherit with `.workspace = true`. This replaces the per-crate `publish = false` lines that `deny.toml` (`private = { ignore = true }`, `allow-wildcard-paths`) relies on; `cargo deny --workspace check -D warnings bans licenses sources` stays green.
- (h) Add `rustfmt.toml` matching umpire (`style_edition = "2024"`, `max_width = 100`, `use_small_heuristics = "Max"`, `reorder_imports`, `newline_style = "Unix"`; stable options only). Land the mass reformat as its own commit containing nothing else, and list its hash in `.git-blame-ignore-revs`. Coordinate with open worktrees: anyone still in flight rebases onto a reformatted tree.
- (i) Add a rustdoc gate: `RUSTDOCFLAGS="-D warnings" cargo doc --workspace --no-deps`, in `lint-rust` and in the driver `rust-lint` stage. It is clean on 2026-10-25, so this only locks that in.

Precondition seen 2026-10-25: `make -C tools/gomad lint-rust` was red on head. `-p gomad --all-targets -D warnings` gave ~370 errors, all in test targets (`allocation_receiving`, the three `cycle_*_owner` fixtures, `cfg(test)` modules under `cycle/`); lib and bins were clean. This task's acceptance assumes fn-242 leaves that gate green.
## Acceptance
grep -rn 'wasmtime::' under wasmhost/src/{model,namespace,protocol.rs,transcript.rs,storage_transport.rs,world,volume,network} is empty; classification kind literals exist once as an enum; flip-byte identity test passes and fails when the closure walk is disabled; inventory scanner test covers a synthetic third crate root; make -C tools/gomad lint-rust passes with the two new lints enabled and the allow-list artifact matches the allows in source; cargo test --workspace, test-rust-engine, validate-wasm-determinism green; identity rotation recorded in .flow/artifacts/fn-240/foundations.md.
- `HostError` derives `thiserror::Error`; later extractions follow the same pattern.
- The kind enum covers every message-prefix classification; extraction tasks convert prefix dispatch with their error enums.
- Every workspace package has `[lints] workspace = true`; `lint-rust` and the driver `rust-lint` stage cover every workspace package (a new crate is linted without editing either list).
- `clippy.toml`, `rustfmt.toml` and `[workspace.package]` exist as described in (f)-(h); the reformat commit contains only formatting and is listed in `.git-blame-ignore-revs`.
- The rustdoc gate (i) runs in `lint-rust` and in the driver, and it passes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
