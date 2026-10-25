# Fast validation gates and a real inner loop (model)

## Conversation Evidence

> user (2026-10-25): "one of the most important things is the validation gates; and making sure they are fast. we need a fast feeback loop. how can we improve them?"
> assistant: stage timings from fn-247's closure (2,093 s, `tests` 82%), slowest binaries from lane logs, TLC's one-JVM-per-property runner, serial test binaries, the dead Scala `model-check-changed`, and eight improvements, measurement first.
> user (2026-10-25): "why is TLC so slow anyway? don't we run a native check locally, too? is that much faster?"
> assistant: fixed start-up cost (about 1.3–2.6 s per run, measured on a 3-state model) paid once per property; native Kettle exploration 88 µs; TLC added about 520 s to fn-237.3's required gate (441 s default vs 961 s required). TLC stays as the independent checker.
> user (2026-10-25): "yes file new speec"

## Goal & Context

<!-- scope: business -->

The owner calls a fast feedback loop one of the most important properties of the project. Today:

- **Full gate:** `BASH_ENV=/dev/null UMPIRE_APALACHE=require make model-validate` took **2,093 s** at fn-247's closure (`.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json`). At fn-237.3's closure, `make model-check` took 441 s and the TLC-required run 961 s. The soft targets in `model/scripts/time-gate.sh` are 60 s for the gate and 10 s for the inner loop; every recent run misses the gate target.
- **Inner loop:** the measured "inner loop" touches `umpire/backends/native/src/lib.rs` and runs `cargo test -p umpire-checker` (about 6.5 s). That is not what an author does: edit a Model, then re-check it natively and with TLC. No command does that quickly today. `make model-check-changed` still calls the archived Scala `ChangeCheck.scala` and is dead.

Goal: an inner loop for a real Model edit that answers in under a minute, and a full gate several times faster than today, **with the same coverage**. TLC stays the independent checker, as the owner requires (fn-238 shared front end, TLC on the real activity). Its cost is cut by not paying start-up repeatedly and not re-running unchanged checks while iterating. No checks are dropped.

## Architecture & Data Models

<!-- scope: technical -->

Findings this spec acts on (verified 2026-10-25):

| Cost | Evidence | Lever |
| --- | --- | --- |
| `tests` stage: 1,707 s of 2,082 s | fn-247 `validation.json` stages | all rows below |
| Test binaries run serially; parallelism only within a binary | `cargo test --workspace`; Apalache `controls.rs` (17 tests, 349 s, three tests >60 s) leaves other cores idle | parallel test runner across binaries (cargo-nextest, owner approval) |
| TLC: one exploration, then one fresh JVM per property, serially | `umpire/backends/tlc/src/runner.rs:149-182`; activity has 17 claims, so about 18 launches per World | one batched run per check, splitting only on failure; parallel property runs |
| TLC fixed cost about 1.3–2.6 s per run | 3-state probe with the runner's flags (`-XX:+UseSerialGC`, `MSBDiskFPSet`, `-workers 1`); 1.24–1.57 s with `-XX:TieredStopAtLevel=1` and the in-memory fingerprint set | measured JVM flag choice |
| Apalache controls run although fn-211 is deferred | 349 s when `UMPIRE_APALACHE=require` | keep Apalache out of routine gates until fn-211 resumes |
| About 134 integration-test binaries, each linking the dependency graph; debuginfo on | `find model -path '*/tests/*.rs'`; `[profile.dev]` has default debuginfo; lane workers already set `CARGO_PROFILE_TEST_DEBUG=0` by hand | one test binary per crate; `debug = "line-tables-only"` for tests; faster linker |
| trybuild `ui.rs`: 163 s; 121 UI files remain | fn-2511 lane log | keep only cases that need rustc's type checker |
| Non-cargo stages run serially | Go host tests about 58 s, guest freshness 62 s, scripts | run beside the Rust stages |

**Two tiers.**
- **Inner loop (`make model-quick`, new):** `cargo fmt --check`; clippy and tests for changed crates and their reverse dependencies (from `cargo metadata`); native and interpreter checks always. TLC only for fixtures whose emitted `.tla`/`.cfg` changed, with results cached on a content hash of the emitted module, cfg, TLC jar SHA, Java version and runner version.
- **Full gate (`make model-validate`, unchanged meaning):** every stage, TLC required, cache never consulted.

**TLC runner.** `check` runs the exploration, then one run with all properties. On a violation, it splits the remaining properties until each has its own verdict and trace. Property runs may execute in parallel within a configured limit. Results must be identical to today's per-property runs.

## API Contracts

<!-- scope: technical -->

- `make model-quick`: the inner loop. Optional `BASE=<rev>` selects the change set, defaulting to the working tree against `HEAD`. Prints per-stage times and whether each TLC result was fresh or cached.
- `make model-check` / `make model-validate`: same stages and meaning; faster.
- `make model-check-changed`: removed (dead Scala target).
- `umpire-tlc` runner: same public result types (`TlcResult`, `PropertyResult`, `Verdict`); batching and parallelism are internal, with a limit setting.
- `model/scripts/time-gate.sh`: times a realistic Model-edit inner loop (touch `temporal/feature/activity/model.rs`, run `model-quick`) beside the framework-crate one.

## Edge Cases & Constraints

<!-- scope: technical -->

- **Exact verdicts.** Batched TLC must produce the same verdict and the same counterexample trace for every property as separate runs, including when several properties are violated in one module and for liveness with fairness. If TLC cannot report a property in a batch exactly, that property keeps its own run.
- **Cache trust.** The cache only serves `model-quick`. A key misses on any change to the emitted module, cfg, jar, Java version or runner. `model-validate` never reads it, consistent with "no old receipt proves new semantics". Cache entries are local scratch under the target or `.flow/tmp`, never committed.
- **No lost tests.** Changing the test runner or merging binaries must keep the exact test set. Compare full test lists before and after, including ignored tests; counts alone are not enough.
- **Memory.** Parallel TLC and test execution must stay within the host; TLC/JVM tests get a concurrency group. The fn-248 lesson applies: per-lane full targets of about 24 GB filled the disk twice, so measure in the main checkout against the shared target.
- **Determinism.** Parallelism must not change outputs, snapshot bytes or report order.
- **Concurrent owners.** fn-246 lanes edit `temporal/realization` and activity tests. Merging test binaries in `model/temporal/**` waits for fn-246's committed closure; `model/umpire/**` can go first.
- **New tool.** cargo-nextest is a new third-party dev tool, approved by the owner on 2026-10-25; pin its version through `mise`.

## Acceptance Criteria

<!-- scope: both -->

- **R1:** A baseline in `model/design/gate-speed.md` records, on a quiet host and warm target:
  - per-stage times for `model-check` and `model-validate`;
  - per-binary and per-test run times;
  - compile/link time separately from run time;
  - the number of TLC launches per fixture.

  Errors: any unmeasured cost is listed as unknown with its reason, never estimated as measured.
- **R2:** `make model-quick` exists. After an edit to `temporal/feature/activity/model.rs` it re-checks the activity natively and through the interpreter. It runs TLC only when the emitted TLA+ changed, and finishes in under 60 s warm. Errors: a changed emitted module always runs TLC fresh; a cache hit is labelled as such; `model-validate` never uses the cache; a planted stale entry with a different key is ignored.
- **R3:** The TLC runner checks all properties of a module in one batched run and splits only on failure, with optional parallel property runs. Errors: an equivalence control over every TLC fixture and mutant shows identical verdicts and traces to per-property runs, including a module with two violated properties and a liveness violation under fairness.
- **R4:** Tests run in parallel across binaries, with integration tests consolidated to one binary per crate where that cuts link time, and a cheaper test debuginfo profile. Errors: the full test list, including ignored tests, is identical before and after; JVM-backed tests run in a bounded concurrency group.
- **R5:** Gate composition:
  - Apalache is out of routine gates while fn-211 is deferred;
  - non-cargo stages run beside the Rust stages;
  - remaining trybuild cases are only those that need rustc;
  - `model-check-changed` is removed.

  Errors: every stage still runs in `model-validate` and a failing stage still fails the gate with its own exit and name.
- **R6:** Final measurement against R1's baseline, recorded in `model/design/gate-speed.md`. `time-gate.sh` gains the realistic Model-edit inner loop. Targets (proposed; owner to confirm): `model-quick` under 60 s for an activity edit, `model-check` under 3 min, TLC-required `model-validate` under 10 min, all warm. Errors: a missed target is reported with its cause, not relabelled as met.
- **R7:** The framework's checks live in `model/umpire` and run without the Temporal tree. One framework entry point (decided in task .9) runs every framework stage: fmt, clippy, docs and tests of `umpire` crates; the stock, control and Kettle guests; core and Kettle protobuf; the generic and Kettle Go hosts; framework controls. The root `Makefile` keeps only Temporal steps and calls that entry point. Framework tools, pins and caches (TLC/Apalache jars, protoc generators, xtask modules) resolve relative to `model/umpire`, not the Temporal repository root. Errors: a control runs the framework entry point on a copy of `model/umpire` alone, with no `model/temporal` and no root `Makefile`, and it passes; a planted dependency from an `umpire` crate on a Temporal crate, or a Temporal name in `model/umpire`, fails the framework gate.
- **R8:** Each gate stage has a committed budget in one table under `model/umpire` (plus a Temporal-side table for Temporal stages):
  - work counts where the stage has them (test count, generated cases, corpus inputs, TLC launches), enforced hard;
  - a warm wall-time budget per stage and per tier.

  `time-gate` reports every stage against its budget. A work count above budget fails the stage. A wall-time overrun above a tolerance (proposal: 25%) is reported, and fails on a repeat run, so one slow run on a busy host does not fail the gate. Adding a stage without a budget entry fails the gate. Errors: a planted over-budget count, a planted unbudgeted stage and two consecutive planted slow runs each fail with the stage's name.

## Boundaries

<!-- scope: business -->

- No coverage removed. Every claim, mutant, fixture and backend agreement check keeps running in `model-validate`.
- TLC is not replaced by the native checker. The symbolic back end (fn-240) and Apalache (fn-211) stay deferred.
- No change to Model semantics, snapshot meaning or checker verdicts.
- No remote CI or build cache service.
- The live Temporal gates (`umpire-check-live-tests`) are out of scope.

## Decision Context

<!-- scope: both -->

2026-10-25: filed from the conversation above. The owner ranks fast feedback among the most important properties.

Rejected options:
- **Dropping TLC from gates:** it is the independent checker.
- **Persistent TLC JVM server:** TLC keeps static state between runs; batching gets most of the gain without that risk.
- **Caching TLC results in the full gate:** the receipt policy forbids it.

2026-10-25, owner: cargo-nextest approved, pinned via mise (R4, task .3).

2026-10-25, owner: "move any umpire-specific tasks/checks/targets out of the Makefile and into model/umpire as the framework stands alone and must be verifiable temporal-agnostic" (R7, tasks .9–.11). Couplings found: 45 framework paths of the form `../../../..` that reach the repository root, the TLC/Apalache caches rooted at `model/build/…`, `dev/xtask/src/apalache_pins.rs` included from a framework crate, protoc generators from the root `.bin/`, and one Cargo workspace and lockfile shared with the Temporal crates.

2026-10-25, owner: fn-254 replaces fn-246 at the front of the current order.

