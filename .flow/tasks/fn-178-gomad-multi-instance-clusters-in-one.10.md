# fn-178-gomad-multi-instance-clusters-in-one.10 Parallel versus serialized instances: measure and decide

## Description
Resolve the parked parallelism question with data, now that the harness and the serialized scheduler exist. Should instances run in parallel on host threads, or stay strictly serialized? The answer is a recorded decision. Any parallel implementation would be a follow-up task, added only if the decision says so.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** S
**Files:** the fn-178 spec (Decision Context, via `flowctl spec set-plan`), `.flow/artifacts/` (measurements)
**Touches:** [.flow/specs/fn-178-gomad-multi-instance-clusters-in-one.md]

### Approach
- Measure on the two-cluster Temporal selection with the serialized scheduler:
  - per-instance runnable overlap: how often more than one instance is runnable between World-visible operations
  - the share of authorized execution spent in guest code versus scheduler handoff
- Use these to bound the best-case parallel speedup.
- Assess determinism constraints: World delivery order and clock settling require a single global order at visible operations. Parallelism is possible only between visible operations, and only if per-instance local effects stay instance-private. That holds for namespace and volume, but not for network sends.
- Recommendation: stay serialized unless the bound shows ≥ 1.5× and the scheduler seam from fn-178.1 supports it without changing evidence. Record the decision and, if it is positive, the follow-up scope.

## Acceptance
- [ ] Measurements retained under `.flow/artifacts/`; decision recorded in the fn-178 Decision Context, replacing the serialized working assumption note.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
