---
satisfies: [R2, R3]
---

# fn-33-run-serial-bounded-semantic-exploration.3 Prepare and run one candidate through the Case Runtime
## Description
Implement the serial path from outstanding canonical Case through `PrepareCase`, one fresh `PreparedCase.Run`, cleanup, closed Verdict, and bridge observation. Preserve exact identities and separate preparation rejection from Run outcomes.

**Size:** L
**Touches:** `tools/umpire/campaign/run.go`, `tools/umpire/campaign/run_test.go`

## Acceptance
- [ ] Exactly one preparation or Run is active and no next candidate arrives before closure.
- [ ] Preparation rejection creates no Run; runtime/Host/monitor/cleanup behavior follows fn-64 precedence.
- [ ] No resident executor, Run Evaluation, scenario adapter, or caller-closure binding is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
