# Nexus.Race authoring design

Status: authorized prototype design under the user's delegated design/replanning authority. Examples below describe a candidate interface and are not compiled Lean; no final grammar or usability result is claimed. The user selected the current lifecycle followed by a cancellation/completion race. Prototype code will live under `model/Temporal/Feature/Nexus/Race`.

Tracking: [fn-65 — Nexus.Race authoring](../../../../.flow/specs/fn-65-design-and-prototype-approachable.md).

## Goal

A developer who understands state machines should be able to define and change ordinary finite feature behavior, state its requirements, choose a scenario, and ask a bounded question without writing Lean proofs or manipulating Umpire encodings. Reading the source should reveal which outcomes are possible and which claims are checked.

Success includes adding a state and transition. It is insufficient to simplify only the Properties and Queries over a Target that still requires an expert to maintain. Capability-law invention, general infinite-state models, and arbitrary correspondence proofs remain expert work.

The stretch goal is a readable view for a technically minded product owner, generated from the checked definitions. Human evaluation will distinguish readability from correct authoring; neither a shorter example nor an agent's success establishes usability.

## Proposed approach

Use typed, finite transition data for the model and keep Property, Behavior, and Query declarations separate. Build deep constructors over existing Umpire types before adding notation. Compare those ordinary Lean constructors with focused declaration syntax over the same constructors. Adopt syntax only for a measured improvement in authoring, reading, or diagnostics.

The finite transition data is the one source of the initial-state and transition enumerators. Do not maintain a table and a separate `step` function with independent meaning. The constructor derives a `FiniteMachine`, packages the existing `DraftModel`, and invokes `checkModel`. It does not introduce a second checked model or an alternate Property evaluator.

Authors explicitly supply domain vocabulary, stable semantic keys, initial states, transition alternatives and their facts, capability/provider selection, requirements, scenario constraints, Query form, and Limits. The interface handles canonical encoding, repetitive metadata, finite closure proofs, checked-result extraction, and planner adaptation.

Properties also support explicit applicability conditions and named cases for special behavior. Every applicable obligation holds together; source order, specificity, and priority never select a winning Property. This requires a bounded extension of the existing `Umpire.Property` language for typed Boolean conditions, not an exception mechanism that suppresses failures.

Three independently developed alternatives informed this proposal:

| Alternative | Strongest benefit | Limitation | Decision |
| --- | --- | --- | --- |
| Ordinary typed records plus a total admission function | Small implementation; generic proofs can travel through successful validator branches | A plain `def` only constructs data; compilation alone does not establish semantic admission or capture precise field locations | Build this semantic foundation, but exercise admission explicitly in tests |
| Focused `property%`/`scenario%`/`query%` forms inside ordinary `def` declarations | Retains Lean declaration names, typed terms, and navigation while making checking/source capture the frontend's job | Requires proof-generation and editor tests; no guarantee it beats good records | Preferred syntax candidate to compare after the foundation works |
| Typed finite tables with generated enum catalogs | Greatest reduction in work when adding states/transitions | Automatic constructor-name encodings make renames semantic; derivation adds tooling | Use the finite table; begin with explicit typed catalogs and stable keys, and defer enum derivation |

For the first prototype, explicit vocabulary catalogs make domain membership and stable wire keys visible. A state addition may require adding its catalog entry and transition rows, but must not require a proof or support-code change. Automatically collecting the domain from transition rows would lose isolated states and turn misspelled names into new states, so it is not an acceptable shortcut.

## Two models, with distinct scopes

### Baseline lifecycle

Reproduce the existing four-state model exactly in behavioral terms:

| Before | Action | After | Model Outcome | Lifecycle fact |
| --- | --- | --- | --- | --- |
| scheduled | start | started | started | started |
| started | cancel | canceled | canceled | canceled |
| started | reportSuccess | succeeded | succeeded | succeeded |

Allow the same two setups: an operation initially scheduled or initially started. Other state/action pairs have no transition in this focused model. `reportSuccess` is modeled handler progress, not a caller command. Cancellation is abstracted to one transition here, as in the existing example.

Nexus.Race has its own explicitly declared identity root. Compare model behavior through a documented mapping to Nexus; new IDs and source paths mean equal canonical artifact bytes are not the baseline expectation. For the two candidate authoring interfaces within Nexus.Race, hold IDs and semantic inputs constant and compare the resulting checked semantics and fingerprints. Source locations may legitimately differ.

### Cancellation/completion race

Use a separate Target so the new asynchronous meaning cannot silently change the baseline. Model one already-started operation and three transitions:

| Before | Action | Possible result | Meaning |
| --- | --- | --- | --- |
| started | requestCancel | cancelRequested | Cancellation was requested; no terminal result is implied. |
| cancelRequested | resolve | canceled | Cancellation wins the race. |
| cancelRequested | resolve | succeeded | Successful completion wins the race. |

`resolve` is an explicit abstract environment-progress Action. It permits two Target-owned outcomes; it is not an RPC that selects an outcome. This is a finite teaching abstraction of the race, not a complete Temporal cancellation protocol. There are no outgoing transitions from either terminal state.

The cancellation request emits a `cancelRequested` fact. Either resolution emits the appropriate lifecycle fact and a Boolean `terminal = true` fact. The actual Model Outcomes remain separately named: cancellation-requested, canceled, and succeeded.

The first race Property is: after a cancellation request, a terminal result is observed within one additional semantic transition. Its response may occur on the triggering transition or the next transition under the existing inclusive `eventuallyWithin` semantics; the model emits no terminal fact at the request transition, so its successful response is the next transition. The Behavior for demonstrating the race requires exactly `requestCancel` followed by `resolve`; the Query limit is two transitions and two selected Actions. This scenario explicitly includes environment progress. It does not prove scheduling fairness or runtime delivery.

An intentionally stronger candidate requirement, “cancellation always wins,” should produce the successful-completion trace as a counterexample. Keep this as a negative test, not as an accepted product requirement. A separate scenario ending after the request shows that a request alone cannot establish the bounded response; do not interpret such a short trace as proof of a runtime timeout.

This first race scope starts before a cancellation request against a started operation. Completion before the request, retries, repeated requests, multiple operations, and caller closure are explicit omissions, not forbidden-product claims. Those cases should exercise later extensions after the initial authoring experience is understood.

An alternative race design would model both cancellation delivery and successful completion as separate events in either order, with explicit behavior for the losing late event. That needs additional product decisions: ignoring, rejecting, or recording a late event is meaningful behavior. The proposed two-outcome `resolve` abstraction tests nondeterminism first without choosing those additional rules. It intentionally does not exercise partial-order authoring; a later comparison must cover that before generalizing from this prototype.

## Authoring shape

The preferred visual shape is a short typed declaration with named fields. Preserve ordinary Lean enum/record syntax where it helps navigation. Prefer whole next-state values in the first prototype so there is no implicit frame rule for omitted state fields.

The following candidate transition row shows all meaningful fields of a row; surrounding vocabulary, initial setups, provider selection, and target metadata are declared once in the same model module:

```text
{ key := "request-cancel"
  from := started
  action := requestCancel
  results := [
    { state := cancelRequested
      outcome := cancellationRequested
      facts := [cancelRequestedFact] }
  ] }

{ key := "resolve"
  from := cancelRequested
  action := resolve
  results := [
    { state := canceled
      outcome := canceledOutcome
      facts := [lifecycleCanceled, terminalTrue] },
    { state := succeeded
      outcome := succeededOutcome
      facts := [lifecycleSucceeded, terminalTrue] }
  ] }
```

`results` denotes all alternatives, never an ordered fallback. Empty alternatives are an authoring error; disabled pairs are absent from the transition data. Duplicate source/action rows are rejected so alternatives cannot accidentally be spread across competing definitions. Each result uses complete next-state/outcome/fact values. Facts are modeled facts, never raw Evidence.

The focused-syntax candidate for the race's existing Property/Behavior/Query languages is:

```text
property resolves_after_cancel on race
  id "resolves-after-cancel"
  requires cancellationLifecycle
  clause "terminal-response"
    when action requestCancel
    eventually fact terminal is true
    within 1 semantic_transition

scenario cancellation_race on race
  id "cancellation-race"
  requires cancellationLifecycle
  role "operation" starts in started
  actions exactly [request: requestCancel, resolution: resolve]

query check_cancellation_race on race
  id "check-cancellation-race"
  verify resolves_after_cancel
  in cancellation_race
  limits
    steps 2
    actions 2
    search 32
  policy exhaustive
```

The finite candidate budget of 32 is a proposed initial budget to verify against the real planner. Exhaustive verification must report Limit Reached if that budget cannot cover the admitted traces; the prototype must not treat the number as evidence of completeness. If the bound needs adjustment, make the declared change visible.

The block spelling above emphasizes readability. The preferred Lean integration to test is `def resolvesAfterCancel := property% race ...`, and equivalently `scenario%` and `query%`, so ordinary definitions, namespaces, and references remain available. Compare this spelling against the constructor-only form before freezing the grammar; this document does not authorize two parallel public syntaxes.

The simple syntax above packages existing patterns, `eventuallyWithin`, Behavior constraints, and Query data. The guarded-case design below deliberately extends the existing Property representation and semantics where single patterns are insufficient. Neither surface accepts arbitrary Lean predicates or supplies a separate evaluator. The plain-Lean comparison expresses the same declarations through named constructors and the same explicit inputs.

Stable IDs expand from an explicitly declared family root, language kind, and author-provided key. Lean declaration names remain editor names; renaming them must not change the explicit semantic key. Display titles, source positions, and prose do not choose identity. Clause IDs also use explicit stable keys.

Named occurrences such as `request` and `resolution` supply stable local keys independently of their positions in the action list. The list order still constrains the trace. The named `exhaustive` policy expands to the existing explicit strategy, seed 17, and Definition-ID tie-breaking, which must remain inspectable.

## Guarded Properties, exceptions, and conflicting requirements

### Applicability and named cases

A Property states what must hold when its conditions apply. An unconditional invariant remains applicable in every context it covers, including special cases. A case does not override another case or an independent invariant: all applicable obligations are conjoined. Overlap is valid when the obligations are compatible.

Use named cases when several behaviors belong to one requirement. Within the existing race scope, the ordinary request and the special resolution step can be described together:

```text
property cancellation_lifecycle on race
  id "cancellation-lifecycle"
  requires cancellationLifecycle
  when action is one of [requestCancel, resolve]
  cases complete exclusive

    case "request"
      when action is requestCancel
        and before operation is started
      ensure in the same transition
        clause "state": resulting operation is cancelRequested
        clause "outcome": outcome is cancellationRequested

    case "resolution"
      when action is resolve
        and before operation is cancelRequested
      ensure in the same transition
        clause "state": resulting operation is one of [canceled, succeeded]
        clause "terminal": fact terminal is true
```

This is proposed notation, not compiled Lean. The two outcomes in the resolution expectation are alternatives; they must not lower to two mandatory equalities. This Property supplements the independently authored bounded terminal-response Property rather than generating either requirement from the transition rows.

Case keys and clause keys are explicit stable identifiers; full clause identity includes both. Renaming a display name or moving a case cannot change its identity or precedence. The same meaning must remain expressible through typed constructors.

Ordinary case groups apply every matching branch and make no implicit coverage promise. Authors may request `complete`, `exclusive`, or both. Under the parent guard, `complete` requires at least one applicable case, and `exclusive` requires at most one. These obligations concern reachable trigger contexts admitted by the explicitly selected Target, Behavior, and analysis Limits. They do not require independent Properties to be exclusive. If there is no reachable parent trigger, report that separately; do not present vacuous completeness as exercised coverage.

The example “canceling an already terminal operation preserves its outcome” remains outside the first race model, which has no terminal cancellation transition. Adding a Property case alone must not invent that transition. A later model extension must first state the allowed late-request behavior, then independently check its requirement.

### Exceptions and temporal obligations

`when A unless E ensure P` means `(A and not E) implies P`. `unless` changes applicability; it does not catch a violation, turn unknown data into success, or waive an independent invariant. Require a named exception key for provenance. Unless an exceptional branch states a replacement obligation, behavior under `A and E` remains unspecified by this requirement and must be visible as such in the declaration explanation.

Case applicability is the conjunction of the effective parent guard, the case guard, and the negation of that case's exception. Complete/exclusive checks use these effective case guards. To express ordinary and exceptional behavior together, put the exception on the ordinary case and give its sibling an explicit exceptional guard. A parent-level exception excludes the whole group, including its children; it cannot contain its own replacement case. Report excluded trigger contexts separately so exclusions cannot be mistaken for exercised coverage.

The initial guard and exception vocabulary reads the prior state and selected Action at the triggering step. Conditions are evaluated once at that point. A missing or unsupported input is an admission/evaluation diagnostic, not a false atom that `not` can turn into an applicable exception. Evaluate negation only over a validated, complete context.

For `eventuallyWithin`, an applicable trigger creates an obligation retaining that trigger's coordinate and declared Limit. A condition becoming true later does not erase it. In particular, “already completed when cancellation was requested” is different from “completed while cancellation was pending.” Conditions over resulting states or future outcomes cannot appear in an initial `unless`; otherwise the result being checked could exempt itself after the fact.

The first prototype has no implicit cancellation of outstanding obligations and no `until` or dynamic-exception operator. The race's response explicitly allows either terminal outcome. A future interruptible obligation would require a distinct portable semantic extension covering the interrupt event, order, and remaining deadline, rather than reinterpreting `unless`.

### Required Property-language extension

The current `PropertyPattern` matches one field/reference/value constraint, and `transitionContract` accepts one antecedent and one consequent pattern. Multiple existing clauses cannot in general represent a compound antecedent: `(A and B) implies P` is not equivalent to requiring both `A implies P` and `B implies P`. This design therefore includes a small typed Boolean predicate vocabulary: atomic typed patterns, nonempty `all`/`any`, and `not`. Literal `one of` lowers to `any` over same-field equality atoms. Reject empty groups and unsupported expressions with source-located diagnostics.

Guards bind all atoms to the same triggering step and restrict them to its prior state and selected Action. The first compound expectations bind all atoms to the same resulting step and allow resulting-state, Model Outcome, and fact references. There are no unbound variables, arbitrary callbacks, general quantifiers, or implicit cross-step joins. Cross-field equality such as “outcome unchanged” is not automatically available; finite literal cases may express it, otherwise reject it until a separately specified operator exists. Existing single-pattern temporal responses retain their semantics; compound temporal response expressions are outside the first extension.

Represent guards, named cases, and optional coverage/exclusivity requirements as checked data owned by `Umpire.Property`. The extended checker validates types, permitted fields, references, capability requirements, identities, and structure. Its evaluator owns conjunction, disjunction, negation, applicability, and branch obligations. Update the semantic agreement proofs, canonical encodings, fingerprints, version handling, and affected consumers together. Preserve existing declarations' meaning and behavior-neutral encodings; reject new unsupported operators in downstream formats instead of omitting conditions or claiming success. No second case interpreter belongs in Nexus.Race or runtime code.

Frontend lowering must retain parent, case, exception, and clause identities plus author-facing source locations, so failures can explain which conditions applied. Static declaration checking does not by itself establish reachability, case coverage, or compatibility of all requirements.

### Conflict and coverage analysis

Conflicts must be exposed rather than resolved by priority or by restricting the Behavior to traces where a preferred Property passes. Genuine product precedence, such as a first-terminal-event rule, belongs in the Target's explicit transition semantics and remains subject to all applicable Properties.

For the first finite prototype, analyze an explicitly selected set of checked Properties against one checked Target, Behavior, and typed Limits. Use the existing planner/Query machinery where it suffices; put any additional bounded analysis behind the existing Property/Query ownership rather than introduce a new authoring language. Report the exact analyzed scope and distinguish:

| Finding | Evidence required |
| --- | --- |
| Uncovered case | Reachable trigger context satisfying the parent guard and no case guard in a group declared complete |
| Overlapping exclusive cases | Reachable trigger context satisfying two case guards in a group declared exclusive |
| Property violation | One admitted trace on which an applicable obligation fails |
| Contradictory same-step expectations | Reachable common trigger and incompatible obligations, such as one resulting state required to equal two distinct values |
| No compatible modeled continuation within Limits | A reachable common trigger with admitted continuations, plus exhaustive bounded evidence that none satisfies the selected joint obligations |
| Unexercised requirement | No matching trigger found, with the analysis's completeness or Limit Reached status stated separately |

A violating trace alone does not prove that Properties are mutually inconsistent. Absence of a compatible continuation can reflect the modeled transitions or the analysis limit; it is not automatically a logical contradiction in the requirements. No modeled continuation at all is a dead end or scope issue, not sufficient evidence of a Property conflict. Exhausted analysis budgets remain inconclusive. There is no promise of general or unbounded conflict detection.

Diagnostics identify the parent Property and case/clause IDs, source locations, triggering state/Action or trace prefix, relevant guards, and incompatible expectations. Coverage and analysis findings remain distinct from the existing per-Property truth result. Complete/exclusive declarations are themselves obligations, so their witnessed failures are reported rather than silently selecting a branch.

## Deep modules and ownership

| Module/interface | Author supplies | Implementation hides |
| --- | --- | --- |
| Finite Target construction | Ordered vocabulary, stable encodings/tags, initial setups, transitions/results, explicit provider selection | Membership/closure evidence, canonical representation, kernel construction, checked Target assembly |
| Property constructors | Typed conditions and expectations, named cases/exceptions, coverage requirements, capabilities, explicit time bounds | Field/reference/payload pairing and construction of checked portable Property data |
| Checked declarations | Existing declaration data and its Target | Source capture, existing checker invocation, success-proof construction and extraction |
| Planning adapter | Checked Query with named Limits and policy | Finite-completeness transport and dependent planner-kernel plumbing |

The first concrete interfaces to investigate are `FiniteMachine.ofTable`, typed pattern constructors, and source-aware checked-declaration commands. These are proposed names, not promises that a particular signature has been implemented. Choose the narrowest owning existing module after testing the design against both models.

Feature declarations and comparison fixtures live in `Temporal.Feature.Nexus.Race`. Reusable domain-neutral mechanics belong behind the corresponding `Umpire.Model`, `Umpire.Property`, `Umpire.Scenario`, `Umpire.Query`, and `Umpire.Search` interfaces. Any syntax frontend imports those owners; the low-level finite adapter itself remains free of new syntax and of Query/Planning imports. Do not build a new catch-all framework for the experiment.

Suggested feature files:

| File under `Nexus.Race/` | Purpose |
| --- | --- |
| `Lifecycle.lean` | Readable vocabulary, initial states, transitions, checked baseline Target |
| `Cancellation.lean` | Baseline Property, Behavior, and explicit Query |
| `Race.lean` | Independent asynchronous race Target and its declarations |
| `Tests.lean` | Behavior equivalence, both race outcomes, bounded claims, guarded cases, coverage/conflict examples, negative requirements |
| `AuthoringTests.lean` | Checked examples and invalid declarations, including unsupported conditions and exception timing, with expected diagnostics |
| `README.md` | Brief learning path and measured comparison results |

Use explicit imports of Nexus.Race during the experiment. It should have a dedicated runnable test root before prototype completion. The existing ordinary Nexus facade and inspector registrations continue to identify the established model. A generated review document is a later prototype deliverable if the authoring baseline works; it must be produced from checked data, not independently maintained examples presented as generated output.

## Correctness and diagnostics

Finite tables must reject missing domain members, malformed or duplicate stable keys, colliding encodings, duplicate source/action rows, empty results, and declared Actions with no executable row. All result states/outcomes/facts must belong to their declared typed domains. Explicit domain order remains the planning input; syntactic row order must not choose outcomes or providers. Preserve ordering semantics deliberately rather than sorting every input indiscriminately.

Action executability establishes that a declared Action has a row somewhere in the admitted domain; it does not establish reachability from every setup, or from any setup. Scenario admission and planning must retain responsibility for those stronger questions.

Mechanical closure evidence should come from generic proved constructors over finite data. For example, an enumerator derived from validated rows can carry membership evidence rather than require a separate proof at every feature definition. This proves properties of the supplied table; it does not prove that the table is the correct product model. Capability-law witnesses must continue to establish their actual stated laws; a dummy law or autogenerated success label is not an acceptable substitute.

The checked-declaration frontend runs the existing checker for diagnostics, then produces a Lean term whose required success evidence is checked by the kernel. Investigate ordinary reduction/`decide` or reusable reflection lemmas. Running native code to obtain a diagnostic is not a license to trust its answer as a proof. Do not silently use the existing `model` default, which invokes `native_decide`. Audit the resulting declaration axioms and measure elaboration; if the approach cannot be efficient under the established trust requirements, record the trade-off before changing the trust policy.

The constructor-only baseline may instead keep admission as a total function returning `Except` and pass a checked value onward only in its successful branch. That route avoids per-declaration extraction proofs honestly, but needs an explicit test/build admission gate. Do not describe successful elaboration of its raw records as successful model validation. Comparing this route with compile-time checked constants is an early prototype experiment.

Unknown references, wrong reference kinds, missing capabilities, duplicate IDs, unsupported expressions, missing units, and contradictory scenarios should point to the offending authored expression. Preserve the raw typed error kind and related Definition IDs for maintainers. Existing Target occurrence capture is reusable; Property/Behavior/Query errors currently require additional frontend source mapping. No partially checked declaration should be published after an error.

Prototype failures should look like “result state `canceld` is not declared” or “bounded progress requires a unit,” not an extraction proof goal. These are illustrative messages; the diagnostic tests will freeze the chosen text and exact source spans once the frontend is selected.

## Semantics that remain visible

- A conditional Property may hold without a trigger. Report trigger coverage as explanatory information separately from the existing pass/fail result; first test and explain the distinction before adding a reporting interface.
- All applicable case obligations and independent invariants hold together. Exceptions narrow named conditions, and any replacement behavior is a separately stated obligation.
- Initial exceptions use the triggering context; later events do not silently cancel an outstanding progress obligation.
- The Behavior defines which requests/traces are selected. It does not force cancellation to win.
- A witness and universal verification have different Query forms and result claims.
- All progress bounds state their stage/unit. Model-step bounds do not become runtime deadlines.
- A missing transition only excludes behavior from this finite model. Scope statements explain omitted real-world behavior.
- A finite model accepting a trace is separate from runtime Evidence that it happened. This prototype exercises offline authoring and planning.

## Prototype sequence and checks

Begin with the baseline Target and one cancellation declaration. A feature-author edit should add a state/transition without changing support code or writing a proof. Then reproduce start and successful completion with the same interface to expose special-case helpers.

Add the race Target, verify both complete outcomes satisfy the bounded terminal Property, and obtain the successful-completion counterexample to “cancellation always wins.” Check the request-only and no-trigger cases separately. Exercise witness and exhaustive forms, unsatisfiable scenarios, unsupported inputs, and Limit Reached.

Before polishing syntax, implement and check the bounded Property-language extension. Use the request and resolution cases above for normal and special behavior. Check compatible overlaps; an exclusive-group overlap; an uncovered complete-group case; contradictory expectations at the same reachable trigger; and a model violation that is not a logical contradiction. Include an unreachable trigger and exhausted analysis budget so absence of evidence cannot become a compatibility claim.

Test `unless` both true and false at the trigger, an exception with no replacement obligation, and an independent invariant that remains applicable during the exception. Reject wrong-kind and missing references, empty Boolean groups, unsupported cross-field comparisons, and guards reading the resulting state. Verify that a later state change does not withdraw a bounded obligation. Reordering cases must preserve meaning, stable IDs, and fingerprints; changing a guard must change its semantic fingerprint. Existing single-pattern regressions must retain their established meanings.

Compare plain constructors and focused syntax using the same declaration meanings. Keep alternative specimens in comparison tests so the default example has one authored definition. Add compilation tests for diagnostics and manually exercise editor completion, hover, navigation, and error recovery; mark each measured or unmeasured explicitly.

Run focused Lake tests, the owning aggregate test roots, `make lint-model`, applicable model regression gates, and the project-required `make lint-code` after implementation. Audit trust-bearing generated declarations. During this design-only phase, validate document links and consistency; illustrative snippets are not test evidence.

For human evaluation, use reading, transition extension, requirement editing, and error repair tasks. Measure semantic correctness, assistance, and recovery time before source length. Check whether a participant can explain why cancellation can lose, why a passing conditional rule might not have been exercised, what an exception leaves unspecified, and why two applicable requirements cannot be resolved by their declaration order.

## Recorded decisions and remaining boundaries

The user instructed: “defer fn-62, prototype fn-65, then retain only fn-62’s uncovered requirements.” The user's standing delegation of necessary design and replanning decisions authorizes the recommendations below for this experiment. This records a delegated design decision, not a line-by-line human grammar review or a human usability study. Implementation task planning follows this decision record; it does not establish that the examples already compile.

| Decision | Authorized prototype choice | Boundary still to establish |
| --- | --- | --- |
| Finite authoring | Explicit typed catalogs, stable keys, whole next states, and explicit transition alternatives; derive enumerators and mechanical evidence from that data | Reject invalid catalogs/tables and demonstrate a state/transition edit without feature proof or support-code editing |
| Model scope | Preserve the existing baseline through an explicit identity mapping, then add the separate request/abstract-resolution race | No late-event policy, fairness, runtime delivery, or complete cancellation protocol is inferred |
| Requirements | Extend the existing `Umpire.Property` owner with typed Boolean guards, named cases and exceptions; conjoin obligations and evaluate applicability at the trigger | Preserve legacy semantics and encoding, update proofs and affected consumers, reject unsupported operators |
| Analysis | Finite, correlated coverage and conflict analysis with reachable contexts and bounded evidence | Separate violation, contradiction, dead end, unexercised guard, modeled incompatibility, and Limit Reached |
| Interface | Constructor admission first, followed by a focused frontend comparison over the same checked declarations | Select one default authoring surface only after actual compilation, diagnostics, trust and editor measurements; unmeasured observations stay labeled |

### Concrete rule exceptions for this experiment

[GOV-02, AUT-07, and AUT-08](../../../../.plans/UMPIRE4_SPEC.md) remain authoritative. Under the human's standing delegation, the following narrow exceptions are accepted for the fn-65 prototype; they do not rewrite those rules or authorize a repository-wide migration.

- **AUT-08 evidence responsibility:** the rule says authors MUST provide ordered domains, encoders, enumerators, domain-closure evidence, and Action-executability evidence. Requiring ordinary authors to provide only finite catalogs/tables while generic constructors supply enumerators and proofs changes that authoring responsibility. The prototype permits those mechanical obligations to be discharged by reusable kernel-checked constructors over validated author data. The resulting `FiniteMachine` still carries every required witness, produces `DraftModel`, and passes `checkModel`. Capability laws remain real obligations, and a rejected table cannot produce a checked Target. This is not permission to omit evidence or manufacture success.
- **AUT-07 syntax path:** a focused wrapper accepting Property/Behavior/Query declarations may constitute another way to define behavior even if it lowers to existing types. Lowering alone does not demonstrate compliance. The prototype permits an isolated comparison frontend, with examples/fixtures in Nexus.Race and reusable syntax outside the low-level finite adapter. Existing pure language owners retain validation and evaluation authority; no independent behavioral interpreter or second production authoring path is authorized. Choosing a production frontend or broader adoption requires a recorded reconciliation with AUT-07 after comparison evidence exists.

AUT-08's prohibition on a macro language inside `FiniteMachine` remains intact. Generic finite construction belongs behind `Umpire.Model`; Property semantics belong in `Umpire.Property`; syntax is only a frontend importing the owners. The guarded-case extension is deliberate semantic work, not behavior-neutral helper cleanup. There is no exception to the no-hidden-native-trust boundary: native diagnostics cannot discharge proof obligations, and the current `model` default cannot be silently reused. If kernel admission proves impractical, report measured cost and retain the successful-branch constructor route or record an explicit further decision before changing trust.

[fn-62](../../../../.flow/specs/fn-62-make-ordinary-temporal-model-authoring.md) remains deferred until the fn-65 prototype is evaluated. Its author-written success-evidence and no-new-syntax constraints are not imposed on this separate experiment and are not globally removed. Reconcile its requirements against actual fn-65 evidence afterward, retaining only uncovered requirements; do not mark requirements covered from this design alone.

The key implementation uncertainties are kernel-checked validation performance, the cost and diagnostic precision of bounded case/conflict analysis, and how much editor tooling the focused syntax requires. The first prototype should resolve those before expanding to caller closure, multiple entities, arbitrary temporal formulas, interruptible progress obligations, System links, Evidence mappings, or live execution.

Sources: [authoring assessment](../../../../.plans/FEATURE_AUTHORING_ASSESSMENT.md), [research](../../../../.plans/FEATURE_AUTHORING_RESEARCH.md), [Umpire specification](../../../../.plans/UMPIRE4_SPEC.md), [Lean guidelines](../../../../.plans/LEAN_GUIDELINES.md), [existing Nexus lifecycle](../Nexus/Lifecycle/Semantics.lean), and [finite adapter](../../../Umpire/Target/FiniteMachine.lean).
