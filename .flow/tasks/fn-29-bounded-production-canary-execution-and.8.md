---
satisfies: [R3, R4, R5, R6, R7, R8, R10]
---

# fn-29-bounded-production-canary-execution-and.8 Compose the external canary controller and run mode
## Description
Compose preflight, one-time `PrepareCase`, bounded serial Runs, exact Run/Verdict admission, fn-26 assessment, cleanup, and exactly-once publication behind a deep canary-owned controller. Expose a closed run mode with fixed Case/Profile/Host authority and no semantic or target override.

**Size:** L
**Touches:** `tools/canary/controller/**`, `tools/canary/cmd/umpire-canary/**`

## Acceptance
- [ ] Stage order and status 0/1/2 distinguish accepted, rejected/incomplete, and tooling failure.
- [ ] The PreparedCase is reused safely while every Run has fresh isolated state.
- [ ] No arbitrary Case, target, Host, checker, retry, executable, endpoint, credential, or release option exists.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
