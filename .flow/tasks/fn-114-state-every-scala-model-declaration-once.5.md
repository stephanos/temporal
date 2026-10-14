---
satisfies: [R1, R10, R11]
---
# fn-114-state-every-scala-model-declaration-once.5 Split the close-policy Model into the four-file layout

## Description
Move the converted close-policy declarations into fn-112's file names and remove the literals the conversion made avoidable.

**Size:** S
**Files:** `model/temporal/nexuscaller/closepolicy/{Model.scala, Claims.scala -> Properties.scala + Queries.scala}`, its task-1 IR-file declaration.
**Touches:** [model/temporal/nexuscaller/closepolicy/**, model/ir/nexus-close.json, tools/umpire/internal/golden/config.json]

### Approach
- Domains, actions, step functions, machines, compositions and monitors in `Model.scala`; Properties and progress claims in `Properties.scala`; Scenarios, Queries and Limits in `Queries.scala`. Omit a file with nothing of its kind. Keep the package so IDs stay exact. Moving functions out of `Claims.scala` renames their `Claims$package$` symbols (e.g. `awaitingOwner`, `nothingOwed`, `knows`); record each as a `function_name_substitutions` entry. If a move would change an IR name outside the allowed list, keep the declaration in its file and say so (R10 errors).
- Count this folder's literals with task 1's command and remove any outside fn-112 R18's three kinds that can go.

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir
make umpire-check-model
```

### Execution constraints
- `nexus-close.json` changes only within the R1 allowed-difference list recorded in the spec (fn-120 inert choice names, file-move source paths/lines and `source` root strings, and moved-function symbols recorded as `function_name_substitutions` in `tools/umpire/internal/golden/config.json`, as fn-112's harness allows).
## Acceptance
- [ ] closepolicy has `Model.scala`, `Properties.scala`, `Queries.scala` and no `Claims.scala`.
- [ ] IR differs only by allowed file-move metadata; R1 goldens and the model gate pass.
## Done summary
Split the close-policy Model into fn-112's file layout. Commits: 48f94d92d8 (split), 32f4fa7baa (claim bundles, review round 1).

**What changed**
- **Layout (R10).** closepolicy has `Model.scala`, `Properties.scala`, `Queries.scala` and `IrFiles.scala`. `Claims.scala` is deleted.
  - `Model.scala` is unchanged: domains, actions, step functions, monitors with their promises, assumptions and machines.
  - `Properties.scala` holds the promises (`closedHistoryIsFrozen` … `noUnnecessaryWait`), the claim bundles and the progress claims with `awaitingOwner`/`ownerKnowsOutcome`. The bundles follow the admission designs: `DesignClaims`/`designClaims(m)`, `SafetyClaims`/`safetyClaims(m)` and `DeadlineClaims`/`deadlineClaims(m)`, each field named after its Property's IR name.
  - `Queries.scala` holds the Limits and the three shared Query defs, which read `claims.x`, and the nine `*Queries` roots.
  - The package is kept, so every ID is unchanged.
- **IR (R1).** `nexus-close.json` differs only in positions, its `source` root string and the moved functions' symbols (`Claims$package$` → `Properties$package$`, which also re-sorts the inventory). Tables, IDs, Property, Query and progress records are equal apart from those.
- **Golden config, appended only:**
  - one directory `source_path_merges` entry, `model/temporal/nexuscaller/closepolicy/`. No nexuscaller directory entry exists, so nothing is swallowed.
  - 19 `source_root_moves`: 9 `*Queries` to `Queries$package$`, and 10 progress claims to `Properties$package$`.
  - 8 `function_name_substitutions`: `closedHistoryIsFrozen`, `knownIsTheHandlersOutcome`, `knows`, `knowledgeIsFinal`, `nothingOwed`, `noUnnecessaryWait`, `awaitingOwner` and `ownerKnowsOutcome`.
- **Harness fix.** `TestMigrationGoldensAdmitOnlyTheProjection` (model) and `TestMigrationProjectionKeepsLoweredCases` (lower) checked the substitutions' closedness against the Nexus caller's frozen IR alone, so any other file's substitution failed. They now read every frozen IR through the new `golden.FrozenModels`.
- **Go tests:**
  - `nexus_close_baseline_test.go` uses the `Properties$package$` symbols and reads default Query names from `claims.<property>`.
  - `diagnostics_test.go` uses the `Queries.scala` path.
- **R11 (`metrics-{before,after}.txt`).** Literals stay at 58: the split moves text and removes none. Lines went from 1,011 to 1,101 because the bundles' case classes and constructors were added. The fn-114 total for the folder is 1,207/148 → 1,101/58.
  - Remaining kinds: computed Query names, explicit names that differ from their val (Properties declared under a predicate's name, assumptions), ids (family, entity key and refer role, IR file name, the shared `outcomeReachesOwner` progress name), `because` prose, and the two evidence exceptions.
  - Outside the three kinds: `Properties.scala:255` `leadsTo("retainedReachesOwner")` and `:259` `leadsTo("retainedWaitsWithoutRecovery")` equal their own val names. They stay because `leadsTo` has no captured-name form, and adding one is a construct this spec's Boundaries exclude. That is a finding for fn-112 or a later spec.

**Decisions taken autonomously**
- **Properties go in bundles, not one file per Property.** Every design's Properties must be declared on that design, so a def per bundle over `m` is the only form. This follows the admission designs rather than leaving the Properties inside the Query defs, as round 1 required.
- **Assumptions stay in `Model.scala`.** The derived machines name them in `assuming`, and the README's rule keeps a machine's declarations with it.
- **Worktree only.** No canary re-pin was needed: no Case changed.

**Verification (gates.status).**
- gen-model ×2; OriginalBaseline + Migration + NexusClose + Validate (golden/model/lower); check-model; lint-model; full Go suite; lint-code-fast, all exit 0 after the final commit.
- `lint-model-2` failed once with a scalafmt native segfault (signal 11), and the rerun `lint-model-3` passed.
- `baseline-1` failed before the harness fix.

**Review.** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus).
- Round 1 was NEEDS_WORK with two findings: the Properties were still in `Queries.scala` (R10), and a test loop was duplicated (P3). Both were fixed in 32f4fa7baa.
- Round 2 was SHIP. Its R11 paperwork note is addressed above.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 48f94d92d8, 32f4fa7baa
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration|NexusClose|Validate' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0; one earlier run hit a scalafmt native segfault), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: