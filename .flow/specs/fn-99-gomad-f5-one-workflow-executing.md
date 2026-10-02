# Gomad F5: one workflow-executing functional test, deterministic

## Goal & Context
<!-- scope: business -->

Milestone F5 of `MILESTONES.md`. `TestUserTimersTestSuite` exercises frontend,
history, matching, SQLite, inter-service gRPC, and virtual time. On linux/amd64, after the switch
to the classic collector, it reproduces eight of eight on both seeds, but the manifest still says
`intermittent` and the acceptance (`--repeat 4`, two seeds, identical evidence, `replay_match`)
was not re-measured after the environment-filtering fix. darwin/arm64 has not been run.

## Architecture & Data Models
<!-- scope: technical -->

Runs through `gomad qualify --repeat 4 --choices --choice-bytes 64MiB --replay-successes` with the
schema read-only mount, closure mode under the `gomad` tag, and the manifest entry
`user-timers-workflow`.

## Edge Cases & Constraints
<!-- scope: technical -->

- Exact choice-tape replay is not required by the milestone; seed-level repeatability plus exact
  I/O transcript replay is the bar (but record whether choice replay was exact).
- The test passes under stock Go before and after; no test body changes.
- A divergence is traced and fixed in Gomad, or recorded as a GC-dimension finding with evidence.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `gomad qualify --repeat 4` on seeds 11 and 17 produces identical canonical evidence per
  seed and replays with `replay_match`, on darwin/arm64.
- **R2:** The report records transcript bytes, transcript records, choice decisions, virtual time,
  wall time, and peak goroutines; the milestone status records the darwin numbers.
- **R3:** Zero watchdog terminations and zero `GOMAD_CAPABILITY_DENIED` across all repetitions.
- **R4:** The manifest expectation for `user-timers-workflow` is `qualified` on the platforms where
  it qualifies, and the CI assertion matches.

## Boundaries
<!-- scope: business -->

- Deterministic GC as a general research item.

## Decision Context
<!-- scope: both -->

The classic collector made this suite reproduce on Linux; the remaining work is proving it on
darwin and tightening the expectation.
