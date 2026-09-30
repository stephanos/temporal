# Gomad v3 capability roadmap

[GOMAD_MILESTONES.md](GOMAD_MILESTONES.md) governs delivery order and the constraints
on every proposal here. The open Flow-Next specs own committed work and acceptance
criteria. This roadmap describes candidates beyond that work; it does not create
a separate task queue.

## Current baseline

Use the [README](../tools/gomad3/README.md) for implemented capabilities and platform
limits, the [specification](../tools/gomad3/SPEC.md) for product requirements, and the
[architecture](../tools/gomad3/ARCHITECTURE.md) for module ownership and replay
contracts. Qualification manifests and reports own the per-workload observations.

The active functional-test gaps are tracked in [GOMAD_GAPS.md](GOMAD_GAPS.md).
[Deferred follow-ups](GOMAD_FOLLOWUPS.md) carry their own revival triggers.
[Downstream findings](GOMAD_CLOUD.md) distinguish Gomad capabilities from changes
that the consuming module must supply.

## Remaining tracks

| Track | Remaining opportunities | Evidence needed before expansion |
| --- | --- | --- |
| [Bug finding](GOMAD3_NEXT_BUG_FINDING.md) | Durable minimizer resume, typed input shrinking, broader adapter faults, measured search policies | Distinct replayable failures or useful outcomes per compute-hour against equal-budget seed sampling |
| [Compatibility](GOMAD3_NEXT_COMPATIBILITY.md) | Workload-driven models/adapters, additional platforms, upgrade impact and rollback evidence | Named blocked workloads, exact reviewed boundaries, platform-specific qualification |
| [Productionization](GOMAD3_NEXT_PRODUCTIONIZATION.md) | Store-wide retention/export policy, qualified releases, CI orchestration, aggregate reports, load evidence | Recoverable bounded operation, clean-host installation, explicit data and compatibility policies |
| [Simulation](GOMAD3_NEXT_SIM.md) | Combined-search benchmarks, independent oracles, generated/shrinkable scenarios, selected model extensions | Declared semantics, independent checks, exact replay, capacity tests, named consumers |

## Investment rules

- Fix the active qualification gaps before broadening the functional-test claim.
- Start a capability spec only when its consumer and measurable exit criteria are known.
- Keep model semantics in deep modules; the Runner composes validated evidence and budgets.
- Judge search by failures and semantic outcomes per compute-hour. Report neutral or
  worse results as well as improvements.
- Preserve separate identities for runtime choices, explicit models, and execution
  profiles. A change to one controller must not silently alter another controller's tape.
- Keep exact replay bound to the target, platform, model, inputs, and profile.
  Expected unsupported outcomes and bounded search completion do not prove correctness.

## Research

The [Loom and simulation-testing assessment](../docs/research/gomad/GOMAD_CMPv2.md)
explores oracles, workload distributions, search policies, and differential testing.
The [system-level determinism study](../docs/research/gomad/GOMAD3_OS.md) proposes a
Linux/arm64 escape firewall and a separate mixed-language machine-replay experiment.
Both are dated research, not prerequisites for the active specs.
