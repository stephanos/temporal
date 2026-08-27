# fn-27-hermetic-ci-execution-and-qualification.7 Run the bounded CI portability proof

## Description
Run one exact v2 Nexus Artifact through the disposable CI runner and shared Run Evaluation boundary, then compare its stable semantic meaning with the local result. Cover success, semantic non-success, cancellation, timeout, Limit N/N+1, and cleanup failure without remote or production authority.

## Acceptance
- [ ] The valid CI run proves byte-identical input and equal stable Run Evaluation meaning.
- [ ] Negative outcomes remain inspectable and never become portable-success claims.
- [ ] All resources are closed and the proof is deterministic and retry-safe.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
