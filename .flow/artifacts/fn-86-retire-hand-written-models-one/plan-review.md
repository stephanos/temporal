# fn-86 plan review — round 1

Spec: fn-86-retire-hand-written-models-one — Retire hand-written Models: one authoring path through
the commands
Plan: 9 tasks, broken out 2026-09-12; R1 to R8 each have a task and every task declares its
`satisfies`.
Reviewed against: the spec markdown, the nine task records, and the tree at
`claude/umpire-order-spec-0ylk31`.

**Verdict: NEEDS_WORK.** No blocker: the ordering is right, the inventory-first shape is right, and the
deletions are correctly gated behind `.4`'s re-anchoring. Three findings need a decision written into
the plan before the tasks they touch, and two are wording.

Conducted in-session and recorded through `flowctl` (`review-rounds increment` then
`record --review-type plan --status-target plan`), so the spec carries the receipt and
`plan_review_status: needs_work`, and this document is the reviewer output it points at. The backend
is recorded as `claude` because the reviewer is this session, not a separate model — read the verdict
with that in mind. `flowctl` was missing when this session started and is installed now from GitHub
through the plugin marketplace (`claude plugin marketplace add gmickel/flow-next`, then
`claude plugin install flow-next@flow-next`).

## Findings

**F1 (P2). `.2`'s proof-point baseline has no lifecycle.** `.1` checks in
`tests/testcore/testpilot/testdata/baseline/typed-unary-contract.json` as the Contract `.2` compares
against, and nothing says what becomes of it. Left in place it is a second copy of a generated Contract
that no generator writes and no gate regenerates — exactly the frozen-baseline liability fn-85 `.1`
retires for the protocol migration. Say in `.1` that it is a scaffold and in `.2` that the comparison's
last step deletes it, or say why it stays and which gate keeps it honest.

**F2 (P2). `.5` leaves `umpire-inspect` an open either/or inside a deletion task.** Its files list
deletes `model/Temporal/Tool/Inspect.lean` and `NexusDiscovery.lean` "with `Makefile:518-527`
`umpire-inspect/list/explain`, unless re-pointed", and its acceptance reads "either removed from the
Makefile or shows the Caller Model". Those are different products: one drops three documented
developer entry points, the other keeps them. fn-85 `.7` lands `umpire-case --list/--render`, which is
plausibly the replacement for `--list` and `--render` but not for `explain`. Decide it in the plan —
including whether `umpire-explain` has a successor — so the task cannot land a silent removal.

**F3 (P2). `.7` expects byte-identical Switch goldens, and the command path is likely to move
Definition IDs.** `model/Umpire/Examples/Fixtures/SwitchCompiledArtifact.json` carries
`definitionId` values (`switch.state.power`, `switch.setup.subject-is-off`, `switch.role.subject`) and
three behavior fingerprints, while the commands derive Definition IDs from `Origin` (fn-85 `.2`). If the
command path cannot reproduce those exact ids, the artifact identity moves and
`model/Umpire/Tests/MigrationCompatibility.lean` moves with it. `.7`'s acceptance already allows "or
each diff is listed with its reason", so the task cannot fail on this — but it can surprise. State up
front that the Definition IDs are the thing to check first, and that the compatibility-family pin is
part of `.7` rather than a discovery inside it. `.5` already touches
`MigrationCompatibility.lean:121` for the deleted families; `.7` should own its own line there.

**F4 (P3). R1's destinations have no "kept" arm, and two inventoried modules are kept.** The inventory
covers `Temporal.Testpilot`, where `Conformance.lean` and `CaseSupport.lean` build Cases by hand and
stay — the spec says so ("Testpilot conformance and synthetic Cases stay, since they test the
runtime") — but R1 and `.1` enumerate destinations as migrate, delete with coverage recorded, or drop
with a reason. "Drop with a reason" reads as dropping the behavior, not as keeping the module. Add
"kept, with the reason" so those two rows and the `Temporal.System.Nexus` exception have a destination
that does not read as deletion.

**F5 (P3). Every task's file list was written against the pre-fn-85 tree.** fn-86 runs after fn-85,
which deletes `Temporal/Case/Template/**`, the `case` command in `Temporal/Case/Syntax.lean` and
`Nexus/Success/Model.lean`, and reshapes `Umpire/Case/Producer.lean` and `Temporal/Case/Registry.lean`.
The three tasks that say "delete its `register_case` line" (`.2`, `.3`, `.6`) are right only if fn-85
leaves `register_case` for the four hand-written Cases, which fn-85 `.7` does not say either way. Put
one line in `.1` — the task that touches nothing — requiring a re-read of the registry, the Producer
and the two `Success/` trees as fn-85 left them, with corrections written into the affected task
records before `.2` starts.

## What the plan gets right

- Inventory first, with a mechanical reconciliation (`lint-model`'s existing import-graph issue kinds
  rather than a new Go tool), so "we deleted something with readers" cannot happen quietly.
- The early proof point is the smaller of the two typed examples and it is a lowering comparison, not a
  fixture comparison, which is the right pin for a field relation.
- `.4` before `.5`: the kept Implementation Link is re-anchored on fn-85's product machine while the
  modules it imports still exist, so nothing is deleted out from under a proof.
- `.3` removes the four superseded Nexus shapes in the same change that removes their last Producer,
  and adds their names to the vocabulary gate — the pattern fn-87 established.
- `.8` is a direct-import rule, not a transitive one, which is the only form that can pass while every
  module still reaches those owners through `Umpire.Command`.
- The behavior of the deleted Race and Experimental models is written into fn-79's and fn-33's specs
  before deletion, and `.5` does not need either spec resumed.

## Revisions applied, same session

- **F1** `.1` says the typed-unary Contract baseline is a scaffold and `.2`'s last step deletes it.
- **F2** decided in `.5`: re-point `Temporal.Tool.Inspect`'s scenario registry at the Caller Model's
  Queries, keeping `Umpire.Examples.Switch`, and keep `make umpire-inspect`, `umpire-list` and
  `umpire-explain`. fn-85's `umpire-case --list/--render` renders Cases, not Plans, so it replaces
  neither `inspect` nor `explain`, and all three are documented in `model/README.md` and
  `.plans/UMPIRE4_COMPONENTS.md`.
- **F3** `.7` checks the Definition IDs the command path produces before touching a golden, and owns
  its line in `model/Umpire/Tests/MigrationCompatibility.lean`.
- **F4** R1 and `.1` gained "keep with the reason" as a destination, for `Temporal.Testpilot`'s two
  runtime-testing modules and the `Temporal.System.Nexus` exception.
- **F5** `.1` re-reads the tree as fn-85 left it — the registry, the Producer and the two `Success/`
  trees — and corrects the file lists of `.2`, `.3` and `.6` before `.2` starts.

Round 2 should be an independent read: the same session both reviewed and revised this plan.
