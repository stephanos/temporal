---
satisfies: [R1, R3, R4, R5, R6, R7]
---
# fn-75-separate-lean-target-semantics-from.1 Extract checked Target semantics and pure admission ownership

## Description
Establish the Target ownership split and prove the narrow semantic import before consumer migration.

**Size:** M
**Files:** model/Umpire/Target/Language.lean, model/Umpire/Target.lean, new Target semantic/projection/frontend modules, Target import tests and finite adapters as required.
**Touches:** [model/Umpire/Target/**, model/Umpire/Target.lean]

### Approach
- Before editing, capture exact canonical compatibility fixture hashes and transitive axiom inventories for moved checked/admission/projection/replacement declarations and checkedTarget's default argument. Persist task-unique baseline artifacts and declaration mapping for task 3; record commands and inventories in evidence. Do not derive the baseline after extraction.
- Separate pure representation/projection from syntax capture and TermElabM integration. Keep private checked construction with its single admission authority; pure occurrence/diagnostic data may remain there when needed. Preserve names and signatures through the existing authoring facade.
- Introduce Umpire.Target.Semantics and a positive import test that exercises kernel and finite planning contracts. Verify actual import closure, including dependencies of default arguments. Preserve checkedTarget's default-call contract without using an unchecked constructor bridge. Resolve any conflict here before migrating consumers.
- Preserve canonical implementations and the withEquivalentKernel proof obligations. Retain finite adapters in their existing owners, with narrower imports only where required by the new ownership. Move all existing comments with declarations and document new module boundaries.

### Investigation targets
**Required:**
- model/Umpire/Target/Language.lean:73 — private assembly, pure checking and elaboration currently colocated
- model/Umpire/Target/ImportTests.lean:47 — constructor rejection guards
- model/Umpire/Target/Tests/Compatibility — frozen byte/fingerprint fixtures
- model/Umpire/Target/Tests/FiniteMachine.lean:474 — existing axiom audit pattern
- model/Umpire/Target/Tests/FiniteTable.lean:124 — finite trust checks
**Optional:**
- model/Umpire/Planning/Tests/Fixtures.lean:219 — checked kernel replacement consumer

### Verification
Run serially with LEAN_NUM_THREADS=1 using the established model toolchain: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Target.ImportTests, plus the new positive semantic-import root. Verify imported module closure and preserved baseline fixtures immediately. No generated outputs or toolchain changes are expected.

## Acceptance
- [ ] Pre-edit trust inventories, declaration mapping and fixture hashes are captured and handed to task 3.
- [ ] Semantic import and ordinary authoring imports compile; constructor-forgery guards remain effective and no elaborator dependency enters the semantic closure.
- [ ] Existing Target validation, authoring diagnostic, finite adapter and compatibility tests pass with unchanged expected bytes and comments preserved.
- [ ] R1/R3/R4 ownership and default-argument trust constraints have an executable early proof; any unresolved split is replanned before task 2.

## Done summary
# fn-75 task 1 frozen handover

Worker handover was in_progress; conductor verified SHIP before recording done. Base commit: `375abfe180dba72da6dd357e6abe33fa75a292a7`. Commits: `[]`; no staging, commits, push, worktrees, rollback, delegation, or Flow lifecycle writes.

Extracted Target into pure `Data`, pure `Projection`, checked representation/admission in `Semantics`, and syntax capture/TermElabM in `Frontend`. `Language` remains the authoring compatibility facade; the ordinary `Umpire.Target` facade is unchanged. FiniteMachine retains its implementation and now imports Semantics. Shared projection helpers moved from private Language names to the implementation namespace `Umpire.TargetProjection`; no unchecked assembly bridge was introduced. Checked and authored constructors remain private in the admission owner.

The seven-file task-only patch is `/tmp/fn75-task1.patch`; its paths and final SHA-256 values are `/tmp/fn75-task1-paths.txt` and `/tmp/fn75-task1-frozen-hashes.json`. Dirty-before-task files were diffed against the conductor's immutable copies and verified hashes; clean-before-task files were compared to the pinned base. Original baseline artifacts were not overwritten.

### Early boundary proof

`SemanticsImportTests` compiles using only the narrow import and exercises authoritative kernel soundness, finite action completeness, the checked APIs and replacement seam, and rejection of private constructors and frontend entrypoints. Before extraction, the metadata checker rejected `Umpire.Target.Language -> Lean.Elab.Term`. After extraction, actual recursively loaded `.olean` metadata reports Semantics's 1,140-module closure without Lean.Elab.Term. Final positive-test closure has 1,141 modules and no forbidden dependency. The checker visits external dependencies and retains metadata regions while traversing. The broad author's imports and located diagnostic tests pass.

`checkedTarget` preserves its exact argument list, default `by native_decide`, and implementation. Its generated `_auto_1` is inert tactic syntax with an empty transitive axiom inventory; before/after printed declarations (including this default) are identical. Caller native evaluation keeps its established trust boundary. No default was replaced by a validity flag, new proof assumption, raw constructor, or elaboration bridge. The actual narrow metadata closure is the proof that removing the direct import did not leave a transitive Term dependency.

### Verification and exact commands

Every Lean command ran strictly serially with LEAN_NUM_THREADS=1 on 4.33.1 using mise/lake and command-local macOS TMPDIR, CC, and SDKROOT. `/tmp/fn75-task1-commands.json` records each exact invocation, cwd, log, terminal exit, and expected failure interpretation. Logs were read without rerunning commands merely to observe status.

- Pre-edit Quick: `lake build Umpire.TargetTests Umpire.Target.ImportTests`, exit 0; `/tmp/fn75-task1-quick-baseline.log`.
- Final focused: `lake build Umpire.TargetTests Umpire.Target.ImportTests Umpire.Target.SemanticsImportTests`, exit 0; `/tmp/fn75-task1-focused-verified.log`. Includes unchanged default-call tests in Validation, finite machine/table tests, canonical fixtures, and authoring diagnostics. The final rerun followed a source cleanup removing an empty namespace scaffold.
- Trust audits: `lake env lean /tmp/fn75-task1-trust-audit.lean`, before and after exits 0.
- Metadata closure: `lake env lean --run /tmp/fn75-task1-trust-closure.lean Umpire.Target.SemanticsImportTests`, exit 0; `/tmp/fn75-task1-trust-semantic-import-closure-final.log`.
- Required `make lint-code GOLANGCI_LINT_FIX=false`, make exit 2 (golangci-lint failure); `/tmp/fn75-task1-lint-code.log`. Comparison to `/tmp/fn74-task3-lint-final.log`: **1,284 raw / 825 normalized distinct on both sides; exactly zero additions/removals**. Full comparison: `/tmp/fn75-task1-lint-comparison.json`. The subsequent make go-vet phase was not reached. No Go source changed after this run.
- `git diff --check -- model/Umpire/Target model/Umpire/Target.lean`, exit 0. All original Language comments retained; exact relocated implementation bodies match the captured source after the documented private helper name mapping: `/tmp/fn75-task1-relocation-check.json`.

There were two resolved working-directory mistakes in scratch/build commands and one intermediate compile failure for three private ordering helpers; all are recorded with their failed logs and successful follow-ups. The intentional red import test failed before the semantic implementation existed.

### Trust and compatibility substrate for task 3

Captured **before the first source edit**, not reconstructed afterward:

- `/tmp/fn75-task1-trust-Language-before.lean`: exact original Language source.
- `/tmp/fn75-task1-trust-before.log` and `/tmp/fn75-task1-trust-before.json`: complete 1,293-declaration transitive inventories, including private assembly/admission, projections, checked extraction, replacement, finite adapters, and checkedTarget default.
- `/tmp/fn75-task1-compatibility-before.json`: SHA-256 inventory of all six compatibility source/fixture files. The pre-edit Quick run verified expected canonical outputs before hashing/extraction.

After/mapping artifacts:

- `/tmp/fn75-task1-trust-after.log` and `/tmp/fn75-task1-trust-after.json`.
- `/tmp/fn75-task1-trust-declaration-mapping.json`: complete old/new declaration mapping with both axiom lists; **zero missing declarations and zero expanded/changed inventories**. One newly generated private projection match auxiliary has an empty inventory (1,294 after declarations).
- `/tmp/fn75-task1-trust-source-mapping.json`: original declaration line ranges and shared private-helper rename mapping.
- `/tmp/fn75-task1-trust-audit.lean`: executable inventory command source using Lean.Util.CollectAxioms over actual imported Target declaration owners, including generated defaults.
- `/tmp/fn75-task1-trust-closure.lean`: executable actual metadata traversal command source. It is a temporary audit utility, not a production assumption or new dependency.
- `/tmp/fn75-task1-compatibility-after.json` and `/tmp/fn75-task1-compatibility-comparison.json`: all six original file hashes unchanged. Metadata fixture SHA-256: `ab4d019ec83d9d40973fd547f9e691e732d65b45a933d915dc8f65e7b711fb8b`; fingerprint fixture SHA-256: `582bc38d3b4085a05305c9f29b31093d5d8a243b90ea0a44f43f46950d99db7d`. Hash command was Python pathlib traversal plus hashlib.sha256 over exact file bytes; expected bytes were never rewritten.

### Scope and limitations

Task 2 owns semantic consumer migration and permanent negative import-graph enforcement; task 3 owns combined model build/lint and additional qualification. Neither was attempted here. No generated/protocol/Go/toolchain surface changed, so no generator or Go tests were warranted. No claim is made that the repository's full lint passes.

Drift check `/tmp/fn75-task1-source-drift-check.json` found the seven owned Target files, administrative `.flow` updates, and `.plans/UMPIRE4_ORDER.md` outside this worker's writes. That planning document was clean in the pinned baseline and now contains delivery-order updates; it was preserved and excluded from the patch. The worker did not modify unrelated implementation sources or prior fn78/fn74 source. Conductor should account for the independently modified planning document.

stage: impl-review - ran; SHIP, no findings; receipt /tmp/fn75-task1-impl-review.json (model: gpt-6-astra at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)

**Frozen: no further source edits will be made by this worker.** Summary/evidence are at the assigned paths; the task remains in_progress. Review verdict, lifecycle completion, memory, gates, and base/receipt files remain conductor-owned.
## Evidence
- Commits:
- Tests: LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.TargetTests Umpire.Target.ImportTests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task1-trust-audit.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task1-trust-pure-probe.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task1-trust-pure-probe.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean --run /tmp/fn75-task1-trust-closure.lean Umpire.Core, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean --run /tmp/fn75-task1-trust-closure.lean Umpire.Target.Language, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean Umpire/Target/SemanticsImportTests.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.Target.SemanticsImportTests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.Target.SemanticsImportTests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.Target.SemanticsImportTests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean --run /tmp/fn75-task1-trust-closure.lean Umpire.Target.Semantics, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 make lint-code GOLANGCI_LINT_FIX=false, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.TargetTests Umpire.Target.ImportTests Umpire.Target.SemanticsImportTests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.TargetTests Umpire.Target.ImportTests Umpire.Target.SemanticsImportTests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn75-task1-trust-audit.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean --run /tmp/fn75-task1-trust-closure.lean Umpire.Target.SemanticsImportTests
- PRs: