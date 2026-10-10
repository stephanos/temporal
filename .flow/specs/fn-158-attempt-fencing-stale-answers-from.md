# Attempt fencing: stale answers from workers the service gave up on

## Conversation Evidence

> user (turn 1, part 1): "it looks like we're modelling the state machine as an "all knowing machine" where it can observe the global state and "nkow" what's going on in the worker etc. that makes sense."
> user (turn 1, part 2): "just wondering, is that the best way to model a distributed system? are there other approaches in other tools that have a lot of merit?"
> user (turn 1, part 3): "we're looking for the best way to represent this behavior across systems/services and time so that humands can reason about it best and we can use automated checks to catch issues."
> user (turn 1, part 4): "in particular I'm also wondering if there are great abstractions to help mdoel these faster/cheaper/with less code"
> user (turn 1, part 5): "for instance, does the model represent the "service retries task because it lost connection with worker, but worker still works on the task" case at all? well?"
> user (turn 1, part 6): "these edge cases are where all the interesting bugs are usually"
> user (turn 2): "yes, prototype (in a separate git worktree); if it seems helpful, create a new flow next spec"
> user (turn 3): "are we modelling a solution just for standalone activities or general?"
> user (turn 4): "also think about whether we have enoguh suface area in the model with satandalone activities to find generalizable patterns. I think drawing in workflows or schedules is too large. and would kill our momentum. so I hope Nexus and activiries are sufficient?"
> user (turn 5, selected): "proceed-anyway"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 60% [paraphrase], 40% [inferred] -->

Edge cases at the boundary between services are where the interesting bugs live: in particular, the service gives up on an attempt and retries it while the worker that holds the attempt keeps working and answers late. The current Models cannot state that case. Their worker has no state of its own and acts on the service's phase, and an answer carries no attempt identity, so a late answer from a superseded attempt is indistinguishable from the current attempt's. [paraphrase]

The goal is a representation that humans can reason about across services and time, that automated checks can hold to, and that is cheaper to write than one hand-built machine per race. The observer-level Product machine stays as it is: what a client reads is rightly an all-knowing view. The change is one level down, in the System: each party acts on what it alone knows, and what crosses between parties travels as messages that can be late, reordered or lost. [paraphrase]

The shared building block is attempt fencing: an owner hands out numbered tokens, holders answer with the token they hold, and the owner settles an answer only when its token names the current holding. It is general rather than activity-specific. Standalone activity attempts, Nexus task retries and Nexus completions addressed to a reset caller run all have this shape. Deferred specs fn-151 (Nexus matching) and fn-152 (named edge-case situations) are its expected consumers. [inferred]

## Architecture & Data Models
<!-- scope: technical -->
<!-- Source: [inferred] from the prototype on branch agent/attempt-fencing-prototype. -->

### What the prototype established

A focused System subject for the standalone activity, outside the main System machine, with the corrected design and its deliberately faulty negative control:

- **State.** The service's record (phase, current attempt, the attempt whose heartbeat last reached it, the attempt whose answer completed it), what workers hold (one holding flag per attempt), and a bounded wire of answers in flight. Two attempts, a wire of capacity two.
- **Locality.** A worker's answer is enabled by what the worker holds alone, never by the service's phase. The service's dispatch and deadlines never read what the worker holds. Giving up an attempt (start-to-close) does not tell the worker.
- **Wire.** Unordered, lossy and bounded. Delivery is a separate step from sending, so an answer sent before a deadline can arrive after the retry is dispatched.
- **Fence.** Delivery settles an answer only when its token names the current, held attempt; otherwise it is rejected as not found. This mirrors the server's task-token validation (attempt count, then a stamp).
- **Claims.** Two state invariants: a completed activity was completed by its current attempt, and only the current attempt's heartbeat reaches it.

Results under the Go checker: the corrected design verifies both claims, including an unscripted six-step search over every interleaving (78 product states, about a second). The negative control yields counterexamples for both. The free search found a race nobody wrote: the worker sends its completion before the deadline, the message is in flight while the retry is dispatched, and it lands on attempt two. Both designs refine the Product machine, which shows the observer-level spec cannot see this class of bug: the fencing obligation must be stated at the System level.

### What generalizes

Fencing as one declared unit with three parts: an owner's epoch, holders' tokens, and the accept-only-current rule at delivery. Its Properties are produced by the unit, not restated per subject. Each subject binds its own owner, holder, answer kinds and the event that supersedes a token (a deadline retry, a reset, a run reset). The unit composes with an existing System through the existing composition and synchronization machinery rather than by adding fields to each System's state.

### Framework gaps the prototype hit

- A channel must be declared at the feature level, because the state type's catalog reads it while the machine object is being constructed. The placement lint previously required channels inside the machine's monitors section, which made channels unusable in any feature file; the prototype relaxes the lint to accept a feature-level channel.
- The lifter cannot count a Query's total for a state that holds a channel, so the prototype states its totals by hand.
- A Scenario names an action class only by constant values, so a channel's message type must be an enumeration, not a record.

## API Contracts
<!-- scope: technical -->

No production API changes. [inferred] The modeled surface is: dispatch of an attempt to a worker; a worker's completion and heartbeat answers, each naming a token; delivery and loss of an answer; the deadline that supersedes an attempt; reset of an activity; and, for Nexus, the task retry and the completion delivered to a caller run. A stale answer is answered as not found, matching the server's token validation. [inferred]

## Edge Cases & Constraints
<!-- scope: technical -->

- **Token reuse after reset.** A reset returns the attempt count to zero. Where either side's stamp is zero (legacy compatibility), the server compares the attempt count alone, so a pre-reset token whose count matches a post-reset attempt may be accepted. The fencing unit must model the supersede-by-reset event so this can be confirmed or refuted. [inferred]
- **By-ID answers** name no attempt and bypass the attempt check by design; they are outside the fence and must stay distinguishable from token answers. [inferred]
- **State space.** Two attempts, two in-flight answers and one zombie holder keep checks to seconds; bounds must be stated with each Query. [inferred]
- **Locality is a review rule, not yet a checked one.** Nothing stops a holder's rule from reading the owner's state. [inferred]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Fencing is declared once, as a reusable unit with an owner's epoch, holders' tokens and the accept-only-current rule, and is applied to at least two subjects: standalone activity attempts and one Nexus subject. Each application gets its "settled by current token" Properties from the unit without restating them. [paraphrase] Errors: an application that binds no supersede event, or no answer kind, is refused at declaration.
- **R2:** The standalone activity's zombie-worker paths are checked: a late completion and a late heartbeat from a superseded attempt, over a lossy, unordered wire. The corrected design verifies over an unscripted bounded search, and the negative control yields a counterexample for each claim. [paraphrase] Errors: no error surface beyond a search past its stated bounds, which reports as not exhaustive.
- **R3:** A reset supersedes outstanding tokens: a Query checks that an answer naming a pre-reset attempt is never settled after the reset, and the legacy-stamp path is modeled so the result either holds or yields a counterexample recorded as a finding. [inferred] Errors: no error surface beyond R2's.
- **R4:** One Nexus stale-answer race is expressed with the same unit: the original reply arriving after the task's retry, or a completion delivered to a superseded caller run. [paraphrase] Errors: no error surface beyond R2's.
- **R5:** A channel can be used in a feature file with no hand-written workaround: the placement lint accepts a feature-level channel, and the lifter computes the Query total of a state that holds one. [inferred] Errors: a channel declared inside a machine object is refused with its line and the place it belongs.
- **R6:** At least one zombie-path witness is lowered to a Case and run against a server, which answers the stale completion as not found while the current attempt stays open. [inferred] Errors: a Run that never reaches the superseded state is reported as not exercising the path, not as passing.
- **R7:** The spec's closing report compares the fencing unit's size and checking time with the existing hand-built race machines for the same races, as evidence for or against "faster, cheaper, less code". [paraphrase] Errors: no error surface.

## Boundaries
<!-- scope: business -->

- Workflows and schedules are out of scope; the pattern is found and proven on standalone activities and Nexus only. [paraphrase]
- The Product machine keeps its observer-level view; this spec changes only the System level. [paraphrase]
- Replacing the existing hand-built race machines is out of scope; R7 compares against them. [inferred]

## Decision Context
<!-- scope: both -->

### Motivation

Momentum beats breadth: drawing in workflows or schedules would be too large, so the abstraction is shaped on the two features already modeled. [paraphrase] Activities and Nexus are expected to be enough surface: between them they carry the three distinct supersede events (a deadline retry, a reset, a caller run reset) and both directions of staleness (a late answer to the owner, and a stale dispatch to a holder, which the activity admission races already model). [inferred]

Rejected: adding attempt identity to the main activity System's state directly. It would grow every Query's state space and repeat for each feature, where a composed unit is written once. [inferred]

## Parked unknowns

- Whether Nexus matching validates a reply against a retried task's identity the way the activity validates its token; fn-151's reading of matching would settle it.
- Whether the legacy-stamp compatibility path is reachable in a current deployment; the server's mixed-version rollout policy would settle it.
