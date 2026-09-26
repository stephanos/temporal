---
satisfies: [R1]
---
# fn-81-delete-the-pre-testpilot-go-generations.1 Write the consumer-evidence ledger and capture baselines

## Description
Implements R1 (spec §Evidence rule). Extends `tools/umpire/CLEANUP_INVENTORY.md` with an fn-81 section in the fn-66 format: one row per deletion-set root and per retained neighbor, consumer evidence per search surface, and disposition. Captures the dependency-closure and package-set baselines the later tasks compare against. No deletion happens here.

**Size:** S
**Files:** `tools/umpire/CLEANUP_INVENTORY.md`, baseline captures under `.flow/tmp/fn81/` (untracked)
**Touches:** [tools/umpire/CLEANUP_INVENTORY.md]

### Approach
- Mirror the existing ledger shape (`CLEANUP_INVENTORY.md:1-60`): preamble with the evidence rule, `Frozen baseline` bullets (task-start commit, `git hash-object` of the prior ledger), a three-column `Package | Decision | Evidence` table, then `Authorized deletion paths` with counts.
- Rows (all consumers already found by research; verify each with the stated command):
  - `tools/umpire1`: consumer `service/history/workflow/cache/cache.go:32` (production) and `tests/testcore/monitor/monitor_test.go:5` — disposition: seam removal in task .2 precedes deletion.
  - `tools/umpire2`: consumers `tests/testcore/functional_test_base.go:51,408`, `tests/testcore/test_env_test.go:13,97`, `tests/testcore/monitor/monitor_test.go:6`, `cmd/umpire-genmodels`, `.github/CODEOWNERS:99-102`, `.gitignore:16-18,46-49`, `umpire-model-verification.yml:40` (sources `tools/umpire2/testdata/genmodels/tools.env`).
  - `tools/umpire3`: `Makefile:85-119,147-178,612-999,1282-1392`, `umpire3.yml`, `.gitignore:27-30,68-69`, `tools/umpire/vocabulary/retired_vocabulary_test.go:91` fixture path, `tools/umpire3/model` Lake project (193 tracked files; `.lake` untracked, 677 MB).
  - `tools/agentworkflow`: own go.mod; `Makefile:290-301,308`; docs links only.
  - `cmd/umpire-genmodels`: `Makefile:84,594-610`, `mise.toml:6-13`, `develop/umpire/install-tools.sh`, `umpire-model-verification.yml:105-116`.
  - `common/testing/umpire` (96 files): consumers `service/history/workflow/cache/cache.go:27` (the only importer under `service`, verified by `git grep -l 'umpireotel\|common/testing/umpire' -- service`), `tests/testcore/monitor/monitor.go:10`, `tests/probe`, `cmd/umpire-genmodels`; `.github/CODEOWNERS:98`. Six other history-service files carry observer comments only (`git grep -n -i 'umpire observer\|.plans/UMPIRE.md' -- service`); list them as comment-only rows.
  - `tests/testcore/monitor`, legacy `tests/umpire[23]_*.go`, `tests/probe`.
  - Retained neighbors with reasons: `tools/fairsim` + `cmd/tools/fairsim` (upstream PR #8158; `Makefile:561,573-575`, `.gitignore:55` stay), `tools/planindex` + `.plans/index.json` (fn-66 carve-out revalidated), `.flow/tmp/fn20.4-base-*` duplicate tree (record tracked or untracked via `git ls-files .flow/tmp | head`).
- Baselines: `go list -deps -tags 'test_dep integration' ./... | sort > .flow/tmp/fn81/deps-before.txt` and `go list ./... | sort > .flow/tmp/fn81/pkgs-before.txt`, plus `du -sh tools/umpire3 model/.lake` and `git ls-files | wc -l`.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/CLEANUP_INVENTORY.md:1-60` — ledger format to mirror
- `Makefile:84-178,290-352,594-610,1139-1172,1282-1392` — legacy variable and target blocks
- `service/history/workflow/cache/cache.go:27-32,342-432` — production seam consumer

**Optional** (reference as needed):
- `.flow/specs/fn-66-remove-unused-umpire-tooling-after.md` — prior sweep's scope statements

## Acceptance
- [ ] Ledger section lists every deletion-set root and every retained neighbor named in the spec with consumers per search surface and a disposition; no row reads merely "unused"
- [ ] fairsim and planindex retentions and the fn-66 carve-out reversal are recorded with reasons
- [ ] `.flow/tmp/fn81/deps-before.txt` and `pkgs-before.txt` exist and the ledger cites the commands; `du -sh` and file-count baselines recorded
- [ ] `.flow/tmp` duplicate tree is classified as tracked or untracked
- [ ] Ledger states the nine pinned failure identities and their defining files as the set R4 retires

## Done summary
Extended `tools/umpire/CLEANUP_INVENTORY.md` with an fn-81 section in the fn-66 ledger format:
thirteen deletion-set rows and eight retained-neighbour rows, each with repository-wide consumer
evidence per search surface (Go imports in the root module and each nested module, Makefile,
workflows, shell scripts, mise, Lake, proto, Lean, generated manifests, CODEOWNERS, ignore files,
documentation) and a disposition naming the task that removes it. Captured the dependency-closure,
package-set, tracked-file, disk, build, vet, lint, and plan-index baselines the later tasks compare
against, including two inherited reds (`go vet` 15 diagnostics, `go run ./tools/planindex` 45
findings) so a task-caused failure stays distinguishable from an inherited one.

stage: impl-review - ran (backend claude, model claude-fable-5-1, effort high, 3 rounds: NEEDS_WORK,
NEEDS_WORK, SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3da1a64f00c1466278c92f8b3a1302cef542789d, eb3da17ea2e3bf3bc9630b287107c35936dc6073, 7da7dc2e56aad663bd74dc261e86064281f61f73, b0dd115b78370dc65948becdc799950d7b06b441
- Tests: go build -tags 'test_dep integration' ./... (rc=0), go vet -tags test_dep ./... (rc=1, 15 inherited diagnostics, unchanged from baseline), go run ./tools/umpire/cmd/umpire-check-retired-vocabulary (rc=0), go run ./tools/planindex (rc=1, 45 inherited findings, unchanged from baseline), go list -deps -test -tags 'test_dep integration' ./... -> .flow/tmp/fn81/deps-before.txt (2889), go list ./... -> .flow/tmp/fn81/pkgs-before.txt (546)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
