---
satisfies: [R3, R4]
---
# fn-81-delete-the-pre-testpilot-go-generations.4 Rewrite the Makefile, workflows, live-test gate, and dotfiles

## Description
Implements R4 and the CI-side half of R3 (spec §Live-test gate, commits 7 and 8). Removes every Makefile block, workflow, and dotfile entry that served a deleted tree, redesigns `umpire-check-live-tests` around the `^TestTestpilot` prefix with an empty baseline and a passing-identity floor, and updates the CI workflow test that pins the gate.

**Size:** M
**Files:** `Makefile`, `.github/workflows/umpire3.yml` (delete), `.github/workflows/gomad3.yml` (delete), `.github/workflows/umpire-model-verification.yml` (delete), `tools/umpire/regression/ci_workflow_test.go`, `.github/CODEOWNERS`, `.gitignore`, `.gitattributes`, `.github/.yamlfmt`
**Touches:** [Makefile, .github/workflows/umpire3.yml, .github/workflows/gomad3.yml, .github/workflows/umpire-model-verification.yml, .github/CODEOWNERS, .github/.yamlfmt, .gitignore, .gitattributes, tools/umpire/regression/ci_workflow_test.go]

### Approach
- Makefile removals (line ranges from the ledger): `:84` genmodels variable; `:85-119` and `:147-178` UMPIRE3 variable blocks; `:201` GOMAD3_GO; prune clauses at `:240,242,1396,1477`; `:290-301` agentworkflow targets; `:303-317` gomad targets; `:319-352` gomad3 targets; `:594-610` genmodels targets; `:612-999` and `:1282-1391` umpire3 target blocks; `:1392` the umpire3 `.PHONY` line. Keep `:120-146`, `:1000-1003`, `:1004-1006` (planindex), and `:1008-1138`.
- Gate at `Makefile:1139-1171`: command becomes `go test -v -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilot'` (the `-v` is required because `--- PASS:` lines only print in verbose mode; the existing non-verbose form prints `--- FAIL:` only); expected file empty; keep the `--- FAIL:` scrape and `diff -u`; add a floor `sed -n -E 's/^[[:space:]]*--- PASS: ([^ ]+).*/\1/p' | grep -c .` that must be at least 1, with a distinct failure message. Keep the "failed without reporting a test identity" branch.
- `ci_workflow_test.go`: update `liveTestCommand` (`:25`) to the new verbose command and selector, replace `inheritedLiveFailures` (`:33-43`) with an assertion that the dry-run contains the empty-baseline and floor messages, keep the `make -n` counts (`:134-143`) and the whole-workflow `require.Equal` (`:86`) unchanged since `umpire.yml` is untouched.
- Delete the three workflows. CODEOWNERS: remove `:99-102`. `.gitignore`: remove `:4-5,16-18,27-30,46-49,68-69,71-72`; keep `:19-22,55,67,70`. `.gitattributes:3-4` and `.github/.yamlfmt:12-13` removed.
- Verify: `make -n umpire-check-live-tests`, `make umpire-check-live-tests` (integration tag, needs a cluster), `CGO_ENABLED=0 go test -tags test_dep ./tools/umpire/regression/...`, `make lint-code`.

### Investigation targets
**Required** (read before coding):
- `Makefile:1139-1175` — the gate and the aggregate target that depends on it
- `tools/umpire/regression/ci_workflow_test.go:20-145` — pinned commands and dry-run assertions
- `.flow/memory/bug/integration/full-integration-gates-must-select-the-2026-09-04.md` — whole-set comparison rule

**Optional** (reference as needed):
- `.github/workflows/umpire.yml` — the retained workflow the test decodes

### Key context
- fn-77 tasks .10 and .11 add `TestTestpilotTypedNexusOperationsCase` and `TestTestpilotTypedUnaryCase`; the prefix selector picks them up without edits.
- fn-80 tasks .4 and .8 currently say the expected-failure list is unchanged; task .5 rewrites those records.

## Acceptance
- [ ] `grep -n -E 'umpire[123]|gomad|agentworkflow|genmodels' Makefile` returns nothing; `make -n umpire-check-regression` succeeds
- [ ] The three workflows are deleted; `.github/workflows/umpire.yml` byte-identical
- [ ] `make umpire-check-live-tests` runs `go test -v` with the `^TestTestpilot` selector and passes against a live cluster with an empty expected set and at least one `--- PASS` identity; forcing an empty selector match fails on the floor message
- [ ] `ci_workflow_test.go` pins the verbose command, the selector, the empty baseline, and the floor; `CGO_ENABLED=0 go test -tags test_dep ./tools/umpire/regression/...` passes
- [ ] CODEOWNERS, `.gitignore`, `.gitattributes`, `.yamlfmt` carry no deleted paths; `/fairsim` and planindex entries remain
- [ ] `make lint-code` passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
