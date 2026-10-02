# Gomad F3: qualify the frontend functional probe

## Goal & Context
<!-- scope: business -->

Milestone F3 of `MILESTONES.md`. `TestFrontendSystemInfo` must carry a checked
determinism claim: the one-box cluster boots, serves an RPC, and shuts down under virtual time
repeatably. On linux/amd64 the probe runs under guarded mode but was left `unrepeatable`
(same-seed choice-trace divergence and replay divergence). Several host-timing channels were
closed since (F5/F6 work: classic collector, host-timed draws off the seeded stream, stack scans
waiting for host syscalls, control-variable filtering before copy), which may already fix it.
darwin/arm64 has not been run.

## Architecture & Data Models
<!-- scope: technical -->

Qualification uses `gomad qualify --seed {11,17} --repeat 2 --choices --replay-successes` against
`go-test ./tests/gomadfunctional -- -test.run '^TestFrontendSystemInfo$'` with tags
`disable_grpc_modules,test_dep` (plus `gomad` where the closure needs it), and the manifest entry
in `tools/gomad3integration/qualification/temporal.json`.

## Edge Cases & Constraints
<!-- scope: technical -->

- No new pack unless the analyzer names a blocker not already admitted; a pack change is amended
  and re-reviewed through discover/review/generate/check/qualify.
- The probe test stays 16 lines.
- A divergence is traced to its channel and fixed in Gomad; the expectation is never relaxed to
  hide it.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** On darwin/arm64, both seeds produce identical canonical evidence across two repetitions
  and replay with `replay_match` and `choice_replay_exact`.
- **R2:** On linux/amd64 the same holds, or the remaining divergence is recorded with its first
  divergent ordinal and cause (this host cannot run linux; record as not re-measured if so).
- **R3:** `temporal.json` lists the probe with the capability mode that actually qualifies it and
  a darwin/arm64 expectation of `qualified`; `make gomad3-qualification` reports it supported.
- **R4:** The CI workflow's Temporal assertion matches the new supported count.
- **R5:** The report records transcript bytes used and choice decisions recorded for the probe,
  and the milestone status records them.

## Boundaries
<!-- scope: business -->

- Workflow-executing tests (F5).

## Decision Context
<!-- scope: both -->

The linux `unrepeatable` expectation was a holding state; the later runtime fixes are the
cheapest thing to test first.
