---
satisfies: [R1]
---
# fn-172-faster-model-feedback-loop-toolchain.1 Warm scala-cli server: find out why --server=false is used, then drop it where safe (Samples, ChangeCheck, Makefile model targets)

## Description
Investigate the history and current behavior of `--server=false` in the R1 callers. Enable the warm scala-cli server for framework samples and guarded tool builds where the pinned toolchain is safe, retaining package-only Temporal Model-header and direct `-Werror` compiler-verdict protection with reasons. Do not change compile multiplicity, lifting roots, caching, Go checking or Model semantics.

**Touches:** Makefile, model/framework/check/Samples.scala, model/framework/check/Samples.test.scala, model/framework/check/ChangeCheck.scala, model/temporal/check/Coverage.test.scala, model/temporal/check/MakeFeatures.test.scala, model/README.md, .flow/tasks/fn-172-faster-model-feedback-loop-toolchain.1.*

### Investigation targets
Required: Makefile model targets; model/framework/check/{Samples.scala,Samples.test.scala,ChangeCheck.scala,Tools.scala,ModelSources.scala}; model/temporal/check/{Coverage.scala,Coverage.test.scala,MakeFeatures.test.scala,Gate.test.scala}; package-only-model-headers-need-normal-2026-10-06 memory entry.

### Quick commands
- `mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite temporal.check.CoverageSuite temporal.check.MakeFeaturesSuite temporal.check.GateSuite` (read JUnit's nonzero test count from the captured log).
- `mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/check/Samples.scala model/framework/check/Samples.test.scala model/framework/check/ChangeCheck.scala model/temporal/check/Coverage.test.scala model/temporal/check/MakeFeatures.test.scala`
- `mise exec -- scala-cli run model/framework/check .flow/tmp/fn172-r1/SamplesBenchmark.scala --main-class samplesBenchmark -- <before-or-after-output>`: actual `Samples.check` pipeline and captured Scala wall times, on the same post-R6 Go implementation and environment, with no Go memory cap.
- `mise exec -- go run .flow/tmp/fn172-r1/receipts.go <all-six-output/*.check.json>`: complete Go Report JSON before and after; compare reports and all six raw exported IR files byte-for-byte.
- `make model-check-changed BASE=965395859e50499fb5a787a67d8433c57d15d714` (post-change repository scoped integration gate; diagnose inherited failures separately without broad fixture migration).
- `make check-model-boundary`
- `make lift-model FEATURES=activity_mini LIFT_OUT=.flow/tmp/fn172-r1/final-lift`

### Verified observations for review
The matched post-R6 before/after sample pipelines passed with unchanged Scala sample sources and Go code. Scala wall times were 165.317 s and 42.942 s; complete Samples.check times were 253.344 s and 159.648 s. All six raw IR files and complete Go Report JSON files compare byte-for-byte under `.flow/tmp/fn172-r1/resume-{before,after}/`.

Final Make policy warms five check-only calls and retains fourteen disabled-server calls, eight for Temporal Model sources and six unguarded direct DSL/export `-Werror` calls. The unused-import fixture in `.flow/tmp/fn172-r1/compiler-error/` proves Bloop can print an error yet exit 0 (`unused-server.log`), whereas normal compilation exits 1 (`unused-normal.log`). Samples and ChangeCheck tool suites retain Tools.orFail/failed diagnostic rejection. Existing tool scalafix calls that were already warm are outside the nineteen disabled-server callers audited here.

The broad scoped integration gate passed 158 tests and mini step-table equivalence in 518.0 s at `.flow/tmp/checks/20261013-113357/record.md`, before the six conservative Make safety restorations. It is a pre-safety observation. Final targeted proof passes all 62 scoped tests, actual framework boundary compilation, and the restored direct mini lift; that mini's IR bytes match the pre-safety gate's lift. Logs are `final-tests.log`, `final-boundary.log`, `final-lift.log`, and `final-compiler-checks.log`. The corrected Make guard test was observed red when the lift flag was temporarily omitted (`unsafe-make-red.log`), then green after restoration. Final formatting passes (`final-fmt.log`).
## Acceptance
- [ ] R1: record the historical reason for disabled Bloop and reproduce/retain required package-only Temporal header protection.
- [ ] R1: framework Samples package, classpath and lift calls use the server safely; tool suites and safe Make tool/framework targets use it, while Temporal Model inputs retain their documented normal-compiler mode.
- [ ] Focused regressions cover the warm sample commands, conditional suite compiler selection and Make target modes without changing coverage policy or existing compiler-error enforcement.
- [ ] Before/after actual sample pipeline wall times are recorded on identical inputs; all six IR artifacts and complete Query/progress reports compare byte-for-byte, preserving identities.
- [ ] Scoped verification, formatting and implementation review pass; any inherited model gate failure is reported accurately and not repaired by a broad migration.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
