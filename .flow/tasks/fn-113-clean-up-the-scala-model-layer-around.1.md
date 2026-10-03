---
satisfies: [R4]
---
# fn-113-clean-up-the-scala-model-layer-around.1 Verify which ScalaPB release fits Scala 3.9.0 and the pinned protoc

## Description
Verify which ScalaPB release fits Scala 3.9.0 and the pinned protoc. Implements R4; the outcome decides whether Part B (tasks 2 to 4) proceeds.

**Size:** S
**Files:** none tracked; scratch builds and the record under `.flow/tmp/fn113-1/`
**Touches:** [.flow/tmp/fn113-1/**]

### Approach
- Candidates to pin: the ScalaPB compiler plugin and `scalapb-runtime` published for Scala 3 (`com.thesamet.scalapb`), a ProtoJSON printer for ScalaPB messages (`scalapb-json4s` or another maintained one; it is a lifter-only dependency, so R3 is not touched), and the way the gate gets the protoc plugin with no manual install step: (a) the `scalapbc` JVM launcher run through `scala-cli run --dep`, only if it can be made to run the pinned `protoc` 29.5 from mise rather than a bundled one; (b) `protoc --plugin=protoc-gen-scala=<path>` with the published `protoc-gen-scala` artifact resolved through scala-cli's dependency resolution; (c) `protoc --descriptor_set_out` plus a small generator program that depends on the compiler plugin's code generator, run by scala-cli. State for each what it needs and whether it adds a tool to `mise.toml` (that would be an owner decision).
- Prove the fit by generating Scala from the unchanged `proto/internal/temporal/server/api/umpire/v1/ir.proto` with the pinned protoc and compiling the output with Scala 3.9.0 under scala-cli 1.17.1 and JDK 27 (the lifter's options: `-deprecation -feature -unchecked -Wunused:imports`), then packaging it with `scala-cli --power package --library` as the gate does today. Record wall time, class count and jar size.
- Check the printer against Go: print one message that has unset message fields, a zero int, a oneof, an enum and an int64 through the candidate printer and `protojson.Unmarshal` it in a scratch Go file (`.flow/tmp/fn113-1/`, inside the module so `go run` works); confirm default-valued fields are omitted (the realization declarations rely on "a default is the empty value, which the IR leaves unset"). Confirm `scalapb.descriptors` offers what `Realizations.scala` reads through `FieldDescriptor` and `Message.Builder`: field by name, the containing oneof, enum values by name, the companion of a message field, repeated and optional handling.
- If no released combination works, say which step failed and why: Part B stops at this task (R4's error clause), the conductor records it in the spec, and Parts C and D stand.
- Nothing tracked changes; the gate is not run.

### Investigation targets
**Required**:
- `mise.toml:1-5` (protoc 29.5, scala-cli 1.17.1, temurin-27)
- `model/lifter/project.scala:1-13` (current deps and the jar)
- `model/gate/Gate.scala:58-108` (`generateIr`: protoc call, packaging, stamp)
- `model/gate/project.scala:1-8` (the gate is standard library only, no -Werror)
- `model/lifter/Realizations.scala:150-269` (the descriptor-driven emitter)
- `model/lifter/Lift.scala:20,128-131` (`JsonFormat.printer()`)
- `proto/internal/temporal/server/api/umpire/v1/ir.proto` (options such as `java_package`, `java_multiple_files`; proto3)

### Quick commands
mise exec -- protoc --version; mise exec -- scala-cli version; in `.flow/tmp/fn113-1/`: mise exec -- protoc --proto_path=proto/internal --plugin=... --scala_out=... (or the chosen mechanism); mise exec -- scala-cli compile <scratch dir> --scala 3.9.0 --dep com.thesamet.scalapb::scalapb-runtime:<version>; CC=/usr/bin/clang mise exec -- go run .flow/tmp/fn113-1/decode.go <sample.json>

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] A table in `.flow/tmp/fn113-1-summary.md` names the ScalaPB compiler plugin, runtime and ProtoJSON printer versions and the plugin delivery mechanism that generate the IR classes from the unchanged schema with protoc 29.5 and compile under Scala 3.9.0 with scala-cli 1.17.1 and JDK 27, with the commands, wall times and logs under `.flow/tmp/fn113-1/`.
- [ ] The printer's handling of default-valued fields, int64, enums and oneofs is checked against `protojson.Unmarshal` on a sample message, and the `scalapb.descriptors` API is shown to cover what `Realizations.scala` reads through `FieldDescriptor` and `Message.Builder`.
- [ ] Each delivery mechanism is rated on "no manual install step" and "pinned protoc runs", and one is recommended for task 2; any need for a new tool in `mise.toml` is flagged as an owner decision.
- [ ] If no released combination fits, the summary states which step failed and why, and that Part B stops here while Parts C and D stand.
- [ ] No tracked file changed; the model gate was not run.


## Done summary
A released ScalaPB combination fits: compiler plugin and runtime `com.thesamet.scalapb` 0.11.20 (Scala 3 build, `flat_package,scala3_sources`), printer `scalapb-json4s` 0.12.2, the pinned protoc 29.5 from `mise.toml`. It generates the IR classes from the unchanged `ir.proto` and compiles warning-free under Scala 3.9.0, scala-cli 1.17.1 and JDK 27 with the lifter's options (659 classes, 3.5 MB jar; 11.9 s warm end to end versus about 3.3 s today). Recommended delivery for task 2: the gate packages a `protoc-gen-scala` launcher with scala-cli and passes it to the pinned protoc; no new tool in `mise.toml`. Part B proceeds.

The printer's output decodes with Go's `protojson.Unmarshal` to `proto.Equal` messages for defaults, int64, enums and oneofs (a oneof member at its default is emitted, as Go and protobuf-java do), and all 12 checked-in IR and expected files round-trip `proto.Equal`; three of them also change field order, which the spec allows. The `scalapb.descriptors` API covers every lookup `Realizations.scala` makes. No tracked file changed.

Independent review (Claude Fable, fresh context) returned SHIP in round 1; its notes for tasks 2 and 3 are in the handover. Handover: .flow/tmp/fn113-1-summary.md; evidence: .flow/tmp/fn113-1-evidence.json; review: .flow/tmp/fn113-1-review/round1-review.md. No agent commits.
## Evidence
- Commits:
- Tests: mise exec -- .flow/tmp/fn113-1/e2e-b1.sh  # launcher package + protoc 29.5 --scala_out=flat_package,scala3_sources + scala-cli --power package --library (Scala 3.9.0, JDK 27, lifter options): rc=0, 0 warnings, 659 classes, 3533296 B, 11.9 s, cd .flow/tmp/fn113-1 && mise exec -- scala-cli run --server=false printer-src -- samples.tsv  # scalapb-json4s 0.12.2 samples + 19 scalapb.descriptors checks: all ok, GOMAXPROCS=2 CC=/usr/bin/clang mise exec -- go run .flow/tmp/fn113-1/decode/decode.go .flow/tmp/fn113-1/samples.tsv  # 9/9 protojson.Unmarshal + proto.Equal, cd .flow/tmp/fn113-1 && mise exec -- scala-cli run --server=false roundtrip-src -- roundtrip <model/ir/*.json> <model/lifter/testdata/lifts/expected/*.json>  # parse + print with scalapb-json4s, GOMAXPROCS=2 CC=/usr/bin/clang mise exec -- go run .flow/tmp/fn113-1/compare/compare.go <original roundtrip pairs>  # 12/12 proto.Equal, mise exec -- scala-cli run --server=false --dep com.thesamet.scalapb:scalapbc_2.13:0.11.20 -M scalapb.ScalaPBC -- --protoc=$(mise which protoc) --version  # libprotoc 29.5 (25.8 without --protoc), mechanisms a, b1, b2 (after chmod copy), c: generated sources diff -r identical to gen-flat3 (logs gen-a.log, gen-b.log, gen-b2.log, gen-c.log), Independent review round 1 SHIP (claude-fable-5-1); .flow/tmp/fn113-1-review/round1-review.md
- PRs: