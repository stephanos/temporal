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
  - gomad family, deleted: nested modules `tools/gomad` and `tools/gomad2` (module path `github.com/temporalio/gomad`, root `go.mod:62` require + `:263` replace, and the consumer `tools/gomad3/simulation/parity/manifest.go` that requires `tools/gomad2/` source paths); root-module tree `tools/gomad1` (no go.mod, zero callers of `ctrl/dropin.go` `RunInSim`); `Makefile:303-317` gomad-prototype targets; `.gitignore:5,71-72`.
  - gomad family, retained per `.plans/GOMAD_MILESTONES.md` F0: `tools/gomad3` plus `qualification/corpus/go.mod`, `tools/gomad3sim`, `tools/gomad3integration` (build tag `gomad3_integration`, `Makefile:344`), `tests/gomadfunctional`, `gomad3.yml`, `Makefile:201,240,242,319-352,1396,1477`, `.gitattributes:3-4`, `.github/.yamlfmt:13`, `.gitignore:4`. Record them as retained neighbors whose only edit is the parity-manifest retirement.
  - `tools/agentworkflow`: own go.mod; `Makefile:290-301,308`; docs links only.
  - `cmd/umpire-genmodels`: `Makefile:84,594-610`, `mise.toml:6-13`, `develop/umpire/install-tools.sh`, `umpire-model-verification.yml:105-116`.
  - `common/testing/umpire` (96 files): consumers `service/history/workflow/cache/cache.go:27` (the only importer under `service`, verified by `git grep -l 'umpireotel\|common/testing/umpire' -- service`), `tests/testcore/monitor/monitor.go:10`, `tests/probe`, `cmd/umpire-genmodels`; `.github/CODEOWNERS:98`. Six other history-service files carry observer comments only (`git grep -n -i 'umpire observer\|.plans/UMPIRE.md' -- service`); list them as comment-only rows.
  - `tests/testcore/monitor`, legacy `tests/umpire[23]_*.go`, `tests/probe`.
  - Retained neighbors with reasons: `tools/fairsim` + `cmd/tools/fairsim` (upstream PR #8158; `Makefile:561,573-575`, `.gitignore:55` stay), `tools/planindex` + `.plans/index.json` (fn-66 carve-out revalidated), `.flow/tmp/fn20.4-base-*` duplicate tree (record tracked or untracked via `git ls-files .flow/tmp | head`).
- Baselines: `go list -deps -tags 'test_dep integration' ./... | sort > .flow/tmp/fn81/deps-before.txt` and `go list ./... | sort > .flow/tmp/fn81/pkgs-before.txt`, plus `du -sh tools/umpire3 model/.lake` and `git ls-files | wc -l`.
- Search surfaces to run and cite: `git grep -n -E 'server/(tools/(gomad[12]?([^0-9a-z]|$)|umpire[123]|agentworkflow)|cmd/umpire-genmodels|common/testing/umpire)' -- '*.go'`, the same over `Makefile`, `.github`, `*.sh`, `*.toml`, `lakefile*`, `*.proto`, `model/**/*.lean`, `*.json`, `*.md`.

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
