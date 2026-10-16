---
satisfies: [R6]
---
# fn-188-framework-only-source-iteration.3 Finish native DSL public-client verification and full batch review

## Description
Complete the native DSL batch gate required by parent R6. This is the next admitted priority-0 task, independent of the blocked broad exporter loop in .2. Native implementation checkpoints are already committed; do not reopen them or add new DSL features merely to fill this task.

**Size:** M
**Files:** model/framework/README.md; model/framework/dsl/**; model/framework/realize/**; model/framework/samples/**; model/framework/check/lint/**; MILESTONES.md; existing native public-client recipes/evidence under .flow/tmp.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/samples/**, model/framework/check/lint/**, model/framework/README.md, MILESTONES.md]

### Approach

- Reuse the native-only commands in model/framework/README.md and existing public-client recipes; no parallel verifier, task-specific Scalafix or new proof machinery.
- Retain the committed-source strict compile, 452 native tests, 29 lint tests, three boundary tests, actual boundary scan and 125-file format checks. Rerun only evidence invalidated by a source correction or otherwise demonstrated stale/incomplete.
- Finish and audit fresh separate-library public clients for record/scalar/channel inputs, domains/guards, Query configuration, durability, illustrations and Inbox ownership. Preserve every failed attempt. Verify intended refusal locations/reasons, not merely nonzero exit status. Historical partial continuations are not a complete passing recipe.
- Run one formal review of the entire native batch from baseline 437016b7d148b87cccee78c926af92119dec7fb4, not only the last fixture fix. Resolve demonstrated native findings with scoped checks and rerun affected final gates. Parallel read-only review/scouting is allowed; source integration and compiler/cache writes retain one coordinated lane.
- Record exact evidence and remaining proof debt in the task receipt and update MILESTONES.md in place. Native admission and literal callback examples do not prove reachability, full Query answers, checked-int64 interpretation, classification or graph equivalence.

### Boundaries

No model/temporal reads, restoration, compilation or migration, including disabled source contents. No broad make check-framework, model/project.scala, Temporal Model/API jars, capture/exporter repair, schema, Go, backend or regeneration. Existing generic IR dependencies do not authorize new IR work. Finishing this task does not release .2 or other held integration tasks without an explicit later scheduling decision.

## Acceptance
- [ ] Final-source native compile/test/lint/boundary/format evidence is complete and audited, with invalidated checks rerun after any correction.
- [ ] Independent public-client positives and intended runtime/compiler refusals pass using fresh native framework artifacts; no partial or stale continuation earns passing credit.
- [ ] Formal full-native-batch implementation review has a passing verdict and surviving native findings are resolved with verification.
- [ ] Receipt and milestone overview preserve deferred IR/Go/backend/Temporal obligations; native completion neither closes their requirements nor releases their holds automatically.


## Done summary
Verified the final native DSL batch and corrected five formal-review findings with regressions committed before the fixes. Refreshed native and cache-isolated public-client gates pass; the full native batch review returned SHIP with all five findings fixed.

Task: fn-188-framework-only-source-iteration.3
Tier: session (jev-unavailable(no_key))
Task base: 5891ec4128770627dbeef0288dbc4411ea92c8d9
Correction source: 461120e29d5b7d44c106dc57d735f215fae0cd5b
Formal reviewed range: 437016b7d148b87cccee78c926af92119dec7fb4..da598694bc2dc6106733b965de812681f6f8ccd4

baseline: green via owner-authorized retained final-native evidence at 2bf90a8ed7. Initial history inspection found only Flow metadata and milestone prose afterward. The explicit task retention policy applied; initial gate check had no honorable exact-HEAD receipt, and classification was FULL for unmatched MILESTONES.md. No fabricated docs-only or receipt-skip line is claimed. Native corrections then triggered fresh complete verification; no old-source or cached-author result substitutes for it.

Final native verification, terminal handle 92877, observed each exit once: strict compile with --test rc0; unfiltered tests rc0, 463/463 across 58 suites; scoped lint rc0, 29/29 across three RuleSuites; BoundarySuite rc0, 3/3; actual production Boundary.findings scan rc0, findings=0; formatting rc0, 126/126 files. Logs are .flow/tmp/fn1883-native-final-{compile,tests,lint,boundary-suite,boundary-production,format}.log. Standard exact-command receipts are .flow/tmp/green-receipts/da598694-unittest.json and da598694-compile.json. No executable native source changed after these gates; subsequent changes record evidence and guidance.

All five complete public recipes and the channel supplement passed with new Scala CLI workspaces for native/author/client/refusal stages. Durability terminal handle 47516 and remaining recipes handle 81285 both exited zero. Logs: .flow/tmp/fn1883-public-fn1659-fresh.log, fn1883-public-channel-fresh.log, fn1883-public-fn17311-fresh.log, fn1883-public-fn17312-fresh.log, fn1883-public-fn12310-fresh.log and fn1883-public-fn1537-fresh.log. Results: 158 numbered positives plus the action/channel clients, 41 intended compiler refusals and 33 runtime refusals. The action client additionally verifies same-step and transition ownership through ordinary methods in a separate author jar, emitting NATIVE_COMPILED_EXPECT_WRAPPERS_OK. Callback/state/inner/default-zero controls and the illustration origin line 14:20 remain enforced. Existing recipes validate intended refusal reasons and locations, not merely nonzero exit.

An initial post-fix durability runtime failed with NoSuchMethodError: javap showed old eight-argument StateField.make calls in cached author bytecode against the new nine-argument native method, despite both package stages returning zero. Scala CLI 1.17.1's documented --workspace option now isolates every complete recipe. Refreshed javap confirms the new signature. Failed/cached artifacts are preserved in .flow/tmp/fn1883-cached-public-attempts.tar.gz; pre-review-fix public evidence is fn1883-before-review-fixes-public.tar.gz. Earlier failed attempts and original StepTables fixture bytes remain retained as described in .flow/tmp/fn1883-native-gate-audit.md. Framework README guidance records the freshness caveat permanently; no compatibility shim, weakened flag/gate or cache deletion was used.

The review fixes and their focused regressions are:

- BudgetInputShapeSuite: zero-input budgeted rules refuse binary guards/effects; record/scalar callbacks remain typed, including scalar EmptyTuple. Widened state-only scalar rules retain their safe existing adapter behavior.
- InputDefaults: copied values reconstruct the replacement declaration's record; identical-shape metadata copies retain exact input identity.
- InputFieldsTest: widened bounded-field selectors refuse; transparent aliases and applied generic fields retain exact type evidence.
- expectauthor.ExpectExternalTest: same-step/transition method wrappers retain composition ownership even in unreachable branches; inaccessible helper-local/parameter predicates refuse without rejecting ordinary collection readers. Separate compiled-author controls also pass.
- NativeCrashSuite: root sibling resets and nested resets plus own numeric-budget consumption reconstruct final records atomically. Existing exhaustion, negative-budget, owner/path/alias validation and callback-zero controls remain intact. The poison-reset test now observes in-memory replacement calculation with unchanged counter/exception assertions.

Formal receipt: .flow/tmp/native-dsl-batch-impl-review.json. First round used correctness/contracts/integration fanout rid 425e5d9c8c91ae952eaac193d649478e; every sidecar was read, the duplicate budget finding collapsed, and official finalize retained five introduced findings. The same-receipt single re-review returned SHIP at round two with all five fixed and zero new blockers (handle 52039 terminal rc0). Logs are fn1883-native-review-dispatch.json, fn1883-native-review-finalize.json and fn1883-native-rereview.json; raw fanout sidecars remain under .flow/review-fanout/425e5d9c8c91ae952eaac193d649478e/. Backend execution metadata records codex:gpt-6.1-sol:high, read-only; the reviewers are from the writer's model family, not a cross-family review. Four delegated source/test workers handled disjoint fixes; compiler and commit ownership remained singular.

Defect route:
- prior fixes: exclusive source handoff, clean initial tree, recent affected-path history and bug-memory searches found no competing local fix. Open PRs/issues were unchecked because gh returned HTTP 401 Bad credentials; tracker is inactive.
- diagnosis: red3 observed accepted widened fields and budget callback shapes, wrong-record carrier admission failure, and both root/nested crash constructor failures; expect-red observed missing same/across owner refusals and missing helper-scope refusals. Initial red/red2 fixture compilation attempts are inconclusive and preserved. Green iterations are preserved; no failed attempt earns final credit. The widened-scalar cast hypothesis was refuted by actual safe execution, so no speculative refusal shipped.
- introduced by: carrier retention 841fb0b583; budget firing evidence 36e1fe870f; selector capture 86c9a9049f; original-only helper forwarding 5ac555663d; sequential crash reset 7d6e144345. Bisect skipped: no known-good revision of these newly introduced cases was supplied. PR attribution unavailable due to gh authentication failure.
- base: failing-test commit 8ec58f5230123e3225eec884d4618e8734d41ddf; .flow/tmp/fn1883-native-regression-red3.log rc1 and fn1883-native-expect-red.log rc1. Head: correction source 461120e29d, all 463 unfiltered native checks and separate public controls pass with refreshed logs above.
- live: no interactive surface; this is native library verification.

Only parent R6's native batch gate closes. Fn-188.2 remains blocked and parent fn-188 stays open; fn-174.24 and all integration/Temporal holds remain. Six retained sample fixtures stay pinned to 53f05f9bce8434ee069ec7717212700217358bbc. Native Int is host-32-bit smoke, illustrations are hypothetical admission rather than reachability/truth/classification, Voting guard removals are approved deltas, Cache history is many-to-one and authored Broadcast receive does not dequeue. Full Query answers, universal replacements, graph/budget/classifier/IR/schema/lift/capture/export, engine/Go/backend and Temporal obligations receive no completion or admission credit. No excluded source body, broad model gate, regeneration or Go/backend task was consumed or released.

stage: impl-review - ran [2026-10-16T13:29:58Z..2026-10-16T14:06:35Z] (official NEEDS_WORK -> SHIP; full native batch, same receipt, no optional phases)
stage: memory-capture - ran (bug/runtime-errors/reconstruct-crash-resets-and-budget-2026-10-16; bug/build-errors/isolate-scala-cli-author-builds-after-2026-10-16; low overlap)

Plan-sync: disabled. Tracker sync: n/a (bridge inactive). No source/compiler/review command remains running; the coordinated lane is released after the committed task receipt is verified.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a5034df3d4befd6e8ce790130948c3459f209c77, 8ec58f5230123e3225eec884d4618e8734d41ddf, 461120e29d5b7d44c106dc57d735f215fae0cd5b, da598694bc2dc6106733b965de812681f6f8ccd4
- Tests: mise exec -- scala-cli compile --server=false --test model/framework/dsl model/framework/realize model/framework/samples, mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only framework.check.lint.BoundarySuite, mise exec -- scala-cli run --server=false model/framework/check --scala-snippet 'object NativeBoundaryBatch { def main(args: Array[String]): Unit = { val _ = args; val findings = framework.check.lint.Boundary.findings(java.nio.file.Paths.get(".").toAbsolutePath().normalize(), Seq("temporal", "io.temporal.api")); require(findings.isEmpty, findings.mkString("\n")); println("NATIVE_BOUNDARY_OK findings=0") } }' --main-class NativeBoundaryBatch, mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/dsl model/framework/realize model/framework/samples, mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1659-external-recipe.sh, mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1659-external-channel-recipe.sh, mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn17311-external-recipe.sh, mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn17312-external-recipe.sh, bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn12310-external-recipe.sh, bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1537-external-recipe.sh, git diff --check
- PRs: