---
satisfies: [R3]
---
# fn-172-faster-model-feedback-loop-toolchain.3 Lift all framework samples in one lifter run, keeping the progress-claim roots

## Description
Run the framework samples' exact per-file lifts in one lifter JVM through an internal generated driver that calls the existing LiftRun.run. Keep each sample's independent namespace and inspection, source discovery, Query and progress roots, output names, fail-fast behavior and checker inputs. Existing lift --ir groups feature packages and checks IDs across files, while these independent samples intentionally repeat names in package samples; it cannot batch them without changing their contract.

Verification evidence and measured artifacts are recorded in `.flow/tmp/fn172-r3/verification.md` and its linked logs and before/after directories.

**Touches:** model/framework/check/Samples.scala, model/framework/check/Samples.test.scala

### Investigation targets
Required: model/framework/check/Samples.scala, model/framework/check/Samples.test.scala, model/framework/export/Lift.scala, model/framework/export/Lifting.scala, model/framework/export/InProcess.test.scala, model/framework/check/Tools.scala, model/README.md, model/SEMANTICS.md, .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_VISION.md, .plans/UMPIRE_MODULES.md, .flow/tmp/handoffs/fn-172.1-summary.md, .flow/tmp/handoffs/fn-172.1-evidence.json, .flow/tmp/handoffs/fn-172.2-summary.md, .flow/tmp/handoffs/fn-172.2-evidence.json, .flow/tmp/handoffs/fn-172.6-summary.md, .flow/tmp/handoffs/fn-172.6-evidence.json.

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
