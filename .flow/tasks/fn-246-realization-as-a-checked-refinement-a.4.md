---
satisfies: [R5]
---
# fn-246-realization-as-a-checked-refinement-a.4 Protocol Model of the standalone-activity API and the Refinement fixture consumer; the activity Model declares refines

## Description
After .1 (parallel with .2 where files do not overlap). `model/temporal/shared/protocol/activity`: a `#[machine]` Model whose states are Describe snapshot classes (status, run state, attempt within the declared World, outcome kind, last-failure kind, cancel requested) and whose actions are the WorkflowService RPCs with gRPC outcomes, sharing the feature's declared World; native goldens and TLC agreement on its invariants. In `umpire-testing`: a `Refinement` consumer selected by `expected/refinement.txt` that explores the concrete Model, projects each edge through the declared state and action projections, and reports starts, preserved edges, stutters (projected state unchanged) and gaps (no Model counterpart), failing on a violation; generalizes `feature/activity/compatibility/checkpoint.rs`, which stays as independent evidence. The activity Model declares `#[refines(protocol::activity, state = ..., action = ...)]`; the Runtime applies the same projection online through Reach. Mutation controls: a changed protocol edge, a removed Model edge, a wrong projection. No TLC-side refinement mapping; record it as a later checkpoint.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
