# Gomad: Native and WASM Milestones

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](.plans/GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

The WASM backend lives in `tools/gomad/internal/backends/wasm` and shares its
campaign, artifact, replay, and choice owners. Read [the WASM README](tools/gomad/internal/backends/wasm/README.md) before working on that
backend; [the WASM section acceptance evidence](.flow/artifacts/wasm-section-acceptance-2026-10-12.md)
lists the gates a shared-owner change must pass.

## Work tracking

Flow specs and tasks own scope, acceptance criteria (R-IDs), blockers, and evidence.
Remaining native linux/amd64 qualification and Linux CI work stay deferred under
[fn-128](.flow/specs/fn-128-gomad-deferred-linux-qualification-and.md). The owner-approved
Darwin deferral transfers remaining native darwin/arm64 qualification to
[fn-149](.flow/specs/fn-149-gomad-deferred-darwin-qualification.md), as mapped in the
[native transfer manifest](.flow/artifacts/native-scope-transfer-2026-10-07.md).
Missing transferred native evidence does not block source-spec completion. Implementation,
ordinary host-source coverage, lint, preservation, both-source-set static checks, generated
validation, source review and other independent acceptance remain required. Native full-host
execution stays with the native owner; portable coverage does not substitute for that gate.
Both platforms remain unverified for the current candidate until their owners retain native proof.
No PR, push or CI action is authorized by the deferral or its revival.
The [2026-10-14 native hibernation decision](#native-backend-hibernation-owner-decision-2026-10-14)
extends this acceptance deferral to the new shared-owner native execution gates.
Both backends are developed together here, and `tools/gomad` remains a shared owner.
List only unfinished tasks of each open spec below; completed tasks and their evidence
remain in Flow and Git history. Remove empty spec sections. Keep descriptions to one line and
spec sections to task tables—no progress prose. Refresh statuses from `flowctl list --json`;
a committed source candidate is not completed acceptance.

Status: 🚧 In progress · ⛔ Blocked · ⬜ Todo · ⏸ Deferred.
Native deferral labels preserve the previously recorded task state; they do not rewrite Flow history.

Work a spec with `/flow-next:work <spec>`; list ready tasks with `flowctl ready`.
Record a deferred task's revival trigger in its owning Flow task before implementation.

Completed specs and their evidence remain in `.flow/` and Git history.

## Immediate delivery order

### Native backend hibernation (owner decision, 2026-10-14)

The owner asked, "why is there a native gate anyway; we can declare the native backend as deferred/hibernating for now". Native backend execution acceptance is deferred while the WASM section is completed. This explicit new waiver supersedes the earlier exception that newly introduced fn-153/fn-154/fn-179 native gates must remain blockers with their source tasks; inherited native gates remain deferred too. It waives those gates as conditions of WASM/source completion, without recording an execution pass or completing native qualification.

| Deferred execution scope | Linux owner | Darwin owner |
| --- | --- | --- |
| Supported patched-runtime baseline, full native `test-host`, runtime/overlay/clock controls | fn-128.1 | fn-149.1 (runtime/clock), fn-149.2 (full-host) |
| fn-153 native World replay and affected runtime consumers; fn-154 R1–R12 native adapter/descriptor/connection, expiry/dial, model conformance, process/fault, framed-stream and composed generic-I/O replay controls; fn-179 R1–R3 native execution/replay consumers; native execution invalidated by fn-166 moves/rename | fn-128.4 | fn-149.2 |
| Current-candidate reconciliation of those obligations and existing native qualification ledgers | fn-128.7 | fn-149.4 |

The original task selectors, requirements and historical receipts remain the deferred execution ledger, including both-backend semantics and fn-154's divided payload replay guarantees. Native owners stay incomplete and deferred until an explicit qualification request, the supported native host/toolchain/profile and a frozen candidate are available. Record failures and unavailable execution as such; stock-Go coverage, cross-builds and historical results supply no native pass. Existing unsupported-platform/configuration refusals, source correctness, native source/tests/fixtures and features remain in place.

Source tasks retain implementation, portable Go coverage (including portable native-package tests), ordinary host-source checks, both-source-set static checks, lint, generated/pin validation, preservation, source review, documentation and all actual WASM execution/replay/resource gates. Classify a gate by the backend/runtime it executes, rather than its filename or package: the SQLite selected-state and catalogue storage-crash controls execute WASM and remain required, including projection through `native_crash.go`. Complete source tasks only after every retained requirement passes and the deferred native obligations are accounted for under the owners above. This decision changes acceptance ownership only and grants no PR, push or CI authority.

Owner execution scope: complete the remaining WASM work in dependency order:

1. Complete [fn-189](.flow/specs/fn-189-gomad-sync-with-upstream-temporal-main.md) upstream sync: merge `temporalio/temporal` `main` into `gomad` (merge, not rebase, so retained evidence SHAs stay valid) and requalify.
2. Follow with [fn-187](.flow/specs/fn-187-gomad-wasm-helper-fixture-tests-and.md) helper fixture tests and module ownership. Its module restructure follows its corpus and coverage work.
3. Finish with [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) host-side cost profile, after fn-187. Extend the moved measurement command and final helper module owners. fn-185 opens no optimization work. This order follows the owner decision on 2026-10-16 to put fn-185 last and fn-187 immediately before it.

### Native backend isolation (owner decision, 2026-10-16)

The owner answered, "yes to isolate native backend". The native backend leaves the default Gomad host gates (build, vet, lint, `go test ./...`, archcheck default inventory, the WASM and host-tools CI jobs) and stays in the tree, buildable and testable through one explicit selection owned by fn-184. This is freezing under the feature-preservation constraint, not removal: every native source, test, fixture, generator and ledger stays, fn-128 and fn-149 keep their deferred scope and revival triggers, and no native execution pass is recorded. A default build refuses a native installation instead of falling back to WASM. This decision grants no PR, push or CI authority.

Preserve prior evidence under its original identities, without claiming it qualifies
changed runtime inputs. Native deferrals, qualification resource limits, and PR,
push and CI restrictions remain in force.

Owner performance decision (2026-10-13): WASM execution-speed work stays in the Rust
helper, accepting Go→Rust duplication where it buys performance; embedding Wasmtime in Go
(`wasmtime-go`) is rejected.

### WASM performance measurement host (owner decision, 2026-10-17)

The owner answered, "since this is WASM, we don't need a particular env. that was for the native backend." WASM performance measurements may run on any host; no host platform qualifies or disqualifies a WASM result. This supersedes the 2026-10-14 Linux/arm64 requirement. Baseline and candidate are still measured on the same host in the same window with the specified selections and inputs, and every threshold, gain-versus-spread rule, seed, repetition, median/range, correctness, choice/replay, identity and resource-budget requirement still applies; retain host evidence, source/helper/module/engine identities, raw measurements and exact rerun commands. Native platform qualification stays deferred under fn-128 and fn-149.

### Performance checks deferred to fn-185 (owner decision, 2026-10-17)

The owner asked, "can we defer the perf checks to fn-185 and then close out fn-182 and fn-184?" The wall-clock measurements of fn-182 R4 (cold compilation, warm preparation/load, fresh-instance startup, execution wall time and helper peak RSS for full versus slim guests) and fn-184 R5 (build, vet, lint and default test target wall times for the default and selected sets) move to [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) R6, task .2. The matched same-host baseline/candidate contract, repetitions, median/range, gain-versus-spread labels, identities and rerun commands are unchanged. fn-182 and fn-184 close with the byte and count measurements that already exist (module and compiled-cache bytes; package and line counts), recorded in their close-out artifacts. No performance gain is claimed until fn-185 measures it.

## Batch-first execution strategy

Owner decision (2026-10-10). Optimize milestone execution for implemented and verified
capabilities per hour. Accept temporary breaking changes and integration risk to finish larger
coherent batches. This strategy supersedes older per-task green-start, source-admission packet,
review, commit and evidence-duplication requirements in milestone specs/tasks. Functional
acceptance, explicit qualification prerequisites, the immediate delivery order, native deferrals and the
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
   order and prove the required contracts before completion. Apply the native hibernation
   amendment to native-first execution gates; a missing external checkout still holds. Put
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
   native full-host gate, `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad test-host`,
   belongs to the deferred native owner under the hibernation amendment. Retain portable constituents
   and actual WASM gates; identify portable checks as portable and account for native runtime,
   overlay, integration, race and platform-specific acceptance under its deferred owner. After a
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
requirements, and native deferrals above. It grants no additional native qualification,
CI, PR, or push authority.

## Constraints

- **No policy widening.** Gomad never grants `syscall`, `os/exec`, `os/signal`, or
  `golang.org/x/sys` generically. Every exception is an exact compatibility pack bound to a
  module version, go.sum hash, per-file SHA-256, owner, and workload, reviewed under the
  existing `discover`, `review`, `generate --approve-review`, `check`, `qualify` flow.
- **No source translation and no test rewriting.** Determinism comes from the patched
  toolchain and the reviewed boundary. A Temporal test that needs a Gomad-specific overlay of
  its own source, as gomad1 required, is a blocker to record, never a fix to ship.
- **Fail-closed stays.** An unmodeled boundary operation terminates the process. Work that needs
  a new modeled operation adds it with a semantic contract, a resource bound, transcript
  coverage, exact replay, and a negative test, per COMPAT-5 in
  [the compatibility roadmap](.plans/GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
- **Evidence over narration.** Work is done when its command produces the stated report on a
  clean checkout. A passing local run that depends on untracked state does not count.
- **Platform.** The boundary manifest qualifies `darwin/arm64` and `linux/amd64`. Each platform
  is its own qualification and artifacts replay only where they were produced. The macOS sandbox
  test and the DTrace clock audit remain `darwin/arm64` only. Each platform's compatibility packs
  are its own; `compatibility-pack-qualification` qualifies the requests that name the host.
- **Server source changes are allowed but bounded.** A change under `common`, `service`,
  `temporal`, or `tests/testcore` is acceptable when it isolates an optional provider behind a
  build tag or an injection seam and the default build is unchanged. A change that alters
  runtime behavior for production builds requires dedicated production review and
  regression evidence.
- **Validation scope.** The full `./tests` set is not run as a gate; a change is validated on the
  smoke selection plus the suites it affects, with `make gomad-tests-qualification` as the
  on-demand local run.
- **Qualification identity.** Reports bind exact source and toolchain identities;
  pre-integration results do not qualify a combined candidate, and Darwin results do
  not substitute for Linux gates. An unavailable host or unexplained regression leaves
  acceptance incomplete in the task that owns the requirement. Transferred Linux and Darwin
  requirements remain incomplete under fn-128 and fn-149 and do not hold source-spec acceptance open.
- **Patch-policy boundary.** The collector-file prohibition remains in force.
  Overwriting collector-owned GC stamps requires an explicit patch-policy owner decision.
- **Preservation.** Keep CLI grammar/defaults, recorded formats, fixed-identity
  canonical bytes, error precedence, existing comments, fresh processes, native timer
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

## WASM

### Upstream sync — [fn-189](.flow/specs/fn-189-gomad-sync-with-upstream-temporal-main.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-189.1](.flow/tasks/fn-189-gomad-sync-with-upstream-temporal-main.1.md) | 🚧 In progress | Merge upstream temporalio/temporal main into gomad and requalify |

### WASM helper fixture tests and module ownership — [fn-187](.flow/specs/fn-187-gomad-wasm-helper-fixture-tests-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-187.1](.flow/tasks/fn-187-gomad-wasm-helper-fixture-tests-and.1.md) | ⬜ Todo | Add the helper transcript corpus and convert the engine scenarios |
| [fn-187.2](.flow/tasks/fn-187-gomad-wasm-helper-fixture-tests-and.2.md) | ⬜ Todo | Move measurement producers out of test files into wasmmeasure |
| [fn-187.3](.flow/tasks/fn-187-gomad-wasm-helper-fixture-tests-and.3.md) | ⬜ Todo | Delete inline tests the corpora already cover after a coverage check |
| [fn-187.4](.flow/tasks/fn-187-gomad-wasm-helper-fixture-tests-and.4.md) | ⬜ Todo | Restructure model, engine, namespace and cluster modules behind the corpus |
| [fn-187.5](.flow/tasks/fn-187-gomad-wasm-helper-fixture-tests-and.5.md) | ⬜ Todo | Measure and document the simplified test suite |

### WASM host-side execution cost profile — [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-185.1](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.1.md) | ⬜ Todo | Attribute host imports to exact ops and model owners in the diagnostic profile with reconciliation |
| [fn-185.2](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.2.md) | ⬜ Todo | Capture wall and CPU profiles on the cluster and single-guest workloads |
| [fn-185.3](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.3.md) | ⬜ Todo | Rank cost centers and record the single optimization verdict |

## Native

Deferred by owner decision: all unfinished specs and tasks below are outside the active WASM delivery queue. Completed work and recorded Flow states are preserved. Changes to shared Gomad owners still require WASM acceptance; this deferral does not revive native qualification.

### Deferred delivery order

Historical native priority (2026-10-10; deferred). Finish the already-running fn-109 task 72 verification,
then take fn-155 next, ahead of the remaining source queue below. Begin with
fn-155 `.1 → .2 → .8 → .3`, then follow its dependencies
through `.7`'s boundary decision. fn-109.17/.18 and fn-110.3 still wait for fn-155.7.
The execution proof needs darwin/arm64 or linux/amd64. Preserve retained acceptance
and native deferrals; this priority grants no PR, push or CI authority.

1. Finish fn-112 task 10's retained source acceptance for the delivered soak gate and shared documentation. Actual native soak runs and measured bounds remain with fn-149.4 and fn-128.5/.7; no workflow dispatch is authorized.
2. Reconcile the combined D26/fn-110 source candidate, fn-112 tasks 5/16/9, D27 and fn-109 tasks 2-6 against their retained source checks, preservation and source reviews. Keep lint, both-source-set static checks, generated validation, first-baseline and preservation requirements open wherever unproved. Native runtime/clock proof belongs to deferred fn-149.1 and fn-128.1/.4/.7.
3. Continue fn-109 tasks 7-11, fn-110 tasks 3-4, then fn-109 tasks 17-21 (tasks 12-16 and the other shared-owner tasks moved to the WASM section) and fn-110 task 5 in coherent implementation batches, including the admitted correction owners. Respect implementation dependencies within each batch; validate and review the integrated batch before completing its tasks. Missing native qualification now owned by fn-149 or fn-128 cannot block source admission or completion; unproved source requirements still do.
4. Resume fn-105 D8-D10's shared source work when the real checkout is available. Native Darwin consumer analyses/packs/replay/guidance belong to fn-149.3; Linux execution belongs to fn-128.6. Neither native owner removes the actual-checkout/source-review prerequisite.
5. Keep fn-149 deferred until an explicit Darwin qualification request and native darwin/arm64 execution are available. Then establish its pinned candidate/runtime, retain inherited integration/model/pack and downstream evidence, and reconcile an actual scheduled/dispatched soak plus final matrix. Publication and CI actions need separate authority.
6. Keep fn-128 and its Linux CI work deferred until an explicit Linux qualification request and native linux/amd64 execution are available. Its eventual sequence remains baseline, causal D12 strict-replay fix, D21-triggered D11 audit, native model/consumer/soak evidence and final Linux matrix. No PR or CI run is part of the current work.

### F10: follow-ups (deferred scope) — [fn-105](.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-105.4](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.4.md) | ⏸ Deferred (recorded: ⬜ Todo) | D4: architecture fitness checks for package coverage, purity, and signature visibility |
| [fn-105.5](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.5.md) | ⏸ Deferred (recorded: ⬜ Todo) | D5: reconcile architecture, platform, and determinism documentation |
| [fn-105.6](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.6.md) | ⏸ Deferred (recorded: ⬜ Todo) | D6: seeded and fixed virtual-clock tick policies |
| [fn-105.8](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.8.md) | ⏸ Deferred (recorded: 🚧 In progress) | D8: closure-mode support for downstream targets |
| [fn-105.9](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.9.md) | ⏸ Deferred (recorded: ⬜ Todo) | D9: shared downstream packs/source checks; native qualification moved to fn-149.3/fn-128.6 |
| [fn-105.10](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.10.md) | ⏸ Deferred (recorded: ⬜ Todo) | D10: downstream-seam guide |
| [fn-105.15](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.15.md) | ⏸ Deferred (recorded: ⬜ Todo) | D15: support larger choice traces when a workload needs them |

### Deep modules and tool interfaces — [fn-109](.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md) — Deferred

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-109.9](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.9.md) | ⏸ Deferred (recorded: 🚧 In progress) | Bounded adapter listing integrated; original acceptance gates remain open |
| [fn-109.10](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.10.md) | ⏸ Deferred (recorded: ⬜ Todo) | Supply build, cache and adapter locations from one validated installation description |
| [fn-109.11](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.11.md) | ⏸ Deferred (recorded: ⬜ Todo) | Capability/source-inventory owners integrated; inventory lint fixed, qualification pending |
| [fn-109.17](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.17.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Select backend-specific network listener and connection implementations at creation; waits for fn-155.7 |
| [fn-109.18](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.18.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Select backend-specific filesystem handle and mapping implementations at creation; waits for fn-155.7 |
| [fn-109.19](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.19.md) | ⏸ Deferred (recorded: 🚧 In progress) | Architecture digest lint repaired; original acceptance remains open |
| [fn-109.20](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.20.md) | ⏸ Deferred (recorded: ⬜ Todo) | Reconcile architectural guidance with delivered owners and interfaces (D5). |
| [fn-109.21](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md) | ⏸ Deferred (recorded: ⬜ Todo) | Run final qualification and retain the finding completion matrix |
| [fn-109.23](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.23.md) | ⏸ Deferred (recorded: 🚧 In progress) | Repair module-aware lint routing and supply the nested host gates |
| [fn-109.24](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.24.md) | ⏸ Deferred (recorded: ⬜ Todo) | Restore repository-relative lint exclusion matching |
| [fn-109.25](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.25.md) | ⏸ Deferred (recorded: ⬜ Todo) | Preserve lifecycle fault resolution while repairing exhaustive lint |
| [fn-109.26](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.26.md) | ⏸ Deferred (recorded: ⬜ Todo) | Restore Runner semantic ownership in CLI callers |
| [fn-109.27](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.27.md) | ⏸ Deferred (recorded: ⬜ Todo) | Correct current R18 preservation disclosures |
| [fn-109.32](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.32.md) | ⏸ Deferred (recorded: ⬜ Todo) | Preserve concrete error callback provenance in architecture checks |
| [fn-109.33](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.33.md) | ⏸ Deferred (recorded: ⬜ Todo) | Preserve artifact reflection helper coverage and clone isolation |
| [fn-109.34](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.34.md) | ⏸ Deferred (recorded: ⬜ Todo) | Check CLI private-mode fixture reader cleanup |
| [fn-109.35](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.35.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Preserve corpus reader cleanup and publication failures |
| [fn-109.37](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.37.md) | ⏸ Deferred (recorded: ⬜ Todo) | Qualification cleanup and import lint clean; original qualification open |
| [fn-109.38](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.38.md) | ⏸ Deferred (recorded: ⬜ Todo) | Target and pure-policy lint clean; original qualification remains open |
| [fn-109.39](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.39.md) | ⏸ Deferred (recorded: ⬜ Todo) | Cleanup lint repaired; fault, pin and original qualification remain open |
| [fn-109.40](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.40.md) | ⏸ Deferred (recorded: 🚧 In progress) | Command-test readiness repaired; retained mechanism acceptance remains open |
| [fn-109.41](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.41.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Canonical JSON exhaustive lint repaired; original qualification remains open |
| [fn-109.42](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.42.md) | ⏸ Deferred (recorded: ⬜ Todo) | Exact-pack exhaustive lint repaired; original qualification remains open |
| [fn-109.43](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.43.md) | ⏸ Deferred (recorded: ⬜ Todo) | Five-import admission repaired; original qualification remains open |
| [fn-109.44](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.44.md) | ⏸ Deferred (recorded: ⬜ Todo) | Four mechanical lint findings repaired; original qualification remains open |
| [fn-109.45](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.45.md) | ⏸ Deferred (recorded: ⬜ Todo) | Seven mechanical lint findings repaired; full lint and qualification remain open |
| [fn-109.46](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.46.md) | ⏸ Deferred (recorded: ⬜ Todo) | Stdout reports repaired; full lint and original qualification remain open |
| [fn-109.47](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.47.md) | ⏸ Deferred (recorded: ⬜ Todo) | Archive cleanup repaired; full lint and original qualification remain open |
| [fn-109.48](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.48.md) | ⏸ Deferred (recorded: ⬜ Todo) | Patch cleanup repaired; full lint and original qualification remain open |
| [fn-109.49](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.49.md) | ⏸ Deferred (recorded: ⬜ Todo) | Adapter cache cleanup repaired; full lint and original qualification remain open |
| [fn-109.50](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.50.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check five terminal generator diagnostics without changing statuses |
| [fn-109.51](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.51.md) | ⏸ Deferred (recorded: 🚧 In progress) | Reconcile one usage-status fixture with the preserved original contract |
| [fn-109.53](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.53.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check CLI diagnostics while preserving primary outcomes |
| [fn-109.54](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.54.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check doctor reports and classify output failures |
| [fn-109.55](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.55.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check verify-only replay report delivery |
| [fn-109.56](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.56.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check builder cleanup while preserving publication outcomes |
| [fn-109.57](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.57.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check test cleanup results without changing resource lifetimes |
| [fn-109.58](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.58.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check inspection and invalid compiler-fixture cleanup |
| [fn-109.59](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.59.md) | ⏸ Deferred (recorded: 🚧 In progress) | Make preserved test-switch no-op cases explicit |
| [fn-109.60](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.60.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check six child-fixture outputs without changing process outcomes |
| [fn-109.61](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.61.md) | ⏸ Deferred (recorded: 🚧 In progress) | Observe competing-build lock contention instead of sleeping |
| [fn-109.62](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.62.md) | ⏸ Deferred (recorded: 🚧 In progress) | Bound progress-test startup and observe early completion |
| [fn-109.64](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.64.md) | ⏸ Deferred (recorded: 🚧 In progress) | Check watchdog fixture readiness writes without masking setup failure |

### Runtime patch minimization — [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md) — Deferred

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-110.3](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.3.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Relocate crypto initialization and syscall declarations to overlays; waits for fn-155.7 |
| [fn-110.4](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.4.md) | ⏸ Deferred (recorded: ⬜ Todo) | Emit the canonical one-context-line patch and pin regeneration to the descriptor |
| [fn-110.5](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.5.md) | ⏸ Deferred (recorded: ⬜ Todo) | Qualify the final candidate and publish measurements and guidance |

### Syscall-level I/O boundary — [fn-155](.flow/specs/fn-155-gomad-syscall-level-io-boundary-from.md) — Deferred

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-155.1](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.1.md) | ⏸ Deferred (recorded: 🚧 In progress) | Port the virtual descriptor layer and syscall edge onto the Gomad toolchain |
| [fn-155.2](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.2.md) | ⏸ Deferred (recorded: ⬜ Todo) | Select the boundary and adapter exclusions per run; bind into artifact identity |
| [fn-155.3](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.3.md) | ⏸ Deferred (recorded: ⬜ Todo) | Record descriptor I/O; prove same-seed determinism for a single-process gRPC workload |
| [fn-155.4](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.4.md) | ⏸ Deferred (recorded: ⬜ Todo) | Serve multi-node simulation traffic through virtual descriptors in both backends |
| [fn-155.5](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.5.md) | ⏸ Deferred (recorded: ⬜ Todo) | Soak single-process and multi-node workloads with the boundary on |
| [fn-155.6](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.6.md) | ⏸ Deferred (recorded: ⬜ Todo) | Classify the 15 adapters and run network adapters excluded |
| [fn-155.7](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.7.md) | ⏸ Deferred (recorded: ⬜ Todo) | Measure size, record the files/DNS decision and annotate dependent work |
| [fn-155.8](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.8.md) | ⏸ Deferred (recorded: ⬜ Todo) | Admit the edge's socket entry points in capability guard and closure policy |

### Determinism assurance and test strategy — [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md) — Deferred

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-112.10](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.md) | ⏸ Deferred (recorded: 🚧 In progress) | Finish retained source acceptance for the soak gate and shared documentation |

### Deferred Linux qualification and repairs — [fn-128](.flow/specs/fn-128-gomad-deferred-linux-qualification-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-128.1](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.1.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Native Linux candidate and runtime/host evidence |
| [fn-128.2](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.2.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Causal Linux replay fix and strict CI restoration |
| [fn-128.3](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.3.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Conditional Linux host-clock audit or owner disposition |
| [fn-128.4](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.4.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Linux model/pack/workload qualification, including new shared-owner native gates |
| [fn-128.5](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.5.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Actual Linux determinism soak and measured bound |
| [fn-128.6](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.6.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Downstream Linux analyses, packs, exact replay and guidance |
| [fn-128.7](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.7.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Final source-bound Linux matrix, review and documentation |

### Deferred Darwin qualification - [fn-149](.flow/specs/fn-149-gomad-deferred-darwin-qualification.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-149.1](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.1.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Native Darwin baseline, runtime and clock evidence |
| [fn-149.2](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.2.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Darwin model/pack/integration qualification, including new shared-owner native gates |
| [fn-149.3](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.3.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Real downstream Darwin qualification and guidance |
| [fn-149.4](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.4.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Scheduled/dispatched Darwin soak and final qualification matrix |
