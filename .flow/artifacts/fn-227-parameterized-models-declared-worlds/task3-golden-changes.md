Activity's default World now selects three attempts. Nine activity expectation payloads and their existing macro expansion mirror change. The comparison at task base `376fda53b344e5503a69e628195b87c71837a4c5` covers 805 existing snapshot, stderr and expected files. The other 795 files remain byte-identical, including snapshot metadata and unrelated expansions.

| Activity expectation | Reason |
|---|---|
| `Activity@cfg.snap.cfg` | Bind maximum_attempts=3 and the finite payload catalog. |
| `Activity@tla.snap.tla` | Independently lift attempt state, Poll input, rejection and retryable failure. |
| `report.snap.txt` | Record starts 1, reached 18, emitted 48, evaluated 162, refused 114, depth 6 and both Verified closure laws. |
| `states.snap.txt` | Retain all six phases at each of the three attempts. |
| `rows.snap.txt` | Record current/stale Poll payloads, both failure payloads, retries, direct deadlines and terminal Cancel rejection. |
| `tlc.snap.txt` | Record independently lifted TLC/native agreement on 18 states and both closure laws. |
| `expansion/closings.snap.txt` | Retain 21 actual witnesses, seven alternatives per attempt, including both direct deadlines and both failure payloads. |
| `coverage.snap.txt` | Record all nine concrete actions touched and 78 witness evaluations per closure law. Rejected outcomes remain covered by separate law/refinement controls. |
| `realization.snap.txt` | Map the incoming callback attempt field to Poll instead of ignoring it. |
| `temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__activity-realization.snap` | Refresh the existing expansion of the same activity realization, retaining payload actions, observed attempt, state argument and mismatch refusal. |

All 15 original activity artifacts remain immutable under `compatibility/original/`, reconstructed from `af0ebafc571d33651fdf2c008d4415ad72467a0d` and verified against the original manifest. They include the historical lift refusal. The compatibility test compares exact canonical starts, reachable phases and action/outcome/fact/destination relationships, deduplicating the two equivalent failure payload assignments in the one-attempt World. It preserves both deadline edges and four terminal rejection loops; changing the original ScheduleToStart destination fails that same comparison.

The direct-deadline, atomic-cancellation checkpoint proves this task's R4 relation. The final Tick-era compatibility World and projection remain with fn-237.3.
