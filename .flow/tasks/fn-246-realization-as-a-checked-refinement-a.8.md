---
satisfies: [R1, R2]
---
# fn-246-realization-as-a-checked-refinement-a.8 Default whole-prefix guest protocol and current Host admission

## Description

Establish one current default whole-prefix proof contract across generic guest exports and Host admission for R1/R2. Replace the former backward-compatibility-only custom-section/dual-version task while retaining this task ID and dependencies. Wire/protobuf/export formats may change and all current producers/consumers migrate together.

**Size:** M
**Files:** ordinary Model protocol/Reset response, generic guest export/runtime, Go Host admission and generated bindings, focused current-contract fixtures and protocol/build documentation.
**Touches:** [model/umpire/protocol/**, model/umpire/hosts/go/{host,gen}/**, model/umpire/reference/kettle/**, model/temporal/guest/**, model/dev/xtask/**, model/GUEST_BUILDS.md]

## Approach

1. Reanchor prerequisite source/build owners. Make whole-observation-prefix witnesses the ordinary current framework contract for all current guests, including Exact/Kettle. Complete-state matching remains supported; old delta presentation and old readers are not maintained.
2. Declare the single current proof-contract revision in the ordinary recorded Reset response, using a required admission value in the new protocol. The framework guest produces it from its real runtime contract; Host validates the actual response before Driver effects. Update native/wasm exports, Host and generated bindings together. No separate version export, hidden probe, opt-in mode, Wasm custom-section carrier, unmarked fallback or fat-LTO marker-retention obligation remains.
3. Fail explicitly for malformed or incompatible current-contract responses; never silently reinterpret an old delta guest as a whole-prefix guest. Historical guest/Run formats need no reader or migration. Current model_hash still binds exact loaded guest bytes, and production source/guest freshness remains required.
4. Demonstrate current-contract Reset and ordinary calls through the production guest builder and generic Host. Coordinate with .9's Run recording and .10's frontier integration so advertised whole-prefix semantics match actual runtime behavior. Document the format break and regenerate current fixtures/goldens; arbitrary old-byte equivalence is not a gate.

## Investigation targets

**Required:**

- Current ordinary Reset/Explanation protocol, runtime `abi.rs`/`lib.rs` and generic guest fixtures.
- Existing Host.New/call/admission and generic protocol generation owners.
- Current Kettle and Temporal guest exports and native/wasm parity controls.
- Final fn-247 production guest builder/xtask and freshness contract.
- .9 Run/replay integration and .10 actual whole-prefix runtime contract.

## Verification

Run focused native/runtime, production guest and generic Go-host tests with test_dep where Go is used. Check actual Reset contract identity, malformed/incompatible responses, current guest hash/freshness and native/wasm/Host agreement. Review regenerated current artifacts. No legacy reader/marker retention controls are required.

## Acceptance

- [ ] All current generic, Kettle and Temporal guest/Host consumers agree on one default whole-prefix contract through the actual recorded Reset response.
- [ ] Malformed/incompatible contract identity fails admission; caller intent cannot substitute for the actual guest response. Current model_hash and source/guest freshness retain their integrity role.
- [ ] Current protocol/export/bindings and goldens regenerate coherently; the format break is documented. Old readers, records, delta mode, custom metadata and unchanged byte layouts are not required.
- [ ] Production native/wasm/Host controls establish the current ordinary protocol path before dependent integration; .10 still supplies the actual bounded-prefix proof and publication correctness.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
