---
satisfies: [R3, R5, R6]
---
# fn-149-safety-and-liveness-groups-for-object.7 Document grouped Scala authoring and commit the final source seal

## Description
**Size:** M
**Files:** Model author documentation, executable layout examples and final Scala grouping source seal.
**Touches:** [model/README.md, model/SEMANTICS.md, model/irgen/testdata/layout/**, model/irgen/test/**, model/framework/*.test.scala]

Finish the Scala authoring surface after .6 (R3/R5/R6). The conductor releases fn-164 only after this task's source seal is committed. Legacy .5 retains tools/umpire reporting docs, full consumer reconciliation, managed publication, gates, review and live closure.

### Approach
- Update Model examples and the executable layout with a safety postcondition and a bounded Progress claim, their distinct assumptions, and witness finding versus universal progress. Document classification by declaration kind, valid omitted groups and source-attributed refusals. Describe existing composition source support and deferred consumer evidence truthfully against fn-150's actual state; introduce no progress support.
- Lift and run the final template examples positively with nonzero independent declaration counts. Include every executable layout change in a final complete source/scratch-lift comparison against the frozen fn-140.10 baseline, composing .6's approved mapping.
- Recheck the independently frozen native/lift universe, applicable-row predicates, bounds/assumptions, Query/expected-run metadata, origins/waivers and monitor attachments. Keep exact identity/source-span mapping separate from StepTables' provenance-free projection; unexplained source differences hold the downstream lane.
- Run the spec's framework/Model/lifter/gate Scala source tests, compile and step-table equivalence. Keep checked-in-artifact comparison failures and strict inherited semantic failures separately attributed, never reported as a passing full suite or normalized away. Existing deferred .4/.5 retain the actual Query/progress verdicts, finite-bound counterexamples/incomplete checks, unsupported results, safety failures despite missing progress prerequisites, Cases and live/replay assessments.
- Commit the final source seal, baseline identity, complete inventory and mapping handoff. The source admission chain is fn-140.10 -> fn-149.7 -> fn-164 expectation/colon seal -> explicit pre-fault step comparison -> fn-123. Publishing and whole-spec closure are not source prerequisites.
- Preserve user-owned uncommitted documentation changes. If edits overlap them, reconcile explicitly with the conductor before changing or merging that file.

### Investigation targets
**Required:**
- model/README.md:33, :672, :923, :1222, :1295 - examples, claim sections and nesting rules.
- model/SEMANTICS.md - current Claims/Progress semantics and owner-edited prose.
- model/irgen/testdata/layout/lamp/product/Product.scala:36 - positively lifted executable author layout.
- model/irgen/test/Fixtures.test.scala - dedicated grouping and layout inventories.
- model/check/StepTables.scala and model/check/Gate.scala - source-table proof plus separate provenance ledger.

### Quick commands
```sh
mise exec -- scala-cli compile model/project.scala model/framework model/temporal
mise exec -- scala-cli test model/irgen --test-only '*PropertyGroupsFixtures*'
make lint-model-syntax
make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1
```

Also execute the framework/Model/lifter/gate Scala source suites with each source/managed-artifact population and result reported separately. Check formatting on changed files. No Go, managed regeneration, full gate, review, live run or consumer closure claim.

## Acceptance
- [ ] Author docs and positively compiled/lifted layout agree on safety/liveness grouping, explicit bounds/assumptions, witness/progress distinctions and actual composition support.
- [ ] Complete independently frozen final source/layout comparison preserves native semantics and full scratch-lift metadata under exact approved identity/span maps; every mutation control has nonzero coverage.
- [ ] Scala compile/source tests, syntax/format and step-table evidence are linked, with inherited strict semantic and stale-artifact failures reported separately rather than counted as green.
- [ ] The source seal and .6 mapping are committed before fn-164 admission; legacy .4/.5 remain open for every consumer, artifact, assessment, full gate, review and live obligation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
