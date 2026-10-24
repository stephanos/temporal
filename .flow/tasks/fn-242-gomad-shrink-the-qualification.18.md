---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.18 Validate command, artifact and preparation custody

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/readers/command/**, tools/gomad/rust/gomad/src/cycle/readers/core_codec/**, tools/gomad/rust/wasmhost/src/linked_storage_inspection.rs, tools/gomad/rust/wasmhost/src/linked_storage_inspection/state.rs, tools/gomad/rust/wasmhost/src/volume/image.rs, tools/gomad/rust/gomad/src/cycle/custody/owner_fixture.rs

## Description
Port receiving common helpers and command/artifact/preparation/result admission, plus core_identity. Admit one bounded authenticated byte buffer per read. Reuse qualify::spec::strict and qualify Result/PublicationReceipt APIs for their actual schemas; retain alias, mode, size and change-during-read checks at every independent open.

Exact implementation/proof ownership: `.flow/artifacts/fn242-port-proof-ownership.json`, task key `reader-command`. This maps every assigned original check, function, endpoint and semantic obligation. Share authoritative validators for the same immutable boundary; independent reopening remains independent. The original aggregate acceptance gates remain in place.

### Quick commands

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle::readers::command::tests:: -- --nocapture
cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check
```

Run from this task worktree, with its assigned exclusive build target. Required genuine-fixture features may be added and recorded. A final test filter selecting zero tests proves nothing.

## Acceptance
- [ ] Mutate PID/argv/source/compiler/build/result/preparation/raw custody and require direct refusal. Export RetainedSession command/artifact/result/preparation methods rather than a generic receipt framework.
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


## Owner-authorized batch B04 (reader-foundation)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.18, fn-242-gomad-shrink-the-qualification.21. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (1 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.

## Shared storage decoder integration

B04 additionally owns the three wasmhost files in Touches solely to expose immutable admitted storage image/state projections through the existing authoritative decoders. Keep recovery/image and state parsing single-owned; preserve current bounds, validation order, raw sections and independently compared effects, selected baselines and engine inputs. Do not expose mutable model internals or add World/network parsing here; those remain B06. Existing recovery_volumes/validate_state callers keep their current behavior. The decoder seam advisory is at .git/flow-notes/fn242-20261024T1800-cycle-wave/b04-decoder-seam.md. No B04 implementation is admitted before actual B01 completion.

## Exclusive-owner receiving fixture

B04 additionally owns the exact existing custody/owner_fixture.rs solely to add its command/RetainedSession family dispatch through the real CycleOwner token. Preserve B01 source/inputs/build arms and B03 bindings arm. This one fixture file is an explicit additive overlap across isolated B03/B04 worktrees: each batch owns only its named arm, and the conductor integrates both. No new Cargo target, direct-rustc bridge or process ownership API.
