---
satisfies: [R2]
---
# fn-228-runner-and-run-the-one-go-host-loop.2 After .1: Propose on the protocol, Runtime, Realize and Host; activity Realization proposes from the world

## Description
Add METHOD_PROPOSE, ProposeRequest { seed }, ProposeResponse { request, action }, Response.propose and REFUSAL_CODE_QUIESCENT to umpire.model.v1; regenerate Rust and Go. Add Realize::propose returning Proposal { action, request } and the Runtime's pending-proposal state (STATE refusal on a second Propose, cleared by the explaining Judge, frozen on Unexplained). Deterministic for the same state and seed; internal actions and Deadline never proposed. Implement it in the activity Realization: start request filled from the world (fn-227), cancel, and worker responses; wire goldens for the method; negative control that a disabled-action proposal is caught. Host.Propose in Go with parity tests.

### Required integration contract

Implement the parent bootstrap/admission, pending-state, incremental Judge completion and stateful control-guest contracts. Reserve METHOD_PROPOSE=6 and Response.propose=9; preserve reserved Response tag 7 and Annotate tag 8. Add ProposalKind and JudgeResponse.continue_polling at fresh tag 3. Realize defaults preserve existing bindings. Negative controls cover failed publication, failed/crossed Start, pre-admission callbacks, disabled actions, intermediate polls and second pending proposals. No Start Model action.

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
