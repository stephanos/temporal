---
satisfies: [R1, R3]
---
# fn-2-agentworkflow-configuration-and-cli.2 Propagate effective stage models through engine and backends

## Description
Carry the admitted model mapping through the workflow engine and both provider adapters for R1 and R3. Keep the whole-run CLI model as the highest-precedence backend construction option.

**Size:** M
**Files:** `tools/agentworkflow/backend.go`, `tools/agentworkflow/request.go`, `tools/agentworkflow/engine.go`, `tools/agentworkflow/workflow.go`, corresponding root tests, `tools/agentworkflow/internal/backend/codex/*`, `tools/agentworkflow/internal/backend/claude/*`, `tools/agentworkflow/cmd/agentworkflow/main.go`, `tools/agentworkflow/cmd/agentworkflow/main_test.go`
**Touches:** [tools/agentworkflow/backend.go, tools/agentworkflow/request.go, tools/agentworkflow/engine.go, tools/agentworkflow/workflow.go, tools/agentworkflow/*_test.go, tools/agentworkflow/internal/backend/codex/**, tools/agentworkflow/internal/backend/claude/**, tools/agentworkflow/cmd/agentworkflow/**]

### Approach
- Mirror the normalized recipe model data in admitted workflow stages and copy mappings when normalizing.
- Select a model from backend identity at each logical workflow call, including plan review/revision, review lenses, and repairs.
- Add invocation-scoped model data to the backend boundary. Provider command construction uses the explicit CLI override first, then invocation model, then no argument.
- Replace the engine's `.spec` protection invariant with `.agentworkflow`.

### Investigation targets
**Required** (read before coding):
- `tools/agentworkflow/request.go:45-117` — admitted workflow conversions
- `tools/agentworkflow/workflow.go:187-262` — discover, plan, and implement calls
- `tools/agentworkflow/workflow.go:473-640` — review, revision, and repair calls
- `tools/agentworkflow/workflow.go:788-806` — centralized invocation construction
- `tools/agentworkflow/internal/backend/codex/codex.go:163-182` — Codex model arguments
- `tools/agentworkflow/internal/backend/claude/claude.go:108-133` — Claude model arguments

**Optional** (reference as needed):
- `tools/agentworkflow/internal/backend/codex/codex_test.go:30-49` — command assertions
- `tools/agentworkflow/internal/backend/claude/claude_test.go:31-50` — command assertions

### Acceptance
- [ ] Failing tests first identify every logical stage's expected provider model and CLI precedence.
- [ ] Plan review/revision, parallel reviews, repairs, and resume use admitted logical-stage models.
- [ ] Omitted values emit no `--model`; configured values reach only the selected backend.
- [ ] A conflicting resume CLI override remains rejected through backend identity.
- [ ] Focused engine and provider tests pass with `-tags test_dep`.

## Acceptance
- [ ] R1 engine protection uses `.agentworkflow`.
- [ ] R3 stage model propagation, omission, precedence, and resume are covered.


## Done summary
Propagated deep-copied provider model mappings through admitted workflow checkpoints and every logical agent invocation, including plan review/revision, parallel reviews, repairs, and resume. Codex and Claude now apply whole-run CLI model precedence over stage models while omitting the argument when neither is configured; the engine now protects `.agentworkflow`.

TDD evidence: focused tests first failed on missing admitted/invocation model fields, Codex resume model input, and the old `.spec` invariant, then passed after implementation. The tagged module suite, command build, import formatting, race tests, vet, and task-scoped 13-analyzer repository lint are green; the canonical branch-wide lint baseline remains the previously approved inherited exception.

stage: impl-review - ran [2026-08-24T18:19:25Z..2026-08-24T18:25:54Z] | verdict: SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: dffe26ee2563d0567c029216f2fe411e7beb5716
- Tests: cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go build ./cmd/agentworkflow, cd tools/agentworkflow && GOWORK=off go test -race -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go vet -tags test_dep ./..., make fmt-imports (disposable case-sensitive clone; task files unchanged), GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT_FIX=false make lint-code (disposable case-sensitive clone; all 13 configured linters, 0 issues)
- PRs:
