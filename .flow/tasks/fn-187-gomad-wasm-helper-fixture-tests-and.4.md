---
satisfies: [R4]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.4 Restructure helper modules behind the corpus safety net

## Description
Split model.rs into guest state, reply encoding and host-import routing; move engine.rs request and outcome types into their own module; fold namespace_call, namespace_image and namespace_transport under namespace/; reconcile broker, cluster/ and network/cluster so one module owns instance handoff; move allow(dead_code) onto marked complete-port submodules; replace the conformance feature's main swap with two bin targets if that is all the feature does. Every corpus and retained unit test must pass with zero corpus diffs. Record helper, model and runtime identity rotation and rerun the WASM qualification gates on the frozen candidate. Update ARCHITECTURE.md and the wasm README module map.

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
