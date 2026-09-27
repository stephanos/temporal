# Gomad F1: restore the checkout on darwin/arm64

## Goal & Context
<!-- scope: business -->

Milestone F1 of `.plans/GOMAD_MILESTONES.md`. A developer on a clean `darwin/arm64` checkout builds
the patched toolchain and runs every Gomad v3 gate, and the integration contract tests pass. All F1
repairs (regenerated outputs, fixture corpus, qualification-set schema, root wrappers, `tagged`
fixture) landed and were verified on linux/amd64; darwin/arm64 has not been run since the Linux
port and the go1.27.1 bump. The milestone ladder is strictly ordered, so every later milestone's
darwin acceptance depends on this one.

## Architecture & Data Models
<!-- scope: technical -->

No new design. The work is running the existing gates on darwin/arm64 and repairing what the
Linux-only work broke there: platform-selected fixtures, darwin-scoped adapter source-set pins,
host-tier tests that pinned a platform or toolchain version, and anything else a darwin run
surfaces. The host needs `GOROOT` pointing at a stock go1.27.1 (the module requires it under
`GOTOOLCHAIN=local`).

## Edge Cases & Constraints
<!-- scope: technical -->

- Constraints from the milestone document apply: no policy widening, no source translation,
  fail-closed stays, evidence over narration.
- A darwin-only failure is fixed without regressing linux/amd64 behavior (keep platform selection
  explicit, never widen a guard to make a test pass).
- The runtime repeatability sweep runs on an unloaded host (known GC-dimension sensitivity).

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `make gomad3` builds the go1.27.1 toolchain from source on darwin/arm64 and
  `tools/gomad3/.bin/gomad doctor` reports the runner available. Errors: a failing build is
  recorded with its first error and repaired, never skipped.
- **R2:** `make -C tools/gomad3 test` passes on darwin/arm64, including `intercept-test`,
  `test-runtime`, the host tier, and every `*_toolchain_test.go`. Errors: a failing test is fixed
  or, if pre-existing and unrelated to Gomad behavior, recorded with its exact failure in the
  milestone status; no test is deleted or narrowed to pass.
- **R3:** `make gomad3-integration-test` passes on darwin/arm64.
- **R4:** The core qualification set reports `selected == 5`, `supported == 5`,
  `unsupported == 0`, and every workload `choice_replay_exact` on darwin/arm64.
- **R5:** The milestone document's F1 status records the darwin/arm64 result.

## Boundaries
<!-- scope: business -->

- Linux-only host-tier failures already recorded in the F1 status are out of scope unless the
  darwin work touches the same code.
- No toolchain version change (F2 owns that).

## Decision Context
<!-- scope: both -->

The milestone was executed on linux/amd64 from cloud sessions that cannot run darwin. This host
is darwin/arm64, so its first job is to close the darwin half of each already-applied milestone
in ladder order.
