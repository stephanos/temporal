---
satisfies: [R1, R2]
---
# fn-246-realization-as-a-checked-refinement-a.8 Default whole-prefix guest protocol and current Host admission

## Description

Establish one current default whole-prefix proof contract across generic guest exports and Host admission for R1/R2. Replace the former backward-compatibility-only custom-section/dual-version task while retaining this task ID and dependencies. Wire/protobuf/export formats may change and all current producers/consumers migrate together.

**Size:** M
**Files:** ordinary Model protocol/Reset response, generic guest export/runtime, Go Host admission and generated bindings, focused current-contract fixtures and protocol/build documentation.
**Touches:** [model/umpire/protocol/**, model/umpire/hosts/go/host/**, model/umpire/hosts/go/gen/**, model/umpire/hosts/go/runner/**, model/umpire/reference/kettle/**, model/temporal/realization/src/live.rs, model/temporal/realization/tests/program.rs, model/temporal/hosts/go/**, model/temporal/guest/**, model/dev/xtask/**, model/GUEST_BUILDS.md, model/scripts/tests/build-guests.sh]

## Approach

1. Reanchor prerequisite source/build owners and verify .1/.7's actual bounded singleton prefix retention/publication before advertising the contract. Make whole-observation-prefix witnesses the ordinary current framework contract for all current guests, including Exact/Kettle. Complete-state matching remains supported; old delta presentation and old readers are not maintained.
2. Declare the single current proof-contract revision in the ordinary recorded Reset response, using a required admission value in the new protocol. The framework guest produces it from its real runtime contract; Host validates the actual response before Driver effects. Update native/wasm exports, Host and generated bindings together. No separate version export, hidden probe, opt-in mode, Wasm custom-section carrier, unmarked fallback or fat-LTO marker-retention obligation remains.
3. Preserve ownership of the loaded guest bytes so caller mutation cannot separate the loaded module from its recorded model_hash. Fail explicitly for malformed or incompatible current-contract responses; never silently reinterpret an old delta guest as a whole-prefix guest. Historical guest/Run formats need no reader or migration. Current model_hash still binds exact loaded guest bytes, and production source/guest freshness remains required.
4. Demonstrate current-contract Reset and ordinary calls through the production guest builder and generic Host. Use .1/.7's independently working runtime for ordinary multi-observation controls, including Current after a nonempty witness and callback delivery; actual responses must already be whole-prefix before this task closes. .9 records this supported singleton behavior. .10 later extends compatible alternatives and changed-canonical-past selection; those controls do not gate .8 and cannot excuse delta responses here. Document the format break and regenerate current fixtures/goldens; arbitrary old-byte equivalence is not a gate.

## Investigation targets

**Required:**

- Current ordinary Reset/Explanation protocol, runtime `abi.rs`/`lib.rs` and generic guest fixtures.
- Existing Host.New/call/admission and generic protocol generation owners.
- Current Kettle and Temporal guest exports and native/wasm parity controls.
- Final fn-247 production guest builder/xtask and freshness contract.
- .1/.7's singleton whole-prefix runtime, .9 Run/replay integration and .10 compatible-frontier extension.

## Verification

2026-10-25 joined-gate amendment: the first full gate found the program test still asserting an empty unconfigured Reset. Preserve the no-program assertion separately from the required proof-contract identity, and strengthen the configured assertion so the contract field alone cannot satisfy program issuance. Reopen .8, run the cohesive Temporal realization crate suite, renew review, then rerun the required joined gate. The failing gate and earlier SHIP/closure remain historical evidence.

The existing `model/scripts/tests/build-guests.sh` owner wires the native current-contract transcript alongside existing native proof assets, so ordinary model validation exercises production native/wasm parity. Reuse its current builder rather than introducing another harness.

Run focused native/runtime, production guest and generic Go-host tests with test_dep where Go is used. Check actual Reset contract identity, malformed/incompatible responses, current guest hash/freshness and native/wasm/Host agreement. Review regenerated current artifacts. No legacy reader/marker retention controls are required.

## Acceptance

- [ ] All current generic, Kettle and Temporal guest/Host consumers agree on one default whole-prefix contract through the actual recorded Reset response.
- [ ] Malformed/incompatible contract identity fails admission; caller intent cannot substitute for the actual guest response. Current model_hash and source/guest freshness retain their integrity role.
- [ ] Current protocol/export/bindings and goldens regenerate coherently; the format break is documented. Old readers, records, delta mode, custom metadata and unchanged byte layouts are not required.
- [ ] Production native/wasm/Host controls establish the actual singleton whole-prefix runtime through multi-observation, Current-after-nonempty-history and callback-delivery cases before dependent integration; .10 still supplies the compatible-frontier proof and changed-canonical-past publication correctness.

## Done summary
Implemented the current WHOLE_PREFIX_V1 Reset contract across generic, Kettle and Temporal producers/consumers. Host validates the actual response before Driver creation; Runner owns the hashed guest bytes. Production native/Wasm parity covers 25 calls across ordinary/configured/callback streams, including complete witness steps/outcomes and Current after history. Bindings, wire fixtures, documentation and the ordinary guest builder migrate together.

Worker scoped runtime/realization/three-module Go tests, strict clippy/fmt, protobuf and guest freshness passed. Joined registry/contract checks renewed Host admission/parity and guest freshness; source/log manifests were verified. Final joined guest publication is committed at f85d9399c4. The first formal three-axis review finalized SHIP at that exact commit with no findings; reviewer fresh Go builds were blocked by its read-only sandbox and committed passing evidence was inspected.

The joined TLC-required full gate is running and is not claimed complete here; it remains mandatory for task .15 closure. Run/replay expansion (.9), compatible alternatives (.10), protocol/refinement and live gates retain their separate owners.

Tier: AGENTS implementer gpt-6.1-sol/medium; reviewer gpt-6.1-sol/high (same family).
stage: impl-review - SHIP (three-axis fanout, rid 1d1cabe246be4ad6930f3ae59ab0fa90).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
