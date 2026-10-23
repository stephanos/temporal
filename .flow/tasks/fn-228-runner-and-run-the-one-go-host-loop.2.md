---
satisfies: [R2]
---
# fn-228-runner-and-run-the-one-go-host-loop.2 After .1: Propose on the protocol, Runtime, Realize and Host; activity Realization proposes from the world

## Description
Add METHOD_PROPOSE, ProposeRequest { seed }, ProposeResponse { request, action }, Response.propose and REFUSAL_CODE_QUIESCENT to umpire.model.v1; regenerate Rust and Go. Add Realize::propose returning Proposal { action, request } and the Runtime's pending-proposal state (STATE refusal on a second Propose, cleared by the explaining Judge, frozen on Unexplained). Deterministic for the same state and seed; internal actions and Deadline never proposed. Implement it in the activity Realization: start request filled from the world (fn-227), cancel, and worker responses; wire goldens for the method; negative control that a disabled-action proposal is caught. Host.Propose in Go with parity tests.

### Required integration contract

Implement the parent bootstrap/admission, pending-state, incremental Judge completion and stateful control-guest contracts. Reserve METHOD_PROPOSE=6 and Response.propose=9; preserve reserved Response tag 7 and Annotate tag 8. Add ProposalKind and JudgeResponse.continue_polling at fresh tag 3. Realize defaults preserve existing bindings. Negative controls cover failed publication, failed/crossed Start, pre-admission callbacks, disabled actions, intermediate polls and second pending proposals. No Start Model action.


### Conductor settlement (2026-10-23)

Use additive ResetRequest.configuration Any at tag1 for a typed StartActivityExecutionRequest; absent retains the old zero-byte Reset wire. Configuration explicitly opts into bootstrap behavior. Once opted in, Reset without configuration reuses the selected configuration while clearing admission, receipt, pending, freeze and callback context for a new Run; configured Reset replaces configuration atomically and clears the same run effects. Unconfigured legacy realizations keep existing Reset behavior. Receipt immutability applies within a Run.

Use a default-refusing Realize::matches(request, observation) correlation hook. Generic Runtime keeps payloads opaque; Temporal checks actual invocation and receipt identities, including correlated Describe reads. A typed Temporal Realization context may retain validated callback/task-token data only after successfully published Answer, reusing the existing admission result. Check run, attempt and current Model state before proposal; invalidate stale context on reset, re-admission or corresponding final observation. Do not fabricate tokens or introduce a Go chooser. Preserve immediate Answer reply behavior and ensure proposal support does not require duplicate worker-response RPCs.

All configuration, capability, receipt, context and pending changes share existing prepare/commit publication atomicity. Add failed-publication controls, callback-between-polls controls and explicit Start correlation controls. This implements the approved parent contract without changing the canonical activity Model or its behavioral goldens.

Focused baselines only: the predecessor's default and required full gates are already verified at committed closure ac2fd68e0e. Final task gates still include explicit post-edit default make model-check and timeout3600 UMPIRE_BACKENDS=require make model-time, fresh guest/live/privacy proof, review and paired conductor audits. The global model capacity error interrupted planning before implementation; preserve any unrecoverable baseline observation as inconclusive and recover only after confirming its process terminal.
## Acceptance
- [ ] Propose on proto, Runtime, Realize, Host with wire goldens
- [ ] Activity Realization proposes start/cancel/responses deterministically per seed
- [ ] QUIESCENT and pending STATE refusals pinned; disabled-action control
- [ ] Gate and Go host parity pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
