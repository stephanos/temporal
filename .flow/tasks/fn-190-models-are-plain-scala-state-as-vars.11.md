---
satisfies: [R17]
---
# fn-190-models-are-plain-scala-state-as-vars.11 Phase 1: composition member facts as a tagged fact enum and member reads as plain calls; retire through, member records, MemberPath (R17)

## Description
Touches: model/framework/dsl/Compose.scala, model/framework/dsl/MemberPath.scala, model/framework/dsl/Syntax.scala (the member `records` overloads and `memberRecorded`), model/framework/samples (any composed sample), model/framework/dsl tests, SEMANTICS.md

### Description
Phase 1, DSL only (owner's phasing, 2026-10-17). Retire the composition vocabulary from the DSL: `through(select, read)`, the member overloads of `records` (`records(_.member -> Machine, fact)` in claims and witnesses), `memberRecorded`, `MemberPath` and `captureBinding`. Replace them with data: a composition declares its fact type as an enum with one case per member carrying that member's fact (`enum Fact: case activity(f: AdmissionFact); case record(f: RecordFact)`), the composed step records member facts as those cases, and a claim writes `records(Fact.activity(AdmissionFact.x))`; a member's helper is called on the member's state directly, `ActivityRecord.states.twoActive(activity)`. Update the composed sample or fixture that uses the old forms, the DSL tests, and SEMANTICS.md's composition section. The native runtime (phase 1) records the tagged case in the composed step's facts. The lifter, the `<field>_<fact>` text keys and the Go lowering are fn-190.12.

**Size:** M

### Verification
- dsl/realize/samples compile under the strict flags; the native suites pass; the block-form lint passes.
- grep shows no `through(`, `MemberPath`, `memberRecorded` or `captureBinding` in model/framework/dsl or samples.
- A claim that writes a member fact without its member case fails to type-check (a fixture asserts it).

## Acceptance
- [ ] `through`, the member `records` overloads, `memberRecorded`, `MemberPath` and `captureBinding` do not exist in the DSL.
- [ ] A composition declares a per-member fact enum; the composed sample or fixture and SEMANTICS.md use it; native suites pass.
- [ ] A member fact written without its member case is a type error (fixture).

## Done summary
Phase 1 (DSL) of R17 is complete. The owner's `6983e0b1d7` already removed `MemberPath`, `through`, the member `records` overloads, `memberRecorded` and `captureBinding` from the DSL. Compositions declare a per-member `Fact` enum, and native composed steps carry the tagged cases (`Expect.test.scala` "tagged witness facts retain their member case across composition order"). `Witness.test.scala` "a composition's recorded fact requires its member case" is the type-error fixture for an unwrapped member fact. `b8dbcf4823` removes the remaining DSL-side `through`/member-key prose from SEMANTICS.md and model/README.md.

Verification: `grep -rn 'through(\|MemberPath\|memberRecorded\|captureBinding' model/framework/{dsl,samples,realize}` is empty. Baseline native run: 636 tests pass (`.flow/tmp/conductor-baseline-native.log`). Source lint passes 29 RuleSuite tests (`.flow/tmp/fn19011-rules.log`). GuideSourcesTest still fails on a Temporal excerpt path disabled by fn-188, a held suite unrelated to this change.

The lifter's `<state>.through.<path>.<def>` functions, the `<field>_<fact>` keys and the Go lowering stay in fn-190.12. The Temporal compositions migrate at the sweep.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6983e0b1d7, b8dbcf4823
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'
- PRs: