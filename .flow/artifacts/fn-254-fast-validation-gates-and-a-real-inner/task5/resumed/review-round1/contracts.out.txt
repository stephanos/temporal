## Review: fn-254.5 at fdf115db99 (contracts-and-consistency axis)

I read the task file, the full source diff (Makefile, both nextest configs, both xtasks, the testing runner and its consumer controls, the fixtures build/test entry, the TLC runner, the parser-refusal table and harness, MAPPING.md, the Temporal trybuild splits) and the current state of the touched files plus their callers. Evidence summaries consulted: final-validation-summary, fixture-identity-equivalence, macro-before-after-comparison, and the decision document at `.flow/tmp/fn254-jvm-parallel-shape/decision.md`.

Verified consistent with the stated contract:

- **24 cases, same binary, admission before selection.** `run_selected` admits every fixture (registry equality, consumer ownership, pending/orphan/skipped goldens, snapshot validity) before the selected fixture renders twice. The generated `mod fixtures { #[test] fn <module>() }` lives in the same `fixtures` test target, so the `package(=umpire-fixtures) & binary(=fixtures)` jvm-group filter still applies. `interpreted_agrees_with_compiled` and the fixture-local tests are untouched. The consumer controls (`consumers.rs:216-390`) cover sibling-not-executed, pre-render admission failures naming the unselected fixture, unknown selection, ambiguous consumers, dynamic orphan/undeclared/nondeterminism refusals and skipped-sibling golden completeness.
- **Parser table equals the production path.** `machine::expand` is `generate(declare(..))` then `finish`, and production (`lib.rs:59-70`) only inserts `lifted::embed` between them, which returns items rather than a `Result`, so no diagnostic can originate there. MAPPING.md's claim that the omitted embedding cannot change refusals is correct. `tests/ui` now holds exactly the 18 files the explicit `refusals()` list names and MAPPING.md documents. Moved sources and stderr files are 100% similarity renames.
- **Thread policy matches the decision doc.** Group 2 in both workspaces, `[profile.required] test-threads = 4` (nextest applies default-profile overrides to custom profiles, and the evidence run observed two owners/four checker JVMs), Apalache opt-in capped at 1, skipped native mode untouched, standalone child's hardcoded 2 replaced by the same `configure`, receipt reports the computed value. The standalone copy requires `.config/nextest.toml` (`standalone.rs:86`), so the `required` profile exists in the copied root.
- **TLC probe heap.** Discovery `tlc2.TLC -h` now gets `-Xmx512m` from the same `Limits::default()` (process/support.rs:32).
- **Dead targets.** `model-check-changed` and `model-compiler-inputs` have no remaining references in Makefile or scripts.
- **Temporal quick selector.** `umpire-temporal-tests=fixtures` is unchanged; the framework `umpire-fixtures` package runs unfiltered in quick, so the removal of the `fixtures` identity breaks no existing filter.

### Findings

**1.**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/test_run.rs:25-30`, `model/dev/xtask/src/test_run.rs:21-27`
- **R-IDs**: [R4]
- **Problem**: `configure` unconditionally assigns `NEXTEST_TEST_THREADS=4` in required mode. The decision doc, the comments and the test names call this a "cap", but it is an override in both directions. Before this change the regular required gate did not set this variable, so an operator-supplied `NEXTEST_TEST_THREADS=1 UMPIRE_BACKENDS=require make model-validate` on a constrained host ran one slot. Now it silently runs four slots with two JVM owners. The same applies to any cross-invocation lease that .17 may express through this env var.
- **Suggestion**: Compute `min(inherited NEXTEST_TEST_THREADS, test_threads(apalache))` when an inherited value parses, and keep the override only for larger inherited values. Add a `("require", inherited "1") -> "1"` row to the existing control matrices.

**2.**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/author/dsl/macros/tests/fixtures.rs:40-59`
- **R-IDs**: [R5]
- **Problem**: `refusals()` replaced `compile_fail("tests/ui/*.rs")` with 18 hard-coded paths. Nothing asserts the directory equals that list, so a new `tests/ui/<case>.rs` plus `.stderr` dropped in by a later author compiles, passes the fixture binary and is never executed. The 61 migrated inputs live under `parser_refusals/inputs/`, so the glob would still select exactly the retained 18 today.
- **Suggestion**: Restore the glob, or keep the explicit list and add an assertion that `tests/ui/*.rs` on disk equals it (MAPPING.md's retained table can be derived from the same list).

**3.**
- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/gate.rs:13-19`, `model/dev/xtask/src/test_run.rs:7-11`
- **R-IDs**: []
- **Problem**: The Temporal tests stage applies `configure` twice: once in `test_run::run` and again inside `stage` for `name == "tests"`. Both derive the same booleans from the same env vars, so the result is identical, but one of the two is dead weight and the duplicated derivation is a Middle Man / Duplicated Code smell.
- **Suggestion**: Keep the `stage` hook (the child-process control exercises it) and drop the call in `test_run::run`, or the reverse.

**FYI (not findings):**
- `.flow/artifacts/fn-257-extract-the-standalone-umpire/agent-throughput.json` renames `integrated_at` to `integration_commit` inside this range. It is unrelated to fn-254.5 and should be mentioned in the task's done summary so the provenance is not surprising.
- `model/design/rust-verdict-measurements.md:66` describes a historical `--exact fixtures` run reporting one test; it is a dated measurement record, so it does not need updating.
- `.plans/UMPIRE_MODULES.md:111` still lists `temporal.check.changed` as a Scala gate entry point; this row describes the archived tree and predates the change.

### Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R4 | partial | Fixture umbrella split into 24 nextest-schedulable cases with identity audit (61 to 84, one removed `fixtures`, 60 preserved, `interpreted::timing` still ignored); jvm group bounded at 2. Binary consolidation and the debuginfo profile belong to other tasks. |
| R5 | partial | Apalache stays opt-in (`gate.rs:239`, `configure` cap); trybuild reduced to 18 rustc-dependent cases with per-case reasons; `model-check-changed` removed with no references. Non-cargo stages beside Rust stages is .17's scope. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>