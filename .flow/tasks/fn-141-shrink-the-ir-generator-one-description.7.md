---
satisfies: [R5]
---
# fn-141-shrink-the-ir-generator-one-description.7 Realization factories build complete values

## Description
The realization DSL discards part of what an author writes, because the lifter reads it from source. Make every factory keep everything, so the values can be exported.

**Size:** M
**Files:** `model/framework/realize/{Realize,Typed,Scripts,Exploration,Syntax}.scala`, `model/temporal/realize/*.scala`, focused munit tests beside their owners, and the existing migration README for a short recipe if author syntax changes.
**Touches:** model/framework/realize/**, model/temporal/realize/**, model/temporal/migrate/README.md, model/temporal/check/Coverage.scala
**Scope:** framework and shared Temporal kit values, exercised by `activity_mini` and focused fixtures. Real features are not edited, lifted or regenerated. This is the prerequisite for fn-174.2, whose DSL types construct their own messages; fn-141.5 and .8–.14 are superseded. The current lifter remains unchanged in this task.

### Approach
- A request scope (`rpc { … }`, `readUntil { … }`, `withFields { … }`) keeps its assignments; today the block runs and its result is dropped.
- An evidence value keeps its operation and fields. `Condition.equal`, `greater`, `not` and `all` keep their operator; today two pairs build identical values.
- `Field(_.a.b)` records its path, and a factory that takes a protobuf type argument records the message name, as `Action.schema` already does through the companion.
- Retain alias targets and action-valued `onPath` inputs without unsafe casts; preserve all factory arguments needed by constructed export. Names/family may remain deferred to fn-174.2's reflection context, but identify those deferred identity fields explicitly. Do not invent authored identities or a span-lookup seam from superseded fn-141.5.
- The lifter still reads realizations from source in this task. Nothing it reads may change.
- Register new focused realization suites in `model/temporal/check/Coverage.scala`; this exception is test-coverage metadata only, not a gate behavior change. The registry refuses unregistered suites.

### Investigation targets
**Required:**
- `model/framework/realize/Scripts.scala` (the scope helpers), `Typed.scala` (`Field`, `Condition`), `Realize.scala` (`Evidence`, `Observed`, `Proto`, `Command`, `family`), `Syntax.scala` and `Exploration.scala`
- `model/framework/export/Realizations.scala`: for each construct, what it reads from the tree that the value lacks; inspect only, no exporter rewrite here
- `model/temporal/realize/` and `model/temporal/features/activity_mini/realization/MiniActivityRun.scala`

### Verification
- Focused construction tests for scoped assignments, nested protobuf scopes, field paths/message descriptors, evidence operands, distinct condition operators, aliases and action-valued path selectors. Start with the affected suites and fixture tests, not the broad exporter suite.
- At task end run `make model-check-changed BASE=<committed predecessor>` once; cite the preceding task's record and mini step-table equivalence. Existing lifter outputs remain unchanged by retained runtime metadata; run only affected exporter fixtures while iterating.
- Formatting of edited Scala files. Full broad Scala gate and independent review remain deferred to the source-batch boundary; Go, regeneration, real-feature migration and lint/manual acceptance stay held.

## Acceptance
- [ ] A test constructs one value per factory and reads every argument back; it fails naming the factory and the argument.
- [ ] No factory of `model/framework/realize` or the shared kit drops an argument; deferred reflection identity fields are explicitly accounted for.
- [ ] Focused tests, formatting and the scoped gate with mini step-table equivalence pass. The current lifter's affected fixture IR remains unchanged; fn-174's oracle and broad batch gates are not claimed by this task.

## Done summary
# fn-141.7 realization values checkpoint

Source commit: `8ad875931a` (`662380279a..8ad875931a`). Implementation and the explicitly memory-constrained task-end gate are verified; Flow closure follows this receipt.

### Implementation and completeness inventory

- Request/proto scopes now collect ordered typed assignments, nested literals and response reads per invocation. `withFields` appends into a copy with a retained shared-name relation; `extended` appends into an independently named copy. No IR body/native evaluator was added.
- Core operands, protobuf literals, recorded reads/events, evidence wrappers, commands/scripts, status/exploration and kit/behavior/section helper arguments were audited. Existing retained fields remain their sole owners; unused empty inner `Evidence.operation`/`fields` holders were removed instead of duplicating the typed wrapper metadata. Aliases retain their target, action paths retain the typed union without the previous unsafe cast, and condition operators are explicit/distinct with ordered operands.
- `Field` retains its original callable selector plus descriptor-derived path, root message and selected message. Direct nested, optional, oneof and repeated selections are pinned. Existing shared `HistoryKind.apply`/`Input.sets` capture through inline propagation; original functions and exactly-once factory evaluation are preserved. Generic roots use threaded standard ScalaPB companion evidence.
- Meaningful protobuf type identities are retained by Field (root/projection), Proto/typed proto, Observed, projected origin, RPC (request/response), recorded reads (request/response/projection), run-event root and even empty message sequences. Scoped RPC retains its original authored argument-list shape, with inline name capture and a runtime scope helper. External-package expansion and ordinary generic helpers are tested.
- StatusFamily retains machine/call base/method/info/operation/status. HistoryEvidence's returned record retains the authored operation key. Deadline bindings retain action/call/value/unset/ordered field bindings and each binding's selector/root identity. Empty `deadlines[M]` has no semantic effect from M: exporter `Realizations.scala:1499` extracts M only for the field selectors (`:1555`); the empty fields branch (`:1585`) selects mask zero and never calls `deadlineSet` (`:1599`). Tests distinguish two empty M instantiations while retaining the actual call's message identity. This evidence-backed phantom exception is conductor-approved.
- Only the two new construction-suite registrations changed in Coverage.scala. Migration README describes companion evidence for generic helpers and the precise unsupported-path boundary. No exporter/lifter/IR schema/model-feature/generated payload changes were made; the owner's unrelated Lift.scala header remains uncommitted by this task.

Audit surface/readback coverage: framework `Realize.scala` (recorded/evidence/operand/proto/instruction/activation/command/options and expectations), `Typed.scala` (fields/typed wrappers/conditions), `Scripts.scala` (scope/copy/alias/script/status factories), `Syntax.scala` (request/literal/nested scopes and read targets), `Exploration.scala` and `Elapsed.scala` (already retaining their authored options/order); Temporal `Realize.scala`, `Modules.scala`, `Kit.scala`, `Behavior.scala` and `Objects.scala` (history/status/deadlines, literal/instruction/evidence helpers, behavior/cause/visibility/bounds, and realization section tuples). Already-complete wrappers/case classes were preserved rather than mechanically rewritten. Tests name each factory family and read the constructed arguments directly.

### Explicit fn-174.2 obligations

Declaration/member/fact names, realization names, command IDs, family/package-relative evidence/source/projection/producer identities still require the approved source-constructed conversion context. Explicit names and object relations are retained; no guessed IDs or superseded span lookup were introduced. Opaque non-inline selector functions remain callable, with root identity where evidence exists, but have no captured path; fn-174.2 must explicitly reject unsupported capture rather than invent a path. Existing indirect mini/shared selectors have exact captures now. Scoped poll reads are retained even where the present source exporter refuses them; the future conversion must handle or explicitly refuse those forms. Deadline expansion and all IR conversions are deferred.

### Test-first evidence

All logs below are preserved under `.flow/tmp/`:

- `fn1417-scopes-red.log`: four actual empty/default scope failures; `fn1417-scopes-green.log`: four passing regressions.
- `fn1417-metadata-red.log`, `fn1417-kit-red.log`, `fn1417-extra-metadata-red.log`, `fn1417-type-identity-red.log`: readback APIs absent before implementation.
- `fn1417-generic-root-red.log`: ordinary generic helper lost the descriptor path despite threaded companion evidence; corrected by companion lookup.
- `fn1417-regression-red.log`: 20 observed runtime regression failures for retained metadata; `fn1417-regression-scopes-red.log`: 13 failures for nested values, reads, deadline bindings/key, aliases/action paths and operators. Only task-owned implementation sensitivity mutations were used, then fully restored.
- `fn1417-final-construction.log`: 37 passing real construction tests (18 framework, 19 Temporal), including literal expected data, no source-grep/mock tests.
- `fn1417-final-MiniRealizationFixtures.log`: 8 passing unchanged exporter fixtures; `fn1417-final-TypedDeclarationsTest.log`: 1 passing compiler-refusal regression suite; `fn1417-final-InputDeadlinePresetsTest.log`: 4 passing fixtures.
- `fn1417-final-package.log`: mini source package succeeds. `fn1417-format.log` / `fn1417-format-check.log`: edited Scala formatted/check passes; final `git diff --check` passes.

### Final gate and recovery

Preceding checkpoint reused by owner request: `.flow/tmp/checks/20261013-165141/record.md` (988.6s, six samples, mini equivalence, 311 lifter / 166 check tests; source c846c6b40e and subsequent fixture-only checkpoint). No redundant broad baseline was run.

The first unchanged task-end `make model-check-changed BASE=662380279a` failed before computational stages, after 34.1s: `.flow/tmp/checks/20261013-181558/record.md` and `fn1417-changed-check.log`. Scalafix found an ignored prior fn1725 `guardFinalChecks` generated snippet without SemanticDB. Read-only inspection confirmed exact snippet content, older mtime, ignore rule and no live consumer. Only `model/framework/check/.scala-build/check_0521aabf39-d5b4a8f27c` was recoverably moved, with conductor approval, into `.flow/tmp/fn1417-stale-guard-cache.JRZQl2/`. No source, gate or lint-selection changes. Unchanged `make lint-model-check` passed (`fn1417-cache-recovery-lint.log`). The identical final gate was rerun against frozen source; record `.flow/tmp/checks/20261013-182222/record.md`, output `fn1417-changed-check-recovered.log`.

That recovered gate also failed (127.3s), after successful compilation/export, when the kernel killed `check-samples` with exit 137. `fn1417-sample-oom-kernel.log` records global OOM and the checker at 10,203,648kB anonymous RSS; process/memory snapshots are preserved in `fn1417-sample-oom-processes.log` and `fn1417-sample-oom-memory.log`. No failing verdict was changed. The conductor approved one bounded recovery: `GOMEMLIMIT=6GiB make model-check-changed BASE=662380279a`, started 01:28:54 UTC, output `fn1417-changed-check-memory-bounded.log`, record `.flow/tmp/checks/20261013-182854/record.md`. This is an explicit runtime Go soft memory budget, not a source/worker-count/scope/verdict change or a default-memory pass. `/proc` confirmed the checker itself inherited the budget; observed VmHWM was 6,740,016kB, with snapshots in `fn1417-memory-bounded-checker-resource.log`. No unrelated process was touched.

The bounded gate terminated exit 0, **passed in 825.5s**: six samples (110.0s), exact `activity_mini` step-table equivalence (15.8s), 181 models + 311 lifter + 166 check + 15 migrate tests = **673 tests**. The record explicitly holds checked-in generated IR regeneration until the sweep; no generated files were changed. Batch-end Models/lifter Scalafix and Go/Case lowering remain not run, exactly as the gate declares. Task source was frozen before the gate and remained unchanged afterward; final whitespace/worktree checks pass for all eleven task source paths.

Independent review and broad fullScala remain deferred by the owner to the source-batch boundary. This task is not a SHIP verdict.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8ad875931ad06d2bd15c0a7b20a1567475aed1b6
- Tests: Final focused construction: mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/framework/dsl model/framework/realize model/project.scala model/temporal/realize --require-tests --test-only '*RealizationConstructionTest'; exit0,37tests (18framework,19Temporal); .flow/tmp/fn1417-final-construction.log, Affected unchanged exporter fixtures: MiniRealizationFixtures 8pass, TypedDeclarationsTest 1pass, InputDeadlinePresetsTest 4pass; .flow/tmp/fn1417-final-MiniRealizationFixtures.log, .flow/tmp/fn1417-final-TypedDeclarationsTest.log, .flow/tmp/fn1417-final-InputDeadlinePresetsTest.log, make model/build/model-scala.jar FEATURES=activity_mini: exit0; .flow/tmp/fn1417-final-package.log, Edited Scala formatting/check and git diff --check passed; .flow/tmp/fn1417-format.log and .flow/tmp/fn1417-format-check.log, Observed test-first red->green: initial scopes4fail; generic-root real readback None failure; retained regression sensitivity20fail plus13scope/kit failures; logs .flow/tmp/fn1417-scopes-red.log, fn1417-scopes-green.log, fn1417-generic-root-red.log, fn1417-regression-red.log, fn1417-regression-scopes-red.log, fn1417-type-identity-red.log; all own sensitivity mutations restored before final verification, Preceding owner-requested source checkpoint reused: .flow/tmp/checks/20261013-165141/record.md,988.6s,6samples,mini equivalence,311lifter/166check tests on c846c6b40e; later fixture-only af0752c97e guarded166exit0, First task-end gate make model-check-changed BASE=662380279a failed34.1s before computations: stale ignored fn1725 generated guard snippet missing SemanticDB; .flow/tmp/checks/20261013-181558/record.md retained. Exact generated profile recoverably moved with conductor approval to .flow/tmp/fn1417-stale-guard-cache.JRZQl2; unchanged make lint-model-check then exit0 (.flow/tmp/fn1417-cache-recovery-lint.log), Same recovered gate failed127.3s at kernel-killed sample checker exit137 after compilation/export: .flow/tmp/checks/20261013-182222/record.md, fn1417-changed-check-recovered.log; kernel globalOOM checker anon-rss10203648kB recorded in fn1417-sample-oom-kernel.log; failing verdict retained, Conductor-approved bounded recovery GOMEMLIMIT=6GiB make model-check-changed BASE=662380279a: exit0,passed825.5s,6samples,exact mini step-table equivalence,181models+311lifter+166check+15migrate=673tests; .flow/tmp/checks/20261013-182854/record.md and fn1417-changed-check-memory-bounded.log. Explicitly constrained runtime, not default-memory pass/capacity fix; checker environment and observed VmHWM6740016kB recorded in fn1417-memory-bounded-checker-resource.log. No scope/worker/verdict/source changes, no source delta after gate; generated regeneration and batch-end lint/Go/Case checks held as record states
- PRs: