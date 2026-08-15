# Gomad v3: Suggested Next Functionality

**Roadmap date:** 2026-08-14

## Purpose

Based on [GOMADv3_SUMMARY.md](GOMADv3_SUMMARY.md), Gomad v3 now has a trustworthy foundation for observing and exactly replaying a bounded set of runtime choices, explaining compatibility blockers, and publishing unambiguous qualification evidence. It is still too narrow, operationally incomplete, and platform-specific for default Temporal use. The next work should pursue four independent goals:

| Goal | Question it answers | Roadmap |
| --- | --- | --- |
| Bug-finding power | Can Gomad find failures that ordinary seeded repetition misses, explain them, and reduce them? | [GOMADv3_NEXT_BUG_FINDING.md](GOMADv3_NEXT_BUG_FINDING.md) |
| Compatibility | Can useful Temporal workloads run without weakening the deterministic contract? | [GOMADv3_NEXT_COMPATIBILITY.md](GOMADv3_NEXT_COMPATIBILITY.md) |
| Productionization | Can developers and CI operate Gomad safely, repeatably, and at scale? | [GOMADv3_NEXT_PRODUCTIONIZATION.md](GOMADv3_NEXT_PRODUCTIONIZATION.md) |
| Distributed-system simulation | Can Gomad reproduce the valuable multi-node, network, storage, crash, and nemesis behavior of v2 on the v3 evidence platform? | [GOMADv3_NEXT_SIM.md](GOMADv3_NEXT_SIM.md) |

These are separate investment tracks. Success in one does not imply success in the others: a stronger explorer is not useful if representative code cannot run, broader compatibility should not be called production-ready without durable recovery and data-handling controls, and explicit multi-node simulation is a different claim from deterministic execution of an unchanged Go test.

The goal documents are strategy and capability designs. Their original first slices are historical where this document marks the capability complete. Each remaining capability should become its own implementation plan after its predecessor meets the stated exit criteria; they should not be executed as one monolithic project.

## Current baseline

The trust and observability work that previously blocked every track is complete:

- World replay transports the recorded transition plan into the child and validates transitions before mutation.
- Resume accepts multiple run records that reference one content-deduplicated failure artifact without borrowing that artifact's seed or ordinal.
- Batch publication writes and syncs its final manifest before deleting prepared resume state.
- Virtual TCP covers queued-data/close and related dial, accept, deadline, and resource-exhaustion races.
- Supervisor cleanup signals only an independently validated target process group.
- Qualification reports separate `expectations_met` from supported, unsupported, failed, and infrastructure counts; support comparison requires explicit approval for boundary changes.
- Journal writers enforce the same 64 MiB bound as readers, and cancellation is distinct from deadline expiry.

Choice trace v2 and exact choice-tape replay are also implemented for the currently controlled runnable and `select` decisions. Complete retained traces produce identity-bound tapes automatically during artifact replay; the runtime checks each decision before applying it. The controller already has an internal prefix mode, but there is no public or durable alternative-prefix explorer yet.

Compatibility analysis and qualification reporting v1 are implemented through `gomad analyze`, `qualify-set`, and `compare-support`. The current checked qualification contracts are:

| Corpus | Selected | Supported | Unsupported | Expectations met | Execution |
| --- | ---: | ---: | ---: | --- | --- |
| Core tier 1 | 5 | 5 | 0 | Yes | Required on Gomad v3 pull requests and pushes |
| Temporal tier 2 | 16 | 4 | 12 | Yes | Scheduled or manually dispatched |

The twelve unsupported Temporal cases are useful blocker evidence, not support. Full runner qualification remains limited to Go 1.26.4 on `darwin/arm64`. Distributed-system simulation remains a design: no v3 multi-node, partition, durable-volume crash, or process-backed cluster capability has been implemented.

## Recommended order

### Milestone 0: restore the contract — complete

The replay, resume, publication, TCP, cleanup, qualification, cancellation, journal-limit, and network-resource corrections are implemented with adversarial regression coverage. Preserve these as required gates for every later capability.

### Milestone 1: make executions observable and compatibility measurable — complete

The two initial functions are implemented:

- bounded runtime-choice tracing, choice coverage, artifact retention, and `inspect --choices`;
- read-only compatibility analysis, tiered qualification manifests, explicit support counts, support comparison, and boundary-diff approval.

These are now the evidence sources for controlled exploration, compatibility prioritization, and simulation qualification.

### Milestone 2: explore controlled schedules and validate the simulation seam — next

Build a durable bounded alternative-prefix explorer above the implemented exact replay and internal prefix controller. Start with raw prefix search as the reference algorithm, record the explored envelope honestly, and compare failures and semantic outcomes per compute-hour against seed sampling before adding PCT or dependency reduction.

Use the sixteen-workload Temporal tier-2 corpus to rank exact compatibility packs and adapters by workloads unlocked. The immediate compatibility goal is to increase actual support beyond 4/16 without generic exemptions or host fallback, then add composed tier-3 scenarios.

For simulation, complete SIM-0: derive a machine-readable v2 behavioral-parity manifest and prototype one two-node request/response scenario and one restart scenario. Use those prototypes to validate the cluster API, fidelity declarations, limits, and in-process versus process-backed seam before building network or storage models.

### Milestone 3: reduce failures and make campaigns durable

Add schedule minimization, crash-consistent segmented journals, artifact quotas/pruning, deterministic campaign sharding, and merge. The output of a long campaign should be a small, durable reproduction that another machine with the same platform bundle can inspect and replay.

In the simulation track, implement only the cluster core, node and incarnation registry, runtime-domain inheritance, boot registry, lifecycle transitions, and stale-incarnation revocation needed by the validated prototypes. Keep network, storage, and fault semantics behind later deep modules.

### Milestone 4: add modeled faults, simulation parity, and a second qualified platform

Introduce bounded fault plans for explicit World and simulation adapters, starting with delay, cancellation, injected errors, and declared capacity outcomes. Deepen the cluster through node-aware network, durable-volume, lifecycle, scenario, nemesis, record, replay, and oracle modules in the staged order defined by the simulation roadmap.

In parallel, add a Linux platform bundle and run the same core and Temporal qualification contracts there. Platform work must remain independent of simulation semantics so neither can accidentally grant the other a support claim.

Avoid transparent network partitions outside the declared cluster model and avoid multi-P scheduling until the simpler event model, platform abstraction, and evidence contracts have proven stable.

### Milestone 5: qualify releases and process-backed simulation

Publish immutable, verifiable platform bundles with their qualification evidence, provenance, data policy, and rollback metadata. Add the process-backed simulation fidelity tier only after in-process network, storage, lifecycle, and oracle semantics pass their parity corpus; only that tier may claim fresh package globals and hard process isolation across node restarts.

### Milestone 6: evaluate research extensions

Use evidence from the earlier milestones to decide whether PCT, preemption bounding, semantic dependency metadata and DPOR, compiler checkpoints, deterministic GC, code-coverage guidance, richer I/O models, or multi-P execution have sufficient expected value. These are expensive and potentially contract-changing; none should be assumed mandatory.

## Cross-goal dependencies

| Capability | Status | Depends on | Enables |
| --- | --- | --- | --- |
| Choice trace and coverage | Implemented | Bounded shared-memory protocol and artifact schema | Forced replay, frontier feedback, minimization |
| Exact choice replay | Implemented | Stable logical choice identities and pre-apply divergence | Prefix exploration and stable reduced reproductions |
| Durable choice frontier | Next | Exact replay, internal prefix mode, bounded campaign state | Controlled exploration, PCT comparison, minimization |
| Compatibility analysis | Implemented | Existing closure review projected as structured evidence | Corpus prioritization and pack governance |
| Temporal support matrix | Implemented; 4/16 supported | Unambiguous qualification semantics | CI gating and platform comparison |
| Cluster core | Designed | Completed trust gate, parity manifest, validated scenario prototypes | Node-aware network, storage, faults, and lifecycle |
| In-process simulation parity | Designed | Cluster core plus network, volume, scenario, fault, record, and oracle modules | Fast distributed-system exploration |
| Process-backed simulation | Designed, later | Stable in-process model semantics and bounded child protocol | Hard crash/restart fidelity and full v2 behavioral parity |
| Platform bundle | Darwin only | Versioned patch, boundary, profile, and host-audit identity | Linux qualification and distributable releases |
| Segmented batch store | Not started | Crash-consistent publication and bounded readers | Large campaigns, sharding, pruning, aggregation |
| Fault plan | Not started | Correct World replay and explicit adapter semantics | Deterministic failure-path exploration |
| Release bundle | Partial installation support | Platform qualification and immutable identities | Repeatable CI and developer installation |

## Investment principles

- **Measure before widening.** Record why real targets are rejected and rank compatibility work by Temporal workload value.
- **Observe before controlling.** A bounded choice trace should precede systematic scheduling or minimization.
- **Validate seams before deepening models.** Prove node identity, lifecycle, and backend semantics with small scenarios before expanding network or storage behavior.
- **Keep simulation claims explicit.** Native deterministic execution, in-process logical nodes, and process-backed nodes are separate fidelity tiers and artifact identities.
- **Keep exactness local.** Promise exact replay only for the same target, platform bundle, inputs, and artifact identities.
- **Fail closed.** New compatibility must be expressed through reviewed platform manifests, compatibility packs, or deterministic adapters—not ambient host behavior.
- **Use deep modules.** Choice control, platform support, batch storage, and artifact policy should each have a small interface and own their invariants.
- **Make limits part of identity.** Search depth, trace bytes, fault count, journal bytes, and resource caps must be explicit in plans and artifacts.
- **Separate evidence from claims.** Report observed outcomes, coverage, unsupported cases, and approved boundary changes independently.
- **Prefer bounded useful search over theoretical completeness.** The near-term goal is finding and reducing more Temporal bugs per compute-hour, not claiming exhaustive verification.

## Portfolio checkpoints

Continue each track only if it demonstrates its intended value:

- **Bug finding:** controlled exploration finds seeded benchmark failures with fewer executions or finds failures not reached by seed sampling, and minimization materially reduces the reproduction.
- **Compatibility:** the supported share of a representative Temporal corpus grows without host escapes or generic exemptions.
- **Productionization:** interrupted and sharded campaigns recover deterministically, artifacts remain within quotas, and a clean machine can install and verify an immutable qualified bundle.
- **Distributed-system simulation:** v2-derived parity cases pass at the declared fidelity tier, at least one representative Temporal scenario finds and replays a modeled failure, and cluster artifacts remain bounded and exact-replayable.

If these checkpoints are not met, narrow the product rather than accumulating additional runtime patching and compatibility obligations.
