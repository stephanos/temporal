---
satisfies: [R1]
---
# fn-249-gomad-retire-the-go-reference-models.1 Inventory Go model consumers and conformance tests

**Touches:** .flow/artifacts/fn-249/**

## Description
List every `internal/model` package with its production and test consumers, and every Go↔Rust conformance/differential test (`internal/backends/wasm/cluster/*_conformance_test.go`, `internal/model/wasi`, `volume`, `world`, `network`, `clock`, `choice` tests, `wasmhost` `*_conformance.rs`, `network_corpus.rs`, `tests/helper_corpus.rs`) with the failure it catches. Mark each kept-Go, convert-to-Rust-corpus or obsolete with a reason. List cycle stages and leaf commands touching them (`cycle/plan/leaf/static_commands.rs`, `declarations.rs`). Record in `.flow/artifacts/fn-249/inventory.md`.

## Acceptance
- [ ] Every package and conformance test has a disposition and named failure; every affected cycle stage is listed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
