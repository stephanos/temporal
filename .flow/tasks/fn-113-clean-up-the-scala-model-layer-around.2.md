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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
