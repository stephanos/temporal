---
satisfies: [R1, R2]
---
# fn-246-realization-as-a-checked-refinement-a.1 Reach, fillers and capability traits in umpire-runtime; Kettle adopts Reach

## Description
In `model/umpire/protocol/runtime`: add `Projection::Reach(M::State)` with a bounded search through the Model's transition graph from the current state, admitting an edge when the action's evidence predicate accepts the observation (or the refinement declares a stutter); unique path judges, several give canonical candidates, none is Unexplained; depth-bounded, never looping. Add `Fill<A>`, `Identity`, `Snapshot`, `Evidence<K>` and `ReplyPolicy` with defaults and assemble `Realize` from them, keeping the Runtime's contract and `umpire.model.v1` unchanged. Kettle (`model/umpire/reference/kettle/realization.rs`) adopts Reach and the traits as the system-agnostic control; its goldens change only where listed. No Temporal detail. Starts after fn-243.4 closes (cross-spec prerequisite through this spec's dependency on fn-243).

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
