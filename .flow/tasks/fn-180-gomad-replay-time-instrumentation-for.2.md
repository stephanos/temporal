---
satisfies: [R1, R4]
---
# fn-180-gomad-replay-time-instrumentation-for.2 Stop-and-inspect: debug goroutine, stop decision and built-in stack/heap dumps

## Description
Implement the R1/R4 stop-boundary contract using the dormant debug goroutine in every guest incarnation. Validate the retained decision before entering debug mode at a choice stop; resolve time and failure stops to the recorded inspectable boundary. Replay all cluster effects up to that point, park other guests during dumps, then end the replay. Verify executed incarnation evidence prefixes and shared World ordering. Emit selected-guest stacks and heap profile, with honest failure-boundary reporting. Requalify WASM after the runtime/model identity rotation.
## Acceptance
- [ ] Choice, time and failure stops produce selected-guest stacks/heap profile and requested/resolved boundary reports
- [ ] Fixtures distinguish equal local ordinals on two nodes and across a restart, and resolve equal-time boundaries by recorded order
- [ ] Terminated-incarnation replay inspects only an available prior boundary; no-boundary failure refused; validated stop not reached reports divergence
- [ ] Verify all executed cluster evidence prefixes and shared World order; perturb a nonselected node/order to prove it cannot escape verification
- [ ] Other guests stay parked during dumps; replay ends afterward
- [ ] Full WASM requalification gates pass on the integrated candidate
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
