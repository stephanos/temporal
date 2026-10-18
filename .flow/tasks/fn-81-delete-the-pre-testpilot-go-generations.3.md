---
satisfies: [R2, R3]
---
# fn-81-delete-the-pre-testpilot-go-generations.3 Delete the legacy Go trees, modules, tests, and tooling hooks

## Description
Implements the agentworkflow portion of R2 and R3, then verifies tidy against the captured closure
and package-set subset. The Umpire trees, tests, and genmodels were removed in task .2.

**Size:** S
**Files:** `tools/agentworkflow/`, `go.mod`, `go.sum`
**Touches:** [tools/agentworkflow/**, go.mod, go.sum]

### Approach
- Commit 5: `git rm -r tools/agentworkflow`.
- After all commits: `go list ./... | sort > .flow/tmp/fn81/pkgs-after.txt` and `comm -13 pkgs-before.txt pkgs-after.txt` must be empty.
- Makefile targets still reference deleted dirs after this task; that is task .4. Do not run `make umpire-check-regression` here.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/vocabulary/retired_vocabulary_test.go:80-100` — fixture path to repoint
- `mise.toml` and `develop/umpire/install-tools.sh` — genmodels-only hooks

**Optional** (reference as needed):
- `tools/umpire/CLEANUP_INVENTORY.md` fn-81 section (task .1) — authorized paths and baselines

## Acceptance
- [ ] The deletion passes `go build -tags 'test_dep integration' ./...`
- [ ] Tidy table in the ledger: every dropped module has zero hits in `deps-before.txt`
- [ ] `comm -13 pkgs-before.txt pkgs-after.txt` is empty
- [ ] `go vet -tags test_dep ./...` passes; `CGO_ENABLED=0 go test -tags test_dep ./tools/... ./common/testing/testpilot/... ./tests/testcore/...` passes

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: