# Programmed workers: the Model issues each worker's program before it runs (model, temporal)

## Conversation Evidence

> user (2026-10-23): "skip the Shim for now"
> user (2026-10-23): "let's think about this more; we'll want workers to know what to do before they exdcute. ie we want to "pre-program" them. because for example in a cloud setup, there won't be a connection between worker and upire/oracle"
> assistant: proposed Model-issued per-proposal worker programs keyed by what a worker sees, judged afterwards from server history; open questions on evidence source, program scope and the in-process Go worker.

## Goal & Context

Replace R40's live worker connection (Answer in process, the Shim for other languages) with programs the Model issues before execution. See `model/design/programmed-workers.md`.

## Architecture & Data Models

- `ProposeResponse` gains an optional program `Any`; the Temporal Realization fills a typed worker program keyed by activity type, activity id, attempt and an optional input digest, with actions complete/fail/heartbeat/delay and a fixed unprogrammed fallback.
- The Temporal Driver delivers the program to the worker and observes; it never reads the program.
- Evidence: the server's history only; worker actions are observed through their RPCs and the timeouts their silence causes.
- The Model judges observed worker behaviour against its program: a deviation is Unexplained (a finding about worker, SDK or server). A program that disagrees with the Model's own answer is a Realization defect, checked at issue and on replay.
- The in-process Go worker follows programs; live worker Answer retires.

## Acceptance Criteria

- **R1:** Propose returns a deterministic program alongside the request; wire goldens pin it; Go never interprets it.
- **R2:** The in-process Go worker follows programs; the live slice (complete, cancel, retry-then-complete, fail, host refusal) passes through `umpire explore` with programs, judged from server history.
- **R3:** A worker that deviates from its program is judged Unexplained at the deviating observation; an unprogrammed task ends the Run Refused; a program that disagrees with the Model's own answer is caught at issue and on replay. Controls for all three.
- **R4:** A worker in one language other than Go follows a program file with no connection to Umpire, if an SDK is available offline; otherwise recorded as not yet proven.
- **R5:** DESIGN.md rewrites R40 and records the superseded Shim.

## Boundaries

- No Model semantics change; the Model decides every worker response, earlier.
- No live connection between a worker and Umpire.

## Decision Context

2026-10-23: the owner deferred the Shim. Owner, same day: worker actions are the evidence (no worker logs), and a misbehaving worker must fail the Model's judgment. Earlier and asked for pre-programmed workers. The design note records the conductor's recommendations on the open questions. Placement in the queue awaits the owner; fn-228.5 stays blocked and is superseded once this lands.
