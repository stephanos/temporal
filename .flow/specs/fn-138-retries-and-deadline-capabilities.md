# Retries and Deadline capabilities

> HTML render lens (local): open `.flow/artifacts/fn-138-retries-and-deadline-capabilities/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 2, part 1): "actually think a little harder; could we provide more reusability evne?"
> user (turn 2, part 2): "there are a few patterns across phases; like retry, failure, timeout etc."
> user (turn 2, part 3): "is there a way we could have several pre-defined reusable states like \"is terminal\""
> user (turn 3): "could we do sth more composable instead? we don't want to define all these types head. we want each of these to be a trait and then connect the actual enum with them"
> user (turn 6): "no problem; I accept the proposal. write a flow next spec"
> user (turn 7, selected): "Roles + caps + Retries"
> user (turn 8, selected): "split-as-proposed"

## Goal & Context
<!-- scope: business -->

The owner pointed at patterns that repeat across phases, "like retry, failure, timeout etc." Phase roles, introduced by fn-136, describe what a phase is. Retry and timeout describe transitions. The activity system and Nexus workflow system already declare these transitions and the windows their timers cover.

Retries and Deadline state the shared promises once. A machine declares their bindings in its capabilities section and receives Properties checked against its own steps. The machine continues to own its transition rules.

The owner approved the adjusted activity work plan and delegated the remaining recommendations. This refresh resolves the earlier planning questions under that authority. It does not attribute new quotations or behavior decisions to a feature owner.

## Architecture & Data Models
<!-- scope: technical -->

**Retries.** Each binding names one failure action class, whether that class is retryable, the attempt-count projection, the finite policy maximum or explicit unlimited policy, and the before-state retries-remaining predicate. Optional pending-pause and pending-cancel predicates declare control branches. An absent control binding contributes no branch and requires no phase case for that branch. These are classification inputs, not arbitrary predicates that define whether the expected landing happened. Properties obtain the phase from the machine's Phased declaration and check the landing against roles.

The failure settlement contract reads the state before the selected failure.

| Failure and before-state control | Required after-state role |
| --- | --- |
| Fatal, including pending pause or cancel | Failed |
| Retryable with cancellation pending, including exhausted attempts | Canceled |
| Retryable with no cancellation pending and no retries remaining | Failed |
| Retryable with retries remaining and pause pending | Suspended |
| Retryable with retries remaining and neither control pending | Waiting |

Cancellation has priority over pause for retryable failure. The exhausted-cancellation row preserves the independent activity Model's existing rule. The ordinary and pause rows preserve fn-128.3's policy exhaustion. A retryable failure need not originate in Held. Nexus handler failures and network faults originate in Waiting.

The policy bound compares the count with the declared finite maximum. A policy of one and a representable count of two violates the Property. The finite counter's representation ceiling is not the policy bound. Unlimited explicitly imposes no finite policy maximum, while the count remains in its declared finite domain. This does not claim an unbounded number of represented attempts. The typed binding is `Retries[S, P, N <: Int]` with `maximumAttempts: S => Option[UpTo[N]]`; its companion bound Property carries the same `N`. `None` means unlimited and `Some(UpTo[N](maximum))` supplies the policy value. `N` bounds the finite value catalog, independently of the attempt-count domain; it is not the selected policy maximum. The existing lifter cannot enumerate `Option[Int]`, so the binding reuses the existing finite Option and UpTo forms without widening expression or schema admission.

**Deadline.** Each binding names its timer class, covered role, before-state armed predicate, typed terminal timeout fact and whether the timer can retry. A retryable binding also supplies explicit retries-remaining and control predicates. Each selected firing must originate in an armed state whose phase has the covered role, whether or not it records a terminal fact.

| Deadline firing and before-state control | Required after-state role and facts |
| --- | --- |
| Cancellation pending | TimedOut and the binding's typed terminal timeout fact |
| Nonretryable deadline, or exhausted retryable deadline | TimedOut and the binding's typed terminal timeout fact |
| Eligible retryable deadline with pause pending | Suspended, with no terminal timeout fact |
| Eligible retryable deadline with neither control pending | Waiting, with no terminal timeout fact |

Deadline cancellation settlement is TimedOut, as fn-128.3 specifies. It differs from retryable worker-failure cancellation settlement. The existing activity statusTimedOut(type) confirms ACTIVITY_EXECUTION_STATUS_TIMED_OUT and remains a terminal fact. The Nexus timeout fact likewise describes terminal timeout. Eligible retries must not emit any value of that terminal timeout fact family. No new fact is needed merely to check the timer window.

Activity schedule-to-close covers Live, schedule-to-start covers Waiting and is armed only while dispatch is now, and start-to-close covers Held and can retry. The Nexus workflow's three deadlines cover Live, Waiting and Held respectively and terminate when they fire. Heartbeat adoption belongs to fn-129 and follows the Deadline contract when that task introduces its timer.

**Checking.** The existing Scala and IR property forms already carry an action selector together with holdsAcross(before, after). The reader's transition verification must retain that selector, evaluate only selected steps and preserve the before-state. Capability-generated transition claims use free verify Queries. The narrow extension keeps transition find refusal and the current Step and protobuf schemas. Check, the private key engine and export agree on selected, skipped and exercised steps.

**Expansion.** Both capabilities follow the existing companion-defined Property shape. New Retries and Deadline instances generate `<machine>.<val>.<property>` names for their Properties, Scenarios and Queries, retaining the companion definition as origin. Their parameter resolution reads the current declaration's own fields first. Cross-capability resolution applies only to fields the current declaration does not bind and retains ambiguity refusal. Existing capability kinds retain their names and IDs. A Property that reads an own binding field does not require Pollable merely because it reads Held or Waiting.

Capability expansion requires an explicit IR root. Task .3 adds `NexusSystem.capabilities` to `exports.nexusWorkflow` in `model/temporal/features/nexus/workflow/Workflow.scala`, preserving its existing roots and their ordering. A machine root or a bound statement in a queries section does not lift the capability declarations. Focused scratch and shared production lifts must contain every expected Nexus Retries and Deadline Property, Scenario and free verify Query; every generated Query must be exercised with a decisive within-limit receipt.

**Equivalence.** Existing Query behavior and the full ordered receipts remain strict against completed post-fn-128.5 source. Additive declarations may mechanically shift existing source positions. Only an explicit closed old-to-new correspondence of the edited source spans may account for those positions, retaining every file, line and column rather than blanking or dropping them. Whole-source Model fingerprints and embedded Case provenance may change because the source adds declarations. Every such difference needs the exact original/new hash-input ledger and independent recomputation from those inputs. Existing semantic IDs, Property truth on applicable rows, Scenario and witness path, Query limits/answers/order, Program and Contract, expected assessment and receipt semantics remain fixed. Unexplained coordinates, hashes or content are differences, not normalization allowances.

## API Contracts
<!-- scope: technical -->

- Retries binds an exact action class, a retryable classification, attempt count, finite policy maximum or explicit unlimited policy, retries remaining and optional pending controls. Each supplied function names a def of the lifted sources. The phase comes from Phased.
- Deadline binds an exact timer class, a covered role through its type witness, an armed predicate and a typed terminal timeout fact. Retry eligibility and pending controls are explicit when that deadline retries. The phase comes from Phased.
- Binding a pause branch requires a Suspended case; binding cancellation for failures requires Canceled. An absent branch requires neither that role nor a companion capability declaration. Waiting and Failed are required for Retries, and TimedOut plus the covered role for every Deadline. A retryable Deadline also requires Waiting and any declared pause branch's Suspended role. Held is not a Retries source prerequisite.
- The same machine may declare several instances of the new capability kinds. Two Deadline declarations with the same terminal timeout type are refused, naming both vals and both positions. Different timeout types retain distinct bindings and generated names.

## Edge Cases & Constraints
<!-- scope: technical -->

- A non-Phased machine, missing required phase role, unbound action class, non-timer Deadline action or malformed binding is refused at its declaration. Required-role refusal names the machine and role. A caller may not pass a lambda where the existing lifter requires a named def.
- A selected firing from an unarmed state or outside its covered role violates the window Property even if the step records no terminal fact. An unrelated action neither evaluates that Property nor marks it exercised.
- Checking retries remaining after the failure would misclassify Nexus's incrementing failure and exhaustion boundaries. Eligibility uses the before-state.
- A wrong retry landing, premature failure, extra retry after exhaustion, wrong control precedence, missing or wrong terminal timeout fact, or a terminal timeout fact on an eligible retry is a counterexample. Resource limits, unknown rows and a never-exercised selector retain the reader's existing separate statuses and evidence.
- A failed shared Property must be investigated without a silent waiver. A real conformance failure requires a human decision under AGENTS.md. Product behavior remains independently authoritative; matching the Go comparator alone cannot justify changing it.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Retries checks the control-aware failure settlement table and the finite policy bound, using before-state eligibility. The activity system and Nexus workflow system declare it, including Nexus's retryable handler error and network fault from Waiting, and the generated free verify Queries hold within their declared limits. Unlimited is explicit, and max one with count two refutes the bound. Errors: non-Phased machines, missing Waiting or Failed, missing roles for declared control branches, unbound failure classes and invalid bindings are refused with located diagnostics; wrong ordinary, pause, cancellation, fatal or exhausted landings produce counterexamples. Held and Pollable are not required source declarations.
- **R2:** Deadline checks every firing's armed predicate and covered role against the before-state and enforces the deadline settlement table. Both systems declare one per existing timer; eligible retries land in Waiting or Suspended without a terminal timeout fact, while exhaustion, a nonretryable deadline or pending cancellation lands in TimedOut with the correct typed terminal fact. Errors: missing covered or required landing roles, non-Phased machines, invalid timer bindings and duplicate timeout types are refused; duplicate-type refusal names both vals and positions; unarmed/out-of-window firings, wrong landings, missing/wrong terminal facts and terminal facts on retries produce counterexamples.
- **R3:** Remaining live, waiting and held timer windows use role tests. Against completed fn-128.5 source, every existing Query of the adopted machines preserves its Property truth on applicable rows, Scenario/witness path, limits, answer, ordering, full ordered receipt semantics, Definition ID and Case semantics/IDs, including identical Contract, controller/worker Program and expectations. The approved fn-138 delta adds declaration-scoped Properties, Scenarios and free verify Queries plus the equivalent remaining timer-window rewrite. Only actual mechanically shifted source positions may differ through a closed old-to-new edited-span correspondence; additive whole-source Model/Case-provenance hashes require an exact original/new hash-input ledger and recomputation. Errors: any unexplained coordinate/hash or unlisted semantic, receipt, identity or Case difference stops the comparison; blanking positions and arbitrary fingerprint normalization are forbidden. Production comparison waits for the shared activity batch regeneration.

## Early proof point

Task fn-138-retries-and-deadline-capabilities.1 proves that the existing action-selector IR supports before-state transition verification, that Retries lifts on a Waiting-source machine without Pollable or optional control roles, and that a max-one/count-two mutant fails. Failure of that proof requires revisiting the narrow binding or selector extension before .2 and .3.

## Quick commands

Focused commands run only after the fn-128.5 source gate and the planning review. Workers use the shared heavy-gate lock for expensive commands.

```bash
mise exec -- scala-cli test --server=false model/irgen --require-tests
mise exec -- scala-cli test --server=false model/project.scala model/umpire model/temporal --test-only '*Capability*' --require-tests
go test -tags test_dep ./tools/umpire/check ./tools/umpire/internal/engine ./tools/umpire/export
```

## Boundaries
<!-- scope: business -->

- Retries and Deadline check the machine's own steps. They generate no transition behavior.
- Backoff durations, retry interval/coefficient/maximum-interval policy and nonretryable error-type lists beyond the bound retryable classes remain outside this spec.
- The implementation extends existing action-filtered transition verification. It adds no Step action field, protobuf schema, alternate evaluator, generic driver/cache framework or per-landing predicate framework.
- fn-129 retains heartbeat/reset/by-ID ownership. This refresh does not implement or expand that spec or later schemas.
- Framework/lifter fixture goldens may be regenerated in a focused task. Production IR, Cases and their mirrors, full gates, review and live execution share the fn-128.6/fn-129.5 batch boundary.

## Decision Context
<!-- scope: both -->

The owner's original choice reads Waiting rather than requiring a Retrying phase. The activity's backoff is a dispatch field; Nexus's backingOff phase has the narrower Retrying role and therefore Waiting. The original statement that both models retry at their bound was superseded by fn-128.3's finite activity policy. Nexus retains explicit unlimited policy with a saturating represented count.

The approved work-plan delegation resolves the prior planning proposals. Strong control-aware landing checks replace the proposed weak "not Failed" guarantee. Nexus uses exact failure classes from its Waiting phase. Finite policy maxima replace representation-ceiling checks. The existing selector/transition IR gains narrow reader support rather than fact-keyed timer windows. New instance names are `<machine>.<val>.<property>`, and field resolution begins at that declaration. These are adopted planning recommendations, not fabricated new owner quotations.

The delegated technical clarification of R3 preserves exact semantic comparisons while accounting for source coordinates and hashes mechanically derived from additive declarations. A closed edited-span correspondence accounts only for actual shifted positions, and exact original/new hash-input ledgers justify changed Model fingerprints or embedded Case provenance. This does not grant a general position or fingerprint ignore list. The existing disk.durableStays framework specimen changing from Unsupported to a supported check is intentional evidence for the reader extension; it is not an exemption for any production Query.

Retry and timeout were first raised as candidate phase classifications, but they describe transitions. A phase may be Retrying or TimedOut; the retry or timeout is the step into it. These capabilities sit beside the existing lifecycle capabilities and read roles.

Maintainability (existing plan review): task .3 keeps existing hand-written timeout Properties and Queries beside the new Deadline claims because R3 preserves them. Retirement remains separate owner-directed work. No new structural framework is justified.

Maintainability (plan review): duplication - Task .3 retains hand-written timeout settlement checks beside generated Deadline checks; this duplication is explicitly required by R3; structure - none identified.

This spec is one of three split from one conversation. It depends on fn-137, which depends on fn-136. The approved activity-batch conductor starts fn-138 implementation only after fn-128.5 is DONE and integrated. fn-128.5 supplies the final raw attempt-count reads for all four activity Realizes declarations, cancel-monotonicity checking and query/timer explanations; fn-138 must re-anchor those completed sources. Its precision work adds no authorization to change retry settlement behavior. Flow cannot express the cross-spec task gate, so the conductor enforces it alongside existing metadata. This scratch preparation does not satisfy that gate.

Each worker runs focused verification and may update scoped lifter fixture goldens. Production artifacts, full Model/Go/lint/fixture/canary gates, independent implementation review and live proof run at the single shared fn-128.6/fn-129.5 boundary. Activity specs do not close from source-only task evidence. The fn-138 equivalence proof uses the post-fn-128.5 source before fn-138 changes; the wider batch comparison separately retains its recorded activity baseline. A stale DSL-batch baseline cannot stand in for fn-138's R3 proof.

The independent activity Model preserves retryable cancellation-requested worker failure as Canceled. The comparator's Failed response is a known disagreement requiring human judgment if live conformance exposes it. Neither capability adoption nor a regenerated artifact may silently reconcile it.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Control-aware failures, before-state eligibility and meaningful finite policy bounds | fn-138-retries-and-deadline-capabilities.1, fn-138-retries-and-deadline-capabilities.3 | Focused proof in .1; production adoption proof at the shared batch boundary |
| R2 | Every-firing deadline windows and retry/terminal settlement with typed facts | fn-138-retries-and-deadline-capabilities.2, fn-138-retries-and-deadline-capabilities.3 | Focused proof in .2; production adoption proof at the shared batch boundary |
| R3 | Existing ordered receipts, answers, IDs and Cases preserved; declared additive delta | fn-138-retries-and-deadline-capabilities.3 | The conductor seals the isolated original/adopted source comparison before fn-129 or fatal observation changes; production artifact matching follows at the shared boundary |
