---
satisfies: [R1]
---
# fn-196-the-scala-checker-is-the-oracle.1 DSL completion: the checker module outside framework.dsl, with one Result shape

## Description
**DSL completion phase (owner, 2026-10-17).** Supersedes fn-190.15's destination: the same move of the native runtime (`effectOf`, `guardHolds`, input-effect adapters, `Finite` enumeration, the `Domain` runtime, the budget runtime, `NativeCheck` and the var-snapshot runtime) out of `model/framework/dsl`, but into a product checker module (`model/framework/checker`, package `framework.checker`) rather than test support, because the Scala checker is the oracle. `framework.dsl` keeps declarations and the references the plugin reads and executes nothing. Expose the questions as `Checker.explore`, `verifyAll`, `monitor`, `progress` with one `Result` carrying verdict (`verified`, `refuted`, `found`, `notFound`, `unknown`), budget, completion and the counterexample or witness path with states and outcomes; a `Report` wraps results and warnings. Refuse a budget under one. Record in fn-190.15 that this task carries it.

**Size:** L

**Files:** `model/framework/checker/*.scala` (new), `model/framework/dsl/NativeCheck.scala` and the runtime files it moves, `model/framework/dsl/project.scala` and the checker module's `project.scala`, every dsl/samples/realize suite that imports the runtime.

### Verification
- A grep records no production source in `model/framework/dsl` invoking an effect or guard callable.
- The dsl, realize and samples native suites pass with unchanged test counts; lint and boundary suites pass.
- Quick: the README Native checks recipe.

## Acceptance
- [ ] Checker module outside framework.dsl; framework.dsl executes nothing
- [ ] One Result shape with verdict, budget, completion and path; budget refusal
- [ ] Native suites pass with unchanged counts; fn-190.15 annotated as carried here

## Done summary
Moved the native evaluator, on-demand checker, domain/input enumeration, budgets, snapshots, claims and current compositions into production framework.checker, retaining DSL declarations and callable references. Checker.explore, verifyAll, monitor and progress expose the shared Result/Report surface with concrete stored-edge paths, completion and budget refusal; four focused controls extend the unchanged original653 native tests to657.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none)

Fresh baseline native653 passed before edits. Final native657, framework checks46, documented lint31, real corpus162 classified suites, declaration-only DSL/realize compilation, README compile, fresh framework/plugin artifacts and combined-source plus separate-library realization sample compilation all returned0. All original passing test names are retained. Failed intermediate runs, exact commands and holds are recorded in `.flow/tmp/checks/fn1961/record.md`; no failed run is credited as green.

The plugin's existing implementation-package admission filter now includes the moved checker package; the initially failed combined-source sample compile and subsequent fresh-plugin green run are retained. No new generic traversal suppression, second explorer, runtime hidden in test support, or scanner exemption was introduced. The exact located Rules diagnostic pin moved625→626 only because of its new import. Declaration contexts moved together out of sugar Syntax, preserving public FQNs.

R1 remains shared with196.2: replay/find/seeded sample/new refusal contracts and gate-budget configuration are not claimed complete. `make model-check-changed` remains held because it routes to temporal.check.changed; the authorized framework substitute is the recorded native/check/lint/build/sample gates. Full exporter/equivalence/IR bodies, Go, schemas/generated IR, disabled Temporal models, frozen migration corpus and broad GuideSourcesTest remain held. No MILESTONES edits. Root already annotated190.15 as carried and owns its later closure. Old work/fn19015-runtime worktree/commits remain unmodified and unmerged.

The author-facing route now imports framework.checker.Checker; the README Native checks recipe and reference describe the product module and shared results. Independent review is deferred to the phase batch by explicit owner override; no review verdict is asserted.

Source commit:0dd5737301eaf2193d3ae89ea71a3dac2110c7ae. The range from persisted base60139e5a6a179188a0d45b6599bd3ec908f43be6 also contains root's config-only38e413cf35b7d50a1c600ec51a53898bd7d50b9e (deprecated ignored keys removed); it is not worker source work. Native and compile green receipts were recorded at the source commit. Sample-client's full concrete classpath recipe is preserved in the check record rather than expanded into a giant evidence command.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 38e413cf35b7d50a1c600ec51a53898bd7d50b9e, 0dd5737301eaf2193d3ae89ea71a3dac2110c7ae
- Tests: timeout 600 mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn1961/final-native-workspace model/framework/checker model/framework/samples --require-tests, timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1961/build mise exec -- scala-cli run --test --server=false --workspace .flow/tmp/fn1961/check-workspace model/framework/check --main-class org.junit.runner.JUnitCore -- framework.check.FrameworkLoopSuite framework.check.lint.BoundarySuite framework.check.lint.BlockFormRuleSuite framework.check.lint.CommentRuleSuite framework.check.lint.SyntaxRuleSuite framework.check.lint.ReferenceSurfaceTest, timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1961/build mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn1961/recipe-lint-workspace model/framework/check --require-tests --test-only framework.check.lint.*RuleSuite, timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn1961/recipe-compile-workspace model/framework/checker model/framework/samples, timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn1961/declarations-workspace model/framework/dsl model/framework/realize, timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1961/build mise exec -- scala-cli run --server=false --workspace .flow/tmp/fn1961/final-lint-workspace model/framework/check .flow/tmp/fn1961/Lint.scala --main-class checkerLint, timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn1961/core-workspace --print-class-path --compilation-output .flow/tmp/fn1961/build/framework.jar model/framework/checker, timeout 600 mise exec -- scala-cli --power package --server=false --workspace .flow/tmp/fn1961/plugin-final-workspace --assembly --preamble=false --main-class-in-manifest=false --provided org.scala-lang:scala3-compiler_3 model/framework/reify/plugin -f -o .flow/tmp/fn1961/build/reify-final.jar, timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn1961/samples-final-workspace model/framework/checker model/framework/samples --scalac-option -Xplugin:/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1961/build/reify-final.jar --scalac-option -P:umpire-reify:mode=realization
- PRs: