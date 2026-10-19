---
satisfies: [R4]
---
# fn-221-rejecting-outcomes-reject-steps-phased.4 Live cancel scenarios, gates and records

## Description
Add two scenarios to the real SDK slice test: cancel after completion (judged NotFound) and cancel before the worker starts (judged Accepted, then Canceled by Describe), plus a mutated status code judged Unexplained. Run the slice, model and boundary gates. Write the DESIGN entry (first evidence-driven DSL expansion, `reject!`/`Phased` decisions, measured latency) and update the realization README with the cancel projection and the Started restriction.

**Size:** S
**Touches:** [tests/testpilot_model_slice_test.go, model/DESIGN.md, model/temporal/realization/README.md, MILESTONES.md (root only)]

## Acceptance
- [ ] Both live scenarios and the mutation control pass under `make umpire-check-model-slice`.
- [ ] `make model-check` and the boundary gate pass; DESIGN and README updated.
- [ ] No Model change made to fit runtime evidence.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
