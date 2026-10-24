---
satisfies: [R17]
---
# fn-190-models-are-plain-scala-state-as-vars.12 Phase 2: remove the composition lifting and the <field>_<fact> keys from the lifter and Go (R17)

## Description
**Recut 2026-10-18 (one-language series):** Only the lifter half remains; the Go lowering half of the `<field>_<fact>` keys is deleted by fn-208. Verify on Scala checker answers, not Go checks.

Touches: model/framework/reify/body (Expressions: `through`, `throughCallee`, `fieldPath`, `memberFactKey`, the two member-fact `lift` arms; Expansion or its successor: `memberCapture`, `selectorArgument`; BodyContext: `throughs`, `throughOf`, `Through`), model/framework/lift composition readers until fn-174.8, tools/umpire/lower (composition facts), tools/umpire/interp, export/testdata composition fixtures

### Description
Phase 2, IR and plugin, batched with fn-190.9. After fn-190.11 the DSL has no `through`, member `records`, `MemberPath` or `captureBinding`. Remove their lifting: the `through` synthesis (`State.through.<path>.<def>` functions), `fieldPath` selector extraction where only they used it, `memberFactKey` and the `<field>_<fact>` text keys, `memberCapture`, and the two `lift` arms that read `StepFacts` member views and `memberRecorded`. A member fact now reaches the IR as an ordinary enum construction inside `records`, which the lifter already handles, and a member read as an ordinary call. On the Go side, the composition lowering stops decoding `<field>_<fact>` text keys and reads the tagged enum case; regenerate the composition fixtures once with the rest of the phase-2 batch. Record the lifter line count change in fn-189's counts receipt.

**Size:** M

### Verification
- The six samples and the composition fixtures lift; the composed samples' Query answers through the Scala checker's checks are unchanged from the retained oracles (a changed answer is a finding).
- grep shows no `through`, `memberFactKey`, `MemberPath` or `Through` in model/framework/reify and no `_` fact-key composition in tools/umpire/lower.
## Acceptance
- [ ] No composition-specific lifting remains in model/framework/reify; member facts lift as enum constructions.
- [ ] The Go composition lowering reads the tagged enum; text fact keys are gone.
- [ ] Composed samples answer the same through the Go checks; the lifter line count is recorded.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: