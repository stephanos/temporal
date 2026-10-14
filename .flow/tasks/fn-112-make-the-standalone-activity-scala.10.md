---
satisfies: [R1, R16, R17, R18, R19, R20]
---
# fn-112-make-the-standalone-activity-scala.10 Close fn112 with refusal coverage, source metrics and full frozen-output gates

Touches: [model/lifter/test/**, model/lifter/testdata/**, model/gate/**, model/temporal/standaloneactivity/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, .flow/tmp/fn112-10/**]

## Description
Audit every added construct once, finish docs/metrics and run the complete closure gates against the original baseline.

**Size:** M
**Files:** lifter/gate fixtures and docs; standaloneactivity final sources; .flow/tmp/fn112-10 evidence.

### Approach
- Complete the R16 matrix, accepting compiler refusals only for forms that cannot produce TASTy and requiring located lifter refusals for every reachable misuse.
- Run the original-baseline comparison across all IR/fixtures/manifests/Cases and ordinary Go admission, then the full model gate, lint-model, Umpire Go tests, lint-code-fast and installed export checks once.
- Recount lines/literals with task 1's method, classify every retained literal and reduce the feature to R17/R18 limits. Report both standalone-only and combined standalone-plus-queue metrics. Verify all authored totals and shared queue entity reuse with the precise R1 metadata deltas.
- Update README, SEMANTICS and module ownership only for final syntax and unchanged meaning; preserve fn114/fn118/fn120/fn119 boundaries.

## Acceptance
- [ ] Every R2/R3/R4/R5/R6/R7/R9 construct, including each claim pattern (`once/keeps`, `never/from`, `stays/unless`) and the function-argument binding, has positive and invalid coverage at the correct compiler/lifter layer with located diagnostics.
- [ ] All six original-baseline IRs, positive fixtures, manifests, rejects, IDs/tables/fingerprints/answers and Case bytes pass the strict equivalence harness under only the exact R1 metadata deltas.
- [ ] Full model gate, lint-model, complete tagged Umpire tooling tests, lint-code-fast, Cases/fixtures/canary and installed exports pass with exact commands/versions/times recorded.
- [ ] Standalone activity is at most 1,600 lines and 60 literals; every retained literal belongs to an allowed category and docs match the final surface.
- [ ] All R1-R20 criteria have direct evidence, including total arithmetic/diagnostics and independent shared-queue reuse. Only fn-120's named-choice foundation was brought forward; fn-118 hint-driven waiting, fn-120's strict refusal/tools and fn-122's capabilities remain in their later phases; the done summary says fn-122.1 and fn-114.1 may start.
## Done summary
Closed fn-112 at the targets with behavior frozen. The standalone activity Model went from 1,994 lines and 98 literals to **1,567 lines and 54 string literals**; the limits are 1,600 and 60. Spec start (fn-112.1) was 2,830 / 462. Combined figures, so extraction is not counted as simplification:
- standalone + taskqueue: 1,984 / 77, down from 2,412 / 123
- standalone + taskqueue + kit: 2,275 / 103, down from 2,700 / 149

Across every regeneration, model/ir, model/cases and the lifter's expected files changed only in positions. `original.json` is untouched. All gates pass (evidence.md).

Commits: 1fe1fdca32, 939fd857e0, 28675ee3e0, 5ae7843f03, 560268aab9, e676bb66e6, 95980fbd4f, d9166cf7b8, 8e48499fd5, 5e8a02ee49.

**What changed**
- **Compaction** (opus sub-agents in worktrees, one per surface):
  - enum cases imported in the vocabulary objects;
  - the dead `productWithoutControls` deleted (unused outside model0/);
  - shared run expectations (`satisfied`, `inconclusive`, `explanationsDisagree`, `neverEvaluated`);
  - Scenarios and Properties in shared defs named after their local vals (an existing lifter behavior), removing 19 name literals;
  - kit evidence-field helpers `attemptField`, `deliveryField` and `activityRunField`, shared by the kit's `delivered` and the feature's `committed`;
  - a shorter `committed` and `attemptFailure`;
  - comments trimmed. Rule reasons, grounding references, the explicit `disabled` reasons and `retryCompletes`' saturation are kept.
- **New lifted constructs.** Each has a lifting fixture and a located refusal, is documented in model/README.md, and is core:
  - an unnamed Query with no val is named `<machine>.<scenario>.<property>`. This replaces the task-2 refusal, per R7's "never refuses a Query for having no val";
  - `.sync(a, b)` is named after its first member's action;
  - `start()` means every input at its first value (fn-112.6's deferred item);
  - `MonitorExpectation(monitor, …)` takes the monitor by value. It is refused if no val declares it or the Query's machine does not watch it;
  - `Party()`, `Entity(key = …)` and `Observation(on = …, read = …)` are named after their val.
  - All of these are adopted in the feature, and the first two also in taskqueue's `any` Query and fault party.
- **R16 coverage gaps found by the read-only audit, now filled:**
  - `synced`/`own` with a selector that names no member, or a nested field;
  - an unnamed Scenario;
  - a computed Family, which now gets a located message;
  - an evidence `records` field naming a foreign fact;
  - a line in a poll scope;
  - crossed-sugar compiler refusals (`implies`, `records`, `accept`, `stay`) in crossed/Sugar.scala.
  - The README's core/sugar split now lists the derivations, `given Family`, `UpTo`, `Declares[S]`, `.total`, `statusTable` and `Actuator`.

**Retained literals.** All 54 fall into R18's three kinds (evidence.md):
- prose (7);
- ids no val supplies (23): Family roots, DefinitionScope owners, `Delivery`, the `statusTimedOut` evidence lines, the `admit`/`settle` sync names, realization ids;
- names stated because no val supplies them (24): the 16 Query names computed in shared defs that are not `<machine>.<scenario>.<property>`, `competingTimers.*`, the three shared-claim names, and `heldStaleDelivery`'s Scenario. That Scenario's name equals the root Query val, and renaming the root would change the IR source and the fn-115 golden config.

**Decisions I made on my own:**
- the Query default-name rule;
- `start()` is core, because Scala does not fall back to an extension when a member's `using` clause fails;
- taskqueue adopts the two defaults that apply to it;
- the kit gains three field helpers;
- I did not rename `MonitorExpectation.name` (P3). It mirrors the IR field the generic emitter fills by name, and follows task 9's `String | X` precedent.

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with two P3s: one fixed (5e8a02ee49), one rejected as above. Round 2 was SHIP.

**Deferred P3/FYI:**
- No refusal fixture for a machine val not built by `machine {}` (audit G4).
- `Action.apply()` on an empty domain throws if Scala evaluates it. The lifter refuses that case at its line.
- The Family-literal check covers `Family(...)` only.
- `release` in `committed` cannot take a default, because the lifter loses a command's val name through a default getter.
- The four control rpcs repeat namespace/activityId, since there is no cross-request helper.

**Scope:** only fn-120's named-choice foundation (fn-120.1) was brought forward. fn-118 hint-driven waiting, fn-120's strict refusal and tools, and fn-122's capabilities remain in their later phases. With the structural Case freeze verified, **fn-122.1 and fn-114.1 may start**.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1fe1fdca32, 939fd857e0, 28675ee3e0, 5ae7843f03, 560268aab9, e676bb66e6, 95980fbd4f, d9166cf7b8, 8e48499fd5, 5e8a02ee49
- Tests: CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, 88-124 s per run; model/ir, model/cases, lifter expected change only in positions), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens|MigrationProjection|IRInventory' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0, 127 s), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, 88 s; after review fix 77 s), mise exec -- make lint-model (exit 0), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 242 s, 46 packages), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main mise exec -- make lint-code-fast (exit 0, 21 s), mise exec -- make umpire-check-cases umpire-check-fixtures canary-check-case (exit 0, 35 s), mise exec -- scala-cli run model/metrics -- model/temporal/standaloneactivity model/temporal/taskqueue model/temporal/realize (standalone 1567 lines / 54 literals), git diff --check 9e64f90727 HEAD (exit 0)
- PRs: