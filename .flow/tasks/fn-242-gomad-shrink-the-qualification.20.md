---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.20 Validate runtime artifacts and forced replay aggregates

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

**Touches:** tools/gomad/rust/gomad/src/cycle/final_readers/native/**, tools/gomad/rust/gomad/src/cycle/final_readers/retained/**, tools/gomad/rust/gomad/src/cycle/readers/runtime/**

## Description
Port receiving runtime preparation/artifact/aggregate methods and runtime_gate. Preserve32 by25 fresh executions and32 forced full replays in production count validators, with bounded controls testing those validators.

Exact implementation/proof ownership: `.flow/artifacts/fn242-port-proof-ownership.json`, task key `reader-runtime`. This maps every assigned original check, function, endpoint and semantic obligation. Share authoritative validators for the same immutable boundary; independent reopening remains independent. The original aggregate acceptance gates remain in place.

### Quick commands

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle::readers::runtime::tests:: -- --nocapture
cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check
```

Run from this task worktree, with its assigned exclusive build target. Required genuine-fixture features may be added and recorded. A final test filter selecting zero tests proves nothing.

## Acceptance
- [ ] Execute semantic124 and runtime mutations for reused PID, wrong configuration, missing execution, altered choice/terminal replay and resource/correctness failure. Export validate_runtime/runtime_aggregate.
- [ ] Resolve every assigned pending_regression using an actually executed normal-return valid control and targeted catching mutation against current Rust code. Port retained keep-row catchers. Record original IDs/source SHA256, current source binding, exact input hashes/bytes, commands, outputs, numeric exits and direct intended predicate attribution.
- [ ] Group checks through shared authoritative validators and fixture tables when input and trust boundary match. Preserve every compound condition term, bound, classification and independent reopening edge. Future test names, test counts, ancestor hash errors and mocked replacements do not resolve proof.
- [ ] Run genuine controlled Rust and approved Go guest/model/tool behavior. Preserve strict JSON/custody, documented current command semantics, all96 stages/145 semantic obligations and3600s stop_at_deadline=False timing-only policy. No Python bridge, placeholder pass, full qualification dispatch or spent attempt.
- [ ] Provide documented current Rust callable interfaces and update actual route consumers together. Retain endpoint/function mappings and fail closed for missing/unmapped routes until implemented.
- [ ] Quick filters select at least one actual test; retain nonzero selected count and numeric exits. Add existing feature flags required by the genuine family fixtures. If work exceeds one roughly100k-token iteration, create a further exact partition before dispatch and extend integration dependencies.

## Done summary
Blocked:
# fn-242 parked by owner

The owner requested: "can we move past fn-242 for now and leave that be?"

Stop fn-242 source changes, preparation cohorts, callbacks, proof reconciliation, reviews and task closure. Resume only on explicit owner direction. Preserve the worktrees, commits, admissions and evidence; parking supplies no completion credit.

The last frozen source is `7e713d3d1a4adee10e85917a72c102a72e470dc1`. Its prepare-only run completed all 15 admissions, then refused configuration because setup supplied empty `live_corpus.families`. There is no genuine configuration or executed B10 callback proof from this run. Preserve `/home/agent/wt/evidence/fn242-b11-workload-reference-oracle-producers/genuine-prepare-7e713-run.json` and `genuine-prepare-7e713-root` unchanged.

The process-field source checkpoints `01902d7970` and `46e77aa9a7` and B10 landing `9469156b9d` remain unaccepted work. Existing passing checks retain only their original scope.

The overall MILESTONES goal continues through independent fn-253 work: inner loop, nextest, standalone entrypoints, gate visibility and evidence-store work that does not resume the parked cycle integration. fn-253.2/.3 still wait for cutover; downstream dependency gates are not waived. No full qualification attempt has been used: 0 of 2.
## Evidence
- Commits:
- Tests:
- PRs:


## Owner-authorized batch B08 (runtime-retained-native)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.20, fn-242-gomad-shrink-the-qualification.22, fn-242-gomad-shrink-the-qualification.23. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (1 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.
