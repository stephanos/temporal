---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.51 Close final partitions and prove cluster admission edges

**Touches:** tools/gomad/CLI.md, tools/gomad/rust/gomad/src/**, tools/gomad/rust/gomad/src/cli/**, tools/gomad/rust/gomad/src/cycle/**, tools/gomad/rust/gomad/src/cycle/config/**, tools/gomad/rust/gomad/src/cycle/final/**, tools/gomad/rust/gomad/src/cycle/final_readers/**, tools/gomad/rust/gomad/src/cycle/final_readers/admission/**, tools/gomad/rust/gomad/src/cycle/preparation/**, tools/gomad/rust/gomad/src/cycle/producers/**, tools/gomad/rust/gomad/src/cycle/readers/**, tools/gomad/rust/gomad/src/cycle/routes/**, tools/gomad/rust/gomad/src/lib.rs, tools/gomad/rust/gomad/src/main.rs, tools/gomad/rust/gomad/tests/**

## Description
Port final merge/partition/collection and21 concrete cluster mutations across four independent verify/storage-selections/reopen/replay edges. Reuse real existing APIs at each edge. Consume completed FinalSection reports after all actual reader sections exist.

Exact implementation/proof ownership: `.flow/artifacts/fn242-port-proof-ownership.json`, task key `final-admission`. This maps every assigned original check, function, endpoint and semantic obligation. Share authoritative validators for the same immutable boundary; independent reopening remains independent. The original aggregate acceptance gates remain in place.

### Quick commands

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle::final_readers::admission::tests:: -- --nocapture
cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check
```

Run from this task worktree, with its assigned exclusive build target. Required genuine-fixture features may be added and recorded. A final test filter selecting zero tests proves nothing.

## Required B15 startup integration — explicit cycle process owner

Wire fn-242.13's actual exclusive CycleOwner initialization at cycle CLI startup before any child or worker thread. All cycle Runner/busy/build/producer spawn paths require that same real PID-bound domain token. Missing/mixed ownership refuses; unrelated CLI commands never initialize the domain. Verify whole-domain abort/drain after supervisor loss through the actual integrated route. A standalone .13 fixture supplies no production startup credit. Preserve all original aggregate and route acceptance. See `.flow/artifacts/fn242-process-custody-decision.md`.

## Acceptance
- [ ] Execute all84 semantic IDs through their real edge with controls; one edge cannot stand in for another. Exact145 final ownership is enforced and missing/duplicate/foreign/synthetic/skipped/unexecuted rows refuse.
- [ ] Resolve every assigned pending_regression using an actually executed normal-return valid control and targeted catching mutation against current Rust code. Port retained keep-row catchers. Record original IDs/source SHA256, current source binding, exact input hashes/bytes, commands, outputs, numeric exits and direct intended predicate attribution.
- [ ] Group checks through shared authoritative validators and fixture tables when input and trust boundary match. Preserve every compound condition term, bound, classification and independent reopening edge. Future test names, test counts, ancestor hash errors and mocked replacements do not resolve proof.
- [ ] Run genuine controlled Rust and approved Go guest/model/tool behavior. Preserve strict JSON/custody, exact unchanged leaf argv, all96 stages/145 semantic obligations and3600s stop_at_deadline=False timing-only policy. No Python bridge, placeholder pass, full qualification dispatch or spent attempt.
- [ ] Provide family callable modes with the old CLI input/output/refusal contract for actual routes integration. Retain endpoint/function mappings and fail closed for missing/unmapped routes until implemented.
- [ ] Quick filters select at least one actual test; retain nonzero selected count and numeric exits. Add existing feature flags required by the genuine family fixtures. If work exceeds one roughly100k-token iteration, create a further exact partition before dispatch and extend integration dependencies.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


## Owner-authorized batch B15 (routes-original-aggregates)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.51, fn-242-gomad-shrink-the-qualification.52, fn-242-gomad-shrink-the-qualification.2, fn-242-gomad-shrink-the-qualification.3, fn-242-gomad-shrink-the-qualification.4, fn-242-gomad-shrink-the-qualification.9, fn-242-gomad-shrink-the-qualification.10. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (1 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.
