# Name the standalone activity's repeated patterns

> HTML render lens: local-only `.flow/artifacts/fn-155-name-the-standalone-activitys-repeated/spec.html` — open the ignored file locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

The standalone activity Model says the same thing many times. The clearest case is "the held attempt ends", which the System machine spells out in ten effects and four rule blocks. Readers cannot tell the essential differences from copy noise, and a change to one copy can silently miss the others. This spec gives those patterns names, keeps related behavior in one place, and makes rule guards and capability declarations read the same named predicates.

It runs after fn-151-split-standalone-activity-into-smaller (the Activity subject split), on its closed baseline, and before batch 2 (fn-140 → fn-123). fn-151 decides which file owns which model. This spec only changes how the logic inside those files is expressed.

Source: a review on 2026-10-09, refined by the planning scouts the same day.

## Architecture & Data Models
<!-- scope: technical -->

Every helper must be a form the lifter (irgen) accepts today. Planning confirmed the following.
- **Accepted:** `def` predicates in `states` used as rule guards, effect defs that call other lifted defs, and `on(a, b)` across two actor objects outside a `from` block. In the last case, binding order follows which action is named first.
- **Refused:** `because` text passed as a parameter (`.because` only lifts on steps written directly in the effect), an extra fact splatted after fixed facts, a `def` wrapping `deadlines[...]`, a `Seq` splat in `scenario.actions(...)`, and reading a `val` declared on an enum case.
- **Changes behavior:** `effect { }` blocks append the assigned status after recorded facts, cannot assign a computed phase, and carry no `because`.

Each section is changed only within these limits. Items that need a lifter or capability change are listed under Boundaries as deferred.

### A. The held attempt ends (System machine)

- **Collapse by landing phase.** Effects that differ only in their landing phase collapse through a named landing function. `pauseRequested`/`resetKeepingPause` land in `paused`; every other held phase lands in `scheduled`. The landing status is computed alongside the phase.
  - Effects with different facts or different `because` texts stay distinct. Today's texts are the retry reason, none, the nominal time window, and the deferred-reset reason, which leaves about six effects instead of ten.
  - The surviving effects keep existing names (`backOff`, `applyReset`, …) where possible, because Go lowering tests read them.
- **`resetSettles`.** Its two branches collapse through the same landing function.
- **Named guards.** `exhausted` and `endsTerminally` (cancel pending or exhausted) replace the inline retry-remaining and cancel-or-exhausted lambdas in the failure, By-ID failure and attempt-deadline rules.
- **Deadline guards read the capability's predicate.** Every deadline rule's guard reads the same `armed` predicate its `Deadline` capability declares, instead of restating it inline.

### B. Product uses its own `Recorded` form (Product machine)

Product effects with a single status and no extra fact or `because` use the existing `pause`/`retry`/`fail` effects or the `Recorded` setter. They do not hand-pair phase and status. Multi-fact steps such as `heartbeatExpires` keep the method form, because the block form would reorder facts.

### C. One reason for the reset-aware overrides

The six `overriding` calls that make the Retries/Deadline laws reset-aware share one named `because` value. The reset-aware law bodies stay activity-local in this spec.

### D. Smaller duplication in the machines

- **One "nothing set" state.** Fixed expected states are written as the machine's initial state with the differing fields replaced. They are not new copies of the 8-field state.
- **`resetDispatch` is called, not inlined.** The direct-reset Properties call the existing reset-dispatch predicate.
- **Shared rejection reasons.** Rejection reasons that cite the same server source line are defined once and cited by both the Product and System levels. Each level keeps its current text unless the identity mapping lists a change.
- **Worker and By-ID rule blocks.** They differ deliberately. By-ID adds `notFound` rejections and cites different sources. They stay separate, with a comment that says which rows must match.
- **By-ID failure examples.** `respondFailedByID` reuses the failure examples of the kind's `respondFailed` instead of repeating them.

### E. Dispatch models (Dispatch* files after fn-151)

- **HeldDispatch derives from the corrected design.** It is restricted to its four actions, rebound to the committed admission and declared to refine the Product (a restricted machine does not inherit a refinement), the pattern the faulty-control design uses. This happens only if the step tables of the hand-written and derived machines are proven equal.
- **One composition capability class.** One capability class serves both queue compositions. Their state types are unrelated case classes, so this needs a shared bound on the composed record, which changes the lifted types; if the lifter refuses it or the change cannot be mapped, the two classes stay.
- **Waiver reasons in their own section.** Waiver reasons move out of `object states` into a section named for what they are.

### F. Realization evidence builders

- **Run-event attempt records.** One builder replaces the three that share the delivery guard and attempt fields.
- **Describe reads.** The heartbeat reads reuse the single Describe-read builder, renamed to say it reads Describe.
- **Shared conditions and bases.** Status and run-state conditions, the worker activation, and the run-scoped Describe base are each defined once.

### Source provenance and artifact identity

The single-definition refactor keeps wait hints attributed to the current declaration that supplies their realized timer. The source-provenance map enumerates exactly eleven Case source-line fields from seven source sites, fixed from the sealed original inventory. Each entry identifies the Case, instruction, hint, realization and complete action identity, then independently binds its original and current coordinate to the corresponding Scala declaration and lifted ServerStep. The explicit heartbeat ServerStep within ResetAfterHeartbeat is a separate source identity from its enclosing realization. An arbitrary positive coordinate is insufficient.

After the separately approved identity mapping, restoring only those eleven mapped source-line values must recover the original Case bytes. Every other Case byte remains equal. Source paths, other coordinate fields, hint identities, guards, bounds, ordering, causal scopes, `confirms`, Program/Contract content, Query coverage, expectations and statuses remain protected. The proof checks every file in the frozen inventory, including the Cases that currently compare equal, and rejects missing or extra files and incomplete or conflicting map entries.

The manifest has one separate source-coordinate exception. Only the original start-to-close timeout Query's existing unsupported `attempts` refusal may update its diagnostic Position to the current declaration of the same source script. Fixed original Query and refusal identities select this field independently of candidate differences. The unchanged producer's original and current default Check and Find must reproduce the same located refusal before task credit. Standing, construct, identifier, owner, reason, source path and every other manifest byte remain identical. The original manifest archive stays immutable. The fresh manifest digest follows its ordinary derivation; this diagnostic exception creates no Case and authorizes no Case identity substitution. Restoring the eleven Case integer leaves and this one manifest diagnostic coordinate must recover all 31 original files byte for byte after the separately approved identity mapping.

Truthful current coordinates change the content-derived Case identity, CaseFingerprint, prepared AssessmentBinding.case and timeout Detail where their inputs change. Each resulting value must follow its existing production derivation. The map grants no arbitrary fingerprint or checksum substitutions. The proof must establish that the affected current IR and Cases have no declaration that consumes the changed timeout Detail semantically; otherwise this amendment's contract fails. Original Case+Run pairs retain their exact bytes and bindings. A new Case must reject a historical Run or assessment factory bound to another Case. Required current execution pins, bindings and receipts are refreshed under their existing contracts, with fresh recording and replay only where those contracts already require it.

The fixed eleven-line exception covers the isolated Realization builder refactor only. The final joined proof must halt on any nonallowlisted Case byte difference, including a coordinate introduced by another task. Such a difference requires independent evidence and an explicit separately reviewed contract decision before any credit. Constructed INCONCLUSIVE identity controls establish binding checks only; they provide no successful matching replay, live execution or semantic conformance credit. Existing live and replay requirements remain in force.

The manifest exception is separately enumerated and must be rederived on the joined candidate from its exact source-script declaration and lifted Script. Any other manifest change, missing or duplicate Query/refusal selector, stale or wrongly attributed coordinate, or changed refusal status or reason fails the complete inventory proof. A copied original manifest establishes archive retention only and supplies no fresh canonical-manifest credit.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Step-table equivalence is the behavior pin.** The step table records each step's outcome, state, facts in order and `because`, for every state and action class. The Pins test pins it for Product and System, but not effect names, so it is the semantic proof that collapsed effects behave as before. It also pins each machine's binding order, which a merged `on(...)` must keep. Machines the Pins test does not cover (the Dispatch designs, HeldDispatch, the compositions) are compared through the interpreter's tables built from the lifted IR, before and after.
- **The IR shows structure, not only names.** Effects are function definitions that rule arms call. Guards are calls to `states` definitions. A fixed state lifts as a `construct`, and `init.copy(...)` as a `copy`. So named guards, merged effects and derived states change IR structure even when behavior does not. The projection normalizes these forms: it inlines lifted `states` and effect definitions at their call sites, folds a `copy` over the initial `construct`, and drops definitions nothing references. Structural differences that survive normalization are listed in a structural-review section, separate from the identity mapping that downstream specs re-anchor to.
- **Many-to-one identities.** Collapsing effects maps several old effect identities to one new identity. The mapping records each such merge. Identities that remain stay distinct, so finite-search identity remains injective.
- **IR shape in Go tests.** Go tests read rule shapes (`respondFailed` stays a Match; lowering reads `effects$.backOff` and the start-to-close rule shape) and state-expression shapes (lowering reads `retryCompletes`'s expected state as a `construct`). They load the checked-in IR, so they change only at the one regeneration, and are updated there.
- **No added validation.** A behavior-neutral refactor adds no rejection, guard, or check that the baseline lacked.
- **New names.** New names pass the glossary gate, which rejects retired terms.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** System effects that differ only in landing phase collapse through one landing function. The failure, By-ID failure and attempt-deadline rules use named `exhausted`/`endsTerminally` guards. Every deadline rule reads the `armed` predicate its `Deadline` capability declares. Errors: a step whose facts, fact order, outcome, state or `because` differ from the baseline step table fails the R7 comparison. A held phase the landing function does not name is unreachable under the rule guards, which the step table confirms.
- **R2:** Product single-fact effects use the shared phase effects or the `Recorded` setter. Multi-fact and reasoned steps keep their baseline fact order and `because`. Errors: any fact-order change fails R7.
- **R3:** The six reset-aware `overriding` calls cite one shared reason value. Their emitted lint and metadata text stays byte-identical to the baseline. No error surface beyond the lift: a refused cross-object reference falls back to a local value, and the fallback is recorded in the mapping.
- **R4:** Fixed expected states derive from the initial state. The direct-reset Properties call the reset-dispatch predicate. Rejection reasons citing the same source are defined once, each level keeping its baseline text. Worker and By-ID blocks carry a must-match comment. By-ID failure reuses the kind's examples where the lifter accepts the reference; otherwise the examples stay repeated and the refusal is recorded. Errors: generated Case names or examples that change are listed in the mapping, or reverted.
- **R5:** HeldDispatch derives from the corrected design, restricted, rebound and refining the Product, if and only if its interpreter-built step table, refinement, monitors and evidence equal the hand-written machine's. Otherwise it stays hand-written, and the reason is recorded. One capability class, bounded by a shared view of the composed record, serves both queue compositions if the lifter accepts it and the type change is mapped; otherwise the two classes stay, and the reason is recorded. Waiver reasons leave `object states`. Errors: a derived-machine or state-type identity change is mapped. A step-table difference keeps HeldDispatch hand-written.
- **R6:** In the Realization file, the run-event attempt record, the Describe read, the status and run-state conditions, the worker activation and the run-scoped Describe base each have one definition. Lowered Cases for every realization stay byte-identical apart from mapped identities and the exactly enumerated current-source line fields allowed by Source provenance and artifact identity. The independent full-inventory proof binds every allowed Case line to the same original and current declaration and ServerStep; all other Case bytes remain identical. The manifest preserves its original refusal bytes except its single separately enumerated current-source diagnostic coordinate. `make umpire-check-cases` passes. Errors: a missing, extra, duplicate or mislabeled inventory/map entry, a stale or wrongly attributed source coordinate, an unlisted Case change, or a changed-detail consumer blocks closure.
- **R7:** Modeled behavior, Query coverage, bounds, expectations and realization semantics are preserved. The behavior proof is the step tables (the Pins test for Product and System, interpreter-built tables for every other activity machine), unchanged from a baseline captured on fn-151's closed tree before the first edit, plus lowered Cases that are byte-identical apart from mapped identities and the exactly enumerated current-source line fields allowed by Source provenance and artifact identity. The final joined candidate repeats the independent full-inventory proof against the original sealed baseline, including fresh unchanged-producer evidence for the manifest's single mapped refusal coordinate. Content-derived Case identities, prepared assessment bindings and timeout diagnostics are disclosed and derived from the current artifacts; historical Case+Run pairs remain retained and crossed bindings are rejected. A projection compares the before and after IR after the normalizations above. Its remaining differences are positions, entries in the identity mapping, and entries in the structural-review section, each with its reason. Go tests that read renamed effects, rule shapes or state-expression shapes are updated at the regeneration. IR and Cases are regenerated and reviewed, and the model gate and model lint pass. Errors: a step-table difference, an unexplained IR difference, a failure of the source-provenance contract, or reuse of a historical binding with a changed Case blocks closure.

## Early proof point

Task fn-155-name-the-standalone-activitys-repeated.1 validates the approach. It captures the baseline step tables and IR, builds the normalizing projection, runs it over one scratch named-guard rewrite and one scratch effect merge, and probes the helper forms against the lifter. If the normalized projection cannot reduce those two rewrites to mapped identities, re-evaluate section A's scope before fn-155-name-the-standalone-activitys-repeated.2 and later tasks.

## Quick commands

```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
make lint-model
make umpire-check-cases
```

## Boundaries
<!-- scope: business -->

- **Layout.** This spec moves no models between files. fn-151 owns the layout, including Realization.scala staying one file.
- **Behavior and coverage.** No server behavior change and no new coverage. The batch 5 inherited failures are neither fixed nor waived.
- **Deferred to fn-138/fn-141 or later.** These are not done here.
  - Capability-level reset preemption and a typed `overriding`. Both change Retries/Deadline laws, their Nexus instances and the lifter's waiver parsing.
  - Reading a product phase declared on each System or Record phase, which makes `toProduct` a single expression.
  - A System `Phase` that declares its status. `unstarted` has none, and `timedOut`'s status is parameterized.
  - A generated per-policy `deadlines` start.
  - A named scenario prefix splat.
  - Grouped start inputs.

  fn-141 makes declaration-level forms free. Lifter recognition is not added here only to be deleted there.
- **Pending controls.** No `held` + `pending` remodel of Phase.

## Decision Context
<!-- scope: both -->

- **Position after fn-151.** The owner placed this spec after fn-151, so the subject-split files are refactored once. Batch 2 starts from this spec's closed baseline. fn-140.4 and fn-140.6 consume its identity mapping.
- **Batch nature.** Meaning-preserving with mapped identities, like fn-151: one regeneration, one gate run, one review, no live run.
- **`because` texts kept.** A collapse that would change them stops at the effect boundary instead.
- **Rejected: capability preemption inside this spec.** It reaches fn-138's sealed comparison, the Nexus IR and the lifter, so it is out of scope for a local refactor.
- **Rejected: merging the worker and By-ID blocks.** Their rows differ, so a comment states the intended correspondence instead.
- **Go-test updates live in one task.** They depend on the regenerated IR, so the closing task owns them; refactor tasks do not edit Go tests.
- Maintainability (plan review): duplication - Go-test updates were decided in two tasks (now only the closing task); structure - none identified.
- **fn-129.3 overlap.** fn-129.3 (in progress, batch 5) authored the reset overrides and reset effects. Its remaining live-Case work re-anchors to this spec's renamed effects and shared reason through the identity mapping.

- **Current-source provenance amendment.** The original Case byte promise failed when the single-definition Realization refactor moved seven timer source sites, changing eleven wait-hint source-line fields in eight freshly lowered Cases. The original strict RED receipt is retained. The root authorized this narrow contract after an independent fixed-candidate v3 proof of all 31 original files and 81 controls (79 rejected candidate mutations and two diagnostic-independence controls). The exception records truthful source attribution, derived content identities, prepared bindings and timeout diagnostics. It grants no canonical, live or semantic-equivalence credit, and it does not cover another task's coordinate changes. Task .6 must repeat the complete proof on the joined candidate; every other Case byte and all original behavior, coverage and gate obligations remain strict.

- **Unsupported manifest diagnostic refinement.** After the first plan review of the eleven Case leaves, the root separately found that the retained manifest was copied rather than freshly generated. The existing start-to-close timeout refusal cites the moved `attempts` script declaration. The root authorized exactly one additional manifest diagnostic coordinate, with unchanged refusal meaning and all other bytes protected, subject to fresh unchanged-producer located-refusal evidence and final joined reproof. The first SHIP receipt covers the earlier artifact only; this refinement requires its own independent review and grants no fresh canonical-manifest credit until its checks run.

## Parked unknowns

- **fn-129.3 ordering.** Should fn-129.3's open remainder land before this spec touches the reset effects, or re-anchor afterwards as planned? Owner decision.
- **Pending-control remodel.** Separating "attempt held" from "pending control" (ordered Cancel > Reset > Pause) needs Finite/role/lifter support. Owner decision for a later spec.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | System effects that differ only in landing phase collapse through one landing function. The failure, By-ID failure and attempt-deadline rules use named `exhausted`/`endsTerminally` guards. Every deadline rule reads the `armed` predicate its `Deadline` capability declares. Errors: a step whose facts, fact order, outcome, state or `because` differ from the baseline step table fails the R7 comparison. A held phase the landing function does not name is unreachable under the rule guards, which the step table confirms. | fn-155-name-the-standalone-activitys-repeated.2 | — |
| R2 | Product single-fact effects use the shared phase effects or the `Recorded` setter. Multi-fact and reasoned steps keep their baseline fact order and `because`. Errors: any fact-order change fails R7. | fn-155-name-the-standalone-activitys-repeated.3 | — |
| R3 | The six reset-aware `overriding` calls cite one shared reason value. Their emitted lint and metadata text stays byte-identical to the baseline. No error surface beyond the lift: a refused cross-object reference falls back to a local value, and the fallback is recorded in the mapping. | fn-155-name-the-standalone-activitys-repeated.2 | — |
| R4 | Fixed expected states derive from the initial state. The direct-reset Properties call the reset-dispatch predicate. Rejection reasons citing the same source are defined once, each level keeping its baseline text. Worker and By-ID blocks carry a must-match comment. By-ID failure reuses the kind's examples where the lifter accepts the reference; otherwise the examples stay repeated and the refusal is recorded. Errors: generated Case names or examples that change are listed in the mapping, or reverted. | fn-155-name-the-standalone-activitys-repeated.2, fn-155-name-the-standalone-activitys-repeated.3 | — |
| R5 | HeldDispatch derives from the corrected design, restricted, rebound and refining the Product, if and only if its interpreter-built step table, refinement, monitors and evidence equal the hand-written machine's. Otherwise it stays hand-written, and the reason is recorded. One capability class, bounded by a shared view of the composed record, serves both queue compositions if the lifter accepts it and the type change is mapped; otherwise the two classes stay, and the reason is recorded. Waiver reasons leave `object states`. Errors: a derived-machine or state-type identity change is mapped. A step-table difference keeps HeldDispatch hand-written. | fn-155-name-the-standalone-activitys-repeated.4 | — |
| R6 | In the Realization file, the run-event attempt record, the Describe read, the status and run-state conditions, the worker activation and the run-scoped Describe base each have one definition. Lowered Cases for every realization stay byte-identical apart from mapped identities and the exactly enumerated current-source line fields allowed by Source provenance and artifact identity. The independent full-inventory proof binds every allowed Case line to the same original and current declaration and ServerStep; all other Case bytes remain identical. The manifest preserves its original refusal bytes except its single separately enumerated current-source diagnostic coordinate. `make umpire-check-cases` passes. Errors: a missing, extra, duplicate or mislabeled inventory/map entry, a stale or wrongly attributed source coordinate, an unlisted Case change, or a changed-detail consumer blocks closure. | fn-155-name-the-standalone-activitys-repeated.5 | — |
| R7 | Modeled behavior, Query coverage, bounds, expectations and realization semantics are preserved. The behavior proof is the step tables (the Pins test for Product and System, interpreter-built tables for every other activity machine), unchanged from a baseline captured on fn-151's closed tree before the first edit, plus lowered Cases that are byte-identical apart from mapped identities and the exactly enumerated current-source line fields allowed by Source provenance and artifact identity. The final joined candidate repeats the independent full-inventory proof against the original sealed baseline, including fresh unchanged-producer evidence for the manifest's single mapped refusal coordinate. Content-derived Case identities, prepared assessment bindings and timeout diagnostics are disclosed and derived from the current artifacts; historical Case+Run pairs remain retained and crossed bindings are rejected. A projection compares the before and after IR after the normalizations above. Its remaining differences are positions, entries in the identity mapping, and entries in the structural-review section, each with its reason. Go tests that read renamed effects, rule shapes or state-expression shapes are updated at the regeneration. IR and Cases are regenerated and reviewed, and the model gate and model lint pass. Errors: a step-table difference, an unexplained IR difference, a failure of the source-provenance contract, or reuse of a historical binding with a changed Case blocks closure. | fn-155-name-the-standalone-activitys-repeated.1, fn-155-name-the-standalone-activitys-repeated.6 | — |
