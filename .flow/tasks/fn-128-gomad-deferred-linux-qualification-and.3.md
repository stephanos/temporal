---
satisfies: [R3]
---
# fn-128-gomad-deferred-linux-qualification-and.3 Resolve conditional Linux host-clock audit

## Description
Implements R3. Transfer source: whole fn-105.11 / D11 / R11 and completed D21's trigger decision.

**Size:** M
**Files:** tools/gomad/internal/backends/native/toolchain/**, tools/gomad/internal/backends/native/conformance/**, .github/workflows/**
**Touches:** [tools/gomad/internal/backends/native/toolchain/**, tools/gomad/internal/backends/native/conformance/**, .github/workflows/**]

### Approach
Read D21's retained findings before choosing execution scope. Preserve its existing deferred decision until the owner establishes need and feasible scope. If triggered, build the bounded vDSO-disabled/seccomp fixture, positive control and seeded native CI run. If not triggered, obtain and retain the explicit owner disposition with the findings. The declined collector-stamp overwrite and generic syscall prohibition stay in force.

### Investigation targets
**Required:**
- `.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.21.md`
- `.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.11.md`
- `tools/gomad/internal/backends/native/toolchain/clock_inventory_test.go`

### Revival prerequisites
Owner requests qualification and supplies a native linux/amd64 host or CI and a pinned supported toolchain/source candidate. This task stays deferred until those prerequisites exist. Retain original policy and exact command/evidence requirements from the transfer manifest and source clauses at commit 10d884c6f9d97681d08aaf2636f5850407f1586a. Commit verified task progress separately; do not convert an unavailable gate into a pass.

## Acceptance
- [ ] Retain either all original D11 native audit acceptance with commands and identities, or an explicit owner-approved not-triggered disposition bound to D21 evidence. Missing native hardware never counts as a false trigger or audit pass.
- [ ] Retain exact commands, exit codes, source/build identities and an independent review for accepted changes; commit task progress before completing it.

## Done summary
Won't do: native backend retired by the 2026-10-19 owner decision; closed without implementation.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: