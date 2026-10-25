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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
