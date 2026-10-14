---
satisfies: [R3]
---
# fn-166-mini-activity-a-small-complete-model-to.2 Mini activity: feature file and product level by subtraction from standalone

## Description
The mini activity's feature file and product level, by subtraction from `features/activity_standalone/` ((R3)).

**Size:** M
**Files:** `model/temporal/features/activity_mini/MiniActivity.scala` (signature: `client.start` with `scheduleToStart`, `startToClose`, `maxAttempts` bounded at two; `client.cancel`; the worker's `poll`, `respondCompleted`, `respondFailed`; the two timers; entity, outcomes, `Bounds` entries), `mini/product/Product.scala` (phases with statuses and status facts, `init`, `end`, facets `Dispatching` and `Completion`, capabilities `Closable` and `Pollable`, one `except` waiver with a reason, Properties, one `verify`, one `find`)
**Touches:** [model/temporal/features/activity_mini/**, model/temporal/Bounds.scala, model/framework/IrFile.scala, model/framework/IrFile.test.scala, model/irgen/Structure.scala, model/irgen/Lifting.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/layout/mini/**, model/testdata/binding-order.tsv, model/temporal/ExportedLaws.test.scala]

### Approach
- Add `mini` to the existing feature-form discovery and layout support, with focused regression coverage, so both compiled-root discovery and lifting name the form `activity-mini`. This enables the new example without changing the DSL's behavior or editing real features.
- Copy the standalone feature's declarations and delete: pause, unpause, reset, heartbeat, describe, the by-ID client, their phases, facts, effects, rules, Properties and Queries. Keep the names of what stays so the real feature's vocabulary is the mini feature's.
- Two facets at the product level, with one rule in `Completion` whose guard reads a `Dispatching` phase, so the cross-facet case exists from the start.
- The feature file's header comment lists the hard instances the mini feature holds and where each is, and is updated by tasks 3 and 4.
- Lift with `make model-mini`; read `activity_mini/generated/ir.json` end to end once and record its size on the task.
- Add only the new mini machine's independently derived binding-order row to `model/testdata/binding-order.tsv`, because exported source laws require every newly exported machine to have one. Preserve every existing row byte-for-byte and leave table pins, equivalence maps/baselines and real-feature artifact inventories untouched. Derive the expected mini order from its declarations, not by copying the test's observed output; retain the existing mutation controls. This is required verification data for the new feature, not a revision of existing pins.
- Repair scoped source-law inventory filtering for non-feature machines: retain all inventory rows within the packages of actually exported machine subjects, so a compiled but unqueried actor does not contaminate a selected mini run. Preserve full-scope coverage and exact missing/extra/order checks inside admitted packages, with negative controls. Do not add an unqueried actor to discovery merely to satisfy the test, alter real rows, or weaken the production DSL gate.
## Acceptance
- [ ] The feature file and product level compile, lift to `features/activity_mini/generated/ir.json` and pass the Go checks and the lint under `FEATURES=activity_mini`.
- [ ] The product level holds two facets with a cross-facet guard, `Closable` and `Pollable`, one waiver with a reason, one `verify` and one `find`, with bounds in `Bounds`.
- [ ] The header comment lists the hard instances present so far.


## Done summary
The mini activity gives Model authors a Product-only feature they can compile, lift and check before adding a Protocol or realization. It keeps the three bounded start inputs, cancellation, shared worker answers and two deadlines in two facets, with a cross-facet guard, Pollable, Closable, one reasoned waiver, a bounded completion witness and terminal verification.

Tier: session (jev-unavailable(no_key)); project implementer routing retained

Verification
- Baseline focused IrFile discovery was green (8 tests). Canonical `make model FEATURES=nexus/workflow MODEL_GATE_ARGS=--skip-lint` failed pre-edit on the owner-held linked claim protobuf prerequisite. Direct Scala fallback is authorized by the current spec; no canonical Make credit is claimed.
- Selected compile, library package, classpath, lift, Go Check and IR diff completed in a 36-second warm direct loop. Stage times were 9/7/7/12/1/0 seconds. See warm-observation.md and warm-*.log.
- Read the complete generated IR, 109,475 bytes, including all 16 functions. Three exercised Go receipts over 25 enabled rows report two verified-within-limits results and one found completion witness. No realization or executable Case was authored or pinned.
- Fresh `make test-model FEATURES=activity_mini` passed 100 tests; fresh default `make test-model` passed 184 tests. Scoped non-feature binding inventory keeps every row inside exported packages. Missing entries, unknown extras and order permutations remain failing negative controls.
- The original full lifter command exited 1 with 194 passing tests, two failures and one existing skip. The new mini fixture's empty general header and the exact generic-discovery list caused those failures. The mini fixture recovery passed all three layout variants (1 test); the fresh affected Fixtures class passed 114 tests with its existing case-variant-header skip. Together with the unchanged 81 tests' full-run receipts, these recoveries cover the complete lifter inventory at the final checkpoint. The original full run is not described as green.
- Every existing binding-order TSV row remains byte-identical; only the independently derived mini row was appended. The relocation oracle still proves 44 original payloads byte-identical and unique, and 340 manifest rows unchanged modulo paths.
- Lint NOT RUN under the owner's fn-166 override. Existing lint companions and default policy remain unchanged. Real-feature IR remains preserved and behind the held schema, never regenerated to manufacture a pass.

Scope and review context
- Implementation stays in task 2's current declared Touches, including the root-approved additive binding row and scoped source-law test repair. Existing DSL semantics, real features, table pins, equivalence baselines and schemas remain unchanged.
- Root-owned fn-167 planning, MILESTONES and task-2 Touches updates are authorized checkpoint passengers. Their independent plan review is `.flow/tmp/fn167/plan-review-receipt.json`; they are not task-2 production changes and do not require another plan review here.
- The configured task-2 writer/reviewer routing is same-family GPT. The explicit `codex:gpt-6.1-sol:high` review covered `7abed6df93667a8a87313cb8355f10650114fc94..eab963b03d87c8e31cd8c750ef601603ed3d7a8c`. All three fresh read-only draws returned SHIP with no findings; finalize returned SHIP. Reviewer sandbox-blocked runtime attempts are not test passes. Receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-166-mini-activity-a-small-complete-model-to.2.json`. No task-1 SHIP is inherited.

stage: impl-review - ran [2026-10-11T22:11:27Z..2026-10-11T22:16:41Z] (SHIP; codex:gpt-6.1-sol:high; three draws, no findings)

Post-review gate classification: FULL (unmatched MILESTONES.md). The exact fresh source-law suites and affected lifter recoveries above remain current: review introduced no executable changes. No green receipt is minted for the red canonical Make prerequisite or original full lifter run.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: eab963b03d87c8e31cd8c750ef601603ed3d7a8c
- Tests: baseline: green focused IrFileDiscoveryTest (8); red canonical make model FEATURES=nexus/workflow MODEL_GATE_ARGS=--skip-lint (owner-held claim protobuf prerequisite), bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn166/task2/warm-loop.sh: PASS compile/package/classpath/lift/Go Check/IR diff; 36s (9/7/7/12/1/0), make test-model FEATURES=activity_mini: PASS 100 tests, make test-model: PASS 184 tests, mise exec -- scala-cli test --server=false model/irgen: EXIT 1; 194 pass, 2 failures, 1 existing skip; original full-run receipt retained, mise exec -- scala-cli test --server=false model/irgen --test-only '*MiniLayoutFixtures': PASS 1 test, all 3 variants, after isolated fixture correction, mise exec -- scala-cli test --server=false model/irgen --test-only 'umpire.irgen.Fixtures': PASS 114 tests, 1 existing skip, after exact mini discovery additions; unchanged remaining 81 tests retain full-run passes, mini generated Go Check: PASS 3 exercised queries/25 enabled rows; 2 verified and 1 found; no-update umpire-gen-cases publication check PASS, node .flow/tmp/fn166/recover-relocation.mjs: PASS 44 original payloads/340 manifest rows preserved, binding-order preservation oracle: PASS every old TSV byte unchanged; one independently derived mini row, Lint NOT RUN: owner fn166 override; canonical Make NO CREDIT; real-feature schema hold preserved
- PRs: