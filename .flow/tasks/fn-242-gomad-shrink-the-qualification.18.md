---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.18 Validate command, artifact and preparation custody

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/readers/command/**, tools/gomad/rust/gomad/src/cycle/readers/core_codec/**, tools/gomad/rust/wasmhost/src/linked_storage_inspection.rs, tools/gomad/rust/wasmhost/src/linked_storage_inspection/state.rs, tools/gomad/rust/wasmhost/src/volume/image.rs, tools/gomad/rust/gomad/src/cycle/custody/owner_fixture.rs, tools/gomad/rust/gomad/src/cycle/build/identity.rs, tools/gomad/rust/gomad/src/cycle/build/mod.rs

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
Implemented and independently reviewed whole B04 at `d2a0b9a85cc0b3f1ce137b68fad417f72ccb90ad` (worker source `587d126e6d7c3d9e031bdb36f9893270e1241b01`); all tracked Rust bytes equal the tested worker tree. One fresh gpt-6.1-sol HIGH correctness review returned SHIP with no findings, finalized rid c2cc32a517b94829b20f112e0e295d9c. Writer and reviewer use the same model family with separate fresh contexts. Retained commands, numeric exits, raw log hashes, source/dependency bindings and actual process cleanup were independently checked; fresh reviewer Cargo execution was blocked by its required read-only sandbox and no new test execution is claimed. Owner-authorized exact-source evidence reuse applies.

Command custody independently reopens actual process/argv/raw streams, immutable source/build/role authorities, preparation inventory, Results and artifact descriptors. 125 original command checks and 19 function mappings resolved using actual owner21 standalone success, panic canary, cooperative two-node cluster, focused receiving tests and existing Store/Result owners. Final shared42test gate and targeted recomputation1test exited0; command stream guard-removal mutant was caught.

Shared proof: `.flow/artifacts/fn242-b04-proof.json`; external `/home/agent/wt/evidence/fn242-b04-reader-foundation/proof-map.json` SHA f20544d998e84231ea78bd2abde8293fd5f2b0974a5d78dc9bb9d5329c336214. All315 original check IDs have dispositions, pending0. Source-freeze, handoff, exact gate logs and retained reviewer transcript are adjacent. Strict GomadClippy/fmt/diff0; focused wasmhostClippy0 permits only inherited chunks_exact_to_as_chunks (fn240R12). No final qualification, timing/resource, global-cycle or public18/R4 claim. Tier: session (jev-unavailable(no_key)); implementer project pin gpt-6.1-sol medium.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d2a0b9a85cc0b3f1ce137b68fad417f72ccb90ad, 587d126e6d7c3d9e031bdb36f9893270e1241b01
- Tests: B04 shared owner21 actual stockGo success/panic canary/cooperative2nodecluster exit0, 42 actual selected reader/Store/Result tests exit0, Targeted command recomputation1test exit0, Two representative source mutants intended2fail/10pass exit101, byteexact restored, Strict Gomad Clippy, scoped wasmhostClippy, fmt and diff checks exit0, Exact-source/dependency and all retained logSHA verification exit0, Independent gpt-6.1-sol HIGH correctness review SHIP rid c2cc32a517b94829b20f112e0e295d9c
- PRs:
## Owner-authorized batch B04 (reader-foundation)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.18, fn-242-gomad-shrink-the-qualification.21. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (1 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.

## Shared storage decoder integration

B04 additionally owns the three wasmhost files in Touches solely to expose immutable admitted storage image/state projections through the existing authoritative decoders. Keep recovery/image and state parsing single-owned; preserve current bounds, validation order, raw sections and independently compared effects, selected baselines and engine inputs. Do not expose mutable model internals or add World/network parsing here; those remain B06. Existing recovery_volumes/validate_state callers keep their current behavior. The decoder seam advisory is at .git/flow-notes/fn242-20261024T1800-cycle-wave/b04-decoder-seam.md. No B04 implementation is admitted before actual B01 completion.

## Exclusive-owner receiving fixture

B04 additionally owns the exact existing custody/owner_fixture.rs solely to add its command/RetainedSession family dispatch through the real CycleOwner token. Preserve B01 source/inputs/build arms and B03 bindings arm. This one fixture file is an explicit additive overlap across isolated B03/B04 worktrees: each batch owns only its named arm, and the conductor integrates both. No new Cargo target, direct-rustc bridge or process ownership API.

## Shared helper identity visibility

B04 additionally owns exactly build/identity.rs to make helper_identity crate-visible and build/mod.rs to re-export it crate-visibly. Reuse this existing authoritative helper identity implementation from the command receiver and remove the duplicate include. No source list, algorithm, schema, identity behavior or unrelated build code changes; no duplicate-mod lint exemption.
