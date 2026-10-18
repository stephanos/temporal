---
satisfies: [R2, R3, R4, R5]
---
# fn-193-compositions-are-plain-scala-members-as.2 Plain compositions in the DSL; fixtures, tests and docs rewritten

## Description
**DSL completion phase (owner, 2026-10-17).** Implement the form the prototype chose: `Composition` without a state type parameter, members as vals, syncs/own/synced by action reference, fn-190 claim sections and scenarios, the chosen derivation form; native runtime over the members' var snapshots. Rewrite the framework composition fixtures and DSL tests, SEMANTICS §17 and the README composition section. Leave nothing in the composition implementation that needs `RecordMachine`, `Owner`, `Declaring` or a retired `DerivedComposition`. The lifter/IR/Go half follows with fn-190.12.

**Size:** L

### Execution notes from the accepted prototype

- Follow model/design/plain-composition-prototype.md's adopted spelling and its fixture migration map. Reuse the prototype's member/schema and sync work where appropriate, but generalize the existing on-demand NativeCheck.Search rather than copying the prototype's separate breadth-first explorer or introducing a synthetic RecordMachine adapter.
- R3 includes progress, monitors, scenarios, starts, fairness, witnesses and derivation-owned claims; the prototype's omitted forms are implementation work here, not a reason to narrow the acceptance gate. Preserve on-demand budget/cut behavior, vacuity and pinned-path ordering.
- R2 admits Machine and Composition members. A nested composition must therefore step and expose its member state and tagged facts correctly; accepting it at compile time and later refusing it as having no vars is the prototype's gap, not the final implementation.
- Preserve the existing fixture assertions through the plain-form rewrites, including reordered facts, derived end predicates, replaced-member claims and refused undeclared actions. Where the accepted prototype refuses a formerly written shape (repeated/shared vars or an unrelated replacement), retain the corresponding semantic control on a valid replacement fixture and add the refusal control.
- Widen the existing claim/target lookup for compositions rather than maintaining parallel copies of the claim words. Remove composition dependencies on RecordMachine, Owner and Declaring, leaving held export/IR adapters for fn-190.12.
- Keep framework/REFERENCE.md user-facing when updating the author surface: no Scala implementation/test filename citations or source/fixture columns, and preserve public-surface coverage.

### Verification
- `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests` passes; lint RuleSuite passes.
- Located refusals for a non-Machine member and a sync of an undeclared action (fixtures).

## Acceptance
- [ ] Compositions declare members as vals; no state case class, selectors or `->` pairing
- [ ] Composition fixtures/tests rewritten and passing natively; docs updated
- [ ] No composition code needs RecordMachine/Owner/Declaring

## Done summary
Authors can declare plain compositions with member vals, sync references, native claim sections, scenarios and witnesses, including nested members and named replacement derivations. The runtime uses the existing on-demand native Search over member snapshots and preserves input classes, result products, ordered tagged facts, first-member outcomes and deterministic paths.

R2-R5 coverage lives in PlainCompositionTest and the migrated composition fixtures. It includes located member/sync refusals, ambiguous and foreign targets, repeated/shared vars, invalid replacements, nested before and tagged facts, derivation-owned and explicitly selected source claims, vacuity, fairness/progress, monitors, starts, witnesses and budget cuts. Cross-jar controls preserve Derived member state metadata and exact Nothing/JoinedFact types. The initialized nested-replacement regression preserves immediate replacement refusals.

The four author documents describe the compiled forms. REFERENCE retains user-facing prose and public-surface coverage without Scala source citations. NativeClaimWords shares the existing Machine vocabulary. Inferred source types retained through Composition and Derived constructors replace class-body inspection; authored constructor spelling stays unchanged. Retired selector/composed-reference implementations were deleted.

Gate commands, observed exits, earlier reds and interrupted/inconclusive attempts are recorded in `.flow/tmp/checks/fn1932/record.md`; final log names and commands are in the adjacent evidence.json. The interrupted native run exited143 after a thread dump identified eager member-table state lookup re-entering initialization. The focused control passes after removing that lookup.

Final gates pass with 653 unfiltered native tests, 46 lint/boundary/reference/loop tests and 161 classified suites. Fresh core/plugin builds, realization-mode sample compilation, tagged separate-source/client verification and final formatting checks returned0. All processes exited before commit.

`make model-check-changed` remains held because it invokes temporal.check.changed. Authorized framework-scoped gates replace it. Composition IR/export/Go lowering, opaque replacement proofs, Temporal compilation and original full-mode var/effect IR proofs remain held for their owning Phase3 tasks. Frozen migration baselines and curated comparison samples were preserved.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none)

Independent review is deferred to the owner's phase batch. No review verdict is claimed. The changed author route is Composition member vals, sync/member references and named withMember derivations.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7f2f75ff44be6dcff96ce7ba7aa66de628cdc6e4, 9b94643894b1297cae1992b706cb9c69330ea430, b8a541c83d6e196f87a3ef4ecab7f7634a0fe72f
- Tests: baseline: green; mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests (644 tests; .flow/tmp/fn1932/baseline-native-observed.log, exit 0), timeout 600 mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn1932/closure-native-workspace model/framework/dsl model/framework/realize model/framework/samples --require-tests (653 tests; exit 0), timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1932/build mise exec -- scala-cli run --test --server=false --workspace .flow/tmp/fn1932/closure-check-workspace model/framework/check --main-class org.junit.runner.JUnitCore -- framework.check.FrameworkLoopSuite framework.check.lint.BoundarySuite framework.check.lint.BlockFormRuleSuite framework.check.lint.CommentRuleSuite framework.check.lint.SyntaxRuleSuite framework.check.lint.ReferenceSurfaceTest (46 tests; exit 0), timeout 600 env UMPIRE_FRAMEWORK_BUILD=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1932/build mise exec -- scala-cli run --test --server=false --workspace .flow/tmp/fn1932/closure-lint-workspace model/framework/check .flow/tmp/fn1932/Lint.scala --main-class compositionLint (161 suites classified; exit 0), timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn1932/closure-core-workspace --print-class-path --compilation-output .flow/tmp/fn1932/framework-closure.jar model/framework/dsl model/framework/realize (exit 0), timeout 600 mise exec -- scala-cli --power package --server=false --workspace .flow/tmp/fn1932/plugin-workspace --assembly --preamble=false --main-class-in-manifest=false --provided org.scala-lang:scala3-compiler_3 model/framework/reify/plugin -o .flow/tmp/fn1932/reify.jar (exit 0), timeout 600 mise exec -- scala-cli compile --server=false --workspace .flow/tmp/fn1932/closure-samples-workspace model/framework/dsl model/framework/realize model/framework/samples --scalac-option -Xplugin:/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1932/reify.jar --scalac-option -P:umpire-reify:mode=realization (six primary samples plus Inputs; exit 0), timeout 600 mise exec -- scala-cli test --server=false --workspace .flow/tmp/fn1932/nested-verified-workspace model/framework/dsl model/framework/realize --test-only framework.dsl.PlainCompositionTest --require-tests (9 focused controls; exit 0), task_cp=$(tail -n 1 .flow/tmp/fn1932/closure-core.log); task_deps=${task_cp#*:}; timeout 600 mise exec -- scala-cli run --server=false --scala 3.9.0 --workspace .flow/tmp/fn1932/closure-client-workspace --classpath "/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1932/framework-closure.jar:/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn1932/source-tagged.jar:$task_deps" .flow/tmp/fn1932/separate/Client.scala .flow/tmp/fn1932/separate/Run.scala --main-class framework.dsl.crossJarComposition (post/progress verified, two witness queries found, exact JoinedFact; exit 0), separate-source-durable.log and separate-client-durable.log: independently compiled source/client Nothing fact equality and Derived member before, post/progress verified (exit 0), task_files=$(git diff --name-only --diff-filter=ACM -- '*.scala'); timeout 600 mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf $task_files model/framework/dsl/CompositionMacros.scala model/framework/dsl/PlainComposition.test.scala (format-verified.log; exit 0), git diff --check (exit 0), HELD make model-check-changed: Makefile invokes temporal.check.changed; authorized framework-scoped substitute above, no Temporal/Go/schema/full-mode IR gate credit, Diagnostic history: .flow/tmp/checks/fn1932/record.md. Includes expected old API/target/replacement reds, cross-jar red-to-green, intermediate failures and interrupted initialization-cycle run exit 143; none earns pass credit., stage: impl-review - skipped(config: REVIEW_MODE=none)
- PRs: