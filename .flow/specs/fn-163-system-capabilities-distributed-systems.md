# System capabilities: distributed-systems patterns as named capabilities

## Conversation Evidence

> user: "since we have capabilities/ should we make system-capabilities such as idempotency, fencing and other distributed systems patterns etc also named, first-class capabilities?"
> user: "I'd also use system/ and product/ packages to separate capacabilities"
> user: "I'd lower the bar from two-entities as even a single-use named entity improves readability; it's also then still easy to change it when a second or third is introduced"
> user: "yes, add new spec after Batch 5b"

## Goal & Context
<!-- scope: business -->

`model/temporal/capabilities/` names what an entity can be (Closable, Pausable, Pollable, Retryable, Expirable, Terminable, Describable). Each capability is declared once and its laws become Properties and Queries for every entity that declares it. Distributed-systems promises such as fencing, a single active attempt and idempotent requests are still hand-written per machine (`atMostOneActive`, the `atMostOneActiveAttempt` monitor, the `staleDelivery*` claims, `requestId` / `repeatedRequestsAnswer`, the repeated-RequestCancel refusal, `singleOutcome`). The goal is to name each such pattern as a first-class capability, so a Model says which promise it makes instead of spelling it out, and the promise has one definition. [paraphrase]

Capabilities split by level: system capabilities (server-side protocol promises: fencing, exclusivity, idempotency, single outcome) and product capabilities (what a user sees an entity can be: Closable, Pausable, ...) live in `capabilities/system/` and `capabilities/product/`. [user]

A pattern becomes a named capability even with a single use: the name improves readability, and changing it when a second or third use arrives stays easy. [user]

## Architecture & Data Models
<!-- scope: technical -->

- Capabilities keep today's mechanism (CapabilityKind, declared on a machine, laws lowering to existing Property / Monitor IR, generated Queries, origin recorded). Names are adjectives (fn-159.8's convention), distinct from facets (feature-local nouns, fn-161); a capability is declared inside the facet that owns its actions.
- Idempotency is a promise about an action (`repeating client.start with the same requestId answers the same`), so it needs an action-targeted declaration (e.g. `Idempotent(client.start, by = requestId)`), not only an entity-level one. [inferred]
- Fencing and single outcome need history (past attempts / outcomes). Today they are hand-written monitors. `.plans/TEMPORAL_PATTERNS.md` §3 names the trigger for history-scoped claim patterns ("a history-scoped claim recurring across two state types"); a capability whose law needs history may require synthesized monitor state types. [inferred]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** An inventory maps every hand-written distributed-systems claim in the Models (at least `atMostOneActive`, `atMostOneActiveAttempt`, `staleDelivery*`, `requestId` / `repeatedRequestsAnswer`, repeated RequestCancel refusal, `singleOutcome`) to a proposed capability and law, or records why it stays hand-written. [inferred]
- **R2:** `model/temporal/capabilities/` is split into `system/` and `product/` packages; existing capabilities move to the level they describe; module map and lint rules updated. [user]
- **R3:** Each inventoried pattern is a named capability (adjective) even with one use, e.g. `Fenced`, `Exclusive` (single active attempt), `Idempotent`, `SingleOutcome`; names are confirmed in the plan. [user]
- **R4:** Each capability's laws lower to existing IR where possible; a law needing history either lowers to a monitor the framework synthesizes or stays hand-written with a recorded reason. [inferred]
- **R5:** The Models declare the new capabilities and drop the hand-written duplicates; verdicts, Query answers and Case outcomes are unchanged, and identity changes are listed in a rename map checked with `make umpire-check-equivalence`. [inferred]
- **R6:** Where a capability makes a promise observable (idempotent repeats), the realization evidence follows, so the by-ID witnesses' `inconclusive(explanationsDisagree)` can be revisited. [inferred]
- **R7:** README / SEMANTICS describe system vs product capabilities and the single-use naming rule. [user]

## Boundaries
<!-- scope: business -->

- Scheduled after Batch 5b (fn-159–fn-162's single regeneration), on the facet, realization-binding and lifter (fn-141) baseline. [user]
- No change to what any claim promises; this names and relocates promises. [inferred]
- A general `always`/`eventually`/`until` DSL stays out of scope (TEMPORAL_PATTERNS level 2). [inferred]

## Decision Context

- Bar for naming: one use suffices (owner), replacing the earlier "two uses" rule from TEMPORAL_PATTERNS for capabilities. [user]
- Level split: system/ vs product/ packages (owner). [user]
