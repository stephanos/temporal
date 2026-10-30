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
Frozen source `c88fdc550694020eb1c6a2e09577813d906c9ffe` and its exact framework/archive trees. The manifest contains all 1,177 framework files plus LICENSE, with blob IDs, SHA-256, modes, sizes and symlink targets. Independent Git archive comparison agrees on every selected byte/mode. The history map records both sibling archive moves, 1,225 Rust-qualified commits, exact support provenance and six representative histories.

Preservation evidence covers 81 local branches, all 36 observed worktrees and 217 additional refs, with merge-base deltas and provisional destination ownership. No checkout/ref/candidate was deleted or transferred. Dirty/unmerged state, unknown legacy ownership and ignored evidence remain explicit final-cutover obligations. Destination access still fails HTTP401; contents/permissions remain unknown and publication is not admitted.

Updated MIGRATE.md to use the existing independent workspace/tooling, preserve Java/TLC and both canonical Go identities, and honor the current baseline waiver/validation hold. MILESTONES.md tracks the full extraction through later current-pair proof and authorized cutover. All three implementation-review axes returned SHIP and were finalized through Flow reservation `4797c20787d149f8bc70dbd71c612620`. No builds/tests ran; this task's verification consists of Git object/inventory comparisons. Actual filtering and runtime proofs are intentionally unfinished tasks 2–6.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1ce541e398
- Tests: python3 .flow/tmp/fn257-capture-source.py --source c88fdc5506 --output .flow/artifacts/fn-257-extract-the-standalone-umpire/task1, Git archive tar entries independently compared against all source-manifest.json records; source-inventory-proof.json records 1178 exact matches, flowctl validate --spec fn-257 --coverage --json, Three source-only implementation review axes verified source/history/candidate inventories against Git; finalized reservation 4797c20787d149f8bc70dbd71c612620: SHIP
- PRs: