# System capabilities: distributed-systems patterns as named capabilities

DSL changes ship a task-named Scalafix rule under `model/migrate/rules/`, prove it on copied `activity_mini` sources before the real-feature sweep, and follow `model/migrate/README.md`; production loading remains held in fn-166.10.

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

## Inventory and decisions (Batch 1d, 2026-10-11; R1, R3, R4)

| Hand-written claim | Where | Capability | Law | IR |
| --- | --- | --- | --- | --- |
| `atMostOneActive` (state count) | ActivityRecord and its compositions | `Exclusive` | `Exclusive.atMostOneActive(m)(twoActive)` | Property, unchanged name |
| `atMostOneActiveAttempt` (records count) | ActivityRecord, HeldDispatch monitors | `Exclusive` (history law) | stays a hand-written monitor | Monitor |
| `staleDelivery*` (`staleDeliveryRejected`) | HeldDispatch | `Fenced` | `Fenced.staleDeliveryRejected(m)(deliver, rejected)` | Property, unchanged name |
| `staleDelivery` verifies (`notPaused` on the stale path) | admission designs | `Fenced` | stay Queries of Pausable's claim on the stale path | Query |
| `repeatedRequestsAnswer`, repeated RequestCancel/Terminate answered OK after close | standalone Nexus | `Idempotent` (action-targeted) | `Idempotent.closedRejectsOrRepeats(m)(closed, repeats, rejected, accepted)`, the machine's override of Closable's `closedIsRejectedUniformly`; `Idempotent.repeatedRequestsAnswer` its reason | Property |
| `singleOutcome` | Nexus close-policy designs | single outcome (history law) | stays a hand-written monitor | Monitor |

- System capabilities are named law definitions in `temporal.capabilities.system` (one definition each, a single use is enough), called where the Model states the promise. They are not `CapabilityKind` sections: a section generates a Query per law, which would add Queries and answers, while R5 keeps every answer.
- History laws stay monitors written in the machine's `monitors` section, because the lifter lifts a monitor only there (a monitor built by a helper def is refused); synthesizing them is left for later (R4).
- `Retryable` and `Expirable` are system capabilities (`capabilities/system/`), the rest product (`capabilities/product/`).

## Scope: the mini feature only (owner, 2026-10-11)

Within the Scala DSL batch (`MILESTONES.md`), this spec changes only the framework, the lifter, the shared code and `features/activity_mini`. The real features (standalone activity, Nexus workflow, Nexus standalone and their legacy payloads) are not edited, migrated, lifted or regenerated, and they stay behind until the owner's sweep, which runs this spec's scalafix rules on them. Equivalence compares `FEATURES=activity_mini` against the mini baseline. This overrides any statement in this spec that edits, migrates or proves on a real feature. See fn-166's Scope. The pattern inventory and the naming proofs cover the mini feature and the shared code.

## Redesign impact (fn-173, 2026-10-13)

Owner-approved redesign `fn-173` (reachable exploration; `model/design/reachable-exploration.md`) drops full catalog tables, Scala step tables and equivalence sealing, and Query totals; the Behavior Fingerprint becomes an IR structural hash. For this spec:

- **Evidence:** "Step tables unchanged" becomes fresh-lift IR identity.
- **Optional:** the Exclusive and SingleOutcome history laws may use `Set`/`List` monitor state once fn-173 Phase 4 lands.
