---
satisfies: [R3, R4]
---
# fn-81-delete-the-pre-testpilot-go-generations.4 Rewrite the Makefile, workflows, live-test gate, and dotfiles

## Description
Implements R4 and the CI-side half of R3 (spec §Live-test gate, commits 7 and 8). Removes every Makefile block, workflow, and dotfile entry that served a deleted tree, redesigns `umpire-check-live-tests` around the `^TestTestpilot` prefix with an empty baseline and a passing-identity floor, and updates the CI workflow test that pins the gate.

**Size:** M
**Files:** `Makefile`, `.github/workflows/umpire3.yml` (delete), `.github/workflows/umpire-model-verification.yml` (delete), `tools/umpire/regression/ci_workflow_test.go`, `.github/CODEOWNERS`, `.gitignore`
**Touches:** [Makefile, .github/workflows/umpire3.yml, .github/workflows/umpire-model-verification.yml, .github/CODEOWNERS, .gitignore, tools/umpire/regression/ci_workflow_test.go]

### Approach
- Makefile removals (line ranges from the ledger): `:84` genmodels variable; `:85-119` and `:147-178` UMPIRE3 variable blocks; `:290-301` agentworkflow targets; `:303-317` gomad-prototype targets; `:594-610` genmodels targets; `:612-999` and `:1282-1391` umpire3 target blocks; `:1392` the umpire3 `.PHONY` line. Keep `:120-146`, `:201` (GOMAD3_GO), `:240,242` and `:1396,1477` (gomad3 prune clauses), `:319-352` (gomad3 targets), `:1000-1003`, `:1004-1006` (planindex), and `:1008-1138`.
- Gate at `Makefile:1139-1171`: command becomes `go test -v -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilot'` (the `-v` is required because `--- PASS:` lines only print in verbose mode; the existing non-verbose form prints `--- FAIL:` only); expected file empty; keep the `--- FAIL:` scrape and `diff -u`; add a floor `sed -n -E 's/^[[:space:]]*--- PASS: ([^ ]+).*/\1/p' | grep -c .` that must be at least 1, with a distinct failure message. Keep the "failed without reporting a test identity" branch.
- `ci_workflow_test.go`: update `liveTestCommand` (`:25`) to the new verbose command and selector, replace `inheritedLiveFailures` (`:33-43`) with an assertion that the dry-run contains the empty-baseline and floor messages, keep the `make -n` counts (`:134-143`) and the whole-workflow `require.Equal` (`:86`) unchanged since `umpire.yml` is untouched.
- Delete the two workflows; `gomad3.yml` stays. CODEOWNERS: remove `:99-102`. `.gitignore`: remove `:5,16-18,27-30,46-49,68-69,71-72`; keep `:4` (`.gomad/` is the gomad3 artifact store), `:19-22,55,67,70`. `.gitattributes` and `.github/.yamlfmt` are untouched; their entries serve the retained `tools/gomad3`.
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
- [ ] `grep -n -E 'umpire[123]|gomad[12]?([^0-9a-z]|$)|agentworkflow|genmodels' Makefile` returns nothing and `grep -c gomad3 Makefile` is unchanged; `make -n umpire-check-regression` succeeds
- [ ] The two workflows are deleted; `.github/workflows/umpire.yml` and `.github/workflows/gomad3.yml` byte-identical
- [ ] `make umpire-check-live-tests` runs `go test -v` with the `^TestTestpilot` selector and passes against a live cluster with an empty expected set and at least one `--- PASS` identity; forcing an empty selector match fails on the floor message
- [ ] `ci_workflow_test.go` pins the verbose command, the selector, the empty baseline, and the floor; `CGO_ENABLED=0 go test -tags test_dep ./tools/umpire/regression/...` passes
- [ ] CODEOWNERS, `.gitignore`, `.gitattributes`, `.yamlfmt` carry no deleted paths; `/fairsim` and planindex entries remain
- [ ] `make lint-code` passes

