---
satisfies: [R1, R2, R7]
---
# fn-14-milestone-a-pilot-baseline-and-lean.1 Freeze the pilot inventory, protocol, and exercise

## Description
### Umpire4 reconciliation (normative)

This task is retained only as historical Milestone A research design. The spec is superseded as an Umpire4 roadmap gate: do not implement it, do not use Agentworkflow evidence or `LEAN_FIRST_GO` for runtime/qualification admission, and do not add it as a dependency. Current Target, Refinement, artifact, runner, conformance, verification, and qualification specs proceed independently.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Create the exact versioned pilot contracts and immutable pre-measurement inputs for R1/R2/R7.

**Size:** M
**Files:** `tools/umpire/pilot/contract.go`, `tools/umpire/pilot/contract_test.go`, `tools/umpire/pilot/testdata/baseline.json`, `tools/umpire/pilot/testdata/thresholds.json`, `tools/umpire/pilot/testdata/fresh-agent-task.md`, `docs/research/umpire-milestone-a-pilot.md`
**Touches:** [tools/umpire/pilot/contract.go, tools/umpire/pilot/contract_test.go, tools/umpire/pilot/testdata/baseline.json, tools/umpire/pilot/testdata/thresholds.json, tools/umpire/pilot/testdata/fresh-agent-task.md, docs/research/umpire-milestone-a-pilot.md]

### Approach
- Define strict baseline, mutation, threshold, exercise, timing, coverage, rubric, and decision-input schemas with canonical identities/digests.
- Freeze the eight named historical defects with unique source/root-cause evidence, exactly twelve mutations across the five named families, the current coverage inventory, and every threshold before any run record exists.
- Freeze the four-file authoring allowlist, one handler-failure exercise, ten-point rubric, infrastructure-only retry rule, sample/warmup policy, and nearest-rank percentile definition.
- Validate source references, chronology, family/link counts, mandatory matrix cells, and that retained results cannot predate their input digests.

### Investigation targets
**Required:**
- `.plans/UMPIRE4_COMPONENTS.md:569-606` — Milestone A evidence and conditional facade decision.
- `.plans/UMPIRE4_COMPONENTS.md:696-714` — required pre-live baseline, metrics, and thresholds.
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:1-245` — shared authoring surface.
- `model/Temporal/Feature/Nexus/Examples/BasicOperations.lean:1-292` — current walkthrough patterns.
- `tools/agentworkflow/internal/agentworkflow/result.go:18-94` — retained Agentworkflow result fields.

### Quick command
`go test -count=1 -tags test_dep ./tools/umpire/pilot -run 'TestContract|TestFrozenInputs'`
## Acceptance
- [ ] Exactly eight distinct source-backed defects and twelve mutations across all five families validate with the required per-family and defect-link counts.
- [ ] Coverage inventory, commands, denominators, timing samples, thresholds, decision precedence, prompt, allowlist, rubric, and retry rules are exact and digest-bound.
- [ ] Duplicate root causes, missing sources, live-only claims, nonsemantic mutations, invalid chronology, and any post-freeze input drift fail closed.
- [ ] The design document distinguishes semantic/core gates from ergonomics gates and claims no runtime conformance.
- [ ] Existing comments in touched files are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
