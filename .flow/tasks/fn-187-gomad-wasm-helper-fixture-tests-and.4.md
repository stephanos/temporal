---
satisfies: [R4]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.4 Restructure helper modules behind the corpus safety net

## Description
Split model.rs into guest state, reply encoding and host-import routing; move
engine.rs request and outcome types into their own module; fold namespace_call,
namespace_image and namespace_transport under namespace/; reconcile broker,
cluster/ and network/cluster so one module owns instance handoff; move
allow(dead_code) onto marked complete-port submodules; replace the conformance
feature's main swap with two bin targets if that is all the feature does.
Every corpus and retained unit test must pass with zero corpus diffs. Record
helper, model and runtime identity rotation and rerun the WASM qualification
gates on the frozen candidate. Update ARCHITECTURE.md and the wasm README map.

Existing source preparation is not implementation or acceptance: protocol.rs
already owns Execute/Outcome and related wire types. Consolidate only engine-local
Continue and transcript Chunk there, preserving Chunk's serde diagnostic name;
keep engine execution State/Limits/Snapshot in engine. The conformance feature
changes module/dependency availability and behavior beyond swapping main, so the
conditional two-bin replacement is not applicable on current evidence. Split
model internals behind its facade without widening outer APIs, preserving
admission-before-mutation, remote validation before clock adoption, pending-clock
hashing and diagnostic/serde contracts. Preserve automatic result-before-finish
versus manual finish-before-result and the eager-network commitment barrier.
Narrow dead-code exceptions to genuinely complete-port modules/items; World is
not a complete Go port because plans/recordings remain unported.

The owner authorized WASM resource-sensitive checks, superseding the old hold.
R3 remains the required safety-net dependency; do not start R4 source changes
while R3's immutable coverage phases are running. All compiler/runtime/LLVM
execution stays serialized under root ownership. No native qualification,
fn185 profiling, ignored benchmarks, push, PR or CI action is authorized.
The approved namespace-readiness fixture and its recurring predicate selector
must remain covered in the final combined frozen candidate. Source preparation
does not waive zero corpus diffs, identity rotation, full qualification or review.

**Size:** L
Current implementation is integrated in primary at owner HEAD5951d8bb96f41fdd1b146ee4a8cbb17613495aa0
plus the uncommitted R1-R4 batch. Source-only worker handover is
.flow/tmp/fn187-r4-handover.md; guarded integration/preimages are under
/home/agent/wt/evidence/fn187/r5-current/integrated-batch-2026-10-19/.
All3026 accepted testdata inputs are unchanged. Both indexes remain untouched.
Primary6-mode Rustfmt/strict Clippy passes (session9541 exit0), and integrated
make -C tools/gomad validate passes (53809 exit0), including template/output,
compatibility pack and qualification manifest checks. No new blanket exceptions.
Retained Rust matrix is terminal session59182 exit0, logs under
/home/agent/wt/evidence/fn187/r5-current/r4-integrated-rust-tests-2026-10-19/.
All six modes pass:201 default,55 conformance unit,204 diagnostic,205 fuel-off,
206 guest-profile and207 combined tests. Only existing2/3 benchmark roots remain
ignored. Inputs are frozen and checked before/after each arm, then checked again
against current primary after terminal success. Task stays in progress:
Go/full host/WASM qualification, identity recording and final independent review
remain required. Old R3 evidence is historical-source-bound, never relabeled.

**Files:** wasmhost/src/**, wasmhost/Cargo.toml, wasmhost/src/main.rs,
tools/gomad/ARCHITECTURE.md, tools/gomad/internal/backends/wasm/README.md
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
