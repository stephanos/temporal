---
satisfies: [R1, R6]
---
# fn-257-extract-the-standalone-umpire.6 Publish non-destructively and cut over source ownership

## Description
Complete migration only after the exact pair is proven and destination/candidate disposition is known.

**Size:** M
**Files:** both repositories' final source ownership, CI/docs/agent/task/milestone references and migration evidence.
**Touches:** [MIGRATE.md, MILESTONES.md, AGENTS.md, .github/**, model/**, .flow/**, ../umpire/**]

### Approach
- Re-probe authenticated destination branches, contents, license and permissions. Preserve existing history; use an authorized branch/PR integration path and resolve any real integration conflict before publication.
- Name each unfinished candidate's repository, branch/recovery ref, tracker record, evidence and cross-repository dependencies; split mixed-source checkpoints rather than blindly replaying old monorepo commits.
- Remove duplicate in-tree framework source only after independent destination/local-consumer availability, task 5 evidence and source-removal authorization. Keep retained Scala archive/caches unchanged.
- Recheck source/archive/cache inventories, actual consumer resolution and commands after cutover. Update current links without closing transferred obligations. Record validation-cost assessment and committed closure, then clean only eligible owned temporary worktrees/artifacts.

### Investigation targets
**Required:** MIGRATE.md execution step 6; AGENTS.md local integration/cleanup policy; task 1 candidate ledger; task 5 exact-pair evidence.

## Acceptance
- [ ] Destination contents/history/license/permissions are known and the authorized publication preserves existing history.
- [ ] Every unfinished candidate remains recoverable and owned; source removal follows verified consumers and authorization.
- [ ] Both repositories have one clear current source owner, working references/commands, preserved archive/cache identity and committed closure evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
