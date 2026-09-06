# Minimal Nexus3 success demonstration

## Goal & Context

Prove that the readable Nexus3 authoring surface can become functioning Lean and produce a Case that executes against Temporal. Implement exactly one operation: scheduled → started → succeeded. The audience is the engineer evaluating authoring usability; the executable feature file must retain the approachable block-shaped spelling from `Nexus.md`, not expose Umpire's record construction details. No product, deployment, or operational platform changes are needed.

The existing drafts remain the broader design reference. The executable demonstration contains only the success model, one Property, one exact Behavior, and one witness Query. Completing it establishes this slice, not full Nexus3 or cancellation support.

## Architecture & Data Models

Use ordinary Lean vocabulary declarations plus a small Nexus3 command-syntax layer that expands the success-only `model`, `property`, `behavior`, `limits`, and `query` blocks into existing finite-target and Property/Behavior/Query owners. Keep the feature-facing file at most 250 physical lines including its relevant teaching comments; put parsing, elaboration, mechanical construction helpers, and integration in separate Nexus3 modules. The supported grammar is only the demonstrated success slice and must reject unsupported spellings during elaboration; it is not a general Umpire parser or a second semantic language. Capture declaration names from each block declaration and derive IDs rather than repeat them in a registry.

The checked Target starts only at scheduled, with awaitStart and awaitSuccess leading to distinct acknowledged/completed outcomes and the corresponding states. Express successfulResult with the existing transition-contract vocabulary on awaitSuccess; the exact two-step Behavior ensures that occurrence is exercised and is the final step. Plan the witness before lowering it.

The success-only Producer consumes the checked Query, its checked meaning, and selected witness. Reuse the existing async Nexus physical Program mechanics, but derive Case provenance and the acceptance monitor from those inputs. The existing async Case entry point delegates to this Producer. No independently authored copy of the same success requirement remains active.

```mermaid
flowchart LR
  N[Small Nexus3 model] --> Q[Checked Query and witness]
  Q --> I[Nexus3 integration]
  I --> C[Existing Case compiler]
  C --> R[Testpilot Prepare and Run]
```

## API Contracts

- Expose one Case-production result for completion, returning the existing typed lowering error on unsupported or inconsistent input; no partial Case.
- Support only this exact witness form, two-action trace, and checked success Property. Inspect semantic fields, not only IDs or display names. Capture their checked fingerprints in Case metadata.
- IDs use `temporal.nexus3.<kind>.<relative-declaration-name>` by default. Pin `temporal.nexus3.target.lifecycle`; lifecycle-owned role, state, Action, outcome, fact, and relation IDs use `<kind>.lifecycle.<member>`; Property clause IDs use `property.successfulResult.<clause>`; Behavior setup and occurrence IDs use `setup.successfulCompletion.operation` and `occurrence.successfulCompletion.<label>`. Pin the public Property, Behavior, and Query IDs listed in R2. Nested identities use named owners/labels, never source order. An explicit override replaces only that declaration's key; a rename otherwise changes identity. Reject malformed or duplicate effective IDs through existing admission boundaries.
- Retain the existing async-nexus renderer selector, Testpilot Case wire format, Profile/Driver, and test fixture location. Do not add a demo CLI or runtime instruction.
- Physical setup schedules the operation; waits recognize correlated history events. A fixture may cause asynchronous handler completion, but only recorded history establishes the modeled result.

## Edge Cases & Constraints

The monitor must correlate scheduled, started, and completed history for the same workflow/run and operation using the existing scheduled-event/request references. Missing, duplicate-only, or foreign-operation evidence never satisfies the Contract. Execution deadlines remain operational bounds: do not turn a model transition count into milliseconds or add a stronger temporal Property; unresolved evidence closes inconclusive.

Cancellation, operation-scoped liveness, and other Query/Property forms are unsupported by this Producer and reject before Case publication or Driver I/O. Keep those examples as inert design documentation. The completion Query carries checked `capability-contract` Known Gaps for cancellation and operation-scoped progress; Task 2 converts them once with `KnownGapSet.toCaseKnownGaps` into Case metadata. They disclose this slice's limitations without weakening or waiving its success requirement.

Fixture generation owns the existing two Temporal example fixtures separately from the fixed six conformance classes. Reuse the current fixture tool and transactional publisher with a separate example manifest/root; build and validate the complete owned tree before comparison or publication. Ordinary Go tests consume checked-in bytes and never invoke Lean. No broad API drift verification or CI changes.

## Acceptance Criteria

- **R1:** The compact executable authoring file builds and visibly retains the approachable `model lifecycle`, `property successfulResult`, `behavior successfulCompletion`, `limits shortTrace`, and `query completion` blocks from `Nexus.md`, narrowed only to scheduled → started → succeeded. Their expansion uses the existing language owners and planning finds exactly that trace through awaitStart/awaitSuccess. Parsing, elaboration, and mechanical construction stay outside the feature file. Errors: an unsupported block form, undeclared result, outgoing terminal row, impossible Behavior, or missing success step cannot publish a successful witness; no placeholder proofs or unchecked extraction fallback.
- **R2:** Actual declaration names supply derived IDs automatically, with an optional per-declaration override and no hand-maintained registry or duplicated name strings. Test exact IDs for lifecycle and every success-slice role, state, Action, outcome, fact, relation, Property clause, Behavior setup/occurrence, plus `temporal.nexus3.property.successfulResult`, `temporal.nexus3.behavior.successfulCompletion`, and `temporal.nexus3.query.completion`. Test rename versus override behavior, reorder/comment stability, and same-ID semantic fingerprint changes. Errors: invalid overrides and a real derived-versus-overridden collision inside the command-authored surface reject at admission.
- **R3:** The Producer lowers checked success semantics into the current Program/Contract format, with source IDs/fingerprints bound to its inputs. Changing/removing a success transition or changing its Property under the same IDs changes the output meaning or rejects; returning the existing constant Case is insufficient. Errors: wrong Target/trace, absent witness, altered unsupported Property/Action/form, or cancellation produces a typed error and no Case. Missing/foreign evidence cannot satisfy the generated monitor; model-step bounds are never converted into runtime timeouts.
- **R4:** The existing async-nexus fixture is generated deterministically from Nexus3 and the existing renderer delegates to the new Producer. Repeated generation yields identical bytes; the example fixture tree is compared/published transactionally and the six conformance classes are unchanged. Errors: renderer/decoder failure, stale bytes, or incomplete generated trees fail without publishing partial output; ordinary tests perform no generation.
- **R5:** The existing live async Nexus test uses that fixture through Testpilot and the real local Temporal Driver. It returns completed disposition, satisfied Verdict, and successful cleanup, supported by the three correctly correlated history events. Missing or foreign terminal evidence fails a focused offline monitor test. Preserve the focused live test as a required pass, not an allowed inherited failure; update the existing demo documentation with exact support and commands. No error surface beyond the admission/evidence/runtime failures named above.

## Boundaries

No cancellation implementation, retries, multi-operation composition, scoped-step monitor, general-purpose DSL/parser beyond the five success-slice blocks, universal Property compiler, new runtime opcode, ID registry, new executable, new artifact format, deployment, broad cleanup, benchmarks, or CI expansion. No migrations of the original Nexus/Nexus2 models.

## Decision Context

- Three sequential tasks: executable authoring, checked lowering, then existing fixture/live-test integration. Avoid a framework project around a single demonstration.
- Preserve the five-block Nexus3 spelling as the demonstrated public surface. Implement only its success forms and expand them into the existing checked owners; avoid a reusable grammar framework until another model requires it.
- Keep integration co-located in Nexus3 as explicitly requested, with imports flowing Integration → Nexus only; do not relax import lint or make the pure model import runtime bindings.
- Preserve the complete Markdown drafts while creating the narrower executable Lean modules alongside them.
- fn-67 supplies design provenance; completed fn-62 and fn-64 supply authoring and runtime dependencies. Independent fn-66 cleanup is not a blocker.
- Broad generated API drift verification and CI expansion remain outside this demonstration, consistent with the existing declined-scope decision.

## Quick commands

```sh
(cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-testpilot)
mise exec -- go test -count=1 -tags test_dep ./tests/testcore/testpilot/...
mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotAsyncNexusCase$'
```

The Go commands run from the repository root. Task acceptance records fixture generation/check commands using the existing tool. Final verification also runs the existing model and lint gates, with unrelated failures reported separately.

## Early proof point

Task 1 must produce the exact checked witness from the compact block-shaped authoring file before runtime extraction begins. If the syntax layer cannot keep that file close to the successful subset of `Nexus.md` within the size bound, simplify the elaboration seam before Task 2; do not expose record assembly in the feature file.

## Requirement coverage

| Req | Task(s) | Gap justification |
| --- | --- | --- |
| R1 | fn-68-minimal-nexus3-success-demonstration.1 | — |
| R2 | fn-68-minimal-nexus3-success-demonstration.1 | — |
| R3 | fn-68-minimal-nexus3-success-demonstration.2 | — |
| R4 | fn-68-minimal-nexus3-success-demonstration.3 | — |
| R5 | fn-68-minimal-nexus3-success-demonstration.3 | — |

## References

UMPIRE4 checked authoring, closed Case, and evidence contracts; Lean Authoring Guidelines; fn-67 design decisions; completed fn-62 authoring and fn-64 Case Runtime work. Concrete source anchors and commands belong to the child tasks.
