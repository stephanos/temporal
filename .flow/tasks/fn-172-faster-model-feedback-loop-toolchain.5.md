---
satisfies: [R5]
---
# fn-172-faster-model-feedback-loop-toolchain.5 Prebuild the Go sample checker and check the samples in parallel

## Description
Build the existing sample Go checker once as a native binary and check independent sample IRs concurrently, with bounded concurrency compatible with ordinary host memory. Preserve all Query/progress verdict rules, negative controls, errors and receipt semantics. Reuse existing Go checking modules; do not change Model semantics or the checker engine. Serialize each sample's diagnostics so concurrent workers cannot scramble lines; return failure when any sample fails, and wait for owned workers before returning.

**Touches:** model/framework/check/Samples.scala, model/framework/check/Samples.test.scala, model/README.md

Run after .4 in the one-writer lane because both edit Samples.scala. Inspect the actual .4 implementation and adapt tests instead of restoring obsolete command assumptions. A Go build/run is permitted for this toolchain requirement inside the Scala batch; full consumers/regeneration stay held. No new dependencies or public API are needed.

### Quick commands
- `mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite framework.check.ToolsSuite`
- `mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/check/Samples.scala model/framework/check/Samples.test.scala`
- `make model-check-changed BASE=<committed .4 closure>` once at task end, citing the .4 record as baseline and retaining mini equivalence.

Measure matched before/after checker stage and complete pipeline wall times. Compare all six complete Go Reports byte-for-byte; preserve six raw/adapted IRs, 25 Query receipts with six negative controls and nine progress rows across three claims. Focused tests cover build-once, parallel independent processing, error propagation and complete worker join. Independent review and broad Scala gates run at the complete source-batch boundary per MILESTONES.md.

## Acceptance
- [x] A single native Go checker build replaces `go run`; sample checks run concurrently with bounded resource use.
- [x] Every sample is checked, diagnostics remain readable, any error fails the run, and all owned checks terminate before return.
- [x] Complete six-sample Query/progress Reports and raw/adapted IRs remain identical, with matched timing evidence.
- [x] Focused tests, formatting and the final scoped gate with mini equivalence pass; held stages are reported as deferred (canonical checkpoint plus approved final fixture-only guarded continuation).

## Done summary
# Native, bounded-parallel sample checker

`Samples.check` builds one native Go checker per run and checks samples through two workers in one process. Each worker retains only its sample's diagnostics after checking; all workers join before diagnostics are printed in discovery order and any failure exits nonzero. The existing Query, negative-control and progress verdict rules are unchanged. There is no Go source cache, engine change, schema change or checked-in regeneration.

The generated-checker integration regression actually compiles and runs the pool. Barriers prove overlap and the two-worker bound; load errors, empty Queries, missing receipts, wrong Query/progress answers and a deliberately delayed worker prove failure propagation, continued processing, joining and deterministic diagnostics. Native-build stubs verify one build per check and reuse of the existing Scala builds across outputs.

### Measurements and parity

The retained .4 warm pipeline is 114.899 s. The quiet final .5 warm pipeline is 126.137 s: 9.8% slower on this host, not a demonstrated pipeline speedup. The initial .5 pipeline was 134.843 s and overlapped focused compile/lint work; it is retained but not used as the quiet comparison. Model sources, schema jars and Go engine are unchanged, and all fresh IRs compare exactly.

The matched six-input serial checker baseline took 105.52 s. The selected parallel-report test took 79.034 s in its clean run; an earlier completed computation took 110.566 s but its JUnit receipt failed the default 30-second timeout. That failed log is retained. The env-selected fixture timeout was corrected, with the ordinary suite timeout unchanged. The substantial variation and slower quiet pipeline preclude a blanket speedup claim. The parallel measurement includes full Report JSON capture, unlike the serial checker-summary baseline, and excludes its native build.

Kernel HWM observations were at least 8,714,080 KiB serial and 11,791,808 KiB with two workers; these are observed high waters, not claimed final process peaks. Full runs completed on the 16,204 MiB host without swap or OOM. Do not increase concurrency on this evidence.

The real-report test instruments the actual generated checker only after its unchanged `check.Check` call, writes complete Reports and preserves the pool and validation path. All 18 comparisons passed: six raw IRs, six adapted IRs and six complete Reports. They retain 25 Query receipts, six negative controls and nine progress receipts. The quiet final pipeline's 12 fresh IRs also compare exactly, mapping its outputs to those parallel Reports.

Logs: `.flow/tmp/fn1725-before-checker.log`, `fn1725-after-pipeline.log`, `fn1725-final-warm-pipeline.log`, `fn1725-parallel-reports.log` (timeout-red) and `fn1725-parallel-reports-final.log`. Outputs: `.flow/tmp/fn1725-after` and `.flow/tmp/fn1725-final-warm`; serial artifacts: `.flow/tmp/fn1724-warm`.

