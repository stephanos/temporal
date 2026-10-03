---
satisfies: [R5, R10]
---
# fn-113-clean-up-the-scala-model-layer-around.2 Generate the IR's ScalaPB classes in the gate, beside the Java ones

## Description
Generate the IR's ScalaPB classes in the gate, beside the Java ones. Implements R5 and the generated-source clause of R10, using task 1's recorded versions and mechanism.

**Size:** M
**Files:** model/gate/Gate.scala, model/gate/test/Gate.test.scala, model/gate/project.scala (only if the gate hosts the generator), Makefile (the `MODEL_PROTO_JARS` rule only if a second jar is added)
**Touches:** [model/gate/**, Makefile (MODEL_PROTO_JARS rule), model/gen/** (ignored build output)]

### Approach
- Extend `Gate.generateIr` with ScalaPB generation using the mechanism task 1 recommended: the pinned protoc from the PATH (mise supplies it; the gate does not nest `mise exec`) with the ScalaPB plugin, the generated Scala compiled against `scalapb-runtime` and packaged with `scala-cli --power package --library` into a jar under `model/gen`. Keep the Java generation and `ir-proto.jar` untouched in this task so the lifter builds throughout; task 4 removes them. Give the new jar its own name (for example `ir-scalapb.jar`) and fold the generator and runtime versions into the stamp with the schema's SHA-256, so a version bump regenerates and a second `--if-stale` run does nothing.
- No manual install step and no checked-in shell script in `model/` (fn-115.10 removed them all). If protoc needs an executable plugin path, the gate may write a launcher under `model/gen` (ignored output) or run the generator as a scala-cli program; record the choice. If the gate itself needs a dependency, add it to `model/gate/project.scala` only and record it; never to `model/project.scala`.
- Failure modes through the existing seams: a missing plugin fails naming it (`Tools.find` pattern); a stale or missing jar fails with the jar and the schema named (`GateError`). Gate suite on stub tools: "packaged when the schema changed" extended to the second jar, "regenerated when the generator version changed", "a missing plugin is named".
- R10's generated-source clause: generated Scala lives only in the jar, never under a root `make lint-model` visits; confirm scalafix sees no generated file. A scalafix finding in generated code is fixed by excluding the sources, never by a rule suppression for the lifter.
- Keep the gate's comments true (its `generateIr` doc names the Java classes today).

### Investigation targets
**Required**:
- `model/gate/Gate.scala:36,58-108` (`irJar`, `generateIr`, the protobuf-java version comment)
- `model/gate/Gate.scala:131-139,274` (the generate step in `run`; the `--generate-ir` flag)
- `model/gate/test/Gate.test.scala` (the "packaged when the schema changed" test and the stub tools)
- `model/gate/Tools.scala` (`find`, `run`, `scalaCli`, `Ran.orFail`)
- `Makefile:664-682` (`MODEL_PROTO_JARS`, the jar rule, `lint-model` prerequisite)
- `model/lifter/project.scala:8` (`//> using jar ../gen/ir-proto.jar`, unchanged here)
- `.flow/tmp/fn113-1-summary.md` (versions and mechanism)

### Quick commands
mise exec -- scala-cli test model/gate; mise exec -- scala-cli run --suppress-outdated-dependency-warning model/gate -- --generate-ir; the same with --generate-ir --if-stale (second run reports nothing to do); CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; mise exec -- make lint-model

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `--generate-ir` produces the ScalaPB classes from the unchanged `ir.proto` with the pinned protoc into a jar under `model/gen`, the Java jar still builds, regeneration happens only when the schema or the generator version changed, and two consecutive `--if-stale` runs do no work the second time.
- [ ] A missing plugin, and a stale or missing jar, fail the gate with the plugin or the jar and the schema named; the gate suite covers both on stub tools and passes.
- [ ] `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` and `make lint-model` pass; `make lint-model` lints no generated source; no file under `model/ir`, `model/cases` or `model/lifter/testdata` changes; `model/project.scala` gains no dependency.
- [ ] The summary records the plugin delivery choice, any gate dependency added, and the stamp format.


## Done summary
The gate now packages the IR's ScalaPB classes beside the Java ones. `Gate.generateIr` runs the pinned protoc twice from the unchanged `ir.proto`: `--java_out` into `model/gen/ir-proto.jar` as before, and ScalaPB 0.11.20 (`flat_package,scala3_sources`) into `model/gen/ir-scalapb.jar`, compiled against `com.thesamet.scalapb::scalapb-runtime:0.11.20`. The real jar is 3,533,296 bytes, the size task 1 measured.

Plugin delivery: the gate packages ScalaPB's generator (`com.thesamet.scalapb::compilerplugin:0.11.20`, main class `scalapb.ScalaPbCodeGenerator`) as a scala-cli launcher at `model/gen/protoc-gen-scala` (ignored output) and passes it to protoc with `--plugin=protoc-gen-scala=...`. It is packaged from an empty scratch directory under `model/gen/history`, so scala-cli's build state stays out of the repository root. No tool was added to `mise.toml`, no dependency to `model/gate/project.scala` or `model/project.scala`, and no script was checked in.

Stamp format: `model/gen/ir.stamp` is one line, `<SHA-256 of ir.proto> com.google.protobuf:protobuf-java:4.29.5 scalapb:0.11.20`. A schema or generator-version change, or either missing jar, regenerates both jars. After a cold run (278 s, mostly dependency fetches), a following `--generate-ir --if-stale` took 0.8 s and did no work.

Failure modes: a launcher scala-cli did not package fails with `the ScalaPB plugin model/gen/protoc-gen-scala is missing: scala-cli did not package it`, before protoc runs. A jar that is missing after packaging fails with `<jar> is missing: the IR schema <schema> was not packaged`. A jar older than the start of packaging fails with `<jar> is stale: the IR schema <schema> was not packaged into it`. A failed generation leaves no stamp. The gate suite covers each case on stub tools, plus regeneration after a generator-version change; it passes 31 tests.

Makefile: `MODEL_PROTO_JARS` names both jars. The ScalaPB jar is chained off the Java jar (`test -f $@ || gate --generate-ir --if-stale`) rather than a grouped `&:` target, which macOS's make 3.81 lacks, so one schema change runs the gate once.

R10: generated Scala exists only in the jar and under `model/gen/history`; no root `make lint-model` visits holds it, and the lint passes. The `java.lang.NoSuchFieldException: path` traces in the lint log are scalafix's reflection under JDK 27, printed for every project, and do not fail the lint.

R25: no library was weighed. The change is gate plumbing around protoc and scala-cli, which already run every tool.

Gates: `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` and `make lint-model` pass, and no file under `model/ir`, `model/cases` or `model/lifter/testdata` changed. On this sandbox (15 GB RAM), the full Go suite under the gate is OOM-killed (`export.test`, `model.test` at about 5.5 GB RSS each) or times out when packages run in parallel. Each package passes run alone (`tools/umpire/model`: 135 s). This task changes no Go code.

Conductor notes: none of the shared documents need a change for this task.

Review: independent review (claude-opus-5-5, fresh context) returned SHIP. Applied: the ScalaPB jar and plugin are compiled with the pinned Scala 3.9.0, which the stamp now carries (`... scalapb:0.11.20 scala:3.9.0`); a launcher left from an earlier run is refused as stale before protoc runs; a comment notes that the ScalaPB classes take the Java classes' names, so the lifter reads one jar (task 3 swaps the jar, never adds the second); the second Makefile rule prints its banner. Left out on purpose: the protoc version is not in the stamp, since `mise.toml` pins it and the Java jar never stamped it; a protoc bump is regenerated by `--generate-ir` without `--if-stale`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 08b25b28f626fc5bc5df34bd46298c7f0103a4f3
- Tests: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/gate -> 31 passed after review fixes (.flow/tmp/fn113-2/gate-suite.log), mise exec -- scala-cli run --suppress-outdated-dependency-warning model/gate -- --generate-ir -> rc=0, generated model/gen/ir-proto.jar and model/gen/ir-scalapb.jar (3533296 B), 277.85 s cold (.flow/tmp/fn113-2/generate.log), mise exec -- scala-cli run --suppress-outdated-dependency-warning model/gate -- --generate-ir --if-stale -> rc=0, no work, 0.78 s (.flow/tmp/fn113-2/generate-ifstale.log), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks -> == ok, 73.6 s (.flow/tmp/fn113-2/check-model-skip-go.log), make lint-model -> rc=0, formatting and scalafix clean (.flow/tmp/fn113-2/lint-model.log), git status -> no change under model/ir, model/cases, model/lifter/testdata, independent review (claude-opus-5-5, fresh context): SHIP; should-fix items applied (Scala version pinned and stamped, stale plugin refused, shared class names noted) and nits 6 and 8; protoc version deliberately left out of the stamp
- PRs: