---
satisfies: [R1]
---
# fn-128-gomad-deferred-linux-qualification-and.1 Establish native Linux candidate and runtime/host evidence

## Description
Implements R1. Transfer source: fn-105.31/.32 (R26/R27), fn-109.13/.21 (R7/R19 and inherited host gates), fn-110.2-.5 (R2-R7), fn-112.5/.9/.16 and fn-114.13.

**Size:** M
**Files:** tools/gomad/internal/backends/native/toolchain/**, tools/gomad/internal/runner/**, tools/gomad/internal/backends/native/cmd/gomadtool/**, tools/gomad/internal/backends/native/generation/**, tools/gomad/internal/backends/native/upgrade/**
**Touches:** [tools/gomad/internal/backends/native/toolchain/**, tools/gomad/internal/runner/**, tools/gomad/internal/backends/native/cmd/gomadtool/**, tools/gomad/internal/backends/native/generation/**, tools/gomad/internal/backends/native/upgrade/**]

### Approach
Freeze the combined candidate and qualify the native build/profile first. Retain runtime time-wire actual-consumer vectors, quiescence/nosplit, baseline/candidate fixtures, entropy/syscall/environment/disabled controls, zero-fuzz regeneration/byte-equivalence checks, seeded draw diagnostics, forward-clock removed-skip seeds 1-24 and traced forward workloads at seeds 11/17. Run all inherited host/process/runtime/overlay/integration gates and preserve their exact outcomes. Separate known D12 replay failures from other failures. R7 owns the final overall pass after repairs.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/native/toolchain/draw_inventory_test.go`
- `tools/gomad/internal/backends/native/toolchain/runtime/go1.27.1.patch`
- `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md`

### Revival prerequisites
Owner requests qualification and supplies a native linux/amd64 host or CI and a pinned supported toolchain/source candidate. This task stays deferred until those prerequisites exist. Retain original policy and exact command/evidence requirements from the transfer manifest and source clauses at commit 10d884c6f9d97681d08aaf2636f5850407f1586a. Commit verified task progress separately; do not convert an unavailable gate into a pass.

## Acceptance

Scope extension (owner decision, 2026-10-14): retain the additional native obligations assigned to this task by [native hibernation](../../MILESTONES.md#native-backend-hibernation-owner-decision-2026-10-14), with their original selectors/semantics and current-candidate evidence. This task stays deferred and incomplete under its existing revival prerequisites.

- [ ] Native linux/amd64 toolchain builds and launches with recorded source/profile/toolchain identity; all inherited initial runtime/host commands and controls execute with source-bound reports, exit codes and outcomes. No failing gate is labeled passed; unresolved acceptance carries forward to task 7. Evidence refresh obligations are recorded for changes from the initial candidate.
- [ ] Retain exact commands, exit codes, source/build identities and an independent review for accepted changes; commit task progress before completing it.

## Done summary
Won't do: native backend retired by the 2026-10-19 owner decision; closed without implementation.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: