# Replay-time instrumentation for WASM failures

## Goal & Context

A Gomad failure replays exactly, but today a developer can only see what the run itself printed. Go emits no DWARF for wasm (golang/go#33503 is unplanned), so debuggers cannot show Go variables, and adding a log line changes the binary and breaks the replay. Developers need to inspect a failing run's state at the moment that matters without losing exact replay.

The contract that makes this possible: instrumentation is always compiled in and dormant by default. When enabled at replay, it writes only to a side channel and changes nothing the guest can observe before the inspection point. That means no extra heap allocation, scheduling, clock or entropy reads, WASI I/O or choice consumption. Gomad's existing divergence detection verifies the contract on every instrumented replay.

## Architecture & Data Models

- **Stop-and-inspect.** A replay request may name a stop point: a choice index, a virtual time, or the recorded failure. The helper replays exactly to that point, then answers the scheduler hook with an "enter debug mode" decision. A debug goroutine, created at initialization in every run so recorded and replayed runs stay identical, is woken. It runs the selected dumps and ends the run. After the stop point, determinism no longer matters, so dumps may allocate and run arbitrary Go code.
- **Built-in dumps:** all goroutine stacks with file and line, and a heap profile.
- **Registered dumps:** programs register named dump functions at initialization (for example Temporal shard contexts, mutable-state cache, task queues). The replay request selects them by name.
- **Dormant probes.** Programs declare named probes at initialization; registration order gives each a stable ID. A probe's enabled check is a single load and branch on a host-set bitmask. An enabled probe encodes primitive values and string bytes, without allocating, into a buffer reserved at initialization in every run, and flushes it through a dedicated debug host import. The host decodes and formats the records.
- **Debug side channel:** a host import module separate from WASI and the cooperative runtime imports. Its calls never yield, never read the clock, never enter the WASI transcript or choice trace, and are not part of evidence identity.
- **Go probe library:** an importable package providing probe declaration, emit and dump registration. It is a no-op outside Gomad and on the native backend until that backend opts in.
- **CLI:** `gomad replay` gains stop-point, dump and probe-pattern options. Debug output is written next to the replay result, never into the artifact.

## API Contracts

- Replay request: optional stop point (`choice=N`, `time=T` or `failure`), dump selection (built-in names and registered names), probe patterns. [inferred]
- Debug output: probe records (probe name, ordered typed fields, choice index and virtual time at emission) and dump sections (name, bytes), bounded in size, with an explicit truncation marker. [inferred]

## Edge Cases & Constraints

- An instrumented replay must still match the recorded choice trace and transcript digests up to the stop point; any mismatch is reported as instrumentation divergence, distinct from an ordinary replay divergence.
- A stop point beyond the end of the recorded run, or a choice index or time the run never reaches, is refused before execution.
- Unknown probe or dump names are refused before execution, not ignored.
- Enabled probes consume fuel. If a replay approaches the recorded fuel limit, it reports that rather than mis-classifying the outcome.
- A registered dump that panics or exceeds its byte bound is reported per dump; other dumps still run.
- Output bounds: total debug bytes and per-dump bytes are capped, with truncation markers.
- Debug output is diagnostic only: never evidence, never input to identity, never published as an artifact.

## Acceptance Criteria

- **R1:** `gomad replay` can stop a WASM failure artifact at a choice index, a virtual time or the recorded failure, and write every goroutine's stack (with file and line) and a heap profile at that point. Errors: unreachable stop point or unknown dump name → refused before execution; output over its bound → truncated with a marker.
- **R2:** A program can register named dump functions, and a replay can select them; their output appears in the debug output. A panicking or oversized dump is reported for that dump while the others still run. No error surface beyond R1.
- **R3:** A program can declare named probes; a replay can enable them by pattern and receives their records with choice index and virtual time. Disabled probes cost one load and branch, and enabled probes perform no Go heap allocation. Errors: unknown probe pattern → refused before execution; probe buffer exhaustion → records dropped with a counted marker, never a guest-visible effect.
- **R4:** Every instrumented replay verifies that the choice trace and transcript digests match the recording up to the stop point. A deliberately perturbing test probe (one that allocates) is reported as instrumentation divergence. No error surface beyond the divergence report.
- **R5:** Given a registered invariant check, `gomad replay` can binary-search the stop point for the first choice at which the check fails, using repeated exact replays. Errors: an invariant that already fails at the first choice, or never fails, is reported as such.
- **R6:** At least one Temporal registered dump (shard and mutable-state summaries) and one probe exist and are demonstrated on a retained Temporal failure artifact. No error surface beyond R1–R3.

## Boundaries

- Debug logs formatted ahead of time (always formatting debug log lines during exploration so they can be captured on replay) are out of scope; they are a possible later campaign mode.
- No DWARF, no debugger attachment and no Go-aware heap walker over raw memory snapshots.
- No instrumentation of the native backend in this spec.
- No change to what recording does: recorded runs never enable probes or dumps.

## Decision Context

Owner request (2026-10-13), following the finding that the WASM backend gives function and line visibility but no variable inspection, and that upstream Go has no plan for wasm DWARF. Stop-and-inspect comes first because one replay to a point plus arbitrary dumps answers most debugging questions. Probes follow for traces leading up to the failure. Ordered after all other WASM specs.
