---
satisfies: [R6, R7, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.5 Pre-cut foundations: HostError, identity closure, inventory scanner, workspace lints

## Description
Precondition for every extraction (Amendment 4). (a) Add HostError in wasmhost and replace the 28 wasmtime::{Error, Result} sites below engine (model, namespace, protocol, transcript, storage_transport, world, volume, network); engine converts at its boundary. HostError is the first R11 error enum: it carries the record classification kinds as an enum with as_str, defined once. (b) Extend gomad/build.rs and wasmhost/build.rs so each implementation SHA hashes every workspace crate the consumer depends on (path-dependency walk over cargo metadata), plus measurement-source-inputs.tsv closure; add a test that flips one byte in a dependency crate and asserts the identity changes. (c) Make tests/inventory/host.rs walk every workspace crate root. (d) Move clippy lints and pinned versions to [workspace.lints]/[workspace.dependencies]; add clippy::too_many_lines and clippy::cognitive_complexity at warn, put explicit #[allow] on every current offender and record the list in .flow/artifacts/fn-240/lint-allows.md (R12). No module moves in this task. Ordering: after fn-242 (house convention: cross-spec order recorded here, spec-level dep already present); before .2 and .3.

Crate amendment (owner, 2026-10-24): `HostError` and every later per-crate error enum (R11) derive with `thiserror` (pinned `=`). Message text is not pinned; goldens and CLI tests that pin message text are updated in the same task, exit statuses stay.

Throughput amendment (owner, 2026-10-24): the classification-kind enum R11 introduces also replaces message-prefix dispatch (`runner_failure:`, `refused:`, `capacity:`). Each later extraction converts that crate's prefix dispatch in the same pass as its `Result<_, String>` signatures, so fn-245.3 only sweeps leftovers.
## Acceptance
grep -rn 'wasmtime::' under wasmhost/src/{model,namespace,protocol.rs,transcript.rs,storage_transport.rs,world,volume,network} is empty; classification kind literals exist once as an enum; flip-byte identity test passes and fails when the closure walk is disabled; inventory scanner test covers a synthetic third crate root; make -C tools/gomad lint-rust passes with the two new lints enabled and the allow-list artifact matches the allows in source; cargo test --workspace, test-rust-engine, validate-wasm-determinism green; identity rotation recorded in .flow/artifacts/fn-240/foundations.md.
- `HostError` derives `thiserror::Error`; later extractions follow the same pattern.
- The kind enum covers every message-prefix classification; extraction tasks convert prefix dispatch with their error enums.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
