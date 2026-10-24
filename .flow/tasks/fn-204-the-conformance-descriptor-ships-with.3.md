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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: