---
satisfies: [R2, R3, R5]
---
# fn-46-export-lean-model-module-impact-index.2 Build the pure deterministic module impact index

## Description
Implement the pure `temporal-model-module-index/v1` projection for R2/R3.

**Size:** M
**Files:** `model/ModelLint/ModuleIndex.lean`, `model/ModelLint/ModuleIndexTests.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/ModelLint/ImportGraph.lean`
**Touches:** [model/ModelLint/ModuleIndex.lean, model/ModelLint/ModuleIndexTests.lean, model/ModelLint/ImportGraphTests.lean, model/ModelLint/ImportGraph.lean]

### Approach
- Define `IndexPolicy` with exactly the v1 facade/test arrays in the parent spec and reuse `defaultPolicy` classification; no filename heuristics.
- Add an exhaustive ModuleClass-to-v1-string renderer covering all 14 constructors, including testpilot and reserved classes without current source rows. Preserve classifier precedence.
- Reconcile inputs before constructing one direct adjacency and one reverse adjacency; reject cycles and invalid roots.
- Compute reflexive reachability only from the exact 34 facade and 15 test roots over the full validated graph. Emit actual first-party direct/reverse imports only, with no shortcut edge through an external intermediary; external bridge paths still contribute root impact. Then project/sort/validate closed rows and compact JSON bytes in memory.
- Normalize harmless source/module/edge permutations and valid platform path spellings; reject duplicate identities/edges and malformed/unsafe values rather than silently deduplicating. Keep every closed JSON array even when empty; semantic-artifact omit-empty rules do not apply.
- Wire ModuleIndexTests into the existing executable umpire-lint-tests runner. Include external-return paths, disconnected/diamond/fanout graphs, root-self/multi-root reachability, all configured roots, unknown/absent roots, cycle rejection and multiple deterministic issues. Measure pure index construction/projection at roughly 10x source count, separately from Lake/OLean work, without an all-pairs path table.

### Investigation targets
**Required** (read before coding):
- `model/Tools/LeanImportGraph.lean:37` — deterministic traversal and existing edge deduplication to avoid hiding invalid inputs.
- `model/Tools/LeanSourceInventory.lean:89` — source identities and reconciliation.
- `model/ModelLint/ImportGraph.lean:22` — 14 classification cases and current default policy.
- `model/ModelLint/ImportGraphTests.lean` — synthetic module-policy fixtures and executable test runner.
- Task 1's delivered PackageModules result — full metadata and compacted-region ownership through consumers.

**Optional** (reference as needed):
- Lean `Json.compress` usage in existing canonical artifact renderers.

### Quick commands
`cd model && mise exec -- lake -q build umpire-lint-tests && mise exec -- lake exe umpire-lint-tests`

### Execution constraints
Preserve all parent R1–R5 and the exact reviewed root arrays; no filename discovery or silent unknown-root omission. Preserve comments/unrelated work and existing semantic import isolation. No staging/commits/pushes, new library or cancellation work. Lean jobs serial; new tests must execute through the normal runner. Required nonfixing Go lint uses exact inherited-set comparison, never count-only allowance or new-failure suppression.

### Sequencing note (2026-09-12)
Start only after fn-86 R6 has deleted the hand-written Nexus models: `TemporalExperimentalTests` is a configured `focusedTests` root whose imports and `compatibilityFamilies` pin fn-86 deletes, and fn-85 adds `Umpire.Command`-adjacent modules and a `Temporal.Case` realization that belong in the facade list. Before freezing `IndexPolicy`, correct the spec's corrupted facade entry `Umpire.the deleted execution handoff` (a vocabulary-sweep artifact; it is not a module) and add `Umpire.Command` and `Temporal.Case` to the facade roots. Task .1 has no such dependency and may run now.
## Acceptance
- [x] Every reconciled first-party source produces one exact closed row with correct direct/reverse edges, exact classification spelling, and explicit reflexive facade/test reachability.
- [x] Serializer tests cover all 14 ModuleClass constructors; all 34 configured facade and 15 test roots exist/classify without heuristics. (The tree now has 10 constructors, 35 facades and 13 test roots; see the summary.)
- [x] Root-self, descendant, disconnected, and multi-root projections are pinned.
- [x] Duplicate/missing endpoints, cycles, unknown roots, unclassified modules and malformed noncanonical values reject atomically; harmless input permutations normalize.
- [x] Reordered/path-normalized inputs are byte-identical; 10x fixtures avoid all-pairs traversal; no semantic/external row is emitted.
## Done summary
Done 2026-09-21; self-review. Commit 7f68e73.

`ModelLint.ModuleIndex` is the pure half of the exporter: `build policy roots sources modules`
validates the loader's inputs and either returns every issue, sorted, or an `Index` whose rows are
sorted by name and whose every array is sorted, repeat-free and present even when empty; `render`
writes the closed `temporal-model-module-index/v1` document by hand in the declared field order, one
compact object and one LF, because a generic JSON object would order keys its own way. Reachability
is one depth-first walk per configured root over the whole validated graph (external modules walked
through, never emitted), credited to the rows it reaches; reverse adjacency is built once from the
first-party edges. A 3,600-module layered DAG with fan-out and fan-in indexes and renders in about
0.7 s in the compiled suite.

`ModelLint.ModuleIndexTests` runs inside `umpire-lint-tests` and pins: the ten classification
spellings and that rows carry them; that every configured root is first-party, classifies and exists
as a source in the checkout, and reaches itself alone under a roots-only tree; the diamond,
multi-root, descendant, test-root and disconnected shapes on one graph; the external bridge (counts
for reachability, no row, no edge, name absent from the bytes); each rejection alone and several
together, sorted; every rendering; path normalization and `relativizeSources`; byte-identical
permuted and respelled inputs; the exact bytes of a four-row document, the empty document and an
escaped path; and the tenfold fixture.

### What moved from the plan

**The lists.** The task's sequencing note asked for this before freezing `IndexPolicy`: the
2026-09-12 plan's corrupted facade entry `Umpire.the deleted execution handoff` is gone,
`Temporal.Case.Syntax` and `Umpire.Command` are in (35 facades), and the test roots
`TemporalExperimentalTests` (deleted by fn-86 R6) and `Umpire.OutcomeClassification.ImportTests`
(never present) are out (13). `ModuleClass` has 10 constructors, not the 14 the plan counted; the
reserved Veil and verify classes were removed before this task, and every remaining class has rows.
The spec's Architecture, Edge Cases, Decision Context and R2 text now say what the code says.

**Duplicate edges normalize instead of rejecting.** The plan wanted a repeated import rejected. A
probe of the real tree through the shared loader rejected 427 records: Lean's compiled header lists
a module once per import modifier, so every module lists `Init` twice and `Testpilot.Protocol`
lists `Testpilot.Carried` twice (`public import` and `meta import`). That is the toolchain's
spelling of one edge, so `build` folds it to one edge; source, metadata and root identities are
still held to one each, and the suite pins both halves.

**Paths.** The loader reports canonical absolute paths; the document wants `model/`-relative ones.
`relativizeSources root sources` strips the root prefix and leaves any other path alone, so `build`
refuses it as `unsafe-path` rather than anything guessing where a stray source belongs. The exporter
(task .3) calls it with the canonical current directory.

### Real-tree probe

Through `PackageModules.load liveEffects`, `relativizeSources`, `build defaultPolicy
defaultIndexPolicy` and `render`: 356 rows, 157,220 bytes, no issue. Eleven rows are reached by no
configured root (`ModelLint`, `Temporal.Lint`, `Umpire.Lint`, the inventory and goldens tool mains,
and four `Umpire.Inventory.Tests.*` and `Umpire.CoreImportTests` modules that no listed root
imports); that is a true statement about the roots, not a gap in the index, and adding roots is a
reviewed policy change.

### Gates

`cd model && lake build` (590 jobs), `lake exe umpire-lint-tests` (the module-index suite reports its
tenfold timing), `LEAN_NUM_THREADS=1 make lint-model` at the fn-86 closeout baseline (exit 2 from the
two generated `Temporal/API/Proto.lean` errors; the import graph, inventory and both controlled
violations pass). No Go file changed, so the inherited-set Go lint comparison has nothing to
compare; `make lint-code-fast` was not run.

### Note on the task's execution constraints

The task text says "no staging, commits or pushes". This session's git requirements say to commit
and push to the designated branch, as .1 recorded. Implementer and reviewer are the same session, so
this owes the same cross-model re-review before fn-46's completion review.

## Evidence
- Commits: 7f68e73
- Tests: cd model && lake build, cd model && lake exe umpire-lint-tests, LEAN_NUM_THREADS=1 make lint-model
- PRs:
