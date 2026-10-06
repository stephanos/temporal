---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-132-group-the-nexus-and-activity-models-by.7 Close: requirement check, MILESTONES, spec

## Description
**Size:** S
**Touches:** [model/README.md, .plans/UMPIRE_MODULES.md, .plans/ACTIVITY_MODEL_COMPARISON.md, MILESTONES.md, .flow/specs/fn-125-represent-dynamic-configuration-in-the.md, .flow/specs/fn-125-represent-dynamic-configuration-in-the.json, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.3.md, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.3.json, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.6.md, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.6.json, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.7.md, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.7.json, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.8.md, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.8.json, .flow/tasks/fn-119-show-one-go-sdk-workflow-driven-end-to.5.md, .flow/tasks/fn-119-show-one-go-sdk-workflow-driven-end-to.5.json]

**Required investigation:** all task handovers, R1-R5, current generated/model trees, downstream Flow plans and review receipts. Preserve existing spec dependency edges as history after dependencies close; only the overview's present-tense gates lose completed entries.
The conductor owns whole-spec completion review and closure. This task prepares/verifies linked requirement evidence and final docs; it does not close the parent before conductor completion SHIP.

Check R1-R5 against the tree and every task's done evidence, including the source-grouping prerequisite. Re-run the Model-path-specific old-name search across the repository. Verify downstream spec paths/gates and record the last applicable full boundary results without repeating unaffected suites. Leave the parent and its MILESTONES block for conductor-owned completion review/closure.

**Downstream path-maintenance scope:** The finite downstream Flow authoring paths above are writable only to make current Model paths and literal references match the completed kind/form tree. Preserve their deferred status, dependencies, semantics and historical snapshots; remove obsolete source-line ranges instead of inventing replacements. Do not resume any deferred implementation. This aligns the write surface with the existing final-docs/downstream-path acceptance.
## Acceptance
- [ ] Each of R1-R5 is met, with the evidence linked.
- [ ] Final docs and downstream Flow paths match the current tree; MILESTONES keeps completed tasks until the conductor closes the parent.
- [ ] Evidence is ready for whole-spec completion review, with no premature parent closure or claim of merged delivery.

## Done summary
I checked R1 to R5 against the tree and every task's done evidence, and all five are met. I also moved the downstream Flow plans and the activity comparison onto the kind/form folders and the new IR file names (fb9094ab54). The parent spec and its MILESTONES block are untouched: closing them is the conductor's job after the whole-spec completion review.

Tier: implementer (AGENTS.md model routing)
stage: impl-review - ran [single claude dispatch, first round] - SHIP. Receipt /tmp/impl-review-receipt-8f37faba39e2-fn-132-group-the-nexus-and-activity-models-by.7.json, output .flow/tmp/fn-132.7/review-1.out. It found 0 introduced issues and 2 pre-existing P3s, listed under Follow-ups.

### Requirement evidence
- **R1, layout:** the tree has `model/temporal/features/{nexus/{Nexus.scala,product,workflow,standalone},activity/{Activity.scala,standalone}}`, and none of the three old folders exists. The tasks that delivered it are fn-132.1 (four Nexus export stems: nexus-caller→nexus-workflow, nexus-control→nexus-workflow-control, nexus-close→nexus-workflow-close, nexus-operation→nexus-standalone) and fn-132.2 (activity→activity-standalone{,-record,-race}). Each has an exact move proof in its done summary.
- **R1, old-name search:** I searched the repository for Model paths and IR stems: `features[/.](nexuscaller|nexusoperation|standaloneactivity)`, `nexus-caller`, `nexus-operation`, `nexus-control`, `nexus-close`, `activity-record`, `activity-race`, `ir/activity.`. Every hit outside the residuals below is allowed:
  - retired-name negative tests in `tools/umpire/ir/layout_test.go`;
  - historical recorded Cases and Runs kept for crossing: canary `nexus-caller-syncCompletion-historical-*`, replay `nexusCallerControl-*`, evaluation receipts and the key/assess tests that read them;
  - pre-rename wire captures in `tools/umpire/ir/schema_test.go`;
  - dated `.plans/*` research snapshots, `.plans/archive`, and `docs/superpowers`;
  - names that only look old: the server package `chasm/lib/nexusoperation`, the `start-nexus-operation` command IDs and the `nexus-operation` log tag.
  The retired-path tests pass at HEAD: `go test -count=1 -tags test_dep ./tools/umpire/ir/ -run 'Layout|Retired|Isolation'` exits 0 over 9 tests (`.flow/tmp/fn-132.7/layout-test.log`).
- **R2:** `features/nexus/Nexus.scala` declares Reply, Resolution, Outcome, `handler` and `client.terminate`. `product/Product.scala` declares NexusProduct. Both forms declare `object refinement extends Refinement(…NexusProduct)` (`workflow/system/System.scala:112`, `standalone/system/System.scala:47`) and verify `NexusProduct.properties.terminalIsFinal` (`:407`, `:184`). The refusal of a step with no product step and the disabled handlerError classes are proved in fn-132.4 and fn-132.5.
- **R3:** `features/activity/Activity.scala:14-56` declares Timeout, AttemptResult, TimeoutType, `result`, `worker.poll`/`respond`, `timers` and `deadline`. `standalone/` keeps only the binding aliases `Standalone.scala:89-90`, as task 4 decided. Evidence: fn-132.6's EXACT PASS over 58 artifacts.
- **R4:** the three refusal fixtures are `model/irgen/testdata/layout/r4/{form-outside-kind,machine-in-general-file,second-general-file}`; the passing fixture is `layout/kinds/`. These come from fn-132.8 and fn-132.3. `model/README.md` (kind section and layout tree) and `.plans/UMPIRE_MODULES.md` (Nexus kind and Activity kind rows) describe the current tree, so they needed no edit.
- **R5:** the spec's Decision Context has the five "Task 4 PROVED" entries, each saying what it changed.

### Downstream paths (fb9094ab54)
- fn-125 spec, fn-125.3/.6/.7/.8 and fn-119.5 now name `features/nexus/workflow/{Workflow.scala,system/System.scala,Realization.scala}` and `features/nexus/standalone/Realization.scala`. The old `object Control` is now called `TrustingCaller` in `system/TrustingCaller.scala`. Obsolete line ranges are removed, and status, dependencies and meaning are unchanged. The JSON sidecars hold no paths, so they are unchanged.
- In `.plans/ACTIVITY_MODEL_COMPARISON.md`, the IR names are now `activity-standalone{,.lint}.json`, and its header describes `Activity.scala`'s general declarations from fn-132.6 in place of "package-only".
- Choice I made (owner away): for files that no longer exist (`Model.scala`, `Queries.scala`), I named the current file that holds their contents (Workflow.scala, system/System.scala, `NexusSystem.queries`). Pointing at the folder alone would have told the reader less.

### Gates
- baseline: green, reused from prior evidence. That evidence is fn-132.6's full boundary (`.flow/tmp/fn-132.6/verification-ledger.md`) plus the conductor's 6a50ba395a re-record (`.flow/tmp/fn-132.7/rerecord2.log`, exit 0). No test, gate or generator reads any file this range changes, so the full suites were not rerun.
- `flowctl gate classify` returned FULL ("unmatched: .plans/ACTIVITY_MODEL_COMPARISON.md"). This range is docs and Flow markdown only, and per the conductor's instructions no unaffected full suite was rerun. I also checked: none of the edited files is in `layout_test.go`'s scanned prose list.

### For the conductor (outside this task's Touches, not edited)
- R1 residuals: `tools/umpire/export/README.md:39-40` still names the slices `activity` and `activity-record`, but `quint_test.go:23` uses `activity-standalone` and `activity-standalone-record`. `model/check/test/Gate.test.scala:396-397` lists the stand-in names `activity-system.json` and `activity-race.json`; `activity-system` was already stale before fn-132.
- `.flow/specs/fn-128-*.md:22`, `fn-129-*.md:9` and their `.1` tasks still say the kind header `Activity.scala` "is package-only; fn-132.6 owns the later extraction". The paths are correct; only that sentence is stale.
- MILESTONES fn-132 block: the fn-132.6 row still says todo. I left it for closure as instructed.
- Review P3s: `fn-125.8.md:18` still cites `Realization.scala:179-181`, which is now 180-182 at `nexus/workflow/Realization.scala`; it could be dropped. The historical Case fixtures above keep `features/nexuscaller` provenance on purpose.
- I made no claim of merged delivery, and the parent is not closed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fb9094ab54462d27de7bde4147c5e345183b5560
- Tests: go test -count=1 -tags test_dep ./tools/umpire/ir/ -run 'Layout|Retired|Isolation', baseline: green via reuse of .flow/tmp/fn-132.6/verification-ledger.md full boundary + 6a50ba395a re-record (.flow/tmp/fn-132.7/rerecord2.log); range is docs/Flow markdown read by no gate
- PRs: