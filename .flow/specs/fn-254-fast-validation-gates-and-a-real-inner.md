# Fast validation gates and a real inner loop (model)

## Conversation Evidence

> user (2026-10-25): "one of the most important things is the validation gates; and making sure they are fast. we need a fast feeback loop. how can we improve them?"
> assistant: stage timings from fn-247's closure (2,093 s, `tests` 82%), slowest binaries from lane logs, TLC's one-JVM-per-property runner, serial test binaries, the dead Scala `model-check-changed`, and eight improvements, measurement first.
> user (2026-10-25): "why is TLC so slow anyway? don't we run a native check locally, too? is that much faster?"
> assistant: fixed start-up cost (about 1.3–2.6 s per run, measured on a 3-state model) paid once per property; native Kettle exploration 88 µs; TLC added about 520 s to fn-237.3's required gate (441 s default vs 961 s required). TLC stays as the independent checker.
> user (2026-10-25): "yes file new speec"
> user (2026-10-25): "can we ignore baseline measurement? we already have historic data"
> user (2026-10-25): "can we postpone most of the validation gates to the end of fn-254? except for the ones we're changing in that task?"

> user (2026-10-25): "can we add fixes for the three concrete sources of accidental complexity to existing tasks"

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
- The gate timer measures a reversible content edit to `temporal/feature/activity/model.rs`, asserts that the activity was selected, and runs `model-quick` beside the framework-crate benchmark. Record emission-preserving/cache-hit and changed-emission/fresh-TLC measurements separately; an mtime-only touch is insufficient.

## Edge Cases & Constraints

<!-- scope: technical -->

- **Exact verdicts.** Batched TLC must produce the same verdict and the same counterexample trace for every property as separate runs, including when several properties are violated in one module and for liveness with fairness. If TLC cannot report a property in a batch exactly, that property keeps its own run.
- **Cache trust.** The cache only serves `model-quick`. A key misses on any change to the emitted module, cfg, jar, Java version or runner. `model-validate` never reads it, consistent with "no old receipt proves new semantics". Cache entries are local scratch under the target or `.flow/tmp`, never committed.
- **No lost tests.** Compare package, target, full test name and ignored status under matching features/backend modes, including doctests separately. Consolidation may rename targets/module paths only through a checked, durable one-to-one identity mapping. Converted UI cases retain an explicit case-to-replacement mapping and diagnostic coverage. Missing, duplicate and unmatched cases fail; counts alone are insufficient.
- **Memory.** Parallel TLC and test execution must stay within the host; TLC/JVM tests get a concurrency group. The fn-248 lesson applies: per-lane full targets of about 24 GB filled the disk twice, so measure in the main checkout against the shared target.
- **Determinism.** Parallelism must not change outputs, snapshot bytes or report order.
- **Concurrent owners.** fn-246 lanes edit `temporal/realization` and activity tests. Merging test binaries in `model/temporal/**` waits for fn-246's committed closure; `model/umpire/**` can go first.
- **New tool.** cargo-nextest is a new third-party dev tool, approved by the owner on 2026-10-25; pin its version through `mise`.

## Acceptance Criteria

<!-- scope: both -->

- **R1:** `model/design/gate-speed.md` consolidates existing historical gate, stage, binary/test, compile/link and TLC-launch evidence. The owner waived fresh baseline measurements on 2026-10-25; no new baseline gate or serial observer runs are required before optimization. Retain source, backend mode, workload, host/profile conditions and evidence references where available. Errors: missing costs/counts are explicitly unknown, structural predictions are labelled estimates, and differing modes/sources/coverage are never presented as equivalent measured runs. Correctness inventories before test migrations remain required by R4/R5. R6's final measurements remain required.
- **R2:** `make model-quick` exists. After an edit to `temporal/feature/activity/model.rs` it re-checks the activity natively and through the interpreter. It runs TLC only when the emitted TLA+ changed, and finishes in under 60 s warm. Errors: a changed emitted module always runs TLC fresh; a cache hit is labelled as such; `model-validate` never uses the cache; a planted stale entry with a different key is ignored.
- **R3:** The TLC runner checks all properties of a module in one batched run and splits only on failure, with optional parallel property runs. Errors: an equivalence control over every TLC fixture and mutant shows identical verdicts and traces to per-property runs, including a module with two violated properties and a liveness violation under fairness.
- **R4:** Tests run in parallel across binaries, with integration tests consolidated to one binary per crate where that cuts link time, and a cheaper test debuginfo profile. Errors: the full logical test inventory, including ignored status, is identical before and after under the checked identity mapping above; JVM-backed tests run in a bounded concurrency group.
- **R5:** Gate composition:
  - Apalache is out of routine gates while fn-211 is deferred;
  - non-cargo stages run beside the Rust stages;
  - remaining trybuild cases are only those that need rustc;
  - `model-check-changed` is removed.

  Errors: every stage still runs in `model-validate` and a failing stage still fails the gate with its own exit and name.
- **R6:** Final warm measurements are recorded beside R1's historical references in `model/design/gate-speed.md`. `time-gate.sh` gains the realistic Model-edit inner loop. Execution targets: `model-quick` under 60 s for an activity edit, `model-check` under 3 min, TLC-required `model-validate` under 10 min. Errors: a missed target is reported with its cause, not relabelled as met; incomparable historical modes/workloads are separated and do not support an attributed speedup claim.
- **R7:** The framework's checks live in `model/umpire` and run without the Temporal tree. One framework entry point (decided in task .9) runs every framework stage: fmt, clippy, docs and tests of `umpire` crates; the stock, control and Kettle guests; core and Kettle protobuf; the generic and Kettle Go hosts; framework controls. The root `Makefile` keeps only Temporal steps and calls that entry point. Framework tools, pins and caches (TLC/Apalache jars, protoc generators, xtask modules) resolve relative to `model/umpire`, not the Temporal repository root. Errors: a control runs the framework entry point on a copy of `model/umpire` alone, with no `model/temporal` and no root `Makefile`, and it passes; a planted dependency from an `umpire` crate on a Temporal crate, or a Temporal name in `model/umpire`, fails the framework gate.
The R7 tool contract also requires a dedicated guest-driver executable independent of the general xtask/gate executable. Guest cache invalidation follows the exact guest driver, current lockfile and compiler inputs. Unrelated general-CLI/gate implementation changes must not select a new guest cache. One admitted context owns framework tool/pin roots, caller workspace/output/target resolution, raw-OS environment encoding and canonical physical-to-logical remaps; commands do not rediscover these independently. Missing local pins or escaping/ambiguous roots refuse explicitly. These requirements retain existing compiler/lock/source admission and cache invalidation for actual guest-driver changes.

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

## Verification cadence

Owner amendment, 2026-10-25: run focused checks for each implementation and defer unrelated broad validation to .8. This supersedes per-task instructions to run an unrelated whole-workspace suite, `make model-check` or `make model-validate`. Their full coverage remains required at the final checkpoint; no deferred command is reported green.

- Keep discriminating regressions, relevant native/backend tests and affected fmt/clippy checks with the change. Gate changes also retain their real wiring, mode/cache propagation, failure/cleanup and coverage-preservation controls.
- .2 retains exact independent-strategy agreement over every TLC fixture/mutant and its safety/liveness/fairness/error controls. Other workspace targets are deferred, with exact identities recorded. No repeated complete workspace run is required for this task.
- .3/.4/.7 retain exact pre/post logical inventories, ignored status, mapping and affected test execution. .5/.6 retain changed gate/cache controls. .13 retains both locked workspace/build closures and affected tool/root/guest identity checks. .10 retains orchestration/mode/failure controls. .11's real isolated framework entry-point proof is itself the gate being changed, so it remains required. .12 retains budget controls.
- .8 runs the full current `model-check` and TLC-required `model-validate`, final quick-cache-hit/fresh-TLC measurements, standalone proof and stage budgets. Reuse already-passing evidence only when its actual inputs and scope remain equivalent; changed inputs need renewed checks. Full validation never uses the quick TLC result cache.
- Preserve completed or active proof work; do not restart a running command merely to change its observation window. Let an active relevant test finish before stopping an unrelated orchestration queue. Record deferred, incomplete and failed work honestly.

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

Placement: first in the current order (owner, 2026-10-25). fn-246's in-flight tasks continue to their boundaries. Shared runtime/front/Host source owners and workspace/generated assets require an integration boundary; disjoint implementation lanes may run in parallel. Test-binary consolidation in `temporal/**` waits for fn-246's committed closure.

