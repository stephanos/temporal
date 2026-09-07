# fn-73-explicit-environment-binding-for.8 Retire Case 1.0 and the legacy Driver resource mode

## Description
Amend fn-73 and its normative Umpire/Testpilot documentation so exact Case 1.1 is the only admitted and generated format. Migrate every Lean producer, generated fixture, Go fixture, and test helper still constructing Case 1.0. Remove the shared Temporal Driver's legacy physical-resource mode and its namespace, task-queue, and Nexus-endpoint fallback configuration; Profile-owned symbolic bindings become the sole source of physical resource names. Transport connections, credentials, SDK clients, callback authority, HTTP clients, and lifecycle settings remain explicit Driver inputs.

Case 1.1 may have an empty environment only when the Program uses no physical resource. Every resource-bearing role and request field must use a declared symbolic binding, and preparation must reject incomplete, unused, crossed, or literal physical-resource configuration before Driver I/O. Retain a focused negative compatibility test proving Case 1.0 is unsupported; retain no executable 1.0 path.

## Acceptance
- The fn-73 spec, UMPIRE4 normative documents, package documentation, and roadmap describe exact Case 1.1 as the sole supported format and contain no requirement to preserve Case 1.0 or a legacy Driver mode.
- All checked Lean producers, canonical ProtoJSON fixtures, conformance fixtures, and Go test constructors emit exact Case 1.1. Fixture generation and staleness checks pass.
- Prepare accepts exact Case 1.1 only and rejects Case 1.0 and every other version before Driver validation, Open, worker registration, or target effects.
- A resource-free Case 1.1 Program may declare no environment bindings. Any Program using namespaces, task queues, named Nexus endpoints, or resource-bearing request fields declares a complete closed symbolic binding graph and obtains physical values only from the immutable Profile snapshot.
- The shared Temporal Driver exposes no legacy namespace, task-queue, or Nexus-endpoint resource mode, fallback, or mixed-mode branch. Existing transport and lifecycle inputs remain explicit.
- Focused Testpilot and Temporal Driver suites, Lean authoring/codec checks, deterministic fixture checks, the two-environment live Nexus3 selector, make lint-model, and make lint-code pass. The protected workflow remains unchanged.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
