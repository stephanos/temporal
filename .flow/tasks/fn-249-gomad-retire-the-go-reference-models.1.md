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
Inventoried all 25 internal/model Go package directories, production/test/transitive consumers, retained Go guests/generator roots, and affected cycle stages/leaf commands. Coverage includes 286 test/support sources, 966 entrypoints, 3,065 JSON fixture paths and all 96 stage declarations. Every package/test has a disposition and named source assertion contract. Five review findings were reproduced and corrected: Rust coverage omissions, source-revision recording, truncated raw-string context, incorrect corpus-header retirement and the retained output-capacity Go guest. Independent same-session re-review SHIP, R1 met; receipt /tmp/impl-review-receipt-a0de947018eb-fn-249-gomad-retire-the-go-reference-models.1.json. Static discovery only: no model deletion, oracle capture, mutation qualification, builds or full cycle. Implementation .2 onward remains blocked until fn242.8.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e6725b34b3d5b8e5eeaf759e1666d3bcd965fd8f, 3fed756bb1
- Tests: PYTHONDONTWRITEBYTECODE=1 python3 /home/agent/wt/evidence/fn249-model-inventory/review-fix/regressions.py, python3 .flow/artifacts/fn-249/collect_inventory.py --check, python3 .flow/artifacts/fn-249/verify_inventory.py, git diff --check
- PRs: