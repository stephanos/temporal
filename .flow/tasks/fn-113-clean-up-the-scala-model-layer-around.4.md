---
satisfies: [R5, R6, R10, R12]
---
# fn-113-clean-up-the-scala-model-layer-around.4 Retire the Java generation path and document the ScalaPB pipeline

## Description
Retire the Java generation path and document the ScalaPB pipeline. Closes Part B: implements the remaining clauses of R5, R6 and R10 and the lifter and gate half of R12.

**Size:** S
**Files:** model/gate/Gate.scala, model/gate/test/Gate.test.scala, Makefile (`MODEL_PROTO_JARS`), model/lifter/project.scala (jar name only), model/lifter/Lift.scala (header comment), model/README.md (the lifter and gate lines)
**Touches:** [model/gate/**, Makefile (MODEL_PROTO_JARS), model/lifter/project.scala, model/lifter/Lift.scala (comments only), model/README.md (lifter and gate lines only)]

### Approach
- Remove `--java_out`, the protobuf-java packaging and its version comment from `Gate.generateIr`; one ScalaPB jar remains. Either keep the name `model/gen/ir-proto.jar` (so the Makefile prerequisite and the lifter's `using jar` stay) or rename everywhere consistently; the stamp covers schema and generator versions. Gate tests follow.
- Prove R5 end to end: move the jar and stamp to `.flow/tmp/trash/fn113-4/`, run `--generate-ir` (regenerates), `--generate-ir --if-stale` (no work), then a check-mode gate; perturb the stamp and show the gate names the jar and the schema.
- Prove R6: `grep -rn 'com.google.protobuf\|protobuf-java' model/lifter model/gate` is empty.
- R10: `make lint-model` visits no generated source; `make lint-code-fast` passes (no Go change expected).
- R12 (this half): the lifter's header comment in `Lift.scala`, the gate's comments and `model/README.md`'s lifter and gate rows describe ScalaPB generation and name no removed thing (no `protobuf-java`, no `JsonFormat`). The README's evaluator wording is task 15's.
- Handover for the conductor: the module map's Lifter row (dependencies) and Gate row (`--generate-ir` now emits ScalaPB), the manifest deltas.

### Investigation targets
**Required**:
- `model/gate/Gate.scala:58-108` (`generateIr` after task 2)
- `model/gate/test/Gate.test.scala` (generation tests)
- `Makefile:671,678-680` (`MODEL_PROTO_JARS` and its rule)
- `model/lifter/project.scala:1-8`
- `model/lifter/Lift.scala:1-18` (header)
- `model/README.md:118,277,305-317` (lifting row, lifter row, "Writing a Model")

### Quick commands
mise exec -- scala-cli test model/gate; mise exec -- scala-cli run --suppress-outdated-dependency-warning model/gate -- --generate-ir; the same with --if-stale; CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; mise exec -- make lint-model; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] The gate generates only ScalaPB classes from the unchanged schema into the jar the lifter reads, regenerates only when the schema or generator version changed, and a stale or missing jar fails with the jar and the schema named; the gate suite covers it.
- [ ] No file under `model/lifter` or `model/gate` names `com.google.protobuf`, `protobuf-java` or `protobuf-java-util`; `make lint-model` lints no generated source.
- [ ] The lifter's header, the gate's comments and the README's lifter and gate lines describe ScalaPB generation and name nothing removed; `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`, `lint-model` and `lint-code-fast` pass with no change under `model/ir`, `model/cases` or `model/lifter/testdata`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
