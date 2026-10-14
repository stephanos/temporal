# Capabilities own their properties

> HTML render lens: [.flow/artifacts/fn-134-capabilities-own-their-properties/spec.html](../artifacts/fn-134-capabilities-own-their-properties/spec.html) — local-only (gitignored): open it from the working tree; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 1): "why does it have   object implements extends Implements(limits = three)( ie why limits?"
> user (turn 2): "I think binding limits there is too early and the wrong abstraction no? we might want to have a dfferent limit in our queries for this type"
> user (turn 3): "we definitely don't want it inside the Machine; are there other places to define it other than irFile? or can/should we require users to define a query with the limit explicitly?"
> user (turn 4): "should laws appear in the IR or be inlined away instead? they are really just a method to inherit behavior. not sth that needs to be propgagated"
> user (turn 5): "what's the  lint law table  for ?"
> user (turn 6, part 1): "lt's remove it. a law is just something we can mix in."
> user (turn 6, part 2): "however; can we retain the fact that the mixed in stuff is coming from a particular law in the IR? it would make it easier to traceback."
> user (turn 6, part 3): "also; s there a better name than \"law\"?"
> user (turn 8): "I like Convention"
> user (turn 10): "> object implements extends Implements( this would change too right?"
> user (turn 11): "why do we have implement, Implements, Convention, capabiltiies and Capabiltiies?"
> user (turn 12): "could we follow the paterns of the other sections and do object capabilities: val Closable = ... val Pausable = ..."
> user (turn 13): "then we not call it capabilitiy everywhere and ditch convention?"
> user (turn 14): "yes, write all this into a new flow next spece"
> user (turn 15, selected): "proceed-anyway"
> user (turn 16): "can we break down the tasks and add it to MILESTONES.md ?"
> user (turn 17, selected): "Drop" (citations)
> user (turn 17): "why merge them; arent' they differenr things??"
> user (turn 17, selected): "Keep as a test" (two-machines rule)
> user (turn 18, selected): "Keep separate"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 60% [paraphrase], 40% [paraphrase] -->

A Model author declaring what a machine can do currently has to deal with six names for three ideas. The six are `Law`, `Catalog`, `Implements`, the `implements` section, the `capabilities(m, limits)(…)` functions and the `Capabilities[S]` value. The three ideas are: a capability binds a protocol to a machine's vocabulary; a capability brings shared Properties; a machine declares its capabilities. On top of that, the declaration fixes the search bounds of the Queries it generates. That is too early and the wrong place. The same capabilities already run under different bounds for different designs, and the owner wants the freedom to bound a generated Query differently for a given machine.

The owner's framing: a law "is just something we can mix in", a way "to inherit behavior", and "not sth that needs to be propgagated". The IR already reflects this. A generated Property, Scenario and Query are ordinary declarations named `<machine>.<law>`. What keeps laws alive downstream is a separate sidecar file per IR file, and its main reader is lint's law table, a review view that the owner chose to remove. The one thing worth keeping is traceability: a reader of the IR should be able to tell which shared Property a generated one was mixed in from.

This spec collapses the vocabulary to two names. A **capability** (`Closable`, `Pausable`, …) carries the Properties it brings. A machine's **`capabilities` section** declares the capabilities it has, in the same object-of-named-members shape as its other sections (`states`, `effects`, `properties`, `queries`). The separate noun for the shared Property, "law" (and the briefly chosen replacement "convention"), goes away. Bounds move to the `queries` section, where every hand-written Query's bounds already live. The IR keeps a slim, inert origin on each generated Property.

## Architecture & Data Models
<!-- scope: technical -->

**A capability carries its Properties.** [paraphrase] Each capability kind keeps its binding fields: the case class, as today. Its companion now defines the Properties it brings, each as a def taking the model and the capability's fields. Declaring the capability brings exactly the Properties its companion defines. The `Law` base class and the `Catalog` (the map from capability kinds, and pairs of kinds, to laws) are deleted. [inferred] The text a law carried for the law table (`promises`, `doesNotPromise`) moves to the Property def's Scaladoc. The law table was its only reader.

