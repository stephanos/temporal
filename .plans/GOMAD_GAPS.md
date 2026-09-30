# Gomad functional-test gaps

[fn-106](../.flow/specs/fn-106-gomad-close-the-remaining-tests-gaps.md) owns the tasks
and acceptance criteria for the remaining functional-test gaps. Use `flowctl brief`
for task state. [GOMAD_MILESTONES.md](GOMAD_MILESTONES.md) retains the findings,
constraints, and measured limitations; qualification manifests retain dispositions.

The scope covers Linux seed-17 replay divergence, configurable I/O transcript
capacity for excluded suites, the single-P dedicated-cluster pool, traceback address
leakage, and the worker-command seed-11 hang. Tests that rely on wall-clock latency
and owned test bugs keep their recorded skips until their tests are fixed.

The [LastGC task](../.flow/tasks/fn-106-gomad-close-the-remaining-tests-gaps.2.md)
classifies the host-clock escape without changing the collector patch prohibition.
Classification is not a claim that the escape is fixed.