2026-10-25, owner: new testing steps (fn-255) must be bounded and must not undo the fast-gate goal. R8 adds committed per-stage budgets that every later stage must declare. Work counts are enforced hard; wall time is enforced with a tolerance and a repeat, to avoid flaky failures on a shared host.

Placement: first in the current order (owner, 2026-10-25). fn-246's in-flight tasks continue to their boundaries. The `umpire/**`, Makefile, scripts and TLC-runner files are disjoint from fn-246's. Test-binary consolidation in `temporal/**` waits for fn-246's committed closure.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | A baseline in `model/design/gate-speed.md` records, on a quiet host and warm target: | fn-254-fast-validation-gates-and-a-real-inner.1, fn-254-fast-validation-gates-and-a-real-inner.8 | — |
| R2 | `make model-quick` exists. After an edit to `temporal/feature/activity/model.rs` it re-checks the activity natively and through the interpreter. It runs TLC only when the emitted TLA+ changed, and finishes in under 60 s warm. Errors: a changed emitted module always runs TLC fresh; a cache hit is labelled as such; `model-validate` never uses the cache; a planted stale entry with a different key is ignored. | fn-254-fast-validation-gates-and-a-real-inner.6 | — |
| R3 | The TLC runner checks all properties of a module in one batched run and splits only on failure, with optional parallel property runs. Errors: an equivalence control over every TLC fixture and mutant shows identical verdicts and traces to per-property runs, including a module with two violated properties and a liveness violation under fairness. | fn-254-fast-validation-gates-and-a-real-inner.2 | — |
| R4 | Tests run in parallel across binaries, with integration tests consolidated to one binary per crate where that cuts link time, and a cheaper test debuginfo profile. Errors: the full test list, including ignored tests, is identical before and after; JVM-backed tests run in a bounded concurrency group. | fn-254-fast-validation-gates-and-a-real-inner.3, fn-254-fast-validation-gates-and-a-real-inner.4, fn-254-fast-validation-gates-and-a-real-inner.7 | — |
| R5 | Gate composition: | fn-254-fast-validation-gates-and-a-real-inner.5 | — |
| R6 | Final measurement against R1's baseline, recorded in `model/design/gate-speed.md`. `time-gate.sh` gains the realistic Model-edit inner loop. Targets (proposed; owner to confirm): `model-quick` under 60 s for an activity edit, `model-check` under 3 min, TLC-required `model-validate` under 10 min, all warm. Errors: a missed target is reported with its cause, not relabelled as met. | fn-254-fast-validation-gates-and-a-real-inner.8 | — |
| R7 | The framework's checks live in `model/umpire` and run without the Temporal tree. One framework entry point (decided in task .9) runs every framework stage: fmt, clippy, docs and tests of `umpire` crates; the stock, control and Kettle guests; core and Kettle protobuf; the generic and Kettle Go hosts; framework controls. The root `Makefile` keeps only Temporal steps and calls that entry point. Framework tools, pins and caches (TLC/Apalache jars, protoc generators, xtask modules) resolve relative to `model/umpire`, not the Temporal repository root. Errors: a control runs the framework entry point on a copy of `model/umpire` alone, with no `model/temporal` and no root `Makefile`, and it passes; a planted dependency from an `umpire` crate on a Temporal crate, or a Temporal name in `model/umpire`, fails the framework gate. | fn-254-fast-validation-gates-and-a-real-inner.10, fn-254-fast-validation-gates-and-a-real-inner.11, fn-254-fast-validation-gates-and-a-real-inner.9 | — |
| R8 | Each gate stage has a committed budget in one table under `model/umpire` (plus a Temporal-side table for Temporal stages): | fn-254-fast-validation-gates-and-a-real-inner.12 | — |

## Parked unknowns

- How compile/link time splits against run time inside the 1,707 s `tests` stage (R1 measures it).
- Whether TLC can report several violated properties exactly in one run, or batching must stop at the first violation and rerun the rest.
