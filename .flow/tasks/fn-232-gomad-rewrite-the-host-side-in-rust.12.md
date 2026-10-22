---
satisfies: [R11, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.12 Keep Go overlay, models, generator and corpora green under the Rust binary

**Touches:** tools/gomad/Makefile, tools/gomad/internal/model/generation/protocol, tools/gomad/internal/archcheck/**

## Description
Ensure `make generate`/`make validate` keep emitting Rust codecs for the binary and Go codecs for the oracle; keep the conformance corpora (helper transcript, replay plan, namespace, volume, network, clock) running against the Rust library used by the binary; update archcheck so the Go host packages scheduled for removal are tracked separately from the retained Go reference set.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
`make validate` passes with the Rust binary present; a deliberate codec change fails validation; archcheck distinguishes retained Go sets from host packages slated for removal.

## Done summary
R11/R14 accepted on integrated implementation eaf4806db1 (merge fb0e6eb1e4). Shared generation emits the schema-owned Rust choice encoder and Go oracle codecs; clock/network/World corpora now exercise actual linked library modules. Validation runs every retained volume producer and guards its selection; archcheck separates retained and retiring Go sets without deleting oracle dependencies.

Final integrated CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn232-retained-validation make -C tools/gomad validate exited 0. Raw log: /home/agent/evidence/fn232/retained-validation-20261022/validate-integrated.log. Independently authenticated all 15 final generated outputs against repeated-generation hashes. Deliberate encoder drift made actual validate exit 2; restoration passed. Worker final validation, release build, Go lint and all-role Rust lint exited 0; linked library 170 passed with two pre-existing diagnostic/measurement ignores. See .flow/artifacts/fn-232.12-retained-validation.md for commands and evidence.

Independent retained_review found omitted Go volume producers; fixed and re-reviewed SHIP, final numeric gates verified. No qualification or Go retirement claimed; those remain .13.

stage: impl-review - ran (model: gpt-6.1-sol at high; same family as writer)
stage: plan-sync - skipped(config: batch ownership and task/R-ID mapping retained in existing Flow records)
## Evidence
- Commits: eaf4806db1, fb0e6eb1e44624dda73016f8c5e99a61578c7944, a4114ebb16
- Tests: CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn232-retained-validation make -C tools/gomad validate (integrated exit 0), make -C tools/gomad/internal/backends/wasm lint-rust (all roles exit 0), make lint-code-fast GOLANGCI_LINT_BASE_REV=394772ef7f (exit 0), all 15 final generated output SHA256 values authenticated; two generations byte-identical, deliberate codec drift: actual make validate exit 2; restored validate exit 0
- PRs: