# fn-254.2 TLC batching design handoff

Read-only design findings from the current runner, emitter, decoder, process support and existing controls. This report is scratch input for implementation. It supplies no review verdict, execution evidence or measured performance result. No source, Model semantics, Flow metadata, builds, tests or commits changed during this investigation.

## Decision

Keep claim-free exploration, batch the properties within one existing check configuration, and use isolated executions for canonical refutations. A complete successful batch can establish every selected property's Verified result. A valid violation splits the failed batch into singleton runs. Keep the existing isolated path for configurations with one property.

This is the smallest first implementation. Do not combine distinct progress specifications, change observer construction, alter authored fairness or introduce semantic defaults. Further batching of distinct configurations requires a demonstrated need and exact equivalence evidence.

## Configuration boundaries and expected launch reduction

`model/umpire/backends/tlc/src/runner.rs:76` checks the main configuration, then each progress configuration, then each negative configuration. Preserve that order. Keep `states` and `raw_state_count` from the main unrestricted exploration.

`model/umpire/backends/tla/src/emit.rs:212` emits the main configuration containing invariants and postconditions. Each progress claim has a separate `ProgressSpec_<claim>`, with its own activation, scope, counters and authored fairness (`emit.rs:265-360`). Negative claims also retain their existing isolated configurations. Module admission already checks selected World assignments and registered property identity in `model/umpire/backends/tlc/src/module.rs:100`.

The activity source currently has three invariants, eight postconditions and six progress claims. Its existing main check therefore predicts twelve model JVM launches, and the six scoped progress checks predict another twelve. Healthy batching predicts two plus twelve, or **24 to 14 model JVM launches per World**. This is a code-derived prediction, not a measurement. The baseline must measure actual launches, including admission's `java -version` and TLC `-h` probes separately. The two-launch healthy target applies per existing check configuration, not per entire module.

## Minimal runner structure

Reuse `split_cfg`, `parse`, `failure`, `selected_violation`, `located`, `distinct_count`, `decode_dump` and `decode_trace` in `runner.rs` and `decode.rs`.

1. Validate the schema, module name and positive execution limits as today.
2. Split the input cfg without changing base specification, constants or declaration order.
3. Execute claim-free exploration first. Require its completed statistics, exhausted queue and exactly decoded dump before property execution.
4. Zero properties return the exploration result. One property uses the existing singleton execution directly.
5. Multiple properties run in one cfg containing all original ordered declarations.
6. A complete successful batch produces Verified for all selected operators, with no traces.
7. A valid property violation triggers isolated runs of the properties in original cfg order. Return those singleton verdicts and traces. This costs one extra property launch on a failing batch and preserves the old counterexample route.

If later measurements justify a narrower fallback, isolate a named failed property and batch the remainder. Obtain the returned refutation trace from the singleton execution even then. Start with the simple complete singleton fallback.

`execute` currently uses `None` to mean exploration and rejects all violations. Use an explicit internal exploration/singleton/batch selection or a small wrapper, rather than reinterpreting `None` as batch. Keep public `TlcResult`, `PropertyResult` and `Verdict` shapes unchanged.

## Why isolated refutations are required

Safety attribution alone does not prove that a batch's chosen counterexample equals the separately executed property's chosen counterexample. Return the singleton trace to satisfy exact assignment, private observer and lasso equality without assuming batch traversal stability.

`selected_violation` at `runner.rs:251` handles liveness message 2116 through the selected property passed by its caller. It extracts no property identity from that message. A failed multi-property liveness batch therefore cannot safely attribute its trace through the existing protocol. Mixed safety/liveness batches can establish all-pass results; failures split.

Preserve `located` for a bounded progress property's state-level conjunct. TLC can report its source span as an invariant violation instead of reporting the top-level PROPERTY operator name. Named violation reports must resolve to a selected group member; unknown names remain failures.

## Evaluation and incomplete-output behavior

`failure` at `runner.rs:417` uses the selected singleton property's identity for unnamed evaluation failures. That fallback is invalid for a group. A batch evaluation failure must recover the old typed error through isolated properties in original order, rather than silently labeling it an initial-predicate or next-state failure. Preserve the earliest error in the old property order.

Malformed messages, unsupported errors, invalid violation attribution, missing completion/success markers, timeout, unsafe files and output/dump limits remain refusals. They never verify unresolved properties. Execution-limit failures should return the refusal rather than starting unbounded retries. An attributed group violation or recoverable group evaluation ambiguity can split into fresh, fully admitted singleton executions.

Keep existing subprocess output, dump and aggregate check/module output bounds. Append retained execution output in deterministic plan order. Raw output bytes necessarily differ from the independent runner and are execution evidence, not semantic equality fields.

## Scratch, concurrency and heap ownership

Every execution owns one `Scratch`, its module, `Run.cfg`, TLC metadir, dump, stdout and stderr. The current single check scratch shares `Run.cfg` and `states/`; it cannot host concurrent executions. Repeated batch labels also need separate scratches because subprocess capture files use `create_new(true)`.

Reuse `Scratch::finish`, and join all owned workers before returning so cleanup failures remain visible. Keep exploration ahead of property checks initially to preserve failure precedence. Bounded worker execution can parallelize singleton fallback. Collect results and outputs by property order, not completion order, and return the earliest ordered error after owned workers drain.

Expose the task's execution limit through the existing limits/settings convention. A small core-derived cap is reasonable; two runner slots is a candidate default requiring measurement. Coordinate it with nextest's JVM group in task .3. The combined budget is concurrent test processes times runner slots times heap per JVM. For example, two test processes times two runner slots permits four model JVMs and 2 GiB of configured Java heap at today's 512 MiB default, plus JVM native memory. A process-local semaphore cannot cap separate nextest processes or simultaneous gate invocations; the gate owns that invocation policy.

Measure peak Java processes including admission probes. Existing process support already tests concurrent supervisors under allocator/spawn pressure in `model/umpire/backends/process/tests/process.rs:232`; reuse its supervision and cleanup path.

Keep current `-workers 1`, SerialGC and disk fingerprint flags until task .1 measures alternatives on real fixtures. An in-memory fingerprint set changes large-model resource behavior and needs corresponding evidence.

## Equivalence control and test reuse

Provide an explicit development feature or equivalent test switch that runs the old independent strategy alongside optimized checks. Place the comparison below both direct `check` calls and `check_module`, so existing fixtures and mutants participate without bespoke copies of their callers.

Compare authored reachable states, raw assignment count, exact property operator set and verdict, every trace state's authored assignment and private observers, trace length and exact `back_to`. For typed evaluation cases, compare property, message, state and trace. `TraceState` already implements equality. Exclude only raw subprocess `output` from semantic equality.

Reuse these current routes:

- `model/umpire/backends/tlc/tests/controls.rs` covers maximality, fairness, finite dead ends, authored/rejecting self-loops, bounded progress, initial invariants, action properties, deadlocks, tool errors and located evaluation failures.
- `model/umpire/backends/tlc/tests/scaling.rs` proves source-lifted TLC remains independent of a cut native reference.
- `model/umpire/dev/fixtures/fixtures.rs` runs every admitted shared TLC fixture through generated discovery. Existing backend replay at `model/umpire/dev/testing/src/backend.rs:122-153` checks typed counterexamples against the compiled Model.
- Existing `model/umpire/backends/tla/tests/**` and activity source-mutant controls call `check` or `check_module`. The comparison switch should cover these calls directly.

Add focused controls for two failed invariants at different depths; failed invariant plus action property; passing safety plus failed liveness; two failed liveness properties; fairness excluding one lasso while another remains; bounded-progress located violations; a batch evaluation error requiring original-order recovery; and malformed/truncated/incomplete batch output. Retain exact rejecting-self-loop outcome/fact observers and lasso checks.

Assert zero-claim, singleton and healthy multi-property launch counts of one, two and two respectively. Count progress/negative configurations explicitly. Test concurrent scratches, deterministic output/result ordering, limits, cleanup and measured peak JVM ownership.

## Task .6 cache boundary

Cache identity covers the complete ordered effective main/progress/negative configurations and decoding schema, alongside module bytes, jar SHA, Java identity and runner/serialization version. Main `.tla` and `.cfg` alone omit separately stored configuration inputs. Cache only complete all-verified module results; full-gate cache disabling remains authoritative. This batching task should keep cache work within .6's ownership.
