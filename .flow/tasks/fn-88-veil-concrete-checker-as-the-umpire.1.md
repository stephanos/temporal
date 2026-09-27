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
Ran the R1 compatibility probe and appended its receipt to experiments/umpire-dsl/VEIL_RESULTS.md. The decision is `defer-incompatible`: Veil main 517f2ba (declared toolchain v4.32.0) compiles Batteries v4.33.0 and Aesop, but Veil.Util.TreeSetMisc fails under Lean 4.33.1 at `:71:30` (`clear` failed: target depends on 'l'). The checker entry is also IO rather than pure, so a toolchain fix alone would give `defer-closure`, not `adopt`. The throwaway copy was deleted, and the real model/ lakefile and manifest are unchanged.

stage: impl-review - ran [2026-09-26..2026-09-26] triage_skip SHIP (docs-only)
## Evidence
- Commits: eefea06d1961ff53d3542d087aff7072e3dc59f8
- Tests: baseline: none (the spec's Quick commands build modules that later fn-88 tasks introduce; the shared model/.lake was being rebuilt by another agent), probe: lake update (cold, 120.99s, rc=0) in a throwaway model copy with require veil@517f2badbf9a7ba2b18a72242351ff20943cbdd7, probe: lake build Veil.Core.Tools.ModelChecker.Concrete.Checker (cold 117.92s, rc=1; warm 16.16s, rc=1; first error Veil/Util/TreeSetMisc.lean:71:30), pin=veil@517f2badbf9a7ba2b18a72242351ff20943cbdd7; veil-toolchain=leanprover/lean4:v4.32.0; batteries-resolved=4488d40d070b9700d4d5a6aa342f0d40c31b2a2d (v4.33.0); checker-entry=IO (findReachable, unfueled while loop, IO.CancelToken), frontier=deterministic FIFO, single-threaded when parallelCfg=none, UInt64 fingerprint dedup; expose=TransitionSystem/ExecutionOutcome/Trace/Interface/Core are @[expose], Checker/Sequential are not; lint-model-complete=not measured (build failed; disk 1.5-2.4 GiB free); decision=defer-incompatible, gate classify --base b9bb1a58adf931e0830bcb5917849ae4f1ea894f: FULL because the dirty worktree holds another agent's uncommitted Makefile edit; the committed diff b9bb1a58adf931e0830bcb5917849ae4f1ea894f..HEAD touches only experiments/umpire-dsl/VEIL_RESULTS.md, GATE_SKIPPED:umpire-check-goldens:docs-only - cumulative committed diff is one markdown file (no executable paths touched), GATE_SKIPPED:lint-model:docs-only - cumulative committed diff is one markdown file (no executable paths touched), GATE_SKIPPED:umpire-check-regression:docs-only - cumulative committed diff is one markdown file (no executable paths touched), git status: no change under model/lakefile.lean or model/lake-manifest.json; the only model/ modifications are another agent's pre-existing edits
- PRs: