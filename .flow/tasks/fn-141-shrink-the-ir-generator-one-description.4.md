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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
