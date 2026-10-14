---
satisfies: [R1, R2, R3, R4, R6, R7]
---
# fn-76-make-lean-semantic-inventory-consume.1 Extract semantic classification and Known Gap carry contracts and migrate consumers

## Description
Relocate the existing shared contracts and complete their import migration in one compilable change. Capture the original trust and output evidence before editing.

**Size:** M
**Files:** Umpire.OutcomeClassification, KnownGap, SemanticInventory.Types, five semantic owners, Umpire umbrella, affected explicit inventory consumers and import tests.
**Touches:** [model/Umpire.lean, model/Umpire/**, model/Temporal/**/*.lean]

### Approach
- Before source edits, capture original moved declaration bodies, generated inventory bytes, relevant canonical fixture hashes, and full raw transitive axiom inventories. Include all ten concrete ExactlyOne proofs, carry mappings/renderings, projection sentinel, and generated auxiliaries; retain complete multiline arrays and qualified-name mapping.
- Extract classifier/projection declarations from SemanticInventory.Types into OutcomeClassification using only the minimal established Lean/Std foundation. Move KnownGapCarryMapping/name to KnownGap. Preserve names, signatures, bodies and comments; leave concrete classifiers/proofs/mappings at their owners.
- Replace all five production Types imports with semantic owners; add explicit KnownGap imports to Result/Evidence consumers where needed. Remove inventory from the Umpire umbrella and migrate exposed catalog consumers to explicit imports. Wider Umpire/Temporal Touches permit these import-only repairs, not unrelated body changes. Do not reexport through Core or leave broken consumers for task 2.
- Add a focused import qualification proving the neutral public contract compiles without inventory or concrete-stage imports.

### Investigation targets
**Required:**
- model/Umpire/SemanticInventory/Types.lean:13
- model/Umpire/KnownGap.lean:1
- model/Umpire/Planning/Engine.lean:576
- model/Umpire/ImplementationLink/Application.lean:3
- model/Umpire/Artifact/Result.lean:268
- model/Umpire.lean:14
- model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean:104

### Verification
Use serial LEAN_NUM_THREADS=1 and mise. From model, build Umpire.SemanticInventory.Tests.PlanningRuntime Umpire.SemanticInventory.Tests.SemanticStages Umpire.SemanticInventory.Tests.KnownGaps Umpire.Planning.Tests.KnownGaps Temporal.Tool.SemanticInventoryTests and the new neutral import test. Run make umpire-check-semantic-inventory without publishing. Record required non-fixing Go lint against the inherited baseline.

## Acceptance
- [ ] Original raw trust inventories and output/fixture hashes are captured before edits and remain immutable.
- [ ] Shared contracts have their planned semantic owners; all production imports and affected catalog consumers compile with unchanged declaration bodies and qualified APIs.
- [ ] Focused outcome/carry/catalog tests and neutral import qualification pass; checked inventory bytes remain unchanged.
- [ ] No new semantic behavior, proof trust, generated expectations, or fn75/fn79 scope changes are introduced.

## Done summary
Task fn-76-make-lean-semantic-inventory-consume.1 passed conductor implementation review.

Extracted existing classifier/projection contracts into Umpire.OutcomeClassification (Init.Data.List.Basic only), relocated carry contracts to Umpire.KnownGap, repaired five semantic-owner imports plus explicit Result/Evidence and catalog-test imports, removed inventory from Umpire umbrella, and added the neutral import qualification. All original source declaration blocks and comments are preserved; other existing-file edits are import-only. No task2 policy or task3 qualification changes.

Baseline: /tmp/fn76-task1-baseline/copies contains original dirty/scoped source and fixture copies; /tmp/fn76-task1-baseline/manifest.json hashes 7,617 original files. HEAD/index receipt: /tmp/fn76-task1-baseline/head-index.json. Immutable baseline artifact manifest: /tmp/fn76-task1-baseline-artifact-hashes.json. Raw full declaration bodies and complete multiline axiom arrays: /tmp/fn76-task1-baseline-trust-complete.log. Parser: /tmp/fn76-task1-parse-trust.py; parsed arrays: /tmp/fn76-task1-baseline-trust.json; exact qualified-name/module mapping: /tmp/fn76-task1-baseline/declaration-mapping.json. The first nine-proof capture was corrected to ten proofs before source edits, and both logs remain retained.

