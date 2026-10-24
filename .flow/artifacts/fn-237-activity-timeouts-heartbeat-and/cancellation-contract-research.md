# Public cancellation contract evidence for fn237.3

Repeated cancellation has a documented delivery guarantee, but the examined public sources do not specify the response status for a second distinct `RequestCancelActivityExecution` while the first remains pending. The future Model author still needs an independent outcome decision. The retained status 9 observation cannot supply that decision.

Research date 2026-10-24. Research tier gpt-6-astra/medium. This report is the sole write. No source, tracking, Git mutation, build, live probe, or task completion was performed. Research and flow-next-prose skills were applied. Canonical fn-237 spec/task .3 and the retained fn-227 cancellation research were read before new research.

## Source versions and authority

The repository pins `go.temporal.io/api v1.63.6-0.20260909222256-20151aa90480` in `go.mod:76`. Public API comments below are from that exact **api-go** commit, checked both in the module cache and online. `20151aa90480` belongs to api-go, not the separate api proto repository. Generated public declarations carry the API comments; no server implementation was inspected for this report.

Official documentation was read live and pinned to documentation commit `56a8a19b04e7fe642e4f8da03ff997afc42b40bd` (2026-09-15). The pinned pages contain the relevant claims also present on the live pages. This documentation revision is independent of the API dependency; it supplies published behavioral guidance, not proof of a particular deployment's conformance.

## Repeated cancellation and status distinctions

**Direct, high confidence.** The standalone [Request Cancel documentation, lines 45-50](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-operations/request-cancel.mdx#L45-L50) states, “Repeating the request doesn't deliver a second Cancellation.” It says cancellation can be requested once. This fixes delivery multiplicity, without naming the second RPC's return status. The broader [Activity Execution documentation, lines 141-142](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-execution.mdx#L141-L142) makes the same delivery promise.

**Direct, high confidence.** The cancellation request's `request_id` is “Used to de-dupe cancellation requests.” [Pinned request declaration, lines 18393-18408](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L18393-L18408). Replaying an identical request ID and issuing a second distinct request are therefore separate research cases. The comment does not specify duplicate-result replay, deduplication lifetime, or the outcome for different IDs.

**Direct, high confidence; applicability remains conditional.** [Activity Operations, lines 68-75](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-operations/index.mdx#L68-L75) assigns `FailedPrecondition` to an existing Activity whose state rejects an operation, and `NotFound` to a missing Activity or run. The page does not enumerate whether pending cancellation accepts a second cancel. Its precedence rule concerns Cancel, Reset and Pause, not an explicit same-operation repetition table.

**Unknown.** Combining single delivery with the conditional status rule does not logically determine that repeated pending cancel must succeed or must reject. Both of these alternatives need an independent Model/design decision, rather than a choice in this report:

- Idempotent operation semantics could return success while leaving the original pending request in place and delivering nothing additional.
- A distinct operation against an already-requested state could reject while preserving that pending request. If the published invalid-state rule applies, `FailedPrecondition` is the documented category.

These alternatives are inference, not promises made by the cited pages. Request-ID replay may have a separate outcome from either distinct-request alternative. Nothing reviewed authorizes treating an existing pending activity as missing or translating status 9 into `NotFound`. A Model disagreement must remain visible.

## Pending, worker notification and closure

**Direct, high confidence.** The [pinned cancel RPC comment, lines 799-805](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/service_grpc.pb.go#L799-L805) defines cooperative cancellation, worker detection and acknowledgement, with notification through the heartbeat response. The [heartbeat response declaration, lines 2826-2830](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L2826-L2830) requires the SDK to notify the running activity when `cancel_requested` is true. Receiving that bit alone is notification, not worker acknowledgement. A successful client Cancel RPC acknowledges acceptance of the request. The separate worker cancellation acknowledgement is `RespondActivityTaskCanceled`; these are different protocol events.

**Direct, high confidence.** [Request Cancel, lines 25-34 and 46-47](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-operations/request-cancel.mdx#L25-L34) distinguishes a running attempt from no running attempt. The latter closes immediately; the former receives cancellation through heartbeat and can accept it by propagating the cancellation error. Completion or non-retryable failure can still win after request acceptance. This independently supports the approved completion/failure race policy for these outcomes.

**Scope issue, high confidence in the documentation; unresolved Model implication.** That same page says a non-heartbeating attempt can reach cancellation at its next attempt boundary, giving start-to-close timeout as an example. Therefore a universal claim that every Canceled transition requires an explicit worker acknowledgement would exceed the public guidance. The approved running-attempt heartbeat/acknowledgement distinction remains sound. The explicit documented race is completion or non-retryable failure. For a worker `RespondActivityTaskFailed` with a retryable failure after request acceptance, the attempt-boundary wording suggests cancellation can intervene before another attempt, but that is an inference. It does not explicitly determine the result for exhausted retries or a complete simultaneous-timeout priority matrix. These cases need their own precise boundary policy. If fn237.3 intends an acknowledgement-only law beyond a live running attempt, its author must independently bound that abstraction or seek an amendment; this report changes none of the approved semantics.

## Terminal and stale attempts

**Direct, high confidence, with scope caveat.** [Pinned worker response comments, lines 287-331](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/service_grpc.pb.go#L287-L331) document `NotFound` for invalid task tokens after timeout, prior completion or nonexistent tasks across Completed, Failed and Canceled responses. Some surrounding event text is workflow-specific and the Canceled comment has an apparent copied method-name typo. These comments are evidence for token invalidation, not a full standalone terminal/status matrix. The heartbeat comment explicitly discusses `NotFound` after workflow-activity timeout.

**Direct, high confidence.** [Activity Execution, lines 216-225](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-execution.mdx#L216-L225) identifies task tokens with task executions and explains how retries invalidate older tokens. This supports keeping stale-attempt evidence distinct from a current pending cancellation. It does not assign every stale acknowledgement's standalone RPC status or specify whether an unsolicited acknowledgement is accepted.

**Unknown.** The examined cancel RPC comment has no terminal-state error table. The operation documentation distinguishes state rejection from absence but does not enumerate every terminal state or deduplicated replay. Worker response `NotFound` rules cannot be transplanted to client cancellation. No universal terminal-Cancel-to-NotFound contract was found.

## Relation to retained evidence and future decision

The retained `.flow/artifacts/fn-227-parameterized-models-declared-worlds/cancellation-evidence-research.md` already established absence of a Start cancellation-mode option, cooperative pending state, heartbeat notification, explicit cancellation response, and completion/failure races. Its server-source observations remain descriptive implementation evidence. This report adds independent official documentation for repetition, operation error categories and attempt-boundary behavior; it does not repeat that implementation investigation.

The parent supplied the counterexample with successful Cancel during Started followed by `CANCEL_REQUESTED`, enabled completion/failure, and a second distinct Cancel returning status 9. That observation was not rerun. It agrees with one possible interpretation of the public guidance but cannot establish the intended Model outcome. fn237.3 must settle repeated pending cancellation, request-ID scope and the boundary cases independently before comparison. No Model, realization mapping, atomic compatibility World, source task or tracking decision is made here.
