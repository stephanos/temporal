---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.27 Validate World, network, boot and cluster wire observations

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/readers/core_lifecycle/**, tools/gomad/rust/gomad/src/cycle/readers/core_world/**

## Description
Port core_world/core_network/core_boot_network/core_cluster_wire. Reuse engine-owned World/network codecs where exposed and retain independent Go-owner corpus comparison and incarnation/endpoint/transition facts.

Exact implementation/proof ownership: `.flow/artifacts/fn242-port-proof-ownership.json`, task key `core-world`. This maps every assigned original check, function, endpoint and semantic obligation. Share authoritative validators for the same immutable boundary; independent reopening remains independent. The original aggregate acceptance gates remain in place.

### Quick commands

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle::readers::core_world::tests:: -- --nocapture
cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check
```

Run from this task worktree, with its assigned exclusive build target. Required genuine-fixture features may be added and recorded. A final test filter selecting zero tests proves nothing.

Throughput amendment (owner, 2026-10-24): validate World, network, boot and cluster wire observations by calling the Rust decoders in `wasmhost`, with independent bound and identity checks around them. Don't re-implement GMWSNAP1/GMWORLD2/GMNETEV1 parsing, so fn-247 changes one decoder.
## Acceptance
- [ ] Execute malformed/trailing/digest/order/endpoint/transition mutants and valid controls. Export World/network observations for workload/live/admission.
- [ ] Resolve every assigned pending_regression using an actually executed normal-return valid control and targeted catching mutation against current Rust code. Port retained keep-row catchers. Record original IDs/source SHA256, current source binding, exact input hashes/bytes, commands, outputs, numeric exits and direct intended predicate attribution.
- [ ] Group checks through shared authoritative validators and fixture tables when input and trust boundary match. Preserve every compound condition term, bound, classification and independent reopening edge. Future test names, test counts, ancestor hash errors and mocked replacements do not resolve proof.
- [ ] Run genuine controlled Rust and approved Go guest/model/tool behavior. Preserve strict JSON/custody, documented current command semantics, all96 stages/145 semantic obligations and3600s stop_at_deadline=False timing-only policy. No Python bridge, placeholder pass, full qualification dispatch or spent attempt.
- [ ] Provide documented current Rust callable interfaces and update actual route consumers together. Retain endpoint/function mappings and fail closed for missing/unmapped routes until implemented.
- [ ] Quick filters select at least one actual test; retain nonzero selected count and numeric exits. Add existing feature flags required by the genuine family fixtures. If work exceeds one roughly100k-token iteration, create a further exact partition before dispatch and extend integration dependencies.
- [ ] Wire parsing reuses the `wasmhost` decoders; no second parser for World or network evidence.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


## Owner-authorized batch B06 (world-lifecycle)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.27, fn-242-gomad-shrink-the-qualification.36. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (1 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.
