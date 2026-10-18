---
satisfies: [R3]
---
# fn-204-the-conformance-descriptor-ships-with.3 Live tests, umpire-run and umpire-assess take the descriptor; pinned runs re-recorded

## Description
Switch model_fixture.go, umpire-run --model and umpire-assess --model to the descriptor-bearing Case; re-record the pinned runs and require every recorded verdict to reproduce.

**Files:** tests/testcore/testpilot/model_fixture.go, tools/umpire/cmd/umpire-run, tools/umpire/cmd/umpire-assess, Makefile umpire-rerecord-pinned-runs, pinned run files.

**Quick commands:** `make umpire-assess-run`; `make umpire-check-live-tests` (deployment)

## Acceptance
- All pinned runs reproduce their verdicts from descriptors.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
