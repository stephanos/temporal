# Named edge-case situations

> HTML render lens (local): open `.flow/artifacts/fn-152-named-edge-case-situations/spec.html` - regenerable, markdown is the record. <!-- flow-next:artifact-link -->

**Deferred** by the owner on 2026-10-08, before task planning. The spec remains unready with no tasks. [paraphrase]

## Conversation Evidence

> user (turn 1): "in umpire (see model/); what if we had the idea of \"interesting/named edge cases\" sth like \"when Nexus task is retried, but the original answer is sent back (ir a race)\" and have a distinct definition of the particular case/state? I could imagine (a) it would be nie for regression testing and (b) for documentation/naming things"
> user (turn 2): "let's flesh this out into a flow next spec (but mark it as deferred) and add it to MILESTONES.md"

## Goal & Context
<!-- scope: business -->
<!-- Source: [paraphrase] for the goal; [inferred] for the proposed design. -->

Give interesting edge cases a stable name and a precise definition that engineers can use in regression tests and documentation. The motivating example is an original Nexus task reply arriving after that task has been retried. A reader should be able to identify the situation, understand why it matters, and follow an example execution to the behavior being checked.

The proposed concept is a **Situation**. It identifies a meaningful condition or event ordering that multiple executions can exhibit. A Scenario constrains how an execution proceeds, a Property states what behavior must hold, and a Case remains the executable Program and Contract. A Situation supplies the reusable definition of what makes an execution interesting.

Regression coverage must establish both that the Situation occurred and that the selected Properties held. A test which completes successfully without reaching the intended race does not establish that regression's coverage.

## Architecture & Data Models
<!-- scope: technical -->
<!-- Source: [inferred]. These are proposed contracts, not separately approved implementation choices. -->

### Definition and ownership

A Situation is a Model-owned declaration with a stable Definition ID, a human-readable explanation, an owning machine, and an executable occurrence condition. Existing identity conventions apply. Documentation and source-order changes leave its semantic identity unchanged; changes to the occurrence condition change its Behavior Fingerprint. References use the declaration's identity rather than copied predicates or free-text labels.

The condition can recognize a state or a bounded ordering of steps. Ordered conditions may bind finite operation and attempt values and require later steps to refer to those same values. They use the Model's declared actions, state, outcomes, and Facts. The author states any required adjacency; an ordering requirement alone permits unrelated intervening steps. Every search or live observation remains bounded by the Query or Case that uses it.

Recognition does not change a machine's transition relation. Historical information needed only to recognize an ordering belongs to the recognizer; distinctions that change legal behavior belong in the machine itself. This avoids adding a synthetic product phase for every named race.

### Occurrence and correctness

A Situation's condition says that the interesting circumstances happened, independently of whether the system responded correctly. It must still match an execution containing the bug. Existing Properties and monitors express the obligations at the matched occurrence; they are referenced separately and evaluated with the occurrence's correlation bindings. Finding an occurrence is an existential coverage result, not verification of every execution exhibiting it.

A Scenario or witness Query can target a Situation and select the Properties to check. Several witnesses may demonstrate one Situation, and a witness may encounter several Situations. References must make the regression's required target explicit; incidental matches do not substitute for it. Repeated matches retain their own bindings and evidence so different operations cannot discharge each other's obligations.

### Model-to-Run boundary

The definition is carried through the Model IR with its meaning intact. Model checking finds bounded witnesses of occurrence and checks the selected obligations. Live lowering derives occurrence requirements and correctness checks through the realization's declared evidence and controls. The generic runtime evaluates the resulting Contract; it acquires no hard-coded knowledge of named Nexus races.

Reports keep target occurrence separate from correctness. A model witness, an executed Program, and a live Run with sufficient evidence are distinct proof levels. A report links each confirmed occurrence to the steps or Run Events that establish it. Existing Verdict semantics remain authoritative; occurrence reporting introduces no competing success verdict.

### Nexus proof example

Use `originalReplyAfterRetry` as the proposed canonical example. Within one operation, attempt A is dispatched, attempt B is dispatched as a retry of A, and a reply originating from A reaches the modeled receiving boundary after B's dispatch. A reply merely produced or sent by A does not prove receipt there. Define two documented variants: A's reply arrives before B's reply, and A's reply arrives after B has settled the operation.

The example must distinguish operation identity, originating attempt, active attempt, and the relevant dispatch/receipt/settlement events. The current operation-level attempt count alone cannot establish those relationships. At least two distinct attempts are needed; their concrete identifier spellings do not define the Situation.

The Situation does not prescribe whether the old reply is accepted, rejected, or otherwise handled. That behavior must come from an independently stated Nexus contract. A task reply and an asynchronous operation-completion callback have different authority and must not be conflated. The example's exact receiving boundary and allowed outcomes require resolution before implementation.

## API Contracts
<!-- scope: technical -->
<!-- Source: [inferred]. Contracts below describe capabilities, not final DSL or wire syntax. -->

- Declaration admits a stable name, explanation, owner, and typed occurrence condition. Duplicate identities, unresolved references, invalid ownership, unsupported predicates, and unbounded recognizer state produce source-attributed diagnostics.
- Targeting references an existing Situation from a compatible Query or witness, with separately selected correctness obligations. Unrelated machines or mismatched correlation domains are refused. Cross-machine recognition requires an explicit composition or mapping; it is not inferred from similar names.
- Recognition reports the Situation identity, the applicable bounds, whether occurrence is established, and the supporting bindings and evidence. Failure to find a model witness within a completed bounded search is distinguished from an incomplete search. Live evidence insufficient to decide occurrence remains explicitly inconclusive.
- A regression requiring occurrence cannot succeed solely because its safety Properties held vacuously. If complete sufficient evidence shows the target was not reached within the declared window, report that unmet target. If evidence or execution is incomplete, preserve the reason for uncertainty. A proven correctness violation remains authoritative even when another assessment is inconclusive.
- Documentation exposes each Situation's name, meaning, occurrence condition, variants, selected Properties, and available witnesses or live evidence. Model-only and live-proven examples are labeled distinctly; an unsupported realization is an explicit gap.

## Edge Cases & Constraints
<!-- scope: technical -->
<!-- Source: [inferred]. -->

- Two operations may retry concurrently. A's reply for one operation cannot satisfy the occurrence or Property obligations of the other, even when their attempt counters match.
- A delayed original reply, a duplicate current reply, a new retry reply, and an asynchronous callback are separate events. Wrong origin or a reply arriving before retry dispatch must not match the canonical example.
- A common terminal state is insufficient to distinguish different event orderings. Test traces with identical final states but different histories must produce different occurrence results where the definition requires it.
- Causal order must be established by declared evidence. Observation arrival order across independent sources does not, by itself, establish the required race. Ambiguous explanations yield an inconclusive occurrence assessment.
- Bounds, truncation, unavailable controls, and missing observations remain visible. No bounded search claims unbounded unreachability, and no sampled live execution claims exhaustive schedule coverage.
- A regression may hold and release work only through declared controls supported by its environment. Sending a late reply without proving its receipt is insufficient coverage evidence.
- Existing declarations retain their behavior when no Situation is referenced. Migration of a representative existing named activity race demonstrates reuse without requiring a new machine per Situation.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Authors can define and reference a named edge-case condition for regression use and documentation, including an explanation and stable identity. Errors: duplicate identities and missing or incompatible references are rejected with source attribution. [paraphrase] The proposed declaration kind is Situation, with behavior-sensitive fingerprints and documentation-neutral identity. [inferred]
- **R2:** State conditions and bounded ordered conditions recognize their declared circumstances, including operation/attempt correlation, independently of correctness. Positive and negative traces with the same terminal state demonstrate ordering sensitivity; a trace with the intended race and incorrect behavior still matches. Errors: crossed operations, wrong attempts, reversed required order, and unsupported or unbounded conditions cannot silently match. [inferred]
- **R3:** A targeted model regression produces a bounded occurrence witness and separately evaluates its selected correctness obligations using the matched bindings. At least two different witnesses can reference the same definition. Errors: no witness within completed bounds, incomplete search, and violated obligations remain distinguishable; a found witness is never reported as universal verification. [inferred]
- **R4:** A generated live regression requires evidence of occurrence as well as the selected correctness checks, through the existing Case/Run/Contract pipeline. Live and offline evaluation of the same Run agree. Errors: a missed target cannot count as covered, missing evidence or controls cannot count as success, and uncertainty cannot erase a proven violation. [inferred]
- **R5:** The original Nexus reply after retry is a named example with distinct attempt identity and the two documented reply-order variants. Its receiving boundary and allowed outcomes are explicitly resolved in the Model before the regression is claimed complete. Positive, wrong-origin, and wrong-order examples demonstrate recognition, and a controlled live witness proves the declared race occurred. Errors: task replies and asynchronous callbacks are not conflated; attempt count or send-side evidence alone is insufficient, and unavailable evidence is reported as a gap. [inferred]
- **R6:** Documentation provides a catalog entry for each named definition linking its meaning, variants, Properties, and available witnesses; reports reuse the same identities and show occurrence evidence separately from correctness. Errors: broken references are detected, and model-only or unsupported examples cannot appear as live regression coverage. [inferred]
- **R7:** One existing named activity race is expressed using the shared definition mechanism, and existing unrelated model checks and Cases retain their meaning. Errors: documentation edits or declaration reordering cannot change behavior fingerprints, and adding Situation reporting cannot change the underlying transition relation or silently strengthen unrelated Contracts. [inferred]

## Boundaries
<!-- scope: business -->

- Regression testing and documentation/naming are the requested outcomes. [paraphrase]
- Proposed initial scope covers declared Situations, targeted model witnesses, and selected live regressions. Automatic discovery or naming of interesting races, campaign-wide coverage dashboards, and scanning every recorded Run are follow-up possibilities. [inferred]
- No second executable test format, replacement Property language, unrestricted temporal-logic language, or separate behavioral oracle in Go. [inferred]
- No complete Nexus matching model, production retry redesign, exactly-once guarantee, or whole-model visualization project. Reuse related modeling and documentation work when available. [inferred]
- Compositions may supply an explicit owner when supported, but new live lowering for arbitrary compositions is outside this spec. [inferred]

## Decision Context
<!-- scope: both -->

- Naming only Scenarios provides useful regression recipes, but ties the name to how a path is constructed. A shared occurrence definition can describe multiple witnesses and survive changes to the recipe. This motivates the proposed Situation concept. [inferred]
- A dedicated machine remains appropriate when a race needs additional behavioral state. Creating a machine for every named condition would duplicate transitions and obscure reuse. Keep recognition separate unless the distinction affects legal behavior. [inferred]
- Separate occurrence from correctness so a violating execution remains recognizable. This also exposes tests that pass without exercising their intended edge case. [inferred]
- Situation is the proposed term because Case already names a Program and Contract. Confirm the vocabulary when this work is revived; no glossary rename is implied by this capture. [inferred]
- The one-sentence witness Query work (fn-140) owns witness authoring. This spec adds reusable occurrence definitions to that surface rather than introducing another witness syntax. Fault declarations (fn-123), IR/lifter changes, evidence-schema work (fn-148), and Model views (fn-130) are coordination points, not automatically hard prerequisites. [inferred]
- The separately deferred Nexus matching model and bug-finding evidence spec may provide the attempt/reply semantics and controls for the example. This spec owns reusable naming and occurrence recognition; it does not duplicate the matching model's broader scope. [inferred]

## Parked unknowns

- Which receiving boundary does the motivating Nexus reply refer to, and what outcomes are allowed for each ordering? Resolve with the owner against the independent Nexus contract before selecting the live proof example. [inferred]
- Should initial reporting recognize registered Situations opportunistically in arbitrary explored Runs, or remain limited to explicitly targeted regressions? Proposed initial scope is targeted recognition; revisit if broader discovery is central to the intended use. [inferred]
