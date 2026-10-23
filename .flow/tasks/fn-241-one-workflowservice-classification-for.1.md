---
satisfies: [R1, R2]
---
# fn-241-one-workflowservice-classification-for.1 Split classify into one service table and per-feature claims

## Description
After fn-228.4 (same Temporal realization files), before fn-228.5. Move the WorkflowService resource-family classification out of model/temporal/feature/activity/api.rs into one service table in model/temporal outside feature/, owning exhaustiveness; the activity feature declares only its eight claimed RPCs; update the catalog syntax/build parsing (system-agnostic) and the services.rs controls; add duplicate-claim and unclassified-RPC negative controls. Goldens byte-identical except listed reason text.

## Acceptance
- [ ] Service table outside feature/ classifies every RPC and owns exhaustiveness
- [ ] Activity feature declares only its eight RPCs; reasons no longer name the activity Model
- [ ] Duplicate claim and unclassified RPC fail the build with named diagnostics
- [ ] Default and required model gates pass; guest refreshed and slice/gaps proof if guest bytes change

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
