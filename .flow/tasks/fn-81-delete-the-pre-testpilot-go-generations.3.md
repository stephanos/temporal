---
satisfies: [R2, R3]
---
# fn-81-delete-the-pre-testpilot-go-generations.3 Delete the legacy Go trees, modules, tests, and tooling hooks

## Description
Implements the gomad and agentworkflow parts of R2 and R3 (spec §Commit order 4 and 5). Deletes gomad, gomad1, and gomad2 with the root go.mod require and replace and the Gomad v3 parity-manifest retirement in one commit, then agentworkflow, then verifies tidy against the captured closure and the package-set subset. The umpire trees, tests, and genmodels were removed in task .2.

**Size:** S
**Files:** `tools/gomad/`, `tools/gomad1/`, `tools/gomad2/`, `tools/gomad3/simulation/parity/` (delete), `tools/gomad3integration/simulation_contract_test.go`, `tools/gomad3/README.md`, `tools/gomad3/internal/gomadtool/validation/script_policy.go`, `tools/agentworkflow/`, `go.mod`, `go.sum`
**Touches:** [tools/gomad/**, tools/gomad1/**, tools/gomad2/**, tools/gomad3/simulation/parity/**, tools/gomad3integration/simulation_contract_test.go, tools/gomad3/README.md, tools/gomad3/internal/gomadtool/validation/script_policy.go, tools/agentworkflow/**, go.mod, go.sum]

### Approach
- Commit 4: `git rm -r tools/gomad tools/gomad1 tools/gomad2 tools/gomad3/simulation/parity`; delete `go.mod:62` require and `go.mod:263` replace for `github.com/temporalio/gomad`; drop the parity assertions from `tools/gomad3integration/simulation_contract_test.go` (keep the `SpecSchema` and `DefaultLimits` checks only if they still have a non-parity source), and the parity references in `tools/gomad3/README.md` and `script_policy.go`; `go vet -tags test_dep,gomad3_integration ./tools/gomad3integration` and `cd tools/gomad3 && go vet ./simulation/...` must pass; `go list ./...` must load before tidy. Then `go mod tidy`, `git diff go.mod`, and for every dropped module `grep -c <module> .flow/tmp/fn81/deps-before.txt` must be 0. Record the table in the ledger. Research predicts zero third-party removals beyond the gomad module itself.
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
- `tools/gomad`, `gomad2`, `agentworkflow` are separate modules and invisible to the root build; their deletion is evidenced by the ledger, not by the build.
- `tools/gomad3`, `tools/gomad3sim`, `tools/gomad3integration`, and `tests/gomadfunctional` are retained per `.plans/GOMAD_MILESTONES.md` F0. The parity manifest is the only gomad3 code that names gomad2, so it is the only gomad3 edit.

## Acceptance
- [ ] Two commits in the stated order, each passing `go build -tags 'test_dep integration' ./...`
- [ ] `go.mod` has no `github.com/temporalio/gomad` require or replace; `go mod tidy` is a no-op afterwards
- [ ] Tidy table in the ledger: every dropped module has zero hits in `deps-before.txt`
- [ ] `comm -13 pkgs-before.txt pkgs-after.txt` is empty
- [ ] `go vet -tags test_dep ./...` passes; `CGO_ENABLED=0 go test -tags test_dep ./tools/... ./common/testing/testpilot/... ./tests/testcore/...` passes
- [ ] `git grep -n -E 'tools/(umpire[123]|gomad[12]?([^0-9a-z]|$)|agentworkflow)|cmd/umpire-genmodels|temporalio/gomad' -- '*.go' '*.toml' '*.sh'` returns nothing
- [ ] `go vet -tags test_dep,gomad3_integration ./tools/gomad3integration` passes and `tools/gomad3/simulation/parity` no longer exists

## Done summary
Deleted `tools/gomad`, `tools/gomad1`, and `tools/gomad2` with the root `go.mod` require and replace
for `github.com/temporalio/gomad`, then `tools/agentworkflow`, in two commits each green under the
tagged build.

The Gomad v3 parity manifest went with them. Every `sources[].path` in
`tools/gomad3/simulation/parity/manifest.json` pointed into `tools/gomad2/`, and `manifest.go:315`
enforced that prefix, so the contract could not outlive the tree it cited. The package is deleted,
`script_policy.go` loses its `parity.Current()` check, `tools/gomad3integration/simulation_contract_test.go`
is deleted because its every assertion derived from that manifest, and the gomad3 README and glossary
now record the manifest and the Parity Case term as historical rather than current.

Tidy dropped twelve modules. The task's research predicted zero third-party removals beyond the gomad
module itself, which was wrong: four of the twelve carried gomad1's transformer and runtime, and
gomad1 compiled inside the root module. The ledger now carries the tidy table naming each dropped
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
- Tests: go build -tags 'test_dep integration' ./... (rc=0, after both commits), go vet -tags test_dep ./... (rc=1, still exactly the 15 inherited diagnostics), go vet -tags test_dep,gomad3_integration ./tools/gomad3integration (rc=0), cd tools/gomad3 && GOWORK=off go vet ./internal/gomadtool/validation/ (rc=0), go mod tidy then a second go mod tidy: no-op on go.mod and go.sum, twelve tidy-dropped modules, each 0 hits in .flow/tmp/fn81/deps-after.txt (retained closure, 2569 pkgs), comm -13 pkgs-before.txt pkgs-after.txt empty (546 -> 422), CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/... ./common/testing/testpilot/... ./tests/testcore/... (34 packages ok; 2 inherited-red packages: tools/planindex on the documented pre-existing .plans registration drift, tools/tests on an absent local Cassandra), git grep -n -E 'tools/(umpire[123]|gomad[12]?([^0-9a-z]|$)|agentworkflow)|cmd/umpire-genmodels|temporalio/gomad' -- '*.go' '*.toml' '*.sh' returns nothing
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
