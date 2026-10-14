---
satisfies: [R1, R4]
---
# fn-180-gomad-replay-time-instrumentation-for.2 Stop-and-inspect: debug goroutine, stop decision and built-in stack/heap dumps

## Description
Create a dormant debug goroutine at initialization in every run (recording and replay identical). At the named stop point the helper answers the scheduler hook with an enter-debug decision, wakes the goroutine, which writes all goroutine stacks (file:line) and a heap profile through the side channel and ends the run. Verify choice-trace and transcript digests up to the stop point and report mismatch as instrumentation divergence, distinct from ordinary replay divergence. Requalify WASM (the runtime/model identity rotates).

## Acceptance
- [ ] Stop at choice index, virtual time and recorded failure on a WASM failure artifact produces stacks with file/line and heap profile
- [ ] Digest verification up to stop point; instrumentation divergence classified separately (test with a perturbing control)
- [ ] Full WASM requalification gate set passes on the integrated candidate

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
