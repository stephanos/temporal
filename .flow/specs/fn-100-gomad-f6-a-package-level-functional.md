# Gomad F6: a package-level functional slice

## Goal & Context
<!-- scope: business -->

Milestone F6 of `MILESTONES.md`. Ten `./tests` suites covering activities, signals,
queries, updates, child workflows, continue-as-new, cron, workflow, cancel, and timers run through
`qualify-set`, each qualified or classified with an exact blocker. On linux/amd64 six of twenty
seed runs qualified; the environment-filtering fix needs requalification; the six seed runs that
diverge between fresh repetitions (signal, update, child, cron 17, timer 17) and the cancel
suite's replay-evidence difference are untriaged; none of the slice is in `temporal.json`.

## Architecture & Data Models
<!-- scope: technical -->

Suites are added to `temporal.json` as tier 3 with per-suite expectations and `required_probes`
where a suite depends on a modeled operation. Divergences are diffed with the runtime's event
instrumentation and `gomad replay --observed DIR`.

## Edge Cases & Constraints
<!-- scope: technical -->

- Suites stay unchanged; `testcore` may gain flags; test bodies may not.
- A suite needing more than two minutes of wall time records why.
- Evidence divergence with the same inputs is a Gomad defect and takes priority; it is fixed in
  Gomad or shown to be a test bug and fixed upstream.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The ten suites are in `temporal.json` with per-suite expectations; each non-qualified
  suite's expectation names a package, an operation, or a finding identity.
- **R2:** `make gomad3-qualification` completes with `infrastructure_errors == 0` and
  `failed == 0` on darwin/arm64.
- **R3:** No suite is classified as evidence divergence; every divergence found is explained and
  fixed.
- **R4:** At least eight of the ten suites qualify.
- **R5:** Per-suite `required_probes` are set where a suite depends on a modeled operation.
- **R6:** Per-suite transcript and decision counts, watchdog terminations, and denials are recorded
  in the milestone status.

## Boundaries
<!-- scope: business -->

- The whole `./tests` package (F7).

## Decision Context
<!-- scope: both -->

The slice turns "any test" into a measurement before the full enumeration.
