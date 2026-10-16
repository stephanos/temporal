---
satisfies: [R1, R3, R6, R8, R9]
---
# fn-165-inputs-as-records-named-classes.9 Migrate native record callers and retire token and arity APIs

## Description
Finish native caller/API migration using .7/.8. Retire the public token and positional arity ladder independently of held exporter adaptations.

**Size:** M
**Files:** framework native DSL tests, realize helpers, six samples and framework migration guidance.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md]

### Approach
- Consume .7 inventory and migrate native callers, examples and tests. Keep intentional compile-negative and independent core-oracle fixtures explicit.
- Remove public token declarations/input(token), Assigned/class :=, Slot/InputAt overloads and the old effect arity ladder with no compatibility layer. Preserve unrelated request-scope :=.
- Preserve .7's terminal scalar `.input[A](name = ..., default = ...)` declaration and actual scalar In; it is not the retired standalone token form. Native scalar callers migrate their verified labels/defaults explicitly rather than requiring one-field records or guessing provenance.
- Do not retain a legacy public API for disabled Temporal or held IR fixtures; original .3/.6 own exporter reader/capture retirement later.
- Document records, declared defaults, scalar/no-input forms and structural selectors in framework-local guidance and Core forms. Record native step/default/class-order pins and unrun integration obligations.

### Investigation targets
**Required:** model/framework/dsl/Action.scala; model/framework/dsl/Syntax.scala; model/framework/dsl/Inputs.test.scala; model/framework/dsl/InputDefaults.test.scala; model/framework/samples; model/framework/realize.

### Verification
Run touched native core/check selections plus external-author old-spelling compile refusals. Compare migrated native fixtures to .7/.8 independent behavior pins. No full exporter/IR gate or Temporal sources.

## Acceptance
- [ ] Native inventory has no unexplained old callers; public token/class :=/arity APIs are inaccessible from an external author package.
- [ ] Native class ordering, defaults, step behavior and custom domain witnesses retain the recorded independent pins; genuine scalar declarations/value calls/effects remain available with exact labels, without obligatory one-field records.
- [ ] Framework-local docs and Core forms describe the final API; request-scope := remains unchanged.
- [ ] Native checks pass; full R4/R6/R7 IR retirement and Temporal migration remain open.

## Done summary
Retired the native token, assignment and positional arity APIs and migrated framework callers to complete records or explicit scalar actions. Native execution retains actual input values, selected catalogs, declared defaults and independent ordering/behavior pins; framework-local guidance includes the migration recipe.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: session (jev-unavailable(no_key))

### Native acceptance

- The refreshed inventory at `.flow/tmp/fn1659-native-migration-inventory.md` and `.json` pins source `7e209546b2` and accounts for 34 migration cases across 45 classified files. The pre-edit base was `05ca5ba322dc84735bee026c754413fe3164ace5`; no native source changed between the inventory's source and this base. The final lexical sweep is `.flow/tmp/fn1659-final-lexical-sweep.log`. No unexplained native token/Assigned/Slot/InputAt/OtherInputs/LegacyAction caller or legacy RuleInput evidence remains. The writtenAction assignment string is explicitly historical, and scalar declaration misuse fixtures remain intentional negatives.
- Bank's Send fields and Voting's IncreaseMaxBal/VoteFor fields remain required and in their original order. ReliableBroadcast, TwoPhaseCommit, Channel and native metadata/realization fixtures remain genuine scalars with exact labels and selected witnesses. Scalar products, Tuple1, Tuple2 and EmptyTuple are actual scalar inputs, with no obligatory record wrapper or tuple exclusion. Channel delivery/loss retain the exact message witness and required label `message`.
- The existing native adapter remains the only input adapter. The public token classes/factories, append overloads, assignment class form, direct/rule arity ladders and runtime domain-count dispatch are removed. Empty calls no longer admit arbitrary widened Action[Tuple]. Zero-input binding and scalar EmptyTuple checks use declaration metadata; an EmptyTuple scalar cannot reopen declarations or acquire a state-only direct binding. These shape checks refuse at construction where static Action[EmptyTuple] cannot distinguish the declarations.
- Record omissions compile-refuse with supplied positive controls. Required scalar omissions still refuse at Machine, Scenario and realization script initialization with the exact label. False/zero/structured/Some/None defaults, ordered values, independent core Steps/PropertyDecls, callback/catalog timing, class restrictions, facets and rebinds retain their pins. The obsolete six-input cap is replaced by seven-field execution and lowering.
- Bare native class targets still refuse at the original call site and name the actual declaring object. RuleSource follows the compiler-retained authored Dynamic call before its synthetic receiver expansion; the existing qualification assertions were not changed. ActionDecl's intentional token-field deletion changes its product arity from 13 to 12; the realization fixture now pins all 12 field names as well as pointer/source identity.
- Request/protobuf `:=` and realization SlotOf remain unchanged. No compatibility layer was added for held exporter or disabled Temporal callers.

### Verification

Baseline: green. The native command below exited 0 with 301 executed tests before edits (`.flow/tmp/fn1659-baseline.log`). Lint exited 0 with 29 tests (`.flow/tmp/fn1659-lint-baseline.log`). The old receipt was not reused because a non-.flow path changed after its verified SHA.

