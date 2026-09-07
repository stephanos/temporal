# Veil feasibility probe

Date: 2026-09-06. **The pinned semantic core works on Lean 4.33.1; the full checker
is blocked by dependency compatibility and this machine's disk limit.** This is a
successful semantic adapter experiment, not a successful Veil model-checker or SMT run.

## Reproduce the successful probe

From `experiments/umpire-dsl/veil`, run `./run-core.sh`. The script fetches Veil only
if absent, checks its exact revision and verifies the two imported upstream files
have no local diff, then compiles them unchanged. It compiles the actual parent
`DslExperiment/Model.lean` into its own `.lake/core`, without modifying the parent
package's build artifacts. Once the source checkout exists, `lake build` and
`lake exe probe` provide the ordinary package build and executable.

Pin: `verse-lab/veil@be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd`.
The narrow Lake library includes only upstream `ExecutionOutcome` and
`TransitionSystem`, which have no external imports. This deliberately bypasses
upstream's whole-library widget and dependency requirements; it is not evidence
that upstream's full Lake package is compatible. `lake-manifest-full.json` records
the independently resolved complete dependency graph; `lakefile-full.toml` retains
the configuration needed to retry that separate experiment.

## What ran

`Main.lean` imports the actual Veil `EnumerableTransitionSystem` and
`RelationalTransitionSystem`. Its optional adapter wraps the shared model's
successors in Veil execution successes, retaining the correlated operation and
semantic event as the label. The relational alternative derives membership from
the same successors. Neither authors a second cancellation transition table.

Lean checks transition equivalence in both directions, initial-state equivalence,
and one state-safety Boolean/proposition equivalence. All three theorem axiom
inventories are printed: transition and initialization use `propext, Quot.sound`;
safety uses `propext`. This experiment's proof boundary accepts those standard Lean
logical axioms; there are no custom axioms, compiler-trust axioms, or `sorryAx`.
These are adapter facts, not a theorem establishing model-checker completeness,
SMT reconstruction, or compatibility with Umpire's production `TransitionKernel`.
The safety predicate deliberately forbids operation zero's successful completion so
the edit probe can demonstrate a concrete violation; it is not the intended Nexus contract.

The executable compares independently traversed labeled path lists from the
finite reference and the actual Veil enumerable data consumer at depths zero
through five: **1, 2, 8, 24, 84, 216 paths**, respectively. All **335** paths pass
shared-model Exact Replay. This covers both terminal outcomes, two operations,
and labeled self-loops. A completion before request confirmation is rejected.
The full experiment's scoped obligations/evidence tests live in the parent package;
this Veil slice does **not** check a history-sensitive property on plain model state.

The list traversal is experimental code consuming Veil's data API, **not Veil's
breadth-first checker**. Equivalent list and relational views establish this narrow
representation seam; they do not demonstrate a symbolic-first model without
mandatory finite enumeration. No code was removed from production.

## Measurements and failure evidence

Measurements are single local runs under concurrent development, not a benchmark.

| Operation | Observed result |
| --- | --- |
| Compile pinned core, parent model, adapter proofs, run path comparison | 6.57 seconds on the initial proof-only run; subsequent complete path comparison reported 4,389,208 ns inside the Lean interpreter |
| Narrow package `lake build` | 1.37 seconds for the compiled executable; 0.14 seconds warm |
| `lake exe probe` | 0.48 seconds including Lake startup; native path comparison 714,208 ns |
| Full `lake update` with Lean 4.33.1 | Failed after 223.96 seconds: mathlib cache rejects mismatch with pinned Lean 4.32.0 |
| Full checker source build at 4.33.1 | Failed after 27.17 seconds: `Batteries/Tactic/Alias.lean:176:16` and `:208:14` pass a pair `(doc, isVerso)` where 4.33 expects `TSyntax Lean.Parser.Command.docComment`; subsequent tasks also hit disk exhaustion |
| Matching 4.32 toolchain installation | Download failed for insufficient disk; no successful matching-toolchain build |
| Actual Veil BFS, symbolic queries, reconstructed SMT proofs | Not run; no results or timing claimed |

The complete graph pulls mathlib through Loom even though the SMT dependency tag
contains `no-mathlib`. The attempted installation accumulated roughly 950 MB in
this experiment's package directory. Experiment-created transitive checkouts and
build output were removed when disk availability reached 122 MiB, restoring about
1.1 GiB. The pinned Veil source and narrow probe artifacts remain. No production
package or shared dependency checkout was changed.

To retry the full graph on a machine with space, use
`lake -f lakefile-full.toml update` then
`lake -f lakefile-full.toml build Veil.Core.Tools.ModelChecker.Concrete.Checker`.
A supported Lean 4.32 environment or a deliberately updated compatible dependency
graph is needed; merely changing the version string is not a verified solution.
Do not infer that a 4.32 build fails from this resource-limited attempt.

## Decision

Retain the optional adapter as a feasible seam and **defer full backend adoption**.
The next discriminating run must exercise the actual checker on the shared model
plus scenario/obligation product state, compare complete and depth-bound receipts,
recover and replay its counterexample, and separately test reconstructed versus
trusted SMT answers. A Veil-centered DSL requires an additional extraction
correspondence proof. None of those results can be substituted by this core probe.
The shared-core experiment currently measures representation compatibility only;
it supplies no evidence of net migration savings or reduced per-feature work.

## Executed subsequent-edit exercise

`/usr/bin/time -p ./run-edit.sh` passed in **1.34 seconds**; the edited one-step
consumer comparison took **739,834 ns** inside the Lean interpreter on that run.
The runner has no dependency installation or network action. It requires the narrow
core artifacts produced by `run-core.sh`, makes an experiment-owned temporary copy,
and deletes only that temporary directory on exit.

The runner first compiles and executes the unchanged model and adapter, including
the regression that rejects completion before cancellation confirmation. It then
adds exactly one transition branch in the copied `Model.lean`:
`started + completed → succeeded`. The original model is unchanged. The same edited
model is recompiled for both the finite consumer and Veil enumerable consumer;
the copied `Main.lean` adapter is byte-compared to its original after the test.

`EditProbe.lean` proves that the new operation-zero completion edge belongs to the
finite model, belongs to the unchanged Veil adapter through the existing generic
correspondence theorem, passes Exact Replay, and violates the existing safety
clause in one step. It also executes a comparison of the complete one-step lists.
All proof inventories are printed; only `propext` and `Quot.sound` appear, with
no placeholder or compiler-trust axioms.

Measured authored behavior change: **one branch in one model file; zero adapter
source edits; zero safety-clause edits**. The exercise adds two harness files
(`EditProbe.lean`, `run-edit.sh`); their code is experiment machinery, not a claimed
reduction in production maintenance. Existing generic correspondence proofs were
rechecked unchanged. The original baseline regression would need intentional
revision if this behavior were adopted; the edited check is separate so that the
baseline is not silently weakened.

This exposes a real boundary: the runtime evidence projection does not yet admit
unsolicited completion from `started`. Adding the model branch alone therefore
**does not implement the whole pipeline**. No scoped temporal compiler change,
production property migration, full Veil checker query, or runtime correspondence
is claimed by this edit exercise.
