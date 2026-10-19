# Mini activity vertical slice: the Model judges and answers one real SDK run (model, testpilot)

> Local HTML render lens: `.flow/artifacts/fn-216-mini-activity-vertical-slice-the-model/spec.html`. Regenerable; markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (2026-10-19): "dor the temporla-intergration; what;s the smallest thing we can do next? or should we focus on the DSL itself first? propose an ordering. and propose a scope of the vertical slice integraiton. we'll want to test a part of a single feature (keep it small) including the SDK." Then "ok" to the proposed slice.

The owner chose a Model interface, exactly three envelopes over upstream Temporal API messages, and a strict boundary keeping every Temporal detail outside Umpire. Testpilot's schema stays untouched.

## Goal & Context

A real Go SDK activity will ask the Rust Model what to return, and the same Model instance will judge the server's observations. Completed and non-retryable failed runs prove this boundary without building the broader Case slice. The Model is fixed independently before the live test. A conformance failure requires human review rather than a Model change to fit the server.

The archived mini activity product supplies the starting semantics. This slice is an explicit six-phase restriction, not an exact port of its seven-phase, parameterized and retry-aware Model. The pure Model retains cancellation and abstract deadlines; the live slice exercises completion and failure only.

## Architecture & Data Models

The owner-fixed placement boundary is `model/umpire/` for system-agnostic machinery and `model/temporal/` for the extension, Models, realizations, generated Temporal API code, guest and extension Go module. Nothing from Testpilot's proto is moved, imported or copied.

The dynamic Model is one `phase` field with Scheduled, Started, Completed, Failed, Canceled and TimedOut. Its six state-changing actions are Client::Cancel, Worker::Poll, Worker::RespondCompleted, Worker::RespondFailed, Deadline::ScheduleToStart and Deadline::StartToClose. Deadline actions are internal abstract events, without a clock. Successful Start establishes the initial Scheduled condition outside this phase-only graph.

Poll takes Scheduled to Started. Completion and fatal failure take Started to their respective closed phases. Cancel takes Scheduled or Started to Canceled. ScheduleToStart applies in Scheduled; StartToClose applies in Started; both reach TimedOut. Closed phases have no successors. Atomic cancellation and fatal-only failure deliberately omit the archive's cancelRequested and retry branches.

Register `terminal_remains`. Because emitted-step checking cannot exercise its terminal antecedent when terminal states have no outgoing edges, supplement the report with exhaustive terminal-state/action checks using the same preservation predicate. A test-only reopening edge must refute that predicate. Add no production edge to manufacture coverage.

The generic reusable guest library owns buffers, dispatch and one run's state. A system guest supplies its Model, realization and Any registry. The generic realization interface distinguishes a positively matching current-state observation from candidate transitions and unsupported input. Judgment replays actual `Machine::step` results. Expected actions come from enabled Model actions, including an empty list at a closed phase.

The generic Go host carries core messages and opaque Any payloads only. The Temporal Go module owns the three generated envelopes. Local require/replace wiring connects the server test to both nested modules, so editing either side requires no push or publication.

## API Contracts

Rename `umpire.oracle.v1` to `umpire.model.v1`, including generated namespaces and the generic host module name `umpire.local/model`. Preserve existing method numbers, protobuf field tags, refusal numbers, request/response bytes and the three ABI signatures. New operations receive distinct method IDs. Namespace metadata changes are explicit; old namespace Any type URLs are not aliases for new ones.

Add Reset, Judge and Answer over generic core messages and `google.protobuf.Any`. Judgment is Explained with an explicitly representable zero-or-more-step explanation, or Unexplained with expected actions. Answer carries an opaque system reply. System-specific completion/failure variants belong in the Temporal payload, never the core interface.

The Temporal extension has exactly three messages:

- `Observation { method, response: Any }` carries the upstream RPC response.
- `Callback { task: Any }` carries a partial upstream PollActivityTaskQueueResponse.
- `Reply { request: Any }` carries a partial upstream RespondActivityTaskCompletedRequest or RespondActivityTaskFailedRequest.

Unknown or wrong outer/inner type URLs, malformed bytes, absent required projection fields, unsupported methods and unknown enum values are typed refusals. Rust compiles the real Temporal API descriptor closure with already-admitted prost tooling. The descriptor set is a generated, ignored build artifact, not a checked-in source file or a permanently fixed file count. Generation must run on a clean checkout. Use generated well-known types with one shared owner rather than silently introducing a new runtime library.

Each run's Start input contains one binary/plain Payload naming Complete or Fail. The SDK activity receives those bytes and constructs the declared partial poll-response projection through the existing data converter and available GetInfo fields. The realization validates the policy and corresponding enabled Model response action. Unknown policy, encoding or payload cardinality is refused. Go branches only on the Model's returned RPC type, translating failure to a non-retryable ApplicationError. Task tokens, identity and timestamps remain SDK/server-owned; whole-request equality is not claimed.

A valid callback supplies delivery evidence for Poll and advances Scheduled to Started. Answer selects a reply while leaving the phase Started. A later observed terminal response takes RespondCompleted or RespondFailed. Validate the complete callback and reply choice before committing delivery; a refused callback cannot partially advance state. Every stateful method preserves state when response encoding/allocation fails.

Describe projection uses actual Info.Status and Info.RunState; there is no Started execution-status enum. A repeated observation is Explained with zero steps only when its validated projection equals the current phase. A terminal observation before delivery is Unexplained; judgment must not invent Poll. Contradictory recognized observations are Unexplained. Refusals preserve state. The first Unexplained freezes the run until Reset; subsequent observations cannot resume advancement and callbacks are refused.

The real test uses one host-owning loop. Register the SDK worker without starting it, start the activity, and synchronously judge initial Scheduled observations. Then start the worker, receive its callback through a channel and return the Model's Answer. Only afterward enter synchronous Describe-to-Judge polling. This prevents an in-flight pre-delivery read from being reinterpreted after delivery. A stale Scheduled observation after delivery remains Unexplained. Cancellation and cleanup must release any activity waiting for a reply.

## Edge Cases & Constraints

- Reset restores the start and clears the frozen result. Separate guest instances and reset runs cannot share state. Repeated delivery cannot take Poll twice.
- Run the mutation control after the positive terminal assertion or in a separate reset run. An opposite terminal projection after closure is a recognized impossible observation.
- Compile and exercise the real generated descriptor closure natively and as WASM. Stub generation or a mocked SDK server does not prove this slice.
- The generic boundary gate scans .rs, .proto, .go and .toml files for the promised API prefixes and case-sensitive temporal/Temporal occurrences, including identifier substrings, comments and literals. Its only exception is the exact temporalio organization segment in module paths. It reports each offending file and line. Keep policy literals/negative controls on the system side so the checker needs no self-exclusion.
- The full model gate keeps the 10 s inner and 30 s full soft targets. Record minor noise; investigate material overruns. Measure the live server test separately.
- The real test belongs in the repository's server-test package. Testpilot's dependency scanner includes external-package test files and forbids its packages from importing the real server harness. Keep that gate intact.
- A dedicated `umpire-check-model-slice` target builds the guest and requires the real SDK test to run and pass. Make the existing Testpilot umbrella depend on that target without dropping any legacy checks. The legacy Case suite currently expects manifests at a pre-archive location; record its before/after baseline separately. New slice failures and introduced legacy regressions block completion. An unchanged pre-existing Case failure is reported explicitly, never called a passing gate or repaired by reviving the frozen Scala build in this slice.

## Acceptance Criteria

- **R1:** The attribute-authored mini activity Model has the six phases and six changing actions above, shared report/states/rows goldens, terminal_remains and the exhaustive closure/negative controls. Its Temporal realization is delivered with the guest. Errors: disabled actions emit no edge; a hypothetical reopening edge fails the preservation predicate. The restricted archive interpretation is explicit.
- **R2:** The renamed Model interface supports Reset, Judge and Answer over Any; a reusable generic guest library, generic Go host, exactly three Temporal envelopes and the Temporal guest agree on generated bytes. The Model judges by replay and answers from its state plus declared callback input. Errors include malformed/unsupported envelopes, policy and status projections, unanswerable callbacks, impossible observations, frozen runs and response-allocation rollback. Existing ABI and old message bytes remain pinned.
- **R3:** A real in-process server and real SDK worker complete both scenarios. Every valid observation is Explained, the worker returns only the Model's Answer, and final phases match. Repeats, callback-before-terminal ordering and reset/isolation are exercised. A mutated impossible observation is Unexplained with expected actions. The dedicated slice gate passes and is invoked by the unchanged-scope Testpilot umbrella; report legacy Case baseline failures separately. Errors name the first failing observation; cancellation and teardown cannot leave a callback blocked. Missing/unselected slice tests and introduced gate regressions fail verification.
- **R5:** The generic subtree contains no Temporal detail and passes the boundary gate in make model-check. Negative controls cover each scanned extension, both spellings, API prefixes, exact file/line reporting and the narrow organization-path allowance. Errors name every offending file and line.
- **R4:** DESIGN records DSL gaps, protocol findings for R33, measured Answer and real-test latency, and the untouched R33-R40 obligations. README and migration guidance identify generic/system ownership and local Cargo/Go references. No error surface beyond the gates above.

## Boundaries

- No Case, Program, Contract, CEL, publish store, replay/fuzz/repeat mode, polyglot shim, Testpilot deletions or schema move.
- No live cancel, retries, heartbeat, timeout or clock scenarios. The abstract Model's cancel/deadline transitions remain. No progress claim, new attributes or input/fact DSL expansion.
- General concurrent read-window inference is deferred. The live test's declared owner ordering supplies its evidence order.
- Starts after fn-210's committed local closure. Owner priority makes this next after fn-214; it creates no dependency on fn-214, fn-213 or fn-215.
- Fn-213 must later include the new Temporal fixture root in its layout migration. Fn-218 retains its eight-case obligation and must resolve impossible pattern alternatives honestly, without adding Model edges or silently reducing Cases(8).
- No extraction, remote publication, push or pull request is part of this slice.

## Decision Context

2026-10-19, owner. Build the smallest real SDK vertical slice before the broader Case redesign. The Model is the oracle. The proto is named umpire.model.v1, its Temporal vocabulary is the upstream API in exactly three envelopes, and nothing system-specific enters Umpire. Testpilot's schema stays untouched.

2026-10-19, planning. Correct the draft's seven-variant actor list, unchanged-phase Start and exact-archive claim with the explicit six-action restriction above. Preserve every requested phase. The original archive also has an intermediate cancellation phase and parameterized retry behavior; this slice does not claim those semantics.

2026-10-19, repository evidence. SDK v1.48.0 exposes ExecuteActivity and the repository already supplies a real in-process SQLite server. The existing TestTestpilot gate can run the slice without a CLI installation or a fake server. The generated descriptor closure currently has 57 files with resolved imports; that observation is not a schema cardinality invariant.

2026-10-19, planning. Keep existing task IDs and split the old oversized protocol/guest task into protocol encoding, reusable core, and Temporal binding. Combine findings, gate hardening and final wiring in one finalization task. Preserve the existing Model-before-protocol execution order, serializing the workspace/lockfile checkpoint before its namespace migration. The pure Model no longer consumes envelopes from the later binding task. Owners may delegate disjoint implementation surfaces internally.

## Quick commands

```bash
make model-check
make model-time
make umpire-check-model-slice
CGO_ENABLED=0 mise exec -- go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice$' -count=1
```

## Early proof point

Task .1 fixes and checks the independent finite product abstraction before a real server can influence it. Task .6 proves the generated API/guest boundary before .3 starts the real SDK run. Resolve a failed abstraction or guest proof before proceeding to its consumer.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The attribute-authored mini activity Model has the six phases and six changing actions above, shared report/states/rows goldens, terminal_remains and the exhaustive closure/negative controls. Its Temporal realization is delivered with the guest. Errors: disabled actions emit no edge; a hypothetical reopening edge fails the preservation predicate. The restricted archive interpretation is explicit. | fn-216-mini-activity-vertical-slice-the-model.1, fn-216-mini-activity-vertical-slice-the-model.6 | — |
| R2 | The renamed Model interface supports Reset, Judge and Answer over Any; a reusable generic guest library, generic Go host, exactly three Temporal envelopes and the Temporal guest agree on generated bytes. The Model judges by replay and answers from its state plus declared callback input. Errors include malformed/unsupported envelopes, policy and status projections, unanswerable callbacks, impossible observations, frozen runs and response-allocation rollback. Existing ABI and old message bytes remain pinned. | fn-216-mini-activity-vertical-slice-the-model.2, fn-216-mini-activity-vertical-slice-the-model.5, fn-216-mini-activity-vertical-slice-the-model.6 | — |
| R3 | A real in-process server and real SDK worker complete both scenarios. Every valid observation is Explained, the worker returns only the Model's Answer, and final phases match. Repeats, callback-before-terminal ordering and reset/isolation are exercised. A mutated impossible observation is Unexplained with expected actions. The dedicated slice gate passes and is invoked by the unchanged-scope Testpilot umbrella; report legacy Case baseline failures separately. Errors name the first failing observation; cancellation and teardown cannot leave a callback blocked. Missing/unselected slice tests and introduced gate regressions fail verification. | fn-216-mini-activity-vertical-slice-the-model.3 | — |
| R5 | The generic subtree contains no Temporal detail and passes the boundary gate in make model-check. Negative controls cover each scanned extension, both spellings, API prefixes, exact file/line reporting and the narrow organization-path allowance. Errors name every offending file and line. | fn-216-mini-activity-vertical-slice-the-model.4 | — |
| R4 | DESIGN records DSL gaps, protocol findings for R33, measured Answer and real-test latency, and the untouched R33-R40 obligations. README and migration guidance identify generic/system ownership and local Cargo/Go references. No error surface beyond the gates above. | fn-216-mini-activity-vertical-slice-the-model.4 | — |
