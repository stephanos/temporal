# Facets: action-owning behaviour areas of a feature

## Conversation Evidence

> user (turn 23): "could we align the grouping of the effects with the idea of \"sub features\" ie standalone activity is the feature and then Cancellation is a \"sub feature\" or maybe a better name would be capability or sth. (I think we already use that word)"
> user (turn 25): "let's find the best name to distinguish between a feature's \"facets/capabiltiies\" and the cross-temporal \"patterns\" aka capabiltiies. both are similiar, eg Cancellation fits into both, but need clear names"
> user (turn 26): "should we colocate more things into a Facet like rules and properties? only making it an effect seems limiting. but then, how would we deal with cross-facet behavior?"
> user (turn 28): "yes, Adopt the action-owning facet as the target -> create separate flow spec"
> user (turn 28): "should we enforce that only Facets can define actions and rules? that ensures that Machine is always split into Facets"
> user (turn 28): "will facets apply to both product/ and system/ ? and do they have to map 1:1 ?"
> user (turn 29): "yes"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [user], 50% [paraphrase] -->

A feature's machine is one large object per level. Its effects are unstructured, and its rules are grouped by actor. Meanwhile the feature's claims are already split by behaviour area: the standalone activity's system level has `Completion`, `RetryFailures`, `Cancellation`, `Pausing`, `Timeouts`, `Heartbeat`, `ByID` and `Reset` claim objects. The user wants the feature's behaviour organised along those same sub-features, with a structure the DSL enforces instead of comments.

A **facet** is one behaviour area of one feature. It owns a set of actions and holds their rules, the effects those rules use, the claims that check them, and the capabilities it declares. The feature stays one machine per level, assembled from its facets.

The name distinguishes facets from **capabilities**, the cross-Temporal behaviour contracts any entity can declare (`Closable`, `Pausable`, `Pollable`, …). A facet is feature-local and named as a noun for what happens (`Pausing`, `Cancellation`). A capability is generic and named as an adjective for what an entity can be (`Pausable`). A facet declares capabilities: the Pausing facet declares `Pausable`. [paraphrase]

## Architecture & Data Models
<!-- scope: technical -->

- **A facet** is a unit of a machine's level that contributes sections: `effects`, `rules` and `capabilities`. A machine object lists its facets, and its sections are the union of its facets' sections plus what stays machine-wide. [paraphrase]
- **What stays machine-wide:** `State`, the phase enum and its accessors (the shared vocabulary every facet reads; facets don't own phases), `init`, `end`, the phase projection, the refinement, monitors that belong to no single facet, and Queries over whole-feature scenarios. [paraphrase]
- **Ownership by action:** every action the machine binds, including timers, faults and `disabled` bindings, belongs to exactly one facet, and all of that action's rules live in that facet. A rule whose guard involves another facet's phase belongs to its action's facet and names the phase. [paraphrase]
- **Cross-facet dependencies are explicit:** a facet that calls another facet's step declares the dependency in its type (e.g. a Scala self-type), and steps many facets use sit in a base facet. Cross-facet claims sit in the facet whose promise they state. Behaviour no single facet owns may get a named interaction facet that depends on both; this is optional, not required up front. [paraphrase]
- **Claims:** each facet's Properties and Queries stay in their own derived claim object beside the facet, in the same file, so Property and Query names keep their identities. [paraphrase]
- **Staging:** stage 1 moves effects and claims into facet files, and rules stay grouped by actor in the machine. Stage 2 moves rules and capability declarations into their owning facets and turns on the ownership check. [paraphrase]

## Edge Cases & Constraints
<!-- scope: technical -->

- The semantics already require an action's rules to be disjoint across every block, so the order of an action's cases never changes the meaning. Gathering an action's cases from facets into one step function is well-defined. [paraphrase]
- With rules spread across facets, the per-actor "who can do what" view is no longer one block of source. The lint's per-machine table by action and state, and its report of requests the model is silent on, serve as that view and as the safety net for gaps. [paraphrase]
- Rows, outcomes, facts, Cases and Query answers must not change. Function bodies (case order follows facet order), content hashes and source positions in the IR may change, which means regenerating the IR. [paraphrase]
- The product and system levels have different `State` types, so each level has its own facets. A facet with nothing left at a level (e.g. product Completion after trivial effects become `enters(...)`) has no file there. [paraphrase]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** The framework provides a facet construct with `effects`, `rules` and `capabilities` sections, documented in the README and SEMANTICS with its core form: a machine's section is the union of its facets' sections in the order the facets are listed. [paraphrase] Errors: two facets defining the same effect name fail to compile; a facet section the lifter cannot lift is refused, naming it.
- **R2:** The lifter lifts a machine assembled from facets to the same rows, outcomes and facts as the equivalent single-object machine. [paraphrase] Errors: no error surface beyond R1 and R3.
- **R3:** Every action a machine binds belongs to exactly one facet, and all of its rules are in that facet. [paraphrase] Errors: an action with rules in two facets, or bound by no facet's rules or `disabled`, fails the lift, naming the action and the facets involved.
- **R4:** A facet that uses another facet's step declares that dependency in its type, so a machine missing the depended-on facet does not compile. [paraphrase] Errors: compile error naming the missing facet.
- **R5:** Stage 1: the standalone activity's effects and claims sit in facet files at both levels, with rules still grouped by actor in the machine. Stage 2: rules and capability declarations sit in their owning facets. The IR keeps every row, outcome, fact, Case and Query answer in both stages. [paraphrase] Errors: any IR difference beyond function bodies, hashes and source positions is explained before it is accepted.
- **R6:** The README defines **facet** next to **capability**: a facet is a feature-local behaviour area named as a noun, and a capability is a cross-Temporal contract named as an adjective that a facet declares. Facet and capability names follow that convention. [paraphrase] Errors: none (documentation and naming).
- **R7:** The layout rules and the lifter's order lint place facets: one facet per file, named after it, and no effect, rule or capability declaration outside a facet or the machine-wide places listed above. [inferred] Errors: a misplaced declaration is refused at its line, naming where it belongs.
- **R8:** In every declared machine (`Machine`), rules, effects and capability declarations are defined only in facets; the machine object holds only machine-wide declarations and its facet list. A machine may consist of a single facet. Actions stay declared by their actor in the feature's signature: facets own actions by binding them, and never declare them. Derived machines and compositions are exempt. [user] ("should we enforce that only Facets can define actions and rules? that ensures that Machine is always split into Facets"; "yes") Errors: a rule, effect or capability declared in a machine object outside a facet fails the lift at its line, naming the facet place it belongs in.
- **R9:** Each feature declares one facet vocabulary, the facet names shared by its levels. Each level's facets use those names; a level holds only the facets it needs; and an action bound at both levels is owned by the same-named facet at both. [user] ("will facets apply to both product/ and system/ ? and do they have to map 1:1 ?"; "yes") Errors: a facet name outside the feature's vocabulary, or an action owned by differently named facets at the product and system levels, fails the lift, naming the action and both facets.

## Boundaries
<!-- scope: business -->

- Splitting other machines into several facets is out of scope: fn-161.4 wraps each remaining declared machine into the facets its rules need (often one), so the global rule can be switched on, without redesigning them. [paraphrase]
- Moving claims into the base machine (renaming every Property and Query) is out of scope; claim objects stay derived. [paraphrase]
- Renaming the capability concept is out of scope; "capability" keeps its meaning. [paraphrase]

## Decision Context
<!-- scope: both — conditionally substructured -->

The user adopted the action-owning facet as the target. [user] Ownership by action was chosen over placing cross-facet cases by their reason, because it can be enforced mechanically, while placement by reason is a judgment call each time. [paraphrase] An effects-only facet was considered and judged too limiting by the user. [user] "Facet" was chosen over aspect (which in software means cross-cutting, the generic concept's meaning), behavior (collides with Behavior Fingerprint), pattern, contract, slice, policy and profile (all already in use), and over renaming "capability" (entrenched in the framework, Go tools and the IR schema). [paraphrase]

## Parked unknowns

- How a facet's sections are expressed in Scala (traits with nested section objects vs another encoding) and whether the lifter can read them without new restrictions. The stage 1 implementation resolves this. [inferred]
- Whether an interaction facet is needed for the standalone activity at all. Stage 2's migration resolves this. [inferred]

## Nexus facet vocabulary (Batch 1d nexus lane, 2026-10-11)

One vocabulary for the kind, shared by the Product (`NexusProduct`) and both forms' Protocol machines (standalone and workflow `NexusSystem`, to be `NexusProtocol`). Facets own actions by binding them; a level holds only the facets it needs. Nouns for what happens:

| Facet | Owns (product) | Owns (standalone protocol) | Owns (workflow protocol) | Capabilities it declares |
|---|---|---|---|---|
| `Scheduling` | (none: the product begins scheduled) | `client.start` | `caller.schedule` | |
| `Starting` | `handler.reply` | `handler.reply` | `handler.reply` | workflow: the two `Retryable` handler-error declarations (their failure is a reply class) |
| `Completion` | `handler.complete` | `handler.complete` | `handler.complete` | standalone: `Closable` with its `overriding` (closed rejection is the completion's promise), `Describable` |
| `Retrying` | `disabled(network.fault, worker.stop)` | | `network.fault`, `worker.stop`, `timers.backoff` | workflow: `Retryable` for `network.fault` |
| `Cancellation` | | `client.requestCancel` | | |
| `Termination` | `client.terminate` | `client.terminate` | | standalone: `Terminable` |
| `Timeouts` | `timers.timeout` | | `deadline.scheduleToClose`, `deadline.scheduleToStart`, `deadline.startToClose` | workflow: the three `Expirable` declarations |

- `Starting` is the handler's answer to the start request (sync settle, async start, or a handler error); `handler.reply` is bound at all three machines, so it is `Starting` at all three (R9). The retryable reply class backs the operation off, but its rules stay with their action (ownership by action), and `Retrying` keeps the faults and the backoff timer that make a retry.
- `Retrying` at the product holds only the `disabled` binding of the two faults, because the workflow protocol binds them and R9 requires the same owner at both levels.
- Design and adversarial machines take one facet named after their concern: `ClosePolicy` for `RejectAfterClose` (its derived designs stay derived) and `ForgedControl` for `TrustingCaller`. They are workflow-protocol subjects with their own state, so the product/protocol same-owner rule does not reach them; the two names join the kind's vocabulary.
- Machine-wide (no facet): each machine's `State`, phase enum, `init`, `end`, phase projection, refinement, `states` sets, and the whole-feature `queries`; `HandlerWorker` (a derivation) and `NexusCaller` (a composition) are exempt (R8).
