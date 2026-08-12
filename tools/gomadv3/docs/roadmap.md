# Gomad v3 roadmap

Gomad provides a deterministic runtime, bounded multi-seed Runner, versioned
artifacts, exact replay, a pure external-event World, and target-specific
transparent I/O profiles. The remaining work is driven by unchanged Temporal
tests that expose unsupported boundaries or progress failures.

The [architecture](../ARCHITECTURE.md) records stable design decisions. The
[testing backlog](testing-backlog.md) tracks runtime and toolchain coverage;
the [dated functional-suite sweep](2026-08-11-functional-suite-sweep.md) records
integration evidence.

## Read-only repository inputs

Many functional suites first fail because an isolated target cannot read the
SQLite schema tree. Complete the lazy read-only mount so record mode captures
only observed entries through a Runner-owned broker and replay uses only
artifact data.

The mount must remain bounded, symlink-safe, fail closed on unsupported entries
or mutation, reject writes with deterministic filesystem errors, and never put
host source paths into semantic replay identities. After qualification, rerun
the schema-blocked suites to discover their next boundary.

## Native-clock and external-event coordination

Background service goroutines keep some Temporal targets runnable indefinitely,
preventing the runtime's quiescence proof from advancing native timers. First
minimize representative watchdog failures and determine whether they reflect
legitimate runnable work, polling, or an external event that lacks a modeled
completion.

Prefer adapter or driver coordination. Add a runtime hook only when a minimized
case proves that the next logical instant cannot be selected outside the
runtime. Any hook must:

- run only at the existing runtime-proven quiescence point;
- compare the earliest native timer and World event;
- advance exactly one shared logical instant;
- make every native and World event at that instant eligible;
- keep native timers in native heaps and external payloads in World;
- preserve nested synctest behavior; and
- leave runtime and World choice streams independent.

## Adapter expansion

Extend transparent profiles only from observed target boundaries. Reuse shared
implementations while keeping each profile's target, inventory, and replay
identity exact.

Add World-backed adapters when a domain needs asynchronous readiness or
explorable event ordering:

- persistence requests, transactions, conflicts, and deterministic iteration;
- network delivery, loss, partition, cancellation, and closure beyond the
  existing synchronous loopback profile;
- registered deterministic subprocess handlers and lifecycle events; and
- captured environment and application entropy with ownership separate from
  runtime schedule randomness.

Each adapter must own canonical resource identities and versioned snapshots,
use stable bounded errors, deny ambient host capabilities, and have semantic
tests independent of scheduling-order tests.

## Temporal qualification

Continue from the root functional-suite inventory rather than broad repository
runs. For each newly reachable suite:

1. record the first unsupported or non-progress boundary;
2. implement the narrowest shared Gomad capability that covers it;
3. retain an exact target-specific profile identity;
4. require repeated same-seed transcript equality; and
5. require exact replay before calling the suite qualified.

Investigate the exact-selector cluster-router failure separately. Do not change
Temporal tests or production code merely to make isolated Gomad execution pass.

## Exploration and minimization

A seed is an opaque schedule selector and is not numerically minimized. Minimize
reproducible cases instead: target selection, arguments, captured external
input, World events, and injected faults. Delta-debugging candidates must be
rerun enough times to establish that they retain the same deterministic
failure.

Seed and external-event replay remain preferred. Add runtime-choice observation
or forced replay only when a minimized failure cannot be reproduced from the
existing record. Such work requires stable choice identities, a versioned trace
contract, bounded instrumentation, and an explicit pinned-runtime maintenance
review.

## Evidence-gated runtime research

The following are not default next steps:

- domain-separated runtime streams for scheduler, map, and synchronization
  choices;
- allocation-count or explicitly driven deterministic GC;
- compiler-inserted scheduling checkpoints; and
- deterministic multi-P execution.

Each requires a minimized real workload, evidence that user-space coordination
cannot solve it, and a review of compatibility, performance, and security cost.

## Success criteria

Gomad is ready for broader Temporal use when:

- qualified tests receive time and external readiness only through recorded
  deterministic boundaries;
- failures reproduce from immutable target and captured input artifacts without
  live host dependencies;
- a useful seed range scales with bounded processes, memory, output, and disk;
- multiple unchanged Temporal suites exercise meaningful schedule diversity;
- unsupported boundaries fail before host behavior becomes semantic input; and
- additional runtime surface is justified by minimized integration evidence.
