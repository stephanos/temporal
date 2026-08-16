# Gomad v3: Suggested Next Functionality

**Roadmap date:** 2026-08-15

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

Choice trace v2, exact choice-tape replay, and bounded alternative-prefix
exploration are implemented for the currently controlled runnable and `select`
decisions. Complete retained traces produce identity-bound replay plans
automatically during artifact replay; the runtime checks each decision before
applying it. Choice-frontier campaigns use one base seed, breadth-first forced
prefixes, explicit run/depth/byte bounds, round-atomic journals, and durable
resume. On the pinned two-outcome `select` fixture, frontier and seed sampling
both reach two declared outcomes in sixteen executions; this records the raw
frontier's current neutral efficiency rather than claiming an advantage.

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

- [BUG-1](GOMADv3_NEXT_BUG_FINDING.md#bug-1-runtime-choice-trace), [BUG-2](GOMADv3_NEXT_BUG_FINDING.md#bug-2-choice-coverage-and-feedback), and [BUG-3](GOMADv3_NEXT_BUG_FINDING.md#bug-3-exact-choice-tape-replay): bounded runtime-choice tracing, choice coverage, artifact retention, `inspect --choices`, and exact replay;
- [COMPAT-1](GOMADv3_NEXT_COMPATIBILITY.md#compat-1-gomad-analyze), [COMPAT-2](GOMADv3_NEXT_COMPATIBILITY.md#compat-2-unambiguous-support-matrices), and the initial [COMPAT-3](GOMADv3_NEXT_COMPATIBILITY.md#compat-3-tiered-temporal-corpus): read-only compatibility analysis, tiered qualification manifests, explicit support counts, support comparison, and boundary-diff approval.

These are now the evidence sources for controlled exploration, compatibility prioritization, and simulation qualification.

### Milestone 2: explore controlled schedules and validate the simulation seam — in progress

[BUG-4](GOMADv3_NEXT_BUG_FINDING.md#bug-4-bounded-alternative-prefix-exploration) is implemented as the durable bounded choice-frontier strategy. The pinned equal-budget comparison currently shows no outcome-efficiency advantage over seed sampling, so BUG-5 and later search refinements remain gated on stronger benchmark evidence rather than being added speculatively.

Continue [COMPAT-3](GOMADv3_NEXT_COMPATIBILITY.md#compat-3-tiered-temporal-corpus) by using the sixteen-workload Temporal tier-2 corpus to rank exact compatibility packs and adapters by workloads unlocked. [COMPAT-4](GOMADv3_NEXT_COMPATIBILITY.md#compat-4-compatibility-pack-development-kit) is implemented with a v2-only exact-source contract, exact-digest approval, generated changed-version/source rejection tests, and independent qualification. The first evidence-ranked [COMPAT-5](GOMADv3_NEXT_COMPATIBILITY.md#compat-5-targeted-deterministic-adapters-and-io-models) slice is complete: an exact `google.golang.org/grpc@v1.80.0` adapter removes the meaningless host keepalive callback from Gomad's virtual TCP path, and `temporal-backoff-overflow` now executes and exactly replays in closure mode. The baseline is 5/16 with no generic exemption or host fallback. [COMPAT-6](GOMADv3_NEXT_COMPATIBILITY.md#compat-6-safer-handling-of-transitive-forbidden-dependencies) has an experimental compiler/linker-backed mode, but the remaining evaluated candidates retain real assembly, linkname, `syscall`, or forbidden-import blockers; closure review remains the default. Rank the remaining eleven blockers before selecting another exact adapter or I/O model, then add composed tier-3 scenarios from the expanded supported set.

For simulation, complete [SIM-0](GOMADv3_NEXT_SIM.md#sim-0-restore-trust-and-define-the-parity-contract): derive a machine-readable v2 behavioral-parity manifest and prototype one two-node request/response scenario and one restart scenario. Use those prototypes to validate the cluster API, fidelity declarations, limits, and in-process versus process-backed seam before building network or storage models.

### Milestone 3: reduce failures and make campaigns durable

Add [BUG-5](GOMADv3_NEXT_BUG_FINDING.md#bug-5-failure-minimization) and make campaigns durable. Move planned, prepared, running, committing, published, and recoverable-failure transitions behind [PROD-1](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-1-crash-consistent-batch-store); add `gomad recover`, make `inspect` explain recoverability, and make `resume` delegate integrity decisions to the store. Replace the single journal with [PROD-2](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-2-segmented-bounded-journals) immutable bounded segments and streaming readers.

Implement [PROD-4](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-4-deterministic-campaign-plans-sharding-and-merge) by separating canonical campaign plans from execution and adding deterministic sharding and identity-checking merge. Begin [PROD-8](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-8-resource-control-and-performance) by enforcing global and per-run resource limits with backpressure and explicit capacity outcomes. The output of a long campaign should be a small, durable reproduction that another machine with the same platform bundle can inspect and replay.

In the simulation track, implement only [SIM-1](GOMADv3_NEXT_SIM.md#sim-1-cluster-core-and-in-process-runtime-domains): the cluster core, node and incarnation registry, runtime-domain inheritance, boot registry, lifecycle transitions, and stale-incarnation revocation needed by the validated prototypes. Keep network, storage, and fault semantics behind later deep modules.

### Milestone 4: add modeled faults, simulation parity, and a second qualified platform

Introduce [BUG-6](GOMADv3_NEXT_BUG_FINDING.md#bug-6-deterministic-fault-plans) for explicit World and simulation adapters, starting with delay, cancellation, injected errors, dropped modeled delivery, and declared capacity outcomes.

Complete [SIM-2](GOMADv3_NEXT_SIM.md#sim-2-virtual-network-parity) through [SIM-4](GOMADv3_NEXT_SIM.md#sim-4-scenarios-nemeses-records-and-oracles) in order: first node-aware virtual-network parity with topology, delay, partition/heal, and lifecycle semantics; then [SIM-3](GOMADv3_NEXT_SIM.md#sim-3-durable-volume-parity) durable-volume parity with volatile/persisted views, sync dependencies, partial-crash outcomes, and bounded crash-state enumeration; then typed scenarios, nemeses, histories, records, replay, and semantic oracles. Do not merge these models into one shallow simulator interface.

In parallel, continue [COMPAT-7](GOMADv3_NEXT_COMPATIBILITY.md#compat-7-platform-bundles) with a Linux platform bundle and run the same core and Temporal qualification contracts there. Platform work must remain independent of simulation semantics so neither can accidentally grant the other a support claim.

Avoid transparent network partitions outside the declared cluster model and avoid multi-P scheduling until the simpler event model, platform abstraction, and evidence contracts have proven stable.

### Milestone 5: qualify releases and process-backed simulation

Implement [PROD-3](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-3-artifact-lifecycle-and-data-policy), covering quotas, retention, reachability, pruning, sensitivity, export, mount and environment capture, and replay inputs that must be resupplied rather than retained. Complete [PROD-5](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-5-immutable-release-and-installation-bundles) with signed checksums or attestations, qualification evidence, provenance, SBOM/notices, installation metadata, and rollback support.

Implement [PROD-6](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-6-ci-integration), a supported CI entry point for planning, sharding, caching, bounded artifact upload, merging, exact reruns, support-baseline gating, and outcome classification. Complete [PROD-7](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-7-observability-and-reporting) with `gomad report`, stable aggregate JSON, metrics adapters, and external trend reporting. Finish [PROD-8](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-8-resource-control-and-performance) with the prepared-target cache only after resource correctness, keyed by the complete immutable execution identity and revalidated on every hit.

Complete [COMPAT-8](GOMADv3_NEXT_COMPATIBILITY.md#compat-8-dependency-and-go-upgrade-impact-reports) by extending dependency and Go upgrade dossiers to report changed targets, packs, adapters, workload support, boundary dispositions, approvals, and rollback availability. Implement [PROD-9](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-9-release-governance) by defining schema/CLI compatibility windows, evidence-preserving migrations, release ownership, qualification state, known limitations, maturity labels, and rollback targets.

Complete [SIM-5](GOMADv3_NEXT_SIM.md#sim-5-process-backed-fidelity-tier) only after in-process network, storage, lifecycle, and oracle semantics pass their parity corpus. The process-backed fidelity tier owns bounded node launch, typed model IPC, time arbitration, hard crash/reap, and fresh-incarnation initialization; only this tier may claim fresh package globals and hard process isolation across node restarts.

### Milestone 6: integrate simulation exploration and expand by evidence

Complete [SIM-6](GOMADv3_NEXT_SIM.md#sim-6-controlled-schedule-and-fault-exploration) by combining runtime-choice, scenario, network, storage, crash-state, and fault tapes behind one bounded frontier. Resume must preserve remaining work without rediscovery; semantic deduplication and minimization must retain the exact schedule-plus-fault failure identity.

Complete [SIM-7](GOMADv3_NEXT_SIM.md#sim-7-evidence-driven-expansion-beyond-v2) by running the Temporal corpus to rank beyond-v2 simulation gaps by workloads unlocked. Add only the smallest deep model justified by named consumers, with a semantic contract, host-escape canary, exact replay, capacity and failure tests, and performance evidence. Do not pursue UDP, DNS, routing, corruption, resource-pressure, or external-service breadth for API-count parity.

### Milestone 7: evaluate research extensions

Use [BUG-7](GOMADv3_NEXT_BUG_FINDING.md#bug-7-later-research-extensions) evidence from the earlier milestones to decide whether PCT, preemption bounding, semantic dependency metadata and DPOR, compiler checkpoints, deterministic GC, code-coverage guidance, richer I/O models, or multi-P execution have sufficient expected value. These are expensive and potentially contract-changing; none should be assumed mandatory.

## Complete track coverage

The tables below account for every named capability and delivery stage in the four detailed roadmaps. The companion documents remain normative for their non-goals, module boundaries, protocols, error classifications, 10×-load behavior, trade-offs, verification plans, and exit criteria.

### Bug-finding power

| Capability | Status | Portfolio placement |
| --- | --- | --- |
| [BUG-1: Runtime-choice trace](GOMADv3_NEXT_BUG_FINDING.md#bug-1-runtime-choice-trace) | Implemented | Milestone 1 |
| [BUG-2: Choice coverage and feedback](GOMADv3_NEXT_BUG_FINDING.md#bug-2-choice-coverage-and-feedback) | Implemented | Milestone 1 |
| [BUG-3: Exact choice-tape replay](GOMADv3_NEXT_BUG_FINDING.md#bug-3-exact-choice-tape-replay) | Implemented | Milestone 1 |
| [BUG-4: Bounded alternative-prefix exploration](GOMADv3_NEXT_BUG_FINDING.md#bug-4-bounded-alternative-prefix-exploration) | Implemented; pinned equal-budget comparison is neutral | Milestone 2 |
| [BUG-5: Failure minimization](GOMADv3_NEXT_BUG_FINDING.md#bug-5-failure-minimization) | Planned | Milestone 3 |
| [BUG-6: Deterministic fault plans](GOMADv3_NEXT_BUG_FINDING.md#bug-6-deterministic-fault-plans) | Planned | Milestone 4 |
| [BUG-7: Later research extensions](GOMADv3_NEXT_BUG_FINDING.md#bug-7-later-research-extensions) | Deferred pending evidence | Milestone 7 |

### Temporal and platform compatibility

| Capability | Status | Portfolio placement |
| --- | --- | --- |
| [COMPAT-1: `gomad analyze`](GOMADv3_NEXT_COMPATIBILITY.md#compat-1-gomad-analyze) | Implemented | Milestone 1 |
| [COMPAT-2: Unambiguous support matrices](GOMADv3_NEXT_COMPATIBILITY.md#compat-2-unambiguous-support-matrices) | Implemented | Milestone 1 |
| [COMPAT-3: Tiered Temporal corpus](GOMADv3_NEXT_COMPATIBILITY.md#compat-3-tiered-temporal-corpus) | Tier 1 and tier 2 implemented; tier 3 planned | Milestones 1–2 |
| [COMPAT-4: Compatibility-pack development kit](GOMADv3_NEXT_COMPATIBILITY.md#compat-4-compatibility-pack-development-kit) | Implemented with v2-only exact-source packs and exact-digest approval | Milestone 2 |
| [COMPAT-5: Targeted deterministic adapters and I/O models](GOMADv3_NEXT_COMPATIBILITY.md#compat-5-targeted-deterministic-adapters-and-io-models) | Evidence-driven | Milestones 2 and 4 |
| [COMPAT-6: Safer handling of transitive forbidden dependencies](GOMADv3_NEXT_COMPATIBILITY.md#compat-6-safer-handling-of-transitive-forbidden-dependencies) | Experimental linked mode implemented; real-workload exit criterion unmet, closure remains default | Milestone 2 |
| [COMPAT-7: Platform bundles](GOMADv3_NEXT_COMPATIBILITY.md#compat-7-platform-bundles) | `darwin/arm64` qualified; Linux planned | Milestones 4–5 |
| [COMPAT-8: Dependency and Go upgrade impact reports](GOMADv3_NEXT_COMPATIBILITY.md#compat-8-dependency-and-go-upgrade-impact-reports) | Boundary approval exists; full impact report planned | Milestone 5 |

### Productionization

| Capability | Status | Portfolio placement |
| --- | --- | --- |
| [PROD-1: Crash-consistent batch store](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-1-crash-consistent-batch-store) | Publication foundation fixed; deep store and recovery planned | Milestone 3 |
| [PROD-2: Segmented, bounded journals](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-2-segmented-bounded-journals) | Planned | Milestone 3 |
| [PROD-3: Artifact lifecycle and data policy](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-3-artifact-lifecycle-and-data-policy) | Planned | Milestone 5 |
| [PROD-4: Deterministic campaign plans, sharding, and merge](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-4-deterministic-campaign-plans-sharding-and-merge) | Planned | Milestone 3 |
| [PROD-5: Immutable release and installation bundles](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-5-immutable-release-and-installation-bundles) | Installation discovery exists; qualified bundles planned | Milestone 5 |
| [PROD-6: CI integration](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-6-ci-integration) | Qualification workflow exists; supported campaign entry point planned | Milestone 5 |
| [PROD-7: Observability and reporting](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-7-observability-and-reporting) | Stable events exist; aggregate reporting and metrics planned | Milestone 5 |
| [PROD-8: Resource control and performance](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-8-resource-control-and-performance) | Bounded protocols exist; campaign-wide limits, backpressure, and cache planned | Milestones 3 and 5 |
| [PROD-9: Release governance](GOMADv3_NEXT_PRODUCTIONIZATION.md#prod-9-release-governance) | Boundary approval and current readers exist; explicit compatibility and release policy planned | Milestone 5 |

### Distributed-system simulation

| Delivery stage | Status | Portfolio placement |
| --- | --- | --- |
| [SIM-0: Restore trust and define the parity contract](GOMADv3_NEXT_SIM.md#sim-0-restore-trust-and-define-the-parity-contract) | Trust restored; parity manifest and prototypes next | Milestone 2 |
| [SIM-1: Cluster core and in-process runtime domains](GOMADv3_NEXT_SIM.md#sim-1-cluster-core-and-in-process-runtime-domains) | Designed | Milestone 3 |
| [SIM-2: Virtual network parity](GOMADv3_NEXT_SIM.md#sim-2-virtual-network-parity) | Designed | Milestone 4 |
| [SIM-3: Durable volume parity](GOMADv3_NEXT_SIM.md#sim-3-durable-volume-parity) | Designed | Milestone 4 |
| [SIM-4: Scenarios, nemeses, records, and oracles](GOMADv3_NEXT_SIM.md#sim-4-scenarios-nemeses-records-and-oracles) | Designed | Milestone 4 |
| [SIM-5: Process-backed fidelity tier](GOMADv3_NEXT_SIM.md#sim-5-process-backed-fidelity-tier) | Designed, after in-process parity | Milestone 5 |
| [SIM-6: Controlled schedule and fault exploration](GOMADv3_NEXT_SIM.md#sim-6-controlled-schedule-and-fault-exploration) | Designed, after SIM-5 | Milestone 6 |
| [SIM-7: Evidence-driven expansion beyond v2](GOMADv3_NEXT_SIM.md#sim-7-evidence-driven-expansion-beyond-v2) | Designed, after measurable parity | Milestone 6 |

The simulation track also carries the adjacent v2 obligations explicitly: same-seed equality and different-seed diversity remain qualification requirements; typed choice/model records replace opaque trace checksums; only the process-backed tier can provide fresh globals and hard cleanup; output remains node/incarnation-aware while preserving Go-test presentation; and race detection remains a separate non-Gomad profile rather than an implied single-P capability.

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
