---
satisfies: [R1]
---
# fn-193-compositions-are-plain-scala-members-as.1 Prototype the plain composition form and record the verdict

## Description
**DSL completion phase (owner, 2026-10-17).** Prototype members as vals, syncs by action reference, claims in sections, a one-member derivation and the native runtime stepping a sync, on the framework's composition fixtures (`Joined`/`Together`/`Reordered` in model/framework/dsl tests). Keep the probe under model/design with a README and run.sh; write the verdict note in model/design.

**Size:** M

### Verification
- The probe runs from its run.sh; the note states a verdict for each Architecture bullet of the spec.

## Acceptance
- [ ] Design note and probe in model/design with a verdict per Architecture bullet

## Done summary
The prototype and verdict note are in `model/design/plain-composition-prototype.md`, with the probe in `model/design/plain-composition-prototype/`: run.sh, README, logs. Commits are `68144edeed` and `a046f78b1a`, merged as `4316db2f1b`.

The plain form works natively: members as vals, a sync by action reference (`left(action)` when two members bind it), claims in sections, and `Composition[F]` with `withMember(right = ...)` replacing `DerivedComposition`. `own`/`synced`/`Composed`, selectors, `Data`/`StateData[S]` and `Owner`/`Declaring` go.

run.sh passes: compile, 12 tests, report. Native suites pass 637. No framework source changed.

Open for fn-193.2:
- Progress, monitors and `only`/`fair` over compositions.
- Nested compositions.
- The `replaces` spelling.
- A Machine allowed only once per composition (TwoPhaseCommit-style same-entity members).
- Generalising NativeCheck.Search over a successor function.
- Avoiding inline calls to `@publicInBinary` private constructors (Scala 3.9.0 inliner hang).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 68144edeed, a046f78b1a, 4316db2f1b
- Tests: model/design/plain-composition-prototype/run.sh, mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests
- PRs: