---
satisfies: [R22]
---
# fn-88-veil-concrete-checker-as-the-umpire.11 Second compatibility probe under Veil's declared toolchain

## Description
Run the R22 probe and append its receipt. Copy `model/` to a temporary directory outside the repository, set the copy's `lean-toolchain` to Veil's declared toolchain (4.32.0 at `517f2badbf9a7ba2b18a72242351ff20943cbdd7`, per the R1 receipt; re-check Veil `main` and record the commit), move Batteries, protobuf and binary to the revisions that toolchain needs, add the one `require` for Veil, and make the whole model build and `make umpire-check-goldens` pass in the copy with the fewest model-side changes. Record every fact R22 lists and the decision under the amended adopt conditions (spec Edge Cases). Do not change the real `model/`.

**Size:** M
**Files:** `experiments/umpire-dsl/VEIL_RESULTS.md` (appended receipt section); a throwaway copy of `model/` outside the repository
**Touches:** [experiments/umpire-dsl/VEIL_RESULTS.md]

### Approach
- Record the model-side diff needed on the older toolchain (files and line counts) as a patch in the receipt or task evidence, so task .12 applies it rather than rediscovering it.
- Record the checker entry's shape again and how Umpire can run it during command elaboration (the `IO` monad lift an elaborator offers), and whether Node/npm is a build prerequisite of the closure.
- Measure cold build and peak RSS under quiet load if possible; record concurrent load if not.
- Delete the temporary copy on every exit path.

### Key context
- The R1 receipt (`eefea06d19`) is the baseline: its first error was `Veil/Util/TreeSetMisc.lean:71:30` under 4.33.1.
- A change that alters a Property's meaning, a fingerprint or a golden is `defer-incompatible`.

## Acceptance
- [ ] The receipt names the toolchain, every requirement revision, the model-side diff, whether the checker closure builds unchanged, the R1 facts, and one decision.
- [ ] No file under the real `model/` changes.


## Done summary
Ran the R22 second compatibility probe and appended its receipt to experiments/umpire-dsl/VEIL_RESULTS.md. Decision: adopt. On Lean 4.32.0 with Batteries v4.32.0 (protobuf and binary already declare 4.32.0), Veil 517f2ba's checker closure builds unchanged. In a throwaway copy the whole model builds, goldens are byte-identical, and lint-model passes. findReachable runs from CommandElabM and TermElabM. The model-side diff for task .12 is experiments/umpire-dsl/veil-r22-toolchain.patch: the pin files plus one Fixtures.lean proof line, rebased onto b90b63e072. That commit landed the pre-existing lint-model warning fixes during the probe; they have not yet been built on 4.32.0, and .12's gate run is their first. Follow-ups: .12 moves mise.toml, and .5 names Node/npm as build prerequisites.

stage: impl-review - ran [2026-09-26T23:10..2026-09-26T23:15] SHIP (claude backend, one P3 wrap finding fixed in 5fcd273aad)
## Evidence
- Commits: 4a33beb2e26148cf68cef37fcad9fef50bc430d5, 244138bbf11bafb79dec4bc92aa17328de019fba, 5fcd273aadd1033d1c36706ca940a680eeed70cf
- Tests: baseline: not run - docs-only task; the spec Quick commands name modules that do not exist before task .2 (Umpire.Search.Product) and would build another agent's uncommitted model edits, throwaway copy of model/ at 59ca34bf47 on Lean 4.32.0 with the Veil require: lake build Veil.Core.Tools.ModelChecker.Concrete.Checker (green), lake build (green, 599 jobs, 0 warnings), umpire-goldens --output-root + diff -ru over UMPIRE_GOLDEN_DIRECTORIES (no diff), lint-model steps under LEAN_NUM_THREADS=1 (inventory, lint-tests, both controlled violations, umpire-lint, lake --wfail lint --builtin-only: green on a quiet host), GATE_SKIPPED:build:docs-only - committed range b90b63e072..HEAD touches only experiments/umpire-dsl/VEIL_RESULTS.md and veil-r22-toolchain.patch; gate classify returned FULL only on another agent's uncommitted .plans/UMPIRE4_ORDER.md, GATE_SKIPPED:goldens:docs-only - same range; no model file changed, GATE_SKIPPED:lint-model:docs-only - same range; no model file changed, GATE_SKIPPED:regression:docs-only - same range; no model file changed
- PRs: