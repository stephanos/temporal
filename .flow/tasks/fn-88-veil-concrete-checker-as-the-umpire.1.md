---
satisfies: [R1]
---
# fn-88-veil-concrete-checker-as-the-umpire.1 Compatibility probe of Veil's concrete checker inside the model's dependency graph

## Description
Run the R1 probe and write its receipt. Copy `model/` to a temporary directory outside the repository, add one `require` for `verse-lab/veil` at a pinned commit to the copy's `lakefile.lean`, and build `Veil.Core.Tools.ModelChecker.Concrete.Checker` plus its transitive imports under the repository toolchain. Record every fact R1 lists and the decision `adopt`, `defer-closure`, or `defer-incompatible`. Every later task reads this decision; none reruns the probe.

**Size:** M
**Files:** `experiments/umpire-dsl/VEIL_RESULTS.md` (appended receipt section); a throwaway copy of `model/` outside the repository
**Touches:** [experiments/umpire-dsl/VEIL_RESULTS.md]

### Approach
- Pick the commit: start from Veil `main` on the probe date; fall back to `be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd` (the 2026-09-06 pin) if `main` has renamed the checker modules. Record which and why.
- Run inside a copy of the model project, not the narrow `lean_lib` the old probe used, so Lake resolves Batteries `v4.33.0` against Veil's pin; the old full-graph failure was in Batteries `Alias.lean:176`.
- Measure `lake update` wall time and `.lake/packages` bytes, cold and warm `lake build` of the checker closure, peak RSS (`/usr/bin/time -l`), and the complete-mode `lint-model` run in the copy, since the semantic-root rule already loads external metadata and will now traverse Veil's.
- Inspect the closure: list every module the checker imports transitively and grep for `Loom`, `Smt`, `Mathlib`, `ProofWidgets`. Inspect the checker entry for `IO`, `partial`, or pure with fuel; inspect `EnumerableTransitionSystem.toRelational` and the checker's result definitions for `@[expose]` (Veil moved to the Lean module system on 2026-09-22). Inspect the frontier data structure and any `Task` or parallel use.
- Apply the `adopt` conditions from the spec's Edge Cases exactly; a build success that fails any other condition is `defer-closure`.
- Delete the temporary copy on every exit path.

### Investigation targets
**Required** (read before running):
- `experiments/umpire-dsl/veil/run-core.sh` and `lakefile-full.toml` — the previous probe's pin, fetch discipline, and failure record
- `experiments/umpire-dsl/VEIL_RESULTS.md` — receipt shape to extend
- `model/lakefile.lean:9-12` and `model/lake-manifest.json` — the requires the copy must keep
- `.flow/specs/fn-23-veil-toolchain-compatibility-and.md` "Decision precedence" — receipt vocabulary to stay compatible with

**Optional:**
- `.plans/VEIL_BACKEND_RESEARCH.md` "What Veil actually exposes" — module names as of the be6a1ce inspection

### Key context
- Do not add the `require` to the real `model/lakefile.lean` here; that is task .5 and only in `adopt` mode.
- A network or disk failure is retried and never becomes a decision.

## Acceptance
- [ ] Receipt section appended to `experiments/umpire-dsl/VEIL_RESULTS.md` with every R1 field filled or marked not-measurable with a reason
- [ ] Decision is exactly one of `adopt`, `defer-closure`, `defer-incompatible`, naming the condition that decided it
- [ ] Task evidence records the pinned commit, Veil's declared toolchain, the Batteries revision resolved, the checker entry's purity and threading, the `@[expose]` facts, and the complete-mode lint time
- [ ] The real `model/` tree, `lakefile.lean`, and `lake-manifest.json` are unchanged (`git status` shows no change under `model/`)
- [ ] Temporary copy removed

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
