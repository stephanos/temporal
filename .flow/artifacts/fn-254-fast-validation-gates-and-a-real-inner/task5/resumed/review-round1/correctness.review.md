**Review of fn-254.5 at fdf115db99 (correctness axis)**

I read the task file, the decision document, every source hunk in the range, and the current state of the touched files. The large artifact tree under `.flow/artifacts/.../task5/resumed/` was used only as supporting evidence.

**What I verified against the code**

- **Selected-fixture runner** (`model/umpire/dev/testing/src/runner.rs:117`). Admission of registry equality, external artifacts, pending goldens, owner uniqueness, skipped-consumer golden completeness and snapshot well-formedness all run for every fixture before the selection filter. Only the selected fixture renders, and it renders twice. The orphan check is scoped to the selected fixture's files plus permitted skipped/external artifacts, so sibling goldens are never reported as orphans. The `None` path reproduces the old umbrella semantics exactly. The four new consumer controls cover each refusal class.
- **24-case generation** (`model/umpire/dev/fixtures/build.rs:29`). Cases are generated from the same `MODEL_AND_TESTS` discovery that produces `registered()`, so names always match the registry and the JVM group filter on `binary(=fixtures)` still covers all of them.
- **Parser harness** (`model/umpire/author/dsl/macros/tests/parser_refusals/mod.rs`). `proc-macro2` carries `span-locations` at the workspace level, so line/column are real. Column is 0-based from `proc_macro2` and the harness adds one. I cross-checked all 65 `error:` and `-->` lines across the 61 moved `.stderr` files against the table rows: every message, line and column matches. No moved stderr contains a rustc-coded error, so none depended on type checking. The machine path calls `machine::expand`, which skips only the infallible `lifted::embed` step that production inserts between generate and finish, so no diagnostic is lost.
- **Retained UI list**. The 18 explicit `compile_fail` paths equal the full current contents of `tests/ui/`.
- **Concurrency from the JVM group raise**. `PinnedJar::admit` reads the jar read-only and `Scratch::new_in` uses a unique `tempfile` directory, so two simultaneous owners in one invocation do not collide.
- **Thread policy plumbing**. `configure` sets the `required` profile and 4 threads, or 1 under explicit Apalache, on gate, update, direct `test-run` and standalone child paths. Nextest applies default-profile overrides to custom profiles, so the `jvm` group still binds under `required`. The standalone receipt now reports the computed value instead of a hardcoded 2.
- **Dead targets**. No active reference to `model-check-changed` or `model-compiler-inputs` remains outside the frozen `model/archived/` tree and the module map's description of that archived tree.
- **Temporal trybuild split**. Splitting pass and compile_fail into separate `TestCases` is safe under nextest's per-process execution and trybuild's directory lock.

**Findings**

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/author/dsl/macros/tests/fixtures.rs:41`
- **R-IDs**: [R5]
- **Problem**: The retained cases are listed explicitly, but after the move `tests/ui/*.rs` matches exactly those 18 files. A future case dropped into `tests/ui/` is silently never compiled, which is a coverage trap the old glob did not have.
- **Suggestion**: Keep the single `compile_fail("tests/ui/*.rs")` glob and let `MAPPING.md` carry the rationale list.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/testing/src/runner.rs:171`
- **R-IDs**: []
- **Problem**: Admission now calls `validate_text_snapshot` on every existing golden unconditionally. `snapshot()` skips that validation under `INSTA_UPDATE=always`, so `xtask update` previously overwrote a corrupt golden and now refuses with "invalid golden" until the file is deleted by hand. Narrow edge, but a behaviour change on the update route.
- **Suggestion**: Either accept and document it, or skip the pre-validation of selected-fixture goldens when `INSTA_UPDATE=always`.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/gate.rs:13`
- **R-IDs**: []
- **Problem**: `test_run::run` applies `configure` and then `stage` applies it again for `name == "tests"`. `stage` has no other `"tests"` caller, so one of the two is dead weight.
- **Suggestion**: Remove the branch in `stage` and keep the call in `test_run::run`, or the reverse.

FYI, not findings: the explicit-Apalache cap to one slot and the hard cap on the discovery probe heap follow the recorded decision. `run_fixture` recomputes `admitted(&root)` per case under required backends, which lifts every fixture 24 times instead of once. That is CPU-only and matches the "complete admission before selection" contract.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R4 | partial | This task's slice is met: umbrella maps to 24 cases in the same binary, 84 identities and 191 snapshots unchanged per `fixture-identity-equivalence.json` and `framework-fixtures-required-summary.json`, JVM group bounded at 2 in both workspaces. Per-crate binary consolidation and the debuginfo profile are owned by sibling tasks. |
| R5 | partial | Apalache stays opt-in and is capped at one owner; `model-check-changed` and `model-compiler-inputs` are removed with no active references; remaining trybuild cases are the 18 rustc-dependent ones with reasons in `MAPPING.md`. Concurrent non-cargo stages are owned by .17. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>