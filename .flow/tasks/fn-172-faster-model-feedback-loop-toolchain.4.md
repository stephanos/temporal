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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
