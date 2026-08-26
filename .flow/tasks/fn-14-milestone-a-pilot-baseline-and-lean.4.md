---
satisfies: [R2, R4, R5, R7]
---
# fn-14-milestone-a-pilot-baseline-and-lean.4 Build and test the fresh Agentworkflow authoring proxy

## Description
### Umpire4 reconciliation (normative)

This task is retained only as historical Milestone A research design. The spec is superseded as an Umpire4 roadmap gate: do not implement it, do not use Agentworkflow evidence or `LEAN_FIRST_GO` for runtime/qualification admission, and do not add it as a dependency. Current Target, Refinement, artifact, runner, conformance, verification, and qualification specs proceed independently.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Define the fresh-session harness and closed usability evaluator for R2/R4/R5/R7 without running the three retained provider trials or applying candidates.

**Size:** M
**Files:** `.agentworkflow/umpire-milestone-a.yml`, `tools/umpire/pilot/usability/runner.go`, `tools/umpire/pilot/usability/evaluator.go`, `tools/umpire/pilot/usability/runner_test.go`, `tools/umpire/pilot/usability/evaluator_test.go`, `tools/umpire/pilot/usability/testdata/**`
**Touches:** [.agentworkflow/umpire-milestone-a.yml, tools/umpire/pilot/usability/runner.go, tools/umpire/pilot/usability/evaluator.go, tools/umpire/pilot/usability/runner_test.go, tools/umpire/pilot/usability/evaluator_test.go, tools/umpire/pilot/usability/testdata/**]

### Approach

- Build the harness that task `.6` will use for exactly three sequential trials from byte-identical source snapshots, unique external stores/workspaces, fresh sessions, and the same pinned backend/model/config/prompt.
- After each terminal attempt, consume only `agentworkflow evidence-export/v1`; verify event and patch digests before adding pilot classifications.
- Enforce no resume/memory/human messages/manual patching, the exact four-file allowlist, append-only failed attempts, and at most one infrastructure-only retry per trial.
- Score scope, semantic ownership, validation, repair count, elapsed time, patch size, scaffold copying, and the fixed ten-point rubric from exported events/patches plus declared checks.
- Prove all behavior with synthetic canonical exports and a fake backend/store; do not make a live provider call or produce retained pilot trials in this task.

### Investigation targets

**Required:**
- `tools/agentworkflow/internal/cli/cli.go:370-430` — report/diff/export command style.
- `tools/agentworkflow/internal/quality/checks.go:46-106` — direct check evidence and candidate immutability.
- `tools/agentworkflow/internal/agentworkflow/evidence.go` — landed strict evidence export.
- `tools/agentworkflow/README.md:58-74` — configuration versus external evidence-store ownership.
- `model/README.md:68-158` — current authoring orientation and projection boundary.

### Quick command

`go test -count=1 -tags test_dep ./tools/umpire/pilot/usability/...`
## Acceptance

- [ ] Synthetic valid/corrupt evidence exports normalize and score deterministically without any provider trial.
- [ ] The harness admits exactly three trial slots with identical source/config/prompt/model inputs and unique stores/workspaces; a fourth, resume, or hidden shared context is rejected.
- [ ] Every attempt and infrastructure retry is append-only; authoring failure cannot be relabeled infrastructure failure.
- [ ] Exported events prove orientation/list/explain and validation-command use, while canonical patch bytes prove allowlist, churn, duplicate semantics, and scaffold-copy classifications.
- [ ] No candidate patch is applied, no live provider is invoked, and no Agentworkflow engine/store layout is scraped.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
