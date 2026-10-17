---
satisfies: [R17]
---
# fn-190-models-are-plain-scala-state-as-vars.12 Phase 2: remove the composition lifting and the <field>_<fact> keys from the lifter and Go (R17)

## Description
Touches: model/framework/reify/body (Expressions: `through`, `throughCallee`, `fieldPath`, `memberFactKey`, the two member-fact `lift` arms; Expansion or its successor: `memberCapture`, `selectorArgument`; BodyContext: `throughs`, `throughOf`, `Through`), model/framework/export composition readers until fn-174.8, tools/umpire/lower (composition facts), tools/umpire/interp, export/testdata composition fixtures

### Description
Phase 2, IR and plugin, batched with fn-190.9. After fn-190.11 the DSL has no `through`, member `records`, `MemberPath` or `captureBinding`. Remove their lifting: the `through` synthesis (`State.through.<path>.<def>` functions), `fieldPath` selector extraction where only they used it, `memberFactKey` and the `<field>_<fact>` text keys, `memberCapture`, and the two `lift` arms that read `StepFacts` member views and `memberRecorded`. A member fact now reaches the IR as an ordinary enum construction inside `records`, which the lifter already handles, and a member read as an ordinary call. On the Go side, the composition lowering stops decoding `<field>_<fact>` text keys and reads the tagged enum case; regenerate the composition fixtures once with the rest of the phase-2 batch. Record the lifter line count change in fn-189's counts receipt.

**Size:** M

### Verification
- The six samples and the composition fixtures lift; the composed samples' Query answers through the Go checks are unchanged from the retained oracles (a changed answer is a finding).
- grep shows no `through`, `memberFactKey`, `MemberPath` or `Through` in model/framework/reify and no `_` fact-key composition in tools/umpire/lower.

## Acceptance
- [ ] No composition-specific lifting remains in model/framework/reify; member facts lift as enum constructions.
- [ ] The Go composition lowering reads the tagged enum; text fact keys are gone.
- [ ] Composed samples answer the same through the Go checks; the lifter line count is recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
