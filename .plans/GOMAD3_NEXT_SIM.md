# Gomad v3 remaining simulation work

The application harness, lifecycle, virtual TCP, durable volumes, scenarios,
faults, and process backend are documented in the
[architecture](../tools/gomad3/ARCHITECTURE.md#in-process-cluster-simulation) and
[specification](../tools/gomad3/SPEC.md#simulation-distributed-system-simulation).
Those documents own their semantics and fidelity limits. Completed SIM-0 through
SIM-5 delivery plans remain in Git history; the obsolete v2 source-parity manifest
is not an active requirement.

Simulation is a separate capability from running an unchanged functional test.
[GOMAD_MILESTONES.md](GOMAD_MILESTONES.md) governs the latter's delivery order.

## SIM-6: Controlled schedule and fault exploration

The combined frontier already coordinates runtime, scenario, network, storage,
fault, and crash-state alternatives with durable rounds and exact artifacts.
Remaining work is representative equal-budget benchmarking and crash-resumable
minimizer state. Typed scenario-input shrinking is shared with
[BUG-5](GOMAD3_NEXT_BUG_FINDING.md#bug-5-failure-minimization).

Compare failures and declared distributed outcomes per compute-hour against seed
sampling. Report neutral results and recovery executions. Keep distinct candidate
prefixes even when their semantic evidence deduplicates; an evidence hash is not a
sound visited-state equivalence for native execution.

An extension must retain bounded inspectable work, durable remaining candidates,
fail-before-mutation replay, and immutable parent artifacts. Accepted reductions
preserve failure identity, exact schedule/model replay, and input validity.

## SIM-7: Evidence-driven expansion beyond v2

Select the smallest deep model justified by a named consumer and retained blocker
or failure evidence. Each addition needs a semantic contract, host-escape canary,
exact replay, positive/negative/capacity tests, and performance evidence.

| Area | Candidate extensions | Required distinction |
| --- | --- | --- |
| Network | Loss/duplication/reordering/corruption; latency distributions, bandwidth and backpressure; half-open connections; explicit discovery records; interfaces/routing; Unix streams, UDP, IPv6; proxy/pool adapters | Specify transport semantics. Arbitrary byte deletion cannot stand in for TCP packet loss. Directional links and fixed delay already exist. |
| Storage | Space/inode/I/O errors; torn/short writes; corruption; detach/remount/snapshots; permissions/links/locks; shared-volume or object-store/queue/database adapters | Declare persistence dependencies and atomic units for each fault. |
| Node/environment | Pause, boot failure/crash loops, clock offset/drift, quotas/stalls, resource/entropy pressure, rolling upgrades | Preserve the monotonic clock and backend fidelity contract. CPU quotas do not imply hardware-race coverage. |
| Workloads/oracles | Typed Temporal actors, generated operation sequences, typed shrinkers, bounded linearizability/serializability, reconciliation, differential scenarios | Keep generation validity, independent expected results, and replay separate. |

A model's evidence names supported operations, simplifications, bounds, workloads,
and known differences from host behavior. Reports distinguish expectation matching,
operation support, backend fidelity, and actual workload completion.

## Oracles and recovery

Prioritize independent sequential reference models and bounded progress monitors
with declared availability assumptions. Exercise fault, heal, then required
progress phases. Record fairness and logical deadlines. Coverage witnesses prove a
required event was reached; they do not replace assertions about its result.

Independently review generators and oracles. Compare model behavior with stock Go
or a local cluster where a meaningful semantic contract exists, and retain native
integration, race, and performance testing. Visited-state pruning requires complete
future-relevant explicit-model snapshots. Partial-order reduction needs conservative
transition dependencies and comparison with an unreduced reference search.

## Verification and fidelity

Cross-backend tests compare detached topology, model transitions, volume state,
histories, oracles, and normalized outcomes. Runtime tapes and raw timing/logs can
differ; one artifact is replayed only on its recorded backend.

Process-only tests must prove fresh package initialization, hard crash/reap, and
unambiguous model-operation commit when a node dies during IPC. Both backends need
stale-incarnation, close/deadline, sync/dependency, malformed-input, capacity,
replay-mutation, publication interruption, and host-load checks. Soak nodes,
connections, volume operations, process churn, histories, and artifact growth at
ten times the representative scale.

Qualify useful scenarios in order of dependency: small service pairs, Temporal
retry/timeout interactions, persistence recovery, partition/heal with convergence,
then crash after acknowledgment with a durability oracle. Hard-crash claims require
the process backend. Race mode and mixed-language machine simulation remain
separate research profiles.

See [comparative research](../docs/research/gomad/GOMAD_CMPv2.md) for workload and
oracle candidates and the [machine study](../docs/research/gomad/GOMAD3_OS.md) for
the separate QEMU replay proposal.