## Done summary
Removed every Makefile variable, target, and `.PHONY` name that served a deleted tree — all
`UMPIRE3_*` variables, `UMPIRE_GENMODELS`, and 83 targets across the umpire3, genmodels,
agentworkflow, and gomad-prototype blocks — plus the `umpire3` and `umpire-model-verification`
workflows, the CODEOWNERS rows over deleted paths, and the `.gitignore` rows for deleted trees. The
gomad3 wiring, the planindex target, and the fairsim rules are untouched, and the gomad3 line count
in the Makefile is unchanged.

The `gomad-prototype` block was not a block whose only purpose was a deleted tree, as the .1 ledger
recorded. Two of its six lines served retained trees: `cd model && lake build Shared`, which
`umpire-build-model` already covers because `model/lakefile.lean:59` declares `Shared` a default
target, and the `tools/common/formal` test invocation, which is preserved here as its own
`common-formal-test` target because fn-81 removed that nested module's only importer.

`umpire-check-live-tests` now runs `go test -v` with the `^TestTestpilot` prefix against an empty
expected-failure set. The verbose flag is required because `--- PASS:` lines only print under `-v`.
The whole-set `diff -u` comparison the recorded pitfall requires is kept; the pinned nine identities
are gone. The empty baseline is paired with a floor requiring at least one passing identity, because
an empty baseline alone cannot distinguish "everything passed" from "the selector matched nothing" —
verified by running the gate's shell against a selector that matches nothing, where `go test` exits 0
with an empty failure set and only the floor catches it. `ci_workflow_test.go` pins the verbose
command, the selector, the empty baseline, the floor scrape and its message, and asserts the nine
retired identities no longer appear.

`make lint-code` fell from the 1,284-issue task-start baseline to 128, since 1,156 findings lived in
the deleted trees. None of the 128 is in a file this spec edited. The gate still exits 2, so it
remains inherited-red rather than passing; the acceptance line "make lint-code passes" was not
achievable at any point in this spec.

Two acceptance criteria could not be met literally and are recorded rather than forced. The Makefile
grep still matches one line, the retained `./.gomad` prune clause for the gomad3 artifact store,
which the task's own Approach explicitly keeps — the acceptance regex is over-broad relative to the
Approach. And `make lint-code` is the inherited red above.

stage: impl-review - ran (backend claude, model claude-fable-5-1, effort high, 2 rounds: NEEDS_WORK
on a comment naming deleted paths and on a collapsed blank line inside `define NEWLINE` that the
block-removal pass had silently changed from a newline to the empty string, then SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a64f9b9d69f324a4a642caadefd9487d556d9b06, 24107bbab1118740d58bf6a4968b3e52dad6dae9
- Tests: make -n umpire-check-live-tests (renders the verbose command, the ^TestTestpilot selector, the empty baseline, and the floor), make umpire-check-live-tests (rc=0 against a live cluster: 'Live Testpilot failure identities match the empty expected set across 4 passing identities.'), floor negative check: the same gate shell with -run '^TestNoSuchSelectorZZZ' exits go test 0 with an empty failure set and the floor still trips - the case an empty baseline alone would have passed, make -n umpire-check-regression (rc=0), make common-formal-test (rc=0, 2 packages ok), CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/regression/... (rc=0), make lint-code GOLANGCI_LINT_FIX=false: 128 issues, down from the 1,284 task-start baseline; none in any file this spec edited (errcheck 220->1, exhaustive 5->0, forbidigo 209->0, goimports 1->0, govet 5->4, revive 732->106, staticcheck 111->17, testifylint 1->0). Still exit 2, so this gate stays inherited-red, Makefile removal audit: every removed non-blank line belongs to a removed target, variable, or .PHONY name, or to the rewritten gate; gomad3 line count unchanged at 29, grep -n -E 'umpire[123]|gomad[12]?([^0-9a-z]|$)|agentworkflow|genmodels' Makefile returns one line: the retained './.gomad' prune clause for the gomad3 artifact store
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
