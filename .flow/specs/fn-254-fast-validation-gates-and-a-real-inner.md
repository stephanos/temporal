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
- **R6:** Final warm measurements are recorded beside R1's historical references in `model/design/gate-speed.md`. The native gate timer ported by .15 gains the realistic Model-edit inner loop. Execution targets: `model-quick` under 60 s for an activity edit, `model-check` under 3 min, TLC-required `model-validate` under 10 min. Errors: a missed target is reported with its cause, not relabelled as met; incomparable historical modes/workloads are separated and do not support an attributed speedup claim.
- **R7:** The framework's checks live in `model/umpire` and run without the Temporal tree. One framework entry point (decided in task .9) runs every framework stage: fmt, clippy, docs and tests of `umpire` crates; the stock, control and Kettle guests; core and Kettle protobuf; the generic and Kettle Go hosts; framework controls. The root `Makefile` keeps only Temporal steps and calls that entry point. Framework tools, pins and caches (TLC/Apalache jars, protoc generators, xtask modules) resolve relative to `model/umpire`, not the Temporal repository root. Errors: a control runs the framework entry point on a copy of `model/umpire` alone, with no `model/temporal` and no root `Makefile`, and it passes; a planted dependency from an `umpire` crate on a Temporal crate, or a Temporal name in `model/umpire`, fails the framework gate.
The R7 tool contract also requires a dedicated guest-driver executable independent of the general xtask/gate executable. Guest cache invalidation follows the exact guest driver, current lockfile and compiler inputs. Unrelated general-CLI/gate implementation changes must not select a new guest cache. One admitted context owns framework tool/pin roots, caller workspace/output/target resolution, raw-OS environment encoding and canonical physical-to-logical remaps; commands do not rediscover these independently. Missing local pins or escaping/ambiguous roots refuse explicitly. These requirements retain existing compiler/lock/source admission and cache invalidation for actual guest-driver changes.

- **R8:** Each gate stage has a committed budget in `model/umpire/dev/gate/budgets.toml` (plus `model/dev/gate/budgets.toml` for Temporal stages), as selected by .9:
  - work counts where the stage has them (test count, generated cases, corpus inputs, TLC launches), enforced hard;
  - a warm wall-time budget per stage and per tier.

  `time-gate` reports every stage against its budget. A work count above budget fails the stage. A wall-time overrun above a tolerance (proposal: 25%) is reported, and fails on a repeat run, so one slow run on a busy host does not fail the gate. Adding a stage without a budget entry fails the gate. Errors: a planted over-budget count, a planted unbudgeted stage and two consecutive planted slow runs each fail with the stage's name.

- **R9:** Host build costs are reduced through lighter dev debuginfo in both independent Rust workspaces, measured local compiler-cache reuse across different checkout paths, and a measured mise-pinned mold adoption decision for supported Linux host targets. Errors: effective flags and cache statistics distinguish compilation reuse from Cargo Fresh and final links; incompatible source/compiler/profile/target inputs do not reuse stale compilation; missing pinned tools and cache failures remain visible under the existing fail-hard policy; bootstrap isolation, guest-driver wrapper/compiler/lock identity, optimized release/guest profiles, WASI linking and framework-only operation remain intact. No speedup is claimed without comparable workload evidence.

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

2026-10-25, owner: file the mise download work in fn-254. Task .16 follows .13/.14: framework-owned mise declarations provision pinned TLC and opt-in Apalache assets; bespoke download/extraction plumbing is removed. Task .10 can compose earlier against .13's existing stable provisioning commands; .16 preserves public entry-point/status behavior and renews changed-input callers, so provider replacement is not a .10 prerequisite. Runtime JAR checksum/path admission remains, since an installed mise artifact can be corrupted without a locked reinstall detecting it. This does not restore .14's removed tool-presence/version probes. Task .15 ports the resulting entry points without recreating native downloaders. Focused provider checks run in .16; broad gates remain in .8.

Maintainability (plan review): duplication - task .10 repeated its validation-cadence paragraph; consolidated to one paragraph. Duplication also exists in tier targets repeated between timers and tables and in stage graph ownership; .12/.8 use the two selected tables and .17 removes or derives duplicate graphs. Structure - `Gate::full` collects scheduler, budget and backend-cap changes; .17 keeps scheduling beneath stable public entry points, .12 consumes its receipts for budget enforcement, and shared JVM admission remains explicit. Ownership advisories are addressed with explicit .14/.15/.16 paths and .10 workflow/script ownership; existing shared-file pairs still serialize.

### Framework feedback and parallel ownership refinement

Frontload .10 immediately after committed .13/.9: an independently callable framework gate
provides framework-only feedback without running Temporal stages or extracting a repository.
Existing public provisioning commands suffice; .14 probe removal and .16 provider internals
retain their contracts and do not block that checkpoint. The checkpoint is now committed.
The 2026-10-26 owner priority puts throughput .5/.17 before .16 provisioning and .15 ports.
Schedule actual overlapping owners serially.

Task .10 leaves the real Cargo test execution in working `test_run.rs` leaves and gate
orchestration in its real gate module. These delivered entry points remain real executable
primitives; the owner's later explicit request adds .17 for uniform nextest and one stage DAG.
Then .3 owns runner leaves/config/pins, while .6 owns quick CLI/Make/cache and their actual
registrations/dependency edits. Their full Touches are disjoint, so admission can run both.
Task .11 can additionally own gate-local standalone registration/controls without sharing
.6's dispatcher or .3's test-runner leaves if .10 realizes the stated boundaries. Correct
actual ownership before admission if implementation differs; do not bypass the scheduler.

Task .11 retains its actual complete directory-only gate on its exact candidate. Later
producer edits do not inherit that proof: .8 already waits for the final producers and renews
changed-input standalone proof. .3/.6/.11 feed exact inventory/tier facts to .12 and consolidated
documentation to .8. No stage/test coverage, actual proof or committed-predecessor requirement
is relaxed. Source parallelism still observes target/cache leases and bounded JVM resources.

### Throughput priority refinement (2026-10-26)

Owner: "let's file a task in fn-254 for this" and "priorizie work in fn-254 to enable faster validation gates asap", following the nextest/mise discussion.

The current correct five-consumer quick workload passes in 444.027 s, including 388.875 s
of realization compiler-admission tests despite 14 TLC cache hits. Earlier roughly 20 s
runs omitted affected consumers and are historical incomplete coverage, not the current SLO.
The isolated framework proof took 678.821 s cold, with 507.204 s in tests. These identify
work to reduce; they do not justify attribution across different sources or workloads.

Priority is .6's correct-selection/cache checkpoint, then .5's test granularity/compiler reuse,
then new priority-0 .17's uniform nextest and single dependency/resource stage DAG. .5 exposes
the framework's 24 serial fixtures as cases in the same binary, preserves both TLC renders,
and batches/reuses compiler work without losing cases. Temporal's exact activity `fixtures`
selector remains unchanged. .17 owns dependency-independent stage scheduling and failure
cleanup, preferably through verified mise capabilities rather than a new custom scheduler;
xtask retains project-specific executable checks. Keep one graph owner, not competing copies.

Downloads .16 and shell ports .15 follow .17. Budget enforcement .12 follows .15 and consumes the final native timer, graph and counts; final measurements .8 follow .12. The budget tables remain `model/umpire/dev/gate/budgets.toml` and `model/dev/gate/budgets.toml`.
fn-254 gets first admission to shared build/JVM/review capacity. Already running independent
work may reach a safe checkpoint; disjoint source work need not stop. No fn-246 dependency
is fabricated, and .7's committed fn-246 prerequisite remains intact.

All Rust unit/integration production gate paths include nextest scheduling, including quick;
Cargo retains doctests and compiler-only probes. Actual selected package/target/test/ignored
semantics and zero-selection refusal must survive migration. Mise jobs, nextest slots and
nested compiler/JVM workers share one bounded resource contract. Shared target locks and
generated output/consumer edges determine legal overlap. Keep no-auto-install, cache-bypassing
full validation, source-only standalone operation, caller context, named failures, owned sibling
cleanup and invocation-local complete stage receipts. A changed scheduler gets focused real
proof, including its standalone control, once frozen; unrelated final suites remain .8.


### Residual build-cost amendment (2026-10-28)

Owner: "file tasks to close gaps" after reviewing dev debuginfo, cross-worktree sccache,
Linux mold and remaining integration binaries. Tasks .18–.20 close build-cost work left
unimplemented by .4: its completion proves the cheaper test profile and 108-to-36 target
consolidation, explicitly not a dev-profile change or alternate-linker measurement.
Task .21 rechecks current residual framework targets against .4/.5's identity/isolation
contracts and consolidates only safe candidates; explicit evidence can justify retaining
separate targets. These tasks feed .8's combined measurement and current-input proof.
The local compilation cache is not a semantic-result cache: full validation still executes
every selected check and bypasses the quick TLC-result cache. .7 stays blocked until
fn-246's committed whole-spec closure; .12 stays owner-paused in_progress; fn-258 stays deferred.

## Quick commands

```bash
cd model/umpire && mise exec -- cargo build --locked -p xtask --bin xtask --profile dev
cd model && mise exec -- cargo build --locked -p temporal-xtask --bin temporal-xtask --profile dev
```

Run each command from the repository root. These are focused host-build checks; task-specific
compiler/cache/linker/identity controls and .8's final full gates remain required.

## Memory findings

| Track | Category | Entry | Why relevant |
| --- | --- | --- | --- |
| bug | integration | `full-integration-gates-must-select-the-2026-09-04` | .21 must preserve complete selection and refuse accidental zero-case success. |

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `model/design/gate-speed.md` consolidates existing historical gate, stage, binary/test, compile/link and TLC-launch evidence. The owner waived fresh baseline measurements on 2026-10-25; no new baseline gate or serial observer runs are required before optimization. Retain source, backend mode, workload, host/profile conditions and evidence references where available. Errors: missing costs/counts are explicitly unknown, structural predictions are labelled estimates, and differing modes/sources/coverage are never presented as equivalent measured runs. Correctness inventories before test migrations remain required by R4/R5. R6's final measurements remain required. | fn-254-fast-validation-gates-and-a-real-inner.1, fn-254-fast-validation-gates-and-a-real-inner.8 | — |
| R2 | `make model-quick` exists. After an edit to `temporal/feature/activity/model.rs` it re-checks the activity natively and through the interpreter. It runs TLC only when the emitted TLA+ changed, and finishes in under 60 s warm. Errors: a changed emitted module always runs TLC fresh; a cache hit is labelled as such; `model-validate` never uses the cache; a planted stale entry with a different key is ignored. | fn-254-fast-validation-gates-and-a-real-inner.6 | — |
| R3 | The TLC runner checks all properties of a module in one batched run and splits only on failure, with optional parallel property runs. Errors: an equivalence control over every TLC fixture and mutant shows identical verdicts and traces to per-property runs, including a module with two violated properties and a liveness violation under fairness. | fn-254-fast-validation-gates-and-a-real-inner.2 | — |
| R4 | Tests run in parallel across binaries, with integration tests consolidated to one binary per crate where that cuts link time, and a cheaper test debuginfo profile. Errors: the full logical test inventory, including ignored status, is identical before and after under the checked identity mapping above; JVM-backed tests run in a bounded concurrency group. | fn-254-fast-validation-gates-and-a-real-inner.17, fn-254-fast-validation-gates-and-a-real-inner.18, fn-254-fast-validation-gates-and-a-real-inner.20, fn-254-fast-validation-gates-and-a-real-inner.21, fn-254-fast-validation-gates-and-a-real-inner.3, fn-254-fast-validation-gates-and-a-real-inner.4, fn-254-fast-validation-gates-and-a-real-inner.5, fn-254-fast-validation-gates-and-a-real-inner.7 | — |
| R5 | Gate composition: | fn-254-fast-validation-gates-and-a-real-inner.17, fn-254-fast-validation-gates-and-a-real-inner.5 | — |
| R6 | Final warm measurements are recorded beside R1's historical references in `model/design/gate-speed.md`. The native gate timer ported by .15 gains the realistic Model-edit inner loop. Execution targets: `model-quick` under 60 s for an activity edit, `model-check` under 3 min, TLC-required `model-validate` under 10 min. Errors: a missed target is reported with its cause, not relabelled as met; incomparable historical modes/workloads are separated and do not support an attributed speedup claim. | fn-254-fast-validation-gates-and-a-real-inner.8 | — |
| R7 | The framework's checks live in `model/umpire` and run without the Temporal tree. One framework entry point (decided in task .9) runs every framework stage: fmt, clippy, docs and tests of `umpire` crates; the stock, control and Kettle guests; core and Kettle protobuf; the generic and Kettle Go hosts; framework controls. The root `Makefile` keeps only Temporal steps and calls that entry point. Framework tools, pins and caches (TLC/Apalache jars, protoc generators, xtask modules) resolve relative to `model/umpire`, not the Temporal repository root. Errors: a control runs the framework entry point on a copy of `model/umpire` alone, with no `model/temporal` and no root `Makefile`, and it passes; a planted dependency from an `umpire` crate on a Temporal crate, or a Temporal name in `model/umpire`, fails the framework gate. The R7 tool contract also requires a dedicated guest-driver executable independent of the general xtask/gate executable. Guest cache invalidation follows the exact guest driver, current lockfile and compiler inputs. Unrelated general-CLI/gate implementation changes must not select a new guest cache. One admitted context owns framework tool/pin roots, caller workspace/output/target resolution, raw-OS environment encoding and canonical physical-to-logical remaps; commands do not rediscover these independently. Missing local pins or escaping/ambiguous roots refuse explicitly. These requirements retain existing compiler/lock/source admission and cache invalidation for actual guest-driver changes. | fn-254-fast-validation-gates-and-a-real-inner.10, fn-254-fast-validation-gates-and-a-real-inner.11, fn-254-fast-validation-gates-and-a-real-inner.13, fn-254-fast-validation-gates-and-a-real-inner.14, fn-254-fast-validation-gates-and-a-real-inner.15, fn-254-fast-validation-gates-and-a-real-inner.16, fn-254-fast-validation-gates-and-a-real-inner.17, fn-254-fast-validation-gates-and-a-real-inner.19, fn-254-fast-validation-gates-and-a-real-inner.20, fn-254-fast-validation-gates-and-a-real-inner.9 | — |
| R8 | Each gate stage has a committed budget in `model/umpire/dev/gate/budgets.toml` (plus `model/dev/gate/budgets.toml` for Temporal stages), as selected by .9: | fn-254-fast-validation-gates-and-a-real-inner.12 | — |
| R9 | Host build costs are reduced through lighter dev debuginfo in both independent Rust workspaces, measured local compiler-cache reuse across different checkout paths, and a measured mise-pinned mold adoption decision for supported Linux host targets. Errors: effective flags and cache statistics distinguish compilation reuse from Cargo Fresh and final links; incompatible source/compiler/profile/target inputs do not reuse stale compilation; missing pinned tools and cache failures remain visible under the existing fail-hard policy; bootstrap isolation, guest-driver wrapper/compiler/lock identity, optimized release/guest profiles, WASI linking and framework-only operation remain intact. No speedup is claimed without comparable workload evidence. | fn-254-fast-validation-gates-and-a-real-inner.18, fn-254-fast-validation-gates-and-a-real-inner.19, fn-254-fast-validation-gates-and-a-real-inner.20 | — |

## Parked unknowns

- How compile/link time splits against run time inside the 1,707 s `tests` stage (R1 measures it).

### Native tool simplification amendment

2026-10-25, owner requested fixes for the three confirmed accidental-complexity sources in existing tasks. Task .13 extracts a dedicated `umpire-guest-driver` package/executable at `model/umpire/dev/guest-driver`, centralizes admitted tool/workspace context and remap/environment encoding there, and leaves `dev/xtask` as the general command front door. The general CLI may reuse the driver library's context utilities but invokes the driver executable for guest operations; the driver never depends on the CLI or gate implementation. Task .10 composes gates through those interfaces and proves a compiled CLI-only change leaves driver bytes and the guest cache namespace unchanged. Task .11 exercises the same local tools/driver in the real directory-only gate. Preserve the full current lock/compiler inputs and exact driver hash, portable sidecar semantics, raw OS bytes, jobserver/loader inheritance, leases, cleanup and atomic publication. No new tasks, Model semantics, third-party dependencies or broad per-task validation gates are introduced.
