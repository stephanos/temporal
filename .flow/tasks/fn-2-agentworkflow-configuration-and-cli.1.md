---
satisfies: [R1, R2, R6]
---
# fn-2-agentworkflow-configuration-and-cli.1 Move the YAML contract to .agentworkflow and add stage models

## Description
Update the centralized project configuration and workflow recipe contracts for R1, R2, and the loader portion of R6. This is the early proof task because every later layer consumes these normalized values.

**Size:** M
**Files:** `tools/agentworkflow/internal/project/profile.go`, `tools/agentworkflow/internal/project/project_test.go`, `tools/agentworkflow/internal/recipe/recipe.go`, `tools/agentworkflow/internal/recipe/recipe_test.go`
**Touches:** [tools/agentworkflow/internal/project/**, tools/agentworkflow/internal/recipe/**]

### Approach
- Change the default path and protected/exclusion rules through the existing config-path and normalization helpers.
- Preserve the existing atomic no-overwrite publisher and path/symlink containment checks.
- Extend the canonical recipe stage contract with a closed provider model mapping and stage-aware validation.
- Remove the former-format probe and diagnostic without weakening strict YAML field, key, tag, alias, and document validation.

### Investigation targets
**Required** (read before coding):
- `tools/agentworkflow/internal/project/profile.go:105-178` — load, resolve, and protection flow
- `tools/agentworkflow/internal/project/profile.go:323-429` — strict YAML validation
- `tools/agentworkflow/internal/project/profile.go:640-661` — default path containment
- `tools/agentworkflow/internal/recipe/recipe.go:23-119` — canonical stage schema and validation

**Optional** (reference as needed):
- `tools/agentworkflow/internal/project/project_test.go:50-290` — path, symlink, and strict-config coverage

### Acceptance
- [ ] Failing tests first demonstrate the new default path, protection/exclusion rules, and model schema.
- [ ] Starter creation preserves existing contents and never overwrites `config.yml`.
- [ ] Agent-stage model mappings accept only non-blank `codex`/`claude`; non-agent stages reject the field.
- [ ] Resolved YAML preserves configured model values and omitted values remain omitted.
- [ ] Focused project and recipe tests pass with `-tags test_dep`.

## Acceptance
- [ ] R1 config loader/starter behaviors are covered.
- [ ] R2 strict stage model contract is covered.
- [ ] The former-format loader branch and diagnostic are absent.


## Done summary
Blocked:
BLOCKED: TOOLING_FAILURE

Both allowed worker attempts stopped before editing because the pre-existing repository-wide
`make lint-code` baseline panics in golangci-lint v2.12.2 / honnef.co tools v0.7.0 `buildir` under
the repository-pinned Go 1.27 toolchain. The focused Agentworkflow tests, build, and formatting are
green. No task implementation edits or commits were made.

Resolution requires upgrading or patching the lint toolchain for Go 1.27 compatibility, then
resetting and retrying this task.
## Evidence
- Commits:
- Tests:
- PRs:
