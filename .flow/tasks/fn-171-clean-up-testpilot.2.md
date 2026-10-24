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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: