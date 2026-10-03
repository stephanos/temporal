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
# fn-113.3 handover: the lifter builds and prints the IR through ScalaPB

Nothing is staged or committed. Changed files: the ten lifter files, `model/lifter/project.scala`, and the twelve JSON files the gate's `--update` rewrote (`model/ir/*.json`, `model/lifter/testdata/lifts/expected/*.json`). `expected/rejects.txt`, `Fixtures.test.scala` and the fixture sources did not change.

### What changed

- `project.scala`: `protobuf-java`, `protobuf-java-util` and `../gen/ir-proto.jar` are gone. It now uses `com.thesamet.scalapb::scalapb-runtime:0.11.20`, `com.thesamet.scalapb::scalapb-json4s:0.12.2` (ScalaPB's printer) and `../gen/ir-scalapb.jar`. protobuf-java still arrives through the runtime, which R6 allows.
- The ten files build the IR with the generated case classes (`ir.X(a = a)`): oneofs are the generated sealed members, message fields are `Option`, and repeated fields are `Seq`. HEAD had 113 `newBuilder` occurrences (the plan counted 108 chains). There are now 0, and `grep com.google.protobuf model/lifter/*.scala` finds nothing.
- `Lift.scala`: `Model` is put together in one constructor call with a `sorted` helper. Output goes through `scalapb.json4s.Printer().toJson`, written by Jackson with a 4-line `Pretty` printer that keeps the `"field": value` layout. File naming and the trailing newline are unchanged. Default-value printing stays off.
- `Realizations.scala`: the emitter is still generic and driven by descriptors over `scalapb.descriptors`. It looks fields up by name, finds a field's containing oneof, looks enum values up by name, and reads the message descriptor of a message field (`ScalaType.Message(d)`) and whether a field is repeated or optional. A small `Message` holder collects `PValue`s per `FieldDescriptor`; setting one oneof member clears the others, as the Java builder did. `emit` turns the result into the generated class with `companion.messageReads.read(PMessage)`. There is no hand-written mapping per message. The refusal texts (`$n is no $p of <msg> in the IR`, `<msg> has no $p in the IR`, `... takes one value`, `... is no <Enum> of the IR`) and the terms they are reported at are unchanged.
- Comments are kept. The only comment-line changes are two trailing comments whose code was ported, plus one new comment.

### Line counts (R25)

| file | before (HEAD) | after |
|---|---|---|
| Claims | 314 | 302 |
| Compositions | 86 | 71 |
| Constants | 53 | 53 |
| Context | 105 | 101 |
| Declarations | 475 | 423 |
| Expressions | 450 | 337 |
| Lift | 131 | 143 |
| Lifting | 39 | 39 |
| Realizations | 269 | 276 |
| Types | 185 | 166 |
| **ten files** | **2,107** | **1,911** (-196) |
| project.scala | 16 | 16 |
| **total** | **2,123** | **1,927** (-196) |

`linecount-before.txt` holds the "before" counts.

R25 weighing:
- ScalaPB wiring added: 2 dependency lines and 1 jar line, which replace 3 lines. Realizations gains the `Message` holder and `emit` (about 19 lines). Lift gains the printer call, the `Pretty` class and 4 imports (about 12 lines). That is about 31 lines of wiring.
- Removed: about 227 lines of hand-written builder chains, net -196. That is more than the 100 to 150 the spec expected.
- JSON libraries: `scalapb-json4s` is ScalaPB's own ProtoJSON printer, and json4s/Jackson come with it. No other JSON library was weighed.
- The `Pretty` printer (4 lines) only keeps the `": "` layout of the old JsonFormat files so the diffs stay readable. R7 equality does not depend on it.

### R11: oneof matches the compiler now checks

1. `Types.optionType` matches `value.ref` over `ir.TypeRef.Ref`, and names every member (`Named`, `Bool`, `IntRange`, `Int | List | Channel | Empty`). Before the port this was a `getRefCase` match with a `_` fallback. It is the only match over a oneof in the lifter. The other oneof uses build values and do not match on them.

### R7 proof

- Command: `GOFLAGS=-p=1 mise exec -- go run .flow/tmp/fn113-3/r7.go`, run from the repo root.
- Method: `protojson.Unmarshal` plus `proto.Equal` on `before/` (the HEAD copies, SHA-256 in `before/SHA256SUMS`) against `after/` (output of the ported lifter).
- Result: `12 files, 0 unequal` (`.flow/tmp/fn113-3/r7.log`; re-run in `r7-rerun.log`).
- The proof ran before `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` (`gate-update.log`).
- I also checked that each `before/` file is byte-identical to its HEAD blob, and that each checked-in file now is byte-identical to its `after/` file.

### Checks (logs under .flow/tmp/fn113-3/)

| command | result | log |
|---|---|---|
| `mise exec -- scala-cli test model/lifter` | pass, 14/14 | lifter-test.log |
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -timeout 30m -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model` | ok (50 s) | goldens-model.log |
| same for `./tools/umpire/lower` | ok (91 s) | goldens-lower.log |
| `make umpire-check-cases` | exit 0 | umpire-check-cases.log |
| `make umpire-check-fixtures` | exit 0 | umpire-check-fixtures.log |
| `make canary-check-case` | exit 0 | canary-check-case.log |
| `make lint-model` | exit 0 | lint-model.log |
| `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` (no `--update`) | `== ok` | gate-check.log |

- The lifter suite includes the refusal and must-not-compile tests (`Evidence.scala.fixture:29:16`, `Crossed.scala.fixture:35:14`/`45:28`, `Unsupported.scala:18`, rejects.txt).
- In the lint log, scalafix prints `java.lang.NoSuchFieldException: path` traces. They come from scalafix's reflection under JDK 27, appear for every project as they did in task 2, and do not fail the lint. Formatting and scalafix report no finding in lifter code.
- Not exercised: no fixture triggers the realization emitter's unknown-constructor or unknown-parameter refusal, before or after the port. Well-typed framework code cannot reach it. The diff shows those refusals keep their texts and positions.

### Notes for the conductor (shared documents, not edited)

- The lifter no longer reads `model/gen/ir-proto.jar`. The gate (`model/gate/Gate.scala`, its test) and the Makefile (`MODEL_PROTO_JARS`, the `ir-proto.jar` rule) still produce it. Whether to drop it belongs to the gate's owner or a later task; it is outside this task's Touches.
- `MILESTONES.md:90` (Part B, ScalaPB) can record that the lifter is ported: 1,927 lines, down from 2,123.
- `.plans/UMPIRE_MODULES.md` and the migration manifest: no change needed, as far as I found.
- `model/lifter/Expressions.scala:102` still has a Stainless reference in the `stripContracts` doc comment. That is Part D / R19 work and was left alone.

### Review (claude-opus-5-5, fresh context)

Verdict SHIP. The conductor applied:
- `Declarations.scala`: a machine's family is read before its name again, so a refusal of both is reported at the family's line, as before (R8). The previous code read the name first.
- `Realizations.scala`: the unreachable fallback refusal spells the field kind in protobuf-java's JavaType names (`FLOAT`, `DOUBLE`, `BYTE_STRING`) through `javaKind`, so its text is unchanged (R9).
- Named arguments where two neighbouring fields are strings: `ir.Construct`, `ir.CasePattern`, `ir.StepBinding`, `ir.Hole`.

After the fixes:
- `scala-cli test model/lifter`: 14/14 pass (`.flow/tmp/fn113-3/lifter-test-review-fixes.log`).
- `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`: ok without `--update`, so the output is unchanged (`gate-check-review-fixes.log`).
- `make lint-model`: ok (`lint-model-review-fixes.log`).

Corrections to this handover:
- The diff adds four comments, not one: `Pretty`'s doc comment in `Lift.scala`, and in `Realizations.scala` the `Message` doc comment, the inline comment on `set` and `emit`'s doc comment.
- `Pretty` keeps the `"field": value` spacing. It does not keep the old files' text, which this port rewrote in full: arrays, field order and escapes.
- `Pretty` reaches Jackson and json4s through `scalapb-json4s`, not through dependencies of its own.

Conductor note: the ScalaPB printer writes fields in declaration order rather than field-number order (e.g. `commitment = 8` after the oneof `run_event = 12`), and no longer escapes `'`, `<` and `>`. Every Go reader decodes with protojson and nothing hashes the bytes, so only future diffs of `model/ir` show the new order.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4e06600ba9fbd9a5aeac7e1b26960014c35fbb60
- Tests: GOFLAGS=-p=1 mise exec -- go run .flow/tmp/fn113-3/r7.go -> 12 files, 0 unequal (protojson + proto.Equal, before vs after), mise exec -- scala-cli test model/lifter -> pass (14/14), GOFLAGS=-p=1 mise exec -- go test -count=1 -timeout 30m -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model -> ok, GOFLAGS=-p=1 mise exec -- go test -count=1 -timeout 30m -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/lower -> ok, make umpire-check-cases -> pass, make umpire-check-fixtures -> pass, make canary-check-case -> pass, make lint-model -> pass, make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks -> ok (no --update), mise exec -- scala-cli test model/lifter (after review fixes) -> 14/14 pass, make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (after review fixes) -> == ok, no --update, make lint-model (after review fixes) -> ok, independent review (claude-opus-5-5, fresh context): SHIP; should-fix (refusal order) and nits applied
- PRs: