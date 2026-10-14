---
satisfies: [R1, R4]
---
# fn-180-gomad-replay-time-instrumentation-for.1 Debug side-channel import, replay request options and pre-execution refusals

## Description
Add the dedicated debug host import module (never yields, no clock, outside WASI transcript, choice trace and evidence identity) in the Rust helper and Go host. Extend the WASM replay request and `gomad replay` CLI with stop point (choice=N, time=T, failure), dump selection and probe patterns; refuse unreachable stop points and unknown dump/probe names before execution. Debug output is written next to the replay result with total/per-section byte caps and truncation markers, never into the artifact. Recording never enables instrumentation.

## Acceptance
- [ ] Debug import exists in helper and Go host, excluded from transcript, choice trace and identity
- [ ] Replay CLI/request accepts stop point, dumps and probe patterns; unreachable stop/unknown names refused before execution (tests)
- [ ] Bounded debug output file with truncation markers next to replay result (tests)
- [ ] Recorded runs and replays without options have unchanged evidence

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
