---
satisfies: [R3, R4, R5, R6, R7]
---
# fn-75-separate-lean-target-semantics-from.3 Qualify Target compatibility and trust and document the semantic boundary

## Description
Complete compatibility/trust evidence and document the final ownership after consumer migration.

**Size:** M
**Files:** Target compatibility/import/kernel tests, Planning replacement tests, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md.
**Touches:** [model/Umpire/Target/ImportTests.lean, model/Umpire/Target/SemanticsImportTests.lean, model/Umpire/Target/Tests/**, model/Umpire/TargetTests.lean, model/Umpire/Planning/Tests/**, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md]

### Approach
- Replace empty expected-error guards in `model/Umpire/Target/SemanticsImportTests.lean` with explicit expected diagnostics, as established by task 2's Planning import test. Demonstrate that each constructor/frontend guard fails when its protected identifier is made available in a controlled test. Preserve the positive semantic checks and independent actual metadata-closure proof.
- Compare moved trust-bearing declarations and default-argument dependencies with task 1's pre-edit inventories. Audit new proof declarations against the explicit existing assurance boundary; do not accept placeholders or extra compiler trust. Keep exact commands and full inventories in evidence.
- Reuse frozen compatibility/mutation/authoring tests without rewriting expected bytes. Verify canonical metadata, fingerprints, selected source spans and fallback errors against the baseline.
- Close only uncovered R6 tests: available and absent replacement planning, unchanged enumeration/order, and insufficient equivalence-proof rejection. Preserve independent completeness obligations and existing finite-machine/table tests.
- Update public import guidance and ownership diagrams; distinguish semantic consumers from ordinary authoring without adding another authoring language. Preserve module/declaration comments.
- Run combined model build and lint gates, including the existing experiment consumer through its established script. Run existing staleness checks when any owned generated surface is affected; do not expand broad generated API drift or CI scope.

### Investigation targets
**Required:**
- model/Umpire/Target/Tests/Compatibility — frozen fixtures
- model/Umpire/Planning/Tests/Fixtures.lean:219 — real replacement-kernel consumer
- model/Umpire/ARCHITECTURE.md:16 — public import table
- model/ARCHITECTURE.md:17 — dependency diagram
- model/README.md:66 — author import guidance
- experiments/umpire-dsl/run-baseline.sh:4 — additional consumer
- Makefile:1404 — complete model lint gate

### Verification
Run serial Lean commands with LEAN_NUM_THREADS=1: make umpire-build-model, make lint-model, and the existing experiment baseline script. Run make lint-code GOLANGCI_LINT_FIX=false with the established macOS compiler/TMPDIR environment. Compare any inherited diagnostics against the verified baseline; do not claim the lint gate clean or claim unreached phases ran. Task 3 owns final combined verification for all three tasks.

## Acceptance
- [ ] Before/after transitive axiom evidence proves no trust expansion, including checkedTarget's default argument.
- [ ] Frozen canonical/fingerprint and authoring diagnostic fixtures remain unchanged; available/unavailable planning and insufficient replacement proofs retain their contracts.
- [ ] Public import documentation describes the actual final boundary and preserves approachable authoring entrypoints.
- [ ] Full model build, model lint, affected experiment consumer and required non-fixing Go lint have recorded outcomes; inherited failures are precisely classified against baseline.

## Done summary
# fn-75 task3 handover

Task: `fn-75-separate-lean-target-semantics-from.3`, **in_progress**. Workspace: `/Users/stephan/Workspace/temporal/umpire`.
Base and HEAD: `375abfe180dba72da6dd357e6abe33fa75a292a7`; commits: `[]`. Conductor owns review, Flow lifecycle, and git. No writer verdict or completion claim.

### Final model-lint recovery passed

The complete `make lint-model` gate was rerun after restoring disk space and exited 0. Log: `/tmp/fn75-task3-conductor-lint-model.log`; command-local LEAN_NUM_THREADS=1, TMPDIR, compiler and SDK environment were unchanged, with `/usr/bin/time -l` recording 918.87 seconds and 7,035,863,040 bytes maximum resident set size. All six source hashes remain frozen. No source changes or test weakening were needed.

The initial builtin-lint SIGKILL and unknown-exit recovery/ENOSPC are historical failures, preserved in `/tmp/fn75-task3-initial-handover/` and original logs. Their precise cause is not inferred from the successful rerun. Disk space was restored by hard-linking 324 byte-identical immutable Git packs in completed fn78 scratch review clones, preserving every snapshot; all seven clone connectivity checks passed. Cleanup evidence: `/tmp/fn75-review-pack-dedup.json`.

### Implemented native scope

- R3: Target semantic and ordinary authoring import tests now require exact expected errors for both private constructors. Semantic import tests also require exact errors for occurrence capture and elaboration entrypoints. Existing positive semantic proof examples and declaration checks are unchanged.
- R4/R5: No canonical, fingerprint, typed error, source-span, fallback diagnostic, mutation, or authoring expectations were edited. All six original compatibility hashes match; focused TargetTests compiled their unchanged checks. Pure Data/Projection/Semantics/Frontend ownership and approachable authoring imports are documented in the three assigned public documents.
- R6: Existing available replacement coverage remains unchanged: `Planning.Tests.Fixtures.target` replaces the kernel with width-indexed duplicate enumeration and explicit relation-indexed planning; Enumeration checks adapter traversal, candidate counts/order, completeness, and bounded stopping. Existing finite-table/machine/validation tests remain unchanged. New generic regression proofs cover omitted replacement planning producing no Query completeness and the real planner returning `missingFiniteCompleteness`. These quantify over any appropriately equivalent checked Target, including one that had planning, without introducing another concrete native witness. A negative check rejects replacement missing behavior equivalence despite the other four obligations, and another independently rejects finite action evidence missing `actionComplete`.
- R7: Original task1 trust and actual metadata closure evidence is requalified below. All required command outcomes are captured, but final builtin lint and the complete aggregate gate now pass as described above.

### Controlled rejection evidence

Every corrected semantic identifier guard was tested with its identifier made available in scratch. Constructor specimens define the qualified name; frontend specimens import the actual `Umpire.Target.Frontend`. The corrected guards each fail with `Docstring on #guard_msgs does not match generated message`, rather than silently passing when no error occurs. Logs: `/tmp/fn75-task3-guard-red-CheckedTarget-mk.log`, `...-AuthoredTarget-mk.log`, `...-captureAuthoringOccurrence.log`, and `...-elaborateTarget.log`.

The absence-control specimen retains `original.planning` during replacement and breaks both generic absence/planner-rejection proofs. The proof-control specimen supplies behavior equivalence and action completeness: the two successful checks then cause their expected-error guards to fail. Logs: `/tmp/fn75-task3-planning-available-control.log` and `/tmp/fn75-task3-planning-proof-control2.log`. The earlier malformed scratch proof-control is retained and superseded, not cited as the valid action-completeness control.

Independent actual `.olean` metadata traversal still finds 1,141 modules in the semantic import test closure and no `Lean.Elab.Term`. Full model graph lint additionally enforces the protected production roots and forbidden Target frontend transitively. The name-exposure controls are not substituted for that actual-closure evidence.

### Original trust and fixture comparison

The comparison uses the **original task1 pre-edit** `/tmp/fn75-task1-trust-before.log`, `.json`, and original Language source, with the immutable task1 declaration mapping. It does not use task3 start as the trust baseline. All 1,293 original declarations map to the final inventory with no missing or changed transitive axiom set. The final 1,294 declarations exactly match task1's complete raw after-log inventory, including its already-accounted empty-axiom generated match auxiliary.

Four original JSON entries truncate multiline axiom output to `[propext,`. The preserved raw log contains the full original `[propext, Classical.choice, Quot.sound]` sets. Task3 parses complete bracketed inventories from both raw logs and records those four JSON truncations explicitly; no original artifact is rewritten. `/tmp/fn75-task3-trust-before-full.json` is the corrected task3-side parsed inventory, and `/tmp/fn75-task3-trust-comparison.json` contains every relocation and default mapping.

`checkedTarget` and its `_auto_1` printed declaration/default are byte-for-byte identical to the original pre-edit print. Its default remains inert `native_decide` syntax with the original caller trust; it adds no new permission. The nine new private test declarations/auxiliaries have only the existing replacement API's `propext`, `Classical.choice`, and `Quot.sound` dependencies, or empty inventories for four generated reflexivity proofs. There are no new custom/compiler axioms or placeholders. Existing anonymous positive import examples remain unchanged checked commands rather than exported declarations.

All six compatibility source/fixture files match task1's original hashes. Canonical metadata fixture: `ab4d019ec83d9d40973fd547f9e691e732d65b45a933d915dc8f65e7b711fb8b`; fingerprint fixture: `582bc38d3b4085a05305c9f29b31093d5d8a243b90ea0a44f43f46950d99db7d`. Comparison: `/tmp/fn75-task3-compatibility-comparison.json`.

### Other terminal verification

Exact commands, cwd, logs, durations, and terminal statuses are in `/tmp/fn75-task3-commands.json` and `/tmp/fn75-task3-evidence.json`. All Lean commands were serial with `LEAN_NUM_THREADS=1`, the established mise 4.33.1 toolchain, and command-local macOS TMPDIR/CC/SDKROOT. Handles were observed to termination; no command was restarted for an observation timeout.

- Focused `lake build Umpire.TargetTests Umpire.Target.ImportTests Umpire.Target.SemanticsImportTests Umpire.Planning.Tests`: exit 0, 78 jobs.
- Full `make umpire-build-model`: exit 0, 514 jobs.
- Existing experiment baseline script: exit 0, 3,510 comparisons over 585 words, lengths 0..3 and bounds 0..2. The script uses the model workspace plus the existing experiment compiled Property module; both toolchains/configurations were inspected.
- Required non-fixing `make lint-code GOLANGCI_LINT_FIX=false`: exit 2, exactly inherited **1,284 raw / 825 distinct (path,message)** diagnostics, zero additions/removals against task2's final baseline. The separate Makefile go-vet phase was not reached. Comparison: `/tmp/fn75-task3-lint-comparison.json`.
- Trust audits, actual semantic closure, scoped whitespace, and final preservation comparison: exit 0.

Intermediate scratch exploration failures, the initial missing scratch-file probe, and the malformed first proof-control are retained in the ledger/logs and not represented as successful tests. No Go/protocol/generated source changed, so Go tests and additional generator checks were not relevant. The required lint-model prerequisite did run its existing semantic-inventory staleness check.

### Frozen source handover

The six-file task3-only patch is `/tmp/fn75-task3.patch`; paths and final hashes are `/tmp/fn75-task3-paths.txt` and `/tmp/fn75-task3-frozen-hashes.json`. It compares inherited dirty files against the **original immutable task3 baseline copies/hashes**, and originally clean files against the pinned base commit. It excludes every inherited source delta.

`/tmp/fn75-task3-preservation.json` proves no unrelated source changes against either the complete start snapshot or original dirty baseline. Every existing test declaration body is identical after removing only the new regression section/guard expectations. All existing Lean comments are preserved. All 18 task2 hashes match; the only task1 frozen overlap changed is the explicitly authorized semantic import test. Production Target bodies and every other task1 source remain byte-identical. Original baseline files, base commit, task1/task2 evidence, Flow records, review metadata, and cancellation source were not written. No git staging, commits, push, worktrees, reverts, delegation, or child bridges occurred.

stage: impl-review - ran SHIP (model: gpt-6-astra at medium)

**No further source edits.** Task remains `in_progress`; the conductor owns resource recovery, review and done.

Implementation review SHIP, zero findings; receipt `/tmp/fn75-task3-impl-review.json`, session `01a080e2-1eec-7be0-ac6a-2443520cc72d`.
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-probe-identifiers.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-probe-identifiers.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-probe.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-probe.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-guard-CheckedTarget-mk.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-guard-AuthoredTarget-mk.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-guard-captureAuthoringOccurrence.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-guard-elaborateTarget.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-probe.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-generic.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-generic.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-beq.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-generic.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-generic.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.TargetTests Umpire.Target.ImportTests Umpire.Target.SemanticsImportTests Umpire.Planning.Tests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-available-control.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-proof-control.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-trust-audit.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-test-trust-audit.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task3-planning-proof-control2.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean --run /tmp/fn75-task3-trust-closure.lean Umpire.Target.SemanticsImportTests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make umpire-build-model, git diff --check -- model/Umpire/Target/ImportTests.lean model/Umpire/Target/SemanticsImportTests.lean model/Umpire/Planning/Tests/Enumeration.lean model/Umpire/ARCHITECTURE.md model/ARCHITECTURE.md model/README.md, python3 /tmp/fn75-task3-compare.py, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-model, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- experiments/umpire-dsl/run-baseline.sh, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false, python3 /tmp/fn75-task3-compare-go-lint.py, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) /usr/bin/time -l mise exec -- lake --wfail lint --builtin-only --lint-only=.all,.extra,-.missingDocs, python3 /tmp/fn75-task3-compare.py, /Users/stephan/.codex/plugins/cache/flow-next-marketplace/flow-next/4.5.1/scripts/flowctl show fn-75-separate-lean-target-semantics-from.3 --json, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) /usr/bin/time -l make lint-model | log=/tmp/fn75-task3-conductor-lint-model.log | exit=0
- PRs: