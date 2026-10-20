# Activity catalog task: implementation review, round 1

Reviewed root `bf74e3f15279b3c3312fd451f8ccdc1bed409f33` against `1b06b2ff66a5926f731e29fe85661e04caed4a52`.
Three concurrent Codex gpt-6.1-sol high draws; writer and reviewers are the same family.

## correctness draw

I’ll read the task specification first, then trace the changed code and its callers for correctness and edge cases.
The interceptor validates actual request bytes and successful responses, and retains background failures for the final assertion. I’m now checking the Rust constraints against the server validators, especially duration normalization, UUID parsing, and nested failures.
The server checks support the catalog’s handling of non-UUID request IDs, optional run IDs, and timeout normalization. I’m checking the new oneof projection behavior and generic string generator next, along with the focused test evidence.
No surviving correctness findings.

Verification: `git diff --check` passed, and 23 focused tests passed using existing cached binaries. Reviewed recorded live-slice and gate results; no fresh build or live gate was run in this read-only session.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | Exhaustive message, enum, service and action declarations retain build-time checks. |
| R2 | met | Typed projections and byte constraints retain admission checks; supplemental expectations have compile-fail controls. |
| R3 | partial | Request constraints carry expectations and qualified validator provenance. Live probes and gap goldens belong to task .4. |
| R4 | deferred | Reverse coverage explicitly belongs to task .4. |
| R5 | met | Activity catalogs cover actual SDK transport, nested consumed fields, request filling and response validation. |
| R6 | met | Generic constraint changes remain system-independent; Temporal dispatch stays in Temporal crates. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge; no blocking introduced findings.

## contracts draw

I’ll read the task specification first, then review the diff for contract and consistency issues within the changed paths.
The transport routes requests through Rust and retains annotation failures from background calls. I’m checking the catalog’s expectations against the server validators next, including shared request/response types, UUID handling, and payload limits.
The catalog handles several specified exceptions: non-UUID request IDs, timeout normalization, retry-disable behavior, and responses returned with RPC errors. I’m now checking nested response validation and confirming that each declared rejection matches the named validator’s code and message.
The 8 cached native catalog tests and 3 annotation-route tests passed. The recorded live run also shows all seven required RPCs passing request and response validation.

One possible contract gap remains: the catalog marks some subtrees as ignored even though the SDK reads them. I’m tracing those consumers to distinguish an allowed scope limitation from a violation of the task’s coverage requirement.
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/temporal/features/activity/api.rs:386`
- **R-IDs**: `[R5]`
- **Problem**: SDK-consumed failure fields are hidden behind opaque declarations. For example, `retry_state: Bytes + Ignored(...) opaque` disables validation of `ActivityFailureInfo.retry_state`. A successful `PollActivityExecution` response containing this failure alternative with `retry_state = 999` passes annotation validation, then reaches the SDK’s `DefaultFailureConverter.FailureToError`, which reads that discriminant. The same omission exists for `ChildWorkflowExecutionFailureInfo.retry_state`. This contradicts the task’s explicit requirement to annotate fields consumed by the SDK.
- **Suggestion**: Declare these fields using the existing `Enum(retry_state)` catalog, review the remaining opaque failure fields against their SDK consumers, and add a response control proving an invalid nested discriminant produces a finding while preserving response bytes.

Verification: 14 focused tests passed using existing cached binaries; `git diff --check` passed. No fresh build or full suite was run.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Exhaustive message, enum, service and action declarations; projected-oneof controls. |
| R2 | met | Typed byte constraints and supplemental compile-failure controls. |
| R3 | partial | Request expectations and admission implemented; live probes and golden assigned to task .4. |
| R4 | deferred | Reverse coverage explicitly assigned to task .4. |
| R5 | partial | Actual transport validation implemented; SDK-consumed failure fields remain opaque as described above. |
| R6 | met | Generic annotation changes introduce no Temporal dependency. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict> - Annotate the SDK-consumed failure fields currently excluded from validation.

## integration draw

I’ll read the task specification first, then trace the diff’s integration points with existing callers and shared types.
The diff adds an interceptor that routes actual SDK messages through the Rust catalog, including responses and cancellation traffic. I’m checking the catalog projections against the server validators and tracing how annotation failures survive retries and shutdown.
A fresh focused test run is blocked by the read-only filesystem. I’m continuing with source tracing and the recorded test evidence; the diff preserves the existing protocol numbers and handles annotations independently of frozen Model state.
No surviving findings on the integration-with-unchanged-code axis.

Verification: 11 existing compiled catalog/route tests passed. Fresh compilation was blocked by the read-only filesystem; live coverage was checked against recorded evidence.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Descriptor checking and exhaustive field, enum, service and action declarations remain wired. |
| R2 | met | Byte constraints and projected oneofs retain typed admission and focused controls. |
| R3 | deferred | Request expectations and compile-time controls implemented; live probes/golden explicitly belong to task .4. |
| R4 | deferred | Reverse coverage explicitly belongs to task .4. |
| R5 | met | Activity catalogs, SDK transport interception, nested validation and ID filling implemented. No Visibility tag declared. |
| R6 | met | Generic constraint changes remain system-independent; Temporal bindings stay in Temporal modules. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>

## Conductor merge and disposition

Kept contracts:1 without alteration; no duplicate or suppressed findings. Correctness and integration each have zero findings. Finalized verdict NEEDS_WORK, one introduced P2. Worker is fixing SDK-consumed failure fields and adding nested response refusal/non-repair regression evidence. Task remains in progress; this is not a completion receipt.

