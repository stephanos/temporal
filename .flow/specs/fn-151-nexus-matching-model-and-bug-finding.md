# Nexus matching model and bug-finding evidence

**Deferred** by the owner on 2026-10-08, before task planning. Not ready for execution; activation requires revisiting the proposed scope and then planning the work. [paraphrase]

## Goal & Context
<!-- scope: business -->

Use Nexus-related matching behavior as a bounded pilot for showing the team that an independent Model can make bugs harder to encode or easier to detect. Smaller code, denser notation, faster execution, and better abstractions are useful, but are not sufficient evidence of correctness value. [paraphrase]

Produce a reviewable behavioral contract and reproducible bug-finding evidence, not a second implementation of matching. Compare the methods on equivalent cases and report limitations as well as successes. [inferred]

## Architecture & Data Models
<!-- scope: technical -->

Source: [inferred] — proposed technical scope from the discussion, to be reviewed when revived.

Model the Nexus dispatch protocol at the boundary between a caller, matching partitions, and worker pollers. Preserve request/attempt identity, waiting and matched requests, response ownership, deadlines, cancellation, routing choices, and forwarding outcomes. Distinguish worker execution, matching's acceptance of a reply, and the caller's observation of that reply.

Use an independent contract with explicit safety Properties and bounded-progress assumptions. Model the distinctions needed to expose bugs, not production maps, channels, locks, goroutine layout, or scheduling algorithms. Current Nexus matching has no persisted task backlog; durable custody and acknowledgement-cursor semantics must not be inherited from an activity-oriented task-queue model.

Start with local dispatch and worker response, then include one forwarding hop and route changes while dispatch waits. Cover both classic and priority matcher realizations against the same applicable contract, with any genuine semantic differences declared explicitly. Initial exploration bounds should allow at least two concurrent requests, two pollers, and two partitions, with small routing-version and fault domains; final state-space and step limits are recorded before evaluation.

Carry executable cases through the existing Model → IR → lowering → Testpilot path into real Go matching behavior. Controls and observations may require generic test hooks, but the behavioral oracle remains in the Model. Reuse existing Nexus abstractions where they fit without importing the complete operation lifecycle.

## API Contracts
<!-- scope: technical -->

Source: [inferred]. No production public API change is required.

The modeled interaction surface is dispatch, worker polling, successful or failed worker response, poll cancellation, caller cancellation/deadline expiry, forwarding, route change, and partition shutdown. Correlation distinguishes attempts even when the logical request is retried.

A reply can satisfy only its corresponding live dispatch. Duplicate or unknown response identities cannot consume another dispatch's result. An accepted reply is not, by itself, proof that the caller received it: caller timeout may race reply acceptance. A timeout or cancellation is not proof that the worker never started or stopped executing. No exactly-once execution guarantee is introduced.

Realization reports distinguish a contract violation, evidence consistent with the contract, and an inconclusive run caused by missing controls, observations, or exceeded bounds. Unsupported schedules and missing evidence must not silently count as passing cases.

## Edge Cases & Constraints
<!-- scope: technical -->

Source: [inferred].

- Completion and failure race each other, cancellation, and deadline expiry; late or duplicate replies arrive after pending state is removed.
- Two concurrent attempts must not exchange replies; a retry does not reuse an earlier attempt's response authority.
- A poller cancels before or during matching; no poller is available; a matched worker never responds.
- Forwarding succeeds, fails, is throttled, or becomes ambiguous after a lost response. Route changes and partition shutdown interleave with a waiting dispatch. Do not turn uncertain remote execution into an exactly-once claim.
- Progress claims state scheduler fairness, worker availability, response-delivery assumptions, finite fault budgets, and search bounds. Permanent worker/network failure is not mislabeled as a progress violation under assumptions that require eventual availability.
- New defects are not promised. Mutation results, historical regressions, model-only counterexamples, and confirmed implementation bugs are reported separately.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** An independently authored bounded Model covers local Nexus dispatch, polling, matching, completion, and failure for concurrent requests, with named correlation and response-ownership Properties. Errors: unknown identities, duplicate replies, and crossed-attempt replies are exercised; rejected replies cannot alter an unrelated request. [inferred]
- **R2:** Executable cases cover reply acceptance versus caller observation, cancellation, deadline expiry, and nonresponding workers. Errors: late responses cannot resurrect a completed dispatch; timeout/cancellation do not assert worker nonexecution or exactly-once execution; ambiguous outcomes remain explicit. [inferred]
- **R3:** The same contract covers one forwarding hop, route changes while waiting, poll cancellation, and partition shutdown for classic and priority matcher realizations. Errors: forwarding failure, throttling, response loss, and rerouting are exercised without adding a persisted Nexus backlog or silently strengthening delivery guarantees. [inferred]
- **R4:** Every claimed safety or bounded-progress result names the explored domains, step limits, fault budget, and environmental assumptions, with a reproducible run. Errors: exhausted bounds, unavailable scheduling controls, and unmet progress assumptions produce a labeled limitation or inconclusive result, not a claim of exhaustive correctness. [inferred]
- **R5:** Generated cases run against real Go matching through the existing IR/lowering/Testpilot pipeline, with trace evidence sufficient to check each claimed distinction. Errors: missing hooks or observations are explicit gaps; a hand-written scenario oracle in Go cannot substitute for the Model's contract. [inferred]
- **R6:** A reproducible evaluation challenges the frozen contract with independently selected historical defects or seeded implementation defects spanning correlation, cancellation/deadline handling, and forwarding/routing. Report detections, misses, false positives, runtime and authoring/maintenance effort, alongside the existing test-suite baseline on the same defects and comparable workloads. Include representative invalid Model encodings and show whether authoring constraints, model checking, or live conformance detect them. Errors: equivalent/unreachable mutants, unsupported cases, and post-challenge Model changes are disclosed; absence of improvement remains a valid reported outcome, not evidence of success. [inferred]
- **R7:** A team-facing demonstration ties every claimed benefit to a reproducible result and a named detection method, includes counterexample/trace explanations, and states what remains unmodeled. Errors: no claim that compactness proves correctness, that bounded exploration proves unbounded behavior, or that finding seeded defects proves discovery of new production bugs. No additional error surface beyond evidence accuracy. [inferred]

## Boundaries
<!-- scope: business -->

Focus on the Nexus-related subset of matching, rather than modeling most of matching at once. [paraphrase]

The following exclusions are proposed scope boundaries, not separately approved implementation decisions: [inferred]

- No endpoint CRUD, endpoint-cache consistency, persistence ownership, or endpoint table-version model.
- No workflow/activity backlog persistence, acknowledgement cursors, backlog GC, or full matching fairness/performance model.
- No complete worker deployment/versioning system; retain only routing changes that affect the selected Nexus dispatch protocol.
- No duplicate model of end-to-end asynchronous Nexus callbacks and history lifecycle; operation-level retries are environment actions unless needed to distinguish dispatch attempts.
- No production matching rewrite, new public API, or claim to model every internal interleaving.

## Decision Context
<!-- scope: both -->

### Motivation

The team needs evidence that modeling prevents mistakes or exposes bugs more effectively, not merely that models are shorter or more abstract. Nexus narrows the investigation to a more tractable slice of matching. [paraphrase]

### Technical choices

Use a protocol model plus real-implementation conformance, with independent defect challenges to reduce the risk that Model and implementation share the same mistaken assumption. Reject a line-for-line matching reimplementation: it increases coupling without establishing an independent oracle. Reject model-only green checks as sufficient evidence of implementation correctness. [inferred]

### Activation

Keep this work deferred and unplanned for now. [paraphrase] When revived, recheck the current Nexus dispatch semantics, available hooks, and ongoing framework changes before choosing tasks. No delivery estimate or dependency chain is committed by this capture. [inferred]

## Conversation Evidence

- [user] "I see the model has some nice beneifts: shorter, denser, faster, better abstractions"
- [user] "we need to prove that the model makes it harder to encode bugs / easier to spot them through various methods. and sell that to the team."
- [user] "how dificult you reckon would it be to model most of the behavor in matching/ in detail? so an extent that it's helpful to find bugs?"
- [user] "what if we focused on the parts in matching related to Nexus only?"
- [user] "let's write a flow next spec; but mark it \"deferred\" for now and add to milestones.md"
