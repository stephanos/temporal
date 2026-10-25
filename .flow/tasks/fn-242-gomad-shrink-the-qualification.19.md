---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.19 Compose fixed stages and surviving endpoint bindings

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/preparation/composition/**

## Description
Port compose plus fixed static-plan construction helpers. Derive96 stages and24 static labels from owned recipes/admitted inputs, with exact leaf argv and final partitions. Preserve dynamic expansion sources and declared orchestration substitutions.

Exact implementation/proof ownership: `.flow/artifacts/fn242-port-proof-ownership.json`, task key `prep-composition`. This maps every assigned original check, function, endpoint and semantic obligation. Share authoritative validators for the same immutable boundary; independent reopening remains independent. The original aggregate acceptance gates remain in place.

### Quick commands

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle::preparation::composition::tests:: -- --nocapture
cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check
```

Run from this task worktree, with its assigned exclusive build target. Required genuine-fixture features may be added and recorded. A final test filter selecting zero tests proves nothing.

## Acceptance
- [ ] Fresh bounded composition emits only surviving named bindings. Missing/duplicate/stale/deleted/unmapped endpoint, wrong partition and changed immutable static plan refuse. An independent golden compares the production-derived plan; production never reads that golden.
- [ ] Resolve every assigned pending_regression using an actually executed normal-return valid control and targeted catching mutation against current Rust code. Port retained keep-row catchers. Record original IDs/source SHA256, current source binding, exact input hashes/bytes, commands, outputs, numeric exits and direct intended predicate attribution.
- [ ] Group checks through shared authoritative validators and fixture tables when input and trust boundary match. Preserve every compound condition term, bound, classification and independent reopening edge. Future test names, test counts, ancestor hash errors and mocked replacements do not resolve proof.
- [ ] Run genuine controlled Rust and approved Go guest/model/tool behavior. Preserve strict JSON/custody, documented current command semantics, all96 stages/145 semantic obligations and3600s stop_at_deadline=False timing-only policy. No Python bridge, placeholder pass, full qualification dispatch or spent attempt.
- [ ] Provide documented current Rust callable interfaces and update actual route consumers together. Retain endpoint/function mappings and fail closed for missing/unmapped routes until implemented.
- [ ] Quick filters select at least one actual test; retain nonzero selected count and numeric exits. Add existing feature flags required by the genuine family fixtures. If work exceeds one roughly100k-token iteration, create a further exact partition before dispatch and extend integration dependencies.

## Done summary
Production composition derives the fixed stage plan and bindings; actual missing/unknown/duplicate/partition/template and fresh-publication controls are mapped in the69-ID proof ledger.

Whole B03 reviewed SHIP after fixing the introduced R2 role-binding finding, same reviewer session01a170c0-bc0c-7d41-bcd1-c30003bc857a. Current source3065f3278a2d31a19db223affb9ed3e0e11f0432: all5genuine role builds, bothnative profiles, nativeadmission, corrected freeze/independent reopen and all9R2path/bytes/existence mutations passed. Original69checks/26functions/2endpoints retain exact current/legacy dispositions. Conductor independently verified2982sourcefiles,10245retainedinputs579930275bytes,21rawpins,16green gate logs and9genuine phase exits. Evidence: .flow/artifacts/fn242-b03-proof.json and external handoff/proof-map, with exact input/source/log hashes and exits.

The bounded fixture reserve is an operational execution decision only; production qualification45GiB/20GiB/12GiB admission and attempt/journal guarantees remain unchanged. No fullcycle qualification or later CLI/aggregate integration credit is claimed. Reviewer verified retained execution evidence; Cargo reruns were blocked by its read-only sandbox.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3065f3278a2d31a19db223affb9ed3e0e11f0432, d26ea2bbdc923a589012862ce5fd118f15d843af
- Tests: cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --lib cycle::preparation:: --config profile.release.package.gomad.opt-level=0 -- --nocapture --test-threads=1, cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --lib cycle::custody::tests:: --config profile.release.package.gomad.opt-level=0 -- --nocapture --test-threads=1, cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --lib cycle::plan:: --config profile.release.package.gomad.opt-level=0 -- --nocapture --test-threads=1, cargo +1.99.1 clippy --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --lib --no-deps --config profile.release.package.gomad.opt-level=0 -- -D warnings, cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check, git diff --check, cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --test cycle_custody_owner --no-run --message-format=json --config profile.release.package.gomad.opt-level=0, cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --lib cycle::preparation::bindings::tests::fixed_cells_rederive_durable_source_and_support_inputs --config profile.release.package.gomad.opt-level=0 -- --nocapture --test-threads=1, cargo +1.99.1 clippy --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --lib --no-deps --config profile.release.package.gomad.opt-level=0 -- -D warnings, cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check, git diff --check, cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --lib consumed_role_ --message-format=json --config profile.release.package.gomad.opt-level=0 -- --nocapture, cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --test cycle_custody_owner --no-run --message-format=json --config profile.release.package.gomad.opt-level=0, cargo +1.99.1 clippy --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad --release --lib --no-deps --config profile.release.package.gomad.opt-level=0 -- -D warnings, cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check, git diff --check, B03_CASE_VERSION=v5 B03_NATIVE_ROOT=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn242-b03-native-controls/v3 python3 /home/agent/wt/evidence/fn242-b03-bindings-composition/run-case-current.py native-controls [exit0], B03_CASE_VERSION=v5 B03_NATIVE_ROOT=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn242-b03-native-controls/v3 python3 /home/agent/wt/evidence/fn242-b03-bindings-composition/run-case-current.py freeze [owner_exit0,wrapper_exit0; immutable case metadata records exact environment]
- PRs:
## Owner-authorized batch B03

Implementation and independent review run under fn-242-gomad-shrink-the-qualification.17 with the complete batch scope in `.flow/artifacts/fn242-execution-batches.json`. This original task retains all acceptance and proof ownership. Its leader dependency prevents separate worker admission. After whole-batch SHIP, start and complete this receipt-only member in original topological order using the actual reviewed source, shared review receipt and its own executed proof.