2026-10-25, execution refinement under the owner's instruction to follow recommendations autonomously:
- Deliver the independent framework/gate checkpoint first. Then resume fn-246 to committed spec closure, perform .7, and finish .8. fn-255's independent lanes may interleave within their stated ownership. Whole-spec priority is not a waiver of .7's existing closure requirement.
- Flow only supports same-spec task dependencies. Keep .7 explicitly blocked in Flow with the external prerequisite and release it only after verifying fn-246's committed spec closure; do not block the whole fn-254 spec on fn-246.
- Task .9 settles the entry point and workspace using its inventory; adopt the recommended framework-owned Rust xtask and separate Cargo workspace unless evidence identifies a smaller design satisfying the directory-only control. This is delegated implementation judgment, not a new request for approval.
- Split the former oversized .10: .13 separates workspace/tool roots, paths and identity inputs; .10 composes the framework gate and Temporal delegation. Serialize shared-file writers and reanchor after the integration boundary; framework paths overlap fn-246's runtime/front/Host owners.
- .12 uses one immediate retry of a slow stage after competing owned work drains, on identical source/tool/budget/tier inputs. Count both attempts in tier wall time; only a second over-tolerance run fails. Interruptions are incomplete evidence, never a pass. Every later stage/count change updates its budget entry.
- Proposed R6 targets and R8's 25% tolerance are the execution defaults under delegated judgment; report misses honestly.
- Owner amendment, 2026-10-25: skip fresh baseline measurements and proceed using historical evidence. Task .1 becomes a historical-reference report; prepared optional probes may support final measurement. This does not waive migration inventories, semantic equivalence, required in-flight validation, final measurements or stage budgets.

### Framework layout selected by task .9

2026-10-25, fn-254.9 delegated selection: adopt framework-owned Rust `cargo xtask gate`
at `model/umpire/dev/xtask` and independent `model/umpire/Cargo.toml`/`Cargo.lock`;
Temporal remains at `model/` with `exclude = ["umpire"]` and path dependencies inward.
The complete stage/script/module/cache inventory and all 45 root joins are recorded in
`model/design/framework-gate.md` at the inventory checkpoint named there. Copy tiny local
policy/tool configs; framework pins/tools/caches/nextest/budgets have no parent dependency.
Keep framework tool root distinct from caller guest workspace/lock/output; .13 owns remaps,
locked closure, current-input reproduction and refreshed Temporal guest publication. .13's
intermediate root gate checks both workspaces; .10 owns final orchestration and mixed-control
splits. Preserve R7's standalone planted-name/dependency rejection using exactly
`dev/gate/policy/framework-boundary.txt` as declarative gate-policy data exempted from its
own name scan, with no directory/source-code exemption. The existing Temporal vocabulary
policy stays Temporal-owned; generic libraries/scanners/compiler/runtime remain agnostic.
Filesystem hygiene and the standalone copy need no Git; use declared current inputs and
preserve path/symlink, raw-byte/status and dependency-containment checks. No Model or
conformance policy changes are authorized by this design.

2026-10-25, owner: "we don't need to check for mise everywhere all the time, we can assume its installed - or fail hard". Tooling assumes mise and its pinned tools are installed and runs them through `mise exec`. Presence and pin probes and their negative controls are removed (task .14, after .13). Auto-install stays disabled so a missing tool fails hard. This replaces fn-247 R2's missing-tool and probe-failed-tool admission.

2026-10-25, owner: file a task after .13 and .10 to port the remaining model shell scripts. Task .15 (after .13, .10 and .14; before .8) moves script logic into xtask or guest-driver subcommands, thin wrappers into the Makefile and the shell controls into Rust tests. Only the xtask bootstrap stays shell. Retired-name patterns move to Rust `regex`, reversing fn-247's keep-`grep -E` decision.

2026-10-25, owner: file the mise download work in fn-254. Task .16 follows .13/.14 and precedes .10: framework-owned mise declarations provision pinned TLC and opt-in Apalache assets; bespoke download/extraction plumbing is removed. Runtime JAR checksum/path admission remains, since an installed mise artifact can be corrupted without a locked reinstall detecting it. This does not restore .14's removed tool-presence/version probes. Task .15 ports the resulting entry points without recreating native downloaders. Focused provider checks run in .16; broad gates remain in .8.

