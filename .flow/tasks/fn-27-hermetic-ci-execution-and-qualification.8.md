# fn-27-hermetic-ci-execution-and-qualification.8 Reject drift without new artifact or policy formats

## Description
Add negative fixtures for Artifact byte/version/checksum/fingerprint drift, incomplete closure, generated semantic differences, unauthorized bindings, Limit drift, and cleanup leakage. Reuse current v2 admission and Result values; add no Evaluation Receipt, provenance schema, new artifact-set version, or migration route.

## Acceptance
- [ ] Every named drift class fails at its responsible boundary before a false portability result.
- [ ] The exact v2 Artifact remains the sole semantic input and no new persisted family is created.
- [ ] Fixtures are independent and report stable failure classifications.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
