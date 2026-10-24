---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.16 Authenticate candidate, compiler, GNU and workload inputs

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/preparation/inputs/**

## Description
Port preparation.Inventory, source_custody, native_gnu and workloads as one input authority. Capture fresh source/compiler/tool inventories, GNU closure and exact original workload specification bytes. Separate fresh construction from independently reopened preparation verification. Reuse installed campaign/guest preparation and digest/identity helpers.

Exact implementation/proof ownership: `.flow/artifacts/fn242-port-proof-ownership.json`, task key `prep-inputs`. This maps every assigned original check, function, endpoint and semantic obligation. Share authoritative validators for the same immutable boundary; independent reopening remains independent. The original aggregate acceptance gates remain in place.

### Quick commands

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle::preparation::inputs::tests:: -- --nocapture
cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check
```

Run from this task worktree, with its assigned exclusive build target. Required genuine-fixture features may be added and recorded. A final test filter selecting zero tests proves nothing.

## Acceptance
- [ ] Genuine bounded guest/tool preparation succeeds and stale compiler/source/tool, changed closure, invalid GNU aliases and replaced input bytes refuse. Export PreparedInputs and verify_preparation with exact original provenance/role/bounds.
- [ ] Resolve every assigned pending_regression using an actually executed normal-return valid control and targeted catching mutation against current Rust code. Port retained keep-row catchers. Record original IDs/source SHA256, current source binding, exact input hashes/bytes, commands, outputs, numeric exits and direct intended predicate attribution.
- [ ] Group checks through shared authoritative validators and fixture tables when input and trust boundary match. Preserve every compound condition term, bound, classification and independent reopening edge. Future test names, test counts, ancestor hash errors and mocked replacements do not resolve proof.
- [ ] Run genuine controlled Rust and approved Go guest/model/tool behavior. Preserve strict JSON/custody, documented current command semantics, all96 stages/145 semantic obligations and3600s stop_at_deadline=False timing-only policy. No Python bridge, placeholder pass, full qualification dispatch or spent attempt.
- [ ] Provide documented current Rust callable interfaces and update actual route consumers together. Retain endpoint/function mappings and fail closed for missing/unmapped routes until implemented.
- [ ] Quick filters select at least one actual test; retain nonzero selected count and numeric exits. Add existing feature flags required by the genuine family fixtures. If work exceeds one roughly100k-token iteration, create a further exact partition before dispatch and extend integration dependencies.

## Done summary
Implemented B01 source/configuration/build custody and immutable preparation inputs as one reviewed batch. This member owns genuine stock/cooperative/go-test preparation inventories, GNU closure and workload input validation. Exact member references remain in the original compact batch map; counts {"check_refs": 169, "function_refs": 48, "endpoint_refs": 4}. Final source fix28c02f4fb118 integrated03832e4d5d; final full tracked Rust tree equals tested worker source. Current gates pass:67 cycle library tests, source-owner cases, genuine preparation/GNU controls, actual two-host-role overwrite/admission controls, scoped Clippy and formatting. Both review findings have direct red-to-green proofs. All1411 new recorded PIDs independently absent; original2797 were separately audited. Same-session re-review SHIP at /tmp/impl-review-receipt-67d1eda7ff47-fn-242-gomad-shrink-the-qualification.14.json. Proof .flow/artifacts/fn242-custody-inputs-review-fix-proof.json plus external compact delta/handoff. This closes the library/family boundary only; actual global CLI startup/routing remains B15. No full96/final145/global qualification attempt; operational journal binding remains absent and fn232.33 remains a receiving prerequisite. Legacy compatibility policy applies.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e79ca50771, 03832e4d5d, d53f9e5e0b
- Tests: cargo +1.99.1 --config profile.release.package.gomad.opt-level=0 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle:: -- --nocapture --test-threads=1, CUSTODY_FAMILY=source/inputs/build cycle_custody_owner: exact separate commands and numeric exits in review-fix/proof-map.json, cargo +1.99.1 --config profile.release.package.gomad.opt-level=0 clippy --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib --no-deps -- -D warnings, cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check, git diff --check
- PRs:
## Owner-authorized batch B01

Implementation and independent review run under fn-242-gomad-shrink-the-qualification.14 with the complete batch scope in `.flow/artifacts/fn242-execution-batches.json`. This original task retains all acceptance and proof ownership. Its leader dependency prevents separate worker admission. After whole-batch SHIP, start and complete this receipt-only member in original topological order using the actual reviewed source, shared review receipt and its own executed proof.
