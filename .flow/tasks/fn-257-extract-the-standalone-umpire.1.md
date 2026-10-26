---
satisfies: [R1]
---
# fn-257-extract-the-standalone-umpire.1 Freeze the source, history map and retained candidate ledger

## Description
Record the fixed Git source and reconcile the execution map before filtering.

**Size:** M
**Files:** MIGRATE.md, MILESTONES.md, this spec's source inventory/history/candidate artifacts.
**Touches:** [MIGRATE.md, MILESTONES.md, .flow/artifacts/fn-257-extract-the-standalone-umpire/**]

### Approach
- Inventory every model/umpire tracked blob, executable mode and symlink target at c88fdc5506; capture root LICENSE separately and archive tree identity.
- Preserve all existing worktrees/branches and record destination ownership of framework-only, Temporal-only and mixed candidates. Record dirty status as an observation, not a backup or transfer.
- Qualify model/ Rust history by descendants of either 6264f11760 or 0fc76f7805. Record older framework/host/native-tool paths and support ownership transitions.
- Update the stale parent-owned workspace/script map to current framework-owned files; preserve the current validation hold, baseline waiver and final fn-254 obligations.
- Record destination HTTP401 without inferring contents or permission.

### Investigation targets
**Required:** MIGRATE.md; model/README.md; model/umpire/README.md; .flow/specs/fn-254-fast-validation-gates-and-a-real-inner.md; .flow/tasks/fn-254-fast-validation-gates-and-a-real-inner.11.md.

### Quick commands
Git object inventories and exact blob/mode comparisons only; no new builds/tests during the owner hold.

## Acceptance
- [ ] The complete selected-file manifest binds a fixed source revision and exact root/archive tree identities.
- [ ] The map includes both archive-move lineages and excludes pre-move Scala.
- [ ] All existing candidates remain present with explicit provisional ownership; inaccessible destination and unrun gates remain open.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
