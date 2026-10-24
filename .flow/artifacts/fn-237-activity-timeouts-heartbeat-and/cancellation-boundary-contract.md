# Independent cancellation boundary contract

The candidate is coherent as an independently chosen domain contract. I recommend adopting it with explicit amendments to the cancellation, retry, deadline and fact laws before .3 starts. It specifies the complete offline boundary matrix while retaining the original mandatory live witnesses and distinguishing the live evidence scope. Its pending retryable-failure, exhausted-retry and heartbeat-timeout cells are design choices. Public documentation does not determine those cells.

The conductor selects this boundary-aware planning amendment under the standing autonomous milestone goal. Fresh plan review must pass before implementation. This is a plan decision, not a claim that the public API specifies every cell or that live evidence proves the full matrix. The sole write is `.flow/tmp/fn2373-boundary-policy-proposal.md`. No canonical file, source, Git, lifecycle, probe, test or build mutation ran; no server implementation was inspected. The active F worker retains sole commit ownership. Brainstorming and flow-next-prose govern the alternatives and artifact prose within this read-only dispatch.

## Independent rationale and provenance

The Model should treat a pending cancellation request as preventing another attempt. Worker completion and nonretryable failure can still commit their own terminal result while the current attempt runs. A retryable failure instead ends that attempt and gives the pending cancellation its boundary. This interpretation makes cancellation insensitive to the configured retry cap, including a cap of one. A whole-execution schedule-to-close limit separately bounds the run and wins when it expires. These are the reasons for the proposed policy; a live outcome supplies none of them.

| Rule | Public evidence | Proposed authority and confidence |
| --- | --- | --- |
| Started Cancel records pending; heartbeat reports pending without closing; explicit worker acknowledgement cancels | Cooperative RPC/heartbeat contract and request-cancel guidance | Direct public guidance, high confidence |
| Scheduled Cancel closes without worker acknowledgement | Guidance for cancellation without a running attempt | Direct public guidance, high confidence |
| Complete or nonretryable Fail can win after pending Cancel | Explicit request-cancel guidance | Direct public guidance, high confidence |
| Non-heartbeating attempt can cancel at an attempt boundary, including start-to-close timeout | Explicit request-cancel guidance | Direct public guidance for the existence/example, high confidence; exact simultaneous priorities remain chosen policy |
| Fresh distinct Cancel while pending rejects FailedPrecondition | Single-delivery guidance plus conditional invalid-state error category | Chosen interpretation, medium confidence; no source specifies the repeated RPC status |
| Pending retryable Fail cancels, including exhausted attempts | Attempt-boundary interpretation and prohibition on another attempt | Chosen domain rule, medium confidence; exact API behavior unknown |
| Pending heartbeat-timeout expiry cancels | Heartbeat timeout ends an attempt in the proposed Model | Chosen extension of boundary cancellation, medium confidence; exact API precedence unknown |
| Whole schedule-to-close expiry wins as TimedOut | Global bounded-run interpretation | Chosen priority, medium confidence; exact cancellation race not settled by the reports |
| Same-request-ID replay and unsolicited current-token cancellation acknowledgement | Deduplication comment identifies replay; no complete outcome contract | Unspecified API policies, no claimed success/rejection rule |

Evidence comes from `.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/cancellation-contract-research.md` and the retained `.flow/artifacts/fn-227-parameterized-models-declared-worlds/cancellation-evidence-research.md`. Primary pins are [request-cancel guidance](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-operations/request-cancel.mdx#L25-L50), [operation error categories](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-operations/index.mdx#L68-L75), [cooperative RPC comments](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/service_grpc.pb.go#L799-L805), and [request-ID declaration](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L18393-L18408). No new external research ran.

## Alternatives and recommendation

Two full variants warrant review. The candidate gives pending cancellation priority over every retryable worker failure, including the final permitted attempt. An alternative gives exhausted retryable failure priority as Failed, and cancellation priority only when a retry would otherwise occur. Both preserve completion/nonretryable-failure races; neither exhausted-retry result is directly specified by the cited pages.

I recommend the candidate. It keeps one rule for retryable failures while cancellation is pending and avoids making cancellation's boundary behavior depend on maximum_attempts. The alternative preserves more of the old failure-at-cap wording, but introduces a cancellation exception at exactly the point where no future attempt remains. That is coherent too, with a different declared promise. The root should choose by intended domain behavior before comparison. A later disagreement cannot justify switching variants without a new policy decision.

There is no need for another cancellation World parameter, a sibling Model, a nondeterministic response choice or a new framework capability. Keep `asynchronous_cancel` as the approved atomic/cooperative selection. The new policy lives in the existing activity handlers and their independently lifted laws.

## Complete transition policy

`Client::Cancel` denotes a new distinct logical cancellation intent. It does not represent replay of the same request ID. In the following table, pending means `asynchronous_cancel && phase == Started && cancel_requested`. Every accepted ordinary worker response concerns the current attempt. Terminal transitions clear `cancel_requested` and all armed budgets; attempt remains the attempt that terminated. This selects the task's allowed terminal-budget representation explicitly.

| Before/action | Result | Attempt/budgets | Facts |
| --- | --- | --- | --- |
| Atomic nonterminal Cancel | Canceled immediately | Preserve attempt; clear budgets | Empty |
| Cooperative Scheduled Cancel | Canceled immediately | Preserve attempt; clear budgets | Empty |
| Cooperative Started first Cancel | Started/pending, Accepted | Preserve attempt and every budget | Empty |
| Cooperative pending fresh Cancel | Rejected(FailedPrecondition), exact self-loop | Preserve all state | Empty |
| Started Heartbeat, pending or ordinary | Remain Started; report existing pending bit | Reset heartbeat budget only; preserve start-to-close and schedule-to-close | Empty |
| Pending explicit current-attempt cancellation acknowledgement | Canceled | Preserve attempt; clear budgets | Empty |
| Started Complete, pending or ordinary | Completed | Preserve attempt; clear budgets | Empty |
| Started nonretryable Fail, pending or ordinary | Failed | Preserve attempt; clear budgets | Empty |
| Pending retryable Fail at any valid attempt, including maximum | Canceled | No increment; no Scheduled state; clear budgets | Empty |
| Nonpending retryable Fail below maximum | Scheduled | Increment once; apply approved schedule-to-start rearming; preserve spanning schedule-to-close | Empty |
| Nonpending retryable Fail at maximum | Failed | Preserve attempt; clear budgets | Empty |
| Tick selects schedule-to-close expiry, pending or ordinary | TimedOut | Preserve attempt; clear budgets | Exactly ScheduleToClose |
| Tick selects start-to-close or heartbeat expiry while pending | Canceled | Preserve attempt; clear budgets | Exactly selected expired deadline |
| Tick selects any applicable expiry without pending | TimedOut | Preserve attempt; clear budgets | Exactly selected expired deadline |
| Nonexpiring Tick | Ordinary budget decrement | Decrement all live budgets once; never underflow | Empty |
| Tick with no live budget, including terminal | Disabled | Unchanged | No edge |
| Terminal Cancel | Existing Rejected(NotFound) self-loop | Unchanged | Empty |

Schedule-to-start is armed only in Scheduled; heartbeat and start-to-close are armed only in Started. Schedule-to-close spans both phases and every retry. Pending state exists only in cooperative Started; cancellation cannot leak into Scheduled after a pending retryable failure because that failure now closes directly. Poll clears no pending request because reachable Scheduled states have none.

A pending worker acknowledgement requires current attempt identity and an actual pending request. Heartbeat notification and Cancel RPC success never substitute for it. Invalid/stale acknowledgement tokens remain a separate rejection matrix, using the existing invalid-token evidence; do not infer their status from fresh pending Cancel. A current-token acknowledgement without a pending request is outside the authored acknowledgement action's semantic precondition and needs an explicit binding refusal unless independently specified. This preserves the planned explicit-ack path without inventing an unrelated unsolicited-cancel promise.

## Tick selection, facts and races

Tick first decrements every applicable live budget once, then selects exactly one exhausted deadline using an authored total priority. Selection precedes cancellation classification. I propose `ScheduleToClose > StartToClose > Heartbeat > ScheduleToStart`. ScheduleToClose is first by the global-limit decision. The current approved task specifies an explicit priority but does not pin the remaining order; current source still has separate direct deadline actions. Therefore the stated remaining order is a proposed Model choice, not an existing approved order. ScheduleToStart cannot compete with the two Started-only budgets on reachable states.

After selection, the handler chooses phase. A selected global deadline yields TimedOut. A selected attempt deadline yields Canceled if pending, otherwise TimedOut. The single fact always identifies the selected expired budget, even when the phase is Canceled. Thus simultaneous global/attempt expiry produces TimedOut with ScheduleToClose, and simultaneous Started attempt expiry while pending produces Canceled with StartToClose. An unselected exhausted budget produces no additional fact.

The fact records the trigger, not a claim that the terminal phase must be TimedOut. Explicit acknowledgement, worker failure, completion and client cancellation still carry no expiry fact. This preserves F's zero-or-one fieldless enum surface. The older engineering report's activity-specific assertions that every factful edge must be TimedOut and every cancellation must have empty facts need a canonical supersession note. Its generic carrier, rejection and replay contracts need no extension. Do not edit active F source to encode this activity policy.

Actions interleave. A worker Complete or nonretryable Fail that occurs before the expiring Tick wins its terminal race. A global Tick that occurs first ends the run and invalidates later worker effects. The priority order resolves budgets within one Tick only; it must not impose an unobserved reordering of concurrent RPCs and timeouts. Cancel and first Poll likewise follow the committed order: cancellation from Scheduled closes immediately, whereas Poll first allows a running pending request.

## Required canonical amendments before implementation

These changes revise approved semantics. Describing them as mere clarification would hide the old acknowledgement-only, expiry and retry promises that change.

**fn-237 Edge Cases and task .3 Approach 3.** Replace the unqualified acknowledgement-only rule with:

> In a cooperative World, Cancel during Started records pending cancellation and changes no budget. A new distinct Cancel while pending rejects with FailedPrecondition and changes no state, under the chosen interpretation that another cancellation intent is invalid in that state. Heartbeat reports pending without closing. Explicit current-attempt worker acknowledgement closes as Canceled. Completion and nonretryable failure may win. A pending request also closes as Canceled when the current attempt ends through retryable worker failure, including at maximum_attempts, or selected start-to-close/heartbeat expiry. Cooperative Scheduled Cancel closes immediately. Same-request-ID replay has no specified outcome in this Model.

**fn-237 Edge Cases retry law and .3 retry controls.** Replace the unconditional failure-at-cap statement with:

> A nonretryable worker failure closes as Failed at every attempt, including while cancellation is pending. A retryable worker failure with no cooperative pending cancellation retries exactly when attempt < maximum_attempts and otherwise closes as Failed. A retryable worker failure while cooperative cancellation is pending closes as Canceled at the current attempt, irrespective of retry eligibility. This is an independent boundary-precedence choice and prevents another attempt.

The old retry law remains unchanged over every atomic World and cooperative state without pending cancellation. Keep its existing cap/stale-delivery controls on those states and add the pending branch. An unqualified old statement cannot remain a claimed invariant over the enlarged state space. Preserve the old assertion in its valid scope; author and check the new exception explicitly. Ordinary terminal closure/rejection laws remain universal.

**fn-237 R2 expiry/error text, .3 Approach 2 and acceptance.** Replace “expiry produces TimedOut” with:

> Tick selects one expired deadline by the authored priority, with ScheduleToClose first, and emits exactly that ExpiredDeadline fact. Selected ScheduleToClose expiry yields TimedOut. Selected StartToClose or Heartbeat expiry while cooperative cancellation is pending yields Canceled. Every other applicable selected expiry yields TimedOut. Ordinary and nonexpiring edges emit no expiry fact. The fact identifies the expired budget independently of the terminal phase.

**fn-237 R2 cancellation/progress text.** State the complete rule above and retain the exact concrete Poll and RespondCompleted weak-fairness assumptions. RespondCompleted stays enabled while Started/pending, including in timeoutless Worlds. Boundary cancellation adds closure paths; it does not replace the fair-worker proof or justify fairness over heartbeat alone. Preserve the schedule-to-close total-tick bound, reset/spanning laws and negative controls without clamping or cuts.

**fn-237 R3 and .3 compatibility acceptance.** State:

> The atomic compatibility World selects maximum_attempts=1, asynchronous_cancel=false, one-tick ScheduleToStart and StartToClose, and no ScheduleToClose or Heartbeat timeout. Every cancellation, worker-failure and deadline policy in that World preserves the retained lifecycle. Cooperative pending exceptions cannot affect it. The exact state/action/outcome/fact projection and changed-old-edge failure control remain required after every policy edit.

The 15 frozen original expectation files, their hashes/manifest and all original source history remain exact. The retained relation has one start, six states and eleven edges. New active activity goldens may change only within the authorized .3 consumer. Project atomic Tick facts to the old deadline actions, erase new internal stutters only under stated lawful rules, and retain all four terminal Cancel NotFound loops. Preserve every existing closure law and its native/TLC mutants. New heartbeat/ack actions must not add old visible edges through an overly permissive projection.

**fn-237 R4 and .4.** Preserve the originally required held-started heartbeat/acknowledgement, completion-after-cancel, schedule-to-start-under-prepoll-hold and inherited retry/control proofs, plus fn-243.5's unheld cancellation witnesses. Exact pending-expiry cause proofs require actual typed/correlated evidence. Missing cause is a predeclared field-named Unsupported for that narrow live scope and a documented gap, never inferred from terminal Canceled, configured durations, elapsed time or a desired action. This does not discharge any original mandatory row. The final evidence-scope addendum below supersedes the earlier proposal to require every offline boundary cell live.

## Native, TLC and live review controls

The Model matrix should cover maximum_attempts 1 and 3, each reachable attempt, atomic/cooperative modes, pending/nonpending, retryable/nonretryable failure, every deadline alone, zero/one/max budgets, simultaneous expiry and absent budgets. Pending must imply cooperative Started. Terminal states clear budgets/pending, no accepted pending repetition can reset a budget, and no pending retryable failure may increment attempt. The compatibility World must continue the exact retained comparison independently of cooperative tests.

| Counterfactual mutation | Required discriminating case |
| --- | --- |
| Heartbeat closes cancellation | Pending heartbeat remains Started with pending notification |
| Only explicit acknowledgement can cancel | Pending retryable failure and pending attempt-timeout traces reach Canceled |
| Retryable failure at the cap wins as Failed | Pending attempt==maximum closes Canceled; nonpending cap still Failed |
| Pending retryable failure schedules another attempt | No intermediate Scheduled or attempt increment after that failure |
| Nonretryable failure loses to pending cancellation | Pending nonretryable failure reaches Failed |
| Complete loses to pending cancellation | Pending Complete reaches Completed |
| Attempt expiry ignores pending | Matching pending/nonpending expiry traces distinguish Canceled/TimedOut |
| Pending cancellation overrides global expiry | Simultaneous global/attempt expiry reaches TimedOut with ScheduleToClose |
| Wrong/missing/duplicate expiry fact, or facts on ordinary failure | Typed TLC decode/native replay fails for the exact action/state edge |
| Repeated Cancel succeeds, becomes NotFound, or resets a budget | Fresh pending Cancel returns FailedPrecondition and exact state self-loop |
| Cancellation remains pending into Scheduled | Pending retryable failure closes; invariant and cap matrix refute leakage |
| Compatibility cancellation becomes pending or old deadline cancels | Frozen exact projection fails |
| Reopening, accepted terminal step, wrong terminal rejection | Existing native and TLC closure mutants still refute |
| Lost exact-action fairness, reset spanning global budget, skipped expiry | Existing starvation/progress and total-tick-bound controls still refute |

Required TLC must lift the authored source independently, check the same complete matrix and replay counterexamples across pending retryable failure and both Canceled expiry facts. Native/TLC agreement on the unmutated graph alone cannot show these precedence claims discriminate mutants. Retain source-only priority/fact mutations and exact-action fairness descriptors. These are future requirements; no verification ran here.

The live binding must correlate current task tokens/attempts, actual Cancel request IDs, notification, acknowledgement and terminal evidence. Same-ID replay has no outcome promise; distinguish it before outcome comparison and refuse its unsupported scope. Missing/unsupported policy revision names asynchronous_cancel. Out-of-domain timeouts retain field-named Unsupported and transactional freeze. Once this boundary policy is approved and admitted, a different deployment outcome for one of its selected cells is a Model/SUT disagreement, not a reason to reclassify that cell as unsupported. The supplied repeated-Cancel status 9 is not a reason for this proposal, and status 9 must never become NotFound through normalization.

## Proposed R4 evidence-scope addendum

This addendum supersedes only this proposal's expanded R4 live-proof recommendations and their implication that every selected boundary cell must be live-proven in .4. It is proposed, not approved. The complete independent .3 Model policy, exact expiry facts, native/TLC matrix and counterfactual controls above remain unchanged. A Model can cover more behavior than the binding can honestly admit or prove.

The new `.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/cancellation-boundary-evidence-shape.md` establishes that public Failure/TimeoutFailureInfo can represent a timeout kind, but the examined API does not guarantee retaining that cause when status becomes Canceled. Describe's last_failure is conditional and may concern an earlier attempt; standalone activities have no Workflow Event History. Typed field existence alone therefore does not establish terminal causality. Sources are the inventory's pinned [Describe contract](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L16889-L17016), [ActivityExecutionInfo](https://github.com/temporalio/api-go/blob/20151aa90480/activity/v1/message.pb.go#L258-L379), and [Failure carriers](https://github.com/temporalio/api-go/blob/20151aa90480/failure/v1/message.pb.go#L665-L702).

My earlier recommendation added mandatory live pending start-to-close/heartbeat expiry, global-dominance and retryable-failure-at-cap proofs. Those extra cells were my proposed expansion; original R4 did not explicitly enumerate them. Preserve original R4's required held-attempt Cancel/pending-heartbeat/explicit-ack path, completion-after-cancel race, schedule-to-start expiry under a hold, actual timeout admission, transactional controls, existing retry proof and Complete/Fail/HostRefusal/context-Cancellation controls. Also preserve current .4 acceptance's actual cancellation-request acknowledgement, pending state, completion/failure races and repeated-pending evidence. Preserve fn-243.5's separate deterministic raw/SDK unheld admission proofs, unsupported-before-effect refusals, original atomic counterexample/replay and closure dependencies.

Add a live pending retryable-Fail boundary proof when actual evidence establishes it. Retain the exact polled run/attempt/token, submitted failure and applicable retry policy, transport result and validation-failure list, then correlate terminal Canceled with that accepted operation. Account for competing timeout or acknowledgement paths. An RPC success plus later Canceled establishes a sequence; it does not alone prove the failure caused closure. If attribution remains ambiguous, record that evidence gap instead of declaring this added boundary cell proved.

Prove a live attempt-expiry Canceled cell and its exact fact only when actual typed cause and fully correlated evidence identify the selected deadline for the terminating run/attempt. A stale last_failure, desired action, configured duration, elapsed time, transition count or terminal Canceled alone is insufficient. If the evidence cannot identify the trigger, use a predeclared narrow Unsupported scope naming `start_to_close` or `heartbeat_timeout` and the missing expiry-cause evidence. Record the live coverage gap and supported binding scope explicitly. Do not synthesize ExpiredDeadline from configuration or silently treat an acknowledgement as an expiry. No new evidence field or server-internal history is authorized by this recommendation.

Missing cause evidence differs from contradictory evidence. An in-scope, fully evidenced transition that contradicts the chosen Model remains a finding. Unsupported cannot be selected after observing a wrong terminal result. Live receipts and final documentation must distinguish the full offline policy from the admitted observation/operation scope and list boundary cells without live proof.

Fresh plan review must resolve whether original R4 and fn-243.5's unheld requirements are satisfied by fully evidenced explicit acknowledgement, completion/nonretryable-failure races and repeated pending classification while pending expiry remains unwitnessed. My proposed reading is that they can be, provided each named mandatory row and unheld proof passes and the narrower evidence scope is explicit. This is not an acceptance verdict. If review establishes an original requirement demands additional pending-boundary live proof, that obligation remains open; a refusal or this addendum cannot discharge it, authorize parent closure or remove it silently.
