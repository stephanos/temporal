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
Enabled warm Scala compilation for guarded framework/sample calls without changing model outputs. On the same post-R6 Go implementation, the six-sample Scala stage fell from 165.317 s to 42.942 s (74%); the complete Samples.check pipeline fell from 253.344 s to 159.648 s (37%). All six raw IR files and all six complete Go Reports remain byte-identical.

Tier: session (jev-unavailable(no_key)); project implementer routing gpt-6.1-sol medium.
stage: impl-review - ran [2026-10-13T18:58:44Z..2026-10-13T19:02:00Z] (gpt-6.1-sol high; explicit same-family override; SHIP)

### Scope and safety

Samples package/classpath/lift calls now use the warm server. ChangeCheck retains normal compilation for Model modules and warms guarded tool modules. Five safe Make calls warm; fourteen retain --server=false. Eight consume package-only Temporal Model headers, which Bloop mishandles. Six direct -Werror calls lack Tools.orFail coverage, and an unused-import reproduction prints a compiler error while exiting zero with Bloop but exits one with normal compilation. Reasons are recorded in Makefile and model/README.md. Tools.orFail and the existing negative compiler-header regressions remain unchanged.

No compile-count, lift-root, checker, Model/DSL source, cache, or gate-policy changes were made. R2 through R5 remain separate work.

### Verification

The continuation baseline uses base 965395859e50499fb5a787a67d8433c57d15d714, after R6. The three prepared regressions failed for their intended assertions, then passed. The final guard regression was also observed red with a required lift flag omitted. Final focused JUnit selection passed 62 tests, formatting passed, actual model boundary compilation passed, and the restored mini lift passed. Its IR matches the earlier scoped-gate export.

The scoped integration command passed in 518.0 s with 158 tests across 22 suites, all six samples, and mini step-table equivalence. This was a pre-safety observation, before six conservative Make flag restorations. The final delta only restored original direct-compiler safety behavior; the final 62-test guard, actual boundary, mini lift, and IR comparison verify that delta. Samples and ChangeCheck were unchanged after the six-sample timing/parity proof. The conductor approved this focused final mapping instead of repeating the unaffected integration stack.

Before/after artifact directories are .flow/tmp/fn172-r1/resume-before and resume-after. Each contains six raw IRs, adapted IRs, complete Report JSON, and Scala timings. All twelve explicit byte comparisons passed. Reports contain 25 query receipts, including six negative controls, and nine progress rows across three named claims. Before package/classpath/lifts were 18.239/13.659/133.419 s; after were 4.622/0.873/37.447 s. Pipeline logs are resume-before.log and resume-after.log.

Final logs are .flow/tmp/fn172-r1/final-tests.log, final-fmt.log, final-boundary.log, final-lift.log and final-compiler-checks.log. The integration record is .flow/tmp/checks/20261013-113357/record.md. Source commit is 5399852996981d588fa08a6d512b2916bd9da2e0.

The initial pre-R6 attempt passed 60 baseline tests but the complete checker was OS-killed. Bounded retries were terminated and remained red/inconclusive. Its 103.177 s Scala observation is characterization, not the paired baseline. R6 resolved the checker allocation problem; the continuation's complete before and after runs both passed without memory caps.

### Review and limits

All three review axes returned SHIP with zero findings. The canonical receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-172-faster-model-feedback-loop-toolchain.1.json; run id 82a31d551fcf48a2becefa4298bb3d7d. Review inspected the complete output parity and compiler-verdict safety evidence. No review fix loop or memory capture was needed.

Gate classification is FULL because owner MILESTONES.md is dirty. A unittest green receipt was refused with NO_RECEIPT for that owner change; no skip or reusable receipt is claimed. Actual checks above passed. The broad repository/spec-end sweep remains the conductor's later gate. Inherited Go fixture paths and held framework.Step drift were not migrated. The mini export still differs from checked-in IR, while native equivalence passes; held regeneration was not folded into R1.

Owner MILESTONES.md, model/design/, sibling task planning and new fn173 planning remain unstaged. All worker-owned commands have exited.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5399852996981d588fa08a6d512b2916bd9da2e0
- Tests: baseline green after R6: actual uncapped Samples.check and complete Report dump; .flow/tmp/fn172-r1/resume-before.log (exit 0), regression red: explicit JUnitCore selection, 62 tests with three intended failures; .flow/tmp/fn172-r1/resume-red-tests.log, mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite temporal.check.CoverageSuite temporal.check.MakeFeaturesSuite temporal.check.GateSuite (exit 0; OK (62 tests); final-tests.log), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/check/Samples.scala model/framework/check/Samples.test.scala model/framework/check/ChangeCheck.scala model/temporal/check/Coverage.test.scala model/temporal/check/MakeFeatures.test.scala (exit 0; final-fmt.log), actual uncapped before/after Samples.check, complete Report JSON and all twelve cmp checks (exit 0; resume-before.log, resume-after.log), make model-check-changed BASE=965395859e50499fb5a787a67d8433c57d15d714 (exit 0; 158 tests; 518.0s; pre-safety observation), make check-model-boundary (exit 0; final-boundary.log), make lift-model FEATURES=activity_mini LIFT_OUT=.flow/tmp/fn172-r1/final-lift (exit 0; final-lift.log), cmp .flow/tmp/checks/20261013-113357/lift/activity-mini.json .flow/tmp/fn172-r1/final-lift/activity-mini.json (exit 0), required direct compiler safety guard red then green; unsafe-make-red.log and final-tests.log, git diff --check (exit 0), NO_RECEIPT: unittest receipt refused because owner MILESTONES.md is dirty; no gate skip claimed
- PRs: