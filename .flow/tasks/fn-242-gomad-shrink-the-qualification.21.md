---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.21 Validate retained storage and engine codecs

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/readers/core_codec/**, tools/gomad/rust/wasmhost/src/linked_storage_inspection.rs, tools/gomad/rust/wasmhost/src/linked_storage_inspection/state.rs, tools/gomad/rust/wasmhost/src/volume/image.rs

## Description
Port core_codec storage/state/engine observations through existing authoritative Rust codecs where contracts match. Keep independent framing, counts, ordering, padding, numeric range, state identity and raw evidence comparisons.

Exact implementation/proof ownership: `.flow/artifacts/fn242-port-proof-ownership.json`, task key `core-codec`. This maps every assigned original check, function, endpoint and semantic obligation. Share authoritative validators for the same immutable boundary; independent reopening remains independent. The original aggregate acceptance gates remain in place.

### Quick commands

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle::readers::core_codec::tests:: -- --nocapture
cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check
```

Run from this task worktree, with its assigned exclusive build target. Required genuine-fixture features may be added and recorded. A final test filter selecting zero tests proves nothing.

## Acceptance
- [ ] Catch malformed/trailing/oversized data, changed storage effects, reordered observations, foreign selected baseline and wrong engine input. Export compact admitted observation/state types.
- [ ] Resolve every assigned pending_regression using an actually executed normal-return valid control and targeted catching mutation against current Rust code. Port retained keep-row catchers. Record original IDs/source SHA256, current source binding, exact input hashes/bytes, commands, outputs, numeric exits and direct intended predicate attribution.
- [ ] Group checks through shared authoritative validators and fixture tables when input and trust boundary match. Preserve every compound condition term, bound, classification and independent reopening edge. Future test names, test counts, ancestor hash errors and mocked replacements do not resolve proof.
- [ ] Run genuine controlled Rust and approved Go guest/model/tool behavior. Preserve strict JSON/custody, documented current command semantics, all96 stages/145 semantic obligations and3600s stop_at_deadline=False timing-only policy. No Python bridge, placeholder pass, full qualification dispatch or spent attempt.
- [ ] Provide documented current Rust callable interfaces and update actual route consumers together. Retain endpoint/function mappings and fail closed for missing/unmapped routes until implemented.
- [ ] Quick filters select at least one actual test; retain nonzero selected count and numeric exits. Add existing feature flags required by the genuine family fixtures. If work exceeds one roughly100k-token iteration, create a further exact partition before dispatch and extend integration dependencies.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


## Owner-authorized batch B04

Implementation and independent review run under fn-242-gomad-shrink-the-qualification.18 with the complete batch scope in `.flow/artifacts/fn242-execution-batches.json`. This original task retains all acceptance and proof ownership. Its leader dependency prevents separate worker admission. After whole-batch SHIP, start and complete this receipt-only member in original topological order using the actual reviewed source, shared review receipt and its own executed proof.

## Shared storage decoder integration

B04 additionally owns the three wasmhost files in Touches solely to expose immutable admitted storage image/state projections through the existing authoritative decoders. Keep recovery/image and state parsing single-owned; preserve current bounds, validation order, raw sections and independently compared effects, selected baselines and engine inputs. Do not expose mutable model internals or add World/network parsing here; those remain B06. Existing recovery_volumes/validate_state callers keep their current behavior. The decoder seam advisory is at .git/flow-notes/fn242-20261024T1800-cycle-wave/b04-decoder-seam.md. No B04 implementation is admitted before actual B01 completion.
