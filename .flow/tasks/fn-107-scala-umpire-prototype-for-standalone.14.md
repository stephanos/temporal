---
satisfies: [R3, R4, R5, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.14 Extend generic finite checker hooks and progress semantics

## Description
**Touches:** [model/go/umpire/**]

Extract the generic checker work from task 3. This task uses existing finite Go tables and callbacks; the IR interpreter is a separate lane. Feature behavior remains authored in Scala.

**Size:** M
**Files:** generic search/claims/refinement/composition modules and focused generic tests.

### Approach
- Extend existing table/search owners with a small reusable boundary for passive observer state. Include that state in visited identity, preserving all machine transitions and existing deterministic witness order.
- Extend generic refinement to check initial correspondence and reject a visible event/result treated as stutter. Preserve invisible stutter, existing Definition IDs, and legacy behavior in the absence of the new declaration.
- Supply generic finite deadline/deadlock/fair-cycle checking hooks with explicit assumptions and resource limits. Distinguish unresolved prefixes and exhausted work from verified progress.
- Keep composition and scoped provider substitution generic; account for all admitted initial states. Never encode activity eligibility, Nexus ownership, or other feature policy in Go.
- Replay witnesses against the same finite transition relation. Preserve existing comments and baseline model behavior.

### Investigation targets
**Required:** model/go/umpire/search.go; model/go/umpire/claims.go; model/go/umpire/refine.go; model/go/umpire/compose.go; model/go/umpire/table.go; model/scalav2/SEMANTICS.md.
**Optional:** reviewed specimens and task 2's source declarations, read-only.

### Quick commands
`mise exec -- go test -tags test_dep ./model/go/...`; scoped generic Go lint. Record any inherited repository-wide lint failure separately.
## Acceptance
- [ ] Distinct passive-observer histories remain distinct explored states; observer state never suppresses a machine behavior.
- [ ] Initial-state and visible-output refinement mutation controls fail with replayable diagnostics; invisible stutters and existing baseline checks pass.
- [ ] Generic deadlock/deadline/fair-cycle controls distinguish proved failures, unresolved prefixes, and explicit work-limit exhaustion under named assumptions.
- [ ] Composition/substitution accounts for admitted initial states and preserves existing table/identity behavior; all generic and baseline Go model tests pass.


## Done summary
Finite Go tables now support passive monitors, visible refinement, bounded progress, composition checks and Definition-ID witness replay.

The independent native rereview returned SHIP. Final integrated Scala-to-Go, focused Go tests and scoped lint passed. Tested source and all original comments are retained; the user owns commits.

stage: implement - ran (model: claude-opus-5-5; CLI --model opus --effort high; owner session c2176cbf-880f-463d-ad38-9fb4d904ec51)
stage: impl-review - ran (codex:gpt-5.6-sol:high; first-round three-axis fanout and same-primary fix rereview; SHIP)
stage: wave-join - ran (2/2 returned; guarded uncommitted copying; final integrated gates rc0; clones retained because they contain uncommitted work)
Tier: session (jev-unavailable(no_key)); retained pinned opus/high.

stage: plan-sync - skipped(config: planSync.enabled != true)

stage: status-replay - ran (2026-09-30: this task's runtime status was recorded in a checkout that no longer exists, so it read todo here; the status is replayed from the summary above. Verified in this checkout: `go test -tags test_dep ./model/scalav2/... ./model/go/...`, `make umpire-check-scala`, `make lint-scala` and `model/scala/run.sh --no-prove` pass, with the Lean-dump comparisons skipped because the git-ignored dumps are absent and the Lean toolchain was removed.)
## Evidence
- Commits:
- Tests: GOFLAGS=-tags=test_dep make umpire-check-scala (rc0; /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/parallel-integrated-gates/scala-wave-foundations-final.log), mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc0; /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/parallel-integrated-gates/go-wave-foundations-final.log), GOFLAGS=-tags=test_dep mise exec -- make lint-code 'LINT_CODE_TARGETS=./model/go/umpire ./model/scalav2/goir' GOLANGCI_LINT=.flow/tmp/fn-107/task2-tools/golangci-lint-v2.13.1 ERRORTYPE=.flow/tmp/fn-107/task2-host-tools/errortype GOLANGCI_LINT_FIX=false (rc0; /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/parallel-integrated-gates/lint-wave-foundations-final.log), Actual independent native rereview SHIP; task14-final-review-receipt.json; task14-review-r2 immutable source artifact, Unchanged reviewed source, user HEAD and raw index guarded at bbc7dab1a4f9f0760f8c9316e9bfc5d9701334b7; task14-post-ship-guards.json; worker evidence at /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/task14-reviewfix1-evidence.json
- PRs: