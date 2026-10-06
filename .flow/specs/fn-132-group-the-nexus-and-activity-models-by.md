# Group the Nexus and activity Models by kind: workflow and standalone under one feature

> HTML render lens: local `.flow/artifacts/fn-132-group-the-nexus-and-activity-models-by/spec.html` — open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

The server implements each of Nexus operations and activities as **one CHASM component** that serves both forms, the one a workflow schedules and the standalone one a caller starts directly:

- `chasm/lib/nexusoperation/operation.go`: one `Operation` with one set of transitions (Scheduled, Started, Succeeded, Failed, Canceled, Terminated, TimedOut, backoff). `Store chasm.ParentPtr[OperationStore]` points back to the workflow, or is nil for a standalone operation.
- `chasm/lib/activity/activity.go`: "can be either standalone activity or one embedded within a workflow", with the same `Store` parent pointer and fields marked `// Standalone only`.

The Models do not show this. `features/nexuscaller` (workflow-scheduled) and `features/nexusoperation` (standalone) are two unrelated Models that repeat `Reply`, `Resolution`, the handler's actions and the terminal-phase logic, and each leaves out what the other has:
- `nexuscaller` has no cancellation (fn-79).
- `nexusoperation` has no retries, no deadlines and no `handlerError(retryable)`.

These are gaps in the Models, not differences in the server. The activity has only its standalone Model (`features/standaloneactivity`); the workflow-scheduled activity is deferred (fn-119.3/.4, `.plans/ACTIVITY_MODEL_COMPARISON.md` P3-11).

The name "Nexus caller" no longer separates the two Nexus Models either: the standalone Model has a `caller` actor too. What separates them is who starts the operation, a workflow or a caller's RPC.

This spec groups each kind under `features/<kind>/`, a general part at the top and `workflow/` and `standalone/` below it, so that:
- a reader finds both forms of a kind in one place, named by what tells them apart;
- what the two forms share is written once, as the server writes it once;
- the deferred workflow-scheduled activity has its place before it is written.

It serves model authors (fn-126's principle: lighten the author's cognitive load) and whoever later closes the cancellation and retry gaps, who then builds on one shared product instead of two drifting ones.

## Architecture & Data Models
<!-- scope: technical -->

Names below are those after fn-126 closes (`NexusSystem`, `ActivitySystem`, `handler.reply`, `worker.poll`, `worker.respond`, `TrustingCaller`, the `product/` and `system/` level folders, IDs as fully qualified Scala names). If fn-126 lands other names, read these as theirs.

**Target layout.**

```
features/nexus/
  Nexus.scala                 types and signature both forms share (Reply, Resolution, handler actor); no machine
  product/Product.scala       NexusProduct: what an operation does (Part B)
  workflow/                   was features/nexuscaller
    Workflow.scala            the workflow form's types, signature (caller.schedule, deadlines, network), exports
    system/System.scala       NexusSystem, the handler's worker, the composition
    system/ClosePolicy.scala  close and reset designs
    system/TrustingCaller.scala  the trusting caller as fn-126 left it
    Realization.scala
  standalone/                 was features/nexusoperation
    Standalone.scala          the standalone form's signature (caller.start, requestCancel, terminate), exports
    system/System.scala       the standalone operation's machine (refines NexusProduct after Part B)
    Realization.scala

features/activity/
  Activity.scala              what both forms share: Timeout, TimeoutType, AttemptResult, the worker's poll and respond, timers, deadline
  standalone/                 was features/standaloneactivity
    Standalone.scala          the standalone form's types and signature (client.start, client.control, Control), exports
    product/Product.scala     ActivityProduct (stays here: it is what DescribeActivityExecution reports)
    system/System.scala, Record.scala, WithTaskQueue.scala, Realization.scala
  (workflow/ arrives with fn-119.3/.4 or P3-11, not in this spec)
```

**Part A. Move and rename (no meaning change).**
- `features/nexuscaller` → `features/nexus/workflow`, `features/nexusoperation` → `features/nexus/standalone`, `features/standaloneactivity` → `features/activity/standalone`, as folders and packages.
- Feature files are named after their folder (`Workflow.scala`, `Standalone.scala`).
- Each Definition ID changes only by its package path (decision 23 of fn-126). IR file names (`nexus-caller`, `nexus-operation`, `activity`, …) are renamed to match (`nexus-workflow`, `nexus-standalone`, `activity-standalone`, …) in the same regeneration; the first task fixes the exact names.

**Part B. One Nexus product.**
- `NexusProduct` moves from the workflow form to `features/nexus/product/Product.scala`. `Reply`, `Resolution` and the handler actor (`reply`, `complete`) move to `features/nexus/Nexus.scala`.
- `NexusSystem` (workflow) refines it as today. The standalone machine gains `object refinement extends Refinement(NexusProduct)`: its `unstarted` reads as `scheduled`, `terminated` needs a product phase (Part B adds one, or the refinement hides it; the task decides). Part A preserves the current standalone Nexus single-level machine; Part B places it and its local vocabulary in the target System level, with the canonical `NexusSystem`/`system.{Phase,State,Fact}` declaration identity map recorded before regeneration. This source restructuring does not change its transition behavior.
- **Facts map by their member names today.** Before sharing the product, the fixture proof aligns the standalone status facts with the abstract product facts. Prefer a finite, explicit same-name fact-member ledger over new projection machinery: `statusStarted` → `nexusOperationStarted`, `statusSucceeded` → `nexusOperationCompleted`, `statusFailed` → `nexusOperationFailed`, `statusCanceled` → `nexusOperationCanceled`; scheduled/cancel-request/termination facts are classified explicitly as product-visible or System-only by the proved state map. Preserve the history/Describe read wiring and every realization evidence contract, translating only declared fact identities. Hiding an observed state change is not a substitute for its missing product carrier.
- **Shared product dependency closure.** Move the existing signature declarations and input types the product uses (including its network fault and timeout references) into the kind core, or remove a reference only if it contributes no product semantics and exact table proof establishes that. Keep workflow-only retry/backoff behavior in the workflow form. The kind Product and general signature may not depend on either form's declarations; the finite move ledger names the actual closure.
- `Reply` takes the union of both forms' members (`handlerError(retryable)` included). A form whose handler cannot answer a member disables it in its rules.
- Properties written on the product (`terminalIsFinal` and those that follow) are carried to both forms by their refinements.

**Part C. General activity declarations.**
- `Timeout`, `TimeoutType`, `AttemptResult`, the worker's `poll` and `respond` on an activity, `timers` and `deadline` move into `features/activity/Activity.scala`.
- `ActivityProduct`, `ActivitySystem`, `Control`, `client` and the `activity` entity stay in `standalone/`: each is the standalone form's (Describe statuses, `terminated`, the control RPCs). A second form decides what of them is general. Level-owned vocabulary remains `product.{Phase,State,Fact}` and `system.{Phase,State,Fact}`; `TimeoutType` starts in the System level and its move is included in Part C's identity ledger.

**Layout lint and docs.** fn-126's structure lint (R20) learns the kind level: a `features/<kind>/` folder holds a general feature file named after the kind, an optional `product/` and the forms as subfolders; a form's `system/` may refine a product in its kind's `product/`. `model/README.md` ("Writing a Model", "Where things are") and `.plans/UMPIRE_MODULES.md` describe the kind level.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Exact equivalence by part.** Part A remains strict: only its explicit package/path/export-file map changes; reader tables, every Query answer, verdict, fingerprint and Case byte are identical after that map. Part C likewise admits only its declared extraction identities/paths. Part B must additionally account for R2's finite shared catalogs and added refinement: enumerate the Reply union's new disabled input classes, required product outcome members, fact-member alignment, canonical standalone System-level identities and any chosen product termination state/carrier. Preserve all previously enabled source transitions, Query answers, verdicts and realization reads under the declared identity map. New input classes unavailable to a form must have no enabled source transitions. Product changes are exactly those needed by the recorded refinement decision, not arbitrary behavior changes. Catalog expansion changes canonical fingerprints and the Case fields containing them; verify those values by fresh derivation from the declared augmented baseline and exact full-artifact comparison, not by stripping fingerprints or allowing arbitrary Case bytes. Stop on any unexplained state, transition, Query, verdict, evidence or Case delta.
- **One entity for shared actions.** The handler's actions and the worker's activity actions are bound with `.on(entity)`, and fn-126 decision 28 infers a machine's entity from its actions. The two Nexus forms bind different identities; activity currently has only its standalone form. Task 4 proves how one kind-level action serves the form identities. Candidate approaches are identity relative to a typed parent scope (the `stamp` prototype's `parent/Type[id]`, `Scope[*Parent]`), one shared entity whose key each realization binds, or form actions mapped by refinement to a shared product action. Select from executable fixtures, not an unproven preference. A types-only extraction does not satisfy R2/R3 and cannot close this spec; if existing DSL primitives are insufficient, plan the smallest necessary binding seam rather than silently omitting shared actions.
- **Outcomes differ.** The workflow form rejects a late completion as `notFound`; the standalone form rejects a control of a closed operation as `alreadyCompleted` (FailedPrecondition, `operation.go`). The shared product catalog must contain both members: refinement admission checks all source outcomes before visibility. `visibleOutcomes` can control whether an otherwise admitted outcome is observed on a stutter; it cannot hide a missing catalog member. Preserve both forms' current rejection/repeat behavior and prove the catalog-versus-visibility distinction with a checker-level fixture.
- **Realizations stay per form.** They differ in start (workflow command vs RPC), evidence (history events vs Describe status) and settings (`nexusoperation.enableStandalone`). Nothing in a `Realization.scala` moves to the kind level.
- **fn-128 and fn-129 work on the new paths.** They are planned against `standaloneactivity`; their specs' paths are updated in this spec's close.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `features/nexus/{workflow,standalone}` and `features/activity/standalone` exist with the layout above; `features/nexuscaller`, `features/nexusoperation` and `features/standaloneactivity` do not. Errors: the projection of Edge Cases differs beyond IDs, paths and IR file names; any reference (Go `tools/umpire`, Makefile targets, Case trees, canary, docs) to an old path or IR file name remains (a repository grep finds none outside history and archived plans).
- **R2:** `NexusProduct`, `Reply`, `Resolution` and the handler's actions are declared once under `features/nexus/`, both Nexus forms refine `NexusProduct`, and a Property on the product is checked on both forms. Errors: the gate refuses a refinement step that maps to no product step; a member of `Reply` that a form cannot answer is disabled in that form's rules, not deleted from the shared type.
- **R3:** `features/activity/Activity.scala` declares `Timeout`, `TimeoutType`, `AttemptResult`, the worker's activity actions, `timers` and `deadline`, and the standalone form reads them from there. Errors: no declaration of these remains in `standalone/`.
- **R4:** The structure lint accepts the kind level and refuses a form folder outside a kind folder, a machine in a kind's general file, and a second general file in a kind folder; one refusal fixture each. `model/README.md` and `.plans/UMPIRE_MODULES.md` show the kind level.
- **R5:** The entity and outcome decisions are recorded in the spec's Decision Context, each with what it changed.

## Boundaries
<!-- scope: business -->

- No workflow-scheduled activity Model; only its place (`features/activity/workflow/`, later).
- No new behaviour: cancellation in the workflow form (fn-79), retries and deadlines in the standalone Nexus form, and the activity's precision gaps (fn-128) stay out. This spec makes them land on a shared product.
- No change to realizations beyond paths, imports and IR file names.
- No shared activity product or system machine; it waits for a second form.

## Decision Context
<!-- scope: both -->

- **Kind folder, not `shared/`.** `shared/worker` and `shared/taskqueue` serve several kinds; the Nexus and activity cores serve one kind each, so they live in that kind's folder.
- **`workflow/` and `standalone/`, not "caller".** Both Nexus forms have a caller; who starts the operation is what differs. `standaloneactivity`'s name becomes the path `activity/standalone`.
- **Activity now, with one form.** Setting the structure up before the workflow form exists costs one path change now instead of a move of a grown Model later, and puts the general declarations where the second form will look.
- **After fn-126, not folded into it.** fn-126 is mid-flight and rewrites every ID in one batch. Folding this in would save one regeneration but widen an in-progress spec; a pure move after it is reviewable on its own.
- **Admission before moves.** The current structure reader groups only a flat feature level; moving first would fail generation. A narrow prerequisite admits the future kind/form shape through synthetic fixtures while retaining current flat/shared contracts. The installed Flow CLI has additive task dependency operations only, so the existing post-move lint/docs task keeps its dependencies instead of requiring a manual state edit.
- **Shared actions are required.** The old types-only fallback conflicted with R2/R3. The executable entity spike must choose a binding that preserves those requirements; under the milestone owner's autonomous direction, a necessary bounded seam is planned and verified, not replaced by weaker success criteria.
- Maintainability (plan review): duplication - none identified; structure - a kind-owned NexusProduct must not retain dependencies on workflow-owned network.fault or timers.timeout. Part B moves the product-used signature closure to the kind core, keeps form-only behavior local and proves the finite identity/catalog delta.

## Ordering
The validated DAG adds only three task edges: fn-132.1 and .2 depend on .8; fn-132.6 also depends on .2. Existing .3 dependencies on both moves remain. Every overlapping mutable Touches pair serializes, not only artifact writers: .1/.2, .2/.4, .3/.4 and .5/.6 cannot share a live writer; all tasks also update MILESTONES. The practical first batch is .8 → .1 → .2, then .3 → .4 → .5 → .6 → .7, each with one checkout owner. Read-only research/review may overlap only without mutating reviewed inputs.

- After fn-126 and fn-124 close: the current canonical level names, System realization placement and split Go reader are the baseline.
- Before fn-128 starts, so fn-128 and fn-129 are written against `features/activity/standalone`.
- Inside the spec: source-grouping prerequisite first; then the two Part A moves sequentially, each verified by projection; then the final kind-layout lint/docs task. The entity/outcome spike follows the Nexus move. Part B and Part C each retain their own regeneration and equivalence proof, and artifact-writing tasks never run concurrently.

## Verification

```bash
make umpire-check-model && make lint-model && make lint-code-fast
go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/...
make umpire-check-cases && make umpire-check-fixtures && make canary-check-case
```

Regenerate with `make umpire-gen-model`, then `make umpire-gen-cases umpire-gen-fixtures canary-gen-case`. Every proof uses an exact declared map/augmentation and independently derived expected output, never a golden re-capture. Tasks 1/2 admit only package paths and export file names. Task 6 admits its exact extraction map. Task 5 proves the finite Part B declaration/fact/catalog/refinement delta from Edge Cases, unchanged previously enabled transitions and existing Query/verdict/evidence results, disabled extra classes per unsupported form, fresh canonical fingerprints and exact generated Case bytes against that expectation. Do not reuse the old helper's lossy normalization as a substitute for these checks; adapt its current package imports and validate its coverage before baseline collection.

Follow MILESTONES verification scoping: reuse applicable fn-124 boundary evidence, run focused acceptance/failure checks at each checkpoint, and run the full model/Go/artifact/runtime/lint batch once after both Part A moves. The prerequisite and first move explicitly defer those broad gates to task 2; no focused run mints a full-gate receipt. Use `MODEL_GATE_ARGS=--skip-go-checks` when Go runs separately, the absolute shared heavy-suite lock, `-p 2 -timeout 30m`, JSON output with numeric exit and separate wall time, and read-only Go lint against the current fn-132 batch base. New relevant inputs invalidate affected evidence only. After Part B/C, run the remaining applicable full boundary once, preserving each part's exact equivalence proof. No live-case or backend run is claimed by compile-only/offline checks.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `features/nexus/{workflow,standalone}` and `features/activity/standalone` exist with the layout above; `features/nexuscaller`, `features/nexusoperation` and `features/standaloneactivity` do not. Errors: the projection of Edge Cases differs beyond IDs, paths and IR file names; any reference (Go `tools/umpire`, Makefile targets, Case trees, canary, docs) to an old path or IR file name remains (a repository grep finds none outside history and archived plans). | fn-132-group-the-nexus-and-activity-models-by.1, fn-132-group-the-nexus-and-activity-models-by.2, fn-132-group-the-nexus-and-activity-models-by.7 | — |
| R2 | `NexusProduct`, `Reply`, `Resolution` and the handler's actions are declared once under `features/nexus/`, both Nexus forms refine `NexusProduct`, and a Property on the product is checked on both forms. Errors: the gate refuses a refinement step that maps to no product step; a member of `Reply` that a form cannot answer is disabled in that form's rules, not deleted from the shared type. | fn-132-group-the-nexus-and-activity-models-by.5, fn-132-group-the-nexus-and-activity-models-by.7 | — |
| R3 | `features/activity/Activity.scala` declares `Timeout`, `TimeoutType`, `AttemptResult`, the worker's activity actions, `timers` and `deadline`, and the standalone form reads them from there. Errors: no declaration of these remains in `standalone/`. | fn-132-group-the-nexus-and-activity-models-by.6, fn-132-group-the-nexus-and-activity-models-by.7 | — |
| R4 | The structure lint accepts the kind level and refuses a form folder outside a kind folder, a machine in a kind's general file, and a second general file in a kind folder; one refusal fixture each. `model/README.md` and `.plans/UMPIRE_MODULES.md` show the kind level. | fn-132-group-the-nexus-and-activity-models-by.3, fn-132-group-the-nexus-and-activity-models-by.7, fn-132-group-the-nexus-and-activity-models-by.8 | — |
| R5 | The entity and outcome decisions are recorded in the spec's Decision Context, each with what it changed. | fn-132-group-the-nexus-and-activity-models-by.4, fn-132-group-the-nexus-and-activity-models-by.7 | — |

## Early proof point

Task fn-132-group-the-nexus-and-activity-models-by.8 proves nested admission against exact future shapes while unchanged production Models generate identical IR/Cases. If it fails, resolve the source-grouping/ownership design before moving either form; never bypass structure lint or recapture baselines.

## Planning evidence

SHORT planning researched current source grouping, cross-spec consumers, documentation and targeted project memory, then ran the flow-gap analysis. Source grouping and independent-level validation reuse the existing Structure implementation and fixture harness; no parallel validator or blanket exception is planned. Memory retrieval used BM25 because the judge had no key, then the fast scout refined targeted queries. The paired-level validator regression entry informed preservation of all existing negative guards. Web scouts were skipped by SHORT depth, not by a relevance guess; tracker projection is inactive.

stage: plan-review - ran [2026-10-06T09:32:28Z..2026-10-06T09:41:24Z], SHIP after two rounds (model: gpt-6.1-sol). Receipt: `.flow/tmp/plan-review-receipt-fn-132-group-the-nexus-and-activity-models-by.json`; same-family fresh read-only Codex review, same receipt/session resumed after fixes. All four introduced findings are fixed, with no unaddressed requirements; no runtime test results are claimed by planning.

Tasks: 8 total, seven M and one S. Dependency waves: .8; then .1/.2; then .3/.4; then .5/.6; then .7. These are DAG candidates only: all declared overlapping writers run sequentially in the practical order recorded under Ordering. Tracker sync: n/a (bridge inactive). The HTML lens is ignored/local-only, not a committed artifact or a browser-verified deployment.

## Parked unknowns
- Server-assigned identities (an activity in a workflow is named by its `scheduledEventId`) need a general form of what `Learned`/`correlated` do for one case today; `stamp` created a model under an alias and set its real id later (`SetID`). This matters when `activity/workflow/` is written.
- Whether workflow activities run on the CHASM `Activity` component today or on the older mutable-state path; it matters when `activity/workflow/` is written, not here.
