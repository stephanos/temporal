---
satisfies: [R4]
---
# fn-33-run-resumable-semantic-exploration.4 Expose the closed serial umpire-fuzz run command

## Description
Add the bounded public command and honest terminal output over the completed serial loop.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-fuzz/**`, `Makefile`
**Touches:** [tools/umpire/cmd/umpire-fuzz/**, Makefile]

### Approach
- Accept one checked Nexus Space binding, retained fn-17 policy, fn-40 PlannerPolicy, and explicit candidate/wall-clock Limits.
- Drive the serial coordinator until exhausted, Limit Reached, stopped, or tooling failure.
- Print selected/executed/admitted counts, Lean-owned semantic coverage, and the first unexecuted candidate when known.
- Keep command output operational and inspectable without defining a persisted Artifact.

### Investigation targets
**Required** (read before coding):
- Existing `tools/umpire/cmd` command conventions.
- Task `.3` serial runner interface.
- Parent spec terminal output contract.

## Acceptance
- [ ] `umpire-fuzz run` has one closed bounded command shape and no arbitrary executable option.
- [ ] Exhausted, Limit Reached, stopped, and tooling failure remain distinct.
- [ ] Focused command tests and root Make wiring pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
