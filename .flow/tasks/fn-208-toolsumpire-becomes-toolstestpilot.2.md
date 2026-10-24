---
satisfies: [R2]
---
# fn-208-toolsumpire-becomes-toolstestpilot.2 Move surviving commands to tools/testpilot/cmd and rename Make targets

## Description
Move umpire-run, umpire-fuzz, umpire-replay, umpire-repeat, umpire-assess and internal/cli under tools/testpilot/cmd with the testpilot- prefix; flags and exit codes unchanged; `--model` takes a descriptor-bearing Case; rename the live and repeat Make targets; the model gate becomes the framework check.

**Files:** tools/umpire/cmd/** -> tools/testpilot/cmd/**, tools/umpire/internal/cli -> tools/testpilot/internal/cli, Makefile.

**Quick commands:** `make testpilot-run` (deployment); `go test -tags test_dep ./tools/testpilot/...`

## Acceptance
- Commands build and keep their contracts; live checks pass against a deployment.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: