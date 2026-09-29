# Gomad: configurable virtual-clock tick

## Goal & Context
<!-- scope: business -->

Under Gomad, virtual time advances only when no goroutine is runnable, so every event inside one
busy stretch carries the same timestamp. Several functional tests implicitly rely on distinct
wall-clock times and fail under Gomad only because of such ties (equal-start-time SQLite
visibility listings, OTEL spans sorted by `StartTime`, same-instant describe results); F7 records
them as named subtest exclusions. A deterministic, opt-in clock tick removes the ties without
giving up determinism, while the strict default keeps Gomad surfacing real tie bugs (the update
admission ordering was one).

## Architecture & Data Models
<!-- scope: technical -->

A deterministic profile option advances the process virtual clock by a configured quantum at a
configured point: per `time.Now`/monotonic read, or per goroutine wake-up. The tick count is a
function of the program and the recorded scheduling decisions only, so same-seed runs and replay
stay exact. The option is part of the deterministic profile and the Campaign/Artifact identity; a
qualification manifest can set it per workload.

## Edge Cases & Constraints
<!-- scope: technical -->

- Default is off; the existing virtual-time contract is unchanged when off.
- With a tick, timers may come due while work is runnable; the contract documents the resulting
  delivery order, and it stays deterministic.
- COMPAT-5 evidence set: contract, bound (maximum drift per run), transcript/tape coverage, exact
  replay, negative test.
- Replay of an artifact recorded with a tick requires the same tick; a mismatch fails before
  execution.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `--clock-tick=<duration>` and `--clock-tick-at=read|wake` (names may differ; both
  configurable) on explore/qualify/qualify-set and in the manifest per workload, recorded in
  profile and artifact identity. Errors: invalid durations, negative, or above a documented bound
  are rejected; replay with a mismatched tick fails closed.
- **R2:** Same-seed repetitions and replay stay exact with the tick on (runtime fixture + core
  workload proving it); with the tick off, every existing gate is unchanged.
- **R3:** Running the tie-excluded suites with the tick on shows which named
  exclusions it removes; suites whose ties are demonstrated test bugs switch to the tick in the
  manifest and their exclusions are removed.
- **R4:** README/ARCHITECTURE document the option, its contract, and why the default stays strict.

## Boundaries
<!-- scope: business -->

- Not a fix for GC-layout or scheduling divergences.
- Not a default change.

## Decision Context
<!-- scope: both -->

2026-09-29, user request: bump virtual time minimally per goroutine wake-up, and make it
configurable. Per-read ticks make every read distinct; per-wake ticks are cheaper but two reads in
one slice still tie — both are offered.
