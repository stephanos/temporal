# fn-27-hermetic-ci-execution-and-qualification.5 Compose the hermetic runner and shared Run Evaluation path

## Description
Compose strict v2 admission, the shared disposable runner, admitted Raw Evidence, and canonical Run Evaluation in one bounded test-facing path. Validate all inputs before environment creation, preserve status separation, and expose no Claim Assessment, Evaluation Profile, Evaluation Receipt, provenance Artifact, or new artifact-set version.

## Acceptance
- [ ] One valid bounded run traverses admission, runner, Run Evaluation, and cleanup exactly once.
- [ ] Cancellation, tooling failure, semantic non-success, and cleanup failure retain exact independent outcomes.
- [ ] No profile/policy selector, second publisher, or CI-specific semantic API exists.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
