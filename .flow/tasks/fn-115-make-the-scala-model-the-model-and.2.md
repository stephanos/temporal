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
Captured immutable semantic and artifact evidence for all twelve checked/fixture IR inputs: 48 reader snapshots and 1,363 artifact snapshots. Coverage includes complete ordered tables and evidence, refinements, every-row and refined Properties, declaration IDs/fingerprints, every Query standing, generated manifests and Cases, separate Program/Contract files, exploration reductions and identities, and original seven typed/keyed producer pairs plus synthetic job claims. Both approved source variants are frozen; opt-in capture rejects existing directories and ordinary tests never rewrite snapshots.

Ordinary verification strictly matches current inputs and regenerates both complete original/mapped variants and original-oracle artifacts. A review-discovered inactive-variant gap was fixed, with sixteen shared-path regression cases covering inactive Contract/proposal/identity/manifest/HTML and missing/extra/unknown inventory. All 1,411 snapshots and 843 protected originals remain unchanged; the index is unchanged and all added files are Git-visible. The module map records test-only support ownership, precise external lowerer integration-test edges and capture/inspection commands; task 3/12 briefs carry those constraints.

Full focused goir/... tests passed (reader155.163s, lowerer205.133s), final shared-comparator regression passed0.615s, make lint-code-fast reported0issues, and diff whitespace/preservation checks passed. Codex gpt-6.1-sol high returned SHIP in round2. Review receipt: .flow/tmp/fn115-2-review/receipt.json. Review used exact uncommitted hashes because the installed wrapper excludes this scope; read-only independent context, same family. No staging or commits.

stage: implementation - ran (model: gpt-6-astra); existing agent reused after host thread limit prevented fresh worker; disk re-anchor.
stage: verification - ran (focused Quick commands, mutation tests, deterministic captures and preservation checks).
stage: impl-review - ran (model: gpt-6.1-sol); SHIP round2 after one P2 fix.
stage: plan-sync - skipped(config: planSync.enabled != true); necessary downstream amendments applied through Flow.
stage: tracker-sync - skipped(bridge inactive).
stage: commit - skipped(user reserves commits and staging).
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/..., make lint-code-fast, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/testpilot -run '^TestMigrationArtifactProjectionIsClosed$' -count=1, mise exec -- go list -tags test_dep -deps -test ./model/scalav2/goir, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/goir/testpilot -run '^TestMigrationArtifactInventoryRejectsInactiveChanges$' -count=1
- PRs: