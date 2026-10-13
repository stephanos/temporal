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
- [ ] A single native Go checker build replaces `go run`; sample checks run concurrently with bounded resource use.
- [ ] Every sample is checked, diagnostics remain readable, any error fails the run, and all owned checks terminate before return.
- [ ] Complete six-sample Query/progress Reports and raw/adapted IRs remain identical, with matched timing evidence.
- [ ] Focused tests, formatting and the final scoped gate with mini equivalence pass; held stages are reported as deferred.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
