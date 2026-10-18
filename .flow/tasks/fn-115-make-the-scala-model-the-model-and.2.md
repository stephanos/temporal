---
satisfies: [R2, R11, R21]
---
# fn-115-make-the-scala-model-the-model-and.2 Capture exhaustive semantic and artifact goldens

## Description
Capture exhaustive semantic and artifact goldens. Implements R2, R11, R21 using the reviewed parent contracts.

**Size:** M
**Files:** model/scalav2/goir golden support/tests and testdata; model/scalav2/goir/testpilot generation support; scoped golden capture entrypoint selected by the map
**Touches:** [model/scalav2/goir/**, tools/umpire/cmd/**, .plans/umpire-migration-*.json, Makefile, .gitignore]

### Approach
- Include the audit’s `test_claim_transfers`: freeze all seven original Nexus typed/keyed parity pairs (full Cases, tables, claims, witnesses and identical realization/source inputs) plus the generic job fixture semantics needed by the ten producer occurrence/projection tests. These oracle baselines remain separate from R22 replacement execution pins.
- Capture every checked-in and fixture IR before any source relocation. Reuse tableSide/sideOf, generic receipts and the full generated Query manifest; include definitions, ordered evidence, tables/disabled pairs, all refinement results, every-row Property answers, Query outcomes/unsupported standings and full lowered Case bytes.
- Keep new baseline files discoverable by Git despite the root testdata exclusion: add narrow current-layout ignore exceptions when needed and verify the exact paths with git check-ignore.
- Keep Program and Contract independently inspectable. Add an explicit capture path separate from ordinary verification; tests never rewrite their expected baseline.
- Implement only the exact path/derived-hash transformation fixed by task 1. Require complete inventory membership and independently recomputed derived values rather than ignored fields. Namespace/wire checks remain a separate task.
- Prove sensitivity by changing a representative table row, every-row Property result, refinement, evidence ordering, Definition ID/fingerprint, unsupported standing and Case Contract; reject unknown/missing baseline entries and unlisted path changes. Record deterministic repeat capture and retain the unmodified original baseline.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/goir/activity_parity_test.go:48`
- `model/scalav2/goir/activity_properties_test.go:278`
- `model/scalav2/goir/checking.go:157`
- `model/scalav2/goir/testpilot/generated.go:44`
- `model/scalav2/goir/nexus_close_baseline_test.go:117`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/...; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.

## Acceptance
- [ ] All IR/fixture inputs and all semantic/artifact categories are represented and ordinary tests verify without rewriting.
- [ ] Closed path migration reproduces only approved source fields and their derived identities; unknown transformations or missing inventory entries fail.
- [ ] Representative semantic and artifact mutations fail, including Property answers not named by a Query.
- [ ] Two explicit captures are byte-identical and the pre-move baseline is preserved for subsequent tasks.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: