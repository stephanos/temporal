---
satisfies: [R3]
---
# fn-172-faster-model-feedback-loop-toolchain.3 Lift all framework samples in one lifter run, keeping the progress-claim roots

## Description
Run the framework samples' exact per-file lifts in one lifter JVM through an internal generated driver that calls the existing LiftRun.run. Keep each sample's independent namespace and inspection, source discovery, Query and progress roots, output names, fail-fast behavior and checker inputs. Existing lift --ir groups feature packages and checks IDs across files, while these independent samples intentionally repeat names in package samples; it cannot batch them without changing their contract.

Verification evidence and measured artifacts are recorded in `.flow/tmp/fn172-r3/verification.md` and its linked logs and before/after directories.

**Touches:** model/framework/check/Samples.scala, model/framework/check/Samples.test.scala

### Investigation targets
Required: model/framework/check/Samples.scala, model/framework/check/Samples.test.scala, model/framework/export/Lift.scala, model/framework/export/Lifting.scala, model/framework/export/InProcess.test.scala, model/framework/check/Tools.scala, model/README.md, model/framework/SEMANTICS.md, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_VISION.md, .plans/UMPIRE_MODULES.md, .flow/tmp/handoffs/fn-172.1-summary.md, .flow/tmp/handoffs/fn-172.1-evidence.json, .flow/tmp/handoffs/fn-172.2-summary.md, .flow/tmp/handoffs/fn-172.2-evidence.json, .flow/tmp/handoffs/fn-172.6-summary.md, .flow/tmp/handoffs/fn-172.6-evidence.json.

### Quick commands
- `mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite framework.check.ToolsSuite`
- `mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/check/Samples.scala model/framework/check/Samples.test.scala`
- Post-change project gate: `make model-check-changed BASE=<persisted task base>` once, with adequate bounds for selected stages. Full spec-end gate remains conductor-owned.

## Acceptance
- [ ] One Scala CLI lifter invocation runs every discovered sample in one JVM using existing LiftRun.run independently. Compile-once behavior stays intact.
- [ ] Preserve each sample's exact Query and direct-member progress roots, independent namespace/identity validation, filename and raw IR bytes. Preserve all six complete Go Report bytes, 25 Query receipts with six negative controls and nine progress rows across three named claims.
- [ ] Preserve fail-fast lift behavior, successful prior-file publication, missing-output rejection, compiler diagnostic rejection and the existing Go checker behavior. The driver passes paths as process arguments, including spaces and source-significant characters.
- [ ] Measure fresh same-input before/after lifter-stage and complete pipeline wall times in the same environment; report retained repeated TASTy inspection cost. No lifter packaging/cache, Go prebuild/parallelism, DSL/Model/schema changes or artifact regeneration.

## Done summary
All six framework samples now lift in one JVM, with independent roots, namespaces, validation and fail-fast publication preserved. The lifter stage fell from 30.153 s to 16.738 s (44.5%). Complete pipeline times were 321.670 s and 111.781 s; variable unchanged Go work prevents attributing the whole reduction to batching.

Verification: the scoped gate record `.flow/tmp/checks/20261013-131559/record.md` passed 162 tests, six samples and mini step-table equivalence in 368.1 s. It started before the final missing-output error correction; the retained targeted red/green proof and fresh continuation verification cover that final branch. Fresh continuation ran SamplesSuite and ToolsSuite: 18 tests passed; formatting passed. All six raw IRs, adapted IRs and complete Go reports compare byte-for-byte in `.flow/tmp/fn172-r3/{before,after} with spaces`. Measurements and snapshot attribution are in `.flow/tmp/fn172-r3/verification.md`.

Source commits: ea07041552fa7c58376fea4aa104328343b72dc5 and 334638ca0dd4f10f0c2b4183300c3284fa898055. The independent review receipt `/tmp/impl-review-receipt-8f37faba39e2-fn-172-faster-model-feedback-loop-toolchain.3.json` binds the final source head, reports SHIP on correctness/contracts/integration and has no findings (gpt-6.1-sol high, same model family).

stage: impl-review - ran [2026-10-13T20:23:49Z..2026-10-13T20:26:18Z] (model: gpt-6.1-sol)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: batch-review - skipped(policy: owner requests review at the complete Scala batch boundary)

Held regeneration, broad Model/lifter lint, full Go consumer and real-feature gates remain deferred. No passing credit is claimed for them. Existing owner planning and design changes remain uncommitted.
## Evidence
- Commits: ea07041552fa7c58376fea4aa104328343b72dc5, 334638ca0dd4f10f0c2b4183300c3284fa898055
- Tests: make model-check-changed BASE=fbe20714a4869fd762bfea02c94c75dc3ccfb3e0 (snapshot plus final focused delta; record .flow/tmp/checks/20261013-131559/record.md), mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite framework.check.ToolsSuite (fresh continuation: 18 passed), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/check/Samples.scala model/framework/check/Samples.test.scala, cmp all six raw IRs, adapted IRs and complete reports: identical
- PRs:
