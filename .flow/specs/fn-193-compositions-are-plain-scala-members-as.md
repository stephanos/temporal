# Compositions are plain Scala: members as vals, no state case class

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user (2026-10-17): "let's priroize the next work by its impact on the DSL. let's define the next phase to focus exclusively on what weneed to do to complete all work that changes the Scala DSL."
> conductor proposal, confirmed by the owner ("confirm"): retiring `RecordMachine` needs a plain form for `Composition` and its derivations; add a short spec for it in the DSL-completion phase.

## Goal & Context

fn-190 made Machines plain Scala: state as `var`s, effects as assignment blocks, claims as `check`
blocks in sections. Compositions were left out: `Composition[S <: Product]` still takes an
author-written state case class with one field per member (`final case class OverQueue(activity:
State, queue: QueueView)`), selects members with field selectors (`_.activity -> ProtocolMember`),
needs `Data[S]`/`StateData[S]`, and keeps the record-state `property`/`scenario`/`query` surface.
`DerivedComposition`, `withMember` and `RecordMachine`-based `Owner`/`Declaring` exist only to serve
it. After fn-190 R17 its facts are already data (a per-member case of the composition's `Fact` enum).

This spec gives a composition the plain form so `RecordMachine` and the record-state vocabulary can
be deleted from the DSL (fn-190.16). It is DSL-only (phase "DSL completion", owner 2026-10-17): the
lifter, IR and Go lowering of compositions follow in the later IR/plugin phase with fn-190.12.

## Architecture & Data Models

Direction, to be confirmed or revised by the prototype (task 1):

- A composition is an `object … extends Composition[O, F]` whose members are `val`s naming
  Machines (`val activity = ProtocolMember`, `val queue = QueueScheduled`); the composed state is
  the members' vars, read through the member (`activity.phase`, `queue.outstanding`). No state case
  class, no field selectors, no `->` pairing.
- `syncs` pair member actions by reference: `sync(activity.worker.poll, queue.deliver)`; `own` and
  `synced` take the member's action directly.
- Claims use the fn-190 sections (`invariants`, `postconditions`, `progress`) and `check` blocks;
  a member's helper is an ordinary call on the member (`ActivityRecord.states.twoActive(activity)`
  becomes a call reading that member's vars); member facts stay `Fact.<member>(f)` (fn-190 R17).
- A composition with one member replaced is a derivation, `Queue.withMember(activity = LateRecord)`
  or a `rebind`-style form chosen by the prototype; `DerivedComposition` goes if the plain form
  keeps the source's fact type without it.
- Native runtime: a composed state is the product of the members' var snapshots; a sync executes
  both members' effects in one step; the native checks (fn-190 R20) run over it as over a Machine.

## Acceptance Criteria

- **R1:** A design note in `model/design` records a prototype of the plain composition form on the
  framework's own composition fixtures (the `Joined`/`Together`/`Reordered` tests): members as
  vals, syncs by action reference, claims in sections, a derivation replacing one member, and the
  native runtime stepping a sync. It states the verdict on each Architecture bullet and the
  alternative chosen where one does not work. Errors: the note is not optional.
- **R2:** `Composition` declares members as vals and has no state type parameter; `Data[S]`,
  `StateData[S]`, field selectors and `->` member pairing are gone from the composition surface.
  Errors: a member that is not a Machine or Composition is a compile error naming it; a sync of an
  action the member does not declare is refused with a located message.
- **R3:** Composition claims, scenarios and witnesses use the fn-190 forms (sections, `check`,
  `only`/`fair`/`starts`, `verify`/`find`, `witness`); the record-state `property`/`scenario`/
  `query` surface is not used by any composition. Errors: as fn-190 R11/R13.
- **R4:** The framework's composition fixtures and DSL tests are rewritten in the plain form and
  pass natively with their assertions preserved; SEMANTICS §17 and the README composition section
  describe the plain form.
- **R5:** Nothing in a composition's implementation requires `RecordMachine`, `Owner`, `Declaring`
  or `DerivedComposition` (if retired), so fn-190.16 can delete them.

## Boundaries

- DSL, realize and native runtime only; the reify plugin, export lifter, IR and Go lowering of
  compositions are the later IR/plugin phase (with fn-190.12).
- Temporal compositions (`activity_mini` Queue, the standalone protocol) stay disabled and migrate
  at the sweep by recipe.
- No new syntax beyond what the prototype shows is needed; plain Scala first.

## Requirement coverage

| Req | Task(s) | Gap justification |
| --- | --- | --- |
| R1 | fn-193.1 | — |
| R2 | fn-193.2 | — |
| R3 | fn-193.2 | — |
| R4 | fn-193.2 | — |
| R5 | fn-193.2 | — |

## Decision Context

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.
