---
satisfies: [R2]
---
# fn-171-clean-up-testpilot.2 One spelling per Driver type: the adapter imports contract; shrink the root alias block

## Description
> Deferred: the parent spec's 2026-10-21 execution hold applies. These criteria preserve the filed scope; they do not authorize work on the frozen Scala tree. Revalidate against current Rust/successor work before resuming.

From the Testpilot research (2026-10-12). `contract` (54 names) has no importer outside Testpilot, yet `contract.go` re-exports them as 60 aliases in the root, so every Driver type has two spellings. Make `temporal/*` import `contract` directly, and shrink the root's alias block to what external callers use (`Driver`/`Session` and the few others the importers need). Behaviour unchanged.

## Acceptance
- [ ] Inventory current internal/external importers before resumption and reconcile fn-228's contract-package retirement; do not restore a deleted contract package.
- [ ] Each surviving Driver type has one owning declaration; internal adapters use it directly and only aliases needed by external callers remain.
- [ ] All affected importers compile and their behavior tests pass; removed public aliases have no remaining consumers.

## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
