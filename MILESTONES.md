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
The [2026-10-19 retirement decision](#native-backend-retirement-and-the-functional-test-goal-owner-decision-2026-10-19)
retires the native backend. The deferred native specs (fn-105, fn-109, fn-110, fn-112, fn-128, fn-149, fn-155) were closed as won't-do on 2026-10-19.
Both backends are developed together here, and `tools/gomad` remains a shared owner.
List only unfinished tasks of each open spec below; completed tasks and their evidence
remain in Flow and Git history. Remove empty spec sections. Keep descriptions to one line and
spec sections to task tables—no progress prose. Refresh statuses from `flowctl list --json`;
a committed source candidate is not completed acceptance.

Status: 🚧 In progress · ⛔ Blocked · ⬜ Todo.

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

1. ~~Complete fn-189 upstream sync.~~ Done 2026-10-20. The integration base is Temporal `main` at `48a1e5ff025e55b3d4d0f44247423d4c279ff636`, retained by merge `0077477965a5b766b95a667247a9f562c100941b`. Frozen candidate `e50c6476fb` passed qualification, and the matched-release comparison moved to fn-185. See the [integration report](.flow/artifacts/fn189-upstream-sync-2026-10-17.md) and the [close-out](.flow/artifacts/fn189-fn187-closeout-2026-10-20.md).
2. ~~Complete fn-225 bounded harness result retention immediately after fn-189.~~ Done 2026-10-20, merged as `76bc9b75f5`. Measurement output roots keep the latest success and one recent failure; see the [spec](.flow/specs/fn-225-gomad-bounded-harness-result-retention.md).
3. ~~Follow with fn-187 helper fixture tests and module ownership.~~ Done 2026-10-20; see the [close-out](.flow/artifacts/fn189-fn187-closeout-2026-10-20.md).
4. ~~Finish with [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) host-side cost profile, after fn-187.~~ Superseded on 2026-10-19: fn-185 moves below the goal-path work (see the retirement decision below). Extend the moved measurement command and final helper module owners. fn-185 opens no optimization work. This order follows the owner decision on 2026-10-16 to put fn-185 last and fn-187 immediately before it.

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

### One-hour validation limit (owner decision, 2026-10-19)

The owner decided, "we can't have validation take longer than 1h". Full validation of a candidate (the WASM qualification ladder) must finish within 60 minutes of wall-clock time. The driver runs independent stages in parallel lanes under core and memory caps, gives timing- and deadline-sensitive stages the host to themselves, and fails when wall time exceeds 60 minutes. Only scheduling changes; no stage command, budget, deadline, seed, oracle or assertion is relaxed. Retired native stages leave the ladder, and the matched-release speed comparison moves out of validation into the fn-185 performance window.

### Native backend retirement and the functional-test goal (owner decision, 2026-10-19)

The owner decided, "we want to retire the native backend (not delete yet)", "our goal is definitely to run arbitrary functional tests with gomad (wasm)", and "yes file flow spec for promotion". Native is retiring: it stays in the tree under its fn-184 selection until [fn-223](.flow/specs/fn-223-gomad-delete-the-native-backend.md) removes it, and no native qualification is pursued. The shared model and the WASM backend stop depending on native sources by promoting the generated volume and network projections to owned source and relocating the protocol generator into the parent module ([fn-219](.flow/specs/fn-219-gomad-promote-the-model-projections-and.md)); relocating the overlay packages as a retained reference was rejected because nobody executes them once native is retired. The product goal is deterministic execution of arbitrary Temporal functional tests on the cooperative WASM profile. This decision grants no PR, push or CI authority.

Delivery order toward that goal, after fn-189 and fn-187. The owner agreed on 2026-10-19 to move [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) below the goal-path work so it profiles sweep workloads rather than fixtures; this supersedes the 2026-10-16 placement of fn-185 directly after fn-187. fn-185.1's attribution tooling is not gated; fn-185.2 is re-scoped to wait for fn-221.2 and include sweep workloads; the ordering is recorded in the task because Flow task dependencies cannot cross specs.

1. [fn-219](.flow/specs/fn-219-gomad-promote-the-model-projections-and.md) projection promotion and [fn-221](.flow/specs/fn-221-gomad-functional-suite-triage-sweep-on.md) functional-suite triage sweep run in parallel; they touch disjoint code.
2. [fn-220](.flow/specs/fn-220-gomad-one-command-temporal-test-setup.md) one-command Temporal test setup; fn-221 uses its installation if landed, otherwise a throwaway configuration.
3. [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md) host-side cost profile on sweep workloads, so timeout and capacity classes can be judged before refusal-closing specs are filed.
4. Refusal-closing specs are filed from fn-221's ranked classification and fn-185's cost verdict; none is filed in advance.
   [fn-224](.flow/specs/fn-224-gomad-ringpop-membership-on-wasm-within.md) ringpop membership within reason is already filed and follows the sweep; its unstub gate may land earlier if the sweep wants it.
5. [fn-222](.flow/specs/fn-222-gomad-qualify-the-one-box-cell-and.md) qualifies the one-box cell and defines the per-test support claim, after fn-220.
6. [fn-188](.flow/specs/fn-188-gomad-wasm-determinism-boundary-audit.md) determinism boundary audit continues alongside; it must hold for a testcore-sized guest.
7. [fn-223](.flow/specs/fn-223-gomad-delete-the-native-backend.md) deletes the native backend after fn-219; fn-128 and fn-149 are already closed as won't-do.

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

### WASM host-side execution cost profile — [fn-185](.flow/specs/fn-185-gomad-wasm-host-side-execution-cost.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-185.1](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.1.md) | 🚧 In progress | Attribute host imports to exact ops and model owners in the diagnostic profile with reconciliation |
| [fn-185.2](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.2.md) | ⬜ Todo | Capture wall and CPU profiles on the cluster and single-guest workloads |
| [fn-185.3](.flow/tasks/fn-185-gomad-wasm-host-side-execution-cost.3.md) | ⬜ Todo | Rank cost centers and record the single optimization verdict |

### Model projection promotion — [fn-219](.flow/specs/fn-219-gomad-promote-the-model-projections-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-219.1](.flow/tasks/fn-219-gomad-promote-the-model-projections-and.1.md) | ⬜ Todo | Promote the volume and network projections to owned source |
| [fn-219.2](.flow/tasks/fn-219-gomad-promote-the-model-projections-and.2.md) | ⬜ Todo | Relocate the protocol generator and I/O wire schema into the parent module |
| [fn-219.3](.flow/tasks/fn-219-gomad-promote-the-model-projections-and.3.md) | ⬜ Todo | Rewire Makefile, archcheck and CI to the default set |
| [fn-219.4](.flow/tasks/fn-219-gomad-promote-the-model-projections-and.4.md) | ⬜ Todo | Record the identity ledger, requalify rotated identities and refresh documentation |

### One-command Temporal test setup — [fn-220](.flow/specs/fn-220-gomad-one-command-temporal-test-setup.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-220.1](.flow/tasks/fn-220-gomad-one-command-temporal-test-setup.1.md) | ⬜ Todo | Factor the fixture configuration derivation into shared code and choose the verb |
| [fn-220.2](.flow/tasks/fn-220-gomad-one-command-temporal-test-setup.2.md) | ⬜ Todo | Expose the one-command setup and run TestFrontendSystemInfo end to end |
| [fn-220.3](.flow/tasks/fn-220-gomad-one-command-temporal-test-setup.3.md) | ⬜ Todo | Document the one-command workflow |

### Functional-suite triage sweep — [fn-221](.flow/specs/fn-221-gomad-functional-suite-triage-sweep-on.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-221.1](.flow/tasks/fn-221-gomad-functional-suite-triage-sweep-on.1.md) | ⬜ Todo | Build the sweep harness and prove it on two packages |
| [fn-221.2](.flow/tasks/fn-221-gomad-functional-suite-triage-sweep-on.2.md) | ⬜ Todo | Run the full sweep and retain the support matrix |
| [fn-221.3](.flow/tasks/fn-221-gomad-functional-suite-triage-sweep-on.3.md) | ⬜ Todo | Rank reasons and recommend follow-on specs |

### One-box qualification and per-test support claim — [fn-222](.flow/specs/fn-222-gomad-qualify-the-one-box-cell-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-222.1](.flow/tasks/fn-222-gomad-qualify-the-one-box-cell-and.1.md) | ⬜ Todo | Qualify the one-box cell on the current candidate |
| [fn-222.2](.flow/tasks/fn-222-gomad-qualify-the-one-box-cell-and.2.md) | ⬜ Todo | Define the per-test support list and produce the first one |

### WASM determinism boundary audit — [fn-188](.flow/specs/fn-188-gomad-wasm-determinism-boundary-audit.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-188.1](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.1.md) | ⬜ Todo | Determinism inventory: schema, loader and completeness over the import table and seeded streams |
| [fn-188.2](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.2.md) | ⬜ Todo | Complete the site classes: runtime clock/draw patches, helper source scan, archcheck rule, engine settings |
| [fn-188.3](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.3.md) | ⬜ Todo | Fail-closed request, reply and engine-setting boundaries on both sides |
| [fn-188.4](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.4.md) | ⬜ Todo | First-divergence report across incarnations and World order, with reproducer |
| [fn-188.5](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.5.md) | ⬜ Todo | Same-seed stress producer with host perturbation axes and retained report |
| [fn-188.6](.flow/tasks/fn-188-gomad-wasm-determinism-boundary-audit.6.md) | ⬜ Todo | Regression gates, version pinning, validate wiring and the documented contract |

### Ringpop membership on WASM within reason — [fn-224](.flow/specs/fn-224-gomad-ringpop-membership-on-wasm-within.md)

Owner decision 2026-10-19: support ringpop enough to verify a multi-node membership ring under gomad, without modeling tchannel internals or changing testcore. Follows fn-221.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-224.1](.flow/tasks/fn-224-gomad-ringpop-membership-on-wasm-within.1.md) | ⬜ Todo | Unstub ringpop in the slim guest and record the module size delta |
| [fn-224.2](.flow/tasks/fn-224-gomad-ringpop-membership-on-wasm-within.2.md) | ⬜ Todo | Run the ringpop unit tests under the cooperative profile and classify every outcome |
| [fn-224.3](.flow/tasks/fn-224-gomad-ringpop-membership-on-wasm-within.3.md) | ⬜ Todo | Add a multi-node ringpop partition/heal scenario and document it |

### Native backend deletion — [fn-223](.flow/specs/fn-223-gomad-delete-the-native-backend.md)

Runs after fn-219. Retains historical artifacts and ledgers under `.flow/artifacts`; changes no WASM semantics.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-223.1](.flow/tasks/fn-223-gomad-delete-the-native-backend.1.md) | ⬜ Todo | Remove the nested native module and prove the default set builds |
| [fn-223.2](.flow/tasks/fn-223-gomad-delete-the-native-backend.2.md) | ⬜ Todo | Remove the linkname glue, Make targets, CI jobs, archcheck tables and native build tags |
| [fn-223.3](.flow/tasks/fn-223-gomad-delete-the-native-backend.3.md) | ⬜ Todo | Record unchanged WASM identities and refresh docs |
