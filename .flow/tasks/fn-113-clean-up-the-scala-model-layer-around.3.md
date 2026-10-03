---
satisfies: [R6, R7, R8, R9, R11, R25]
---
# fn-113-clean-up-the-scala-model-layer-around.3 Port the lifter's IR construction and output to ScalaPB

## Description
Port the lifter's IR construction and output to ScalaPB. Implements R6, R7, R8, R9, R11 and R25 for the lifter. The largest task of the spec: mechanical across ten files, with one proof before the checked-in JSON is rewritten.

**Size:** M
**Files:** model/lifter/*.scala (ten files), model/lifter/project.scala; model/ir/*.json and model/lifter/testdata/lifts/expected/*.json rewritten by `--update` after the R7 proof; the R7 proof under `.flow/tmp/fn113-3/` (or a throwaway Go test)
**Touches:** [model/lifter/**, model/ir/**, model/lifter/testdata/lifts/expected/**, .flow/tmp/fn113-3/**]

### Approach
- Before editing: copy `model/ir/*.json` and `model/lifter/testdata/lifts/expected/*.json` to `.flow/tmp/fn113-3/before/` with their SHA-256; record the lifter's line count (2,108 Scala lines in the ten files at planning time, 2,121 with `project.scala`) and the 108 `newBuilder` chains (Expressions 50, Declarations 22, Types 18, Claims 10, Realizations 4, Compositions 4, Context 1, Lift 1).
- `project.scala`: the ScalaPB jar from task 2, `scalapb-runtime` and the printer; drop `protobuf-java` and `protobuf-java-util` as direct dependencies (R6; a transitive one through the runtime is allowed). Replace every builder chain with the generated case class (`ir.X(a = a)`): oneofs are the generated sealed member types, message fields `Option`, repeated fields `Seq`. Keep helpers such as `expr(at)(...)` where they shorten code. Count the matches over oneofs the compiler now checks for exhaustiveness (R11).
- Realization emitter (`Realizations.scala`): stays generic and descriptor-driven over `scalapb.descriptors` (field by name, containing oneof, enum value by name, the companion of a message field, repeated/optional), building through the companion's reflective reader (for example `PMessage`/`PValue` and `messageReads`) rather than a hand-written mapping per message. Refusal texts and positions unchanged (R9). Defaults stay unset: do not enable default-value printing (R7 proves it).
- Output: the ScalaPB printer in place of `JsonFormat.printer()` in `Lift.scala`; same file naming and trailing newline.
- R7 proof before any `--update`: lift all six IR files and six fixture files to a scratch directory (the gate's check mode leaves them under `model/gen/history`, or run the lifter directly as `Fixtures.test.scala` does), then a Go check (a throwaway `_test.go` or a `go run` file under `.flow/tmp/fn113-3/`) that `protojson.Unmarshal`s each `before/` file and the new output and asserts `proto.Equal`. Only when all twelve compare equal, run `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` to rewrite the checked-in JSON. Any unequal file stops the task (R7's error clause). Keep the proof's command and result in the summary.
- R8: `expected/rejects.txt`, the refusal positions (`Evidence.scala.fixture:29:16`, `Crossed.scala.fixture:35:14`, `45:28`, `Unsupported.scala.fixture:18`) and `Fixtures.test.scala` do not change; the refusal format `lift: <file>:<line>: <message>` stays.
- R13: the goldens compare IR inputs with `proto.Equal` after `protojson.Unmarshal` (`golden.go:163`), so a decode-equal rewrite passes them strictly; no projection is needed in this task.
- R25: record ScalaPB's wiring lines against the removed hand-written lines (the spec expects 100 to 150 saved) and whether any JSON library beyond the printer was weighed.
- Scalafix runs over `model/lifter` as a directory; generated code is in the jar (task 2), so no exclusion is needed; if a finding appears in lifter code, fix the code.

### Investigation targets
**Required**:
- `model/lifter/project.scala:6-8`
- `model/lifter/Lift.scala:20,58-62,128-131` (imports, `Model` assembly, printer)
- `model/lifter/Expressions.scala:28-41,88-91,185-190,222-244,345-347,376-384` (expression, function, match, option, lambda and pattern builders)
- `model/lifter/Realizations.scala:1-6,150-269` (descriptor imports; `valueOf`, `fieldOf`, `declaration`, `realizationOf`)
- `model/lifter/Types.scala`, `model/lifter/Declarations.scala`, `model/lifter/Claims.scala:56-67,233-241`, `model/lifter/Compositions.scala:34-70`, `model/lifter/Context.scala:98`
- `model/lifter/test/Fixtures.test.scala:202-275` (how fixtures are lifted and compared)
- `tools/umpire/model/load.go` (`protojson.Unmarshal`: the Go side is unchanged)
- `tools/umpire/internal/golden/golden.go:163-175` (`Match`)
- `.flow/tmp/fn113-1-summary.md`, `.flow/tmp/fn113-2-summary.md`

### Quick commands
mise exec -- scala-cli compile model/lifter; mise exec -- scala-cli test model/lifter (needs the jars a gate run packages); CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (fails at the IR compare until --update, which is expected); the R7 proof; CC=/usr/bin/clang mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks; CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model ./tools/umpire/lower; make umpire-check-cases umpire-check-fixtures canary-check-case; mise exec -- make lint-model

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] The lifter imports nothing from `com.google.protobuf`, `project.scala` names neither `protobuf-java` nor `protobuf-java-util`, and the ten files build the IR through ScalaPB case classes and print ProtoJSON through the ScalaPB printer.
- [ ] A Go check with `protojson` and `proto.Equal` shows every checked-in IR file and every expected fixture output equal before and after the port; its command and result are in the summary; the checked-in JSON was rewritten with `--update` only after that.
- [ ] `expected/rejects.txt`, the refusal positions and the lifter's refusal and must-not-compile tests are unchanged and pass; the realization emitter is still descriptor-driven and refuses an unknown constructor or parameter at its source line.
- [ ] The summary states the lifter's line count before and after, the count of oneof matches the compiler now checks, and the R25 weighing; the goldens, `umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` and `lint-model` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
