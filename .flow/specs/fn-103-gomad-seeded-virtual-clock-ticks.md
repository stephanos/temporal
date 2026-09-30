# Gomad: seeded virtual-clock ticks

## Goal & Context
<!-- scope: business -->

Under Gomad, virtual time advances only when no goroutine is runnable, so every event inside one
busy stretch carries the same timestamp. Several functional tests implicitly rely on distinct
wall-clock times and fail under Gomad only because of such ties (equal-start-time SQLite
visibility listings, OTEL spans sorted by `StartTime`, same-instant describe results), while other
ties exposed real bugs (the update admission ordering). Making the clock's advance between events a
seeded choice turns time into an exploration dimension like scheduling: different seeds produce
ties, near-ties, and spread-out timestamps, and every seed stays exactly reproducible.

## Architecture & Data Models
<!-- scope: technical -->

A tick policy in the deterministic profile advances the process virtual clock by a drawn amount at
a configured point (per `time.Now`/monotonic read, or per goroutine wake-up). Policies:
- `seeded` (proposed default): a mixture — probability p of 0 (a tie), otherwise a bounded draw
  (e.g. geometric/log-uniform between 1 ns and a small upper bound) — drawn from a dedicated
  stream derived from the seed and a draw counter, never from the scheduling stream;
- `forward`: every draw is at least 1 ns (a bounded seeded draw, never 0), so the clock is
  strictly increasing at the tick point and ties arise only where real systems produce them —
  in layers that truncate timestamps (storage precision, encoders, log formats);
- `fixed=<d>`: a constant quantum;
- `strict`: today's behavior (no tick).
Policy, parameters, and application point are recorded in the profile and in Campaign/Artifact
identity; replay derives the same draws without a tape. A manifest may set the policy per
workload.

## Edge Cases & Constraints
<!-- scope: technical -->

- Changing the default changes every identity; the core, representative, and smoke sets are
  requalified once, and the change carries the COMPAT-5 evidence set (contract, bounds on per-draw
  and cumulative drift, transcript/tape coverage, exact replay, negative tests).
- With ticks, timers may come due while work is runnable; the delivery order is documented and
  deterministic.
- The tick stream must not share state with scheduling, GC, or host-timed runtime draws (lesson of
  the F3/F5 channels).
- A test that relies on distinct timestamps fails on tie-producing seeds; it is fixed upstream,
  pinned to a tie-free policy per workload with a finding, or excluded with the seeds named.

## Scope cut (2026-09-29)
<!-- scope: both -->

Only `forward` (every draw at least 1 ns) and `strict` (today) are implemented here; `forward` alone
removes the known tie failures. `seeded` and `fixed` moved to `fn-105-gomad-follow-ups-deferred-scope`
as D6. The Architecture section still describes all four so D6 has its design.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The tick policy (`forward` and `strict`; `seeded` and `fixed` moved to `fn-105` D6 on
  2026-09-29) and application point are
  configurable on explore/qualify/qualify-set and per workload in manifests, and recorded in profile
  and artifact identity. Errors: invalid parameters or bounds are rejected; replay with a mismatched
  policy fails closed.
- **R2:** Under every policy, same-seed repetitions and replay stay exact (runtime fixture and a
  core workload per policy); `strict` reproduces today's behavior byte-for-byte.
- **R3:** The default policy is decided from measurement: the tie-excluded suites and the smoke
  selection are run under `forward` across several seeds; if the evidence supports
  it, one of them becomes the default (preferring `forward` when it removes the tie failures
  without hiding the known real tie bugs, since real clocks move forward), the qualified sets are requalified, and tie exclusions resolve (fixed,
  pinned, or seed-named); otherwise `strict` stays the default with the evidence recorded.
- **R4:** README/ARCHITECTURE document the policies, the contract, the chosen default and why.

## Boundaries
<!-- scope: business -->

- Not a fix for GC-layout or scheduling divergences.
- No full `./tests` runs (validation is the smoke selection plus affected suites).

## Decision Context
<!-- scope: both -->

2026-09-29, user: bump virtual time minimally per wake-up, configurable; then: maybe make it the
default, with a seed-based distribution for more variation. A seeded mixture that still produces
ties with some probability keeps both bug classes reachable across seeds; strict and fixed remain
available. User follow-up: real time moves forward — added `forward` (minimum 1 ns), where ties come
only from truncating layers, as a first-class candidate for the default. A wall-clock-steps-backwards
policy is a possible later addition, not in scope here.

