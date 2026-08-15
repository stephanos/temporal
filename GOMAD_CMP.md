# Gomad v1, v2, and v3 comparison

Gomad v3 is not a direct successor to v2; it is effectively a different
product. V2 is the strongest distributed-systems simulator. V3 is a
deterministic concurrency runner with a much larger replay, evidence,
qualification, and toolchain-management platform around it.

| | v1 | v2 | v3 |
| --- | --- | --- | --- |
| Core approach | AST rewriting plus replacement libraries | Typed translation into a custom runtime and simulated OS | Patched Go compiler/runtime; unchanged target source |
| Engine complexity | High and ad hoc | High but conceptually cohesive | Smaller runtime core, highest total-system complexity |
| Maintainability | Worst | Best capability/complexity balance | Strong local modules and tests; expensive whole-system upgrades |
| Main strength | Temporal-specific experimentation | Multi-machine network and disk simulation | Repeatable campaigns and exact evidence/replay |
| Main weakness | Brittle replacement and skip matrix | Reimplements much of Go and Linux | Narrow simulation capabilities and very large operational surface |

## Gomad v1

V1 rewrites dependencies and redirects language and library operations into
`SIMAPI` and `SIMLIB`. Its large package-skip matrix illustrates the fundamental
maintenance problem: type mismatches or unsupported syntax are handled through
increasingly specific exceptions in
[`tools/gomadv1/transformer/transform.go`](tools/gomadv1/transformer/transform.go).

Its capabilities include seeded cooperative scheduling, channels and `select`,
virtual time, maps, synchronization, fake HTTP/gRPC/network components, and
experimental checkpoint/restore.

Its complexity is mostly accidental:

- 42,397 lines are copied replacement libraries under `api/ext-lib`.
- Compatibility depends on higher-level API substitution.
- Skipped code can retain native behavior, weakening determinism.
- The 1,471-line AST transformer and replacement tables concentrate fragility.

V1 is the hardest implementation to trust and maintain.

## Gomad v2

V2 has the clearest simulator architecture: a custom deterministic runtime,
standard-library hooks, and a simulated Linux OS, as described in
[`tools/gomadv2/docs/design.md`](tools/gomadv2/docs/design.md). It translates
goroutines, channels, maps, globals, and synchronization, while its OS models
TCP, filesystems, `fsync` and crash behavior, and separate simulated machines.

This gives it the broadest distributed-systems capabilities:

- Multiple machines with independent globals, disks, and addresses.
- Crash, restart, partial disk persistence, partitions, and latency.
- Simulated TCP and POSIX-like filesystem behavior.
- Seeded scheduling, virtual time, trace checksums, and multi-seed testing.

The complexity is substantial but mostly inherent: roughly 6.3K production
lines in translation, 11.8K in simulation, and 3.8K in the runtime. Its main
maintenance risks are compiler and AST evolution, unexported standard-library
hooks, global rewriting, and copied `reflect` and `testing` compatibility code.

For deterministic distributed-systems simulation, v2 currently has the best
capability-to-complexity ratio.

## Gomad v3

V3 patches a pinned Go runtime and compiler instead of translating application
source. It uses the real runtime scheduler and timer heaps, one P, virtual time,
and fresh processes per seed. The supported execution contract is documented in
[`tools/gomadv3/README.md`](tools/gomadv3/README.md).

Its strongest capabilities are operational:

- `explore`, `qualify`, `resume`, `inspect`, `replay`, and `doctor`.
- Parallel seed campaigns and configurable failure policies.
- Bounded, content-addressed artifacts.
- Exact binary, input, and I/O transcript replay.
- Semantic coverage and a guided corpus.
- Dependency-closure review and fail-closed unsupported operations.
- Deterministic entropy, an in-memory filesystem, read-only mounts, basic
  loopback TCP, and a version-pinned libc adapter.

V3 is not a capability superset of v2. It has no simulated machines,
partitions, packet loss, distributed disks, or realistic network fault model.
`World` is an explicit event-modeling library with a mailbox pilot, not a
general simulated OS. It is also currently tied to pinned Go and qualified
`darwin/arm64` execution.

## Why v3 is so large

### Source size

Source-wise, v3 is only modestly larger than v2:

| Current tracked source | v1 | v2 | v3 |
| --- | ---: | ---: | ---: |
| Total lines | 58.8K | 48.8K | 57.4K |
| Go lines | 57.6K | 44.4K | 49.4K |
| Go test lines | 2.9K | 14.4K | 15.3K |
| Production package directories | 33 | 21 | 53 |

The Go patch itself is only 323 lines, and the overlay is approximately 5K
lines. Most v3 source bulk instead comes from:

- 23.1K production and 12.6K test lines under `internal`.
- Artifact journaling, process supervision, containment, replay, and resume.
- Target provenance and dependency-closure validation.
- Qualification sets, upgrade dossiers, toolchain builders, and generators.
- Guided-corpus management and semantic coverage.
- A 4,174-line reviewed boundary manifest containing 129 intercepts: 62 modeled,
  64 denied, and 3 delegated.
- Forty small typed packages under `internal`, compared with twelve in v2.

Some of this size is justified by v3's stronger replay and integrity guarantees.
Some appears premature relative to its narrow runtime and I/O capabilities,
particularly guidance, extensive qualification and upgrade machinery, and
packaging infrastructure before the supported workload is broad.

### Local disk size

The much more visible size difference is generated local state:

| Directory | Current size |
| --- | ---: |
| `tools/gomadv1` | 2.2 MB |
| `tools/gomadv2` | 1.1 GB |
| `tools/gomadv3` | 2.5 GB |

Almost all of v3's 2.5 GB is its ignored `.toolchain` directory:

- 1.3 GB for five immutable custom Go builds at approximately 260 MB each.
- 943 MB for ten retained Temporal qualification runs at approximately 94 MB
  each.
- 256 MB of generator and build cache.
- Approximately 40 MB of downloads and compiler-test material.

These are ignored build outputs rather than committed source. The disk growth
mainly reflects repeated complete toolchain builds and replayable target
artifacts without an automatic retention or garbage-collection policy. V2 has a
similar, though smaller, issue: almost all of its 1.1 GB is the ignored
`.gomad` cache and generated test state.

## Assessment

V2 is the better simulator; v3 is the better reproducibility and evidence
platform. V3 is large because it attempts to be a hermetic toolchain, campaign
runner, artifact database, replay system, compatibility auditor, and
qualification framework simultaneously, while its simulated-world capabilities
remain narrower than v2's.