Verification: baseline and final focused Lean builds passed; final includes all five required test modules, neutral import test, and Umpire umbrella (174 jobs). Neutral test failed before extraction as expected. Baseline/final make umpire-check-semantic-inventory passed without publishing. All 616 inventory/testdata hashes match, including checked inventory SHA-256 d3de733b9a0d6aa7e163671ae152f84d581a108e35f2377a883b8b88b90ff6ec. Full command strings, physical cwd, logs, timings and terminal exits are in /tmp/fn76-task1-evidence.json and /tmp/fn76-task1-commands.jsonl. Lean jobs ran serially with LEAN_NUM_THREADS=1, pinned mise toolchain, and command-local TMPDIR/CC/SDKROOT.

Trust: all 140 qualified declaration names and complete raw axiom arrays match. All ten owner ExactlyOne proofs remain axiom-free; seven generated/representation declarations retain only inherited propext. Elaborated declaration text is identical except for two generated noConfusion universe binders whose module qualifiers changed; the exact two-name alpha mapping and raw diffs are retained in /tmp/fn76-task1-trust-comparison.json. No assumptions, compiler-trust paths, owner bodies, status descriptions/order, carry renderings, sentinel membership or checked boundaries changed.

Inherited failure: baseline and final `mise exec -- make lint-code GOLANGCI_LINT_FIX=false` exited 2. Both match fn75 exactly at 1,284 raw / 825 distinct path-and-message diagnostics, including raw multisets; no additions/removals. This is not clean lint. Make's separate go-vet phase was not reached. Comparison: /tmp/fn76-task1-lint-comparison.json. Full model lint/build and renderer IO regressions remain task3 gates under the dispatch charter. No Go tests were needed for this Lean-only extraction.

Verification tooling corrections: the first source-manifest comparison misclassified an existing gitlink directory as a new file; file-only filtering corrected the helper, and final verification passed. Strict raw body comparison initially detected the two generated universe qualifiers above; the explicit alpha mapping resolves only those names, retaining every raw body. No source edits were needed for either correction.

Final source verification found precisely 12 scoped changed paths, no unrelated file changes, all original comments retained, all baseline artifacts retained unchanged, and unchanged HEAD/index. Disk headroom last observed approximately 2.6 GiB; no cleanup or unrelated process termination. No git staging, commits, push, worktrees, Flow mutations, agents, or reviews performed. Commits=[]; base_commit=375abfe180dba72da6dd357e6abe33fa75a292a7.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

No further source edits: source is frozen for conductor scoped-clone review. Patch against the original dirty baseline: /tmp/fn76-task1.patch. Changed paths: /tmp/fn76-task1-changed-paths.txt. Final source/full-workspace and evidence hashes: /tmp/fn76-task1-frozen-hashes.json. Handoff evidence: /tmp/fn76-task1-evidence.json.

Conductor review: SHIP, no findings, /tmp/fn76-task1-impl-review.json; actual Codex session 01a080fe-a93e-74d1-bba7-7856446e7989, gpt-6-astra medium. Scoped scratch clone base d916c809df7e6a8e5d53320b24631727474e804d, head 89c5ef8a7da4bdf87425e87fe6eb2ca0545c412b. Reviewer verified all12 frozen source hashes and140 axiom arrays. Scratch commits only; real checkout remains uncommitted. Tracker sync inactive; plan-sync unnecessary because downstream tasks already cover final interfaces.
## Evidence
- Commits:
- Tests: LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime Umpire.SemanticInventory.Tests.SemanticStages Umpire.SemanticInventory.Tests.KnownGaps Umpire.Planning.Tests.KnownGaps Temporal.Tool.SemanticInventoryTests, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn76-task1-trust.lean, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- make umpire-check-semantic-inventory, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn76-task1-trust.lean, mise exec -- make lint-code GOLANGCI_LINT_FIX=false, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean Umpire/OutcomeClassification/ImportTests.lean, mise exec -- make lint-code GOLANGCI_LINT_FIX=false, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime Umpire.SemanticInventory.Tests.SemanticStages Umpire.SemanticInventory.Tests.KnownGaps Umpire.Planning.Tests.KnownGaps Temporal.Tool.SemanticInventoryTests Umpire.OutcomeClassification.ImportTests Umpire, python3 /tmp/fn76-task1-verify-source.py, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- lake env lean /tmp/fn76-task1-trust.lean, python3 /tmp/fn76-task1-verify-source.py, LEAN_NUM_THREADS=1 TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) mise exec -- make umpire-check-semantic-inventory, python3 /tmp/fn76-task1-compare-trust.py, python3 /tmp/fn76-task1-verify-source.py
- PRs: