---
satisfies: [R4]
---
# fn-254-fast-validation-gates-and-a-real-inner.7 Temporal crates: one test binary per crate after fn-246


## Description
Apply .4's consolidation to `model/temporal/**`.

**Cross-spec gate:** start after fn-246's committed closure; its lanes edit `temporal/realization` and activity tests.

**Size:** S
**Files:** `model/temporal/**/tests/*.rs`.
**Touches:** [model/temporal/**/tests/**, model/temporal/**/Cargo.toml, model/temporal/testing/*.rs, model/scripts/**, model/dev/gate/budgets.toml, model/umpire/dev/gate/budgets.toml]

### Approach
- Same rules and exceptions as .4. Inventory explicit targets in `temporal/testing/Cargo.toml` (`fixtures`, `snapshot_ownership`, `boundary`) and realization crate tests; preserve root-level sources and the dedicated boundary filter. Do not assume a stale `realization` target in the testing crate.
- **Equivalence pin:** a checked package/target/full-name/ignored bijection retains every case and snapshot. If .12 has installed the budget tables, update their changed count/stage entries at `model/dev/gate/budgets.toml` and `model/umpire/dev/gate/budgets.toml`. Otherwise prepare the exact inventory/count accounting for .12 to consume. This task does not gain a .12 dependency. Flow keeps this task explicitly blocked on the external prerequisite; root verifies fn-246's committed spec closure before releasing it. Source merge alone is insufficient, and Flow does not support cross-spec task dependency edges.

## Owner validation cadence (2026-10-25)

After its existing committed fn-246 closure prerequisite, keep affected Temporal test execution and exact identity mapping now; unrelated full model gates wait for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] Binary count and compile/link time recorded before and after.
- [ ] Test list identical modulo mapping, snapshots unchanged and affected Temporal tests pass; unrelated full `model-check` execution is deferred to .8 under the owner cadence amendment.

## Done summary
Blocked:
Wait for fn-246-realization-as-a-checked-refinement-a to close and for its closure record to be committed in umpire. This preserves the owner's whole-spec prerequisite for Temporal test consolidation. The conductor verifies closed status and committed closure ancestry before resetting .7 to todo. Framework/gate work proceeds first; resume fn-246 after that checkpoint, then finish .7/.8. Flow rejects cross-spec task dep edges, so the explicit blocked state carries this external admission gate.
## Evidence
- Commits:
- Tests:
- PRs:
