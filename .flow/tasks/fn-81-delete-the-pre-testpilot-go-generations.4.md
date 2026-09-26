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
- Makefile removals: the genmodels variable, Umpire3 variable blocks, agentworkflow targets,
  genmodels targets, Umpire3 target blocks, and their `.PHONY` entries.
- Gate at `Makefile:1139-1171`: command becomes `go test -v -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilot'` (the `-v` is required because `--- PASS:` lines only print in verbose mode; the existing non-verbose form prints `--- FAIL:` only); expected file empty; keep the `--- FAIL:` scrape and `diff -u`; add a floor `sed -n -E 's/^[[:space:]]*--- PASS: ([^ ]+).*/\1/p' | grep -c .` that must be at least 1, with a distinct failure message. Keep the "failed without reporting a test identity" branch.
- `ci_workflow_test.go`: update `liveTestCommand` (`:25`) to the new verbose command and selector, replace `inheritedLiveFailures` (`:33-43`) with an assertion that the dry-run contains the empty-baseline and floor messages, keep the `make -n` counts (`:134-143`) and the whole-workflow `require.Equal` (`:86`) unchanged since `umpire.yml` is untouched.
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
- [ ] `make umpire-check-live-tests` runs `go test -v` with the `^TestTestpilot` selector and passes against a live cluster with an empty expected set and at least one `--- PASS` identity; forcing an empty selector match fails on the floor message
- [ ] `ci_workflow_test.go` pins the verbose command, the selector, the empty baseline, and the floor; `CGO_ENABLED=0 go test -tags test_dep ./tools/umpire/regression/...` passes
- [ ] CODEOWNERS, `.gitignore`, `.gitattributes`, `.yamlfmt` carry no deleted paths; `/fairsim` and planindex entries remain
- [ ] `make lint-code` passes

## Done summary
Removed every Makefile variable, target, and `.PHONY` name that served a deleted tree — all
`UMPIRE3_*` variables, `UMPIRE_GENMODELS`, and 83 targets across the umpire3, genmodels,
and agentworkflow blocks — plus the `umpire3` and `umpire-model-verification` workflows, the
CODEOWNERS rows over deleted paths, and the `.gitignore` rows for deleted trees.

The `tools/common/formal` test invocation is preserved as its own `common-formal-test` target.

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

The `make lint-code` acceptance criterion remained inherited-red as recorded above.

stage: impl-review - ran (backend claude, model claude-fable-5-1, effort high, 2 rounds: NEEDS_WORK
on a comment naming deleted paths and on a collapsed blank line inside `define NEWLINE` that the
block-removal pass had silently changed from a newline to the empty string, then SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a64f9b9d69f324a4a642caadefd9487d556d9b06, 24107bbab1118740d58bf6a4968b3e52dad6dae9
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
