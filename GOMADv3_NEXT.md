# Gomad v3: Suggested Next Functionality

**Roadmap date:** 2026-08-13

## Purpose

Based on [GOMADv3_SUMMARY.md](GOMADv3_SUMMARY.md), Gomad v3 has enough runtime, evidence, and replay machinery to support useful experiments, but it is not yet broad, powerful, or reliable enough for default Temporal use. The next work should pursue three independent goals:

| Goal | Question it answers | Roadmap |
| --- | --- | --- |
| Bug-finding power | Can Gomad find failures that ordinary seeded repetition misses, explain them, and reduce them? | [GOMADv3_NEXT_BUG_FINDING.md](GOMADv3_NEXT_BUG_FINDING.md) |
| Compatibility | Can useful Temporal workloads run without weakening the deterministic contract? | [GOMADv3_NEXT_COMPATIBILITY.md](GOMADv3_NEXT_COMPATIBILITY.md) |
| Productionization | Can developers and CI operate Gomad safely, repeatably, and at scale? | [GOMADv3_NEXT_PRODUCTIONIZATION.md](GOMADv3_NEXT_PRODUCTIONIZATION.md) |

These are separate investment tracks. Success in one does not imply success in the others: a stronger explorer is not useful if representative code cannot run, and broader compatibility should not be called production-ready without crash recovery, release provenance, and data-handling controls.

The goal documents are strategy and capability designs. Each capability is intended to become its own implementation plan after its predecessor meets the stated exit criteria; they should not be executed as one monolithic project.

## Shared entry gate

New functionality should not be built on known false guarantees. The following corrections are prerequisites for relying on new capabilities:

1. Send the recorded World replay plan to the child and reject divergence before applying a transition.
2. Make resumed journals valid when multiple runs share one deduplicated failure artifact.
3. Make batch publication crash-consistent across plan removal and final manifest publication.
4. Correct virtual TCP close/data ordering and its related readiness races.
5. Restrict process cleanup to independently trusted target identities.
6. Separate `expectations_met` from actual supported/unsupported counts in qualification output.
7. Align journal writer and reader limits, distinguish cancellation from timeout, and add explicit network resource exhaustion.

This is not a demand to perfect every subsystem before learning anything new. It is a narrow trust gate: replay, resume, publication, and qualification labels must mean what they claim before other features depend on them.

## Recommended order

### Milestone 0: restore the contract

Complete the shared entry gate and add adversarial tests around every corrected boundary. Keep artifact formats backward-readable where practical; otherwise introduce a new schema rather than silently changing meaning.

### Milestone 1: make executions observable and compatibility measurable

Build two small, high-leverage functions in parallel:

- a bounded runtime-choice trace with artifact and `inspect` support;
- a read-only `gomad analyze` command that explains target compatibility blockers and dependency paths.

Choice tracing is the foundation for controlled exploration and minimization. Compatibility analysis identifies which workload and platform investments will unlock actual Temporal value. Neither requires prematurely building a full systematic scheduler or broad I/O simulator.

### Milestone 2: control schedules and establish a real Temporal corpus

Add exact choice-tape replay and bounded prefix exploration. At the same time, expand qualification from five examples into tiered Temporal workloads and publish support coverage separately from expectation matching.

The decision to expand compatibility packs, I/O models, or platforms should be driven by the blockers observed in this corpus, not by standard-library API count.

### Milestone 3: reduce failures and make campaigns durable

Add schedule minimization, crash-consistent segmented journals, artifact quotas/pruning, and deterministic campaign sharding. The output of a long campaign should be a small, durable reproduction that another machine with the same platform bundle can inspect and replay.

### Milestone 4: add deterministic faults and a second qualified platform

Introduce bounded World fault plans for explicit adapters, starting with delay, cancellation, and injected error outcomes. In parallel, add a Linux platform bundle and run the same core and Temporal qualification contracts there.

Avoid transparent network partitions or multi-P scheduling until the simpler event model and platform abstraction have proven stable.

### Milestone 5: evaluate research extensions

Use evidence from the earlier milestones to decide whether compiler checkpoints, deterministic GC, code-coverage guidance, richer I/O models, or multi-P execution have sufficient expected value. These are expensive and potentially contract-changing; none should be assumed mandatory.

## Cross-goal dependencies

| Capability | Depends on | Enables |
| --- | --- | --- |
| Choice trace | Bounded shared-memory protocol and artifact schema | Choice coverage, forced replay, minimization |
| Forced choice replay | Choice trace plus pre-apply divergence | Prefix exploration and stable reduced reproductions |
| Compatibility analysis | Existing closure review projected as structured evidence | Corpus prioritization and pack governance |
| Temporal support matrix | Unambiguous qualification semantics | CI gating and platform comparison |
| Platform bundle | Versioned patch, boundary, profile, and host-audit identity | Linux qualification and distributable releases |
| Segmented batch store | Crash-consistent publication and bounded readers | Large campaigns, sharding, pruning, aggregation |
| Fault plan | Correct World replay and explicit adapter semantics | Deterministic failure-path exploration |
| Release bundle | Platform qualification and immutable identities | Repeatable CI and developer installation |

## Investment principles

- **Measure before widening.** Record why real targets are rejected and rank compatibility work by Temporal workload value.
- **Observe before controlling.** A bounded choice trace should precede systematic scheduling or minimization.
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

If these checkpoints are not met, narrow the product rather than accumulating additional runtime patching and compatibility obligations.
