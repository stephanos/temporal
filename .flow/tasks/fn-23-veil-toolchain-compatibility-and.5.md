---
satisfies: [R3, R5, R6]
---
# fn-23-veil-toolchain-compatibility-and.5 Expose the opt-in compatibility command and root target

## Description
Expose the diagnostic module through a zero-configuration `umpire-veil-compatibility` command and the repository-root `make umpire-check-veil-compatibility` target. Enforce the exact stdout/stderr/status envelope and bounded canonical progress-event stream, reject arguments and environment-based overrides, create the caller-owned temporary root, and map transport, acquisition integrity, host-tool, sandbox, resource enforcement, cancellation, cleanup, invariant, and reporting failures to the closed error schema. Keep the target opt-in and absent from ordinary model, regression, CI, and production dependency paths. Preserve existing comments and put every Make change in the top-level Makefile.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-veil-compatibility/main.go`, `tools/umpire/cmd/umpire-veil-compatibility/main_test.go`, `Makefile`
**Touches:** [tools/umpire/cmd/umpire-veil-compatibility/main.go, tools/umpire/cmd/umpire-veil-compatibility/main_test.go, Makefile]

## Acceptance
Status 0 emits only canonical receipt JSON plus LF for completed adopt/defer decisions, status 2 does so for inconclusive, and status 1 leaves stdout empty. Stderr contains at most 128 ordered canonical progress lines and, only for status 1, exactly one final canonical error line. Argument, signal, phase ordering, event/stream N/N+1, temp-root, cleanup, and write-failure tests preserve this split; target-graph tests prove no existing Make/Lake/CI/runtime target invokes the command and the command cannot write the checkout.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
