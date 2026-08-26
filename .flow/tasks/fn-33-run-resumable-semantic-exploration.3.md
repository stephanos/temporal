---
satisfies: [R1, R2, R3]
---
# fn-33-run-resumable-semantic-exploration.3 Bind campaign workers to runner and conformance

## Description
Execute leased complete ExperimentSpecs through the existing runner/conformance interfaces and return admitted Results to Lean (R1–R3).

**Size:** M
**Files:** `tools/umpire/campaign/**`
**Touches:** [tools/umpire/campaign/**]

### Approach
- Treat runner and conformance as injected fixed interfaces, not plugins.
- Bind every result to lease/spec/environment and preserve cleanup/tooling outcomes.
- Observe only fully admitted results and transmit at most eight result projections per bounded bridge frame; runner and conformance remain read-only injected dependencies.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/runner` — fn-19 public runner after completion
- `tools/umpire/conformance` — fn-20 public conformance API after completion
- `tools/umpire/artifact` — admitted result relationships

### Acceptance
- [ ] Workers cannot execute outside the admitted lease/spec/profile closure.
- [ ] Semantic non-success and tooling failure remain distinct.
- [ ] Cleanup, operational, Observation, Refinement, Property, tooling, and Result outcomes remain separately bound and are never dropped.
## Acceptance
- [ ] R1–R3 adapter/failure matrices pass.
- [ ] Crossed lease/spec/result mutations reject.
- [ ] Campaign adds no runtime or semantic evaluator.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
