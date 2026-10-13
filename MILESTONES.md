# Gomad: Native and WASM Milestones

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](.plans/GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

The WASM backend lives in `tools/gomad3/backends/wasm` and shares its
campaign, artifact, replay, and choice owners. The completed
[fn-151](.flow/specs/fn-151-wasm-gomad-execution-backend.md) determinism prototype recorded
**GO** on 2026-10-10 ([report](.flow/artifacts/fn151-prototype-report-2026-10-10.md));
[fn-156](.flow/specs/fn-156-wasm-gomad-productionization.md) productionized it (completed
2026-10-13). Read [the WASM README](tools/gomad3/backends/wasm/README.md) before working on that
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
fn-151 migrated from the retired `temporal_wasm` checkout on 2026-10-10 and is complete;
both backends are developed together here, and `tools/gomad3` remains a shared owner.
List only unfinished tasks of each open spec below; completed tasks and their evidence
remain in Flow and Git history. Remove empty spec sections. Keep descriptions to one line and
spec sections to task tables—no progress prose. Refresh statuses from `flowctl list --json`;
a committed source candidate is not completed acceptance.

Status: ✅ Done · 🚧 In progress · ⛔ Blocked · ⬜ Todo · ⏸ Deferred.
Native deferral labels preserve the previously recorded task state; they do not rewrite Flow history.

Work a spec with `/flow-next:work <spec>`; list ready tasks with `flowctl ready`.
Record a deferred task's revival trigger in its owning Flow task before implementation.
<a id="maintenance-cost"></a>
<a id="search-path-findings-fn-114"></a>

Completed specs and their evidence remain in `.flow/` and Git history, including
[version-pin maintenance](.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md)
and [search-path findings](.flow/specs/fn-114-gomad-correct-search-path-defects-and.md).

## Immediate delivery order

Owner execution scope: complete the WASM section. The
[fn-164 layout](.flow/specs/fn-164-gomad-backend-layout-shared-core-with.md),
[fn-165 checker](.flow/specs/fn-165-gomad-architecture-checks-structural.md) and
[fn-156 productionization](.flow/specs/fn-156-wasm-gomad-productionization.md)
batches were completed on 2026-10-12/13; fn-156 was qualified on final candidate
`9d6012b52b` ([evidence](.flow/artifacts/wasm-section-acceptance-2026-10-12.md)).
The shared-owner queue in the WASM section (fn-166, fn-109 shared subset, fn-152,
fn-153, fn-154) is next.
Preserve prior evidence under its original identities, without claiming it
qualifies changed runtime inputs. Remove completed task rows only after verified
Flow completion. Native deferrals, qualification resource limits, and PR, push
and CI restrictions remain in force.

Owner performance decision (2026-10-11): accept the measured 2.911x workflow
execution speedup for now (final fn-156 matched costs: workflow 3.739x, activity 4.636x); the previous all-workload 5x target no longer blocks
fn-156 completion. Defer further speedup optimization. Preserve current-source
cost measurement, correctness/replay, scheduling, source/review and resource gates.
The measured full-wall ratio was 0.694x, not an end-to-end speedup.

Owner performance decision (2026-10-13): resume WASM execution-speed work in the
Rust helper, accepting Go→Rust duplication where it buys performance; embedding
Wasmtime in Go (`wasmtime-go`) is rejected. The work is tracked in the
[execution performance](#wasm-execution-performance) specs fn-172–fn-178.

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
   order and prove the required contracts before completion. An explicit native-first gate or
   missing external checkout still holds. Put tightly coupled edits in one worker's batch
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
   `make -C tools/gomad3 validate` when generator, protocol or toolchain inputs changed. Run one
   full host gate per batch with
   `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` on a supported native host;
   count it toward overlapping Runner/CLI task commands. Runtime, overlay, integration, race
   and platform-specific requirements retain their own coverage. On an unsupported host,
   identify portable checks as portable and retain native acceptance with its owner. After a
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
  smoke selection plus the suites it affects, with `make gomad3-tests-qualification` as the
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
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`.
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

### Shared-owner queue

Owner decision (2026-10-12): these deferred specs change owners the WASM backend shares, so they form the WASM queue. Work them in this order: fn-166 tasks 1–5 and 9 (identity-neutral structure and test layout; done 2026-10-13), the fn-109 shared subset (closeout amendment 2026-10-13 in the spec) (Runner orchestration split and portable scripted Runner coverage first), then fn-152, then fn-153, fn-154 and fn-166 tasks 6–8 and 10 as one batch (including the owner-approved fn-109.14 volume/process codec and fn-153 strict-JSON and host-helper amendments) so their codec, projection, rename and identity rotations need a single WASM requalification. Each shared-owner change requires the WASM gates in the [acceptance evidence](.flow/artifacts/wasm-section-acceptance-2026-10-12.md) to pass on the integrated candidate. Record each revived task's revival trigger in its Flow task before implementation.

### Readable package structure — [fn-166](.flow/specs/fn-166-gomad-readable-package-structure-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-166.6](.flow/tasks/fn-166-gomad-readable-package-structure-and.6.md) | ⬜ Todo | Identity-rotating renames; batched with fn-153/fn-154 |
| [fn-166.7](.flow/tasks/fn-166-gomad-readable-package-structure-and.7.md) | 🚧 In progress | Inspection/library retirement source joined; integrated checks and full grouping remain |
| [fn-166.8](.flow/tasks/fn-166-gomad-readable-package-structure-and.8.md) | ⬜ Todo | Delete network v1 and choice v1 (diagnostic profile kept); batched with fn-153/fn-154 |
| [fn-166.10](.flow/tasks/fn-166-gomad-readable-package-structure-and.10.md) | ⬜ Todo | Move gomad3sim into gomad3 as internal/model/cluster; batched with fn-153/fn-154 |
| [fn-166.11](.flow/tasks/fn-166-gomad-readable-package-structure-and.11.md) | ⬜ Todo | Rename the gomad3 module and tree to gomad after the rotation batch |

<a id="canonical-json-publication-cleanup-fn-153"></a>

### JSON and file-publication cleanup - [fn-153](.flow/specs/fn-153-gomad-retire-canonical-json-and-private.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-153.18](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.18.md) | 🚧 In progress | Reconcile contracts and verify the integrated cleanup |
| [fn-153.20](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.20.md) | 🚧 In progress | Shared decoder source reviewed; final integrated WASM gates remain open |
| [fn-153.21](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.21.md) | 🚧 In progress | Publication/decoder corrections integrated; review and current-candidate gates remain |

<a id="virtual-network-stalling-fn-154"></a>

### Virtual network stalling and simplification - [fn-154](.flow/specs/fn-154-gomad-simpler-virtual-network-with.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-154.2](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.2.md) | 🚧 In progress | Generated limits/timeout codec source reviewed; integrated acceptance remains open |
| [fn-154.3](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.3.md) | 🚧 In progress | Timeout propagation source reviewed; patched-native adapter execution remains open |
| [fn-154.4](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.4.md) | 🚧 In progress | Shared byte-bounded connection checkpoint integrated; batch/native acceptance open |
| [fn-154.5](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.5.md) | 🚧 In progress | Held FIFO and atomic reservation checkpoint integrated; batch/native acceptance open |
| [fn-154.6](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.6.md) | 🚧 In progress | Autonomous expiry checkpoint integrated; batch/native acceptance open |
| [fn-154.7](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.7.md) | 🚧 In progress | Pending-dial/registration checkpoint integrated; batch/native acceptance open |
| [fn-154.8](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.8.md) | 🚧 In progress | Payload-free history/codec checkpoint integrated; batch/native acceptance open |
| [fn-154.9](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.9.md) | 🚧 In progress | Conformance source checkpoint integrated; native execution remains open |
| [fn-154.10](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.10.md) | 🚧 In progress | Process boundary/fault cases registered; native execution remains open |
| [fn-154.11](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.11.md) | 🚧 In progress | Framed/composed replay fixtures registered; native execution remains open |
| [fn-154.12](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.12.md) | 🚧 In progress | Contracts integrated; final frozen/native verification remains open |

<a id="wasm-execution-performance"></a>

### Execution performance (Rust helper)

Owner decision (2026-10-13): move per-call and per-run hot paths into the Rust helper; Go keeps build, campaign, evidence verification and publication. fn-172 and fn-173 start now, in parallel with the shared-owner queue; the other specs follow their recorded dependencies. Requalification batches: fn-173 joins the fn-153/fn-154/fn-166 rotation batch if ready, otherwise lands right after it; fn-174 and fn-175 form a second batch with one shared WASM requalification; fn-176 and fn-178 requalify on their own. fn-173 and fn-174 coordinate with fn-153.2 (helper cache) and fn-153.20 (helper-frame decoding). fn-177 builds only the Rust network owner and its Go differential harness for fn-178, because network-v2 runs always attach a broker; fn-177 and fn-178 port post-fn-154 network and post-fn-153 World semantics, so they wait for those specs. fn-166 R8 keeps the helper `diagnostic-profile` feature as the measurement tool for this work. Task paths predate fn-166's renames and need a plan-sync after them.

#### Guest-side cost experiments and baselines — [fn-172](.flow/specs/fn-172-gomad-guest-side-cost-experiments-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-172.1](.flow/tasks/fn-172-gomad-guest-side-cost-experiments-and.1.md) | 🚧 In progress | Sampler/fuel-off checkpoint integrated; source review remains open |
| [fn-172.2](.flow/tasks/fn-172-gomad-guest-side-cost-experiments-and.2.md) | 🚧 In progress | Profile driver/parser integrated; actual Darwin attribution remains open |
| [fn-172.3](.flow/tasks/fn-172-gomad-guest-side-cost-experiments-and.3.md) | ⬜ Todo | A/B report for GC, GOWASM and fuel-off variants |
| [fn-172.4](.flow/tasks/fn-172-gomad-guest-side-cost-experiments-and.4.md) | ⬜ Todo | Multi-instance cluster baseline |

#### Per-run preparation fast path — [fn-173](.flow/specs/fn-173-gomad-wasm-helper-per-run-preparation.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-173.1](.flow/tasks/fn-173-gomad-wasm-helper-per-run-preparation.1.md) | 🚧 In progress | No-unwind engine checkpoint integrated; required Darwin exit proof remains open |
| [fn-173.2](.flow/tasks/fn-173-gomad-wasm-helper-per-run-preparation.2.md) | ⬜ Todo | Skip module bytes on compile-cache hits and verify inside the miss path |
| [fn-173.3](.flow/tasks/fn-173-gomad-wasm-helper-per-run-preparation.3.md) | ⬜ Todo | Drop the Go adapter's per-run module hash |
| [fn-173.4](.flow/tasks/fn-173-gomad-wasm-helper-per-run-preparation.4.md) | ⬜ Todo | Compiled-artifact envelope v3 with a CRC-32 integrity check |
| [fn-173.5](.flow/tasks/fn-173-gomad-wasm-helper-per-run-preparation.5.md) | ⬜ Todo | Measure warm preparation and evidence equality on the Temporal module |

#### Local host-call fast path — [fn-174](.flow/specs/fn-174-gomad-wasm-helper-local-host-call-fast.md) — waits for fn-153

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-174.1](.flow/tasks/fn-174-gomad-wasm-helper-local-host-call-fast.1.md) | ⬜ Todo | Link every helper import as a typed host function |
| [fn-174.2](.flow/tasks/fn-174-gomad-wasm-helper-local-host-call-fast.2.md) | ⬜ Todo | Shared binary WASI transcript records in Go and the helper |
| [fn-174.3](.flow/tasks/fn-174-gomad-wasm-helper-local-host-call-fast.3.md) | ⬜ Todo | JSON-free typed local model path in the helper |
| [fn-174.4](.flow/tasks/fn-174-gomad-wasm-helper-local-host-call-fast.4.md) | ⬜ Todo | Compact clock run records in both owners |
| [fn-174.5](.flow/tasks/fn-174-gomad-wasm-helper-local-host-call-fast.5.md) | ⬜ Todo | Answer random_get, args and environ inside the helper |
| [fn-174.6](.flow/tasks/fn-174-gomad-wasm-helper-local-host-call-fast.6.md) | ⬜ Todo | Measure local host-call cost and reconcile docs |

#### Filesystem namespace in the helper — [fn-175](.flow/specs/fn-175-gomad-filesystem-namespace-in-the-rust.md) — waits for fn-174

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-175.1](.flow/tasks/fn-175-gomad-filesystem-namespace-in-the-rust.1.md) | ⬜ Todo | Namespace conformance harness: Rust driver, Go differential and state digest |
| [fn-175.2](.flow/tasks/fn-175-gomad-filesystem-namespace-in-the-rust.2.md) | ⬜ Todo | Rust namespace core: tree, descriptors, path resolution and metadata/path ops |
| [fn-175.3](.flow/tasks/fn-175-gomad-filesystem-namespace-in-the-rust.3.md) | ⬜ Todo | Rust namespace data ops, fd 1/2 output accounting and fd-subscription polling |
| [fn-175.4](.flow/tasks/fn-175-gomad-filesystem-namespace-in-the-rust.4.md) | ⬜ Todo | Namespace image in the execute request: Go flattening, chunked frames, Rust validation |
| [fn-175.5](.flow/tasks/fn-175-gomad-filesystem-namespace-in-the-rust.5.md) | ⬜ Todo | Route filesystem imports to the Rust namespace and return stdout/stderr from the helper |
| [fn-175.6](.flow/tasks/fn-175-gomad-filesystem-namespace-in-the-rust.6.md) | ⬜ Todo | Activity-selection gate: zero fs callbacks, >=25% faster on darwin/arm64, docs |

#### Durable volume crash model in the helper — [fn-176](.flow/specs/fn-176-gomad-durable-volume-crash-model-in-the.md) — waits for fn-175

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-176.1](.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.1.md) | ⬜ Todo | Volume conformance harness: storage ops, faults, enumeration and recovery against the Go projection |
| [fn-176.2](.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.2.md) | ⬜ Todo | Rust volume persistence core: durable/volatile state, sync semantics, snapshots and identities |
| [fn-176.3](.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.3.md) | ⬜ Todo | Rust crash enumeration, selection, lifecycle and self-contained recovery image |
| [fn-176.4](.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.4.md) | ⬜ Todo | Rust storage fault engine, storage-model validation and namespace-to-volume routing |
| [fn-176.5](.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.5.md) | ⬜ Todo | Helper-owned storage runs: protocol, helper-originated storage_crash and strict Go evidence decoder |
| [fn-176.6](.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.6.md) | ⬜ Todo | Broker split ownership: helper-owned namespace and volume under the Go broker |
| [fn-176.7](.flow/tasks/fn-176-gomad-durable-volume-crash-model-in-the.7.md) | ⬜ Todo | Durable gates with the Rust volume, Go reference retention and docs |

#### Rust network owner for clusters — [fn-177](.flow/specs/fn-177-gomad-network-models-in-the-rust-helper.md) — waits for fn-154

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-177.1](.flow/tasks/fn-177-gomad-network-models-in-the-rust-helper.1.md) | 🚧 In progress | Build operation corpus against integrated fn-154 source; currency and drift pins |
| [fn-177.2](.flow/tasks/fn-177-gomad-network-models-in-the-rust-helper.2.md) | ⬜ Todo | Rust network owner: connections, FIFO queues, topology and history |
| [fn-177.3](.flow/tasks/fn-177-gomad-network-models-in-the-rust-helper.3.md) | ⬜ Todo | Rust network owner: stall expiry, dial stall and history reservations |

#### Multi-instance clusters in one helper process — [fn-178](.flow/specs/fn-178-gomad-multi-instance-clusters-in-one.md) — waits for fn-153, fn-154, fn-172, fn-176, fn-177

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-178.1](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.1.md) | ⬜ Todo | Design record and handoff spike: control protocol, threading, terminal taxonomy, watchdog, memory budget |
| [fn-178.2](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.2.md) | ⬜ Todo | World and clock conformance harness with Go drift guard |
| [fn-178.3](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.3.md) | ⬜ Todo | Rust World model: ordering, seeded equivalence ranking, capacity and digests |
| [fn-178.4](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.4.md) | ⬜ Todo | Generated Rust World snapshot writer from the shared schema, Go envelope round trip |
| [fn-178.5](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.5.md) | ⬜ Todo | Rust simulation clock: arbiter, progress credits and restart holds |
| [fn-178.6](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.6.md) | ⬜ Todo | Multi-store cluster host: shared engine, isolated per-instance stores, limits and results |
| [fn-178.7](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.7.md) | ⬜ Todo | In-process broker scheduler: continuations, clock, World ordering and cross-instance network |
| [fn-178.8](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.8.md) | ⬜ Todo | In-process incarnation lifecycle: boundaries, revocation, restart holds and recovery-image restart |
| [fn-178.9](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.9.md) | ⬜ Todo | Go cluster client behind WASMExecution with Go-broker reference parity |
| [fn-178.10](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.10.md) | ⬜ Todo | Parallel versus serialized instances: measure and decide |
| [fn-178.11](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.11.md) | ⬜ Todo | Multi-instance gates, in-process memory budgets, speedup record and docs |
| [fn-178.12](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.12.md) | ⬜ Todo | Rust net_stream ABI, cluster network glue and binary network evidence |
| [fn-178.13](.flow/tasks/fn-178-gomad-multi-instance-clusters-in-one.13.md) | ⬜ Todo | Go cluster network configuration and strict decoding of helper network evidence |

<a id="server-consistency-oracles"></a>

### Server consistency oracles — [fn-179](.flow/specs/fn-179-gomad-temporal-server-consistency.md)

Owner decision (2026-10-13): fn-179 starts now, in parallel with the shared-owner queue and with fn-172/fn-173. It has no dependency on the execution-performance specs. Its oracle code sits next to files the rotation batch renames (fn-166.6–.7, .10), so rebase onto whichever lands first.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-179.1](.flow/tasks/fn-179-gomad-temporal-server-consistency.1.md) | 🚧 In progress | Implement soft-assertion oracle alongside shared-owner cleanup |

<a id="replay-instrumentation"></a>

### Replay-time instrumentation — [fn-180](.flow/specs/fn-180-gomad-replay-time-instrumentation-for.md)

Owner decision (2026-10-13): last in the WASM queue, after every other WASM spec; it also depends on fn-174. Not yet planned into tasks.

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-180](.flow/specs/fn-180-gomad-replay-time-instrumentation-for.md) | ⬜ Todo | Stop-at replay with goroutine/heap/registered dumps, dormant probes, invariant bisect |

<a id="quality-assessment-2026-10-01"></a>

## Native

Deferred by owner decision: all unfinished specs and tasks below are outside the active WASM delivery queue. Completed work and recorded Flow states are preserved. Required fixes to shared Gomad owners remain in fn-156's scope; this deferral does not waive WASM acceptance or revive native qualification.

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

<a id="open-findings"></a>
<a id="determinism-gaps"></a>
<a id="f10-follow-ups-deferred-scope"></a>

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

<a id="deep-modules-and-tool-interfaces-fn-109"></a>

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

<a id="runtime-patch-minimization-fn-110"></a>

### Runtime patch minimization — [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md) — Deferred

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-110.3](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.3.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Relocate crypto initialization and syscall declarations to overlays; waits for fn-155.7 |
| [fn-110.4](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.4.md) | ⏸ Deferred (recorded: ⬜ Todo) | Emit the canonical one-context-line patch and pin regeneration to the descriptor |
| [fn-110.5](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.5.md) | ⏸ Deferred (recorded: ⬜ Todo) | Qualify the final candidate and publish measurements and guidance |

<a id="syscall-boundary-fn-155"></a>

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

<a id="append-log-storage-fn-152"></a>

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
| [fn-128.4](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.4.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Linux model, architecture, pack and affected-workload qualification |
| [fn-128.5](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.5.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Actual Linux determinism soak and measured bound |
| [fn-128.6](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.6.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Downstream Linux analyses, packs, exact replay and guidance |
| [fn-128.7](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.7.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Final source-bound Linux matrix, review and documentation |

### Deferred Darwin qualification - [fn-149](.flow/specs/fn-149-gomad-deferred-darwin-qualification.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-149.1](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.1.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Native Darwin baseline, runtime and clock evidence |
| [fn-149.2](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.2.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Darwin model, pack and integration qualification |
| [fn-149.3](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.3.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Real downstream Darwin qualification and guidance |
| [fn-149.4](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.4.md) | ⏸ Deferred (recorded: ⛔ Blocked) | Scheduled/dispatched Darwin soak and final qualification matrix |
