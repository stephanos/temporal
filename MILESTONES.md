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
pending under fn-232 R12/task .13's frozen Rust qualification cycle. Native deletion
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

### fn-232 first (owner decision, 2026-10-23)

The owner asked, "why aren't we focused on fn-232 to get merged?" and approved the conductor's plan to focus on it. Until fn-232.13 closes, both worker slots serve it:
- One drives the final frozen qualification cycle.
- The other finishes inputs that must land before the freeze (fn-188.6 gate wiring, fn-233 shim fixes). It then prepares the .13 retirement change on a branch: Go host packages out of the default build, archcheck, Make/CI on the Rust binary, ARCHITECTURE.md/CLI.md and the size comparison. That branch merges as soon as the cycle passes.

No other milestone work starts before fn-232.13 closes. This changes ordering only: no stage, bound or acceptance item is waived.

### Performance deferred to the quiet-host window (owner decision, 2026-10-23)

The owner asked, "can we merge the blocked work and address any perf issues later when we have the stress test?"

Until the owner declares a quiet-host window, only timing and performance results stop being blocking. They are still measured and recorded:
- the cycle's 3,600-second wall-clock limit, and the stage and cost wall-time budgets;
- fn-185.2/.3 profiles and the optimization verdict;
- the fn-188.5 stress matrix;
- fn-221 sweep results that fail only from load (timeouts, deadline-sensitive inspection fixtures). These are classified as contention and rechecked in that window.

Correctness stays binding: outcomes, replay and determinism, assertions, refusals, memory and capacity bounds, and identities.

Completed work merges as soon as its gate passes. A task closes on its correctness acceptance. Its deferred performance items stay open, recorded in the task with the quiet-host window as revival trigger. The quiet-host window reruns the cycle timing, stress and profiles together, and any performance regression found then becomes a new task.

### fn-232.13 scope rulings (conductor, under the owner's 2026-10-23 delegation)

The owner's goal says to follow the conductor's own recommendations when in doubt. These rulings apply until the owner says otherwise:

1. **Behavioral reach of `test-wasm`.** Map each original test root and named subtest onto the Rust suites, parity lanes, the 39 transferred live roots and the qualification cells.
   - A Go-host-only unit test that checks an implementation detail of deleted Go code is retired with that code. Each retirement gets a one-line reason in the reach map. An example is the helper's `wait4` telemetry.
   - Every user-visible behavior of a retained feature keeps a mapped Rust test with its original assertions and bounds.
   - The 752 original callables are not ported one by one.
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
7. **Stress postponed (owner, 2026-10-23).** The fn-188.5 41×4 stress matrix is not a cycle stage. The owner postponed it because no quiet host will be available for a while. fn-188.5 stays parked and fn-188 stays open. Revival trigger: the owner declares a quiet-host window. It then runs on the candidate current at that time, using the existing stress producer and assembler. fn-242 does not wait for it.
8. **Pre-freeze source.** fn-227.1's inventory rows cite the Rust map-order tests before the freeze. The pre-freeze dry run also runs the Rust sweep against a real guest and runs fn-233's launcher, shim and Make lanes.
9. **Post-freeze windows run one after another on a quiet host.** fn-233 formal rerun (~1 h), fn-221 full sweep (5–8 h), then fn-185.2 profiles (7–9 h). fn-221 reruns the whole sweep on the frozen Rust binary instead of resuming the Go-candidate journal. The Go host's helper link-count infrastructure rows and its changed evidence bounds make that journal non-transferable.
10. **fn-240 crate count.** The owner's 8–12 crate target stands, even though the main crate has grown to about 110k lines. The read-only survey (.1) may run now. Extractions wait until the post-freeze windows are done, because each one rotates the host and helper identities.
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

Listed in delivery order (2026-10-23). fn-232 comes first. fn-233 and fn-188 are next because they land in its final freeze. The rest follow its candidate.

### Rust host rewrite — [fn-232](.flow/specs/fn-232-gomad-rewrite-the-host-side-in-rust.md)

First in the delivery order (owner decision 2026-10-20). New evidence lineage; behavioral equivalence gate; Go overlay, models, generator and corpora stay.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-232-gomad-rewrite-the-host-side-in-rust.13](.flow/tasks/fn-232-gomad-rewrite-the-host-side-in-rust.13.md) | 🚧 Running | Rust workspace/default host integrated; final qualification, behavioral reach and retirement acceptance pending ([development checkpoint](.flow/artifacts/milestones-rust-integration-2026-10-23.md)) |

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
| [fn-188.5](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.5.md) | ⏸ Parked | Stress producer and assembler ready; full matrix postponed until the owner declares a quiet-host window |
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
| [fn-185.1](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.1.md) | ⏸ Parked | Attribution, reconciliation and witness proven; R4 release-preservation proof pending fn-232.13's cycle |
| [fn-185.2](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.2.md) | ⬜ Todo | Capture wall and CPU profiles on the cluster and single-guest workloads |
| [fn-185.3](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.3.md) | ⬜ Todo | Rank cost centers and record the single optimization verdict |

### Functional-suite triage sweep — [fn-221](.flow/specs/fn-221-gomad-functional-suite-triage-sweep-on.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-221.2](.flow/tasks/fn-221-gomad-functional-suite-triage-sweep-on.2.md) | ⏸ Parked | Run the full sweep and retain the support matrix |
| [fn-221.3](.flow/tasks/fn-221-gomad-functional-suite-triage-sweep-on.3.md) | ⬜ Todo | Rank reasons and recommend follow-on specs |

### Qualification machinery cleanup — [fn-242](.flow/specs/fn-242-gomad-shrink-the-qualification.md)

Owner request 2026-10-23. Starts after the fn-232.13 cycle and the post-freeze windows. Each check keeps a named failure it catches or is deleted, and the Python cycle driver becomes a Rust `gomad` verb. Tasks come from `/flow-next:plan`.