### Validation and final-source mapping

Canonical `make model-check-changed BASE=1d1eec3d5b` passed on production/source checkpoint `c846c6b40e`: `.flow/tmp/checks/20261013-165141/record.md`, 988.6 s, activity_mini, 69 selected suites (47 lifter and 22 check), 311 lifter tests and 166 check tests. Syntax, scalafix, boundary, all six samples, feature compile/lift and mini step-table equivalence passed. The owner's pre-existing header-only `Lift.scala` change caused the broader lifter selection; it was not modified or attributed to this task.

After that terminal result, approved test-only commit `af0752c97e` moved the synthetic tail worker's release signal after its completion log, eliminating a fixture-ordering race. No production, report, timing, model or lifter code changed. Final affected SamplesSuite/ToolsSuite passed (22 reported tests; real-report case separately selected above), and the exact 22 check suites passed through unchanged `Tools.scalaCli(...).orFail`: 166 tests, exit 0, errors 0. Logs: `.flow/tmp/fn1725-fixture-focused.log`, `fn1725-final-guarded-check.log`, `fn1725-final-guard-wrapper.log`, `fn1725-fixture-lint.log`, `fn1725-fixture-fmt.log`. Formatting, lint and diff checks passed. The broad gate was not rerun after this fixture-only delta; its unchanged computational/lifter stages map explicitly to the final guarded check receipt.

Baseline .4 record `.flow/tmp/checks/20261013-160713/record.md` remains red from its synthetic diagnostic, despite passed computational/suite stages; its final guarded 164-check continuation is `.flow/tmp/fn1724-final-guarded-check.log` and `fn1724-guarded-check-wrapper.log`. No retrospective canonical pass is claimed.

Independent review and broad/full Scala validation are deferred to the complete Scala batch boundary per the owner. Checked-in mini IR regeneration remains held for the sweep; equivalence passed. Other held features and full Go consumer gates were not run. Owner README macro text, Lift header, plans, docs and unrelated Flow dirt were preserved; only this task's README paragraph was committed. MILESTONES.md was not edited.

Tier: session (`jev-unavailable(no_key)`); explicit project implementer `gpt-6.1-sol medium`. Review mode none; independent review deferred. Base `1d1eec3d5b21360a8dd335a6adc6cb5b2a6d7b35`; task source commits `c846c6b40ef489b2f995d35934c6447caad1c57e` and `af0752c97e89ec93e2ac2f1941421eebbc0d5756`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c846c6b40ef489b2f995d35934c6447caad1c57e, af0752c97e89ec93e2ac2f1941421eebbc0d5756
- Tests: make model-check-changed BASE=1d1eec3d5b: passed988.6s on c846c6b40e before final fixture-only delta; .flow/tmp/checks/20261013-165141/record.md; activity_mini,69suites,311lifter tests,166check tests,all6samples,mini equivalence, SAMPLE_REPORT_INPUT=.flow/tmp/fn1725-after SAMPLE_REPORT_BASE=.flow/tmp/fn1724-warm mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite framework.check.ToolsSuite:22tests passed,18byte-identical IR/complete parallel Report pairs; .flow/tmp/fn1725-parallel-reports-final.log; earlier timeout-red computation retained in fn1725-parallel-reports.log, Final fixture-only delta: mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite framework.check.ToolsSuite:22reported tests passed; .flow/tmp/fn1725-fixture-focused.log, Final exact22check suites via existing Tools.scalaCli(...).orFail inline evaluation:166tests,exit0,errors0; .flow/tmp/fn1725-final-guarded-check.log and fn1725-final-guard-wrapper.log; no broader gate rerun after test-only delta, Final make lint-model-check, scala-cli fmt --check Samples.scala Samples.test.scala and git diff --check passed; .flow/tmp/fn1725-fixture-lint.log and fn1725-fixture-fmt.log, Retained baseline .flow/tmp/checks/20261013-160713/record.md is canonical-red; .4 final guarded164check continuation .flow/tmp/fn1724-final-guarded-check.log and fn1724-guarded-check-wrapper.log, Warm pipeline114.899s retained baseline versus quiet126.137s final (9.8% slower); serial checker105.52s, parallel complete-Report capture79.034s clean/110.566s timeout-red; .flow/tmp/fn1725-before-checker.log, fn1725-final-warm-pipeline.log, fn1725-parallel-reports-final.log, All12quiet final raw/adapted IRs cmp identical to fn1724-warm; parallel Report comparison retains25Query receipts,6negative controls,9progress receipts; jq recount confirmed, Observed HWM at least serial8714080KiB / parallel11791808KiB; completed on16204MiB host,no swap/OOM; no higher concurrency admitted
- PRs: