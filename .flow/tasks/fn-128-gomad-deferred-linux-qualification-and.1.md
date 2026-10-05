---
satisfies: [R1]
---
# fn-128-gomad-deferred-linux-qualification-and.1 Establish native Linux candidate and runtime/host evidence

## Description
Implements R1. Transfer source: fn-105.31/.32 (R26/R27), fn-109.13/.21 (R7/R19 and inherited host gates), fn-110.2-.5 (R2-R7), fn-112.5/.9/.16 and fn-114.13.

**Size:** M
**Files:** tools/gomad3/toolchain/**, tools/gomad3/runner/**, tools/gomad3/internal/gomadtool/**
**Touches:** [tools/gomad3/toolchain/**, tools/gomad3/runner/**, tools/gomad3/internal/gomadtool/**]

### Approach
Freeze the combined candidate and qualify the native build/profile first. Retain runtime time-wire actual-consumer vectors, quiescence/nosplit, baseline/candidate fixtures, entropy/syscall/environment/disabled controls, zero-fuzz regeneration/byte-equivalence checks, seeded draw diagnostics, forward-clock removed-skip seeds 1-24 and traced forward workloads at seeds 11/17. Run all inherited host/process/runtime/overlay/integration gates and preserve their exact outcomes. Separate known D12 replay failures from other failures. R7 owns the final overall pass after repairs.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/draw_inventory_test.go`
- `tools/gomad3/toolchain/runtime/go1.27.1.patch`
- `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md`

### Revival prerequisites
Owner requests qualification and supplies a native linux/amd64 host or CI and a pinned supported toolchain/source candidate. This task stays deferred until those prerequisites exist. Retain original policy and exact command/evidence requirements from the transfer manifest and source clauses at commit 10d884c6f9d97681d08aaf2636f5850407f1586a. Commit verified task progress separately; do not convert an unavailable gate into a pass.

## Acceptance
- [ ] Native linux/amd64 toolchain builds and launches with recorded source/profile/toolchain identity; all inherited initial runtime/host commands and controls execute with source-bound reports, exit codes and outcomes. No failing gate is labeled passed; unresolved acceptance carries forward to task 7. Evidence refresh obligations are recorded for changes from the initial candidate.
- [ ] Retain exact commands, exit codes, source/build identities and an independent review for accepted changes; commit task progress before completing it.

## Done summary
Blocked:
Deferred by owner-authorized Linux scope transfer on 2026-10-04. No native linux/amd64 host or CI is available. Revival requires an owner request, native execution, pinned supported toolchain/profile and a frozen source candidate. This task owns R1; Linux evidence is incomplete and no pass or waiver is claimed.
## Evidence
- Commits:
- Tests:
- PRs:
