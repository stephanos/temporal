# Programmed workers: the Model issues each worker's program before it runs (model, temporal)

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

**Evidence and judgment.** Server history only. The Describe projection reads `outcome` and `info.last_failure` (the Realization's Describe request sets `include_last_failure`) and projects multi-step paths: a read that jumps from Scheduled to Completed is Poll then RespondCompleted. It projects nothing when the observed result or failure disagrees with the program row for that scenario and attempt; that observation is Unexplained, a finding about worker, SDK or server. The Run records which executor kind and version ran, next to the Driver's namespace and task queue, so a deviation is attributed correctly.

**Derivation.** The program is derived from the one reply policy by enumerating scenarios and admitted attempts. There is no second code path and no agreement check; replay's comparison of the recorded Reset response pins drift. A unit test checks coverage: every scenario and every admitted attempt has a row.

## API Contracts

- `umpire.model.v1`: `ResetResponse.program` (Any, optional, tag 1); `PROPOSAL_KIND_READ = 3`. Existing tags preserved; wire goldens pin both. `METHOD_ANSWER` stays for other systems; `Realize::answer` gains a refusing default and the Temporal Realization drops its implementation.
- `umpire.temporal.v1`: `WorkerProgram { namespace, task_queue, repeated Rule rules }`, `Rule { Any key; repeated Any respond }`, `ActivityTaskKey { scenario, attempt, activity_type }`, the hold instruction field beside the poll fields, and the header name `umpire-scenario` documented on the envelope. `ObserveActivity.worker_tasks` removed.
- `umpire program --model <guest.wasm> --world <cfg> --out <file>`: performs a Reset against a fresh guest and writes the program.
- Go `runner.Driver.Start` unchanged; the Driver factory takes the executor kind and the program bytes from `UMPIRE_TEMPORAL_*` configuration and records the kind in the Run.
- Worker interface, Go: `Follow(ctx, conn, program) error` in `raw` and `sdk`; both import only the Temporal API or SDK and `umpire.temporal.v1`.

## Edge Cases & Constraints

- Same World gives the same program bytes; same state and seed give the same proposal. Goldens pin both.
- A task outside the admitted World (attempt beyond the maximum, another activity) is a scope Refusal, as today; the unprogrammed failure lands consistently with it.
- An unprogrammed task inside the World is a Realization defect: Refused, named "unprogrammed", never a finding about the system.
- Cancel-before-start uses the hold; cancel-after-completion is unchanged. Without a hold the cancel races the worker and either explained outcome is accepted.
- Silence from the worker is a timeout in history, judged as the Model's Deadline actions allow; it is never confused with a program gap because the gap has its own failure type.
- Executors keep no state across tasks. A Model that would need that is wrong, not the worker.
- No non-Go SDK is available offline in this sandbox (Go SDK only in the module cache; no Python SDK or gRPC). R4 is conditional.

## Acceptance Criteria

- **R1:** Reset returns a deterministic program for the admitted World; `umpire program` writes it; wire goldens pin `ResetResponse.program` and `PROPOSAL_KIND_READ`; the Runner and Driver never decode the program (import graph). Errors: a Reset without a configured World returns no program; an unknown key type is refused by the worker at start.
- **R2:** The raw executor follows the program; the live slice (complete, retry-then-complete, fail, cancel-before-start under a hold, cancel-after-completion, host refusal) passes through `umpire explore` with Read proposals, judged from server history, with the Run as the only artifact and the executor kind recorded. The SDK executor then passes the same slice with identical judgments. Errors: an unsupported hold is a DriverFault; a read timeout stays a DriverFault.
- **R3:** A worker that deviates from its program is Unexplained at the deviating observation (control: a mutant raw executor completing where the program says fail, and one completing with a different result). An unprogrammed task ends the Run Refused with the reason named (control: an empty program). A coverage test proves every scenario and admitted attempt has a row.
- **R4:** A raw executor in one language other than Go follows the same program file with no connection to Umpire, if a gRPC stack is available offline; otherwise the missing pieces are recorded in the task.
- **R5:** DESIGN.md's Runner and Run section states R40 in its final form: a worker in any language follows a Model-issued program and the Run records what happened; the Shim is recorded as superseded; the proposal vocabulary (action, bootstrap, read) and the hold fault are recorded.

## Boundaries

- No Model semantics change; the Model decides every worker response, earlier, by enumeration.
- No live connection between a worker and Umpire; no predicate or expression language in programs.
- No workflow or Nexus Model, key type or executor in this spec; the envelope is shaped for them and that is all.
- No heartbeat or delay response kinds until a Model can judge them.
- No Temporal detail in `model/umpire`; the envelope, key types and executors live under `model/temporal`.

## Decision Context

2026-10-23: the owner deferred the Shim and asked for pre-programmed workers; worker actions are the evidence, a misbehaving worker must fail the Model's judgment.

2026-10-24, owner decisions after a walk-through: (1) Read proposal kind; worker actions observed, not proposed. (2) No agreement check; program derived from the one policy, coverage test instead. (3) Scenario tag in the `header`, not the input. (4) Cancel-before-start by a hold fault in the proposal's instruction, not `start_delay` and not a program delay. (5) Program on the Reset response plus `umpire program`. (6) Executors under `model/temporal/hosts/go/reference/worker`, raw and sdk. (7) Placement after fn-228.6 and fn-244.2, before fn-228.7, so one lane edits the Driver at a time and R40 is written once; fn-237 needs the hold fault and follows.

Also settled: the envelope is feature-agnostic with per-feature key types; no fallback field, the unprogrammed behaviour is an executor contract; the worker is part of the system under test and the Run records the executor kind; the program is constant per Model and World with per-Run variation carried in proposed inputs. fn-228.5 is superseded by this spec.
