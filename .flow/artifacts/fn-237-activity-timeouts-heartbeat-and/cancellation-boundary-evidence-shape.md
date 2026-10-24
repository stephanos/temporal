# Public evidence shape for cancellation at an attempt boundary

The pinned API can represent timeout causes, but the examined public contract does **not guarantee a timeout cause is retained when the execution closes Canceled**. A Canceled terminal status alone cannot distinguish start-to-close expiry, heartbeat expiry, retryable worker failure or explicit cancellation acknowledgement. The proposal's exact expiry facts therefore have an unresolved public-evidence requirement for future .4.

This is a schema and documentation inventory, not a policy decision. The unapproved `.flow/tmp/fn2373-boundary-policy-proposal.md` was read as the question's context. API pin is `go.temporal.io/api v1.63.6-0.20260909222256-20151aa90480`, already established from `go.mod`. Public generated declarations and comments were read from that module and checked online. Research date 2026-10-24; gpt-6-astra/medium. Only this report was written. No server implementation, live probe, build, source edit, Flow mutation or Git mutation was used.

## Read surfaces

| Surface | Exact public fields | Contract limit |
| --- | --- | --- |
| `DescribeActivityExecution` | Request `include_last_failure`, `include_outcome`; response `run_id`, `info`, `outcome`, `long_poll_token` | Last failure is included only if available. Outcome is conditional on completion and requested inclusion. Long polling may skip intermediate state changes. |
| `PollActivityExecution` | Response `run_id`, `outcome` | Waits for completion and returns result/failure; carries no separate attempt-boundary event. |
| `ActivityExecutionInfo` | `status`, `run_state`, `attempt`, `last_failure`, `canceled_reason`, `close_time`, `last_attempt_complete_time`, `state_transition_count` | `last_failure` describes the last failed attempt, not an explicitly correlated terminal-transition cause. No comment promises population or replacement on cancellation. |
| `ActivityExecutionOutcome` | Result/failure oneof; `retry_state` | The schema can report an unsuccessful outcome and retry disposition; it has no separate cancellation-boundary discriminator. |

Sources: [Describe request/response](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L16889-L17016), [Poll response](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L17153-L17161), [Info](https://github.com/temporalio/api-go/blob/20151aa90480/activity/v1/message.pb.go#L258-L379), [Outcome](https://github.com/temporalio/api-go/blob/20151aa90480/activity/v1/message.pb.go#L33-L46), and [RPC purpose](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/service_grpc.pb.go#L773-L786). Field existence and the stated conditions have high confidence.

The decisive request wording is “if available.” Setting `include_last_failure=true` requests a field; it does not promise that a pending-cancel timeout creates or preserves that failure. The Info comment calls it “Failure details from the last failed attempt.” No attempt number or terminal-transition identifier is embedded beside that field. It could describe prior failure history; treating it as the current cancellation trigger requires evidence beyond the field's existence.

`canceled_reason` identifies a cancellation request reason, not a typed deadline cause. Timestamps, configured durations and transition counts do not declare which competing event closed the execution. Elapsed-clock calculations cannot supply the missing normative attribution.

## Failure carriers

The [Failure schema](https://github.com/temporalio/api-go/blob/20151aa90480/failure/v1/message.pb.go#L665-L702) has recursive `cause` and a failure-info oneof. [TimeoutFailureInfo](https://github.com/temporalio/api-go/blob/20151aa90480/failure/v1/message.pb.go#L107-L114) contains `timeout_type` and `last_heartbeat_details`. [TimeoutType](https://github.com/temporalio/api-go/blob/20151aa90480/enums/v1/workflow.pb.go#L692-L701) distinguishes start-to-close, schedule-to-start, schedule-to-close and heartbeat. [CanceledFailureInfo](https://github.com/temporalio/api-go/blob/20151aa90480/failure/v1/message.pb.go#L160-L167) has payload details and identity, with no timeout-kind field.

Consequently the wire schema can carry a cancellation failure with a nested timeout cause, or a separate timeout in `info.last_failure`. **Unknown:** whether either is populated for pending cancellation at start-to-close or heartbeat expiry. Neither the schema nor the reviewed cancellation guidance specifies that mapping. The available `RETRY_STATE_CANCEL_REQUESTED` [enum value](https://github.com/temporalio/api-go/blob/20151aa90480/enums/v1/workflow.pb.go#L606-L618) also cannot by itself distinguish a timeout boundary from a worker-failure boundary; no reviewed comment guarantees its population for those terminal cases.

This is bounded absence of a promise in the inspected sources, not a claim that deployments never return these values.

## Worker failure acceptance and attempt identity

[PollActivityTaskQueueResponse](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L2482-L2540) provides `task_token`, `activity_id`, standalone `activity_run_id`, namespace and `attempt`. [RespondActivityTaskFailedRequest](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L3346-L3378) submits that token and a `failure`. Its [response](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L3482-L3489) has `failures`, documented as server validation failures. It carries no final status or boundary cause.

[ApplicationFailureInfo](https://github.com/temporalio/api-go/blob/20151aa90480/failure/v1/message.pb.go#L28-L43) exposes `non_retryable` and failure type. A false flag alone does not establish retry eligibility independently of retry policy, failure-type exclusions and limits. The actual request and applicable policy must remain available to interpret the proposed retryable-failure action.

For a future recorded failure RPC, transport success, its validation-failure list and the exact submitted token/failure are distinct evidence. The token can be correlated with the public poll's run and attempt without decoding internal token contents. A subsequent Describe can separately establish terminal Canceled for that run. These observations can establish that sequence; the response does not explicitly assert that this worker failure caused cancellation. Inferring that causality amid a competing timeout or acknowledgement would exceed this inventory. No claim is made here about whether a server accepts or rejects the proposed boundary cell.

## Limits on obtaining exact expiry evidence

The [pinned request-cancel guidance](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-operations/request-cancel.mdx#L25-L34) documents next-attempt-boundary cancellation and gives start-to-close expiry as an example. It does not name a response field retaining that trigger. The official [Standalone Activity page](https://docs.temporal.io/standalone-activity#standalone-activity-versus-workflow-activity), checked 2026-10-24, states standalone activities have no Workflow Event History. A workflow activity timeout event therefore cannot be assumed available for this standalone evidence path.

Current feasibility conclusion, with high confidence in field inventory and bounded confidence in documentation completeness: no reviewed public promise establishes a guaranteed exact STC/HB cause when terminal status is Canceled. Future .4 may encounter an actual typed failure cause, but this report neither predicts one nor treats observed implementation population as a public guarantee. Missing cause evidence cannot be replaced by status alone, desired action, configured timeout, elapsed time or internal server knowledge. The unapproved Model policy and any evidence admission decision remain with their owners.