Maintainability (plan review): duplication - task .10 repeated its validation-cadence paragraph; consolidated to one paragraph. structure - none identified. Ownership advisories are addressed with explicit .14/.15/.16 paths and .10 workflow/script ownership; existing shared-file pairs still serialize.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `model/design/gate-speed.md` consolidates existing historical gate, stage, binary/test, compile/link and TLC-launch evidence. The owner waived fresh baseline measurements on 2026-10-25; no new baseline gate or serial observer runs are required before optimization. Retain source, backend mode, workload, host/profile conditions and evidence references where available. Errors: missing costs/counts are explicitly unknown, structural predictions are labelled estimates, and differing modes/sources/coverage are never presented as equivalent measured runs. Correctness inventories before test migrations remain required by R4/R5. R6's final measurements remain required. | fn-254-fast-validation-gates-and-a-real-inner.1, fn-254-fast-validation-gates-and-a-real-inner.8 | — |
| R2 | `make model-quick` exists. After an edit to `temporal/feature/activity/model.rs` it re-checks the activity natively and through the interpreter. It runs TLC only when the emitted TLA+ changed, and finishes in under 60 s warm. Errors: a changed emitted module always runs TLC fresh; a cache hit is labelled as such; `model-validate` never uses the cache; a planted stale entry with a different key is ignored. | fn-254-fast-validation-gates-and-a-real-inner.6 | — |
| R3 | The TLC runner checks all properties of a module in one batched run and splits only on failure, with optional parallel property runs. Errors: an equivalence control over every TLC fixture and mutant shows identical verdicts and traces to per-property runs, including a module with two violated properties and a liveness violation under fairness. | fn-254-fast-validation-gates-and-a-real-inner.2 | — |
| R4 | Tests run in parallel across binaries, with integration tests consolidated to one binary per crate where that cuts link time, and a cheaper test debuginfo profile. Errors: the full logical test inventory, including ignored status, is identical before and after under the checked identity mapping above; JVM-backed tests run in a bounded concurrency group. | fn-254-fast-validation-gates-and-a-real-inner.3, fn-254-fast-validation-gates-and-a-real-inner.4, fn-254-fast-validation-gates-and-a-real-inner.7 | — |
| R5 | Gate composition: | fn-254-fast-validation-gates-and-a-real-inner.5 | — |
| R6 | Final warm measurements are recorded beside R1's historical references in `model/design/gate-speed.md`. `time-gate.sh` gains the realistic Model-edit inner loop. Execution targets: `model-quick` under 60 s for an activity edit, `model-check` under 3 min, TLC-required `model-validate` under 10 min. Errors: a missed target is reported with its cause, not relabelled as met; incomparable historical modes/workloads are separated and do not support an attributed speedup claim. | fn-254-fast-validation-gates-and-a-real-inner.8 | — |
| R7 | The framework's checks live in `model/umpire` and run without the Temporal tree. One framework entry point (decided in task .9) runs every framework stage: fmt, clippy, docs and tests of `umpire` crates; the stock, control and Kettle guests; core and Kettle protobuf; the generic and Kettle Go hosts; framework controls. The root `Makefile` keeps only Temporal steps and calls that entry point. Framework tools, pins and caches (TLC/Apalache jars, protoc generators, xtask modules) resolve relative to `model/umpire`, not the Temporal repository root. Errors: a control runs the framework entry point on a copy of `model/umpire` alone, with no `model/temporal` and no root `Makefile`, and it passes; a planted dependency from an `umpire` crate on a Temporal crate, or a Temporal name in `model/umpire`, fails the framework gate. | fn-254-fast-validation-gates-and-a-real-inner.10, fn-254-fast-validation-gates-and-a-real-inner.11, fn-254-fast-validation-gates-and-a-real-inner.13, fn-254-fast-validation-gates-and-a-real-inner.9 | — |
| R8 | Each gate stage has a committed budget in one table under `model/umpire` (plus a Temporal-side table for Temporal stages): | fn-254-fast-validation-gates-and-a-real-inner.12 | — |

## Parked unknowns

- How compile/link time splits against run time inside the 1,707 s `tests` stage (R1 measures it).
- Whether TLC can report several violated properties exactly in one run, or batching must stop at the first violation and rerun the rest.

### Native tool simplification amendment

2026-10-25, owner requested fixes for the three confirmed accidental-complexity sources in existing tasks. Task .13 extracts a dedicated `umpire-guest-driver` package/executable at `model/umpire/dev/guest-driver`, centralizes admitted tool/workspace context and remap/environment encoding there, and leaves `dev/xtask` as the general command front door. The general CLI may reuse the driver library's context utilities but invokes the driver executable for guest operations; the driver never depends on the CLI or gate implementation. Task .10 composes gates through those interfaces and proves a compiled CLI-only change leaves driver bytes and the guest cache namespace unchanged. Task .11 exercises the same local tools/driver in the real directory-only gate. Preserve the full current lock/compiler inputs and exact driver hash, portable sidecar semantics, raw OS bytes, jobserver/loader inheritance, leases, cleanup and atomic publication. No new tasks, Model semantics, third-party dependencies or broad per-task validation gates are introduced.
