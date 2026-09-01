---
satisfies: [R3, R4, R8, R9]
---
# fn-20-local-execution-semantic-conformance.8 Prove causal ordering against skewed wall clocks

## Description
Add one independent, test-only skew fixture at the Temporal Lean checker boundary.

**Touches:** `model/Temporal/Tool/RunEvaluationTests.lean`

Use strict TDD: first add the exact executable assertion and capture its RED before adding the private fixture/oracle. The fixture assigns deliberately contradictory wall-clock timestamps to the existing caller-closure history while keeping the admitted facts source-local ordinals and causal parents unchanged. Compare the authoritative causal/source-local projection with an explicitly test-only hypothetical timestamp-authoritative projection. Do not add timestamps to production RawEvidence, alter production ordering, weaken closure, or invent wall-clock authority.

## Acceptance
- [ ] The fixture contains deliberately skewed wall-clock timestamps whose ascending order contradicts both the admitted history source ordinals and causal-parent chain.
- [ ] The authoritative unchanged request evaluates to accepted/applied/satisfied and asserts the exact Evidence Link ordering support plus Result semantic status/checksum.
- [ ] An independent test-only timestamp-authoritative projection over the same fixture changes Observation Evaluation or Property Result, proving the scenario is discriminating rather than decorative.
- [ ] Production request/schema/evaluator code is unchanged unless the RED demonstrates a real defect; no timestamp becomes semantic authority.
- [ ] The focused Lean target and the full fn20/model gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
