---
satisfies: [R4]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.4 Restructure helper modules behind the corpus safety net

## Description
Split model.rs into guest state, reply encoding and host-import routing; move engine.rs request and outcome types into their own module; fold namespace_call, namespace_image and namespace_transport under namespace/; reconcile broker, cluster/ and network/cluster so one module owns instance handoff; move allow(dead_code) onto marked complete-port submodules; replace the conformance feature's main swap with two bin targets if that is all the feature does. Every corpus and retained unit test must pass with zero corpus diffs. Record helper, model and runtime identity rotation and rerun the WASM qualification gates on the frozen candidate. Update ARCHITECTURE.md and the wasm README module map.

Read-only preparation (2026-10-18; not implementation or acceptance): existing `protocol.rs` already owns Execute/Outcome and related wire types. Consolidate only engine-local Continue and transcript Chunk into that owner, preserving Chunk's serde diagnostic name; keep engine execution State/Limits/Snapshot there. The conformance feature changes module/dependency availability and behavior beyond swapping main, so the conditional two-bin replacement is not applicable on current evidence. Split model internals behind the existing facade without widening outer APIs, preserving admission-before-mutation, remote validation before clock adoption, pending-clock hashing and diagnostic/serde contracts. Broker/cluster ownership must preserve automatic result-before-finish versus manual finish-before-result and the eager-network commitment barrier. Narrow dead-code exceptions to genuinely complete-port modules/items; World is not a complete Go port because plans/recordings remain unported. These findings do not waive R3's safety-net dependency, zero corpus diffs, frozen-candidate identity rotation or qualification requirements. Performance/resource-sensitive qualification stays parked under the owner's current scope.

**Size:** L
**Files:** wasmhost/src/**, wasmhost/Cargo.toml, wasmhost/src/main.rs, tools/gomad/ARCHITECTURE.md, tools/gomad/internal/backends/wasm/README.md
## Acceptance
- [ ] Module map matches the described ownership; no module-wide allow(dead_code)
- [ ] Zero corpus diffs; retained unit tests pass
- [ ] Identity rotation recorded and WASM gates rerun on the frozen candidate
- [ ] Docs updated

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
