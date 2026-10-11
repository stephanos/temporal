---
satisfies: [R16]
---
# fn-159-qualified-action-names-in-rules.9 One spelling of a bounded integer: UpTo[N] on the field, ranges read per field

## Description
Retire the per-state `given Finite[Int] = Finite.upTo(n)` as the way to bound an `Int`, so a bounded integer has one spelling, `UpTo[N]`, and the IR generator reads every range from the field's own type (`model/SEMANTICS_FEEDBACK.md` 1.3; fn-131 records the lifter-side gap: "integer ranges are per state type, not per field").

**Today.** `UpTo[N]` is the field bound (`attempts: UpTo[2]`). The older form survives in one Model and three fixtures: the nexus workflow Protocol's `State` (`given Finite[State] = { given Finite[Int] = Finite.upTo(attemptBound); Finite.derived }`, `protocol/Protocol.scala:70`), `Rejects.scala`'s `CounterState`, `Presence.scala`'s `polls` field, and `Channels.scala`'s `channel[Int](...)(using Finite.upTo(2))`. The lifter's `readFinites` (`irgen/Types.scala`) records one Int range per *state type* (`intRanges(state)`), and `messageRef` (`irgen/Declarations.scala`) accepts an `Int` channel message only through `Finite.upTo(n)`. Two bounded `Int` fields in one record therefore cannot differ, and a reader of a Model meets two spellings of one thing. `Finite.upTo` is also how a named opaque type gets its range (`Presence.scala`'s `Retries`, `Rejects.scala`'s empty `Unreached`); that use stays.

**Size:** S
**Files:** `model/temporal/features/nexus/workflow/protocol/Protocol.scala`, `model/irgen/testdata/lifts/{Rejects,Presence,Channels}.scala` and their expected IR, `model/irgen/Types.scala` (`readFinites`, the two error messages that recommend `given Finite[Int] = Finite.upTo(...)`), `model/irgen/Declarations.scala` (`messageRef`), `model/framework/Domain.scala` (the `upTo` comment), `model/README.md` (Types bullet), `model/SEMANTICS.md` (step 1 and step 18)
**Touches:** [model/temporal/features/nexus/workflow/protocol/Protocol.scala, model/irgen/Types.scala, model/irgen/Declarations.scala, model/irgen/testdata/lifts/*.scala, model/irgen/testdata/expected/*.json, model/framework/Domain.scala, model/README.md, model/SEMANTICS.md]

### Approach
- **Model.** The Protocol's attempt field becomes `UpTo[2]`. Keep `attemptBound` for its readers as `final val attemptBound = 2`, whose type is the literal `2`, so the field can be written `UpTo[attemptBound.type]` and the number is still declared once; drop the `given Finite[State]` block, since the catalog now derives.
- **Channels.** An `Int`-message channel is declared over `UpTo[N]`: `channel[UpTo[2]](capacity = 1, order = Order.fifo, loss = Loss.reliable)`. `messageRef` already lifts an opaque type's range; delete its `Finite.upTo` branch and make its refusal of a plain `Int` message name `UpTo[N]`.
- **Lifter.** Delete the `Finite[Int]`-in-a-state-block path (`intRanges` keyed by state) from `readFinites`; keep the opaque-type and `Inbox` handling. A plain `Int` field is refused at its declaration with a message naming `UpTo[N]`; the two messages that today recommend `given Finite[Int] = Finite.upTo(...)` change accordingly. Ranges then come from each field's type, so two `UpTo` fields of one record may differ.
- **Fixtures.** Migrate `CounterState.n`, `State.polls` and `tally` to `UpTo[N]`; keep `Retries` and `Unreached` as they are (opaque-type ranges). Add one lift fixture with two differently bounded `UpTo` fields in one record, and one reject fixture for a plain `Int` field.
- **Docs.** One sentence each in the README's Types bullet, SEMANTICS step 1 (remove "the older form") and step 18 (channels of integers), and the `upTo` comment in `Domain.scala`: `Finite.upTo` ranges a named opaque type; a field or message is bounded by `UpTo[N]`.
- **No IR change expected.** The Protocol field already lifts as the range 0..2; it now comes from the field instead of the state. The fixtures' expected IR should be byte-identical apart from the new fixtures.

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): Scala and lifter changes only. Verify with the Scala build, the framework, Model and lifter tests, and the step-table equivalence; the IR is regenerated once in Batch 5b.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
```

## Acceptance
- [ ] No `given Finite[Int] = Finite.upTo(...)` remains in the Models or the lifter's fixtures; `Finite.upTo` ranges only a named opaque type.
- [ ] The IR generator reads an Int range from the field's or message's own type; `readFinites` has no per-state `Int` path and `messageRef` no `upTo` branch. A plain `Int` field and an `Int` channel message are refused at their declaration with a message naming `UpTo[N]` (reject fixtures).
- [ ] A lift fixture with two differently bounded `UpTo` fields in one record lifts each field with its own range.
- [ ] `model/ir/nexus-workflow.json` and every existing fixture's expected IR are byte-identical; the step-table equivalence passes.
- [ ] README, SEMANTICS and the `Domain.scala` comment describe one spelling.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
