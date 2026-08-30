---
satisfies: [R5, R6]
---
# fn-44-seal-observation-traces-and-centralize.5 Migrate accepted traces through Run Evaluation and artifacts

## Description
Carry Task 4's accepted type through composed Run Evaluation and artifact projection (R3, R5) and migrate Temporal mutation tests that still forge accepted records.

**Size:** M
**Files:** `model/Umpire/Observation/Check.lean`, `model/Temporal/Tool/RunEvaluation.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`, `model/Temporal/Tool/RunEvaluationMutationTests.lean`
**Touches:** [model/Umpire/Observation/Check.lean, model/Temporal/Tool/RunEvaluation.lean, model/Temporal/Tool/RunEvaluationTests.lean, model/Temporal/Tool/RunEvaluationMutationTests.lean]

### Approach
- Update the composed Observation → Implementation Link → Property flow to pass only admitted trace values and never invoke later stages after Observation non-success.
- Project artifacts only through read-only accepted-trace operations, retaining exact semantic fields, stage statuses, JSON bytes, fingerprints, and checksums.
- Replace Run Evaluation mutation fixtures that record-update accepted traces with unchecked-carrier admission failures at the Observation stage.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Check.lean:170-230` — composed Run Evaluation control flow.
- `model/Temporal/Tool/RunEvaluation.lean:208-225` — artifact trace projection.
- `model/Temporal/Tool/RunEvaluationTests.lean` — accepted pipeline/status coverage.
- `model/Temporal/Tool/RunEvaluationMutationTests.lean:281-307` — forged accepted-trace fixtures.
- `model/Umpire/Artifact/Result.lean:105` — persisted accepted-trace shape that must not change.

### Key context
This task changes only the in-memory handoff. Artifact schemas and exact bytes remain owned by the completed artifact boundary and must not acquire an unchecked carrier.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Tool.RunEvaluationTests TemporalModelTests
make umpire-check-regression
```
## Acceptance
- [ ] Composed Run Evaluation passes only admitted traces and never invokes Implementation Link or Property after Observation non-success.
- [ ] Artifact projection uses read-only accepted-trace operations and preserves every field, stage status, JSON byte, checksum, and fingerprint.
- [ ] Run Evaluation mutation tests exercise malformed unchecked-carrier admission rather than forging accepted values.
- [ ] Focused Run Evaluation, aggregate Temporal model, and regression gates pass with no import or trust-boundary drift.
- [ ] No generated file, artifact schema, runtime behavior, persisted byte, fingerprint, or checksum changes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
