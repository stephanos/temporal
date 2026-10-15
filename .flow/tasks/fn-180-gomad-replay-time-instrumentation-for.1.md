---
satisfies: [R1, R4]
---
# fn-180-gomad-replay-time-instrumentation-for.1 Debug side-channel import, replay request options and pre-execution refusals

## Description
Add the dedicated debug import and replay request plumbing for R1/R4. Resolve the spec's target and stop coordinates from retained metadata before launching debug execution. Extend CLI parsing and debug reports with instance/incarnation, requested/resolved stops and bounded output. Keep the side channel outside recorded choice/WASI evidence and preserve recordings and replay without debug options. Retain enough registration and boundary metadata for preflight; reject older or incomplete artifacts that cannot resolve the request.
## Acceptance
- [ ] Debug import never yields or reads clock and remains outside transcript/choice/evidence identity
- [ ] Single-guest inference and required cluster instance/incarnation selectors follow R1; missing, unknown and ambiguous selectors refused
- [ ] Preflight validates local ordinal/range, recorded time/failure boundary and target-scoped dump/probe names; missing metadata/evidence refused without executing
- [ ] CLI and reports carry target, requested and resolved stop; debug bytes bounded with truncation markers outside the artifact
- [ ] Recordings never enable probes/dumps; replay without debug options preserves evidence
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
