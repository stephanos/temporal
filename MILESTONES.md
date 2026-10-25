# Gomad: WASM Milestones

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](.plans/GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

The WASM backend lives in `tools/gomad/internal/backends/wasm` and shares its
campaign, artifact, replay, and choice owners. Read [the WASM README](tools/gomad/internal/backends/wasm/README.md) before working on that
backend; [the qualification contract](tools/gomad/internal/backends/wasm/QUALIFICATION.md)
lists the gates a shared-owner change must pass.

## Work tracking

Open work follows the [owner compatibility amendment](.flow/artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema and CLI compatibility gates are retired; current behavior, integrity and same-candidate deterministic replay remain required.

Flow specs and tasks own scope, acceptance criteria (R-IDs), blockers, and evidence.
Native linux/amd64 qualification and Linux CI work were deferred under
[fn-128](.flow/specs/fn-128-gomad-deferred-linux-qualification-and.md); native
darwin/arm64 qualification was deferred under
[fn-149](.flow/specs/fn-149-gomad-deferred-darwin-qualification.md). Historical transfers
remain in Flow and Git history. The
[2026-10-14 native hibernation decision](#native-backend-hibernation-owner-decision-2026-10-14)
extended that deferral to shared-owner native execution gates.
The [2026-10-19 retirement decision](#native-backend-retirement-and-the-functional-test-goal-owner-decision-2026-10-19)
retired the native backend. The deferred native specs (fn-105, fn-109, fn-110, fn-112, fn-128, fn-149, fn-155) were closed as won't-do on 2026-10-19.
The native backend was subsequently removed by fn-223. The historical decisions below
record its earlier ownership; they do not prescribe current native execution.
Final changed-candidate runtime, mandatory workload and retained-replay qualification,
including the complete behavioral reach of `make -C tools/gomad test-wasm`, remain
pending under fn-232's original acceptance, executed and reconciled by fn-246 after
fn-242 and fn-240. Native deletion
source checks supply no final qualification pass. `tools/gomad` remains a shared owner.
List only unfinished tasks of each open spec below; completed spec/task records remain in
Flow and Git history. Delete closed-spec worktrees, branches and evidence under the cleanup
policy in AGENTS.md. Remove empty spec sections. Keep descriptions to one line and
spec sections to task tables—no progress prose. Refresh statuses from `flowctl list --json`;
a committed source candidate is not completed acceptance.

Activity: 🚧 Running · ⏸ Parked · ⛔ Blocked · ⬜ Todo.
Running and parked both retain Flow's `in_progress` lifecycle status. Running means
work is executing now; parked means unfinished work with no active execution.
Parking neither completes acceptance nor waives dependencies.

Execution scope (2026-10-22): complete this delivery ladder in dependency order.
The owner's autonomous continuation supersedes the earlier replay-only closeout;
use the documented recommendations when a routine implementation decision is needed.

Work a spec with `/flow-next:work <spec>`; list ready tasks with `flowctl ready`.
Record a deferred task's revival trigger in its owning Flow task before implementation.

## Immediate delivery order

### Cleanup first (owner amendment, 2026-10-24)

Finish the narrow fn-232.30 checkpoint, then complete fn-242's Rust driver implementation
and cutover, then the complete approved fn-240 crate extraction scope.
[fn-246](.flow/specs/fn-246-gomad-final-qualification-after-driver.md) owns the remaining
qualification and independent receiving after both cleanup specs finish. Focused tests,
conformance and review continue throughout implementation.

fn-232 remains open as the original acceptance aggregate. Its .13/.14/.22–.25/.31/.32
obligations transfer explicitly to fn-246, together with fn-242's actual full-cycle
requalification. Local implementation handoff does not prove qualification. Preserve every
original behavioral assertion, bound, identity, custody requirement and receiving obligation.
The historical handovers below describe earlier attempts and ordering.

The global allowance remains two new full qualification dispatches, with zero used at this
amendment. Each actual full driver start counts even if it fails early or aborts; standalone
focused/canary/diagnostic runs and preparation without a driver start do not count. Candidate
changes, output tags and process restarts never reset the allowance. Ambiguous starts remain
charged; missing journal bindings cannot reset history. Recheck resource admission before
dispatch. If both attempts fail, stop stage patching and cycle launches, reassess the
architecture and report a revised approach before resuming.

fn-245 retains its separate recorded scope and ordering; this amendment adds no third cleanup
prerequisite. Existing later stress, sweep, shim and quiet-host measurement obligations remain
pending and retain their original acceptance.

### Throughput order: delete before moving or typing (owner amendment, 2026-10-24)

The 2026-10-24 crate and complexity reviews added three cleanup specs. They are ordered so no
code is ported, moved or typed and then deleted or rewritten:

1. **[fn-242](.flow/specs/fn-242-gomad-shrink-the-qualification.md)** finishes the driver
   port and cutover. The completed analysis-only **fn-249.1** inventory identifies the stages
   that exercise test-only Go models, the `reference-helper`/`conformance` binaries or
   `helper_corpus`, and fn-242 ports those stages minimally. Readers of retained binary formats
   call the `wasmhost` decoders (fn-242.21/.27) rather than re-implementing them.
2. After fn-242.8, two lanes run in parallel:
   - **[fn-249](.flow/specs/fn-249-gomad-retire-the-go-reference-models.md)** retires the
     test-only Go reference models and the stdio helper protocol. fn-249.2 is blocked until
     fn-242.8; the removed stages go to fn-246 as a recorded stage-plan delta.
   - **[fn-248](.flow/specs/fn-248-gomad-linux-and-darwin-parity-for-the.md)** brings Linux and
     Darwin to parity.
3. **[fn-247](.flow/specs/fn-247-gomad-protobuf-host-side-evidence.md)** (after fn-249) moves
   host-side evidence to protobuf and `flate2`, with no Go side to generate.
4. **[fn-240](.flow/specs/fn-240-gomad-context-sized-rust-crates-with.md)** extracts crates
   from the smaller, portable code. Each extraction converts message-prefix dispatch to the
   fn-240.5 kind enum together with its R11 error enums, so fn-245.3 only sweeps leftovers.
5. **[fn-245](.flow/specs/fn-245-gomad-typed-boundaries-for-the-rust-host.md)** types the final
   layout. **[fn-246](.flow/specs/fn-246-gomad-final-qualification-after-driver.md)** qualifies
   once, after fn-240, which now follows fn-247, fn-248 and fn-249. Identities changed by
   fn-247, and CLI message text changed by fn-240's `clap`/`thiserror`, are regenerated once
   before fn-246 binds its candidate.

flowctl only supports task dependencies within a spec. Cross-spec order is therefore
spec-level, plus the fn-249.2 block and the ordering notes in task descriptions.

### Current handover (2026-10-24)

fn-232.13 remains open: the full 96-stage cycle has not passed. The current integration input is
`fn232-composition` at `4e4b8b51a5b65d76868712b588b2ed1067ee82cd`, containing `gomad`
base `318427522b009f5a27eeeb3c7c17b211b0ce2c34`. Its driver suite (567 tests) and full
mandatory worker merge gate exited 0; the independent implementation re-review returned
SHIP. The conductor's full merge gate exited 0 on the same commit, which is now integrated
on `gomad`. Its first cold-cache attempt failed because an offline Go closure lacked modules;
after declared dependency downloads, the unchanged-source full gate passed. These checks
supply no cycle result. The retained launch controller is
`/home/agent/wt/evidence/fn232-resume-launch/launch.py`; it prepares fresh candidate bindings
and runs `driver.py parallel` without a formal freeze ceremony.
Current evidence: `/home/agent/wt/evidence/fn232-resume-composition/`.

The subsequent `resume-4e4-assembly4` attempt assembled all 96 stages successfully, then
failed its first stage, `validate`: the selected Go test passed, but the driver listed all
36 package roots and incorrectly required execution of the 35 unselected roots. The driver
exited 1 with zero passing stage receipts; all attempt processes are terminal. Preserve its
configuration and raw evidence under `/home/agent/wt/evidence/fn232-resume-launch/`.
The remaining closure fixes are split into fn-232.15 (Go selector accounting), fn-232.16
(counterfactual declaration/reach and child configuration), and fn-232.17 (shared harness
initialization). Tasks .15 and .17 may run in parallel; .16 follows .17 because they share
the broad-stage call site. Task .13 retains the complete acceptance gate and waits for all
three fixes. Completed tasks .1–.12 and the separate .14 reach-porting scope are preserved.

All three initial closure fixes are now reviewed, integrated and marked done in Flow. The final
mandatory gate passed on `b15d9b5e0eb0c044612c8fb28db9a701aeea25de`, including all 583
driver tests. The next cycle uses the fresh-output controller
`/home/agent/wt/evidence/fn232-split-closure/launch-next.py`; it authenticates retained
implementation inputs and produces a genuine current-commit R4 bridge-helper receipt.
These checks do not supply a qualification pass. Source remains unchanged during the run.

The C99 run on `69fcf11c13` assembled successfully and reached validation beyond the old
selector failure. It then stopped because the static reader rejected four standard Go
package-level `[no test files]` records. The Go listing/run both exited 0 and all 543 listed
roots ran/passed twice. Task fn-232.18 owns authenticated empty-package accounting while
keeping real skipped-test refusals strict. C99 is retained as INCOMPLETE/exit 1, with zero
passing stage receipts and 95 unstarted stages. No qualification result is inferred.

Task .18 is reviewed, integrated and done. The mandatory gate exited 0 on
`211bdaaacf5ce22fdfb9bbc7beea9f5a41db7f7b`, including 586 driver tests.
The empty-package fix covers both static and ordinary recursive receiving. The next
fresh cycle uses the resulting clean candidate; C99 remains unchanged and incomplete.

The subsequent C0277 cycle on `abbea52d43` is terminal INCOMPLETE/exit1: four stages passed,
three lint stages failed, and 89 did not start. The source defects are repaired by .19,
reviewed and integrated with the full mandatory gate passing on `c6f77620ee` (587 driver
tests). Its Go source edits require genuine fresh SQLite and cooperative installation
admissions; `launch-rebound.py` prepares those before the next full cycle. Prior failures
and their identity/custody ledgers remain preserved; no cycle pass is claimed.

The C2f97 run on `ea26efc078` is terminal INCOMPLETE/exit1: 11 stages passed,
`rust-lint` and `default-build` failed, and 83 did not start. Task .20 repairs the
strict Clippy time-unit finding and the test-only findings exposed by the complete lint matrix, preserving all cases and assertions. The default build succeeded but differed from the
selected binary because the builds embed different Cargo target paths. The next
source-bound cohort will use one fresh target/context for release preparation and
qualification, preserving the exact byte guard and all historical inputs.

Task .20 is integrated and verified done: formatting, all 13 qualification Clippy modes, six focused tests, the combined mandatory gate (587 driver tests), and three-axis SHIP review pass. The next run uses the fresh-cohort controller after a final clean-source preflight; qualification remains pending.

The subsequent C8d0a cycle on `5ef7c85c6d` is terminal INCOMPLETE/exit1:
16 stages passed, workflow-lint failed before actionlint because its selector
required intentionally deleted native smoke, and79 stages did not start. Source
and protected input custody are unchanged. Task .21 repairs the selector and its
module-directory boundary case; it is integrated and verified done after focused
regressions, pinned actionlint, same-session SHIP rereview and the clean mandatory
gate on `0337bda4bb` (589 driver tests). Task .25's independently audited size
classifier correction is integrated; final evidence/ledger acceptance remains open.

The next controller is `/home/agent/wt/evidence/fn232-split-closure/launch-reuse-strict.py`
(SHA6690a6b5044c41dc1c97434319640585bea428a081f2b1844f393fbfb5f21fe4).
It permits only reviewed Python/tracking changes, authenticates unchanged Rust,
native and SQLite inputs while preserving original build provenance, and requires
fresh public/live/install/R4/current-source bindings and a new complete cycle.
It reuses the exact released Cargo context and retains both actual byte checks.
Independent operational audit and clean read-only smoke preflight pass; final-head
preflight and actual execution remain required. The older broad-allowlist launcher
and all failed attempts remain preserved. Tasks .22–.25 own bounded receiving and
closeout; .13 alone produces the cycle and owns final96/145 composition.

The C744b cycle on `d36e998c5c` is terminal INCOMPLETE/exit1:22 stages
passed, `rust-diagnostic-test` failed, and73 did not start, with zero evidence
mismatches. Compilation passed; the actual diagnostic test binary reported
264 passed,16 failed and16 ignored. Profile-neutral linked execution fixtures
retained ordinary conditions instead of selecting their compiled role. Task .26
corrects these test fixtures and the matching latent streaming cases while
preserving strict admission, assertions, bounds and ignored status. Task .26 is done: the five-profile matrix and both canonical merge gates passed
on `5738907d3c`; three fresh Codex review contexts finalized SHIP with no findings.
The repair is integrated. A fresh helper-bound cohort and the next complete cycle
remain required. C744b and all earlier
attempts remain unchanged; no receiving or final qualification pass is claimed.

The C9fc08 cycle on `fb016c995f` is terminal INCOMPLETE/exit1:23 stages passed,
two failed and71 did not start, with zero receipt-binding mismatches. Driver elapsed
2986.593s excludes preparation; the outcome's3554.137s includes assembly. All recorded
source, original807, sealed806 and target-pin inventories remain equal; no owned process
survived. Retain the attempt and terminal audits under
`/home/agent/wt/evidence/fn232-split-closure/resume-closure-cycle8` and `evidence-preparation/`.
The diagnostic integration target exposed two additional role-neutral fixture helpers;
Temporal inspection exposed both a tracked Gitlink inventory error and a1MiB transition-byte
capacity refusal. Tasks .27 and .28 repair the independently diagnosed inventory and
fixture defects; both are integrated, pass worker and conductor gates, and have independent
SHIP reviews. The combined source tree passes the canonical merge gate.
Task .29 retains two matching diagnostic reproductions: repeated pending network polls
exhaust correctly accounted broker transitions after 0.471 seconds of virtual time. Its
diagnosis passed independent SHIP review; .30 owns the polling repair and complete original canary.
After the owner asked whether to raise its trace budget, a separately labelled16MiB diagnostic
completed the original three-minute workflow and retained4718416 native broker transition bytes;
idle calls alone required1060466 bytes. The conductor selected8MiB for this onebox canary with
measured headroom, preserving generic1MiB capacity-refusal tests and all other workload bounds.
Detached replay exposed a producer/reader mismatch between authenticated1154MiB evidence
descriptors and fixed1GiB callers; .30 also owns that targeted repair. The diagnostic is not
qualification. Original timers, retries, scenarios, assertions and receiving obligations remain.
All96 stages and the final145 reader remain required on the repaired
candidate. The deferred .26 review metadata is integrated separately as `c84db120f1`.

The composition preserves the flake/reach fixes, operator cluster-failure checks and
artifact detached/kill-resume families. Children load their driver configuration explicitly;
role reads validate current source, features and compiler context while preserving original
build provenance. Partial dry-run plans may execute bound families; final readers still
refuse missing evidence. Remaining reach gaps, including named unexecuted slim roots, report
under fn-232.14 until its completeness switch is reenabled.

Other local inputs, using the rewritten refs:
- `fn221-rust-sweep` (`911833fb481f8f51d3af6dd20699fc435032c78a`): sweep fixes and fn-221.2 launch plan. Before the full sweep, investigate the historical batch-test slowdown versus Go and exercise resume/verify.
- `fn232-reach-fixes-2` (`1f85fbfcf95604757a6742d54a934e13395c2bc4`): fn-232.14 progress; its handover retains the remaining partial entries and pending full gate.

Next, in order (cleanup-first owner amendment):
1. Complete and review the narrow fn-232.30 checkpoint.
2. Complete fn-242's Rust driver implementation/cutover and focused checks.
3. Complete the full fn-240 crate extraction scope, preserving focused conformance.
4. Execute fn-246: remaining reach and payload fixes, measured workload budgets, fresh final qualification and independent receiving. Reconcile actual proof into the original fn-232/fn-185/fn-235 owners; no missing obligation receives credit from transfer.
5. Resume the remaining sweep, stress and shim windows in their recorded dependency order. fn-245 retains its separately recorded scope; it is not added as another prerequisite by this amendment.

The owner-authorized history cleanup removed the oversized generated `.wire` blobs from all
11 affected local refs. Main and composition file trees were unchanged by that cleanup, and
nothing was pushed. Exact ref mapping, reachability check and backup custody:
`/home/agent/wt/evidence/fn232-history-cleanup/applied.json`. Historical task/evidence SHAs
remain provenance; use the rewritten refs for current commands.

Historical deadline flakes and silent loop exits remain recorded in fn-232.13's handovers.
Capture actual exits/signals on subsequent runs. Timing-only results stay deferred to the
quiet-host window; correctness, replay/determinism, identities and memory/capacity bounds stay binding.

### Opportunistic merge (owner decision, 2026-10-24)

The owner noted that nobody uses Gomad yet and asked whether to "merge opportunistically and fix issues after". They approved the conductor's proposal. It supersedes the freeze requirements in the rulings below where they conflict:
1. **Merging.** Branches merge into `gomad` once they build and the merge gate is green. Known load-sensitive flakes don't block a merge. The gate itself stays.
2. **Closing fn-232.13.** fn-232.13 closes on one green run of the full 96-stage cycle on current `gomad`. There is no formal freeze, no 13-role receipt rebuild, and no re-freeze when the source moves. Evidence binds to the commit that ran it.
3. **Reach-map gaps.** The remaining `gap:` and `partial:` entries move to fn-232.14. Until that task closes, the reach check reports them instead of refusing them. Bugs the cycle run finds are fixed, or filed as tasks.
4. **Post-cycle work.** The fn-221 sweep, stress and the fn-233 rerun start on `gomad` as soon as the cycle is green. Support claims from that cycle (fn-222) are labeled provisional until fn-242's requalification.

### Performance deferred to the quiet-host window (owner decision, 2026-10-23)

The owner asked, "can we merge the blocked work and address any perf issues later when we have the stress test?"

Until the owner declares a quiet-host window, only timing and performance results stop being blocking. They are still measured and recorded:
- the cycle's 3,600-second wall-clock limit, and the stage and cost wall-time budgets;
- fn-185.2/.3 profiles and the optimization verdict;
- stress timing results (stress divergence findings are correctness and stay binding);
- fn-221 sweep results that fail only from load (timeouts, deadline-sensitive inspection fixtures). These are classified as contention and rechecked in that window.

Correctness stays binding: outcomes, replay and determinism, assertions, refusals, memory and capacity bounds, and identities.

Completed work merges as soon as its gate passes. A task closes on its correctness acceptance. Its deferred performance items stay open, recorded in the task with the quiet-host window as revival trigger. The quiet-host window reruns the cycle timing, stress and profiles together, and any performance regression found then becomes a new task.

### fn-232.13 scope rulings (conductor, under the owner's 2026-10-23 delegation)

The owner's goal says to follow the conductor's own recommendations when in doubt. These rulings apply until the owner says otherwise:

1. **Behavioral reach of `test-wasm`.** Map each original test root and named subtest onto the Rust suites, parity lanes, the 39 transferred live roots and the qualification cells.
   - A Go-host-only unit test that checks an implementation detail of deleted Go code is retired with that code. Each retirement gets a one-line reason in the reach map. An example is the helper's `wait4` telemetry.
   - Every user-visible behavior of a retained feature keeps a mapped Rust test with its original assertions and bounds.
   - The 752 original callables are not ported one by one.
   - Once fn-232.14 closes, the reach gate refuses any map entry recorded as `partial:` or `gap:`, so a known gap can never pass as complete reach. Until then it reports them (opportunistic-merge decision). Each such entry is either ported or converted to a mapped entry with an explicit Go-implementation-only reason.
2. **`TestDeterminismExecutedDivergenceArtifacts/handoff`.** Its "every incarnation exits" assertion also fails on the original Go baseline. It is superseded by the fn-235 fail-fast behavior the owner has already authorized. The original failure is kept as history.
3. **One-hour limit.** First time the critical-path stages on the Rust binary. Raise the limit with the owner only if those stages measurably don't fit. Until then the limit stays unchanged.
4. **Cache-path identity.** Compiled files located in the Go cache are recorded by content, not by absolute path. A guest's identity must not depend on the caller's cache location.
5. **Retirement before the freeze.** The .13 retirement change merges before the final freeze, so the frozen candidate and the published support list bind to the retired source. After the freeze, commits that touch only `.flow/**` and Markdown docs keep the frozen candidate. Any other change requires a new freeze.
6. **Go-era cost gates.** These are ported to the Rust host before the freeze, with their original checks and bounds:
   - the warm measurement required for each resident guest;
   - the corpus and SQLite catalogue reader's own peak memory and monotonic CPU, plus the corpus CPU total;
   - the `Matched` comparison of two qualification Results;
   - the initial-preparation compile-flag classification against the actual compiler commands.

   Per-phase helper timings may stay in the campaign metrics side files outside the authenticated Result cost. That loses a record, not a check, and QUALIFICATION.md documents it.
7. **Stress as a correctness check (owner, 2026-10-23).** The fn-188.5 41×4 stress matrix is not a one-hour cycle stage. It runs right after the freeze on the frozen candidate as a determinism check: every same-seed divergence is a finding, whatever the host load. Its wall time and other timing results are recorded but not judged until the quiet-host window. fn-242 does not wait for it.
8. **Pre-freeze source.** fn-227.1's inventory rows cite the Rust map-order tests before the freeze. The pre-freeze dry run also runs the Rust sweep against a real guest and runs fn-233's launcher, shim and Make lanes.
9. **Post-freeze runs.** On the frozen candidate, as correctness work under the current load: the fn-233 formal rerun (~1 h), stress (2–3 h) and the fn-221 full sweep (5–8 h). Load-only timeouts are classified as contention and rechecked later. fn-221 reruns the whole sweep on the frozen Rust binary instead of resuming the Go-candidate journal, because the Go host's helper link-count infrastructure rows and changed evidence bounds make that journal non-transferable. Only the fn-185.2/.3 profiles wait for a quiet-host window.
10. **fn-240 crate count.** The owner's 8–12 crate target stands, even though the main crate has grown to about 110k lines. The read-only survey (.1) may run now. Extractions wait until the post-freeze windows are done, because each one rotates the host and helper identities. Superseded in part on 2026-10-24: fn-240 Amendment 4 fixes the end state at 17 library crates and 3 bins under `rust/crates/`, `rust/bins/`, `rust/tools/`, with `wasm-engine` the only crate exposing wasmtime; the fold order back to twelve is recorded in the spec. The cleanup-first amendment supersedes the wait for post-freeze windows: complete fn-242, then all fn-240 tasks, then fn-246 qualification.
11. **No new Python driver code.** Until fn-242 runs, `tools/gomad/rust/driver/` gets only the Python the fn-232.13 freeze and its post-freeze windows strictly need. New checks go into Rust tests or `gomad` verbs.
12. **fn-222 inside the final cycle.** The one-box fn-222 group runs alongside the inspection stage's own onebox inspection, not after it, and the cycle stays at 96 stages. The cycle is launched with `driver.py parallel`. fn-222.1 then closes on the cycle's gate exits and support list.

### Find bugs early (owner-approved, 2026-10-23)

1. **Merge gate.** Before anything merges into `gomad`, the merged tree must pass the gate command, with stock Go 1.27.1 first on `PATH`:

   ```
   make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism
   ```

   This covers the Go archcheck and support checks, the `gomad` crate tests, the helper engine and transcript corpus in every feature configuration, the determinism inventory, and the driver suite. A change to the interception shim also passes the offline shim harness. Workers run the gate before they report; the conductor reruns it before merging. A red gate blocks the merge.
2. **Dry-run before freezing.** Every final-cycle stage first runs on the current head without the freeze ceremony, and failed stages are rerun one at a time. That surfaces failures and real stage timings early. The formal frozen one-hour cycle runs only after the dry run is green.
3. **A red head stops the line.** The first worker that observes the gate red on `gomad` bisects and fixes it with a regression test before any other merge. Goldens are never regenerated to get a green result.

### Native backend hibernation (owner decision, 2026-10-14)

The owner asked, "why is there a native gate anyway; we can declare the native backend as deferred/hibernating for now". Native backend execution acceptance was deferred while the WASM section was completed. This waiver superseded the earlier exception for newly introduced fn-153/fn-154/fn-179 native gates and inherited native gates. It waived those gates as conditions of WASM/source completion without recording an execution pass or completing native qualification. The ledger below is historical; retirement and removal superseded its revival conditions.

| Deferred execution scope | Linux owner | Darwin owner |
| --- | --- | --- |
| Supported patched-runtime baseline, full native `test-host`, runtime/overlay/clock controls | fn-128.1 | fn-149.1 (runtime/clock), fn-149.2 (full-host) |
| fn-153 native World replay and affected runtime consumers; fn-154 R1–R12 native adapter/descriptor/connection, expiry/dial, model conformance, process/fault, framed-stream and composed generic-I/O replay controls; fn-179 R1–R3 native execution/replay consumers; native execution invalidated by fn-166 moves/rename | fn-128.4 | fn-149.2 |
| Current-candidate reconciliation of those obligations and existing native qualification ledgers | fn-128.7 | fn-149.4 |

The original task selectors, requirements and historical receipts recorded the deferred execution ledger, including both-backend semantics and fn-154's divided payload replay guarantees. Native owners originally required an explicit qualification request, supported host/toolchain/profile and frozen candidate. Stock-Go coverage, cross-builds and historical results supplied no native pass. Those owners later closed as won't-do, and fn-223 removed the native source, tests, fixtures and commands.

Source tasks retained implementation, portable Go coverage (including portable native-package tests), ordinary host-source checks, both-source-set static checks, lint, generated/pin validation, preservation, source review, documentation and all actual WASM execution/replay/resource gates. SQLite selected-state and catalogue storage-crash controls execute WASM and remain required, including projection through `native_crash.go`; their filename does not make them native-backend gates. The historical decision changed acceptance ownership without granting PR, push or CI authority.

Owner execution scope: complete the remaining WASM work in dependency order:

1. ~~Complete fn-189 upstream sync.~~ Done 2026-10-20. The integration base is Temporal `main` at `48a1e5ff025e55b3d4d0f44247423d4c279ff636`, retained by merge `0077477965a5b766b95a667247a9f562c100941b`. Frozen candidate `e50c6476fb` passed qualification, and the matched-release comparison moved to fn-185. Historical details remain in Flow and Git history; closed-spec evidence is deleted under the cleanup policy.
2. ~~Complete fn-225 bounded harness result retention immediately after fn-189.~~ Done 2026-10-20, merged as `76bc9b75f5`. Measurement output roots keep the latest success and one recent failure; see the [spec](.flow/specs/fn-225-gomad-bounded-harness-result-retention.md).
3. ~~Follow with fn-187 helper fixture tests and module ownership.~~ Done 2026-10-20; historical details remain in Flow and Git history.
4. ~~Finish with [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) host-side cost profile, after fn-187.~~ Superseded on 2026-10-19: fn-185 moves below the goal-path work (see the retirement decision below). Extend the moved measurement command and final helper module owners. fn-185 opens no optimization work. This order follows the owner decision on 2026-10-16 to put fn-185 last and fn-187 immediately before it.

### Native backend isolation (owner decision, 2026-10-16)

The owner answered, "yes to isolate native backend". The native backend left the default Gomad host gates (build, vet, lint, `go test ./...`, archcheck default inventory, the WASM and host-tools CI jobs), remaining buildable and testable through one explicit fn-184 selection. That freeze preserved native sources and deferred owners without recording an execution pass. A default build refused native installations instead of falling back to WASM. The later retirement and fn-223 removal superseded the freeze; this decision granted no PR, push or CI authority.

Preserve prior evidence under its original identities, without claiming it qualifies
changed runtime inputs. Qualification resource limits and PR,
push and CI restrictions remain in force.

Owner performance decision (2026-10-13): WASM execution-speed work stays in the Rust
helper, accepting Go→Rust duplication where it buys performance; embedding Wasmtime in Go
(`wasmtime-go`) is rejected. The [2026-10-20 Rust host decision](#rust-host-rewrite-owner-decision-2026-10-20)
supersedes the duplication clause: new host-side logic lands in Rust, not in both.

### WASM performance measurement host (owner decision, 2026-10-17)

The owner answered, "since this is WASM, we don't need a particular env. that was for the native backend." WASM performance measurements may run on any host; no host platform qualifies or disqualifies a WASM result. This supersedes the 2026-10-14 Linux/arm64 requirement. Baseline and candidate are still measured on the same host in the same window with the specified selections and inputs, and every threshold, gain-versus-spread rule, seed, repetition, median/range, correctness, choice/replay, identity and resource-budget requirement still applies; retain host evidence, source/helper/module/engine identities, raw measurements and exact rerun commands. Native platform qualification was deferred under fn-128 and fn-149 before their closure and backend removal.

### Performance checks deferred to fn-185 (owner decision, 2026-10-17)

The owner asked, "can we defer the perf checks to fn-185 and then close out fn-182 and fn-184?" The wall-clock measurements of fn-182 R4 (cold compilation, warm preparation/load, fresh-instance startup, execution wall time and helper peak RSS for full versus slim guests) and fn-184 R5 (build, vet, lint and default test target wall times for the default and selected sets) move to [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) R6, task .2. The matched same-host baseline/candidate contract, repetitions, median/range, gain-versus-spread labels, identities and rerun commands are unchanged. fn-182 and fn-184 close with the byte and count measurements that already exist (module and compiled-cache bytes; package and line counts), recorded in their close-out artifacts. No performance gain is claimed until fn-185 measures it.

### One-hour validation limit (owner decision, 2026-10-19)

The owner decided, "we can't have validation take longer than 1h". Full validation of a candidate (the WASM qualification ladder) must finish within 60 minutes of wall-clock time. The driver runs independent stages in parallel lanes under core and memory caps, gives timing- and deadline-sensitive stages the host to themselves, and fails when wall time exceeds 60 minutes. Only scheduling changes; no stage command, budget, deadline, seed, oracle or assertion is relaxed. Retired native stages leave the ladder, and the matched-release speed comparison moves out of validation into the fn-185 performance window.

### Native backend retirement and the functional-test goal (owner decision, 2026-10-19)

The owner decided, "we want to retire the native backend (not delete yet)", "our goal is definitely to run arbitrary functional tests with gomad (wasm)", and "yes file flow spec for promotion". The native backend initially remained under its fn-184 selection without further qualification, then was removed by [fn-223](.flow/specs/fn-223-gomad-delete-the-native-backend.md). The shared model and the WASM backend stopped depending on native sources by promoting the generated volume and network projections to owned source and relocating the protocol generator into the parent module ([fn-219](.flow/specs/fn-219-gomad-promote-the-model-projections-and.md)); relocating the overlay packages as a retained reference was rejected because nobody would execute them after retirement. The product goal is deterministic execution of arbitrary Temporal functional tests on the cooperative WASM profile. This decision grants no PR, push or CI authority.

Delivery order toward that goal, after fn-189 and fn-187. The owner agreed on 2026-10-19 to move [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) below the goal-path work so it profiles sweep workloads rather than fixtures; this supersedes the 2026-10-16 placement of fn-185 directly after fn-187. fn-185.1's attribution tooling is not gated; fn-185.2 is re-scoped to wait for fn-221.2 and include sweep workloads; the ordering is recorded in the task because Flow task dependencies cannot cross specs.

1. [fn-219](.flow/specs/fn-219-gomad-promote-the-model-projections-and.md) projection promotion and [fn-221](.flow/specs/fn-221-gomad-functional-suite-triage-sweep-on.md) functional-suite triage sweep run in parallel; they touch disjoint code.
2. [fn-220](.flow/specs/fn-220-gomad-one-command-temporal-test-setup.md) one-command Temporal test setup; fn-221 uses its installation if landed, otherwise a throwaway configuration.
3. Refusal-closing specs are filed from fn-221's ranked classification once the relevant cause is established; none is filed in advance. Unmeasured timeout and capacity cases remain unresolved, not unsupported, until fn-185 supplies the cost evidence.
   [fn-224](.flow/specs/fn-224-gomad-ringpop-membership-on-wasm-within.md) ringpop membership within reason is already filed and follows the sweep; its unstub gate may land earlier if the sweep wants it.
4. [fn-222](.flow/specs/fn-222-gomad-qualify-the-one-box-cell-and.md) qualifies the one-box cell and defines the per-test support claim, after fn-220.
5. [fn-188](.flow/specs/fn-188-gomad-wasm-determinism-boundary-audit.md) determinism boundary audit continues alongside; it must hold for a testcore-sized guest.
6. [fn-223](.flow/specs/fn-223-gomad-delete-the-native-backend.md) deletes the native backend after fn-219; fn-128 and fn-149 are already closed as won't-do.
7. [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) task .1's attribution tooling continues in the helper's diagnostic profile. Tasks .2/.3 profile the Go host side and are re-scoped by the [2026-10-20 Rust host decision](#rust-host-rewrite-owner-decision-2026-10-20) to measure the Rust host after fn-232; they are not run against the Go host. Bring targeted profiling forward only for a demonstrated performance blocker; no performance gain is claimed before measurement.

### Rust host rewrite (owner decision, 2026-10-20)

The owner decided, "let's schedule the rewrite to Rust now as a flow spec", and "we don't need byte-for-byte parity with existing evidence. that can be broken." [fn-232](.flow/specs/fn-232-gomad-rewrite-the-host-side-in-rust.md) rewrites the Go host side (CLI, Runner, evidence store, qualification, installation) in Rust as one binary that links the helper crate and runs guests in-process. The Rust binary starts a new record schema, storage lineage and installation manifest version; Go-lineage artifacts, campaigns, checkpoints and manifests are refused, never migrated. The gate is behavioral equivalence on fresh executions (outcome class, failure grouping, replay divergence ordinal and reason, accepted minimization reductions, qualification verdict), not byte parity. Go stays for the toolchain overlay, the reference models, the protocol generator and the conformance corpora. Performance and code size were explicitly rejected as justifications. This decision grants no PR, push or CI authority.

[fn-228](.flow/specs/fn-228-gomad-helper-owned-replay-tape.md) moves replay-tape projection and validation into the helper library first; fn-232 depends on it. The choice-site path, [fn-229](.flow/specs/fn-229-gomad-symbolize-choice-sites-in-inspect.md) symbolization in inspect, [fn-230](.flow/specs/fn-230-gomad-explore-site-scope-for-choice.md) explore site scope and [fn-231](.flow/specs/fn-231-gomad-site-aware-minimizer-ordering-and.md) minimizer ordering, interleaves with the Rust host after its inspect, exploration and minimize tasks respectively. These features do not wait for the entire rewrite or build a second Go-host implementation.

The owner then decided, "let's pull fn-232 forward to avoid any double work," and on the same day merged the existing specs into one order so nothing is built on the Go host and again on the Rust host. The merged delivery order:

1. [fn-232](.flow/specs/fn-232-gomad-rewrite-the-host-side-in-rust.md) tasks .1 to .4 (binary crate, evidence layer, store, harness) with [fn-228](.flow/specs/fn-228-gomad-helper-owned-replay-tape.md) as the first batch. In parallel: [fn-223](.flow/specs/fn-223-gomad-delete-the-native-backend.md) as pure deletion; [fn-221](.flow/specs/fn-221-gomad-functional-suite-triage-sweep-on.md) .2/.3 sweep and [fn-222](.flow/specs/fn-222-gomad-qualify-the-one-box-cell-and.md) .2 support list on the Go binary, since their results are guest knowledge that carries over; the carry-over parts of fn-219.4 (ledger, docs), fn-227.1 (regression tests, inventory rows) and fn-188.5 (stress report).
2. fn-232.5 inspect in Rust, then [fn-229](.flow/specs/fn-229-gomad-symbolize-choice-sites-in-inspect.md) symbolization. Its artifact projection lands first; campaign divergence R2 is received by fn-232.7/.8 and keeps fn-229.2 open until the real campaign consumer is wired and tested. This follows the provisional-development batch policy, not an R2 waiver.
3. fn-232.6 to .8 replay and exploration in Rust; fn-227 R1/R2 (shutdown and crash-validation ordering) are implemented once in fn-232.8; then [fn-230](.flow/specs/fn-230-gomad-explore-site-scope-for-choice.md).
4. fn-232.9 minimize in Rust, then [fn-231](.flow/specs/fn-231-gomad-site-aware-minimizer-ordering-and.md).
5. fn-232.10 qualify and installation-free report in Rust, then fn-222.1 one-box qualification on the Rust binary.
6. fn-232.11 to .13 install and Go host retirement; fn-188.6 gate wiring lands in .13; then fn-185.2/.3 against the Rust host.
   [fn-233](.flow/specs/fn-233-gomad-intercept-the-go-command-so-make.md) go-command interception (`gomad -- make …`, `gomad -- go test …`) follows fn-232.7 and .11 as Rust verbs; it adds no Go host work and no new verb grammar beyond `run-guest` and the `--` wrapper.
7. [fn-224](.flow/specs/fn-224-gomad-ringpop-membership-on-wasm-within.md) ringpop and refusal-closing specs filed from the sweep run whenever the sweep wants them and land in the Rust verb that owns the behavior; a sweep-blocking host fix before that verb ships is the smallest Go change that unblocks the sweep, with its R-ID noted for the Rust verb.

No other Go host-side work starts. The fn-222.2 Go support-list phase records only proven rows or pending guest dispositions; final Rust support requires fn-222.1's complete gates after fn-232.10. Re-homing notes are recorded in tasks fn-227.1, fn-188.6 and fn-222.1; Flow task dependencies cannot cross specs, so this list is the ordering record. Native deletion retains its fn-219-closed start gate.

## Batch-first execution strategy

Owner decision (2026-10-10). Optimize milestone execution for implemented and verified
capabilities per hour. Accept temporary breaking changes and integration risk to finish larger
coherent batches. This strategy supersedes older per-task green-start, source-admission packet,
review, commit and evidence-duplication requirements in milestone specs/tasks. Functional
acceptance, explicit qualification prerequisites, the immediate delivery order and the
safety constraints below remain binding. Reconcile conflicting Flow orchestration metadata
once when selecting a batch; preserve historical results and their meaning.

1. **Batch the whole repair.** Default to 4-8 related existing tasks or one complete subsystem
   migration, adjusting for actual ownership and coupling. Group repeated Runner preparation,
   cleanup and lint repairs by affected subsystem. Use existing Flow owners and R-IDs; create
   a new owner only for a genuinely uncovered requirement. Record the batch's outcome, write
   surfaces, integration dependencies and final checks once. Tiny call-site fixes do not each
   need another research fan-out, spec, plan review or admission packet.
2. **Parallelize implementation.** Dispatch independent ownership surfaces together and keep
   available agents working on code, tests, consumer migrations or necessary research. Use
   isolated worktrees for concurrent writers. A dependent surface may develop against a
   declared provisional interface while its predecessor is implemented; integrate in dependency
   order and prove the required contracts before completion. The native hibernation
   amendment is historical after fn-223 removal; a missing external checkout still holds. Put
   tightly coupled edits in one worker's batch
   instead of making artificial parallel tasks that collide in the same file.
3. **Allow intermediate breakage.** Compile failures, red tests/lint and temporary API/format
   incompatibilities are acceptable inside the batch. Record failures, finish the coordinated
   changes, update consumers and regenerate outputs before final acceptance. Preserve intended
   final capabilities and safety contracts; tests must verify the intended behavior rather than
   conceal a regression. Whole-file byte reconstruction and exhaustive environment/manifest
   sealing are not default requirements for ordinary test or refactoring batches. Use them
   only where a concrete identity, replay, qualification or preservation contract requires them.
4. **Validate the integrated batch once.** Run focused regressions during development. On the
   frozen integrated candidate, run affected lint and package/boundary checks, plus
   `make -C tools/gomad validate` when generator, protocol or toolchain inputs changed. The
   native full-host gate was removed with its backend. Retain portable constituents
   and actual WASM gates; identify portable checks as portable. After a
   small review fix, rerun affected checks; repeat broad gates only for changed shared inputs
   or a concrete uncovered risk. Documentation-only edits need document/diff checks.
5. **Review by batch; commit after every task.** Obtain one independent review of the
   integrated batch, covering its task/R-ID mapping, then address actionable findings together.
   After each task is complete, create a separate local commit containing its implementation,
   tests, documentation, Flow records and milestone updates. Stage only task-owned changes;
   leave unrelated work untouched. Local checkpoints during a temporarily broken batch are
   allowed. Per-task reviews and green baselines are not start barriers for the next edit
   within that batch. Complete each Flow task only when the batch evidence proves every
   requirement it still owns. A required gate that remains red or unavailable stays open;
   it does not halt unrelated implementation batches.
6. **Keep one evidence handover.** Record candidate revision, relevant tool versions, commands,
   numeric exits, meaningful regression results, remaining failures and the independent review
   verdict in one batch handover. Map each existing task/R-ID to that evidence. Reference raw
   logs and prior valid results rather than copying reports into successive verification
   packets. Add capture machinery only to answer a specific unresolved risk. Keep bulk traces,
   binaries and scratch snapshots local unless delivery requires them.

### Orchestration

- **Root ownership.** The conductor selects batches, defines ownership and interfaces,
  dispatches workers, integrates changes and owns Flow completion. Keep decisions and evidence
  pointers in the root context; route implementation and review fixes to workers.
- **Worker ownership.** Assign a fresh implementation worker per coherent batch or disjoint
  ownership surface. Reuse that worker across related tasks and review fixes. Workers may
  delegate independent surfaces; keep one committer per mutable worktree. Independent reviewers
  assess integrated changes; workers cannot certify their own acceptance.
- **Shared resources.** Parallelize independent Go/build/lint checks when isolated worktrees,
  temporary directories and cache/output ownership prevent interference. Serialize writes to
  the same files, shared generation outputs and target integration. Freeze the source checked
  by each command and bind final batch results to the integrated candidate.
- **Throughput feedback.** If planning, repeated gates or evidence paperwork dominates a batch,
  consolidate it before dispatching more tiny tasks. Report completed capabilities, remaining
  failures and the next batch; commit counts and artifact volume are not progress metrics.

Apply this division of work within the immediate delivery order, Flow rules, verification
requirements above. It grants no additional qualification,
CI, PR, or push authority.

## Constraints

- **No policy widening.** Gomad never grants `syscall`, `os/exec`, `os/signal`, or
  `golang.org/x/sys` generically. The removed native backend admitted exact compatibility
  packs bound to a module version, go.sum hash, per-file SHA-256, owner and workload.
  Those historical approvals do not admit WASM imports or host access.
- **No source translation and no test rewriting.** Determinism comes from the patched
  toolchain and the reviewed boundary. A Temporal test that needs a Gomad-specific overlay of
  its own source, as gomad1 required, is a blocker to record, never a fix to ship.
- **Fail-closed stays.** An unmodeled boundary operation terminates the process. Work that needs
  a new modeled operation adds it with a semantic contract, a resource bound, transcript
  coverage, exact replay, and a negative test, per COMPAT-5 in
  [the compatibility roadmap](.plans/GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
- **Evidence over narration.** Work is done when its command produces the stated report on a
  clean checkout. A passing local run that depends on untracked state does not count.
- **Platform.** The removed native backend's boundary manifest named `darwin/arm64` and
  `linux/amd64`; its sandbox, DTrace and compatibility-pack gates are historical.
  WASM qualification binds the recorded engine, host, profile and workload under
  [the qualification contract](tools/gomad/internal/backends/wasm/QUALIFICATION.md).
- **Server source changes are allowed but bounded.** A change under `common`, `service`,
  `temporal`, or `tests/testcore` is acceptable when it isolates an optional provider behind a
  build tag or an injection seam and the default build is unchanged. A change that alters
  runtime behavior for production builds requires dedicated production review and
  regression evidence.
- **Validation scope.** The full `./tests` set is not run as a gate; a change is validated on the
  smoke selection plus the suites it affects. The native `gomad-tests-qualification`
  command was removed with its backend; retained WASM gates follow the qualification contract.
- **Qualification identity.** Reports bind exact source and toolchain identities;
  pre-integration results do not qualify a combined candidate, and Darwin results do
  not substitute for Linux gates. An unavailable host or unexplained regression leaves
  acceptance incomplete in the task that owns the requirement. Historical native Linux and Darwin
  requirements closed as won't-do under fn-128 and fn-149 without recording a pass.
- **Patch-policy boundary.** The collector-file prohibition remains in force.
  Overwriting collector-owned GC stamps requires an explicit patch-policy owner decision.
- **Preservation.** Keep CLI grammar/defaults, recorded formats, fixed-identity
  canonical bytes, error precedence, existing comments, fresh processes, guest timer
  ownership, and separate seed/round/corpus transactions unless the owning spec
  explicitly authorizes a change.
- **Qualification claims.** Same-seed repeatability does not establish choice-tape
  exact replay; retain each workload's recorded disposition and replay requirements.
- **Feature preservation.** Removing or freezing shipped features requires an explicit
  owner decision; cleanup must not silently drop capabilities.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad/internal/model/cluster`. Those are the simulation track and cannot host `testcore`.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](.plans/GOMAD_CMP.md) records why.

## Runtime limits

- GC timing remains uncontrolled; a replay divergence needs a reproduced cause,
  not a speculative collector or topology change.
- Finite qualification and forced replay choices do not establish an unconditional
  determinism guarantee; retained soak cohorts supply measured statistical bounds.
- Spin loops with non-blocking selects stall virtual time; pollers must back off.
- Every Go bump requires porting and requalification. Dependency bumps affecting
  packed or adapted modules invalidate their pins and reopen the capability closure.

## Open specs

Listed by spec dependencies and the current delivery order (2026-10-24): fn-242 → (fn-249 ∥ fn-248) → fn-247 → fn-240 → (fn-245 ∥ fn-246) → fn-232 acceptance closeout. Deferred work follows the active ladder.

### Qualification machinery cleanup — [fn-242](.flow/specs/fn-242-gomad-shrink-the-qualification.md)

Owner cleanup-first amendment 2026-10-24: starts after the narrow fn-232.30 checkpoint, before the complete fn-240 extraction. Each retained check names a failure it catches; the Python cycle driver becomes a Rust `gomad` verb. Local completion proves implementation/cutover and focused handoff. Actual full-cycle qualification remains pending in fn-246 after both cleanup specs.

Execute this spec through [16 owner-authorized batches](.flow/artifacts/fn242-execution-batches.md): six coding lanes, two heavy build slots, shared genuine fixtures and one review per batch; final cutover receives three-axis review. Original task acceptance and proof ownership remain binding.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-242.24](.flow/tasks/fn-242-gomad-shrink-the-qualification.24.md) | 🚧 Running | Validate actual guest and preparation observations |
| [fn-242.25](.flow/tasks/fn-242-gomad-shrink-the-qualification.25.md) | ⬜ Todo | Validate inspection decoding, search and refused mappings |
| [fn-242.35](.flow/tasks/fn-242-gomad-shrink-the-qualification.35.md) | ⬜ Todo | Validate installed inspection and recovery evidence |
| [fn-242.26](.flow/tasks/fn-242-gomad-shrink-the-qualification.26.md) | ⬜ Todo | Validate retained inputs, presence and mutation matrix |
| [fn-242.34](.flow/tasks/fn-242-gomad-shrink-the-qualification.34.md) | ⬜ Todo | Validate mandatory Results, SQLite selections and Temporal histories |
| [fn-242.45](.flow/tasks/fn-242-gomad-shrink-the-qualification.45.md) | ⬜ Todo | Validate durable cells and broad closure readback |
| [fn-242.20](.flow/tasks/fn-242-gomad-shrink-the-qualification.20.md) | ⬜ Todo | Validate runtime artifacts and forced replay aggregates |
| [fn-242.22](.flow/tasks/fn-242-gomad-shrink-the-qualification.22.md) | ⬜ Todo | Implement retained, operator and ringpop final sections |
| [fn-242.23](.flow/tasks/fn-242-gomad-shrink-the-qualification.23.md) | ⬜ Todo | Validate native command observations and role custody |
| [fn-242.28](.flow/tasks/fn-242-gomad-shrink-the-qualification.28.md) | ⬜ Todo | Execute static and broad adapters with raw publication |
| [fn-242.31](.flow/tasks/fn-242-gomad-shrink-the-qualification.31.md) | ⬜ Todo | Execute installed lifecycle, interception and ringpop adapters |
| [fn-242.29](.flow/tasks/fn-242-gomad-shrink-the-qualification.29.md) | ⬜ Todo | Build and authenticate CLI and broad harness producers |
| [fn-242.40](.flow/tasks/fn-242-gomad-shrink-the-qualification.40.md) | ⬜ Todo | Execute primary, runtime, cell and corpus adapters |
| [fn-242.41](.flow/tasks/fn-242-gomad-shrink-the-qualification.41.md) | ⬜ Todo | Execute identity and divergence harnesses |
| [fn-242.30](.flow/tasks/fn-242-gomad-shrink-the-qualification.30.md) | ⬜ Todo | Execute workload, inspection, recovery and support adapters |
| [fn-242.32](.flow/tasks/fn-242-gomad-shrink-the-qualification.32.md) | ⬜ Todo | Execute frozen reference profile and public role preparation |
| [fn-242.33](.flow/tasks/fn-242-gomad-shrink-the-qualification.33.md) | ⬜ Todo | Execute independent SQLite oracle and terminal controls |
| [fn-242.39](.flow/tasks/fn-242-gomad-shrink-the-qualification.39.md) | ⬜ Todo | Validate terminal sidecars, compiler inputs, allocation and cost |
| [fn-242.42](.flow/tasks/fn-242-gomad-shrink-the-qualification.42.md) | ⬜ Todo | Validate frozen reference profile and public role readback |
| [fn-242.43](.flow/tasks/fn-242-gomad-shrink-the-qualification.43.md) | ⬜ Todo | Validate independent SQLite catalogue and oracle readback |
| [fn-242.46](.flow/tasks/fn-242-gomad-shrink-the-qualification.46.md) | ⬜ Todo | Validate current host, terminal attribution and profile resources |
| [fn-242.48](.flow/tasks/fn-242-gomad-shrink-the-qualification.48.md) | ⬜ Todo | Validate diagnostic, release and frozen reference preservation |
| [fn-242.37](.flow/tasks/fn-242-gomad-shrink-the-qualification.37.md) | ⬜ Todo | Validate installed cluster and interception readback |
| [fn-242.38](.flow/tasks/fn-242-gomad-shrink-the-qualification.38.md) | ⬜ Todo | Validate original clusters, ringpop, counterfactual and ordinary evidence |
| [fn-242.47](.flow/tasks/fn-242-gomad-shrink-the-qualification.47.md) | ⬜ Todo | Validate public verb lanes, identity and executed divergence |
| [fn-242.44](.flow/tasks/fn-242-gomad-shrink-the-qualification.44.md) | ⬜ Todo | Validate live families and transferred behavioral semantics |
| [fn-242.49](.flow/tasks/fn-242-gomad-shrink-the-qualification.49.md) | ⬜ Todo | Validate actual reach proof and ordered two-cell support |
| [fn-242.50](.flow/tasks/fn-242-gomad-shrink-the-qualification.50.md) | ⬜ Todo | Compose real retained reader sections and validate combined readback |
| [fn-242.51](.flow/tasks/fn-242-gomad-shrink-the-qualification.51.md) | ⬜ Todo | Close final partitions and prove cluster admission edges |
| [fn-242.52](.flow/tasks/fn-242-gomad-shrink-the-qualification.52.md) | ⬜ Todo | Integrate actual routes and prove bounded preparation through readback |
| [fn-242.2](.flow/tasks/fn-242-gomad-shrink-the-qualification.2.md) | ⛔ Blocked | Rust cycle verb: DAG, resource scheduler, serial and parallel dispatch |
| [fn-242.3](.flow/tasks/fn-242-gomad-shrink-the-qualification.3.md) | ⬜ Todo | Port kept core, receiving and cell readers into the verb |
| [fn-242.4](.flow/tasks/fn-242-gomad-shrink-the-qualification.4.md) | ⬜ Todo | Port kept final readers: profiles, retained, terminal, reach, onebox, cost |
| [fn-242.9](.flow/tasks/fn-242-gomad-shrink-the-qualification.9.md) | ⬜ Todo | Rust candidate preparation/configuration composition with focused genuine-input proof |
| [fn-242.10](.flow/tasks/fn-242-gomad-shrink-the-qualification.10.md) | ⬜ Todo | Rust producer adapters; preserve leaf commands and prove replacement contracts |
| [fn-242.5](.flow/tasks/fn-242-gomad-shrink-the-qualification.5.md) | ⬜ Todo | Parity-lane disposition; preserve baseline custody for final receiving |
| [fn-242.7](.flow/tasks/fn-242-gomad-shrink-the-qualification.7.md) | ⬜ Todo | Zero-gap proof audit and cutover: remove Python, repoint commands/inventory, record sizes |
| [fn-242.8](.flow/tasks/fn-242-gomad-shrink-the-qualification.8.md) | ⬜ Todo | Focused implementation handoff; actual qualification transferred to fn-246 |

### Retire Go reference models and the stdio helper protocol — [fn-249](.flow/specs/fn-249-gomad-retire-the-go-reference-models.md)

Filed 2026-10-24 from the complexity review. Production runs the engine in-process through `linked`; the test-only Go models (~37k production + 28k test lines of Go) and the stdio frame path only serve conformance tests. Task .1 runs during fn-242; code tasks wait for fn-242.8.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-249.2](.flow/tasks/fn-249-gomad-retire-the-go-reference-models.2.md) | ⛔ Blocked | Capture Go oracle outputs as Rust corpus goldens (blocked until fn-242.8) |
| [fn-249.3](.flow/tasks/fn-249-gomad-retire-the-go-reference-models.3.md) | ⬜ Todo | Delete the test-only Go reference models |
| [fn-249.4](.flow/tasks/fn-249-gomad-retire-the-go-reference-models.4.md) | ⬜ Todo | Delete the stdio helper protocol path; in-process `linked` API only |
| [fn-249.5](.flow/tasks/fn-249-gomad-retire-the-go-reference-models.5.md) | ⬜ Todo | Update stage plan, Make, docs and inventories; record the stage delta for fn-246 and sizes |

### Linux and Darwin parity — [fn-248](.flow/specs/fn-248-gomad-linux-and-darwin-parity-for-the.md)

Filed 2026-10-24. Near-identical code on both platforms: one `host` module for OS queries, `libc` for every C binding, `signal-hook`, and a portable supervisor. The owner accepted Darwin's weaker descendant custody (no subreaper) for development runs. Runs after fn-242, in parallel with fn-249.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-248.1](.flow/tasks/fn-248-gomad-linux-and-darwin-parity-for-the.1.md) | ⬜ Todo | Make `gomad` build and test on macOS |
| [fn-248.2](.flow/tasks/fn-248-gomad-linux-and-darwin-parity-for-the.2.md) | ⬜ Todo | One host module for OS queries |
| [fn-248.3](.flow/tasks/fn-248-gomad-linux-and-darwin-parity-for-the.3.md) | ⬜ Todo | Replace hand-written C bindings with `libc` and `signal-hook` |
| [fn-248.4](.flow/tasks/fn-248-gomad-linux-and-darwin-parity-for-the.4.md) | ⬜ Todo | Portable cycle supervisor on `libc` with a per-OS custody backend |

### Protobuf host-side evidence and flate2 — [fn-247](.flow/specs/fn-247-gomad-protobuf-host-side-evidence.md)

Filed 2026-10-24. Host-only binary evidence (World snapshot, network evidence, volume image, storage evidence, heap profile) moves to `prost`; `flate2` replaces the Go deflate port. Guest-boundary wires stay custom. Runs after fn-249, so there is no Go side; changed identities are regenerated once before fn-246.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-247.1](.flow/tasks/fn-247-gomad-protobuf-host-side-evidence.1.md) | ⬜ Todo | Inventory custom binary formats; set up `prost`/`protox` generation |
| [fn-247.2](.flow/tasks/fn-247-gomad-protobuf-host-side-evidence.2.md) | ⬜ Todo | Replace the Go deflate port and hand-written gzip with `flate2` |
| [fn-247.3](.flow/tasks/fn-247-gomad-protobuf-host-side-evidence.3.md) | ⬜ Todo | Move World snapshot evidence to protobuf |
| [fn-247.4](.flow/tasks/fn-247-gomad-protobuf-host-side-evidence.4.md) | ⬜ Todo | Move network evidence (GMNETEV1) to protobuf |
| [fn-247.5](.flow/tasks/fn-247-gomad-protobuf-host-side-evidence.5.md) | ⬜ Todo | Move volume image and storage evidence to protobuf |
| [fn-247.6](.flow/tasks/fn-247-gomad-protobuf-host-side-evidence.6.md) | ⬜ Todo | Build the heap profile from pprof protobuf messages |
| [fn-247.7](.flow/tasks/fn-247-gomad-protobuf-host-side-evidence.7.md) | ⬜ Todo | Regenerate changed identities and goldens once |

### Context-sized Rust crates — [fn-240](.flow/specs/fn-240-gomad-context-sized-rust-crates-with.md)

Target layout fixed by the owner on 2026-10-24 (spec Amendment 4); R11 typed error enums and R12 size lints ride on every extraction. Survey landed 2026-10-23 (`.flow/artifacts/fn240-crate-survey-2026-10-23.md`); extractions follow fn-242, fn-249, fn-248 and fn-247. Each extraction also converts that crate's message-prefix dispatch to the fn-240.5 kind enum. `clap` (R9) and `thiserror` (R11) are adopted here.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-240.1](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.1.md) | ⏸ Parked | Survey done; `cargo build --timings` baseline still to record |
| [fn-240.5](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.5.md) | ⬜ Todo | Foundations: `HostError`, identity closure with flip-byte test, inventory scanner, workspace lints and allow-list |
| [fn-240.2](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.2.md) | ⬜ Todo | Extract `gomad-evidence` with boundary tests (three listed back-edges first) |
| [fn-240.3](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.3.md) | ⬜ Todo | Extract `wasm-sim` (clock, network, volume, event_order, sim controller) |
| [fn-240.4](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.4.md) | ⬜ Todo | Measure and decide go/no-go for continuing to the Amendment 4 layout |
| [fn-240.6](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.6.md) | ⬜ Todo | Hooks instead of features: `wasm-hooks`, `wasm-diagnostic`, `gomad-fixture` |
| [fn-240.7](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.7.md) | ⬜ Todo | Extract `wasm-model` and the `wasm-conformance` bin; drop the conformance feature |
| [fn-240.8](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.8.md) | ⬜ Todo | Split `wasm-engine` from `wasm-cluster`; `wasmhost` helper bin |
| [fn-240.9](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.9.md) | ⬜ Todo | Extract `gomad-install` and `gomad-execution`; remove cli back-edges |
| [fn-240.10](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.10.md) | ⬜ Todo | Extract `gomad-replay`, `gomad-qualify`, `gomad-sweep`, `gomad-shim` |
| [fn-240.11](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.11.md) | ⬜ Todo | Extract `gomad-campaign` |
| [fn-240.12](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.12.md) | ⬜ Todo | Split `gomad-measure` and `gomad-stress` |
| [fn-240.13](.flow/tasks/fn-240-gomad-context-sized-rust-crates-with.13.md) | ⬜ Todo | Directory relayout, dispatch-only `bins/gomad`, Makefile `--workspace`, final measurement |

### Final qualification after cleanup — [fn-246](.flow/specs/fn-246-gomad-final-qualification-after-driver.md)

Runs after fn-242 and the complete fn-240 scope. Original acceptance remains pending until actual final-candidate proof and receiving reconciliation.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-246.1](.flow/tasks/fn-246-gomad-final-qualification-after-driver.1.md) | ⬜ Todo | Close remaining behavioral reach and restore strict admission |
| [fn-246.2](.flow/tasks/fn-246-gomad-final-qualification-after-driver.2.md) | ⬜ Todo | Audit every literal payload bound and prove boundary behavior |
| [fn-246.3](.flow/tasks/fn-246-gomad-final-qualification-after-driver.3.md) | ⬜ Todo | Measure SQLite/multi-instance World budgets and pass standalones |
| [fn-246.4](.flow/tasks/fn-246-gomad-final-qualification-after-driver.4.md) | ⬜ Todo | Prepare the final candidate and produce the complete qualification cycle |
| [fn-246.5](.flow/tasks/fn-246-gomad-final-qualification-after-driver.5.md) | ⬜ Todo | Independently receive runtime, workloads, retained/operator and ringpop proof |
| [fn-246.6](.flow/tasks/fn-246-gomad-final-qualification-after-driver.6.md) | ⬜ Todo | Independently receive R4, public18, retirement, custody and size proof |
| [fn-246.7](.flow/tasks/fn-246-gomad-final-qualification-after-driver.7.md) | ⬜ Todo | Audit coverage and reconcile actual proof into original acceptance owners |

### Rust host rewrite — [fn-232](.flow/specs/fn-232-gomad-rewrite-the-host-side-in-rust.md)

Original acceptance closes from fn-246 receiving after fn-242 and fn-240. New evidence lineage; behavioral equivalence gate; Go overlay, models, generator and corpora stay.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-232.22](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.22.md) | ⛔ Blocked | Receive runtime and mandatory workload evidence |
| [fn-232.23](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.23.md) | ⛔ Blocked | Receive retained/operator replay and ringpop proof; reconcile fn235 |
| [fn-232.24](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.24.md) | ⛔ Blocked | Receive R4 preservation proof; reconcile fn185 |
| [fn-232.25](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.25.md) | ⛔ Blocked | Retirement/public18 parity and common identity/size/docs ledger; grouping correction gated separately |
| [fn-232.31](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.31.md) | ⛔ Blocked | Measure sqlite/multi-instance World transition budgets under enforced recording before the cycle reaches them; .13 depends on it |
| [fn-232-gomad-rewrite-the-host-side-in-rust.13](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.13.md) | ⛔ Blocked | Original final acceptance; execution/receiving transferred to fn-246 after cleanup |
| [fn-232-gomad-rewrite-the-host-side-in-rust.14](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.14.md) | ⛔ Blocked | Original reach acceptance; fn-246.1 restores strict admission before final qualification |
| [fn-232.32](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.32.md) | ⛔ Blocked | Unify literal payload read bounds with authenticated limits (same class as the .30 replay refusal) |
| [fn-232.33](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.33.md) | ⬜ Todo | producer.py installs without `--compile-cache`; warm-resident budget refuses storage, network and clusters cells (found standalone 2026-10-24; fix proven) |

Historical closure plan, superseded by the cleanup-first amendment above: tasks .21–.26 were closure predecessors of the already-running .13. The conductor
launches one fresh cycle after the integrated .21 and .26 repairs pass; .22–.25 prepare independently and finish
from its actual outputs. They do not depend on .13 being done. All original
acceptance and post-cycle ordering remain unchanged; .13 owns final composition.
Cycle5 failed before actionlint because native smoke was intentionally deleted
but the harness still required it. Its16 passing receipts, failed receipt,
79 unstarted stages and unchanged source/protected custody remain historical.

### Go-command interception — [fn-233](.flow/specs/fn-233-gomad-intercept-the-go-command-so-make.md)

Owner request 2026-10-20: a Go one-liner and `gomad -- make <target>` interception. Rust verbs after fn-232.7 and .11; intercepted evidence is marked `origin: intercepted-go` and never qualification evidence.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-233-gomad-intercept-the-go-command-so-make.1](.flow/tasks/fn-233-gomad-intercept-the-go-command-so-make.1.md) | 🚧 Running | run-guest launcher verb behind go -exec with intercepted-origin evidence |
| [fn-233-gomad-intercept-the-go-command-so-make.2](.flow/tasks/fn-233-gomad-intercept-the-go-command-so-make.2.md) | ⬜ Todo | go shim: argv[0] dispatch, classification, guest build environment, refusals, host passthrough |
| [fn-233-gomad-intercept-the-go-command-so-make.3](.flow/tasks/fn-233-gomad-intercept-the-go-command-so-make.3.md) | ⬜ Todo | gomad -- wrapper, gomad.toml and auto-derived installation |
| [fn-233-gomad-intercept-the-go-command-so-make.4](.flow/tasks/fn-233-gomad-intercept-the-go-command-so-make.4.md) | ⬜ Todo | Temporal Makefile through the wrapper and operator documentation |

### WASM determinism boundary audit — [fn-188](.flow/specs/fn-188-gomad-wasm-determinism-boundary-audit.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-188.5](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.5.md) | ⏸ Parked | Stress producer and assembler ready; full matrix runs right after the freeze as a correctness check |
| [fn-188.6](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.6.md) | ⏸ Parked | Regression gates, version pinning, validate wiring and the documented contract |

### Map-order determinism repairs — [fn-227](.flow/specs/fn-227-gomad-close-wasm-map-order-determinism.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-227.1](.flow/tasks/fn-227-gomad-close-wasm-map-order-determinism.1.md) | ⏸ Parked | Sort live-node shutdown and invalid-volume validation with behavioral regressions and qualification |

### One-box qualification and per-test support claim — [fn-222](.flow/specs/fn-222-gomad-qualify-the-one-box-cell-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-222.1](.flow/tasks/fn-222-gomad-qualify-the-one-box-cell-and.1.md) | ⏸ Parked | Wire prepare-only, workload-intent and support-list validation, then run both one-box cells and every inspection/replay gate on the frozen Rust candidate |

### WASM host-side execution cost profile — [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-185.1](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.1.md) | ⏸ Parked | Attribution, reconciliation and witness proven; R4 release-preservation proof pending fn-246 receiving |
| [fn-185.2](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.2.md) | ⬜ Todo | Capture wall and CPU profiles on the cluster and single-guest workloads |
| [fn-185.3](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.3.md) | ⬜ Todo | Rank cost centers and record the single optimization verdict |

### Functional-suite triage sweep — [fn-221](.flow/specs/fn-221-gomad-functional-suite-triage-sweep-on.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-221.2](.flow/tasks/fn-221-gomad-functional-suite-triage-sweep-on.2.md) | ⏸ Parked | Run the full sweep and retain the support matrix |
| [fn-221.3](.flow/tasks/fn-221-gomad-functional-suite-triage-sweep-on.3.md) | ⬜ Todo | Rank reasons and recommend follow-on specs |

### Typed boundaries for the Rust host — [fn-245](.flow/specs/fn-245-gomad-typed-boundaries-for-the-rust-host.md)

Filed 2026-10-24 from the complexity review; depends on fn-242, fn-249 and fn-240. Typed errors and size lints are fn-240 R11/R12, not here. fn-245.3 only sweeps prefix dispatch left after the fn-240 extractions.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-245.1](.flow/tasks/fn-245-gomad-typed-boundaries-for-the-rust-host.1.md) | ⬜ Todo | One authenticated-read abstraction (`gomad_evidence::io`) plus durable publication on `cap-std` replaces per-module read/digest/authenticate/publish copies; after fn-240 |
| [fn-245.2](.flow/tasks/fn-245-gomad-typed-boundaries-for-the-rust-host.2.md) | ⬜ Todo | Typed receipts with `deny_unknown_fields` replace `serde_json::Value` outside the canonical encoder; after fn-242.7 |
| [fn-245.3](.flow/tasks/fn-245-gomad-typed-boundaries-for-the-rust-host.3.md) | ⬜ Todo | Replace message-prefix dispatch (`runner_failure:`, `refused:`, `capacity:`) with typed outcome kinds |
| [fn-245.4](.flow/tasks/fn-245-gomad-typed-boundaries-for-the-rust-host.4.md) | ⬜ Todo | Engine phase as a type: no Option-state unwraps in `wasm-engine` |
| [fn-245.5](.flow/tasks/fn-245-gomad-typed-boundaries-for-the-rust-host.5.md) | ⬜ Todo | Guest path and limit constants defined once |
| [fn-245.6](.flow/tasks/fn-245-gomad-typed-boundaries-for-the-rust-host.6.md) | ⬜ Todo | `crc32fast`, `base64` and `wat` replace small hand-written codecs and test wasm bytes |
| [fn-245.7](.flow/tasks/fn-245-gomad-typed-boundaries-for-the-rust-host.7.md) | ⬜ Todo | One cluster-run entry point for the five scenario callers |
| [fn-245.8](.flow/tasks/fn-245-gomad-typed-boundaries-for-the-rust-host.8.md) | ⬜ Todo | `insta` snapshots replace hand-rolled corpus regeneration |

### General-purpose output predicates — [fn-250](.flow/specs/fn-250-gomad-general-purpose-output-predicates.md)

Filed 2026-10-25 (owner: Gomad must not be tied to Temporal's `softassert` package). Replaces the hardwired `campaign/soft_assertions.rs` oracle with a declared list of output predicates (JSON-field or text-line match, explicit identity derivation, fail/report severity) bound into recipe and plan identity; Temporal's softassert rule becomes a launcher preset. Depends on fn-240 and fn-245; fn-191 depends on it. Deferred work.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-250.1](.flow/tasks/fn-250-gomad-general-purpose-output-predicates.1.md) | ⬜ Todo | Predicate schema, validation and evaluator replace `campaign/soft_assertions.rs` |
| [fn-250.2](.flow/tasks/fn-250-gomad-general-purpose-output-predicates.2.md) | ⬜ Todo | Bind the predicate list into recipe/plan identity; thread the typed report through receipts, ledger and `inspect` |
| [fn-250.3](.flow/tasks/fn-250-gomad-general-purpose-output-predicates.3.md) | ⬜ Todo | Temporal softassert preset on the launcher; `--output-predicates FILE`; non-Temporal golden fixture |

### Sometimes assertions and site catalog — [fn-191](.flow/specs/fn-191-gomad-sometimes-assertions-and-a-site.md)

Rewritten onto the Rust host 2026-10-24; depends on fn-240, fn-242 and fn-245 (the `semantic` record section is typed). The Always oracle is `campaign/soft_assertions.rs` until fn-250 replaces it with declared output predicates; fn-191 depends on fn-250 (added 2026-10-25).

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-191.1](.flow/tasks/fn-191-gomad-sometimes-assertions-and-a-site.1.md) | ⬜ Todo | `Sometimes`/`Unreachable` in `common/softassert`; retained Go catalog generator and `validate` check |
| [fn-191.2](.flow/tasks/fn-191-gomad-sometimes-assertions-and-a-site.2.md) | ⬜ Todo | Guest channel in the cooperative overlay; helper semantic import; identities rotated |
| [fn-191.3](.flow/tasks/fn-191-gomad-sometimes-assertions-and-a-site.3.md) | ⬜ Todo | Typed `semantic` record section, catalog identity binding, `inspect` output, legacy replay test |
| [fn-191.4](.flow/tasks/fn-191-gomad-sometimes-assertions-and-a-site.4.md) | ⬜ Todo | Campaign flags, never-hit summary, `--must-hit` outcome, corpus features |
| [fn-191.5](.flow/tasks/fn-191-gomad-sometimes-assertions-and-a-site.5.md) | ⬜ Todo | One-box demonstration; cooperative qualification unchanged |

### WASM checkpoint-restore experiment — [fn-190](.flow/specs/fn-190-gomad-wasm-checkpoint-restore-experiment.md)

Rewritten onto the Rust host 2026-10-24; depends on fn-240 (snapshot/restore are `wasm-sim`/`wasm-model` boundary operations, so after fn-240.7). Experiment with a kill switch in .1; .5 runs on either path.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-190.1](.flow/tasks/fn-190-gomad-wasm-checkpoint-restore-experiment.1.md) | ⬜ Todo | Dormant re-entry prototype in the runtime overlay; recorded-suffix fixture at three boundaries (kill switch) |
| [fn-190.2](.flow/tasks/fn-190-gomad-wasm-checkpoint-restore-experiment.2.md) | ⬜ Todo | Crate `snapshot`/`restore` boundaries and `--dump=image` |
| [fn-190.3](.flow/tasks/fn-190-gomad-wasm-checkpoint-restore-experiment.3.md) | ⬜ Todo | `checkpoint-probe` developer verb: restore, recorded and fresh suffix, forced-prefix baseline |
| [fn-190.4](.flow/tasks/fn-190-gomad-wasm-checkpoint-restore-experiment.4.md) | ⬜ Todo | Temporal one-box boot-probe measurement |
| [fn-190.5](.flow/tasks/fn-190-gomad-wasm-checkpoint-restore-experiment.5.md) | ⬜ Todo | Research finding and go/no-go |

### Race profile for wasip1 — [fn-241](.flow/specs/fn-241-gomad-a-race-profile-for-wasip1-deferred.md)

Deferred by owner decision 2026-10-23; broken into eleven tasks on 2026-10-24, all ⛔ Blocked on the revival trigger recorded in the spec (saturated fault matrix, benchmark table showing a reachable unreported race). No work until revived.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-241.1](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.1.md) | ⛔ Blocked | Lift the wasip1 race gate; rebuild pinned `go` and `compile` for the race profile |
| [fn-241.5](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.5.md) | ⛔ Blocked | Stub `runtime/race` for wasm; instrumented choices fixture matches uninstrumented (early proof) |
| [fn-241.6](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.6.md) | ⛔ Blocked | Race overlay embedding with separate identity; determinism inventory rows |
| [fn-241.2](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.2.md) | ⛔ Blocked | FastTrack core: vector clocks, epochs, goroutine seeding, sync edges, ordered/unordered corpus |
| [fn-241.7](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.7.md) | ⛔ Blocked | Span-allocated shadow memory with recorded sampling fallback |
| [fn-241.8](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.8.md) | ⛔ Blocked | One-seed detection of PR 10549 on the §3 workload; fixed revision clean |
| [fn-241.3](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.3.md) | ⛔ Blocked | Race report class: wire fields, forcing schedule, retention and replay |
| [fn-241.9](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.9.md) | ⛔ Blocked | Symbolize both race stacks through the inspect path |
| [fn-241.10](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.10.md) | ⛔ Blocked | `sync/atomic` edges, stale-size variant, atomic-counter negative fixture, happens-before fingerprint |
| [fn-241.11](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.11.md) | ⛔ Blocked | `gomad.wasi-race/v1` identity, per-run selection, refusal on mismatch; cooperative byte-unchanged |
| [fn-241.4](.flow/tasks/fn-241-gomad-a-race-profile-for-wasip1-deferred.4.md) | ⛔ Blocked | Race profile capacities, qualification cells, operator docs, closing audit |
