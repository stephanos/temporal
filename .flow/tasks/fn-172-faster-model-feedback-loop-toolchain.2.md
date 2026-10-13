---
satisfies: [R2]
---
# fn-172-faster-model-feedback-loop-toolchain.2 Compile each set of inputs once: take the classpath from the package step instead of compiling again (Samples.scala, Makefile model jar)

## Description
Remove the redundant source compilation from framework sample checking and the Make model jar recipe. The pinned Scala CLI supports `compile --print-class-path --compilation-output <path.jar>`: one build writes the library jar and its authoritative transitive classpath. Capture compiler diagnostics separately from classpath output and reject compiler errors before publishing outputs. Keep normal compilation for Temporal Model sources.

**Touches:** Makefile, model/framework/check/Samples.scala, model/framework/check/Samples.test.scala, model/framework/check/Tools.scala, model/framework/check/Tools.test.scala, model/temporal/check/MakeFeatures.test.scala, model/README.md

### Investigation targets
Required: `model/framework/check/{Samples,Tools}.scala`, their co-located tests, `model/temporal/check/MakeFeatures.test.scala`, Makefile model jar recipe, pinned Scala CLI compile help.

### Quick commands
`mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- framework.check.SamplesSuite framework.check.ToolsSuite temporal.check.MakeFeaturesSuite`

`mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/check/Samples.scala model/framework/check/Samples.test.scala model/framework/check/Tools.scala model/framework/check/Tools.test.scala model/temporal/check/MakeFeatures.test.scala`

Run `make model-check-changed BASE=<task-base>` once after implementation; batch-end full gates remain conductor-owned. Measure the jar/classpath stage before and after serially. Compare all six raw sample IRs and complete Go Reports against retained `.flow/tmp/fn172-r1/resume-after` artifacts; preserve progress roots. Verify a real Make jar/classpath with transitive Scala/API dependencies and refusal on compiler failure.

### Review scope and evidence
The owner committed the tested R2 source/test/docs subset in mixed WIP commit `173a04fc798c8ad92ccd9dae95c7b223efeed674`. Preserve that commit. Review R2 only against base `d71fb7f5a41ebf3d7244a70947e2f04f313561d2`, restricted to the seven implementation paths in **Touches:** and this task's own planning/receipt files. Other Flow specs/tasks, fn173 planning, MILESTONES.md and model/design are owner work outside this task and outside its review. The R2 subset matches the tested edits; Model inputs and the Go checker remain unchanged. Read `.flow/tmp/fn172-r2/task.diff` and `.flow/tmp/fn172-r2/verification.md` for the scoped code diff and actual evidence, including the initial scoped gate timeout and separate remaining-module continuation. Review the scoped implementation without treating unrelated owner planning as task changes.
## Acceptance
- [x] R2: Samples.check and the Make model jar recipe compile each input set once, emitting the library jar and authoritative classpath together, with no second source compilation or guessed/stale metadata.
- [x] The classpath retains transitive dependencies and paths containing spaces; lifter roots, IR bytes, identities and complete sample Reports stay unchanged.
- [x] Samples retains compiler-diagnostic rejection even when the warm compiler exits zero. Make Model compilation retains --server=false and leaves the prior jar/classpath intact on compiler failure.
- [x] Focused regressions are observed red then green; real sample lifting/checking and Make jar use pass. Record paired stage wall times and six-sample byte parity.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
