---
satisfies: [R1, R2, R6]
---
# fn-2-agentworkflow-configuration-and-cli.1 Move the YAML contract to .agentworkflow and add stage models

## Description
Update the centralized project configuration and workflow recipe contracts for R1, R2, and the loader portion of R6. This is the early proof task because every later layer consumes these normalized values.

**Size:** M
**Files:** `tools/agentworkflow/internal/project/profile.go`, `tools/agentworkflow/internal/project/project_test.go`, `tools/agentworkflow/internal/project/discovery.go`, `tools/agentworkflow/internal/recipe/recipe.go`, `tools/agentworkflow/internal/recipe/recipe_test.go`, `tools/agentworkflow/workspace.go`, and focused tests
**Touches:** [tools/agentworkflow/internal/project/**, tools/agentworkflow/internal/recipe/**, tools/agentworkflow/workspace.go, tools/agentworkflow/workspace_test.go]

### Approach
- Change the default path and protected/exclusion rules through the existing config-path and normalization helpers.
- Preserve the existing atomic no-overwrite publisher and path/symlink containment checks.
- Keep `.agentworkflow` readable in isolated snapshots while protecting it from candidate mutation.
- Extend the canonical recipe stage contract with a closed provider model mapping and stage-aware validation.
- Remove the former-format probe and diagnostic without weakening strict YAML field, key, tag, alias, and document validation.

### Investigation targets
**Required** (read before coding):
- `tools/agentworkflow/internal/project/profile.go:105-178` — load, resolve, and protection flow
- `tools/agentworkflow/internal/project/profile.go:323-429` — strict YAML validation
- `tools/agentworkflow/internal/project/profile.go:640-661` — default path containment
- `tools/agentworkflow/internal/recipe/recipe.go:23-119` — canonical stage schema and validation
- `tools/agentworkflow/workspace.go` and `tools/agentworkflow/internal/project/discovery.go` — snapshot exclusions

**Optional** (reference as needed):
- `tools/agentworkflow/internal/project/project_test.go:50-290` — path, symlink, and strict-config coverage

### Acceptance
- [ ] Failing tests first demonstrate the new default path, protection/exclusion rules, readable snapshots, and model schema.
- [ ] Starter creation preserves existing contents and never overwrites `config.yml`.
- [ ] Agent-stage model mappings accept only non-blank `codex`/`claude`; non-agent stages reject the field.
- [ ] Resolved YAML preserves configured model values and omitted values remain omitted.
- [ ] Focused project, recipe, and workspace tests pass with `-tags test_dep`.
## Acceptance
- [ ] R1 config loader/starter behaviors are covered.
- [ ] `.agentworkflow` remains readable in isolated snapshots while protected from candidate mutation.
- [ ] R2 strict stage model contract is covered.
- [ ] The former-format loader branch and diagnostic are absent.
## Done summary
Moved the project-owned YAML contract to `.agentworkflow/config.yml`, preserved atomic no-overwrite and containment behavior, protected the complete tool directory while keeping it readable in workspace snapshots/discovery, and added strict per-agent-stage Codex/Claude model mappings with resolved-YAML preservation. Removed the former-format loader probe and aligned CLI expectations with the normal new-path behavior.

TDD evidence: initial focused RED failed because `recipe.Stage` had no model contract; the next RED exposed the old `.spec` default/protection behavior; review-fix RED proved `.agentworkflow` config/instructions were absent from snapshots and discovery; the full-module RED identified the two remaining stale CLI expectations. Focused project/recipe/workspace/CLI tests and the complete nested-module suite are GREEN.

Verification: command build and import formatting pass. Task-scoped lint on a case-sensitive source copy ran all 13 configured analyzers plus the repository vet tool with 0 issues. The user-approved inherited exception remains: default `make lint-code` compares against a six-month-old `main` and reports 1,811 unrelated findings.

The task commits are non-contiguous because concurrent fn-1/Lean work advanced the shared branch between task commits; evidence lists only the three commits owned by this task.

stage: impl-review - ran [2026-08-24T17:56:24Z..2026-08-24T18:02:31Z] | verdict: SHIP
## Evidence
- Commits: b249b093fc494c38346fdaf7b30b1130aad01d65, 5124db1c6ba50b83958be75cab6b0a550cbc473e, 8b441b7f591e37da6a6450cd02fb50bb424865b4
- Tests: RED: GOWORK=off go test -tags test_dep ./internal/project ./internal/recipe - Stage.Models and Models undefined, RED: GOWORK=off go test -tags test_dep ./internal/project ./internal/recipe - old .spec default/protection behavior, RED: GOWORK=off go test -tags test_dep ./internal/project ./internal/workspace - .agentworkflow absent from snapshots and discovery, RED: cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./... - stale CLI .spec and former-format expectations, cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./internal/project ./internal/recipe ./internal/workspace, cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./cmd/agentworkflow, cd tools/agentworkflow && GOWORK=off go test -tags test_dep ./..., cd tools/agentworkflow && GOWORK=off go build ./cmd/agentworkflow, make fmt-imports, GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT_FIX=false make LOCALBIN=/tmp/fn2-1-lint-tools.aROtrL lint-code (case-sensitive clone; 13 analyzers; 0 issues), INHERITED_RED: make lint-code - 1811 pre-existing findings against stale main baseline, NO_RECEIPT: unittest receipt not warrantable because unrelated Makefile state is dirty
- PRs: