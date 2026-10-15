---
satisfies: [R4]
---
# fn-181-gomad-wasm-operator-integration.4 Recover and resume interrupted typed cluster campaigns

## Description
Carry prepared scenario plans through existing campaign recovery/resume for R4.

**Size:** M
**Files:** tools/gomad/internal/runner/resume.go, recovery.go, backend_prepared.go, internal/campaign/resume_plan.go, tools/gomad/internal/backends/wasm/command/kill_resume_linux_test.go
**Touches:** [tools/gomad/internal/runner/**, tools/gomad/internal/backends/wasm/backend/**, tools/gomad/internal/backends/wasm/command/kill_resume_linux_test.go, tools/gomad/cmd/gomad/internal/cli/recover.go, tools/gomad/cmd/gomad/internal/cli/resume.go]

### Approach
- Reuse append-log commit/recovery and backend restoration. Persist the declared scenario, seed ordering and every prepared boot/input required for unexecuted seeds; observed replay transitions cannot replace the original plan.
- Resume consumes retained inputs and identities, never modified scenario files or ambient configuration. Reexecute only uncommitted work and retain committed results/artifacts.
- Extend TestInstalledOperatorKillResume with completed successful and failed outcomes plus an interrupted execution. Follow the memory fix: stop the process group, confirm quiescence, inspect its durable prefix, kill/reap descendants, then compare before resume.
- Compare logical seeds/outcomes/evidence with uninterrupted control, excluding host metrics and campaign paths. Test deletion/change of original scenario, compiler, sources and mounts.

### Investigation targets
**Required:**
- tools/gomad/internal/runner/resume.go
- tools/gomad/internal/runner/recovery.go
- tools/gomad/internal/runner/backend_prepared.go
- tools/gomad/internal/runner/internal/campaign/resume_plan.go
- tools/gomad/internal/backends/wasm/command/kill_resume_linux_test.go:29

### Quick commands
From tools/gomad: `go test -tags test_dep ./internal/runner ./internal/runner/internal/campaign`; run TestInstalledOperatorKillResume with documented installed-helper inputs on Linux.

## Acceptance
- [ ] Recover/resume preserves frozen committed prefix and matches uninterrupted logical results
- [ ] Remaining seeds execute from retained prepared scenario inputs with originals changed or unavailable
- [ ] Changed installation, corrupt log or missing inputs refuse without overwriting valid evidence
- [ ] Kill/cancellation controls reap all descendants and never report incomplete work as success

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
