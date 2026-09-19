---
satisfies: [R4]
---

# fn-33-run-serial-bounded-semantic-exploration.4 Expose the closed serial umpire-fuzz command
## Description
Add the bounded `umpire-fuzz run` command over the completed serial coordinator. Emit canonical summary/error output that separates selection, preparation, Run start, decisive result, coverage, exhaustion, limits, stop/lost iteration, and tooling failure.

**Size:** M
**Touches:** `tools/umpire/cmd/umpire-fuzz/**`, `Makefile`

## Acceptance
- [ ] Command arguments expose only the checked campaign, policy, seed, fixed Profile, and bounded limits.
- [ ] Counters and exit statuses cannot count unexecuted, inconclusive, lost, or cleanup-uncertain work as coverage.
- [ ] Output is bounded and secret-free with no persisted resume token or arbitrary executable option.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
