---
satisfies: [R3]
---
# fn-253-gomad-fast-validation-gates.2 Cheap stages first: measured cost classes in the cycle scheduler

## Description
R3: schedule cheap stages first in the Rust cycle verb.

1. From cycle evidence (`stages/<label>/receipt.json` `start_ns`/`end_ns`, cycles 6–8 under `/home/agent/wt/evidence/fn232-split-closure/`), derive a measured cost class per stage. Use leaf time, not boundary overhead.
2. Store the class in the stage table and have the scheduler prefer cheap stages: no long stage is admitted while a cheap stage without a long ancestor is pending.
3. Fail-fast stays as it is.

Today's long stages (cycle 8): `temporal-inspection` 727 s, `rust-default-test` 585 s, `helper-build` 528 s, the workload cells. Cheap: fmt, the lint matrix once fn-242 G3 lands, `validate`, `architecture`.

Ordering: after the fn-242 cycle verb has replaced the Python scheduler (R2 cutover); do not change `.flow/artifacts/wasm-qualification-driver/parallel/scheduler.py`.

## Acceptance
- Every stage has a cost class backed by recorded evidence; the scheduler enforces cheap-first order; a scheduler golden test shows the admission order for the real DAG.
- An injected failure in a cheap stage stops the cycle before any long stage is admitted (e2e fixture).


## Done summary
Blocked:
# fn-242 parked by owner

The owner requested: "can we move past fn-242 for now and leave that be?"

Stop fn-242 source changes, preparation cohorts, callbacks, proof reconciliation, reviews and task closure. Resume only on explicit owner direction. Preserve the worktrees, commits, admissions and evidence; parking supplies no completion credit.

Ordinary Rust lint and test suites under fn-253 may exercise existing cycle modules. This does not resume fn-242 acceptance work or authorize its standalone preparation/proof campaigns.

The last frozen source is `7e713d3d1a4adee10e85917a72c102a72e470dc1`. Its prepare-only run completed all 15 admissions, then refused configuration because setup supplied empty `live_corpus.families`. There is no genuine configuration or executed B10 callback proof from this run. Preserve `/home/agent/wt/evidence/fn242-b11-workload-reference-oracle-producers/genuine-prepare-7e713-run.json` and `genuine-prepare-7e713-root` unchanged.

The process-field source checkpoints `01902d7970` and `46e77aa9a7` and B10 landing `9469156b9d` remain unaccepted work. Existing passing checks retain only their original scope.

The overall MILESTONES goal continues through independent fn-253 work: inner loop, nextest, standalone entrypoints, gate visibility and evidence-store work that does not resume the parked cycle integration. fn-253.2/.3 still wait for cutover; downstream dependency gates are not waived. No full qualification attempt has been used: 0 of 2.
## Evidence
- Commits:
- Tests:
- PRs:
