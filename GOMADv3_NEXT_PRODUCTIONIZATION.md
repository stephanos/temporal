# Gomad v3 Next: Productionization

## Goal

Make the existing Gomad capability safe and economical for repeated developer and CI use. “Productionization” here means a dependable internal test service/tool, not enabling Gomad mode in production binaries or accepting hostile target code.

The recommended progression is:

> recover correctly → bound storage and work → distribute immutable bundles → integrate CI → operate and observe

The first deliverable should be a crash-consistent, bounded batch store with explicit recovery.

## What success means

Operators and users should be able to answer:

- Can an interrupted campaign always be inspected, recovered, resumed, or explicitly declared unrecoverable?
- What are the maximum disk, memory, process, and wall-time costs before execution starts?
- Can work be sharded across CI machines and merged without duplicate or mismatched evidence?
- Can a clean machine install the exact qualified toolchain and verify its provenance?
- Which data will an artifact retain, for how long, and how is sensitive input handled?
- Which campaigns are healthy, divergent, timing out, or consuming abnormal resources?
- Can the prior qualified release be restored quickly?

## Non-goals

- Treating Gomad as a production application runtime.
- Treating trusted-process containment as an OS security sandbox.
- Building a distributed scheduler before deterministic plans, shards, and merges work locally.
- Automatically uploading arbitrary artifacts or secrets to a shared service.
- Preserving unlimited backward compatibility while artifact semantics are still evolving.

## Capability 1: crash-consistent batch store

Move batch lifecycle and recovery behind a deep storage module. Runner should submit immutable run completions and a final summary; it should not manage publication ordering itself.

Use an explicit state machine:

```text
planned → prepared → running → committing → published
                     ↘ recoverable-failure
```

Required properties:

- publication writes and syncs the final manifest before deleting data required for resume;
- every state transition is atomic or reconstructible from validated prior state;
- recovery never guesses between two identities;
- duplicate failure artifacts can be referenced by many run records without borrowing the first run's identity;
- cancellation and deadline expiry have distinct terminal states;
- completed evidence is immutable and incomplete staging is clearly separated.

Add operator functions:

- `gomad recover <batch>` validates and repairs only permitted incomplete transitions;
- `gomad inspect <batch>` reports lifecycle state and exact reason it is or is not resumable;
- `gomad resume` delegates all integrity decisions to the batch store.

Crash/fault injection at every file create, sync, rename, and delete boundary is part of the feature, not a later test improvement.

## Capability 2: segmented, bounded journals

Replace the single unbounded `runs.jsonl` contract with a versioned segmented journal or another streaming format that has equivalent inspectability.

The batch plan should declare:

- maximum runs and journal bytes;
- segment byte/record limits;
- retained success, failure, output, transcript, mount, World, and choice-trace quotas;
- maximum partial runs and total artifact bytes;
- behavior at each capacity limit.

Each segment should be immutable after close, independently hashed, and referenced from a compact index/final manifest. Readers validate and stream segments rather than loading an entire campaign into memory. Resume appends a new validated segment and never edits a closed one.

Capacity outcomes should be explicit: stop-before-next-run, publish-partial, discard-optional-success, or infrastructure failure. Defaults should never create a batch that its own reader rejects.

## Capability 3: artifact lifecycle and data policy

Add a versioned artifact policy to every campaign:

- retention duration and maximum store bytes;
- success/failure retention rules;
- stdout/stderr and transcript limits;
- mount capture permissions;
- environment capture mode;
- sensitivity labels;
- permitted export destinations.

Add:

- `gomad gc` or `gomad prune` with dry-run, age, quota, and reachability policies;
- store verification and usage summaries;
- parent/child reachability for original and minimized artifacts;
- an explicit export command that revalidates and inventories retained data.

For environment secrets, support non-retained replay inputs as a distinct mode. The artifact records the name and digest/requirement, not the value, and replay requires the operator to resupply it. Such artifacts should be labeled “replay requires external input,” not exact self-contained replay. Self-contained exact replay should reject values marked secret unless the configured storage policy explicitly permits them.

Do not invent application-level encryption in the first version. Rely on private files and approved encrypted storage, and make export policy explicit.

## Capability 4: deterministic campaign plans, sharding, and merge

Separate campaign planning from execution. A canonical plan should contain target, platform bundle, profile, environment/mount identities, selection, strategy, bounds, and ordinal-to-seed/prefix mapping.

Add functions such as:

```text
gomad plan ... --output campaign.plan.json
gomad run-shard campaign.plan.json --shard=2/8
gomad merge campaign.plan.json shard-*/ --output merged-batch
```

Each shard owns a disjoint canonical ordinal set. Merge validates plan identity, rejects overlapping or missing ordinals unless partial merge is requested, deduplicates evidence by content identity, and publishes a new aggregate without changing shard artifacts.

Start with filesystem artifacts and CI-native distribution. A remote scheduler can later consume the same plan/shard protocol without changing runner semantics.

## Capability 5: immutable release and installation bundles

Publish a versioned platform bundle containing or identifying:

- `gomad` and helper binaries;
- exact Go archive/checksum, patch, overlay, and boundary manifests;
- compatibility packs and deterministic adapters;
- supported platform and minimum host requirements;
- core and Temporal qualification reports tied to the source commit;
- signed checksums or repository-native attestations;
- SBOM/third-party notices;
- install, verify, upgrade, rollback, and uninstall metadata.

Add conventional commands:

- `gomad version` prints source, platform-bundle, toolchain, profile, and schema identities;
- `gomad help` and subcommand help return conventional success status;
- `gomad doctor --read-only` validates availability without writing;
- a separate doctor storage probe makes mutation explicit.

Installation should stage, verify, and atomically activate a bundle. Keep the previous qualified bundle addressable for rollback. Never resolve an unpinned “latest” toolchain during target execution.

## Capability 6: CI integration

Provide a supported CI entry point rather than relying on Makefile knowledge:

- plan and shard campaigns deterministically;
- cache platform bundles and prepared targets by immutable identity;
- upload bounded summaries by default and full artifacts only under policy;
- merge shard results and emit one support/failure summary;
- distinguish unsupported target, target failure, replay divergence, timeout, cancellation, capacity, and infrastructure failure in checks;
- rerun an exact failing artifact without rebuilding its target;
- gate real Temporal qualification on affected Gomad, dependency, and server paths.

Add baseline comparison for support coverage, runtime, divergence, artifact bytes, and failure signatures. Avoid a check that is green merely because expected unsupported counts increased.

## Capability 7: observability and reporting

Keep stable JSON events as the primary machine interface and add an aggregate reporting layer:

- campaign throughput, active runs, outcomes, stop reasons, and queue depth;
- preparation, execution, replay, and publication duration;
- artifact and journal bytes by category;
- support/unsupported counts and blocker groups;
- distinct failure signatures and reproduction status;
- watchdog, divergence, capacity, and recovery counts.

Add `gomad report <batch...>` to produce canonical JSON and a concise human report. Metrics export should be an adapter over the same typed events; the runner should not depend directly on a metrics backend.

Trend data belongs outside immutable artifacts. Artifacts remain evidence; an external collector can aggregate their projections.

## Capability 8: resource control and performance

Make all significant resource limits explicit and enforce them at the owning module:

- global and per-run memory/disk/output/transcript/World/choice bytes;
- process count and file-descriptor budget;
- network listeners, connections, queued bytes, and valid port ranges;
- preparation and execution concurrency;
- journal/frontier/minimizer work budgets.

Use backpressure instead of spawning all work and relying on cancellation. Publish partial, validated summaries when a declared campaign limit is reached.

After correctness, add an immutable prepared-target cache keyed by toolchain, source/build inputs, build tags, environment contract, compatibility packs, adapters, and profile identity. Cache hits must revalidate the prepared binary and capability manifest. Fresh execution processes and per-run work directories remain mandatory.

## Capability 9: release governance

Define supported schema and CLI compatibility explicitly:

- artifact readers support a declared window of older schemas;
- writers emit only the current schema;
- migrations produce new artifacts and never rewrite evidence in place;
- boundary or compatibility changes require an approved baseline diff;
- every release records owner, qualification status, known limitations, and rollback target.

Use maturity labels such as experimental, preview, and qualified-platform. Do not use one boolean to represent build success, expectation matching, support coverage, and release approval.

## Module boundaries

- **Batch store:** owns lifecycle, journal segments, commit, recovery, and streaming read.
- **Artifact policy:** owns quotas, retention, sensitivity, export, and pruning decisions.
- **Campaign plan:** owns canonical work identity and shard partitioning; execution cannot mutate it.
- **Merge:** validates plan/shard identity and publishes aggregate evidence.
- **Platform bundle:** owns release and qualification identity.
- **Event/report layer:** owns typed progress and projections; storage and runner emit events without knowing their consumers.

These should be testable independently with injected filesystem/process boundaries. Runner remains an orchestrator, not the owner of storage transactions, release policy, or metrics.

## Data flow

```text
immutable platform bundle + canonical campaign plan
  → disjoint shard executions
  → crash-consistent local batch stores
  → validated immutable run artifacts
  → identity-checking merge
  → aggregate report and CI result
  → policy-controlled retention, export, or pruning
```

Evidence flows forward as immutable identities. Recovery may complete an interrupted state transition, but it never rewrites a completed run; reporting and retention consume validated projections rather than runner internals.

## Error handling and failure modes

- Disk full during any commit leaves the prior state recoverable and reports the failed transition.
- A corrupt segment invalidates only the containing batch and identifies the segment; it is never skipped silently.
- Shard overlap, wrong plan identity, or conflicting ordinal evidence fails merge.
- Missing secrets or external replay inputs fail preflight before target start.
- Bundle signature/checksum or qualification mismatch prevents activation.
- Metrics/reporting failure cannot alter execution evidence; required output sinks may stop a campaign explicitly.
- Pruning never removes reachable parent artifacts unless the selected policy says so and the dry-run lists them.

At 10× campaign size, readers stream segments, shards cap local work, event sinks apply backpressure or bounded loss policies, and quotas stop work before storage becomes unreadable. Soak tests should cover process churn, file descriptors, port exhaustion, store growth, and repeated recovery.

## Trade-offs

- More durable storage adds schema and fsync cost but is essential for long campaigns.
- Self-contained replay conflicts with secret minimization; the product must label the chosen mode honestly.
- Sharding improves throughput while increasing merge and artifact-distribution complexity.
- Prepared-target caching reduces startup cost while enlarging the identity and invalidation surface.
- Multi-platform release bundles multiply build and audit work; start with the platforms that unblock real CI.
- Broad metrics integrations create dependencies; stable typed events keep the core small.

## Verification plan

1. Filesystem fault injection at every batch-store mutation, followed by recover/inspect/resume assertions.
2. Journal tests beyond 64 MiB proving streaming open, resume, and bounded memory.
3. Duplicate-signature and duplicate-content tests across runs and shards.
4. Merge property tests for partitioning, overlap, missing ordinals, different plans, and deterministic aggregate identity.
5. Artifact-policy tests for quotas, dry-run pruning, reachability, secret requirements, and export inventory.
6. Clean-machine install, verify, upgrade, rollback, and offline tests for each bundle.
7. CI end-to-end tests that rebuild nothing during exact replay and classify every exit condition correctly.
8. Load and soak tests at 10× expected runs, outputs, connections, and artifact volume.
9. Compatibility tests for supported historical artifact schemas and explicit rejection beyond the support window.

## Exit criteria

### Durable campaigns v1

- Every injected storage failure leaves a published or recoverable state.
- Large batches remain inspectable and resumable within declared memory bounds.
- Duplicate failure signatures do not break recovery.

### CI operation v1

- A canonical plan can be split across machines and merged deterministically.
- A failed shard artifact can be replayed without rebuilding.
- Checks distinguish support, target, replay, timeout, cancellation, capacity, and infrastructure outcomes.

### Qualified release v1

- A clean supported host can install and verify an immutable bundle offline after acquisition.
- Bundle identity is present in every artifact and enforced on replay.
- Qualification evidence, boundary approval, SBOM/notices, and rollback metadata are attached to the release.
- Artifact retention and sensitive-data behavior are explicit and tested.

## Recommended first slice

Build the batch-store state machine, segmented bounded journal, and `gomad recover` together. Prove recovery with injected failures before adding sharding, remote artifacts, caches, or dashboards. This removes the largest operational risk and creates the storage substrate those later features require.
