# Checked property examples pilot

## Current admission order (owner override, 2026-10-16)

Native checkpoint .7 is committed; fn-188.3 is complete with formal native-batch review SHIP. Native Duration coverage/guidance follow-up fn-147.5 is also complete (SHIP). The audit of all open specs and remaining milestone tasks found no additional currently authorized native DSL gap; do not invent work or revive deferred proposals. The native sequence below records completed implementation order, not instructions to reopen those tasks. Build/export, IR/capture, schema, Go and backend work remain separately held; neither native closure nor fn-147.5 completion releases them automatically. Temporal remains entirely excluded, including disabled source contents. Native results do not close full parent requirements or deferred integration proofs.

## Native DSL task order (owner override)

Execute native .7 after committed fn-123.10; release its explicit task admission hold then. It builds typed optional illustration attachment and native owner/type/domain/metadata validation using final expect and record-input APIs, independently of .1's IR carriers. Preserve ordinary Property behavior and distinguish hypothetical illustrations from reachability evidence.

Original .1-.6 stay held for schema/IR transport, Go classification, checked documentation/freshness, pilot and final compatibility proofs. Native attachment does not complete those requirements or add a second predicate evaluator. The historical mini pilot stays excluded with all Temporal sources. Finish framework-native DSL before new IR/Go/backend work.

Native DSL changes ship a concise migration recipe in framework-local guidance, not a task-specific Scalafix implementation or copied-Temporal proof. This owner iteration policy supersedes older rule/sweep mandates below. Temporal migration, production rule loading and full integration proofs remain held.

### Native illustration admission contract

Final same-step and transition `expect` values retain typed state/outcome/fact and kind evidence while remaining usable by ordinary `Property[S]` consumers. Attach optional immutable ordered metadata through that typed evidence; do not recover erased types by casts or add a competing predicate language. Existing pattern finishers and explicitly widened Properties remain valid ordinary consumers; blanket attachment on erased values is not required. Preserve the exact predicate, selector and owner when attaching or copying, including the association retained by Queries. Properties without illustrations acquire no new validation/evidence or enumeration duty.

Each illustration retains its explicit positive/counterexample kind, exact nonblank label and explanation, and author-call file/one-based line/column origin distinct from declaration-name provenance. Labels are unique per Property; multiple attachments append. Preserve facts in authored order, including repetitions. No illustration is required and no expected kind is inferred.

Native admission checks the initialized individual machine's actual action binding identity, concrete class/input domains and selected outcome/fact domains, not same-spelled declarations or freshly summoned generic witnesses. A refining individual machine remains its own owner; composed owners reject. An otherwise valid same-machine action outside the Property selector is retained unclassified: the held classifier, not attachment, owns its not-applicable result and expected-classification mismatch.

Validate one supplied state using supported structural and known bounded evidence, including actual Inbox capacity, redelivery/message domains, exact field-channel ownership and existing unordered canonical representation. Do not silently repair malformed values, enumerate the state domain or collection powersets, invent bounds for plain Int/collections, or treat Data comparison as validity. Reuse suitable native evidence left by fn-123.10; explicitly refuse unsupported opaque validity evidence. Attachment never executes predicates, selectors, guards, transitions, starts/end conditions or reachability searches. A valid hypothetical unreachable state remains admissible; attachment proves no classification or reachability result.

The committed durability validator selects Finite evidence before structural recursion and explicitly refuses Inbox resets. It is not a whole-state illustration validator. Validate state and composite record/container structure without acquiring or enumerating their finite catalogs, even when such witnesses are present. Preserve the separate selected-field reset semantics; finite action/input/outcome/fact/message domains and adequate explicit leaf evidence retain their intended roles.

Provide a public typed, individual-machine-owned association between an Inbox state-field coordinate and its exact Channel, independently of any supplied illustration. Retain rooted nested/branch coordinates, matching message types, the actual stable channel receiver and declaration origin; never infer ownership from names, the supplied Inbox, init, effects, lifted schemas or private fixture setup. Acquire metadata after owner initialization and inherit it only through the actual state-preserving machine lineage, retaining the current Property owner. Missing/foreign/conflicting ownership refuses; repeated authored field associations and duplicate channel storage coordinates refuse rather than silently overriding. Preserve already supported structural/branch cases with exact coordinates, without adding an arbitrary selector interpreter. No-illustration machines gain no mandatory association declaration or validation/catalog duty.

Public external controls must admit direct and nested Inbox fields and reject crossed channels with identical message types/configuration, same-S foreign owners, over-capacity values and missing/conflicting evidence. Inspect only the supplied deliveries against the declared channel's selected message domain, redelivery bounds and FIFO/unordered representation; repeated equal unordered entries remain legal. Do not enumerate channel contents or repair supplied values. This bounded ownership and one-value admission seam belongs to native .7; it neither expands durability reset support nor completes channel transport, classification or reachability proofs.

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

