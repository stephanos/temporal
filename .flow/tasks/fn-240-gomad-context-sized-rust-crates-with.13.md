---
satisfies: [R9, R10, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.13 Directory relayout, bins/gomad dispatch-only, Makefile --workspace, final measurement

## Description
Move crates under rust/crates/, bins under rust/bins/ (gomad with its three bin targets holding argv parsing and dispatch only; wasmhost; wasm-conformance), Python tooling and publish-executable.sh under rust/tools/, STRESS.md under rust/docs/. Rewrite rust/README.md as one paragraph per crate plus the dependency diagram from Amendment 4. Makefile: --workspace for fmt/clippy/test, bins by package name, driver path updated. Re-key inventory rows for the new paths (never drop one). Record final cargo build --timings clean and incremental against the .1 baseline, lines per crate, public item counts. R11 audit: grep for 'Result<' with ', String>' in pub items across crates/*/src is empty. R12: final allow-list recorded.

Crate amendment (owner, 2026-10-24): `bins/gomad` parses argv with `clap` (derive), replacing positional matching such as `arguments[0] == "--rerun" && arguments[2] == "--root"`. Flags, exit statuses and JSON outputs stay; usage/error text may change and its goldens are updated here. The `gomad -- <command>` wrapper and the go-flag classification in `shim` stay hand-written.
## Acceptance
tree matches Amendment 4; bins/gomad/src contains no verb logic; make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism lint-rust green; clean build wall time within 20% of baseline or an explicit widening decision recorded; final report in .flow/artifacts/fn-240/final.md.
- All verbs parse argv through `clap`; no hand-written verb flag matching remains outside the wrapper/shim go-flag classifier.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
