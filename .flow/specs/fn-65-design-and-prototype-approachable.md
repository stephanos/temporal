# Design and prototype approachable Nexus2 feature authoring

## Goal & Context

Design an authoring experience for ordinary developers with little Lean knowledge, then prototype it under `model/Temporal/Feature/Nexus2`. Authors must be able to define finite states/transitions as well as Properties, Behaviors, and Queries. The user selected the current Nexus lifecycle followed by a cancellation/completion race. Product-owner readability is a stretch goal.

This is a separate experiment from fn-62. The working design is `model/Temporal/Feature/Nexus2/DESIGN.md`; its candidate syntax remains proposed until reviewed. The current stage is design, not implementation authorization for a finalized interface.

## Architecture & Data Models

Use finite typed transition data to derive the existing FiniteMachine/AuthoredTarget/checkTarget path. Existing Property, Behavior, Query, and planning modules remain authoritative. Extend `Umpire.Property` with typed Boolean conditions and named guarded cases while preserving legacy clause meanings; update checking, evaluation, proofs, canonical formats, and affected consumers together. Compare ordinary typed constructors and a focused frontend over those same declarations. Preserve explicit typed domains, stable keys, alternatives, provider selection, and bounds. Generic support lives behind the existing Umpire owners; feature examples live in Nexus2.

## API Contracts

The proposed interface and examples are in `model/Temporal/Feature/Nexus2/DESIGN.md`. The baseline preserves the existing four-state/three-transition behavior through explicit identity mapping. A separate proposed race Target distinguishes a cancellation request from either canceled or successful terminal resolution. Checkers return language-owned typed failures; a frontend may map them to authored syntax. Routine authors must not edit extraction/coverage proofs.

Every applicable Property/case obligation holds together. Named `unless` conditions narrow applicability at the triggering prior-state/Action context; they neither waive independent invariants nor cancel outstanding temporal obligations when later events occur. Case groups may explicitly require completeness and exclusivity. Bounded analysis reports coverage, overlaps, contradictions, and modeled incompatibility with exact scope and evidence; it never chooses a winning Property.

## Edge Cases & Constraints

Keep model-owned outcomes, distinct witness/verification claims, stage-specific bounds, and conditional Property semantics. Reject malformed or duplicate identities, unknown typed references, unsupported expressions, invalid finite tables, and contradictory scenarios. Preserve explicit inconclusive search results. Never hide new compiler trust behind syntax or turn a generated representation guarantee into product correctness.

No existing Nexus behavior, inspector registrations, or runtime execution is changed by the design phase. Existing unrelated working changes are outside this work. Do not commit unless requested.

## Acceptance Criteria

- **R1:** Record a concrete authoring design for the user-selected baseline and race, including examples, alternatives, ownership, scope, and technical uncertainties. Errors: predicates outside the designed Boolean vocabulary and unmodeled race cases are explicitly identified; illustrative syntax is never presented as compiled.
- **R2:** Ordinary finite model changes can be expressed without proof editing, encoded ModelValue assembly, or support-code changes. Errors: invalid domains/results, duplicate keys, unused Actions, and ambiguous encodings must reject through the stated authoring boundary.
- **R3:** Properties, Behaviors, and Queries retain their existing checked semantics, including explicit Query form and units. Errors: invalid references, missing capabilities, contradictory constraints, unsupported syntax, and Limit Reached preserve their responsible status.
- **R4:** The prototype checks baseline behavioral equivalence and both race outcomes, plus a counterexample to cancellation always winning. Errors: request-only, no-trigger, and unsatisfiable cases are separately explained and tested; bounded model results make no runtime claim.
- **R5:** Compare constructor and focused-syntax authoring with actual compilation, admission, source diagnostics, trust audits, editor observations, and the relevant repository gates. Errors: unmeasured UX and human-usability claims remain explicitly unproven; no silent native-proof fallback.
- **R6:** Review the concrete design with the user before freezing the prototype implementation plan. Errors: unresolved design choices stay visible rather than being represented as approval; no error surface beyond design status and artifact consistency.
- **R7:** Named guarded cases and exceptions use a typed portable Boolean vocabulary owned by `Umpire.Property`, with all applicable obligations conjoined and explicit trigger-time evaluation. Update the checker, evaluator, agreement proofs, identities, canonical encoding, version handling, and affected consumers while preserving legacy semantics. Errors: reject missing/wrong-kind references, empty Boolean groups, unsupported comparisons or future/result-state guards; exceptions never waive independent invariants, turn unknown inputs into truth, or silently withdraw a pending bounded obligation; missing replacement behavior remains explicit.
- **R8:** The finite prototype demonstrates ordinary/special cases, compatible overlaps, uncovered complete groups, overlapping exclusive groups, contradictory same-step expectations, and bounded modeled incompatibility with source-linked evidence. Errors: distinguish a single violation from contradiction, unexercised guards from coverage, dead ends from conflicting Properties, and Limit Reached from exhaustive results; no general/unbounded compatibility claim is permitted, and case order never selects a winner.

## Boundaries

No migration of established Nexus, general external DSL, arbitrary temporal predicates, interruptible progress obligations, general conflict solving, infinite-state verification, System links, Evidence collection, or live execution. The bounded typed Boolean/guarded-case extension is included; compound temporal responses and arbitrary cross-field equality remain outside it. Enum derivation and a generated product-owner view are optional later experiments, not first-baseline requirements. No broad revision of fn-62 or Umpire rules is implied.

## Decision Context

The user asked to design and eventually prototype the assessed interface using Nexus2, selected the current lifecycle then cancellation/completion race, and requested explicit property conditions, exceptions, and conflict handling in the design. The recommended candidate is a typed table with explicit alternatives, deep generic validation, guarded cases within the existing Property language, and separate authoring languages. The principal early experiments are checked-admission ergonomics, kernel-proof cost, and bounded case/conflict analysis. See the working design for alternatives and the rationale for the initially abstract race.