**A Property's parameters say which capabilities bring it.** [user] Pausable and Pollable stay separate capabilities. A capability Property lives in one capability's companion and reads that capability's fields. It may also read fields of other capabilities. It is brought to a machine when every capability whose fields it reads is declared there, and not otherwise. That is how `Pausable.pausedIsNotDispatched` (reading Pausable's `paused` and Pollable's `running`) replaces today's pair entry without a catalog.

**The `capabilities` section.** [paraphrase] A machine or composition object declares its capabilities in `object capabilities extends Capabilities:`, with one named `val` per capability (lowercase, so the kind's companion stays unshadowed) and its waivers as statements of the body: `except(<Capability>.<property>, because = …)` and `overriding(<Capability>.<property> -> ownDef, because = …)`. [paraphrase] The base class gives the section the machine's types. A capability that binds another machine's outcome, fact or state type does not compile, and a capability whose state type appears in no field still resolves. [inferred] A capability set that several designs share is written once, as a parameterised base the designs' sections extend (preferred) or as shared defs each section's vals call. Either way it is written once, waivers included.

**Bounds live in `queries`.** [paraphrase] A capability declaration names no Limits. A machine's `queries` section bounds the Queries generated from its capabilities' Properties [inferred]: one statement bounds every Property they bring, with per-Property overrides. The Query names stay `<machine>.<property>`. Hand-written Queries that read a generated Property through `capabilities.claim(<Capability>.<property>)`, or the section's equivalent, choose their own bounds, as today.

**The IR keeps an origin, nothing more.** [paraphrase] A generated Property carries an inert `origin`: the fully qualified name of the capability Property it was expanded from, and the position of that Property's def. Its own `position` stays the capability's `val` in the machine's `capabilities` section. So the two positions trace from the place a Property was mixed in to the place it was defined. Scenarios and Queries carry no origin, since a generated Query reaches it through its Property. [paraphrase] There is no law-shaped data anywhere downstream: no law sidecar file, and no lint law table.

**Relation to fn-131.** [paraphrase] fn-131's R7 plans a `LawOrigin` carrying the law, capabilities, bindings, citations and overriding def, and has lint's law tables read the IR. This spec replaces that law-provenance part. The origin here is the whole of what the IR keeps, and the law tables are gone. fn-131's other parts (positions beside the tree, one canonical projection, the level check, `doc` on declarations) are unaffected. `origin` is one of the inert fields fn-131's canonical projection marks.

## API Contracts
<!-- scope: technical -->

- [paraphrase] **Capability kind**: a case class of binding fields that extends the capability marker typed by state, outcome and fact. Its companion is the kind and defines the kind's Properties as defs `(model, <fields>…) => Property`.
- [paraphrase] **`capabilities` section**: `object capabilities extends Capabilities`, with named vals of capability values, plus `except` and `overriding` statements naming a capability Property by `<Capability>.<property>`.
- [inferred] **`queries` section**: one statement that bounds all generated Queries of the machine's capabilities under a `Limits`, with an override that bounds one named capability Property under another `Limits`.
- [paraphrase] **IR `Property.origin`**: exactly two fields. One is the capability Property's fully qualified name (e.g. `temporal.capabilities.Closable.closedIsRejectedUniformly`); the other is the position of its def. It is unset on an author's own Property. It is inert: no fingerprint, answer, lowering or exploration identity reads it.
- [paraphrase] **Removed surface**: `Law`, `LawRef`, `Catalog` with its `single`/`pair` builders, `Implements`, every `implements` section, the `capabilities(m, limits)(…)` function and its two in-machine helpers, the `Capabilities[S]` value class, `cited(…)` and the parameter-without-citation lint, the law sidecar's reader and writer, and lint's law table.

## Edge Cases & Constraints
<!-- scope: technical -->

- [paraphrase] A machine that waives a Property with `except` generates no Property and no Query for it. A bounding override of a waived Property is refused.
- [inferred] A capability set shared across designs keeps one waiver list. A design must not be able to silently drop a waiver the shared set states.
- [inferred] Generated Query names are unchanged (`<machine>.<property>`), so receipts, accepted findings keyed by them, and Case names stay stable across the change.
- [inferred] A law sidecar file left behind in the IR directory is not skipped silently. The reader refuses it as a file that is not IR, so the regeneration must delete them.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** [paraphrase] A capability kind's companion defines the Properties it brings, and declaring the capability brings exactly those. `Law` and `Catalog` no longer exist. A capability Property is brought exactly when every capability whose fields its parameters read is declared, so `Pausable.pausedIsNotDispatched` is brought to a machine that declares both Pausable and Pollable, and to no other. A test holds that every capability Property is brought to at least two declared machines with their own state types. Errors: a capability Property that reads no field of its own capability, or takes a parameter no capability field of that name binds, or one two declared capabilities both bind, is refused by the IR generator, naming the Property and the capabilities; a capability declared twice for one machine is refused, naming both positions; the two-machines test fails naming a Property brought to fewer.
- **R2:** [paraphrase] Every machine and composition declares its capabilities in an `object capabilities extends Capabilities` section of named vals, with its waivers as `except`/`overriding` statements in that section. A capability set several designs share is written once and reused by each design's section, waivers included. `Implements`, the `implements` sections, the `capabilities(m, limits)(…)` functions and the `Capabilities[S]` value class no longer exist. Errors: a capability binding another machine's outcome, fact or state type does not compile; a waiver naming a Property no declared capability brings, or with an empty reason, is refused by the IR generator.
- **R3:** [paraphrase] No capability declaration names a `Limits`. Each Query generated from a capability Property is bounded in its machine's `queries` section, by one statement for all of them and an optional override per Property. Errors: a machine whose capabilities bring a Property that no `queries` section bounds is refused by the IR generator, naming the machine and the Property; an override naming a Property its capabilities do not bring, or one it waives, is refused.
- **R4:** [paraphrase] Each Property the IR generator expands from a capability Property carries an `origin` with exactly the Property's fully qualified name and the position of its def. No other Property carries one. Errors: no error surface beyond the generator always setting it; the Go reader accepts a Property without `origin`.
- **R5:** [paraphrase] `origin` is inert. Adding, changing or removing it changes no fingerprint, Query answer, lowering, exploration identity or Definition ID (no error surface beyond R7's identity check).
- **R6:** [paraphrase] No law sidecar is written or read, `umpire-lint` prints no law table, and `cited(…)` and the parameter-without-citation lint no longer exist. The waiver reasons a machine states still reach its accepted findings, keyed `<machine>.<property>`. Errors: a sidecar file left in the IR directory is refused by the reader as not IR; the model gate writes each waiver's reason into its IR file's accepted findings on `--update`, and its check fails where an accepted finding is missing or stale.
- **R7:** [inferred] One regeneration moves IR, fixtures and docs. Every generated Query's name, its answer, every Check receipt, every Definition ID, the exploration identity map and every Case byte are unchanged. The IR diff contains only `origin` fields, the removal of the sidecar files and the positions of generated Properties. The model README and semantics docs describe capabilities and capability Properties, and no source or doc outside `.flow/` and `.plans/` uses "law" or "convention" for this concept. Errors: any other difference stops the regeneration.

## Early proof point

Task fn-134-capabilities-own-their-properties.2 validates the core approach: a `capabilities` section, including one extending a shared parameterised base with its waivers, lifts capability Properties with their `origin`, bounds them from `queries`, and writes waiver reasons into accepted findings, on irgen fixtures beside the untouched old path. If the lifter cannot follow a shared base's members and waivers, fall back to shared defs that each design's vals call, and re-check R2's no-dropped-waiver error before continuing with .3+.

## Quick commands

```bash
scala-cli test model/irgen
make umpire-check-model
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/...
```

## Boundaries
<!-- scope: business -->

- [paraphrase] fn-131's other parts are not this spec's: positions beside the tree, the canonical projection, the level check, `doc` on declarations and the IR trimmings.
- [user] Law-shaped data is not propagated past the IR generator: no bindings, capability lists or catalog entries in the IR or beside it.
- [paraphrase] No new capability kinds and no new capability Properties; Pausable and Pollable are not merged.
- [inferred] `.flow/` specs and the `.plans/` archive keep the old vocabulary as historical record.

## Decision Context
<!-- scope: both -->

[paraphrase] Planning decisions, 2026-10-06. Citations are dropped: nothing outside the law table read them, and a Property's own Scaladoc can cite the server code. The two-machines rule stays as a Scala test. Waiver reasons reach accepted findings through the model gate, which knows every waiver after lifting, rather than through a file beside the IR. The migration is additive first (the new section beside the old one, never both on one machine), so the gate stays green between tasks.

[paraphrase] Bounds were part of the declaration only because the generated Queries had nowhere else to get them from. The same capabilities already run under different bounds for different designs. That makes the bound a property of the Query being run, so it belongs with the Queries. Two other homes were rejected. The IR file root would be a second, unusual place for bounds. The shared Property itself could know its depth, but not the size of each machine's state space.

[paraphrase] Laws were already inlined in the IR. The sidecar existed to carry text the IR had no slot for, and its main reader was the law table. With the table removed, all that's left worth keeping is traceability, which the slim `origin` provides. A richer origin (fn-131's `LawOrigin`) was rejected: nothing would read the extra fields.

[paraphrase] Naming went through "law", "convention" and "pattern". "Pattern" was rejected because it is already the IR's match pattern, the design-pattern notes and the "claim patterns". In the end the separate noun was dropped altogether. A shared Property is just a capability's Property. [user] Merging Pausable and Pollable to remove the one pair-brought Property was rejected: they are different concepts (control and dispatch). Instead, a Property's own parameters name the capabilities it needs, which removes the pair table without merging.

[user] Layout, 2026-10-06: one file per capability in `model/temporal/capabilities/` (`Closable.scala`, `Terminable.scala`, `Cancelable.scala`, `Pausable.scala`, `Pollable.scala`, `Describable.scala`), each holding the case class and its companion with its Properties. It replaces the split into `Capabilities.scala` (declarations), per-law files and `Catalog.scala`. Positions-only in the IR; fn-138's `Retries.scala` follows the same shape.

[inferred] Requiring one hand-written Query per capability Property was considered, as a way to make every bound explicit. One bounding statement with per-Property overrides was preferred, because it avoids writing about forty Query lines by hand and avoids rebuilding the Scenarios the generator derives for single-class Properties. R3's refusal keeps coverage from being lost silently either way.

[inferred] fn-134.2 implementation choices, 2026-10-06 (owner unavailable; worker's recommendation). The shared parameterised base is feasible, so the shared-def fallback was not needed: a shared set is `abstract class Shared(m: M)(using Declaring[S, O, F]) extends Capabilities`, each design's section is `object capabilities extends Shared(this)`, and the lifter reads the base's vals and waivers with its parameters bound to the section's arguments (a section may add waivers of its own; it cannot drop the base's). Each capability val is typed `: Capability`, the section's alias for `CapabilityOf[S, O, F]` of its machine: that is what makes a capability of another machine's types fail to compile and gives a phantom state type the machine's, and the lifter refuses an untyped val that does not conform. Bounds are one statement in any `queries` section, `X.capabilities.bound(limits, <Capability>.<property> -> limits, ...)` (a statement, not a val, so `IrFile` reads nothing new); the lifter finds it in every `queries` section of the lifted sources, so shared designs whose Queries sit in another machine's `queries` section can still be bounded. A hand-written Query reads a generated Property with `capabilities.claim(<Capability>.<property>)`, and a shared def can take the section as a `Capabilities[S, O, F]` parameter. A capability Property is a def of its kind's companion that returns `Property`; a parameter that no capability kind of the lifted sources has as a field is refused, one that a kind has but no declared capability binds leaves the Property unbrought. Waiver reasons reach the gate as `<file>.waivers.json`, which the lifter writes beside the lifted IR and the gate never checks in; the gate merges them into `<file>.lint.json` under `waived-law`, keeping every other acceptance in place.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | [paraphrase] A capability kind's companion defines the Properties it brings, and declaring the capability brings exactly those. `Law` and `Catalog` no longer exist. A capability Property is brought exactly when every capability whose fields its parameters read is declared, so `Pausable.pausedIsNotDispatched` is brought to a machine that declares both Pausable and Pollable, and to no other. A test holds that every capability Property is brought to at least two declared machines with their own state types. Errors: a capability Property that reads no field of its own capability, or takes a parameter no capability field of that name binds, or one two declared capabilities both bind, is refused by the IR generator, naming the Property and the capabilities; a capability declared twice for one machine is refused, naming both positions; the two-machines test fails naming a Property brought to fewer. | fn-134-capabilities-own-their-properties.2, fn-134-capabilities-own-their-properties.3, fn-134-capabilities-own-their-properties.4 | — |
| R2 | [paraphrase] Every machine and composition declares its capabilities in an `object capabilities extends Capabilities` section of named vals, with its waivers as `except`/`overriding` statements in that section. A capability set several designs share is written once and reused by each design's section, waivers included. `Implements`, the `implements` sections, the `capabilities(m, limits)(…)` functions and the `Capabilities[S]` value class no longer exist. Errors: a capability binding another machine's outcome, fact or state type does not compile; a waiver naming a Property no declared capability brings, or with an empty reason, is refused by the IR generator. | fn-134-capabilities-own-their-properties.2, fn-134-capabilities-own-their-properties.3, fn-134-capabilities-own-their-properties.4 | — |
| R3 | [paraphrase] No capability declaration names a `Limits`. Each Query generated from a capability Property is bounded in its machine's `queries` section, by one statement for all of them and an optional override per Property. Errors: a machine whose capabilities bring a Property that no `queries` section bounds is refused by the IR generator, naming the machine and the Property; an override naming a Property its capabilities do not bring, or one it waives, is refused. | fn-134-capabilities-own-their-properties.2, fn-134-capabilities-own-their-properties.3 | — |
| R4 | [paraphrase] Each Property the IR generator expands from a capability Property carries an `origin` with exactly the Property's fully qualified name and the position of its def. No other Property carries one. Errors: no error surface beyond the generator always setting it; the Go reader accepts a Property without `origin`. | fn-134-capabilities-own-their-properties.1, fn-134-capabilities-own-their-properties.2, fn-134-capabilities-own-their-properties.3 | — |
| R5 | [paraphrase] `origin` is inert. Adding, changing or removing it changes no fingerprint, Query answer, lowering, exploration identity or Definition ID (no error surface beyond R7's identity check). | fn-134-capabilities-own-their-properties.1 | — |
| R6 | [paraphrase] No law sidecar is written or read, `umpire-lint` prints no law table, and `cited(…)` and the parameter-without-citation lint no longer exist. The waiver reasons a machine states still reach its accepted findings, keyed `<machine>.<property>`. Errors: a sidecar file left in the IR directory is refused by the reader as not IR; the model gate writes each waiver's reason into its IR file's accepted findings on `--update`, and its check fails where an accepted finding is missing or stale. | fn-134-capabilities-own-their-properties.4, fn-134-capabilities-own-their-properties.5 | — |
| R7 | [inferred] One regeneration moves IR, fixtures and docs. Every generated Query's name, its answer, every Check receipt, every Definition ID, the exploration identity map and every Case byte are unchanged. The IR diff contains only `origin` fields, the removal of the sidecar files and the positions of generated Properties. The model README and semantics docs describe capabilities and capability Properties, and no source or doc outside `.flow/` and `.plans/` uses "law" or "convention" for this concept. Errors: any other difference stops the regeneration. | fn-134-capabilities-own-their-properties.3, fn-134-capabilities-own-their-properties.6 | — |

