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
Documented the kind-checked `properties.safety` and `properties.liveness` authoring contract and made the lamp layout an executable example with a safety postcondition, a bounded Progress claim, and its weak-fairness assumption. The dedicated layout test was written red first, then pins two Properties, one Progress declaration and one Assumption while preserving VERIFY versus FIND semantics.

The source seal is commit `956e046984`. Its complete layout comparison is `.flow/tmp/scala-dsl/fn149-7-proof/layout-comparison.json`: existing semantics are exact under the approved mapping, all 221 current positions resolve to real in-range sources, the five native StepTables are byte-identical, and mutations to existing semantics, a group root, the progress bound, its assumption and a source span are rejected. The layout inventory is 2 actions, 1 assumption, 21 functions, 5 machines, 1 progress claim, 2 properties, 2 queries, 1 realization, 2 scenarios and 8 types.

The inherited fn-149.6 frozen-universe proofs were revalidated rather than trusted. Native evidence covers 43 subjects, 163 Properties, 12 Progress claims, 246 Queries and 5,299,233 applicable predicate rows (1,776,924 enabled and 3,937,539 disabled); whole-group, predicate, bound, assumption and Query-population mutations are rejected. Lift evidence is exact under the frozen-to-WIP-to-grouped root/span ledger, keeps all three waiver files byte-identical, and rejects semantic, source and span mutations. A fresh production lift is byte-identical to the verified fn-149.6 output and contains 71 actions, 12 assumptions, 11 compositions, 646 functions, 40 machines, 6 monitors, 12 Progress claims, 238 Properties, 338 Queries, 14 realizations, 266 scenarios and 83 types.

Final green checks: Scala compile (1.006s); `PropertyGroupsFixtures` 13/13 (35.927s); `LayoutWitnessFixtures` 1/1 (23.051s); framework source suite 77/77 (~13.6s); gate source suite 97/97 (13.564s); syntax lint (12.145s); scoped formatting (0.439s); `git diff --check`; direct Scala steps-only equivalence over 46 nonempty tables (57.337s); fresh production lift (~14.8s); native capture/comparison (~4.7s plus ~3.7s revalidation). The canonical Make equivalence wrapper remains inherited red after 1.751s because the linked `realization.proto` differs from its current internal proto and asks for the forbidden `make protoc`; the task-authorized direct Scala command is green.

The combined framework/Model source run populated 185 tests: 183 pass and two inherited strict artifact-pin tests remain red because they expect the old split export set and omit the current taskqueue export. The full lifter run populated 78 tests: 60 pass, including all 13 grouped-property tests and the layout test, while 18 held managed-fixture/package tests remain red on pre-migration fixture source and stale diagnostic pins. These failures are separately attributed, are outside this task's Touches, and are not reported as green. Deferred fn-149.4/.5 retain managed regeneration, consumer/gate reconciliation, review, live execution and whole-spec closure; fn-150 retains generated composition-consumer evidence. No later task was admitted or modified.

Baseline: compile, grouped-property tests and syntax lint were green; the canonical Make route had the same inherited linked-protobuf failure, and its authorized direct Scala fallback was green over 46 tables. Base commit: `cbf3c275d8b3af52edb8e8aa429830552a4c6b6e`.

Tier: session (jev-unavailable(no_key))

stage: impl-review - skipped(config: REVIEW_MODE=none; policy: task/MILESTONES hold independent review during Scala-only mode)
stage: plan-sync - skipped(config: planSync.enabled != true; conductor owns milestone synchronization)
## Evidence
- Commits: 956e0469844647de4382371a830f29d595b2cd18
- Tests: mise exec -- scala-cli compile model/project.scala model/framework model/temporal (green; 1.006s), mise exec -- scala-cli test model/irgen --test-only *PropertyGroupsFixtures* (green; 13/13; 35.927s), mise exec -- scala-cli test --server=false --require-tests model/irgen --test-only *LayoutWitnessFixtures* (green; 1/1; 23.051s; prior intended RED captured), mise exec -- scala-cli test --server=false --require-tests model/project.scala model/framework (green; 77/77; ~13.6s), mise exec -- scala-cli test --server=false --require-tests model/project.scala model/framework model/temporal (inherited strict red; 183 pass/2 stale artifact pins; ~31s), mise exec -- scala-cli test --server=false --require-tests model/irgen (held managed-fixture red; 60 pass/18 stale pre-migration fixtures; ~166.6s), mise exec -- scala-cli test --server=false --require-tests model/check (green; 97/97; 13.564s), make lint-model-syntax (green; 12.145s), make umpire-check-equivalence BASE=.flow/tmp/fn160/baseline-ae3f36ee51 MAP=model/testdata/equivalence/batch-1d.map STEPS_ONLY=1 (inherited prerequisite red; 1.751s; linked proto requires forbidden regeneration), mise exec -- scala-cli run --server=false model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn160/baseline-ae3f36ee51 model/testdata/equivalence/batch-1d.map --steps-only (authorized fallback green; 46 tables; 57.337s), fresh production lift plus byte comparison to fn-149.6 (green; 4 IR and 3 waivers; ~14.8s), python3 .flow/tmp/scala-dsl/fn149-6-resume-proof/compare-native.py (green; all mutations rejected), python3 .flow/tmp/scala-dsl/fn149-6-resume-proof/compare-lift.py (green; all mutations rejected), python3 .flow/tmp/scala-dsl/fn149-7-proof/compare-layout.py (green; native and lift maps exact; all mutations rejected), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check changed Scala files (green; 0.439s), git diff --check (green)
- PRs: