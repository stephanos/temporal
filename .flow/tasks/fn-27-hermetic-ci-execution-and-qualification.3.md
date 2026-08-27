# fn-27-hermetic-ci-execution-and-qualification.3 Reuse the canonical Run Evaluation authority in CI

## Description
Feed the CI run's admitted Evidence to the same fixed Run Evaluation boundary used locally. Preserve Execution, Observation Evaluation, Implementation Link, Property, cleanup, and tooling outcomes independently. CI code and workflow YAML must not interpret Evidence, translate System facts to Feature facts, or evaluate Properties.

## Acceptance
- [ ] CI uses the shared runner and Run Evaluation API without a CI-specific mapper or evaluator.
- [ ] Equivalent local and CI Evidence produces the same stable meaning and Behavior Fingerprints.
- [ ] Non-success and malformed Evidence remain distinct and fail closed at their owning boundaries.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