The generated document has its own managed directory. The handwritten assessment lives outside that directory. Directory-wide freshness checks must preserve the assessment and other handwritten documentation unchanged.

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
| R2 | Authors can declare optional typed illustrations for both same-step and before/after Properties, and those declarations survive lifting with their Property association, explanation and source location. Errors: wrong-owner actions, incompatible or out-of-domain values, invalid facts or outcomes, missing transition or extraneous same-step before-state, unsupported composed owners, empty or duplicate labels, empty explanations and unspecified or unknown expectations reject at the relevant declaration. | fn-153-checked-property-examples-pilot.1, fn-153-checked-property-examples-pilot.2, fn-153-checked-property-examples-pilot.6, fn-153-checked-property-examples-pilot.7 | — |
| R3 | The gate classifies each illustration with the existing selector and predicate semantics, accepting positive examples only when satisfied and counterexamples only when violated. Errors: unrelated actions produce not applicable without evaluating the predicate; classification mismatches fail with expected and actual results; predicate holes, evaluator or declaration errors cannot pass as negative examples. | fn-153-checked-property-examples-pilot.3, fn-153-checked-property-examples-pilot.6 | — |
| R4 | Documentation and check results distinguish illustrative violations from discovered model counterexamples. A well-typed illustration may be unreachable without changing the model's transition relation. Errors: no illustration is presented as reachable, as proof of a model defect, or as live coverage without the corresponding existing witness or execution evidence; an expected illustrative violation cannot mask an actual model failure. | fn-153-checked-property-examples-pilot.3, fn-153-checked-property-examples-pilot.4, fn-153-checked-property-examples-pilot.5, fn-153-checked-property-examples-pilot.6, fn-153-checked-property-examples-pilot.7 | — |
| R5 | A deterministic generated pilot document presents all three selected Properties with their labels, concrete inputs, explanations and checked classifications. The applicability example explicitly explains a not-applicable action. Errors: stale expected classifications fail the gate, broken associations are rejected, missing, stale or orphaned document output rejects, and generated documentation cannot silently retain an earlier classification after the declaration or predicate changes. | fn-153-checked-property-examples-pilot.4, fn-153-checked-property-examples-pilot.5, fn-153-checked-property-examples-pilot.6 | — |
| R6 | The pilot includes one same-step, one before/after and one applicability-focused Property, each with at least one passing and one failing illustration. The synchronous-success example includes success with a completion fact, success without that fact, and a wrong resulting phase with that fact. Controlled predicate mutations to always true and always false cause the appropriate counterexample and example checks to fail. Errors: the mutation checks must fail for classification mismatch rather than malformed fixtures; existing model behavior and behavioral identities retain the compatibility guarantees above. | fn-153-checked-property-examples-pilot.1, fn-153-checked-property-examples-pilot.3, fn-153-checked-property-examples-pilot.5, fn-153-checked-property-examples-pilot.6, fn-153-checked-property-examples-pilot.7 | — |
| R7 | The pilot concludes with a short assessment of all three authoring examples and their rendered explanations, recording setup burden, duplicated state detail, misleading cases, drift detected by the checks and a recommendation to expand, revise or stop. Errors: missing evidence or absent developer feedback is stated explicitly; successful mechanical checks alone are not reported as proof of improved comprehension. | fn-153-checked-property-examples-pilot.6 | — |

Native tasks contribute only their stated portions. Held tasks retain the complete IR/integration requirements; coverage is assignment, not completion.

## Scope: the mini feature only (owner, 2026-10-11)

Within the Scala DSL batch (`MILESTONES.md`), this spec changes only the framework, the lifter, the shared code and `features/activity_mini`. The real features (standalone activity, Nexus workflow, Nexus standalone and their legacy payloads) are not edited, migrated, lifted or regenerated, and they stay behind until the owner's sweep, which runs this spec's scalafix rules on them. Equivalence compares `FEATURES=activity_mini` against the mini baseline. This overrides any statement in this spec that edits, migrates or proves on a real feature. See fn-166's Scope. The pilot's three Properties are the mini feature's. Task .5's contrast pairs are drawn from the mini feature, not Nexus.

## Redesign impact (fn-173, 2026-10-13)

Owner-approved redesign `fn-173` (reachable exploration; `model/design/reachable-exploration.md`) drops full catalog tables, Scala step tables and equivalence sealing, and Query totals; the Behavior Fingerprint becomes an IR structural hash. For this spec:

- **.1/.3/.6:** validate illustration value types, supported state structure/bounds and selected finite input/outcome/fact domains without whole-state enumeration. Reached-state membership is not an admission condition: R4 explicitly permits well-typed hypothetical unreachable illustrations. `Data` equality/order alone does not establish value-domain validation. Frozen complete transition tables become reached rows for checking, not a catalog of permitted illustrations.
