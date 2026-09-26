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
Deleted `tools/agentworkflow`; the tagged build remained green. The ledger records each dropped
module, the deleted root that owned it, and its zero count in the retained closure. The acceptance
criterion as written compared against `deps-before.txt`, which was captured over the whole
pre-deletion tree and therefore still contains the deleted roots' dependencies; the substantive R2
check is against `deps-after.txt`, the retained closure, and the ledger states that basis explicitly.

Two test packages are red and both are inherited, verified against the task .1 baselines:
`tools/planindex` on the pre-existing `.plans` registration drift, and `tools/tests` on an absent
local Cassandra. The planindex finding count rose from 45 to 50 because four `.plans` documents now
carry dangling local links into deleted trees; those are R7's work and are listed for task .5.

stage: impl-review - ran (backend claude, model claude-fable-5-1, effort high, 1 round: SHIP with a
P2 on the missing tidy table and a P3 on the glossary term, both fixed in 268fbdb9d)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e5fd1750a420ee323d5059a605041ca36819be86, edf4f86ec1fbc6929bcac49f2bf20d15ce8d19ef, 268fbdb9dda8cc98ffec50b129527bab5fa29ae6
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
