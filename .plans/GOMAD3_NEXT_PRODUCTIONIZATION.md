# Gomad v3 remaining productionization work

Make Gomad economical and dependable for repeated developer and CI use.
The [README](../tools/gomad3/README.md) documents implemented campaign recovery,
bounded journals, portable static plans, sharding/merge, prepared-target caching,
qualification pruning, and disk preflight. This document covers extensions beyond
those features. Gomad continues to execute trusted test code.

## PROD-3: Artifact lifecycle and data policy

Define a versioned store-wide policy for retention age, total bytes, reachability,
sensitivity, mount/environment capture, and permitted exports. Existing per-run
bounds and qualified-artifact pruning do not answer those store-wide questions.

A pruning workflow needs dry-run, age/quota rules, and reachability checks for
campaigns and original/minimized artifact lineage. Export must validate and
inventory every retained payload. Store verification and usage summaries should
consume the same validated ownership data.

If secrets may be supplied without retention, record their names and digest or
requirements and require resupply on replay. Label these artifacts as requiring
external input. Self-contained replay requires every input to be retained under
an explicitly permitted storage policy. Use private files and approved encrypted
storage; keep data-policy decisions outside Runner execution semantics.

## PROD-4: Deterministic campaign plans, sharding, and merge

Dynamic choice/combined frontier distribution remains distinct from static seed
sharding and qualification-set sharding. A round coordinator must prove unique
candidate ownership, durable round commit, complete global ordinals, and resume
without duplicate logical work. A remote scheduler consumes validated plan/shard
protocols without changing evidence or Runner semantics.

## PROD-5: Immutable release and installation bundles

Publish qualified platform bundles with binary/toolchain/profile identities,
source provenance, reviewed packs/adapters, qualification reports, attestations,
SBOM/notices, minimum host requirements, and install/rollback/uninstall metadata.
Stage and verify before atomic activation; keep the prior qualified bundle
addressable and support offline verification after acquisition.

Exercise help/version/doctor behavior on a clean host, separating read-only
installation inspection from explicit storage probes. Target execution never
resolves an unpinned latest toolchain.

## PROD-6: CI integration

Provide a supported campaign orchestration entry point over the existing CLI,
plans, shards, caches, and merge. Emit bounded summaries by default, export full
artifacts only under policy, and replay failures from retained binaries.

Compare support, runtime, divergence, artifact cost, and failure signatures with
a declared baseline. Checks distinguish target failures, unsupported targets,
watchdogs, cancellation, capacity, replay divergence, and infrastructure errors.
Increasing expected unsupported coverage cannot conceal a support regression.

## PROD-7: Observability and reporting

Add aggregate campaign reports over validated typed events and evidence. Include
throughput, active runs, outcomes, stop reasons, preparation/execution/replay/store
durations, bytes by category, blocker groups, reproduction status, and recovery
counts. JSON and human reports share one projection; metrics backends consume it
through adapters. Trend storage stays outside immutable artifacts.

## PROD-8: Resource control and performance

Measure process, descriptor, memory, disk, network, preparation, frontier, and
minimizer costs on representative clean-host and multi-host soak workloads.
Bound independent resource owners and use backpressure. Report capacity exhaustion
and partial completion explicitly.

Preserve complete source/build/toolchain/pack/adapter/profile cache identities and
revalidate cached binaries. Every execution still uses a fresh process and private
run directory. Compare overhead, artifact growth, and cleanup at ten times the
representative workload size before widening concurrency or quotas.

## PROD-9: Release governance

Declare schema-reader windows and CLI compatibility. Writers emit current formats;
migrations publish new evidence instead of rewriting artifacts. Releases record
owner, qualification evidence, approved boundary differences, known limitations,
and rollback target.

Include the nested Go module in lint, vet, vulnerability, and license checks.
Separate build success, expectation matching, actual support, and release approval.
A release gate needs clean-host installation, storage crash/fault matrices,
wrong-identity rejection, sensitive-input tests, and retained qualification evidence.

## Ownership

Storage owns lifecycle and recovery; policy owns retention/export; plans own work
identity; merge owns aggregate validation; release tooling owns bundle governance;
reports own projections. The Runner composes these modules through small interfaces
and does not own their transactions or external services.
