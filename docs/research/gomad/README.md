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

Use the [capability roadmap](../../../.plans/GOMAD_NEXT.md) for remaining
candidates and [milestones](../../../.plans/GOMAD_MILESTONES.md) for delivery order.
Current contracts live in the [specification](../../../tools/gomad3/SPEC.md) and
[architecture](../../../tools/gomad3/ARCHITECTURE.md).
