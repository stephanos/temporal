# Programmed workers: the Model issues each worker's program before it runs (model, temporal)

> HTML render lens (local, ignored): open `.flow/artifacts/fn-243-programmed-workers-the-model-issues/spec.html` locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (2026-10-23): "skip the Shim for now"
> user (2026-10-23): "let's think about this more; we'll want workers to know what to do before they exdcute. ie we want to "pre-program" them. because for example in a cloud setup, there won't be a connection between worker and upire/oracle"
> user (2026-10-24): "some features when they are new don't hve a worker impl; they need a "fake worker". how do we represent that?"
> user (2026-10-24): "that's too specific to activities. it needs to scale to anything, like kitchensink."
> user (2026-10-24): "it would be acceptable to use CEL if that helps. actually that's not true, no CEL"
> user (2026-10-24), confirming the vision: raw worker responses for every feature; scripted responses where an SDK exists; one pre-programmable handler per feature driven by the program; the program's only logic is to tell scenarios apart within one worker.
> user (2026-10-24): "could we use a delay as a fault to ensure we hold the request in the right place before the worker gets it?"
> Decisions 1 to 7 below were walked through one by one on 2026-10-24 and each accepted with "ok".

## Goal & Context

Replace R40's live worker connection (Answer in process, the Shim for other languages) with a program the Model issues before any worker runs. A worker follows the program with no connection to Umpire; its actions reach the server; the Model judges the server's history against the program it issued. Design of record: `model/design/programmed-workers.md`.

The in-process Go worker follows programs too, so the live Answer path for workers retires along with the Shim.

## Architecture & Data Models

The terminal-program envelope, activity-row and derivation descriptions below record the delivered .1/.2 checkpoint. Remaining .5 work consumes fn-246's evolved key/codec, observable-worker policy, registry/lens and whole-prefix witness contracts through committed fn-237.4. Preserve that checkpoint's proof and provenance while reanchoring current interfaces against those committed prerequisites.

**A program is a table, not a script and not a language.** A program maps what a worker can see when a task arrives to what it sends back. Keys are exact matches over a few task fields; there is no predicate language (R35 removed CEL from Go, and none returns here). A behaviour the key cannot express is not programmed: the Realization refuses it.

**The program is a property of the Model and the World.** It is issued once, on Reset, and is constant across Runs and seeds: `ResetResponse` gains `google.protobuf.Any program = 1`, optional. The variation between Runs travels in the inputs the Model proposes. A worker provisioned once from `umpire program` serves any number of Runs. Every Run carries the program in its Reset event, so replay judges against the program that was in force. `ProposeResponse` is unchanged.

**Envelope (`umpire.temporal.v1.WorkerProgram`).** Temporal-wide and feature-agnostic: guards (`namespace`, `task_queue`) and ordered `rules`, each a `key` Any and `respond` Anys. The first rule whose key matches the task wins. Each Temporal feature defines its own key and response types in its own proto file; the envelope never changes for a new feature. A worker handed a key type it does not understand refuses the whole program at start; that is the version check.

**Scenario tag.** The Realization stamps `umpire-scenario` into the activity or workflow `header` when it fills a Start; the worker reads it from the task. The scenario is Umpire's instruction to the worker and stays out of the application input. The activity Model's input text (Complete, RetryThenComplete, Fail) moves to this header.

**Activity feature.** Key `ActivityTaskKey { scenario, attempt }` with `activity_type` as a guard; responses are the raw `RespondActivityTaskCompletedRequest` or `RespondActivityTaskFailedRequest` the Realization already builds, token and identity left for the worker. One row per scenario and admitted attempt; the activity Model's program is at most nine rows.

**Workflow feature (vision, not in this spec).** One rule: for this workflow type, execute the action set carried in the input. The kitchen sink worker is the SDK executor. Recorded so the envelope is shaped for it; no workflow Model exists yet.

**Unprogrammed tasks.** A fixed executor contract, not a program field: a task no rule matches is failed non-retryably with failure type `umpire.unprogrammed`. The Realization refuses on sight. The fallback is not configurable so a program cannot mask a gap.

**Executors.** Two kinds behind one worker interface per language: `raw` polls with the plain gRPC client and sends the response message as is; `sdk` registers one generic handler per feature with the Temporal SDK. Raw first for activities (the only option for a feature without SDK support, the shortest proof of "no connection", and a byte-level comparison against the program); `sdk` second, with a parity control that both produce identical judgments on the live slice. `respond` may take a different form per executor kind where a key type says so (a workflow script is not a raw command). Executors live under `model/temporal/hosts/go/reference/worker/{raw,sdk}` following the Kettle precedent: the worker is part of the system under test, and the import graph proves the Driver never reads the program.

**Proposals.** `ProposalKind` gains `PROPOSAL_KIND_READ = 3`: a Describe loop with no committed action, pending until the Realization's continue-polling says a change was explained. Worker actions are observed, never proposed. Cancel stays an Action proposal. The Driver's worker gate and the `worker_tasks` instruction field are deleted.

**Hold as a fault.** A proposal's instruction may ask the Driver to hold worker polls from reaching the server until the proposed request is acknowledged. The Realization fills it deterministically from the seed; the Model does not model it. The Driver implements it as transport mechanics on a connection it owns; for a worker it does not own, an unsupported hold is a DriverFault and the Run is Refused, never a verdict. This replaces the gate for cancel-before-start and is the mechanism fn-237's schedule-to-start timeout needs.

**Hand-off to fn-237.** Heartbeat and cooperative cancellation need two more activity response kinds (`RecordActivityTaskHeartbeat`, `RespondActivityTaskCanceled`) and one more situation a worker can see (the cancel-requested bit a heartbeat response reveals) as a key field on a heartbeat rule. Existing fn-246.14 implements the approved activity heartbeat key/codec/raw-and-SDK continuation capability earlier; fn-237.4 consumes its verified result and retains actual evidence/journal/gates, timeout/cooperative admission and every original mandatory raw/native-SDK live witness. The activity key and response types evolve there; the envelope does not change. The schedule-to-start timeout is realized with the hold (worker polls held past the budget), never with a delayed or sleeping worker.

**Evidence and judgment at the delivered terminal-only checkpoint.** Judgment uses actual server history. The Describe projection reads `outcome` and `info.last_failure` (the Realization's Describe request sets `include_last_failure`) and projects multi-step paths: a read that jumps from Scheduled to Completed is Poll then RespondCompleted. It projects nothing when the observed result or failure disagrees with the program row for that scenario and attempt; that observation is Unexplained, a finding about worker, SDK or server. The Run records which executor kind and version ran, next to the Driver's namespace and task queue, so a deviation is attributed correctly.

For remaining cooperative proof, .5 consumes fn-246's feature-local projection and whole-prefix witness contract plus fn-237.4's actual ActivityEvidence journal. Describe supplies phase/outcome/last-failure evidence. Correlated actual heartbeat responses and terminal RPC acknowledgements establish cancellation notification and explicit worker acknowledgement, with actual native-SDK cancellation delivery proved separately. Program rows and terminal Describe alone supply neither proof. The Run retains the evidence used by judgment and server-free replay.

**Derivation.** The program is derived from the one reply policy by enumerating scenarios and admitted attempts. There is no second code path and no agreement check; replay's comparison of the recorded Reset response pins drift. A unit test checks coverage: every scenario and every admitted attempt has a row.

## API Contracts

- `umpire.model.v1`: `ResetResponse.program` (Any, optional, tag 1); `PROPOSAL_KIND_READ = 3`. Current tags/layouts may change; regenerate wire goldens and consumers together. `METHOD_ANSWER` stays for other systems; `Realize::answer` gains a refusing default and the Temporal Realization drops its implementation.
- `umpire.temporal.v1`: `WorkerProgram { namespace, task_queue, repeated Rule rules }`, `Rule { Any key; repeated Any respond }`, `ActivityTaskKey { scenario, attempt, activity_type }`, the hold instruction field beside the poll fields, and the header name `umpire-scenario` documented on the envelope. `ObserveActivity.worker_tasks` removed.
- `umpire.run.v1.Run` gains a worker record (executor kind, executor version, namespace, task queue) in the current format; regenerate wire goldens and consumers together. The fake Driver fills it with its own kind. `Event.answer` stays for other systems.
- `umpire program --model <guest.wasm> --world <cfg> --out <file>`: performs a Reset against a fresh guest and writes the program.
- Go `runner.Driver.Start` unchanged; the Driver factory takes the executor kind and the program bytes from `UMPIRE_TEMPORAL_*` configuration and records the kind in the Run.
- Worker interface, Go: `Follow(ctx, conn, program) error` in `raw` and `sdk`; both import only the Temporal API or SDK and `umpire.temporal.v1`.

## Edge Cases & Constraints

- Same World gives the same program bytes; same state and seed give the same proposal. Goldens pin both.
- A task outside the admitted World (attempt beyond the maximum, another activity) is a scope Refusal, as today; the unprogrammed failure lands consistently with it.
- An unprogrammed task inside the World is a Realization defect: Refused, named "unprogrammed", never a finding about the system.
- Cancel-before-start uses the hold; cancel-after-completion is unchanged. The original requirement, "Without a hold the cancel races the worker and either explained outcome is accepted," remains an outstanding R2 obligation. The atomic-Model checkpoint refuses unheld cancellation of an open activity before issuing a Cancel RPC. Task .5 restores admission and proves the original obligation after the independently planned cooperative Model and live binding are committed.
- Silence from the worker is a timeout in history, judged as the Model's Deadline actions allow; it is never confused with a program gap because the gap has its own failure type.
- Executors keep no state across tasks. A Model that would need that is wrong, not the worker.
- No non-Go SDK is available offline in this sandbox (Go SDK only in the module cache; no Python SDK or gRPC). R4 is conditional.

## Acceptance Criteria

- **R1:** Reset returns a deterministic program for the admitted World; `umpire program` writes it; wire goldens pin `ResetResponse.program` and `PROPOSAL_KIND_READ`; the Runner and Driver never decode the program (import graph). Errors: a Reset without a configured World returns no program; an unknown key type is refused by the worker at start.
- **R2:** The raw executor follows the program; the live slice (complete, retry-then-complete, fail, cancel-before-start under a hold, cancel-after-completion, host refusal) passes through `umpire explore` with Read proposals, judged from server history, with the Run as the only artifact and the executor kind recorded. The SDK executor then passes the same slice with identical judgments. After the committed corrected cooperative Model and live binding, deterministic actual raw and native-SDK unheld cancellation witnesses prove Cancel while Started, pending state, completion/failure races, explicit worker acknowledgement and distinct repeated-request classification. Unsupported combinations refuse before effects. This original unheld obligation remains open until .5's committed live and server-free replay proof. Errors: an unsupported hold is a DriverFault; a read timeout stays a DriverFault.
- **R3:** A worker that deviates from its program is Unexplained at the deviating observation (control: a mutant raw executor completing where the program says fail, and one completing with a different result). An unprogrammed task ends the Run Refused with the reason named (control: an empty program). A coverage test proves every scenario and admitted attempt has a row.
- **R4:** A raw executor in one language other than Go follows the same program file with no connection to Umpire, if a gRPC stack is available offline; otherwise the missing pieces are recorded in the task.
- **R5:** DESIGN.md's Runner and Run section states R40 in its final form: a worker in any language follows a Model-issued program and the Run records what happened; the Shim is recorded as superseded; the proposal vocabulary (action, bootstrap, read) and the hold fault are recorded.

## Quick commands

```bash
cd model && mise exec -- cargo test -p umpire-temporal --test program_history --test proposal
```

## Boundaries

- No Model semantics change; the Model decides every worker response, earlier, by enumeration.
- No live connection between a worker and Umpire; no predicate or expression language in programs.
- No workflow or Nexus Model, key type or executor in this spec; the envelope is shaped for them and that is all.
- No heartbeat or delay response kinds until a Model can judge them.
- No Temporal detail in `model/umpire`; the envelope, key types and executors live under `model/temporal`.

## Cancellation admission checkpoints

The delivered fn-243 live binding remains admitted against the atomic cancellation World. The independently extended activity Model already represents cooperative cancellation; live admission remains deferred until the corrected contract and binding prerequisites are committed. An actual retained SDK Run demonstrates that the server can acknowledge cancellation of a started attempt while leaving cancellation pending, permit completion or failure, and reject a distinct second cancellation request. The checkpoint does not reinterpret that evidence or change the Model to match it.

Task .2 completes the admitted held-before-start and terminal-cancellation slice. It explicitly refuses unheld cancellation of an open activity with `unheld cancellation of an open activity is outside the atomic cancellation Model`; the refusal publishes no cancellation proposal and causes no Cancel RPC. Ordinary supported Runs retain strict explanation and program-deviation controls.

The original unheld-race requirement stays open under R2. Task .5 consumes the independently specified cooperative semantics and committed live binding from fn-237.4, including fn-246.15's timeout-retry correction and fn-246.13's explicitly approved pending-cancellation Failed/TimedOut correction with renewed native/TLC proof. Scheduled cancellation and explicit worker acknowledgement retain Canceled. The conductor verifies that fn-237.4 consumed committed whole fn-246 and fn-247 before admitting .5. Task .5 restores only justified admission and proves actual raw/SDK races; it authors no further Model policy. The original counterexample retains its semantic finding and provenance, including historical guest identity; later Model support is a separate proof. Current readers need not load its historical format; old binary/hash replay is not a gate.

Task .4 and fn-228.7 document the delivered worker architecture and this admission boundary. Neither closes this spec or claims the deferred behavior passed. Task .5 updates the final R40 status and fn-228 R7 evidence after its live and offline-replay proof passes. Both parent obligations remain open until that committed evidence exists. The CLI supports native dependencies only within a spec. Cross-spec prerequisites use explicit blocked tasks and conductor admission checks against committed closures, without whole-spec edges that would block the producer of the cooperative semantics.

## Decision Context

2026-10-24, owner amendment: "can we drop byte-for-byte compatability from open tasks to speed things up? we don't need backwards compatabiltiy". Remaining work may change guest/wire/API formats and regenerate bindings/goldens together. Old-format readers, old-byte/hash pins and historical-layout preservation are unnecessary. Preserve semantic Model/refinement obligations, the meaning and provenance of original lifecycle/counterexample evidence, current source/guest freshness and identity, deterministic replay within the current format, actual raw/native-SDK live proofs and discriminating negative controls. This supersedes older compatibility-only restrictions without changing completed task records.


2026-10-23: the owner deferred the Shim and asked for pre-programmed workers; worker actions are the evidence, a misbehaving worker must fail the Model's judgment.

2026-10-24, owner decisions after a walk-through: (1) Read proposal kind; worker actions observed, not proposed. (2) No agreement check; program derived from the one policy, coverage test instead. (3) Scenario tag in the `header`, not the input. (4) Cancel-before-start by a hold fault in the proposal's instruction, not `start_delay` and not a program delay. (5) Program on the Reset response plus `umpire program`. (6) Executors under `model/temporal/hosts/go/reference/worker`, raw and sdk. (7) Placement after fn-228.6 and fn-244.2, before fn-228.7, so one lane edits the Driver at a time and R40 is written once; fn-237 needs the hold fault and follows.

Also settled: the envelope is feature-agnostic with per-feature key types; no fallback field, the unprogrammed behaviour is an executor contract; the worker is part of the system under test and the Run records the executor kind; the program is constant per Model and World with per-Run variation carried in proposed inputs. fn-228.5 is superseded by this spec.

2026-10-24, conductor under the owner's standing autonomous-work instruction: preserve the frozen atomic Model and the actual cancellation counterexample. Scope-refuse unsupported unheld open-state cancellation in .2, keep its original R2 obligation in .5, and consume the already-approved fn-237 cooperative Model rather than fitting semantics to the server. The interleaving is .1-.3, fn-228.7, .4, fn-237.1-.4, .5, then fn-237.5.


The current prerequisite chain for remaining work is committed fn-237.3, committed whole fn-247, committed whole fn-246 including .13's approved correction and renewed proof, committed fn-237.4, then fn-243.5 and fn-237.5. This updates remaining admission checks without changing completed records, the .5 block or the historical interleaving decisions above.

## Verification adoption

Use `make model-validate` for remaining final validation. It runs one complete gate with TLC required and one inner-loop benchmark. Apalache remains independently opt-in; none of these prior plans required it. Keep all standalone focused controls, actual raw/SDK live slice/gaps/replay and SUT proofs, and preserved original semantic counterexample evidence. Label new gate timing as required-mode and historical default-mode timing as historical; retain actual command provenance. Current focused commands remain available. No TLC batching, caching or reduced case coverage is introduced.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Reset returns a deterministic program for the admitted World; `umpire program` writes it; wire goldens pin `ResetResponse.program` and `PROPOSAL_KIND_READ`; the Runner and Driver never decode the program (import graph). Errors: a Reset without a configured World returns no program; an unknown key type is refused by the worker at start. | fn-243-programmed-workers-the-model-issues.1 | — |
| R2 | The raw executor follows the program; the live slice (complete, retry-then-complete, fail, cancel-before-start under a hold, cancel-after-completion, host refusal) passes through `umpire explore` with Read proposals, judged from server history, with the Run as the only artifact and the executor kind recorded. The SDK executor then passes the same slice with identical judgments. After the committed corrected cooperative Model and live binding, deterministic actual raw and native-SDK unheld cancellation witnesses prove Cancel while Started, pending state, completion/failure races, explicit worker acknowledgement and distinct repeated-request classification. Unsupported combinations refuse before effects. This original unheld obligation remains open until .5's committed live and server-free replay proof. Errors: an unsupported hold is a DriverFault; a read timeout stays a DriverFault. | fn-243-programmed-workers-the-model-issues.2, fn-243-programmed-workers-the-model-issues.5 | — |
| R3 | A worker that deviates from its program is Unexplained at the deviating observation (control: a mutant raw executor completing where the program says fail, and one completing with a different result). An unprogrammed task ends the Run Refused with the reason named (control: an empty program). A coverage test proves every scenario and admitted attempt has a row. | fn-243-programmed-workers-the-model-issues.2 | — |
| R4 | A raw executor in one language other than Go follows the same program file with no connection to Umpire, if a gRPC stack is available offline; otherwise the missing pieces are recorded in the task. | fn-243-programmed-workers-the-model-issues.3 | — |
| R5 | DESIGN.md's Runner and Run section states R40 in its final form: a worker in any language follows a Model-issued program and the Run records what happened; the Shim is recorded as superseded; the proposal vocabulary (action, bootstrap, read) and the hold fault are recorded. | fn-243-programmed-workers-the-model-issues.4, fn-243-programmed-workers-the-model-issues.5 | — |
