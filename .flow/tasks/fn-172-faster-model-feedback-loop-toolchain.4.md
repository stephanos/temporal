---
satisfies: [R4]
---
# fn-172-faster-model-feedback-loop-toolchain.4 Package the lifter once, and reuse the samples jar and classpath when their inputs are unchanged

## Description
Package the samples' batch lifter once and reuse the compiled samples jar/classpath across checks when their actual inputs are unchanged. Keep a small repository-local build cache, following established build conventions. Its validity covers source additions, removals and content changes, relevant build directives/tool versions and generated schema dependencies. Missing or invalid cached outputs trigger rebuild; failed builds must never publish a reusable entry. Every check still discovers roots, performs fresh independent sample lifts, adapts temporary IR and runs the existing Go checker. Preserve compiler diagnostic rejection and filename/source positions. No DSL, Model, IR schema or checked-in generated-output changes.

**Touches:** model/framework/check/Samples.scala, model/framework/check/Samples.test.scala, model/framework/check/Tools.scala, model/framework/check/Tools.test.scala, model/README.md

Use the simplest adequate cache and package launch supported by the pinned Scala CLI; inspect its actual help and established Make packaging commands before choosing flags. Do not add libraries. Retain normal compiler-verdict protections. Record cold/warm lifter and compilation stage wall times on identical inputs, with complete six-sample IR and Report parity against the previous implementation. Write the short author-facing cache explanation in the existing guide.

### Quick commands
- `mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite framework.check.ToolsSuite`
- `mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/check/Samples.scala model/framework/check/Samples.test.scala` (include any other edited Scala files)
- `make model-check-changed BASE=69907b317a` once at task end; cite `.flow/tmp/checks/20261013-131559/record.md` as the preceding baseline and retain mini step-table equivalence.

Full source-batch validation and independent review are deferred to the owner's batch boundary. Real features, broad lint/manual acceptance, regeneration and consumer work remain held. R5's Go prebuild/parallelism is the next separate task.

## Acceptance
- [ ] Samples compilation and packaged lifter are reused on unchanged inputs, including across distinct check output directories; roots and lifted outputs are always fresh.
- [ ] Source additions/removals/content changes, build/tool changes, missing outputs and failed compilation invalidate safely; compiler diagnostics still fail even with exit zero.
- [ ] Exact sample roots, fail-fast behavior, raw/adapted IR and all six complete Query/progress reports are preserved.
- [ ] Matched cold/warm timings and focused regressions demonstrate actual reuse, with a passing scoped check and mini equivalence record.

## Done summary
Unchanged framework sample checks now reuse the compiled samples jar/classpath and a packaged lifter across output directories. Content stamps cover source paths/bytes/presence, directives, generated API/IR jars, tool versions and compiler environment. Missing, altered or malformed outputs rebuild safely; rejected/incomplete builds leave no reusable stamp. A process lock protects builds through consumption. Roots, individual sample lifts, temporary Step adaptation and Go checking remain fresh.

Cold samples compilation took 0.591 s, lifter packaging 42.386 s and fresh lifts 8.276 s. The warm run reused both builds and lifted in 7.343 s, versus 20.09 s through the previous Scala CLI route (63.4% lower). Whole-pipeline before/cold/warm times were 129.372/146.383/114.899 s; variable unchanged Go work prevents assigning their entire difference to caching. All 18 fresh raw-IR/adapted-IR/complete-Report pairs are byte-identical, including 25 Query and nine progress receipts. Measurements and snapshot mapping are in `.flow/tmp/fn1724-verification.md`.

Verification preserves two red canonical Make records. `.flow/tmp/checks/20261013-160352/record.md` failed scalafix on mutable test counters, fixed with immutable deltas. `.flow/tmp/checks/20261013-160713/record.md` passed syntax, scalafix, boundary, six samples, mini compile/lift/step-table equivalence, 311 lifter tests and 164 check tests, but its overall verdict remained red because a repeated expected mock stderr diagnostic triggered the outer compiler-verdict guard. The owner accepted a narrow test-only continuation. The mock now uses captured stdout; independent stderr regressions and production `Ran.errors` remain unchanged. Final 20 focused tests, check lint, formatting and the exact 22 check suites passed. The latter passed through unchanged `Tools.scalaCli(...).orFail` with 164 tests, exit 0 and zero diagnostic errors. Its output/verdict are in `.flow/tmp/fn1724-final-guarded-check.log` and `.flow/tmp/fn1724-guarded-check-wrapper.log`. No final canonical Make pass or repeated lifter-stage credit is claimed.

The final fixture-only delta maps to the successful computational stages and exact parity above; source/compiler/lifter/checker behavior is unchanged. The owner's pre-existing Lift.scala header-only change selected the additional 47 lifter suites. The preceding baseline is `.flow/tmp/checks/20261013-131559/record.md`, with its original snapshot attribution retained. Owner plans, MILESTONES.md, Lift.scala and the unrelated README macro-design hunk remain preserved. Broad inherited Model/lifter lint, regeneration, real features and held Go consumer checks remain deferred.

Tier: session (jev-unavailable(no_key)); project tier override gpt-6.1-sol medium.
stage: impl-review - skipped(config: REVIEW_MODE=none)
stage: batch-review - skipped(policy: owner defers independent review to the complete Scala batch boundary)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 74eb6966e773c820c4098f1d9561985ad5ea588e, db99e466827b60fdd5dc9592223517fb137442d7, d030021a8f14a0c7dad78a64ebf64fbb59ddd083, 0a6f0f3a2eeb9ffb15540dd84f341193029f2e7a, 6aaed2289ad375e94d2c115848522671f3cbd75f, 0e638bb0959757c288561c7abaaa7c494c57e0a5
- Tests: mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite framework.check.ToolsSuite (20 final tests passed), make lint-model-check (final fixture passed), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/check/Samples.scala model/framework/check/Samples.test.scala model/framework/check/Tools.scala model/framework/check/Tools.test.scala, make model-check-changed BASE=69907b317a (retained red records 20261013-160352 and 20261013-160713; computational stages, 311 lifter tests and 164 check tests passed in the latter; canonical Make verdict red from synthetic mock diagnostic), mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.ModelSourcesSuite framework.check.RetainedSuite framework.check.SamplesSuite framework.check.ToolsSuite framework.check.equivalence.EquivalenceSuite framework.check.equivalence.InterpreterTablesSuite framework.check.equivalence.LiftsSuite framework.check.equivalence.StepLedgerSuite framework.check.lint.BlockFormRuleSuite framework.check.lint.BoundarySuite framework.check.lint.CommentRuleSuite framework.check.lint.GuideSourcesTest framework.check.lint.SyntaxRuleSuite temporal.check.CapabilityVocabularySuite temporal.check.CoverageSuite temporal.check.EqualitySuite temporal.check.GateEquivalenceOutputsSuite temporal.check.GateSuite temporal.check.GateLintSuite temporal.check.MakeFeaturesSuite temporal.check.SourceMetricsSuite temporal.check.lint.ProtoLiteralsSuite (guarded by unchanged Tools.scalaCli(...).orFail; 164 tests, exit 0, errors 0), baseline/cold/warm Samples.check runs and 18 exact baseline/warm raw-IR/adapted-IR/complete-Report comparisons; .flow/tmp/fn1724-verification.md, mini step-table equivalence passed in .flow/tmp/checks/20261013-160713/record.md; preceding baseline .flow/tmp/checks/20261013-131559/record.md, git diff --check
- PRs: