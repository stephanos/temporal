---
satisfies: [R3, R4, R5, R6, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.10 Build the public-boundary end-to-end canary harness
## Description
Build a controlled harness exercising protected preflight, fn-64 server/worker Host separation, one-time preparation, two isolated serial Runs, public Observations, Verdicts, assessment, cleanup, recovery, and publication without production credentials.

**Size:** L
**Touches:** `tools/canary/testharness/**`, `tests/umpire_canary_test.go`

## Acceptance
- [ ] Server and worker authority remain separate and only public server observations reach the Contract.
- [ ] Crash-before-Run, crash-during-Run, cleanup uncertainty, and recovery produce no fabricated Verdict or accepted receipt.
- [ ] Synthetic evidence is labeled and the harness cannot publish or retain an accepted production receipt.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