- `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests`: exit 0, 304 executed tests, `.flow/tmp/fn1659-native-final2.log`. The net +3 is two new InputDefaults compile-contract tests, one EmptyTuple shape test and a separately named historical parser control, minus the retired standalone-token Param test. Input test replacements have net zero growth. Receipt `.flow/tmp/green-receipts/0daa2063-native-unittest.json` certifies only this native command.
- `mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'`: exit 0, 29 tests; native scan 102 files, zero findings. Log `.flow/tmp/fn1659-lint-final.log`.
- Scoped Scalafmt covered 28 changed Scala files; the second pass preserved every SHA-256 and fmt --check exited 0. Logs `.flow/tmp/fn1659-format-final{,-second,-check,-idempotence}.log`; manifest `.flow/tmp/fn1659-format-final-before.sha256`. `git diff --check` passed.
- `mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1659-external-recipe.sh`: exit 0, `.flow/tmp/fn1659-external-final4.log`. A fresh native jar, separate declarations jar and external runtime client passed. Ten separately compiled clients refused standalone token input, input(token), Assigned, Slot, InputAt, OtherInputs, LegacyAction, class assignment, the old multiargument effect and widened tuple omission. Each refusal pins exit 1, the call-site line and the actual diagnostic. The old effect reaches Dynamic's unsupported-class-call refusal at its binding line; its paired actual-input binding succeeds. Native controls cover scalar products/tuples, named/value records, generic bindings, no-input rules and request/protobuf assignments.
- `mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1659-external-channel-recipe.sh`: exit 0, `.flow/tmp/fn1659-external-channel-final3.log`. An external package uses the public channel factory with product messages and a custom true-before-false catalog. It pins one actual labeled message, the exact witness and object identities, required omissions, system/internal metadata and eight independently written ordered step lines for delivery and loss.
- Gate classification was FULL, exit 1 (`unmatched: model/framework/README.md`), `.flow/tmp/fn1659-gate-classify-final.log`. No docs-only skip or broad integration receipt is claimed.

Red-to-green evidence: `.flow/tmp/fn1659-empty-scalar-red.log` exposed interception of a valid scalar EmptyTuple callback; `.flow/tmp/fn1659-shape-guard-red.log` exposed declaration reopening; `.flow/tmp/fn1659-noinput-binding-red.log` exposed an input callback admitted on a zero-input action. The final native suite passes these controls. The first full run (`fn1659-native-final.log`) exposed the three unchanged owner-diagnostic assertions and the intentional product-arity pin; the owner-focused final run passed all 14 cases before the final full run. `.flow/tmp/fn1659-owner-tree.log` retains the synthetic receiver evidence; transient diagnostic code and the refuted traversal change were removed.

Earlier migration compiler errors, the invalid repeated test-selection option, external fixture operand/initialization mistakes, the initial old-effect diagnostic needle, and initial Channel fixture constructor/rendering mistakes are inconclusive attempts, not passing evidence or production regression proofs. Their attempt logs remain under `.flow/tmp/fn1659-*`. No assertion, source root, lint exemption or gate was weakened.

### Handover and held work

Implementation commit `0daa20639ada8adb3b9e39f5c018c010f3307d54`. The evidence range also includes conductor-owned plan-only commit `e56b3bfd15c5d2a4139c356347e154808e3e2d0e`, not .9 source. Six disjoint assistants handled caller/sample/docs surfaces; one returned an additional external-proof scaffold on follow-up. This worker owned production reconciliation, every compiler command and Git/Flow completion.

Only native .9 closes. Original .1-.6 remain held, including full R4/R6/R7 capture/reader retirement, exact IR/Param/Class/default equivalence and Temporal migration. No exporter/capture/schema/Go/backend work, regeneration, Temporal source reads, old Model/API jars, model/project.scala/toolchain edits, MILESTONES edits, broad Make gates or formal review are claimed. The owner retains batch-end review and next-task admission.

Tracker sync: n/a (bridge inactive).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e56b3bfd15c5d2a4139c356347e154808e3e2d0e, 0daa20639ada8adb3b9e39f5c018c010f3307d54
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1659-external-recipe.sh, mise exec -- bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1659-external-channel-recipe.sh, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/dsl/Action.scala model/framework/dsl/Channel.scala model/framework/dsl/CoreBindings.scala model/framework/dsl/CoreBindings.test.scala model/framework/dsl/DerivationBindings.test.scala model/framework/dsl/Expect.test.scala model/framework/dsl/FacetBindings.test.scala model/framework/dsl/InputDefaults.test.scala model/framework/dsl/InputExecution.test.scala model/framework/dsl/InputRecords.test.scala model/framework/dsl/InputSource.scala model/framework/dsl/Inputs.test.scala model/framework/dsl/Machine.scala model/framework/dsl/ResidualSignature.test.scala model/framework/dsl/RuleBindings.scala model/framework/dsl/RuleBindings.test.scala model/framework/dsl/RuleInput.scala model/framework/dsl/RuleSource.scala model/framework/dsl/Rules.test.scala model/framework/dsl/Signature.scala model/framework/dsl/Signature.test.scala model/framework/dsl/Syntax.scala model/framework/realize/Conversion.test.scala model/framework/samples/Bank.scala model/framework/samples/Expect.test.scala model/framework/samples/ReliableBroadcast.scala model/framework/samples/TwoPhaseCommit.scala model/framework/samples/Voting.scala, sha256sum -c .flow/tmp/fn1659-format-final-before.sha256, git diff --check
- PRs: