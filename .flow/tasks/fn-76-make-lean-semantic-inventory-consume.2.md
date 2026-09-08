---
satisfies: [R2, R7]
---
# fn-76-make-lean-semantic-inventory-consume.2 Enforce inventory and neutral-classification dependency boundaries

## Description
Extend the existing graph policy for the migrated ownership boundary.

**Size:** M
**Files:** ModelLint.ImportGraph and ModelLint.ImportGraphTests.
**Touches:** [model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean]

### Approach
- Reuse existing production/test classification and fn75 external-aware traversal. Reject SemanticInventory reachability from every production Umpire source outside that namespace, including root umbrella, newly added helpers, Artifact.Result and ImplementationLink; do not enumerate only current offenders.
- Enforce neutral OutcomeClassification independence from concrete owners, Core/KnownGap, inventory, Temporal and aggregation, allowing only its selected minimal foundation. Do not disguise it as inventory.
- Add direct, helper, facade, external-bridge and test-fixture-bridge rejection; allowed inventory-to-owner and dedicated Planning KnownGap test cases; cycles and deterministic shortest paths. Preserve fn75 Target isolation and missing-owned-metadata reconciliation.

### Investigation targets
**Required:**
- model/ModelLint/ImportGraph.lean:237
- model/ModelLint/ImportGraph.lean:258
- model/ModelLint/ImportGraphTests.lean:314
- model/Tools/LeanImportGraph.lean
- model/Tools/LeanImportGraph/Metadata.lean
- model/Umpire/Planning/Tests/KnownGaps.lean:1

### Verification
From model, serial mise exec -- lake build modelLintTests modelLint; then lake exe modelLintTests and lake exe modelLint. Preserve the exact controlled-violation diagnostic. Run required non-fixing Go lint and classify inherited output precisely.

## Acceptance
- [ ] Universal production inventory isolation and minimal neutral-module independence are enforced transitively with deterministic diagnostics.
- [ ] Direct/indirect/external/test-bridge negatives and reverse-direction/test-consumer positives pass.
- [ ] Existing graph policies, inventory reconciliation and controlled diagnostic remain unchanged.
- [ ] Real whole-source modelLint passes in addition to the focused synthetic regressions.

## Done summary
Implemented universal transitive production Umpire → SemanticInventory isolation and minimal Init-foundation isolation for Umpire.OutcomeClassification by extending the existing graph policy. Only model/ModelLint/ImportGraph.lean and model/ModelLint/ImportGraphTests.lean changed relative to the original dirty checkout; existing traversal, fn75 Target rules, reconciliation, test assertions and comments remain intact.

The three new table-driven regressions cover umbrella/helper/facade/Artifact/ImplementationLink sources, direct and external/test-support/test-consumer bridges, missing-record endpoints, reverse inventory-to-owner consumption, dedicated Planning KnownGap tests, forbidden neutral owners/aggregation/external dependencies, allowed Init prelude/list foundation, cycles, lexical ties, reversed metadata and direct shortest paths. TDD captured successful test compilation followed by the expected missing inventory rejection, then the expected missing neutral rejection; the final synthetic suite passed.

Fresh task2 baseline and final gates passed: serial modelLintTests/modelLint build; modelLintTests executable; real whole-source modelLint (406 jobs, full graph reconciliation and Batteries lint for Shared, Temporal.Lint and Umpire.Lint). Controlled-violation baseline/final exited 1 with exactly the same expected diagnostic bytes. Complete commands, physical cwd, terminal exits, durations and logs are recorded in /tmp/fn76-task2-evidence.json and /tmp/fn76-task2-gates.jsonl. Lean ran serially using LEAN_NUM_THREADS=1, pinned mise, and command-local TMPDIR=/private/tmp plus xcrun CC/SDKROOT. No suite was repeated merely to observe its result and no gate was skipped via handoff or Flow receipt.

Inherited failure: baseline/final non-fixing `mise exec -- make lint-code GOLANGCI_LINT_FIX=false` each exited 2. Both match the task1 reference log exactly: 1,284 raw / 825 distinct path-and-message diagnostics, identical multisets, no added or removed findings. This is not clean lint. Make's separate go-vet phase was not reached. Exact comparison and log hashes: /tmp/fn76-task2-lint-comparison.json. No Go tests were needed. Full builtin model lint and remaining task3 qualification are still task3 work; the real whole-source modelLint task2 gate was executed.

Original dirty copies and HEAD/index receipt: /tmp/fn76-task2-baseline/manifest.json (copies under files/). Full 7,619-file baseline hashes: /tmp/fn76-task2-baseline/workspace-hashes.json. Baseline artifact hash manifest: /tmp/fn76-task2-baseline-artifact-hashes.json. Source integrity: /tmp/fn76-task2-source-integrity.json. Final freeze rechecked the full tracked/untracked workspace and found only the two scoped changes, unchanged HEAD/index, unchanged captured task1 artifacts and unchanged baseline copies. Patch is against the original dirty files, not HEAD: /tmp/fn76-task2.patch. Changed paths: /tmp/fn76-task2-changed-paths.txt. Frozen source/workspace/evidence hashes: /tmp/fn76-task2-frozen-hashes.json.

No semantic or trust-bearing proof declarations were changed, and no axiom/compiler-trust path was added. Init is Lean's implicit foundation; the actual neutral source retains Init.Data.List.Basic as its sole explicit import. Disk headroom last observed about 1.4 GiB; no cleanup, unrelated process termination, git staging/commits/push/worktrees/revert, Flow mutations, delegation, review dispatch or task3/cancellation implementation occurred. Commits=[]; base_commit=375abfe180dba72da6dd357e6abe33fa75a292a7; task remains in_progress for the conductor.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

No further source edits: source is frozen for conductor scoped review. No source edits will occur after this summary.

Conductor review: SHIP, no findings, receipt /tmp/fn76-task2-impl-review.json, actual gpt-6-astra medium session01a0810e-ffff-7d72-b2a1-5a152aa7c73b. Scratch base2fbac9c2418997ba6c8b8776dd0420bb44592da8 head1c598b249cd765514a8c6d3149644792ae9da75f; both final source hashes verified unchanged by conductor. Reviewer could not independently rerun tests because scratch mise config was untrusted; conductor verified recorded real-checkout gates and exact inherited lint comparison. Scratch commits only; real checkout uncommitted. Tracker inactive. Downstream task3 already targets final ownership and policy, so no plan synchronization changes required.
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake build modelLintTests modelLint, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake exe modelLintTests, mise exec -- make lint-code GOLANGCI_LINT_FIX=false, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake exe modelLint, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake exe modelLintTests --controlled-violation, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake exe modelLintTests, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake exe modelLintTests, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake build modelLintTests modelLint, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake exe modelLintTests, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake exe modelLint, python3 /tmp/fn76-task2-verify-source.py, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 mise exec -- lake exe modelLintTests --controlled-violation, mise exec -- make lint-code GOLANGCI_LINT_FIX=false, python3 /tmp/fn76-task2-compare-lint.py /tmp/fn76-task2-post-lint-code.log
- PRs: