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
- Resume consumes retained inputs and identities, never modified scenario files or ambient configuration. Honor the current admitted/terminal-receipt/committed outcome rules: reuse acknowledged completed receipts, rerun only durably incomplete work and retain committed results/artifacts in original seed order.
- Extend TestInstalledOperatorKillResume with completed successful and failed outcomes plus an interrupted execution. Follow the memory fix: stop the process group, confirm quiescence, inspect its durable prefix, kill/reap descendants, then compare before resume.
- Compare logical seeds/outcomes/evidence with uninterrupted control, excluding host metrics and campaign paths. Test deletion/change of original scenario, compiler, sources and mounts.
- Preserve every future boot/input/config and explicit crash-selection operations/digest context through existing prepared backend payload references and `loadPreparedBackend`; a prior observed lifecycle cannot stand in for the executable scenario. Default nil payload bytes remain unchanged.
- Add controls where a completed execution has a durable terminal receipt but its outcome is not yet committed, and where a seed has no terminal receipt, so resumption neither duplicates acknowledged execution nor loses incomplete work.

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
- [ ] Acknowledged terminal receipts survive interruption before outcome commit; only durably incomplete attempts rerun in deterministic seed order
- [ ] Context-bearing restart boots and captured inputs restore without originals/caches; missing/tampered context refuses without modifying valid evidence

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
