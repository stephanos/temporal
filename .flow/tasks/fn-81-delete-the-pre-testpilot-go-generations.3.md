---
satisfies: [R2, R3]
---
# fn-81-delete-the-pre-testpilot-go-generations.3 Delete the legacy Go trees, modules, tests, and tooling hooks

## Description
Implements the gomad and agentworkflow parts of R2 and R3 (spec §Commit order 4 and 5). Deletes the gomad family with the root go.mod require and replace in one commit, then agentworkflow, then verifies tidy against the captured closure and the package-set subset. The umpire trees, tests, and genmodels were removed in task .2.

**Size:** S
**Files:** `tools/gomad/`, `tools/gomad1/`, `tools/gomad2/`, `tools/gomad3/`, `tools/gomad3sim/`, `tools/gomad3integration/`, `tools/agentworkflow/`, `go.mod`, `go.sum`
**Touches:** [tools/gomad/**, tools/gomad1/**, tools/gomad2/**, tools/gomad3/**, tools/gomad3sim/**, tools/gomad3integration/**, tools/agentworkflow/**, go.mod, go.sum]

### Approach
- Commit 4: `git rm -r` the six gomad trees; delete `go.mod:62` require and `go.mod:263` replace for `github.com/temporalio/gomad`; `go list ./...` must load before tidy. Then `go mod tidy`, `git diff go.mod`, and for every dropped module `grep -c <module> .flow/tmp/fn81/deps-before.txt` must be 0. Record the table in the ledger. Research predicts zero third-party removals beyond the gomad module itself.
- Commit 5: `git rm -r tools/agentworkflow`.
- After all commits: `go list ./... | sort > .flow/tmp/fn81/pkgs-after.txt` and `comm -13 pkgs-before.txt pkgs-after.txt` must be empty.
- Makefile targets still reference deleted dirs after this task; that is task .4. Do not run `make umpire-check-regression` here.

### Investigation targets
**Required** (read before coding):
- `go.mod:55-70,260-265` — the gomad require and replace
- `tools/umpire/vocabulary/retired_vocabulary_test.go:80-100` — fixture path to repoint
- `mise.toml` and `develop/umpire/install-tools.sh` — genmodels-only hooks

**Optional** (reference as needed):
- `tools/umpire/CLEANUP_INVENTORY.md` fn-81 section (task .1) — authorized paths and baselines

### Key context
- `tools/gomad`, `gomad2`, `gomad3`, `agentworkflow` are separate modules and invisible to the root build; their deletion is evidenced by the ledger, not by the build.
- `tools/gomad3/.toolchain` does not exist on disk; only prune clauses reference it.

## Acceptance
- [ ] Two commits in the stated order, each passing `go build -tags 'test_dep integration' ./...`
- [ ] `go.mod` has no `github.com/temporalio/gomad` require or replace; `go mod tidy` is a no-op afterwards
- [ ] Tidy table in the ledger: every dropped module has zero hits in `deps-before.txt`
- [ ] `comm -13 pkgs-before.txt pkgs-after.txt` is empty
- [ ] `go vet -tags test_dep ./...` passes; `CGO_ENABLED=0 go test -tags test_dep ./tools/... ./common/testing/testpilot/... ./tests/testcore/...` passes
- [ ] `git grep -n -E 'tools/(umpire[123]|gomad|agentworkflow)|cmd/umpire-genmodels' -- '*.go' '*.toml' '*.sh'` returns nothing

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
