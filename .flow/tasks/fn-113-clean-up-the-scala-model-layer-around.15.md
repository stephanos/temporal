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
# fn-113.15 handover

The model README, SEMANTICS introduction, lifter header, and gate header now state the current pipeline: Scala declares, the ScalaPB lifter reads, and Go alone evaluates the IR. The README distinguishes compilation and lift refusals from Go-reported semantic Model problems at recorded Scala lines, and keeps its terms, diagram, worked example and gate instructions. Its Query totals were corrected from the checked-in manifest: 264 total, 16 lowered, 150 verify/nothing-to-realize, 95 no-realization, 3 unsupported; the Nexus caller has seven find Queries plus two verify Queries. The public Go `Table` reference in SEMANTICS remains accurate. `model/specimens/README.md` and `model/specimens/` are absent, so no stale-path edit was possible. No generic machinery or library was added or weighed (R25).

Baseline before edits: check-mode model gate exit 0 (`.flow/tmp/fn113-15/baseline-gate.log`); `mise exec -- make lint-model` exit 0 (`baseline-lint.log`). Verification after edits: `CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` exit 0 (`verify-gate.log`); `mise exec -- make lint-model` exit 0 (`verify-lint.log`); `git diff --check` exit 0 (`diff-check.log`). The lint command reports successful scalafmt, but both baseline and verification logs contain a Scalafix `NoSuchFieldException: path` despite exit 0; this is an inherited tooling limitation, not evidence that Scalafix rules ran cleanly. The only retired-vocabulary grep hit in these four files is `Table` in SEMANTICS, the real public Go type. The gate's vocabulary step passed.

Before/after SHA-256 maps for the four source files and all IR, Case and expected lift fixtures are `.flow/tmp/fn113-15/before-files.json` and `after-files.json` (34 files). Only the four touched source files changed; all IR, Cases and expected fixtures are byte-identical. The task-focused review snapshot is `.flow/tmp/fn113-15/snapshot.diff` (SHA-256 `8f792fa06401ba56808eedf774c4f747e4874ae5f1f74bec9240591ae4f671f1`). The Scala comment edits preserve the line counts in `Lift.scala` and `Gate.scala`, so they do not shift generated positions. The full Go model/lower/export packages, JSON, commands and walls passed in task 13 (`.flow/tmp/fn113-13/`); these documentation/comment changes do not invalidate those results, so they were reused rather than rerun.

Conductor module-map facts: the DSL row should describe declarations/realizations without the retired evaluator or prelude; the Lifter row should name ScalaPB generated IR and its runtime/ProtoJSON support; the Gate row should name ScalaPB generation. The Scala layout paragraph should say the native Scala evaluator is gone, the Nexus step/domain declarations live in `model/temporal/nexuscaller/Nexus.scala`, and compiler/lifter refusal fixtures remain under `model/lifter/testdata`. The conductor's current `.plans/UMPIRE_MODULES.md` diff includes these updates; it owns that shared file and MILESTONES.md. No further module-map inconsistency was found in the task's required rows.

No staging, commit, push, worktree, flowctl completion, plan-sync, tracker operation or review was performed. The task remains in progress for conductor review.

stage: impl-review - ran (model: gpt-6-sol; receipt: .flow/tmp/fn113-15-review/receipt.json; verdict: SHIP; uncommitted source pinned by snapshot.diff/source-hashes.json)

Conductor lint evidence: task 8's `.flow/tmp/fn113-8-review/lint-probe-forbidden.log` proves DisableSyntax.var enforcement despite the inherited caught path exception. The final task 15 changes affect prose and comment text only; the same tooling/environment evidence remains applicable.

stage: wave-dispatch - ran (model: gpt-6-sol; sequential worker in current checkout)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive).
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (baseline and verification: exit 0), mise exec -- make lint-model (baseline and verification: exit 0; inherited Scalafix exception in logs), git diff --check (exit 0), rg -n 'Table\b|Search|UmpireSet|Coverage|prelude|kernel|protobuf-java|JsonFormat|munit' model/README.md model/SEMANTICS.md model/lifter/Lift.scala model/gate/Gate.scala (one intended Go Table hit), Task13 full Go model/lower/export baseline reused: .flow/tmp/fn113-13/go-*-full-result.json, Independent codex:gpt-6-sol:high review SHIP; current source hashes verified
- PRs: