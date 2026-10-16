---
satisfies: [R2, R3, R4, R6, R7, R8]
---
# fn-164-expect-blocks-for-properties.10 Finish native expect callers, public API and colon authoring

## Description
Finish the framework-native portions of R2-R4/R6-R8 after .2. Original .3/.7-.9 retain their exact IR, exporter and complete migration proofs; this checkpoint cannot close them.

**Size:** M
**Files:** model/framework/dsl/{Claims,Syntax,Machine}.scala, native tests, realize, samples, check/lint and framework guidance.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/samples/**, model/framework/check/lint/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Refresh .flow/tmp/fn16410-native-caller-inventory.md after committed .2. At checkpoint 8d45f95573 it pins 33 native finisher calls (22 sample Properties, six IrFileDiscovery callers, five internal pattern constructors), with live .2 additions explicitly unstable. Account for every author caller, internal constructor, core oracle and intentional negative fixture. Migrate author callers to expect, preserving predicates, explicit contextual names, exact selectors and native truth/exception behavior. Keep direct core oracles independent.
- Remove public holds/holdsAcross together, including renamed/deprecated compatibility routes. Retain private/direct core construction without core importing sugar. Disabled Temporal and held exporter fixtures do not constrain the final public API.
- Consume .11's native colon/block lint guard, then convert native authoring examples to colon blocks with the pinned formatter. Preserve language-required braces and deliberate invalid fixtures. Check formatting twice for idempotence and run the guard over a nonzero real native corpus. Do not duplicate the scanner implementation or restore Temporal sources to make its scan run.
- Update Core form comments and framework-local migration guidance. Deferred export fixtures adapt later; do not keep a public legacy API merely to compile held IR tests.

### Investigation targets
**Required:** model/framework/dsl/Claims.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Witness.test.scala; model/framework/dsl/Effects.test.scala; model/framework/check/lint/BlockFormRule.scala; model/framework/check/lint/SyntaxRule.scala.

### Verification
Use .2's isolated native compile/test recipe. Run touched Expect, Witness, Effects, Rules and lint selections with nonzero executed counts; compare old independent core predicates with new expect predicates on explicit truth/exception cases. No make check-framework, capture/IR, Go, schema, regeneration or model/temporal inputs.

## Acceptance
- [ ] Native callers use expect or inventoried independent core constructors; external author-package compile negatives reject holds/holdsAcross and compatibility routes.
- [ ] Native equivalence pins preserve selector/parameter identity, predicate truth values, holes and short-circuiting across witnesses, patterns and monitors.
- [ ] Colon-style scanner visits nonzero native source inputs and rejects reintroduced optional brace DSL blocks without comment/string or language-required-brace false positives; second formatter pass is unchanged; guidance matches final API.
- [ ] Focused native/check suites pass; original .3/.7-.9 exact IR obligations and all Temporal migration remain open.

## Done summary
Finished the native expect author surface: migrated all 22 sample Properties and six discovery callers, removed public holds/holdsAcross together, and retained named predicate member evidence through final check forwarding. The native corpus uses colon blocks and is guarded by the existing BlockFormRule; independent core oracles preserve sample, pattern and monitor behavior.

Task: fn-164-expect-blocks-for-properties.10
Tier: AGENTS implementer gpt-6.1-sol at medium (configured; executed model metadata unavailable).
stage: impl-review - skipped(config: REVIEW_MODE=none)

### Native acceptance

- R2/R3/R6: all migrated sample predicates retain explicit `after` or `before, after` contextual names, with sourceParameters asserted for every declaration. Eight restricted sample declarations retain exact-class `after` selectors. Ordinary functions, core stored `.holds`/`.holds2` predicates and witness construction remain independent of sugar.
- R4/R6: external-author tests reject holds, holdsAcross, private core constructors and an expect(BooleanFunction) compatibility route. Named Function1/Function2 expectations preserve transitive exact field/machine evidence through final expect/check wrappers, including unreachable wrong-owner references, reordered constructors and repeated machines. Never/unless preserve the same attachment refusals. The typed macro retains actual function references without executing their Boolean bodies at construction.
- R4: six sample tests compare all 22 migrated declarations to independent direct PropertyDecl ordinary-lambda oracles on explicit true/false cases, wrong facts and transition guards. Existing witness truth/key/short-circuit proofs remain. The added pattern/monitor bridge test compares direct core semantics, same-step/across sticky monitors and readAfter, including reached/skipped holes. Existing Expect tests retain unrelated exception/control-signal propagation and before/path/step/mutation refusals.
- R7/R8: framework-local author and migration guidance describe the final heads, typed contexts, field-reading rule, records versus Boolean queries, alternatives, exact composition identity and native commands. Core-form comments describe private construction. Native scan visits 96 files with zero findings; the existing fixture tests still refuse optional braces and parenthesized on cases while accepting comments, strings, required braces and colon nesting.

### Baseline and verification

Baseline: green. Before source edits, native dsl/realize/samples compiled and RulesTest executed 25 tests, exit 0 (`.flow/tmp/fn16410-baseline-rules.log`). The earlier Expect baseline's output handle was suppressed; its observation is inconclusive and earns no exit-code credit. It was not rerun solely to observe the result.

Fresh final native command, exit 0:

`mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests`

276 executed tests in `.flow/tmp/fn16410-native-final-all.log`. Relevant counts: ExpectTest 16; ExpectExternalTest 3; SampleExpectTest 6 (all 22 sample Properties); WitnessTest 6; EffectsTest 8; RulesTest 25; SugarNativeTest 5; IrFileDiscoveryTest 8; FacetsTest 12; RuleSourceTest 14; RuleBindingsSuite 10; realization construction 19 and conversion 26. Every discovered native suite ran; no zero-selection pass.

Fresh final check command, exit 0:

`mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'`

29 executed tests: CommentRuleSuite 3, BlockFormRuleSuite 7, SyntaxRuleSuite 19 (`.flow/tmp/fn16410-check-final.log`). Its native corpus scan visits 96 sources and requires zero findings.

Pinned formatter on 106 native/check Scala files: final first pass, second pass and fmt --check all exited 0. The second pass left every source byte unchanged: `.flow/tmp/fn16410-format-before-second.sha256`, `.flow/tmp/fn16410-format-idempotence.log`, `.flow/tmp/fn16410-format-final-{first,second}.log`, `.flow/tmp/fn16410-format-check.log`. Required braces and deliberate invalid source strings remain. `git diff --check` passed.

Red-to-green proof: `.flow/tmp/fn16410-forwarding-red.log` failed the expected wrong-owner exception before dependency retention; the same external selection passed after it. `.flow/tmp/fn16410-api-samples-red5.log` failed only the public-finisher refusals (indexes 0 and 1), while all sample/core-oracle cases passed; final removal passed. Main-source samples exposed a private generic macro constructor visibility failure; the binary-visible private core factory fixes it without exposing author construction. One RulesTest diagnostic pin deliberately moved 715:12 to 721:12 after colon migration; the exact equality remains and the final native suite passes.

### Boundary and handover

Inventory refreshed at `.flow/tmp/fn16410-native-caller-inventory.md`: admission 95 native files, 46 finisher calls plus one invalid nested fixture; final 96 files, only the two intentional external old-finisher negatives remain. The original preparation tables are explicitly historical. The source commit is `5ac555663d`; the base is `33a852c0c4c8f37266def56c2956d3e6912340fd`. The evidence range also includes the conductor's intervening plan/MILESTONES-only commits.

Gate classify returned FULL, `unmatched: MILESTONES.md`, exit 1 because the range includes conductor planning commits. The owner-amended native gates above are authoritative for this task; no broad green receipt or SHIP verdict is claimed. Review remains at the owner's source batch boundary.

NOT RUN (owner-held): make check-framework, make model-check-changed, broad exporter/capture suites, exact IR equality/reification, original spec Quick commands that consume model/project.scala or model/temporal, Go/backend/schema work and regeneration. Held exporter fixtures can still name removed finishers; no compatibility shim was kept for them. Original .3/.7-.9 and all Temporal migration remain open; this is not whole-spec closure. Native .10 closure releases fn-165.7's explicit admission hold for the conductor to reconcile.

Three disjoint assistants handled native docs, samples/oracles and colon authoring; the worker alone owned compiler commands and commits. All assistants returned. No compiler command remains running; compiler/Git lane is released after the committed Flow receipt.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 96ce220724feacf73901b9310c705d95f205b44c, 4fe615fdb9357eb4482131c967a419a4217c0f81, e1483c6dcf592b747934f19780f23d674ed00500, 647f8cde3400dbb2a159cbc2e577e24fde4b7620, 5ac555663dc4b70cea3297cd772679c261bf7ce8
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/dsl model/framework/realize model/framework/samples model/framework/check/lint, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf model/framework/dsl model/framework/realize model/framework/samples model/framework/check/lint (two passes; SHA-256 comparison unchanged), sha256sum --check .flow/tmp/fn16410-format-before-second.sha256, git diff --check
- PRs: