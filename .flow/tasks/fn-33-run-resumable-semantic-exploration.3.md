---
satisfies: [R1, R2, R3]
---
# fn-33-run-resumable-semantic-exploration.3 Bind campaign workers to runner and conformance

## Description
Execute leased complete ExperimentSpecs through the existing runner/conformance interfaces and return full admitted Results through the bridge for Lean-owned observation (R1–R3).

**Size:** M
**Files:** `tools/umpire/campaign/**`
**Touches:** [tools/umpire/campaign/**]

### Approach
- Treat runner and conformance as injected fixed interfaces, not plugins.
- Bind every result to lease/spec/environment and preserve cleanup/tooling outcomes.
- Send only full fn-18-admitted Results to `observe`; the Lean bridge validates each complete closure and constructs fn-17's opaque checked admission identity and reproduction digest. The fn-17 observation remains domain-neutral; Go never constructs semantic/evidence fields, coverage, corpus, priority, mutation feedback, or a reduced projection.
- Transmit at most eight Results per observe request, enforce the 8 MiB per-Result and 72 MiB aggregate ceilings, and test item/member/frame N/N+1 independently. Runner and conformance remain read-only injected dependencies.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/runner` — fn-19 public runner after completion
- `tools/umpire/conformance` — fn-20 public conformance API after completion
- `tools/umpire/artifact` — admitted result relationships
## Acceptance
- [ ] Workers cannot execute outside the admitted lease/spec/profile closure.
- [ ] Semantic non-success and tooling failure remain distinct.
- [ ] Cleanup, operational, Observation, Refinement, Property, tooling, and Result outcomes remain separately bound and are never dropped.
- [ ] Observe carries full admitted Results, accepts eight maximum-size members within 72 MiB, and rejects item/member/aggregate N+1 before state credit.
- [ ] R1–R3 adapter/failure matrices pass; crossed lease/spec/result mutations reject; campaign adds no runtime or semantic evaluator.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
