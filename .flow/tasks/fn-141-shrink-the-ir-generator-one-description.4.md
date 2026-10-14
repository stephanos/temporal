---
satisfies: [R3, R4, R16]
---
# fn-141-shrink-the-ir-generator-one-description.4 Declaration-level sugar by definition; the lifter's sugar file and its lint exemption go

## Description
Part A, second half. The claim patterns and the sticky monitors are compositions of core builders, so their definitions can be followed like any helper def. After this task the lifter names no sugar that Part A can reach, and the syntax lint says so.

**Size:** M
**Files:** `model/framework/export/{Syntax,Claims,Declarations,Expansion,Expressions}.scala`, `model/framework/dsl/{Syntax,Claims}.scala`, `model/framework/check/lint/**`, `model/framework/inspect/**`, affected fixtures and coverage, `model/SEMANTICS.md`.
**Touches:** [model/framework/export/**, model/framework/dsl/**, model/framework/check/**, model/framework/inspect/**, model/framework/lint/**, model/temporal/check/**, model/SEMANTICS.md, model/README.md]
**Activation (2026-10-13):** fn-141.3 is committed and done (receipt `550ce56a0e`). Current homes replace the old `irgen`, `umpire`, and `check` paths below. Re-read and recount before editing. Mini/shared/framework/kit/sample scope only; no real activity/Nexus, generated payloads, schema, Go, tooling or Makefile edits. Stay on `umpire`, single writer, review none under the batch policy, planSync=false. Independent review, broad lint and strict IR/Go propagation run at the source-batch boundary, not as waived requirements.

### Approach
- Write `once(...).keeps(...)`, `never(...)`, `never(...).from(...)`, `stays(...)` and `stays(...).unless(...)` as bodies over `holds` and `holdsAcross`. Today they construct the Property record directly, which the lifter cannot follow.
- `sticky` and `stickyAcross` already call `monitor`; follow them.
- Names must not move: a predicate passed as a lambda lifts today as a function named after the word that takes it, and a `keeps` projection as a field path. Reproduce those names through the generic binding, so the expected IR does not change.
- Constructed-value-only forms retain their arms until their Part B owner: named inputs in .9, request assignments and script helpers in .8. Recount all remaining sugar reductions after .3, including declaration witnesses/rules/blocks; list each with its .4 or .10-.12 owner. No permanent name-dispatch exception, relabeling as core, or new sugar-specific intrinsic.
- Extend the syntax lint: no lifter file names a sugar definition, apart from the arms just listed, each with the task that removes it. Delete the lifter's `Syntax.scala` if nothing is left in it.
- Record production export, lint and inspect lines before/after Part A (R16), excluding tests/testdata/project files and counting new generic infrastructure. The .3 census is export 12,426, lint 2,995, inspect 107, total 15,528. Do not claim net shrink unless the whole census proves it.
- Reuse indexed definition expansion from .3, preserve strict/default/by-name evaluation, caller positions, callback names and all frozen expected IR/refusal contracts. Framework definitions remain in the shared Index without becoming authored Model roots; explicit framework-package lint fixtures remain supported by source provenance.

### Investigation targets
**Required:**
- `model/irgen/Syntax.scala` (`patterned`, `pattern`, `stickyMonitor`, `keptPath`, `named`, `requestAssignment`)
- `model/irgen/Declarations.scala` (`stepFunction`, `monitorOf`) for the names a lambda lifts under
- `model/umpire/Syntax.scala` (`Once`, `Never`, `Stays`, `sticky`) and `model/umpire/Claims.scala`
- `model/check/SyntaxRule.scala` (`sugarNames`, the three rules) and its test


**Author-time liftable-subset check (owner, 2026-10-13).** Beyond holding the lifter to core constructs, the syntax lint (now under `framework/check/lint` after fn-169.1) refuses non-liftable Scala in step functions and predicates while they are written, not only at lift. The check covers tree forms outside the subset and library operations outside the allowlist, and it names the construct and points to the subset table. It runs in `make model-check-changed`.

**One documented subset (owner, 2026-10-13).** `model/SEMANTICS.md` gets one table of the liftable subset: each supported tree form and each allowed library operation (today `contains`, `copy`, `map`, `find`, `min`, `max`, `++`, `toList`, `toVector`, ...), with its IR lowering. Each row has one fixture. Adding an operation means adding a row and a fixture, so growth is a visible decision. The lint and the lifter read the same allowlist.
## Acceptance
- [ ] No lifter code names a claim pattern or a sticky monitor.
- [ ] The sugar fixture's pattern and monitor pairs lift to the same IR as before, function names included.
- [ ] The syntax lint fails on a sugar name in a lifter file, naming file, line and sugar; its test covers it.
- [ ] The remaining arms are listed with the task that removes each.
- [ ] Focused red/green tests cover declaration sugar and author-time refusal controls; then a fresh terminal `make model-check-changed BASE=<task-base>` passes (allow at least 1,100 seconds). All samples, mini baseline step-table equivalence and selected suites pass. Frozen fixture expected IR, function names, positions and refusal contracts are unchanged. Checked-in mini payload drift and full generated IR/Go gates remain explicitly held for propagation; no regeneration here.
- [ ] Line counts before and after Part A are in the done summary.

- The lint refuses a non-liftable construct in a step function at author time, naming it; the lint and the lifter share one allowlist.
- SEMANTICS has the liftable-subset table, with one fixture per row.
## Done summary
# fn-141.4 verified implementation

Claim patterns and sticky monitors lift through indexed definitions. The author-time subset lint runs in the checked-lift gate and shares typed library admission with function lifting. The final committed tree passed the fresh change-scoped gate; the conductor records Flow closure from this evidence.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none; source-batch policy)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker inactive under the dispatched snapshot `.flow/tmp/fn-141-4-sync-active.json`.

### Checkpoint and closure evidence

- Base: `36de9c9081a57570d7bb2cb4047bc8753a4b1d0f` (unchanged).
- Source commits: `6aab636863122713e8e8f4c11114873b740ab473`, then `f406f437679e00650562dfef0304dd6e2dd1e977`.
- The conductor ran a fresh `timeout 1200s make model-check-changed BASE=36de9c9081` on the final tree. Session 6920 exited 0. Record `.flow/tmp/checks/20261013-081114/record.md` reports passed in 698.8 seconds, with 100 selected suites, 144 Model tests, 311 lifter tests and 155 check tests (610 total), three samples and mini step-table equivalence. The failed predecessor record remains intact.
- Original 100-minute cap was extended once, conditionally after focused/syntax green, to 14:52 UTC. The worker returned a committed partial within that cap. The conductor verified the handover and ran the remaining canonical gate without another implementation-worker budget.
- Checkout clean and no command live at handover. No push, PR, rebase, generated payload, schema, Go, tooling, Makefile, real-feature or MILESTONES edits.

### Implemented

Claim patterns now compose core `holds`/`holdsAcross` builders in their indexed marked definitions. Marked constructor/type/field bindings and callback origins preserve existing function names, field projections and caller positions. Sticky monitors follow their definitions into `monitor`. The dedicated pattern/sticky reducers were deleted; the remaining exporter Syntax file is temporary Part B work, not a permanent exception.

Pattern no-`when` policy is ordinary `require` in the framework definitions, checked generically against already-folded builder metadata. It no longer rejects arbitrary marked wrappers solely for being marked. A restricted kit helper compares with a core twin on a distinct Model, preserving the genuine duplicate-promise invariant. `Never.from(before = ...)` retains its public named parameter through a generic immutable callback alias. Authored section-block references are retained rather than eagerly unfolded into invalid capture sites.

Author-time lint checks semantic capture sites and reachable authored helpers. It shares typed Scala/Java library admission with the lifter, distinguishes expression and pattern trees, keeps draft Unit scoped to effect captures, and checks authored contextual arguments while recognizing compiler metadata. `checkAndLift` enables this phase on the same admitted source set; plain `lift` retains frozen refusal precedence. Explicit framework-package lint fixtures still participate; production framework definitions remain indexed without becoming Model roots. No corpus/path exemption was added.

SEMANTICS documents tree/operation lowerings, typed receiver and declaration/body context restrictions, with fixtures including declaration-only `map`/`toVector`, Query `find`, and negative body controls. Syntax lint checks code, qualified DSL references and dispatch strings, including the remaining exporter Syntax file, with bounded owner/name exemptions.

The original 152-line pattern/core comparison was moved out of `Fixtures.test.scala` into `DeclarationSugar.test.scala`, with assertions retained and the new suite registered in coverage. Frozen expected IR/refusal ledgers were not edited.

### Evidence

- Pre-edit baseline: green, SugarFixtures, 1 test; `.flow/tmp/fn-141.4-baseline.log`.
- Meaningful reds: declaration callback expansion (19/1), missing exporter sugar diagnostics (14/1), missing author-time subset refusal (21/2), isolated empty draft Unit refusal (2/1), missing named `before` API (compiler diagnostic), marked restricted helper over-refusal (19/1), and qualified DSL reference omission (14/1). Corresponding logs use the `fn-141.4-` prefix in `.flow/tmp`.
- Intermediate failures were not waived: stale packaged DSL, invalid annotation syntax, typed factory/pattern/metadata false positives, draft-context loss, typed core constructor shape, precondition receiver/name handling, and the invalid initial same-Model twin setup were corrected. The two wrong JUnit class selectors and an incorrectly placed temporary syntax control were inconclusive, not green evidence.
- Focused declaration/subset/sugar: green 23 tests, `.flow/tmp/fn-141.4-focused21.log`.
- Syntax: green 14 tests, `.flow/tmp/fn-141.4-syntax-green4.log`.
- Native semantics/named API/pattern policy: green 5 tests, `.flow/tmp/fn-141.4-native-final.log`.
- Canonical gate on `6aab636863`: **failed**, outer exit 2, 590.8 s. Author syntax/style/boundary, all 3 samples, mini compile/export, mini step-table equivalence and 144 Model tests passed. Lifter suites: 311 tests, 1 failure (Boolean match-result union rejected by exact-type equality). Checks suites not reached. Authoritative record: `.flow/tmp/checks/20261013-073035/record.md`; full log `.flow/tmp/fn-141.4-scoped-final.log`.
- Final fix uses typed Boolean subtyping, not a name whitelist. Final focused run after that fix and test migration: **green 121 tests**, exit 0, 175.761 s, `.flow/tmp/fn-141.4-partial-final-focused.log`. Suites: DeclarationSugarFixtures, DefinitionExpansionTest, LiftableSubsetFixtures, SugarFixtures, ConventionsFixtures, Fixtures. This covers the enum-pattern regression, moved pattern pairs, complete frozen IR/refusal contract, callbacks, library wrong-receiver/contextual controls and dual author/lift diagnostics.
- Formatting and `git diff --check` passed. The fresh final canonical gate passed with exit 0; `.flow/tmp/fn-141.4-conductor-gate.log` retains the complete output. Gate receipt and Flow closure follow verification of that terminal record.

### Remaining reducer inventory

| Reduction/capture | Removal owner |
| --- | --- |
| Named action inputs and `:=` slots | fn-141.9 |
| Request assignments/scopes, protobuf/response scopes, script helpers and realization vocabulary (`field`, `proto`, `read`, script `when`/records/from) | fn-141.8 |
| RuleBook headings/from/on/cases/disabled actions, rule effects (`enters`, `rejects`), phase writers/role headings, `is`/`effect` blocks and captured `Ok` outcomes | fn-141.10 |
| Composition member selectors, sync/own/synced and replacement readers | fn-141.11 |
| Witness paths, records/excludes/recordsWhen and modifiers; property/scenario/query/progress declaration folding | fn-141.12 |
| Capability fact-family and generated Property expansion | fn-141.12/.13 |
| Reserved core `Bindings` oracle retirement | fn-141.10/.14 |

The file/name exemptions are enumerated in `SyntaxRule.remainingExportSugar`; claim patterns and sticky monitors have none. `Bindings` stays only as the reserved oracle, with retirement still open. Independent review, broad lint, strict/full generated IR and Go propagation remain the source-batch boundary obligations. Mini checked-in payload drift remains explicitly held; no regeneration occurred.

### Production census

Physical Scala lines, excluding tests, testdata and project.scala; new generic infrastructure is included.

| Module | After .2 | After .3 / task base | Final .4 checkpoint |
| --- | ---: | ---: | ---: |
| export | 12,238 | 12,426 | 12,541 |
| framework/lint | 2,995 | 2,995 | 3,149 |
| inspect | 95 | 107 | 184 |
| Total | 15,328 | 15,528 | 15,874 |

The three-module census grew by 346 since this task base and by 546 since .2. Its pre-.1 census was 15,225, so Part A adds 649 in these modules. This is not net shrink. Supplemental framework/check/lint is 892 lines (788 at task base); exporter Syntax is 1,163, generic Expansion 508, DSL Syntax 1,001. Formatting expansion is included in the physical counts.
## Evidence
- Commits: 6aab636863122713e8e8f4c11114873b740ab473, f406f437679e00650562dfef0304dd6e2dd1e977
- Tests: BASELINE GREEN: timeout 600s mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.SugarFixtures (1 test), GREEN: DefinitionExpansionTest LiftableSubsetFixtures SugarFixtures; fn-141.4-focused21.log (23 tests), GREEN: framework.check.lint.SyntaxRuleSuite; fn-141.4-syntax-green4.log (14 tests), GREEN: framework.dsl.SugarNativeTest; fn-141.4-native-final.log (5 tests), FAILED: timeout 1200s make model-check-changed BASE=36de9c9081; exit 2, 590.8s; .flow/tmp/checks/20261013-073035/record.md; 3 samples, mini equivalence and 144 Model tests passed; lifter 311 tests/1 failure; checks suites not reached, GREEN: timeout 300s mise exec -- scala-cli run --server=false --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.DeclarationSugarFixtures framework.export.DefinitionExpansionTest framework.lint.LiftableSubsetFixtures framework.export.SugarFixtures framework.lint.ConventionsFixtures framework.export.Fixtures; exit 0, 121 tests, 175.761s; .flow/tmp/fn-141.4-partial-final-focused.log, GREEN: selected Scala formatting and git diff --check, GREEN: timeout 1200s make model-check-changed BASE=36de9c9081; exit 0, 698.8s; .flow/tmp/checks/20261013-081114/record.md; 610 tests (144 models, 311 lifter, 155 checks), 3 samples and mini step-table equivalence
- PRs: