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

## fn-88 R1 probe: concrete checker inside the model's dependency graph

Date: 2026-09-26 (local; the Lake log's commit timestamps are UTC 2026-09-27). Task
`fn-88-veil-concrete-checker-as-the-umpire.1`. **Decision: `defer-incompatible`.** The checker
closure does not compile under `model/lean-toolchain` (Lean 4.33.1). The first error is:

```text
error: Veil/Util/TreeSetMisc.lean:71:30: Tactic `clear` failed: target depends on 'l'
```

The decisive condition is the first `adopt` condition, "the checker closure builds unchanged under
`model/lean-toolchain`". The failure is a compiler error, not a network or disk error, so it is a
decision under R1 and not a retry. The other `adopt` conditions were checked from source and are
recorded below. The checker entry would also have failed the purity condition, so a toolchain
fix alone would move the decision to `defer-closure`, not to `adopt`.

### Setup

- The copy was taken with `git archive HEAD model` plus the nine testpilot `.proto` inputs and
  `proto/api.binpb`, which the copy's `lakefile.lean` reads through `../proto`. It lived in a
  session scratch directory outside the repository and was built cold, without the checkout's
  `.lake`. It was deleted after the run.
- The only edit was one `require` in the copy's `lakefile.lean`, placed after `protobuf`:
  `require veil from git "https://github.com/verse-lab/veil.git"@"517f2badbf9a7ba2b18a72242351ff20943cbdd7"`.
- Build command: `lake build Veil.Core.Tools.ModelChecker.Concrete.Checker`, run with the
  Makefile's macOS `SDKROOT`/`CC` environment under `mise exec`.

### Receipt

| Field | Value |
| --- | --- |
| Commit | `verse-lab/veil@517f2badbf9a7ba2b18a72242351ff20943cbdd7` (`main` HEAD on the probe date, 2026-09-23 "chore: remove `Veil.Core` (#72)"). `main` was used because it keeps every checker module under `Veil/Core/Tools/ModelChecker/Concrete/`; #72 removed the separate `Veil.Core` Lake library, not the module paths. The `be6a1ce` fallback was not needed. |
| Veil's declared toolchain | `leanprover/lean4:v4.32.0`. The model builds with `v4.33.1`. |
| Batteries resolved | `4488d40d070b9700d4d5a6aa342f0d40c31b2a2d` (`v4.33.0`). The root requirement shadows Veil's `v4.32.0` pin (`023ce7d6…`). All 192 Batteries modules in the closure built, so the 2026-09-06 failure at `Batteries/Tactic/Alias.lean:176` does not recur. |
| Packages `lake update` added | Veil's packages at its manifest revisions: proofwidgets `6e311e2a`, aesop `a7dbf0c6` (`v4.32.0`), Loom `27c03ba8`, smt `922af463` (`v4.32.0-veil-no-mathlib`), Qq `38d591e7`, cvc5 `a3ffc29a`, auto `1175ff6b`. Mathlib is gone from Veil's graph since #69. protobuf and batteries keep the model's revisions. |
| Imported closure | 355 non-core modules: 31 Veil, 192 Batteries, 132 Aesop (Aesop through `Veil.Util.Equiv`), plus `Init`, `Lean`, and `Std`. The 31 Veil modules are the 14 `Veil.Core.Tools.ModelChecker.{Concrete.*, ExecutionOutcome, Interface, Trace, TransitionSystem}` modules, `Veil.Frontend.DSL.Module.Names`, `Veil.Frontend.DSL.State.Types`, and 15 `Veil.Util.*` modules. The closure was read from source import headers and matches the build, which scheduled only Batteries, Aesop, and Veil modules. |
| Loom, lean-smt, mathlib, widget modules in the closure | None as Lean imports. Two caveats. `lake update` still clones Loom, smt, cvc5, auto, Qq, and proofwidgets. The `Veil` library declares `needs := #[widgetJsAll]`, so building any Veil module runs `npm clean-install` and a rollup build of `widget/` first (`veil/widgetJsAll`, 7.5 s, 88 MB of `node_modules`). Node and npm become build prerequisites of the model. |
| `lake update` | 120.99 s wall, cold. This includes a first-time elan download of the 4.33.1 toolchain into `~/.elan`, which the timing cannot separate. `.lake/packages` was 80,288 KiB (78 MiB) after the update and 476 MiB after the build, mostly Batteries and Aesop build output plus the widget's `node_modules`. |
| Cold build of the checker closure | Failed after 117.92 s wall (319.80 s user). It built 344 of 366 jobs: all Batteries and Aesop modules and 19 Veil modules. Two modules failed: `Veil.Util.TreeSetMisc` (the first error above) and `Veil.Frontend.DSL.State.Types` (`:127:74` `apply` failed to unify `ih`, `:213:72` `rfl` failed, `:229:8` unsolved goals, `:233:4` `simp` made no progress). The failing proofs are over core `Std.TreeSet`/`DTreeMap.Internal` and `IteratedProd` lemmas, so the cause is 4.32 to 4.33 core drift. `Veil.Core.Tools.ModelChecker.Concrete.Checker` itself was never reached. |
| Warm build | 16.16 s wall. It rebuilt nothing that had succeeded, retried the two failing modules, and failed with the same errors. |
| Peak RSS | 922,058,752 bytes (cold) and 869,449,728 bytes (warm): `maximum resident set size` from `/usr/bin/time -l`, which is the largest single process in the tree. |
| Complete-mode `lint-model` in the copy | Not measured, for two reasons. (1) The build failed, so no model module can import Veil. The lint's metadata walk (`Tools/LeanImportGraph/Metadata.lean`) reads `.olean` data only for modules reachable from the model's roots, so a copy with only the `require` would measure today's lint and nothing of Veil. (2) The volume had 1.5 to 2.4 GiB free during the run, and a cold model build takes about 2.7 GiB of `.lake`. Other agents' builds were using the same disk, and filling it would have broken them. |
| Checker entry purity | Monadic `IO`. The entry `Veil.ModelChecker.Concrete.findReachable` runs in `m` with `[MonadLiftT IO m]` and takes an `IO.CancelToken`. Its sequential driver `breadthFirstSearchSequential` is an unfueled `while` loop in the same monad that also publishes progress. Trace recovery (`recoverTrace`) is monadic and throws through `IO.ofExcept`, and its helper `retraceSteps` is `partial`. The one-step function `SequentialSearchContext.bfsStep` is pure and total, and invariant-preservation theorems cover it. An adapter could drive it with fuel only by owning the loop and the trace recovery. |
| Frontier order and threading | Deterministic and single-threaded when `parallelCfg = none`. The frontier is `fQueue`, a FIFO over two lists. Successors are processed with `List.foldl` in `sys.tr` order. The parent log is a `Std.HashMap` used only for lookup and first-insert (`log.insert` runs only when the fingerprint is absent), so parents are first-discovery parents and hash-map iteration never orders the search. With `parallelCfg = some _` the search uses `breadthFirstSearchParallel`, which spawns `IO.asTask` shards (MapReduce). One fact for R15 and R10: `findReachable` fixes the fingerprint type to `UInt64`, with default view `hash`, so two distinct states with equal 64-bit hashes are merged. That is hash compaction, not a visited set of exact states. The lower-level functions are generic in the fingerprint type. |
| `@[expose]` facts | `TransitionSystem.lean` (`EnumerableTransitionSystem`, `next`, `toRelational`, `reachable`, `reachable_equiv_relational`), `ExecutionOutcome.lean`, `Trace.lean`, `Interface.lean` (`ModelCheckingResult`, `TerminationReason`, `SearchParameters`), `Concrete/Core.lean`, `Concrete/Containers.lean`, and `Concrete/Subtypes.lean` are `@[expose] public section`. The definitions R7 unfolds are therefore exposed. `Concrete/Checker.lean`, `Sequential.lean`, `SearchContext.lean`, `MapReduce.lean`, and `Progress.lean` are plain `public section`, so the bodies of `findReachable`, `breadthFirstSearchSequential`, `bfsStep`, and `recoverTrace` are not exposed to importers. |
| Bounds the checker supports | A depth bound (`EarlyTerminationCondition.reachedDepthBound`), stop-at-first-violation, deadlock, and assertion stops. It has no visited-state-count bound, so `Limits.search` counted in states would need adapter-side enforcement. |

Concurrent load: the host is a macOS arm64 machine with 8 cores and 16 GiB of RAM. Another
agent's `lake build` and a separate clone's `make umpire-build-model` ran throughout. Load
averages were 79 to 83 (1 min) during `lake update` and 19 to 30 during the builds. All wall
times are contended single runs, not baselines. No threshold decision depends on them: the
decision rests on a deterministic compiler error.

### Consequence

Under the spec's precedence, `defer-incompatible` closes every later fn-88 task as not
applicable, and each cites this receipt (R13). The real `model/lakefile.lean` and
`lake-manifest.json` were not touched. The next attempt needs either a Veil revision that declares
a Lean toolchain at or above the model's, or a model toolchain that Veil supports. Such an attempt
should re-check the purity condition too, because the entry is `IO` whatever the toolchain.

## fn-88 R22 probe: concrete checker under Veil's declared toolchain

Date: 2026-09-26 (local). Task `fn-88-veil-concrete-checker-as-the-umpire.11`. **Decision: `adopt`**
under the amended adopt conditions (spec Edge Cases, 2026-09-27). On Lean 4.32.0 the checker
closure builds unchanged, the whole model builds, and `umpire-check-goldens` passes byte-identical.
`lint-model`'s steps pass in the copy with Veil required. On today's `HEAD` the move needs one
changed proof line plus the three pin files. None of it alters a Property, a fingerprint, or a
golden.

### Setup

- The copy was taken the way R1 took it: `git archive HEAD model` plus the nine testpilot `.proto`
  inputs and the checkout's `proto/api.binpb`, at `59ca34bf47`. It lived in a session scratch
  directory outside the repository, was built cold without the checkout's `.lake`, and was deleted
  after the run. The real `model/` was not touched.
- Lean 4.32.0 was installed with `elan toolchain install leanprover/lean4:v4.32.0` (39 s). Lake ran
  from that toolchain's `bin` with the Makefile's macOS `SDKROOT`/`CC`/`CXX`, the pinned `protoc`
  29.5 (`Testpilot.Protocol` shells out to it), and Node 24.14.1 on `PATH`.
- Edits to the copy: `lean-toolchain` set to `leanprover/lean4:v4.32.0`; Batteries moved from
  `v4.33.0` to `v4.32.0`; one `require veil from git
  "https://github.com/verse-lab/veil.git"@"517f2badbf9a7ba2b18a72242351ff20943cbdd7"` added
  after `protobuf`; then the model-side changes below.

### Receipt

| Field | Value |
| --- | --- |
| Commit | `verse-lab/veil@517f2badbf9a7ba2b18a72242351ff20943cbdd7`. Re-checked `main` with `git ls-remote` on the probe date: still HEAD, the R1 commit. |
| Toolchain | `leanprover/lean4:v4.32.0` (Lean 4.32.0, commit `8c9756b2`), Veil's declared toolchain. |
| Requirement revisions | Batteries `v4.32.0` = `023ce7d62a0531e22a5331e20b587817a80d49ff`, the same revision Veil pins. protobuf stays at `406da521c0ebb47207be28e3d9ef738de95a4dd3` and binary at `c1adb7380ea3a538cd800bc5974a1fa05d8b488e`: both already declare `leanprover/lean4:v4.32.0` in their own `lean-toolchain` at those revisions, so neither moves. Veil's graph resolves as in R1: proofwidgets `6e311e2a`, aesop `a7dbf0c6` (`v4.32.0`), Loom `27c03ba8`, smt `922af463`, Qq `38d591e7`, cvc5 `a3ffc29a`, auto `1175ff6b`. Batteries no longer shadows Veil's pin. |
| Checker closure builds unchanged | Yes. `lake build Veil.Core.Tools.ModelChecker.Concrete.Checker` completed (362 jobs) with no Veil source edit. R1's first error (`Veil/Util/TreeSetMisc.lean:71:30`) and the `Veil.Frontend.DSL.State.Types` failures do not occur. |
| Imported closure | Unchanged from R1: 31 Veil modules, 132 Aesop, Batteries, `Init`/`Lean`/`Std`. Batteries arrived as a prebuilt Lake cache archive (`build.barrel`, 97 MB) instead of a source build; Aesop and Veil were compiled. Veil's own `preferReleaseBuild` fetch failed ("building from source; failed to fetch GitHub release") and fell back to source. |
| Loom, lean-smt, mathlib, widget modules in the closure | None as Lean imports, as in R1. `lake update` still clones Loom, smt, cvc5, auto, Qq, and proofwidgets. `veil/widgetJsAll` still runs `npm` and a rollup build (5.2 s, 88 MiB of `node_modules`), so **Node and npm are build prerequisites** of any model that requires Veil. `mise.toml` pins no Node today. |
| Model-side changes on 4.32.0 | Against `HEAD` after `b90b63e072`: 4 files, 5 inserted and 5 deleted lines, saved as [`veil-r22-toolchain.patch`](veil-r22-toolchain.patch) (applies with `git apply`). Counts are inserted/deleted lines. Pin files: `model/lean-toolchain` (1/1), `model/lakefile.lean` (1/1, the Batteries tag), `model/lake-manifest.json` (2/2, the Batteries `rev` and `inputRev`). Lean: `model/Umpire/Search/Tests/Fixtures.lean` (1/1). The patch omits the Veil `require` and its manifest entries, which belong to task .5. Outside `model/`, task .12 must also move `mise.toml` (`"github:leanprover/lean4" = "4.32.0"`); CI installs Lean through `jdx/mise-action` from that file, and `.github/workflows/umpire.yml` pins no Lean version itself. |
| Why the Lean change | `Fixtures.lean:316`: 4.32's `simp` also rewrites `some a = some b` to `a = b`, so `cases Option.some.inj evidenceEq` becomes `subst evidenceEq`. Without it the module fails to compile. This is the only change the toolchain itself forced. |
| Pre-existing `lint-model` failures | The copy was taken at `59ca34bf47`, and its builtin lint (`lake --wfail lint`) failed on 24 warnings in `Syntax.lean`, `Encoding.lean`, and three Nexus files. The warnings were deprecated `levelZero`/`String.trim` calls, an unused `doc?` binding in the `machine` command, a possibly looping `simp` call, and unused-variable reports on `enum` constructor fields. They are not 4.32-specific: the same step failed on 4.33.1, and `b90b63e072` (committed on the checkout during this probe) fixed them on 4.33.1. The copy carried an equivalent local fix. It used an `@[unused_variables_ignore_fn]` for the `enum` syntax kind, the same `rw [encodeNat]` rewrite, `Level.zero`, and `trimAscii.toString`, and dropped the `doc?` binding where `b90b63e072` attaches it. With that fix the builtin lint passed on 4.32.0. `b90b63e072`'s own versions of those edits have not been built on 4.32.0. `Syntax.lean` there does not import `Lean.Linter.UnusedVariables` explicitly, which the copy did. Task .12's gate run is their first 4.32.0 build. |
| Properties, fingerprints, goldens | Unchanged. `umpire-goldens --output-root` compared with `diff -ru` against the four `UMPIRE_GOLDEN_DIRECTORIES` produced no difference, both before and after the Lean changes. No Property, Scenario, Query, or model table is edited by the patch. |
| Whole-model build | `lake build` of every default target completed (599 jobs) with zero warnings once the proof change and the local lint fix were in. |
| `lake update` | 30.89 s wall, cold, with the 4.32.0 toolchain already installed. `.lake/packages` was 81,184 KiB (79 MiB) after the update and 1,232,156 KiB (1.2 GiB) after the model build. The copy's whole `.lake` was 2.3 GiB after the model build. |
| Cold build of the checker closure | 73.52 s wall (161.83 s user), 362 jobs. |
| Warm build of the checker closure | 1.21 s wall. |
| Cold build of the whole model | 1,803 s wall in two passes: 278.48 s until the first pass stopped on the `Fixtures.lean` error and a missing `protoc` on `PATH`, then 1,524.93 s. The long poles were `Temporal.Feature.Nexus.Caller.Tests` (635 s), `Testpilot.Carried` (159 s), and `Temporal.API.Types` (108 s). Rebuilding after the `Syntax.lean` change took 1,003.48 s. |
| Peak RSS | 917,798,912 bytes for the checker closure build and 3,348,480,000 bytes for the model build: `maximum resident set size` from `/usr/bin/time -l`, the largest single process in the tree. |
| `lint-model` in the copy | Passes with Veil required, run as the Makefile's steps under `LEAN_NUM_THREADS=1`. `umpire-check-inventory` matched. `umpire-lint-tests`, both controlled violations, and `umpire-lint` passed; `umpire-lint`, the complete-mode import-graph walk, took 66 s. The `lake --wfail lint --builtin-only --lint-only=.all,.extra,-.missingDocs` step first failed its build phase on the pre-existing warnings above (2,094 s single-threaded). With the local fix, the build phase completed all 409 jobs with no warning. The lint pass after it was killed with `SIGKILL` three times while another agent's `lint-model` ran on the checkout, once with the Veil `require` removed. On a quiet host it passed: 66.66 s wall, 0 warnings, 6,020,644,864 bytes peak RSS for the `lake` process, and a 19.6 GB peak memory footprint (compressed). A sampled 4.33.1 run of the same step on the checkout reached 5.09 GB RSS, so most of the memory belongs to the step itself, not to Veil or to 4.32.0. |
| Checker entry purity | Unchanged from R1: `findReachable` is monadic, `[MonadLiftT BaseIO m] [MonadLiftT IO m]`, takes an `IO.CancelToken`, and drives an unfueled `while` loop in `breadthFirstSearchSequential`; `retraceSteps` is `partial`. |
| Running it during command elaboration | Shown. A throwaway file importing only `Lean` and `Veil.Core.Tools.ModelChecker.Concrete.Checker` defined a five-state counter `EnumerableTransitionSystem` with one `tick` action. It called `findReachable (m := IO)` with `parallelCfg := none` from an `elab … : command` (`CommandElabM`) and from `#eval` in `TermElabM`. Both lift `IO` implicitly. With the invariant `n < 3` it returned `foundViolation … safetyFailure [belowThree]` with a trace from state 0 of 3 steps; with `True` it returned `noViolationFound` after exploring 5 states. Elaboration took 2.12 s wall and printed no progress output. A second throwaway file importing `Umpire`, `Temporal`, and the checker together elaborated with no name clash. |
| Frontier order and threading | Unchanged from R1: a deterministic single-threaded FIFO with first-discovery parents when `parallelCfg = none`, and `IO.asTask` shards otherwise. `findReachable` fixes the fingerprint to `UInt64` (`hash`), so distinct states with equal 64-bit hashes merge. Under the amended conditions this is the trust assumption of a `veil` absence answer, which `AUTHORING.md` must state. |
| `@[expose]` facts | Unchanged from R1, re-read at the same commit: `TransitionSystem`, `ExecutionOutcome`, `Trace`, `Interface`, `Concrete/Core`, `Concrete/Containers`, and `Concrete/Subtypes` are `@[expose] public section`, so the definitions R7 unfolds are exposed. `Checker`, `Sequential`, `SearchContext`, `MapReduce`, and `Progress` are not. |
| Bounds the checker supports | Unchanged from R1: a depth bound, stop at first violation, deadlock, and assertion stops. There is no state-count bound. |

Concurrent load: the host is a macOS arm64 machine with 8 cores and 16 GiB of RAM. Other agents'
`lake` builds and a `lint-model` run shared it. One-minute load averages were 13 during the checker
build, 8 to 18 during the model build, 6 to 13 during the first lint run, and 45 to 68 while the
builtin lint was being killed, and 3 when it passed. Free disk fell from 15 GiB to 6 GiB. All times are contended single
runs, not baselines, and no decision depends on them.

### Decision

Each amended `adopt` condition holds. The closure builds unchanged under Veil's declared toolchain.
It imports no Loom, lean-smt, mathlib, or widget module. The `IO` entry runs from a command
elaborator. The definitions R7 unfolds are exposed. The R22 `defer-incompatible` clause does not
apply: no model file needed a change to a Property, a fingerprint, or a golden. Three
obligations carry forward:

- Task .12 applies `veil-r22-toolchain.patch`, moves `mise.toml`, and re-runs every gate on the
  checkout. That run is the first 4.32.0 build of `b90b63e072`'s lint fixes. The builtin lint needs
  a host with no concurrent `lint-model`.
- Task .5 adds the `require` and names Node and npm as build prerequisites.
- Task .6 wraps every `veil` witness in the R10 replay gate, and `AUTHORING.md` states the 64-bit
  hash-compaction assumption.
