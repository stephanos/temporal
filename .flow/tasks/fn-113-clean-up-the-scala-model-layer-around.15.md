---
satisfies: [R12]
---
# fn-113-clean-up-the-scala-model-layer-around.15 Describe the layer as it is: README, SEMANTICS, lifter and gate comments

## Description
Describe the layer as it is: README, SEMANTICS, lifter and gate comments. Implements R12 for the model's documentation after Parts B, C and D landed.

**Size:** S
**Files:** model/README.md, model/SEMANTICS.md (two sentences), model/lifter/Lift.scala (header), model/gate/Gate.scala (comments), model/specimens/README.md (stale paths only)
**Touches:** [model/README.md, model/SEMANTICS.md, model/lifter/Lift.scala (comments), model/gate/Gate.scala (comments), model/specimens/README.md]

### Approach
- The README describes the layer as it is: Scala declares, the lifter reads (ScalaPB), Go evaluates; one evaluator; Model problems (a start outside the domain, a stuck state, a class bound twice, a failed refinement) are reported by `make umpire-check-model` from the IR at the Scala line, not by a munit test. The "Writing a Model" section says that, and names no removed file or type (`Table`, `Search`, `UmpireSet`, `Coverage`, `kernel`, `prelude`, `protobuf-java`, `JsonFormat`). Keep it readable for a newcomer (fn-115 R24's standard): the terms, the layers diagram, the worked example and the gate section stay coherent.
- `SEMANTICS.md:4` ("`tools/umpire/model` is one evaluator of these") and `:114` are reworded only where they imply a second evaluator.
- Lifter and gate comments: whatever tasks 4 and 13 left that still describes builders, the Java jar, the prelude or the kernel.
- The vocabulary step of the gate must pass; run it through a check-mode gate (documents under `model/` are read by `TestModelNamesNoRetiredFrontEnd`).
- Handover: the module map rows that changed across the spec (DSL row: no prelude, no evaluator; Lifter row: ScalaPB deps; Gate row; the sentence "The native Scala evaluator stays in the DSL until fn-113" is past) for the conductor.

### Investigation targets
**Required**:
- `model/README.md:1-20,83-125,127-152,271-330`
- `model/SEMANTICS.md:1-6,110-116`
- `model/lifter/Lift.scala:1-18`, `model/gate/Gate.scala:58-66`
- `model/specimens/README.md:75-85,175-185`
- `.plans/UMPIRE_MODULES.md:22-28,265-281`

### Quick commands
grep -n 'Table\b\|Search\|UmpireSet\|Coverage\|prelude\|kernel\|protobuf-java\|JsonFormat\|munit' model/README.md model/SEMANTICS.md model/lifter/Lift.scala model/gate/Gate.scala (only intended hits); CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; mise exec -- make lint-model

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `model/README.md`, `model/SEMANTICS.md`, the lifter's header and the gate's comments describe ScalaPB generation, one evaluator and Go-reported Model problems, and name no removed file, type or dependency.
- [ ] The README remains a newcomer's description of the whole system (terms, layers, worked example, gate); the vocabulary step passes in a check-mode gate; `lint-model` passes; the handover lists the module map rows for the conductor.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
