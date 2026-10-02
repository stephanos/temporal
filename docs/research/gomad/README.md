# Gomad research

These reports preserve dated source snapshots, external references, trade-offs,
and experimental proposals. They do not define current support or task state.

| Report | Assessment date | Question |
| --- | --- | --- |
| [Loom and deterministic simulation testing](GOMAD_CMPv2.md) | 2026-09-27 | Which oracle, workload-generation, search, and debugging ideas could improve Gomad? |
| [System-level determinism](GOMAD3_OS.md) | 2026-08-15 | What would a Linux/arm64 escape firewall or mixed-language QEMU replay experiment require? |
| [Host-clock escapes](GOMAD_HOST_CLOCK_ESCAPES.md) | 2026-09-30 | Which host-clock values reach a target, do they affect replay, and what remedies fit the patch policy? |
| [D18 worker cancel command after a workflow timeout](GOMAD_D18_WORKER_CANCEL_DELIVERY.md) | 2026-09-30 | Why does the worker cancel command never reach the control queue on some seeds, and who owns the correction? |
| [D16 forward-clock poll deadline](GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md) | 2026-10-01 | Why does a 3 s long poll hit the client deadline before the empty response under the forward clock, and who owns the correction? |
| [D17 Nexus operation test with two clusters](GOMAD_D17_NEXUS_OTEL_TWO_CLUSTERS.md) | 2026-10-01 | Why do same-seed runs of the two-cluster Nexus tracing test differ, and who owns the correction? |
| [D19 activity fairness backlog readiness](GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md) | 2026-10-01 | Why does the activity fairness test measure a partial backlog under Gomad, is matching unfair, and who owns the correction? |
| [D20 heartbeat timeout counting](GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md) | 2026-10-01 | Why does the workflow task heartbeat test count one timeout instead of two under virtual time, and who owns the correction? |
| [What is next for Gomad](2026-10-01-gomad-vision.md) | 2026-10-01 | Which GOMAD_CMP.md ideas are feasible, what new ideas exist, and what direction follows? |
| [Feasibility: schedule search](2026-10-01-feasibility-schedule-search.md) | 2026-10-01 | What do the runtime, choice records, and explorers offer search policies today? |
| [Feasibility: workload, feedback, diagnosis](2026-10-01-feasibility-workload-diagnosis.md) | 2026-10-01 | What exists for inputs, guidance, oracles, minimization, and failure inspection? |
| [Industry simulation-testing practice](2026-10-01-industry-dst-practice.md) | 2026-10-01 | Which mechanisms from other teams' deterministic simulation testing transfer? |
| [Academic concurrency-testing survey](2026-10-01-academic-concurrency-testing.md) | 2026-10-01 | Which search algorithms and feedback signals fit Gomad's controller? |

Use the [capability roadmap](../../../.plans/GOMAD_NEXT.md) for remaining
candidates and [milestones](../../../MILESTONES.md) for delivery order.
Current contracts live in the [specification](../../../tools/gomad3/SPEC.md) and
[architecture](../../../tools/gomad3/ARCHITECTURE.md).
