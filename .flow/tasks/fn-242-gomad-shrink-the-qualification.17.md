---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.17 Freeze authenticated role and configuration bindings

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/preparation/bindings/**, tools/gomad/rust/gomad/src/cycle/preparation/composition/**, tools/gomad/rust/gomad/src/cycle/plan/leaf/mod.rs

## Description
Port configure cell/native/live/SQLite/runtime input helpers and freeze. Authenticate roles, public bindings, installations and source/input seals. Consume endpoint descriptors from foundation without requiring endpoint implementations to exist yet.

Exact implementation/proof ownership: `.flow/artifacts/fn242-port-proof-ownership.json`, task key `prep-bindings`. This maps every assigned original check, function, endpoint and semantic obligation. Share authoritative validators for the same immutable boundary; independent reopening remains independent. The original aggregate acceptance gates remain in place.

### Quick commands

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle::preparation::bindings::tests:: -- --nocapture
cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check
```

Run from this task worktree, with its assigned exclusive build target. Required genuine-fixture features may be added and recorded. A final test filter selecting zero tests proves nothing.

## Acceptance
- [ ] Fresh freeze is independently loadable. Changed role/compiler/installation/public binding/input refuses before launch. Export freeze(PreparedInputs,CandidateRoles,StageBindings) for fn-246.4.
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


## Owner-authorized batch B03 (bindings-composition)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.17, fn-242-gomad-shrink-the-qualification.19. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (1 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.

## Shared endpoint catalog integration

B03 additionally owns the exact leaf module file in Touches solely to expose read-only access to the existing endpoint mapping. Reuse that single catalog in composition; preserve all mappings, recipe argv and production derivation. Keep final partition descriptors in the owned composition module. Existing plan/leaf goldens remain the regression check. This authorizes no implementation before B01 completion and changes no original acceptance or proof owner.
