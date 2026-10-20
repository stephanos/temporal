---
satisfies: [R2]
---
# fn-228-runner-and-run-the-one-go-host-loop.2 After .1: Propose on the protocol, Runtime, Realize and Host; activity Realization proposes from the world

## Description
Add METHOD_PROPOSE, ProposeRequest { seed }, ProposeResponse { request, action }, Response.propose and REFUSAL_CODE_QUIESCENT to umpire.model.v1; regenerate Rust and Go. Add Realize::propose returning Proposal { action, request } and the Runtime's pending-proposal state (STATE refusal on a second Propose, cleared by the explaining Judge, frozen on Unexplained). Deterministic for the same state and seed; internal actions and Deadline never proposed. Implement it in the activity Realization: start request filled from the world (fn-227), cancel, and worker responses; wire goldens for the method; negative control that a disabled-action proposal is caught. Host.Propose in Go with parity tests.

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
