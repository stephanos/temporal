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
Implemented and independently reviewed whole B04 at `d2a0b9a85cc0b3f1ce137b68fad417f72ccb90ad` (worker source `587d126e6d7c3d9e031bdb36f9893270e1241b01`); all tracked Rust bytes equal the tested worker tree. One fresh gpt-6.1-sol HIGH correctness review returned SHIP with no findings, finalized rid c2cc32a517b94829b20f112e0e295d9c. Writer and reviewer use the same model family with separate fresh contexts. Retained commands, numeric exits, raw log hashes, source/dependency bindings and actual process cleanup were independently checked; fresh reviewer Cargo execution was blocked by its required read-only sandbox and no new test execution is claimed. Owner-authorized exact-source evidence reuse applies.

Retained Store, enumeration, effects, recovery state and requests use existing authoritative wasmhost codecs with independent receiving comparisons. 190 original core checks:158 current guarantees proved and32 exact compatibility-only terms explicitly retired by owner amendment. Partial durability correctly accepts dependency-closed durable bytes over capacity; inode, capacity metadata, checked arithmetic and namespace bounds remain enforced. Existing decoder acceptance is unchanged. Effect-order/coverage guard-removal mutant was caught.

Shared proof: `.flow/artifacts/fn242-b04-proof.json`; external `/home/agent/wt/evidence/fn242-b04-reader-foundation/proof-map.json` SHA f20544d998e84231ea78bd2abde8293fd5f2b0974a5d78dc9bb9d5329c336214. All315 original check IDs have dispositions, pending0. Source-freeze, handoff, exact gate logs and retained reviewer transcript are adjacent. Strict GomadClippy/fmt/diff0; focused wasmhostClippy0 permits only inherited chunks_exact_to_as_chunks (fn240R12). No final qualification, timing/resource, global-cycle or public18/R4 claim. Tier: session (jev-unavailable(no_key)); implementer project pin gpt-6.1-sol medium.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d2a0b9a85cc0b3f1ce137b68fad417f72ccb90ad, 587d126e6d7c3d9e031bdb36f9893270e1241b01
- Tests: B04 shared owner21 actual stockGo success/panic canary/cooperative2nodecluster exit0, 42 actual selected reader/Store/Result tests exit0, Targeted command recomputation1test exit0, Two representative source mutants intended2fail/10pass exit101, byteexact restored, Strict Gomad Clippy, scoped wasmhostClippy, fmt and diff checks exit0, Exact-source/dependency and all retained logSHA verification exit0, Independent gpt-6.1-sol HIGH correctness review SHIP rid c2cc32a517b94829b20f112e0e295d9c
- PRs:
## Owner-authorized batch B04

Implementation and independent review run under fn-242-gomad-shrink-the-qualification.18 with the complete batch scope in `.flow/artifacts/fn242-execution-batches.json`. This original task retains all acceptance and proof ownership. Its leader dependency prevents separate worker admission. After whole-batch SHIP, start and complete this receipt-only member in original topological order using the actual reviewed source, shared review receipt and its own executed proof.

## Shared storage decoder integration

B04 additionally owns the three wasmhost files in Touches solely to expose immutable admitted storage image/state projections through the existing authoritative decoders. Keep recovery/image and state parsing single-owned; preserve current bounds, validation order, raw sections and independently compared effects, selected baselines and engine inputs. Do not expose mutable model internals or add World/network parsing here; those remain B06. Existing recovery_volumes/validate_state callers keep their current behavior. The decoder seam advisory is at .git/flow-notes/fn242-20261024T1800-cycle-wave/b04-decoder-seam.md. No B04 implementation is admitted before actual B01 completion.

## Recovery capacity contract correction

For original compound row reader:core_codec:recovery_image:430:8, the historical Python sum(durable object bytes)<=volume capacity guard is erroneous: the authoritative volume/image.rs validate_tree contract admits dependency-closed partial durable states above volume capacity, while volatile growth and recovered-namespace resource bounds remain enforced. Preserve that current behavior rather than add a new receiver refusal. Keep the original row retained for inode<next_inode and all surviving integrity/bounds terms; identify the corrected byte-capacity term explicitly and prove an actually accepted partial-durability over-capacity control through the real decoder/receiver, plus retained-boundary catchers. This is a receiving correctness correction, not a legacy byte-compatibility waiver or an automatic caught/retired row.
