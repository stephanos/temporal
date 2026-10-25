---
satisfies: [R8, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.8 Move the stock test fixture out of umpire_runtime

## Description
Implement **Runtime fixture (B12)**: move the `#[path]` Kettle fixture from `umpire/protocol/runtime/src/stock.rs` into the stock guest crate.

## Acceptance
- [ ] `umpire_runtime::Model` is no longer exported; the stock guest builds for wasm32-wasip1.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
The stock fixture and constructor are private to the stock guest; umpire_runtime no longer exports Model or a test fixture. Runtime integration tests include the guest-owned source and retain all existing assertions and goldens. Three implementation review draws returned SHIP with no findings.

Runtime/stock Rust tests, strict combined clippy, canonical wasm guest builds, native panic controls, current Temporal guest freshness and all three Go host modules passed on the joined tree. The integrated model tree is byte-identical to that tested tree. No semantic golden changed. Parent R9 retains its final full model gate after the remaining consolidation work.

Tier: session (jev-unavailable(no_key)). Reviewer: codex:gpt-6.1-sol:high; writer/reviewer are the same model family. Logs and review receipts: `.flow/tmp/milestones-resume-20261024/`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: aaaf2271324a4f4ce46bf8a2c1211e725cd81dc1
- Tests: Joined scoped cargo test for runtime, stock guest, catalog, Temporal syntax, Temporal and API; new-join-tests.exit=0, Combined --all-targets --all-features strict clippy; new-join-host-validation.exit=0, guest.sh umpire-temporal-guest and guest-freshness.sh; canonical build-guests.sh; all passed, CGO_ENABLED=0 mise exec -- go -C model/temporal/hosts/go test -tags test_dep -count=1 ./..., CGO_ENABLED=0 mise exec -- go -C model/umpire/hosts/go test -tags test_dep -count=1 ./... with current canonical guests, CGO_ENABLED=0 mise exec -- go -C model/umpire/reference/kettle test -tags test_dep -count=1 ./... with current Kettle guest
- PRs: