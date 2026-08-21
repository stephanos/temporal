# Gomad v3: Suggested Next Functionality

**Roadmap date:** 2026-08-20

## Purpose

Based on [GOMAD3_SUMMARY.md](GOMAD3_SUMMARY.md), Gomad v3 now has a trustworthy foundation for observing and exactly replaying a bounded set of runtime choices, explaining compatibility blockers, and publishing unambiguous qualification evidence. It is still too narrow, operationally incomplete, and platform-specific for default Temporal use. The next work should pursue four independent goals:

| Goal | Question it answers | Roadmap |
| --- | --- | --- |
| Bug-finding power | Can Gomad find failures that ordinary seeded repetition misses, explain them, and reduce them? | [GOMAD3_NEXT_BUG_FINDING.md](GOMAD3_NEXT_BUG_FINDING.md) |
| Compatibility | Can useful Temporal workloads run without weakening the deterministic contract? | [GOMAD3_NEXT_COMPATIBILITY.md](GOMAD3_NEXT_COMPATIBILITY.md) |
| Productionization | Can developers and CI operate Gomad safely, repeatably, and at scale? | [GOMAD3_NEXT_PRODUCTIONIZATION.md](GOMAD3_NEXT_PRODUCTIONIZATION.md) |
| Distributed-system simulation | Can Gomad reproduce the valuable multi-node, network, storage, crash, and nemesis behavior of v2 on the v3 evidence platform? | [GOMAD3_NEXT_SIM.md](GOMAD3_NEXT_SIM.md) |

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
- New journals use independently hashed immutable segments and a digest-bound compact index; historical single-file journals remain readable, and cancellation is distinct from deadline expiry.

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
| Temporal tier 2 | 16 | 5 | 11 | Yes | Scheduled or manually dispatched |

The eleven unsupported Temporal cases are useful blocker evidence, not support. Full runner qualification remains limited to Go 1.26.4 on `darwin/arm64`. PROD-1, PROD-2, and the static-seed filesystem slice of PROD-4 are implemented: campaigns have an explicit crash-consistent lifecycle, bounded segmented journals, portable prepared-target and mount bundles, deterministic ordinal shards, and identity-checking aggregate merge. The distributed-system simulation parity contract, application harness, and SIM-1 through SIM-5 are implemented. Logical nodes now have exact lifecycle, network, storage, fault, scenario, history, observation, oracle, output, and terminal replay; modeled partition/restart and partial-persistence nemeses; bounded inspection; stable failure identities; and a representative Temporal duplicate-delivery failure reproduction. The process backend adds bounded private bootstrap and model IPC, host-owned time arbitration, fresh package initialization, hard crash/reap, and cross-backend detached-model conformance.

The first real local functional gate under `tests/` is now executable.
`tests/gomadfunctional.TestFrontendSystemInfo` starts the normal one-box
harness with in-memory SQLite, static membership, and unused telemetry resource
detection disabled, then completes a frontend RPC under guarded Gomad execution.
The target may link optional process, signal, telemetry, cloud-archiver,
Elasticsearch, Cassandra, SDK, and worker-provider paths, but any unmodeled
forbidden API still terminates at runtime. The only signal operation modeled
for this workload is the deterministic no-op `os/signal.Stop` cleanup path.
Closure analysis remains intentionally stricter than this guarded execution
claim. AWS, GCP, Kubernetes control planes, credential discovery, and
external-service emulation remain explicit non-goals.

## Recommended order

### Milestone 0: restore the contract — complete

The replay, resume, publication, TCP, cleanup, qualification, cancellation, journal-limit, and network-resource corrections are implemented with adversarial regression coverage. Preserve these as required gates for every later capability.

### Milestone 1: make executions observable and compatibility measurable — complete

The two initial functions are implemented:

- [BUG-1](GOMAD3_NEXT_BUG_FINDING.md#bug-1-runtime-choice-trace), [BUG-2](GOMAD3_NEXT_BUG_FINDING.md#bug-2-choice-coverage-and-feedback), and [BUG-3](GOMAD3_NEXT_BUG_FINDING.md#bug-3-exact-choice-tape-replay): bounded runtime-choice tracing, choice coverage, artifact retention, `inspect --choices`, and exact replay;
- [COMPAT-1](GOMAD3_NEXT_COMPATIBILITY.md#compat-1-gomad-analyze), [COMPAT-2](GOMAD3_NEXT_COMPATIBILITY.md#compat-2-unambiguous-support-matrices), and the initial [COMPAT-3](GOMAD3_NEXT_COMPATIBILITY.md#compat-3-tiered-temporal-corpus): read-only compatibility analysis, tiered qualification manifests, explicit support counts, support comparison, and boundary-diff approval.

These are now the evidence sources for controlled exploration, compatibility prioritization, and simulation qualification.

### Milestone 2: explore controlled schedules and validate the simulation seam — in progress

[BUG-4](GOMAD3_NEXT_BUG_FINDING.md#bug-4-bounded-alternative-prefix-exploration) is implemented as the durable bounded choice-frontier strategy. The pinned equal-budget comparison currently shows no outcome-efficiency advantage over seed sampling, so BUG-5 and later search refinements remain gated on stronger benchmark evidence rather than being added speculatively.

Continue [COMPAT-3](GOMAD3_NEXT_COMPATIBILITY.md#compat-3-tiered-temporal-corpus) by using the sixteen-workload Temporal tier-2 corpus to rank exact compatibility packs and adapters by workloads unlocked. [COMPAT-4](GOMAD3_NEXT_COMPATIBILITY.md#compat-4-compatibility-pack-development-kit) is implemented with a v2-only exact-source contract, exact-digest approval, generated changed-version/source rejection tests, and independent qualification. The first evidence-ranked [COMPAT-5](GOMAD3_NEXT_COMPATIBILITY.md#compat-5-targeted-deterministic-adapters-and-io-models) slice is complete: an exact `google.golang.org/grpc@v1.80.0` adapter removes the meaningless host keepalive callback from Gomad's virtual TCP path, and `temporal-backoff-overflow` now executes and exactly replays in closure mode. The baseline is 5/16 with no generic exemption or host fallback. [COMPAT-6](GOMAD3_NEXT_COMPATIBILITY.md#compat-6-safer-handling-of-transitive-forbidden-dependencies) has an experimental compiler/linker-backed mode and now accepts an explicit bounded analysis timeout for large targets. Guarded execution is proven on the first local functional workload while preserving runtime termination for unmodeled forbidden calls; closure review remains the default for support claims. Next qualify exact successful replay for that gate, add it to the checked Temporal corpus, and rank further local workloads by the exact packs or modeled boundaries they require.

For simulation, [SIM-0](GOMAD3_NEXT_SIM.md#sim-0-restore-trust-and-define-the-parity-contract) through [SIM-5](GOMAD3_NEXT_SIM.md#sim-5-process-backed-fidelity-tier) are complete. The canonical v2 behavioral-parity manifest names thirteen implemented v3 cases with sixteen declared in-process and process prototypes. Runtime domains bind every modeled network and volume operation to an exact incarnation; lifecycle, terminal, transition, fault, scenario, history, observation, oracle, and final model state replay exactly. The process tier provides the fresh-global and hard-isolation evidence that the in-process tier intentionally cannot claim.

### Milestone 3: reduce failures and make campaigns durable

[PROD-1](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-1-crash-consistent-batch-store) and [PROD-2](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-2-segmented-bounded-journals) are complete: the store owns explicit lifecycle and locked recovery, while new batch-plan v5 and batch v3/v4 records bind immutable bounded segments, partial-run and artifact ceilings, and typed capacity outcomes. Historical batch v1/v2 and interrupted plan v1-v4 readers remain covered. The first [BUG-5](GOMAD3_NEXT_BUG_FINDING.md#bug-5-failure-minimization) slice is implemented for exact combined simulation failures: `gomad minimize` runs bounded fresh-process suffix, forced-range, and fault-entry reductions, preserves exact choice/simulation replay and normalized failure identity, keeps the parent immutable, and publishes inspectable lineage. Durable minimizer checkpoint/resume and typed scenario shrinkers remain before BUG-5 is complete.

The static-seed filesystem slice of [PROD-4](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-4-deterministic-campaign-plans-sharding-and-merge) is complete. Canonical plans capture verified targets and bounded read-only inputs, zero-based ordinal-modulo shards retain global ordinals, and merge validates exact plan identity, completeness, content deduplication, and capacity before publishing a new aggregate. Dynamic choice-frontier sharding remains gated on a round coordinator because prefixes are discovered by prior executions. The initial [PROD-8](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-8-resource-control-and-performance) slice enforces journal, simultaneous partial-run, artifact, and merge capacities with explicit infrastructure outcomes.

In the simulation track, [SIM-1](GOMAD3_NEXT_SIM.md#sim-1-cluster-core-and-in-process-runtime-domains) through [SIM-5](GOMAD3_NEXT_SIM.md#sim-5-process-backed-fidelity-tier) are complete. Preserve their lifecycle, runtime-domain, network, storage, fault, scenario, history, oracle, replay, output, capacity, disabled-mode, process-isolation, and cross-backend conformance gates while SIM-6 integrates controlled exploration.

### Milestone 4: add modeled faults, simulation parity, and a second qualified platform

Introduce [BUG-6](GOMAD3_NEXT_BUG_FINDING.md#bug-6-deterministic-fault-plans) for explicit World and simulation adapters, starting with delay, cancellation, injected errors, dropped modeled delivery, and declared capacity outcomes.

[SIM-2](GOMAD3_NEXT_SIM.md#sim-2-virtual-network-parity), [SIM-3](GOMAD3_NEXT_SIM.md#sim-3-durable-volume-parity), and [SIM-4](GOMAD3_NEXT_SIM.md#sim-4-scenarios-nemeses-records-and-oracles) are complete with node-aware topology, dependency-aware durability, typed scenarios and faults, stable histories and oracles, bounded records, and exact replay. Keep their model identities and deep interfaces separate while platform qualification proceeds.

In parallel, continue [COMPAT-7](GOMAD3_NEXT_COMPATIBILITY.md#compat-7-platform-bundles) with a Linux platform bundle and run the same core and Temporal qualification contracts there. Platform work must remain independent of simulation semantics so neither can accidentally grant the other a support claim.

Avoid transparent network partitions outside the declared cluster model and avoid multi-P scheduling until the simpler event model, platform abstraction, and evidence contracts have proven stable.

### Milestone 5: qualify releases and process-backed simulation

Implement [PROD-3](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-3-artifact-lifecycle-and-data-policy), covering quotas, retention, reachability, pruning, sensitivity, export, mount and environment capture, and replay inputs that must be resupplied rather than retained. Complete [PROD-5](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-5-immutable-release-and-installation-bundles) with signed checksums or attestations, qualification evidence, provenance, SBOM/notices, installation metadata, and rollback support.

Implement [PROD-6](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-6-ci-integration), a supported CI entry point for planning, sharding, caching, bounded artifact upload, merging, exact reruns, support-baseline gating, and outcome classification. Complete [PROD-7](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-7-observability-and-reporting) with `gomad report`, stable aggregate JSON, metrics adapters, and external trend reporting. Finish [PROD-8](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-8-resource-control-and-performance) with the prepared-target cache only after resource correctness, keyed by the complete immutable execution identity and revalidated on every hit.

Complete [COMPAT-8](GOMAD3_NEXT_COMPATIBILITY.md#compat-8-dependency-and-go-upgrade-impact-reports) by extending dependency and Go upgrade dossiers to report changed targets, packs, adapters, workload support, boundary dispositions, approvals, and rollback availability. Implement [PROD-9](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-9-release-governance) by defining schema/CLI compatibility windows, evidence-preserving migrations, release ownership, qualification state, known limitations, maturity labels, and rollback targets.

[SIM-5](GOMAD3_NEXT_SIM.md#sim-5-process-backed-fidelity-tier) is complete after the in-process network, storage, lifecycle, and oracle semantics passed their parity corpus. The process-backed fidelity tier owns bounded node launch, typed model IPC, time arbitration, hard crash/reap, and fresh-incarnation initialization; only this tier claims fresh package globals and hard process isolation across node restarts.

### Milestone 6: integrate simulation exploration and expand by evidence

Complete [SIM-6](GOMAD3_NEXT_SIM.md#sim-6-controlled-schedule-and-fault-exploration). Runtime-choice, scenario, network, storage, crash-state, and fault decisions now share one bounded, round-atomic frontier with exact artifact projection, resume without committed-candidate rediscovery, evidence-only semantic deduplication, and bounded schedule-plus-fault minimization. Remaining exit work is a checked controlled-versus-seed benchmark for the combined frontier and crash-resumable minimizer state.

Complete [SIM-7](GOMAD3_NEXT_SIM.md#sim-7-evidence-driven-expansion-beyond-v2) by running the Temporal corpus to rank beyond-v2 simulation gaps by workloads unlocked. Add only the smallest deep model justified by named consumers, with a semantic contract, host-escape canary, exact replay, capacity and failure tests, and performance evidence. Do not pursue UDP, DNS, routing, corruption, resource-pressure, or external-service breadth for API-count parity.

### Milestone 7: evaluate research extensions

Use [BUG-7](GOMAD3_NEXT_BUG_FINDING.md#bug-7-later-research-extensions) evidence from the earlier milestones to decide whether PCT, preemption bounding, semantic dependency metadata and DPOR, compiler checkpoints, deterministic GC, code-coverage guidance, richer I/O models, or multi-P execution have sufficient expected value. These are expensive and potentially contract-changing; none should be assumed mandatory.

## Complete track coverage

The tables below account for every named capability and delivery stage in the four detailed roadmaps. The companion documents remain normative for their non-goals, module boundaries, protocols, error classifications, 10×-load behavior, trade-offs, verification plans, and exit criteria.

Status legend: ✅ complete · 🟡 in progress or partial · 🧪 experimental · ⏳ planned · ⏸️ evidence-gated or deferred.

### Bug-finding power

| Capability | Status | Portfolio placement |
| --- | --- | --- |
| [BUG-1: Runtime-choice trace](GOMAD3_NEXT_BUG_FINDING.md#bug-1-runtime-choice-trace) | ✅ Complete | Milestone 1 |
| [BUG-2: Choice coverage and feedback](GOMAD3_NEXT_BUG_FINDING.md#bug-2-choice-coverage-and-feedback) | ✅ Complete | Milestone 1 |
| [BUG-3: Exact choice-tape replay](GOMAD3_NEXT_BUG_FINDING.md#bug-3-exact-choice-tape-replay) | ✅ Complete | Milestone 1 |
| [BUG-4: Bounded alternative-prefix exploration](GOMAD3_NEXT_BUG_FINDING.md#bug-4-bounded-alternative-prefix-exploration) | ✅ Complete; pinned equal-budget comparison is neutral | Milestone 2 |
| [BUG-5: Failure minimization](GOMAD3_NEXT_BUG_FINDING.md#bug-5-failure-minimization) | 🟡 Partial; exact combined-failure reducer is complete, durable minimizer resume and typed scenario shrinkers remain | Milestone 3 |
| [BUG-6: Deterministic fault plans](GOMAD3_NEXT_BUG_FINDING.md#bug-6-deterministic-fault-plans) | ⏳ Planned | Milestone 4 |
| [BUG-7: Later research extensions](GOMAD3_NEXT_BUG_FINDING.md#bug-7-later-research-extensions) | ⏸️ Deferred pending evidence | Milestone 7 |

### Temporal and platform compatibility

| Capability | Status | Portfolio placement |
| --- | --- | --- |
| [COMPAT-1: `gomad analyze`](GOMAD3_NEXT_COMPATIBILITY.md#compat-1-gomad-analyze) | ✅ Complete | Milestone 1 |
| [COMPAT-2: Unambiguous support matrices](GOMAD3_NEXT_COMPATIBILITY.md#compat-2-unambiguous-support-matrices) | ✅ Complete | Milestone 1 |
| [COMPAT-3: Tiered Temporal corpus](GOMAD3_NEXT_COMPATIBILITY.md#compat-3-tiered-temporal-corpus) | 🟡 In progress; tiers 1 and 2 are complete and the first local functional probe executes successfully in guarded mode | Milestones 1–2 |
| [COMPAT-4: Compatibility-pack development kit](GOMAD3_NEXT_COMPATIBILITY.md#compat-4-compatibility-pack-development-kit) | ✅ Complete with v2-only exact-source packs and exact-digest approval | Milestone 2 |
| [COMPAT-5: Targeted deterministic adapters and I/O models](GOMAD3_NEXT_COMPATIBILITY.md#compat-5-targeted-deterministic-adapters-and-io-models) | 🟡 Ongoing, evidence-driven | Milestones 2 and 4 |
| [COMPAT-6: Safer handling of transitive forbidden dependencies](GOMAD3_NEXT_COMPATIBILITY.md#compat-6-safer-handling-of-transitive-forbidden-dependencies) | 🧪 Experimental guarded compiler/linker mode executes the first real workload; runtime guards remain fail-closed and closure remains the default support claim | Milestone 2 |
| [COMPAT-7: Platform bundles](GOMAD3_NEXT_COMPATIBILITY.md#compat-7-platform-bundles) | 🟡 Partial; `darwin/arm64` is qualified and Linux is planned | Milestones 4–5 |
| [COMPAT-8: Dependency and Go upgrade impact reports](GOMAD3_NEXT_COMPATIBILITY.md#compat-8-dependency-and-go-upgrade-impact-reports) | 🟡 Partial; boundary approval exists and the full impact report is planned | Milestone 5 |

### Productionization

| Capability | Status | Portfolio placement |
| --- | --- | --- |
| [PROD-1: Crash-consistent batch store](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-1-crash-consistent-batch-store) | ✅ Complete with explicit lifecycle, locked recovery, interrupted inspection, store-owned resume preflight, and mutation-fault matrices | Milestone 3 |
| [PROD-2: Segmented, bounded journals](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-2-segmented-bounded-journals) | ✅ Complete with batch-plan v5, batch v3/v4, immutable indexed segments, historical readers, typed capacities, and crash recovery | Milestone 3 |
| [PROD-3: Artifact lifecycle and data policy](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-3-artifact-lifecycle-and-data-policy) | ⏳ Planned | Milestone 5 |
| [PROD-4: Deterministic campaign plans, sharding, and merge](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-4-deterministic-campaign-plans-sharding-and-merge) | 🟡 Partial; static-seed filesystem v1 is complete and dynamic choice-frontier distribution is planned | Milestone 3 |
| [PROD-5: Immutable release and installation bundles](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-5-immutable-release-and-installation-bundles) | 🟡 Partial; installation discovery exists and qualified bundles are planned | Milestone 5 |
| [PROD-6: CI integration](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-6-ci-integration) | 🟡 Partial; qualification workflow exists and a supported campaign entry point is planned | Milestone 5 |
| [PROD-7: Observability and reporting](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-7-observability-and-reporting) | 🟡 Partial; stable events exist and aggregate reporting and metrics are planned | Milestone 5 |
| [PROD-8: Resource control and performance](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-8-resource-control-and-performance) | 🟡 Partial; journal, partial-run, artifact, and merge bounds are complete, while load evidence and a prepared-target cache are planned | Milestones 3 and 5 |
| [PROD-9: Release governance](GOMAD3_NEXT_PRODUCTIONIZATION.md#prod-9-release-governance) | 🟡 Partial; boundary approval and current readers exist, while explicit compatibility and release policy are planned | Milestone 5 |

### Distributed-system simulation

| Delivery stage | Status | Portfolio placement |
| --- | --- | --- |
| [SIM-0: Restore trust and define the parity contract](GOMAD3_NEXT_SIM.md#sim-0-restore-trust-and-define-the-parity-contract) | ✅ Complete; thirteen named parity cases and bounded backend/fidelity claims | Milestone 2 |
| [SIM-1: Cluster core and in-process runtime domains](GOMAD3_NEXT_SIM.md#sim-1-cluster-core-and-in-process-runtime-domains) | ✅ Complete; in-process lifecycle fidelity only | Milestone 3 |
| [SIM-2: Virtual network parity](GOMAD3_NEXT_SIM.md#sim-2-virtual-network-parity) | ✅ Complete; in-process network fidelity only | Milestone 4 |
| [SIM-3: Durable volume parity](GOMAD3_NEXT_SIM.md#sim-3-durable-volume-parity) | ✅ Complete; in-process durability fidelity only | Milestone 4 |
| [SIM-4: Scenarios, nemeses, records, and oracles](GOMAD3_NEXT_SIM.md#sim-4-scenarios-nemeses-records-and-oracles) | ✅ Complete; in-process typed faults, scenarios, histories, oracles, artifacts, and Temporal failure replay | Milestone 4 |
| [SIM-5: Process-backed fidelity tier](GOMAD3_NEXT_SIM.md#sim-5-process-backed-fidelity-tier) | ✅ Complete; bounded process launch, model/time IPC, hard isolation, and cross-backend conformance | Milestone 5 |
| [SIM-6: Controlled schedule and fault exploration](GOMAD3_NEXT_SIM.md#sim-6-controlled-schedule-and-fault-exploration) | 🟡 Partial; combined frontier, durable rounds, exact evidence, semantic deduplication, and the first minimizer slice are complete, while benchmark and minimizer-resume gates remain | Milestone 6 |
| [SIM-7: Evidence-driven expansion beyond v2](GOMAD3_NEXT_SIM.md#sim-7-evidence-driven-expansion-beyond-v2) | ⏸️ Evidence-gated; designed to follow measurable parity | Milestone 6 |

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
