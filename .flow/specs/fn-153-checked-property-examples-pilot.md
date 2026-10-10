# Checked property examples pilot

> HTML render lens (local): open `.flow/artifacts/fn-153-checked-property-examples-pilot/spec.html` - regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 1): "explore the idea of allow to define examples and counterexamples for a property"
>
> user (turn 1): "the idea is that we could illustrate what the property means better to a developer with an example."
>
> user (turn 1): "investigate how that would work and if that idea is good"
>
> user (turn 2): "write a flow next spec for the pilot and add it it MILESTONES.md (at the end)"

## Goal & Context
<!-- scope: business -->
<!-- Source: developer-facing examples and the pilot request are paraphrased from Conversation Evidence. Local repo, spec, memory and gap analysis confirmed the pilot design during planning. -->

Help a developer understand what a Property promises by reading a concrete example and a closely related counterexample beside it. Run a small pilot to determine whether these illustrations explain the promise without requiring so much setup that they obscure it.

The proposed pilot makes illustrations checked documentation. Each illustration names its expected classification, and the existing Property evaluator checks that classification. A failing illustration can expose an accidentally weakened predicate as well as stale documentation. These checks establish what the Property accepts or rejects; model verification and live execution retain their separate meanings.

## Architecture & Data Models
<!-- scope: technical -->

An illustration belongs to one Property and carries an explanatory label, an action class, a concrete resulting step, and an expected classification. The step contains its outcome, resulting state and facts. A transition Property also requires the before-state. All values are checked against the owning machine's domains. Illustrations are optional and colocated with their Property.

The pilot supports same-step and before/after Properties on an individual machine. Its three existing Properties are Nexus workflow `syncSucceeds`, standalone Nexus's expanded `closedRejectsOrRepeats` transition promise, and Nexus workflow `completionSucceeds` for the applicability boundary. Preserve their predicates and resolved ownership when the implementation baseline changes. These choices explain synchronous success, preservation of a closed before-state, and completion-event recording without an implied phase promise.

The authoring layer declares illustrations, the lifter carries them through the Model IR, and the Go checker validates and classifies them using the existing Property selector and predicate semantics. The model gate checks their expected classifications. A deterministic generated document presents the Property, its illustrations, their explanations and their checked classifications together. A compact document is sufficient for this pilot; the broader model-view feature is not required.

An authored counterexample is an illustrative violation. It may describe a well-typed step the model never permits. Its classification alone makes no reachability claim. A discovered model counterexample remains a violating execution supported by a replayable witness. Existing witness references may supplement the documentation where supported, but the pilot introduces no new witness-search or trace-authoring language.

## API Contracts
<!-- scope: technical -->

Authors can attach a positive example or an illustrative counterexample to a Property, with a non-empty explanation and concrete values. The exact builder spelling is settled against the authoring API present when implementation starts. An illustration must preserve its association with the Property and its source location through lifting.

Attach illustrations while the builder still retains the machine's state, outcome and fact types, or retain equivalent typed evidence in the completed value. Do not require casts or string-based authoring. Labels are non-empty and unique within a Property; whitespace-only explanations and unspecified or unknown expectation values reject. Same-step illustrations omit a before-state; transition illustrations require it. Composed-owner illustrations reject explicitly. An individual machine's existing refinement does not exclude it from the pilot.

Classification applies the Property's action selector first. A selected step is satisfied or violated according to the existing predicate. A non-selected step is not applicable and must not read the predicate. Malformed data and evaluation errors remain errors, never a false predicate or a successful negative example.

A positive example checks successfully only when it is applicable and satisfied. A counterexample checks successfully only when it is applicable and violated. The gate reports a mismatch with the Property, illustration label, source location, expected classification and actual classification or error. An unrelated action is rejected as a mislabeled positive or negative illustration. Its not-applicable result is also shown in the pilot's applicability explanation.

Illustrations never add transitions, generate live Cases automatically, or modify the Property's predicate. They use the existing interpreter, without a second evaluator in the authoring language. Documentation edits do not change Definition IDs or Behavior Fingerprints. Illustration changes may change their own check results and generated documentation, while existing behavioral answers remain unchanged.

The checker binds a Property directly, including one without a Query, through the existing Property reads. It checks supplied values against actual owning domains before evaluation. Binding and evaluation preserve existing sequential-use restrictions. Predicate holes and selector or predicate errors fail the illustration with nested diagnostics and its source position.

The applicability explanation derives a separate unrelated-action probe using this same classifier. That probe is presentation evidence, not a third authored expectation kind. The compact document escapes author text, uses stable declaration order and labels every hypothetical input with reachability unclaimed. A narrow stdout-only documentation entry point feeds the gate's transient output; the gate alone installs or compares the managed document. Checks recompute from current declarations and predicates before publication, and a mismatch fails rather than publishing successful-looking stale output.

## Edge Cases & Constraints
<!-- scope: technical -->

- Wrong-machine actions, values outside declared domains, invalid facts or outcomes, missing before-states, and empty explanations are located declaration errors. A missing or malformed value must never serve as an expected violation.
- Applicability is independent of truth. An unrelated action cannot satisfy an illustration through a vacuous implication and cannot count as a counterexample.
- Typed but unreachable states or steps are valid illustration inputs. Generated documentation labels them as illustrations with reachability unclaimed unless an existing replayable witness establishes that claim.
- A counterexample's expected violation is a successful illustration check. It must not be aggregated as an actual model failure. Conversely, expected illustration failure must not suppress a genuine model-checking failure.
- Existing Properties without illustrations remain valid. Illustrations are not required for every Property, and a finite example set never proves completeness or universal correctness.
- Existing model answers, transition tables, Definition IDs, Behavior Fingerprints and generated Case behavior remain unchanged. Review regenerated artifacts and account for any carrier bytes or whole-artifact digests that change when illustration metadata is added; do not claim that all serialized artifacts must remain byte-identical.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Developers can read named examples and counterexamples alongside the Property they explain in the pilot documentation. Each explanation identifies the detail that makes the illustration informative. Errors: presentation and classification failures follow R2 through R5. [paraphrase]
- **R2:** Authors can declare optional typed illustrations for both same-step and before/after Properties, and those declarations survive lifting with their Property association, explanation and source location. Errors: wrong-owner actions, incompatible or out-of-domain values, invalid facts or outcomes, missing transition or extraneous same-step before-state, unsupported composed owners, empty or duplicate labels, empty explanations and unspecified or unknown expectations reject at the relevant declaration.
- **R3:** The gate classifies each illustration with the existing selector and predicate semantics, accepting positive examples only when satisfied and counterexamples only when violated. Errors: unrelated actions produce not applicable without evaluating the predicate; classification mismatches fail with expected and actual results; predicate holes, evaluator or declaration errors cannot pass as negative examples.
- **R4:** Documentation and check results distinguish illustrative violations from discovered model counterexamples. A well-typed illustration may be unreachable without changing the model's transition relation. Errors: no illustration is presented as reachable, as proof of a model defect, or as live coverage without the corresponding existing witness or execution evidence; an expected illustrative violation cannot mask an actual model failure.
- **R5:** A deterministic generated pilot document presents all three selected Properties with their labels, concrete inputs, explanations and checked classifications. The applicability example explicitly explains a not-applicable action. Errors: stale expected classifications fail the gate, broken associations are rejected, missing, stale or orphaned document output rejects, and generated documentation cannot silently retain an earlier classification after the declaration or predicate changes.
- **R6:** The pilot includes one same-step, one before/after and one applicability-focused Property, each with at least one passing and one failing illustration. The synchronous-success example includes success with a completion fact, success without that fact, and a wrong resulting phase with that fact. Controlled predicate mutations to always true and always false cause the appropriate counterexample and example checks to fail. Errors: the mutation checks must fail for classification mismatch rather than malformed fixtures; existing model behavior and behavioral identities retain the compatibility guarantees above.
- **R7:** The pilot concludes with a short assessment of all three authoring examples and their rendered explanations, recording setup burden, duplicated state detail, misleading cases, drift detected by the checks and a recommendation to expand, revise or stop. Errors: missing evidence or absent developer feedback is stated explicitly; successful mechanical checks alone are not reported as proof of improved comprehension.

## Early proof point

Tasks .1 through .3 prove that typed optional metadata survives lifting, passes owning-domain admission and reaches the existing selector and predicate without a Query or another evaluator. If that vertical proof fails, reconsider the attachment and binding seam before the renderer and pilot tasks proceed.

## Quick commands

```bash
go test -tags test_dep ./tools/umpire/ir -run 'Property|Illustration|Origin|Identity'
go test -tags test_dep ./tools/umpire/check -run 'Property|Illustration'
make lint-model
```

At closure, serialize regeneration and full verification under the shared heavy-gate lock. Use the milestone canonical Go suite with `-json -tags test_dep -p 2 -timeout 30m`, model gate with `MODEL_GATE_ARGS=--skip-go-checks`, managed Cases/fixtures, lint and dependency checks. Retain exit codes and separate wall times. The pilot requires no new live Case run because its contract preserves behavior and Case contents.

## Boundaries
<!-- scope: business -->

- Deliver the small property-example pilot proposed in the preceding discussion. [paraphrase]
- Limit initial support to individual-machine same-step and before/after Properties. Temporal sequences, bounded-liveness illustrations, composed-machine illustrations and a repository-wide rollout are outside this pilot.
- No automatic example discovery, general mutation-testing framework, new live-test format, new Property language, or changes to the modeled product behavior. The controlled mutations serve only the pilot's verification.
- No interactive viewer or full model-visualization project. Existing action-input examples retain their separate abstraction-claim meaning.

## Decision Context
<!-- scope: both -->

- Checked documentation is the proposed approach because prose alone can drift, while witness-only examples cannot illustrate forbidden behavior when a model satisfies the Property throughout the explored scope.
- Keep the Property predicate authoritative. An illustration asserts what that predicate should say about one input; it supplies no alternate behavioral rule. Contrast pairs should change one relevant detail whenever practical.
- A not-applicable step is a third classification, distinct from satisfying and violating the Property. The pilot explains it without requiring a third authoring declaration kind.
- Coordinate with fn-140's witness authoring, fn-149's Property organization, fn-141's constructed-value export and fn-130's model views. Use the APIs that exist when this pilot starts; those efforts are coordination points rather than feature prerequisites. The named-Situation proposal in fn-152 is not a prerequisite.
- Schedule this pilot after Batch 5's closure to preserve the existing delivery chain and separate its schema/source/identity baseline. This is a scheduling decision under the owner's recommendation mandate, not an invented feature dependency. Each task re-anchors to the compiler, claim organization, lifter and schema left by that chain.
- Keep hypothetical violations visibly separate from found counterexamples. Requiring every negative illustration to replay as legal model behavior would prevent illustrating promises the model actually keeps.
- Preserve empty-extension canonical bytes, independently validate each illustration and retain exact nested source diagnostics. Local memory and the existing evaluator support these obligations; broad generated-API drift verification remains outside this focused check.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Developers can read named examples and counterexamples alongside the Property they explain in the pilot documentation. Each explanation identifies the detail that makes the illustration informative. Errors: presentation and classification failures follow R2 through R5. | fn-153-checked-property-examples-pilot.4, fn-153-checked-property-examples-pilot.5 | — |
| R2 | Authors can declare optional typed illustrations for both same-step and before/after Properties, and those declarations survive lifting with their Property association, explanation and source location. Errors: wrong-owner actions, incompatible or out-of-domain values, invalid facts or outcomes, missing transition or extraneous same-step before-state, unsupported composed owners, empty or duplicate labels, empty explanations and unspecified or unknown expectations reject at the relevant declaration. | fn-153-checked-property-examples-pilot.1, fn-153-checked-property-examples-pilot.2, fn-153-checked-property-examples-pilot.6 | — |
| R3 | The gate classifies each illustration with the existing selector and predicate semantics, accepting positive examples only when satisfied and counterexamples only when violated. Errors: unrelated actions produce not applicable without evaluating the predicate; classification mismatches fail with expected and actual results; predicate holes, evaluator or declaration errors cannot pass as negative examples. | fn-153-checked-property-examples-pilot.3, fn-153-checked-property-examples-pilot.6 | — |
| R4 | Documentation and check results distinguish illustrative violations from discovered model counterexamples. A well-typed illustration may be unreachable without changing the model's transition relation. Errors: no illustration is presented as reachable, as proof of a model defect, or as live coverage without the corresponding existing witness or execution evidence; an expected illustrative violation cannot mask an actual model failure. | fn-153-checked-property-examples-pilot.3, fn-153-checked-property-examples-pilot.4, fn-153-checked-property-examples-pilot.5, fn-153-checked-property-examples-pilot.6 | — |
| R5 | A deterministic generated pilot document presents all three selected Properties with their labels, concrete inputs, explanations and checked classifications. The applicability example explicitly explains a not-applicable action. Errors: stale expected classifications fail the gate, broken associations are rejected, missing, stale or orphaned document output rejects, and generated documentation cannot silently retain an earlier classification after the declaration or predicate changes. | fn-153-checked-property-examples-pilot.4, fn-153-checked-property-examples-pilot.5, fn-153-checked-property-examples-pilot.6 | — |
| R6 | The pilot includes one same-step, one before/after and one applicability-focused Property, each with at least one passing and one failing illustration. The synchronous-success example includes success with a completion fact, success without that fact, and a wrong resulting phase with that fact. Controlled predicate mutations to always true and always false cause the appropriate counterexample and example checks to fail. Errors: the mutation checks must fail for classification mismatch rather than malformed fixtures; existing model behavior and behavioral identities retain the compatibility guarantees above. | fn-153-checked-property-examples-pilot.1, fn-153-checked-property-examples-pilot.3, fn-153-checked-property-examples-pilot.5, fn-153-checked-property-examples-pilot.6 | — |
| R7 | The pilot concludes with a short assessment of all three authoring examples and their rendered explanations, recording setup burden, duplicated state detail, misleading cases, drift detected by the checks and a recommendation to expand, revise or stop. Errors: missing evidence or absent developer feedback is stated explicitly; successful mechanical checks alone are not reported as proof of improved comprehension. | fn-153-checked-property-examples-pilot.6 | — |
