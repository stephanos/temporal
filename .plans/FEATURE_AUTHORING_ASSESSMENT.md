# Feature spec authoring assessment

Assessment date: 2026-09-04. This is a research assessment and a proposed evaluation, not an approved implementation design. Related implementation work: [fn-62](../.flow/specs/fn-62-make-ordinary-temporal-model-authoring.md). External language research: [FEATURE_AUTHORING_RESEARCH.md](FEATURE_AUTHORING_RESEARCH.md).

## Judgment

The current Feature surface is a useful semantic foundation, but it does not yet meet the intended experience for developers without deep Lean knowledge. Its small lifecycle relation is approachable; constructing and checking the surrounding model exposes considerably more framework and proof knowledge. Moving that work into another file helps readers but does not remove the burden from someone adding behavior.

The strongest direction to evaluate is a small, typed Lean authoring interface, with routine proof and encoding work inside Umpire. Add focused declaration syntax only where it demonstrably improves writing, reading, or diagnostics. Keep states/transitions, Properties, Behaviors, and Queries distinct. Product owners should initially review a generated explanation of the checked declarations, with examples and explicit scope.

This recommendation is based on code inspection, not observed author performance. Human trials are necessary before calling any alternative the best. The unresolved audience decision is whether ordinary authors define new states/transitions or only use maintained Targets. For this assessment, success should include routine finite state/transition edits; restricting authors to existing Targets is a materially narrower goal.

## What works

| Current choice | Why retain it |
| --- | --- |
| Named states/events and a small pattern-matching `step` | Developers can see the complete focused lifecycle without understanding tactics. |
| Feature/System separation | Product requirements can be reviewed independently of mechanisms and runtime Evidence. |
| Separate Target, Property, Behavior, Query | Distinguishes possible behavior, required behavior, selected scenarios, and the question being asked. |
| Target-owned outcomes | A scenario cannot make an expected result true simply by requesting it. |
| Typed checks and explicit finite Limits | Invalid input and incomplete search have defined meanings. |
| Stable IDs and canonical artifacts | Support review, references, reproducibility, and traceability. |
| Expected and deliberately wrong traces in tests | Demonstrate how clauses discriminate and which layer owns a failure. |

The clearest entry point is [Lifecycle/Semantics.lean](../model/Temporal/Feature/Nexus/Lifecycle/Semantics.lean), particularly lines 33–53. It is a deliberately small teaching model, not a complete Nexus product specification.

## Where the experience breaks down

1. **A small semantic change has a large authoring surface.** The 71-line Semantics module is accompanied by a 451-line [Target module](../model/Temporal/Feature/Nexus/Lifecycle/Target.lean). The latter includes useful contracts and compatibility declarations, but also hand-built encodings, decoding to domain types, domain lists, and coverage/executability proofs. Those line counts are a navigation signal, not a productivity metric. The decisive question is how many of these concepts an author must edit to add a state or transition.
2. **Ordinary walkthroughs still expose checker plumbing.** [Cancellation.lean](../model/Temporal/Feature/Nexus/Operations/Cancellation.lean) is 107 lines. It repeats raw result, `toOption.isSome` proof using `native_decide`, and checked extraction for Property, Behavior, and Query. It also includes teaching traces and planner runs. A production authoring entry point should make the declarations prominent, with validation fixtures and planner mechanics available separately.
3. **Values lose domain meaning at the interface.** Expressions such as `PropertyPattern.exact .selectedAction cancelActionId cancelAction.value` make authors join a field, reference, and encoded payload. Typed references should own that pairing. Existing canonical encoding can remain behind the interface.
4. **Repeated IDs obscure the rule.** Stable identity is necessary; repeatedly writing a full family prefix is not. Authors should declare stable semantic suffixes, with the complete IDs inspectable. Neither line numbers nor declaration order should supply identity. Preserve explicit wire tags when names are renamed.
5. **Source data is too coarse.** Operations share a source pointing at `Operations.lean`, and `Temporal.Shared.sourceLocation` fills in line/column 1. That identifies a facade, not the offending declaration in `Operations/Cancellation.lean`. An improved interface needs source capture and errors at the offending expression, including through helpers.
6. **The question and bounds are hidden.** `Internal.queryDeclaration` chooses `.witness`; the lifecycle supplies `QueryLimits.bounded 1 1 8`. A reader must navigate to discover that this seeks one satisfying example within transition/action/candidate bounds. It is not universal verification and those numbers are not execution deadlines.
7. **Some conceptual distinctions need explicit explanation.** `transitionContract` and `inputOutput` currently evaluate the same per-step implication. Their low-level constructor names do not explain temporal scope. The ordinary public facade also exports a synthetic Observation example, which introduces Evidence mechanics early. Keep that path discoverable without making it prerequisite reading for product requirements.

## Meaning that a friendlier surface must preserve

- **Disabled versus unsupported:** `step ... = none` means no modeled transition. It cannot by itself explain whether the product rejects the action or the model simply omits the case. State the model scope and Known Gaps separately.
- **Requests versus facts:** asking for cancellation, receiving cancellation, and reaching a canceled state are different claims. The teaching model abstracts these into one transition; do not present that abstraction as real-world timing or delivery behavior.
- **Conditional success versus exercising a rule:** `evaluateTransitionContract` uses implication for every step; no matching trigger can satisfy the clause. `eventuallyWithin` likewise quantifies over matching triggers. Behaviors can require the trigger, and reports should show whether it was exercised. Do not silently change Property truth conditions to solve a reporting problem.
- **Same transition versus bounded progress:** use wording that says which one applies. Progress must name a Limit and its unit; logical time requires its declared time source. A semantic transition is not a wall-clock duration.
- **Example versus verification:** a satisfying witness, a counterexample search, and exhaustive verification within Limits are distinct Query forms. Make this choice visible at the Query declaration.
- **Nondeterminism:** a scenario asks for actions; the Target may permit multiple outcomes. A state table must not implicitly make the first matching row win. Initial states, frame conditions for structured updates, and terminal behavior need defined semantics.
- **Correlation and cardinality:** a phrase such as “every operation receives its own cancellation exactly once” needs entity binding, ordering, and count semantics. The current portable patterns have a field, reference, and scalar constraint; they are not a general quantified relational language. New wording must either lower faithfully to target-owned facts/relations or require an explicit semantic extension.
- **Independent requirements:** automatically generating every Property from the very transition it checks mostly checks consistency. Authors still need to state meaningful requirements, with negative cases that would violate them.

These conclusions follow from [Property/Language.lean](../model/Umpire/Property/Language.lean), [Property/Evaluation.lean](../model/Umpire/Property/Evaluation.lean), [Query/Language.lean](../model/Umpire/Query/Language.lean), and the ordinary operation tests.

## Three approaches

| Approach | Benefit | Cost or limitation | Assessment |
| --- | --- | --- | --- |
| Deepen ordinary Lean constructors and checked interfaces | Least language maintenance; retain Lean names, types, navigation, existing semantics | Badly chosen helpers merely shorten boilerplate; explicit extraction proofs would still exclude novices | Required baseline and first candidate |
| Focused Lean declaration syntax over those same interfaces | Domain phrases, automatic source capture, concise declarations, tailored errors | Requires elaborator/tooling ownership, trust discipline, and precise lowering; syntax alone cannot repair missing semantics | Preferred additional candidate to test, not yet a proven winner |
| External YAML/Gherkin/standalone language | Familiar presentation and possible non-developer editing | Another compiler/tooling surface; awkward relations and nondeterminism; handwritten external behavior conflicts with current Lean authority rules | Poor fit for the current architecture; useful as a generated review view |

Lean technically supports custom syntax and elaborators. That establishes feasibility, not usability. See the separate research note for primary sources and version limits.

## Concrete candidate for discussion

The following is **illustrative proposed notation, not implemented or compiled Lean**. It sketches the existing cancellation example, preserving all three state/outcome/fact clauses and the distinction between Property and Behavior. Full declaration IDs, capabilities, role identity, and clause IDs must remain explicit or deterministically expanded from declared stable suffixes. An editor should expose the expanded IDs and checked declaration.

```text
property cancellation on Nexus.lifecycle
  requires Nexus.lifecycleCapability
  when action cancel
  in the same transition
    clause state:       resulting state is canceled
    clause outcome:     outcome is canceled
    clause observation: lifecycle fact is canceled

behavior cancel_started on Nexus.lifecycle
  requires Nexus.lifecycleCapability
  role operation starts in started
  actions exactly [cancel]

query cancellation_example
  find witness of cancellation
  in cancel_started
  limits transitions 1, selected_actions 1, candidate_evaluations 8
  policy shortest
```

`when` means a conditional rule, `actions exactly` requires the request, and `find witness` states the limited claim. None of the three blocks runs an RPC. A request to verify all admitted traces would use an explicit different Query form, not change the meaning of this example.

For new finite lifecycle behavior, retain the readable native `step` relation if its adapter can remove the repetitive work. An alternative to evaluate is a typed finite transition table whose rows are the single behavioral definition. Umpire would derive membership/closure support from the table through generic proved constructors and obtain the same checked Target. Do not maintain a table and a hand-written step function as independent authorities. Arbitrary independently specified relations and new capability-law proofs remain expert work.

## Technical design criteria

1. Put repeated encoding, checked extraction, completeness transport, and planning adaptation behind deep Umpire modules. Ordinary inputs remain states, actions, outcome alternatives, facts, requirements, IDs, selected providers, and explicit bounds.
2. Elaborate any notation directly to the existing inert declarations and checker path. Preserve serializable semantics, separate languages, Feature isolation, deterministic ordering, and canonical artifacts. Do not introduce a callback evaluator or a second interpreter.
3. Routine author files should not contain `native_decide`, `Except.toOption`, or dependent equality transport. This is a desired usability criterion, not a claim that the current APIs achieve it. Generic proofs and kernel-checked generated terms are potential mechanisms; evaluate their cost and audit their transitive axioms. Never hide a new compiler-trust dependency behind friendly notation.
4. Offer diagnostics in domain terms: unknown state, unsupported transition, duplicate identity, missing capability, inconsistent scenario, missing unit. Retain detailed typed diagnostics for maintainers. A compiler error about `isSome = true` is insufficient author guidance.
5. Keep hover information, completion, go-to-definition, and formatting in the acceptance criteria. Expansion should preserve locations for nested clauses. A syntax example that compiles is not evidence that these editor behaviors work.
6. At larger model sizes, measure incremental elaboration and bounded search independently. Smaller source does not avoid state-space growth; never silently shrink the model or relax checks for speed.
7. Generate the product-owner view from checked declarations: scope, states/transitions, requirements, representative allowed and violating traces, limits, Known Gaps, and the exact Query claim. Bind it to source identity/fingerprint. Free prose can explain rationale but must not silently become a second behavioral definition.

## How to choose with evidence

Use the current source as baseline, then compare an improved plain-Lean candidate and focused-syntax candidate. Test against the same meanings and negative cases. Include developers who have not worked on Umpire and have little Lean experience; do not substitute agent-only trials for this audience. Counterbalance candidate order to reduce learning effects. A small formative pilot of roughly five to eight developers can expose major failures; it does not establish statistical superiority.

Give each participant a short concepts guide, then ask them to:

1. Explain cancellation, the no-trigger case, and what the witness Query establishes.
2. Change one requirement and repair a deliberate wrong-state/reference error.
3. Add a new finite state and transition, including its outcome and fact.
4. Express a bounded response and a two-outcome race without fixing the outcome in the scenario.
5. Inspect a duplicate-cancellation counterexample and explain why a missing trigger is different from a violation.

Measure correctness of the resulting meaning, time to first correct declaration, help needed, error-recovery time, files/concepts touched, and accuracy of explaining the result. Record confidence separately from correctness. Measure compilation and editor response under the same toolchain and warm/cold conditions. Ask one or two technically minded product owners to review the generated view and identify scope and a planted requirement error.

Proposed first gate: ordinary tasks require no proof editing; participants distinguish requests from outcomes, witness from verification, and no-trigger satisfaction from exercised coverage; invalid inputs identify the authored cause. A candidate that looks better but increases semantic mistakes fails. Cases needing new core operators should be scored separately from authoring existing semantics.

## Relationship to existing plans

[fn-62](../.flow/specs/fn-62-make-ordinary-temporal-model-authoring.md) already proposes identity/source helpers, named Limits, narrower constructors, and planner adaptation. Retain those useful directions. Its explicit success-proof requirement and exclusion of new syntax should not be mistaken for evidence that they meet this broader audience goal. It also permits stronger Lean knowledge for Target maintenance.

[UMPIRE4_SPEC.md](UMPIRE4_SPEC.md) requires Lean authority, explicit semantics, existing public Property/Behavior/Query languages, and checked Targets. AUT-07 forbids alternate behavioral authoring paths; AUT-08 says FiniteMachine must not introduce a macro language. These are not blanket bans on every Lean macro. Any proposed syntax must demonstrate one-to-one lowering and reconcile its scope with those rules and fn-62 before implementation. Changing the rules is a deliberate design decision, not a technical impossibility or an implicit approval in this assessment.

The older fn-14 usability pilot is explicitly superseded and should not be resumed as a roadmap gate. Its historical status does not remove the need for fresh human usability evidence.

## Verification performed

Source review covered the authoring guidelines, authoritative Umpire spec, model architecture, current fn-62 plan, Nexus lifecycle/Target and operation walkthroughs, selected experimental caller-closure and Observation material, finite-machine adapter, Property semantics, Query forms, and associated tests.

`cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests` passed (43 jobs). This validates the inspected ordinary baseline, not any proposed interface or human-usability claim. No implementation changes were made by this assessment. Full lint, full regression, prototype compilation, trust audits of proposed interfaces, and human trials were not run.
