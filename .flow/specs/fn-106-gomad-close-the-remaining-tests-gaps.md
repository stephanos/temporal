# Gomad: close the remaining ./tests gaps

## Goal & Context
<!-- scope: business -->

F0–F9 delivered deterministic execution for the Temporal functional package with every test given
a disposition. The dispositions still leave real gaps: four suites excluded for transcript
overflow, Gomad-side limits behind three skipped subtests, a linux replay divergence that made the
F5/F6 suites intermittent on linux and the smoke gate seed-11-only, and a host-clock escape
through `MemStats.LastGC`. This spec tracks them; `MILESTONES.md` "Remaining `./tests` gaps"
and "Open findings" keep the rationale.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The linux seed-17 replay divergence is bisected to its runtime change and fixed, or its
  channel is identified and recorded; the F5/F6 linux expectations return to `qualified` where the
  evidence shows both seeds qualified, and the smoke gate runs both seeds again.
- **R2:** `gcMarkTermination` no longer reads the host clock when Gomad is enabled; the host-clock
  inventory records the change and the runtime tier passes.
- **R3:** The I/O transcript bound is configurable with an artifact-identity field, so the four
  excluded suites run; each is qualified or classified with a finding and its exclusion removed.
- **R4:** The dedicated-cluster pool in `testcore` no longer deadlocks under Gomad's single P, so
  `TestNexusOTELSuite/TestOperation` runs; it is qualified or classified.
- **R5:** The SDK panic-traceback output no longer differs between same-seed runs, or the leak is
  classified with its channel; `TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout`
  seed 11 is explained and fixed or classified.

## Boundaries
<!-- scope: business -->

- Tests that assume wall time passes during server work, and test bugs, stay owned skips unless a
  fix lands upstream in the test itself.
- No policy widening; server changes stay behind the `gomad` tag or an injection seam.
