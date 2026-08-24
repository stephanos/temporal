# Umpire observation and semantic verdicts

## Goal & Context
<!-- scope: business -->

Complete the offline semantic loop by interpreting synthetic execution evidence into qualified
semantic traces and evaluating reusable properties without mixing runtime facts into property
definitions. The primary user is an Umpire model engineer debugging why a check is satisfied,
violated, unknown, conflicting, or unsupported.

Success means the same pure property can be evaluated over model-generated traces and interpreted
evidence, with every verdict explaining its bounds and evidentiary basis.

## Architecture & Data Models
<!-- scope: technical -->

Define a separate typed Observation language. Small profile-specific mapping rules compile into one
checked interpretation plan. The plan normalizes evidence, binds symbolic and runtime identities,
establishes source-local and causal ordering, checks source closure and gaps, and constructs a
`QualifiedTrace` around the pure `SemanticTrace` supplied by the authoring core.

Qualification and provenance gate property evaluation outside the property itself. Every established
semantic observation carries a compact derivation linking the mapping version, matched evidence
identities, bindings, ordering facts, and source-closure evidence. Mapping declarations also state
whether each field is retained, redacted, hashed, or rejected.

## API Contracts
<!-- scope: technical -->

- Observation mappings consume declared evidence profiles and emit only vocabulary they are
  authorized to establish.
- Composed mapping rules reject overlaps, incompatible bindings, and ordering conflicts before
  processing evidence.
- Incomplete, gapped, unclosed, or ambiguous evidence cannot invoke a property as though it were a
  complete semantic trace.
- Compatible alternative interpretations return `unknown` with the alternatives and missing
  discriminator. They are not guessed, and initial support does not quantify over all alternatives.
- Each selected trace and property retains an independent result. The strict query summary cannot
  report success when a required result is unknown, conflicting, unsupported, or divergent.
- Structured verdicts identify the responsible property clause, relevant trace span, evaluated
  bound, semantic provenance, and observation derivations.

## Edge Cases & Constraints
<!-- scope: technical -->

Missing source closure, sequence gaps, duplicate evidence, conflicting sources, ambiguous identity
bindings, unsupported vocabulary, misdirected fault receipts, and disallowed sensitive values all
fail closed. One unsupported property does not erase supported property results, but the aggregate
query remains incomplete.

Synthetic evidence is the only first-release source. Raw evidence is a separately controlled
artifact and is never copied wholesale into portable semantic traces. The evaluator must agree with
the portable property's Lean denotation.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Typed composable mapping rules compile into a deterministic interpretation plan and reject overlaps, incompatible bindings, wrong-kind output, and ordering conflicts before evidence processing. [paraphrase]
- **R2:** Synthetic evidence is normalized, identity-bound, ordered, checked for closure and gaps, and converted into a qualified semantic trace before any property runs. Errors: incomplete or unclosed evidence produces `unknown`, not success. [paraphrase]
- **R3:** Every semantic observation has an auditable derivation covering mapping version, evidence identities, bindings, ordering, and closure. Errors: missing derivation data prevents a qualified verdict. [paraphrase]
- **R4:** Ambiguous compatible interpretations return `unknown` with alternatives and the missing discriminator; conflicting facts return `conflict`. Errors: stable ordering never chooses an interpretation silently. [paraphrase]
- **R5:** Field disposition is enforced as retain, redact, hash, or reject, and portable traces contain only approved normalized values. Errors: undeclared sensitive values are rejected. [paraphrase]
- **R6:** Properties evaluate independently through capability-limited views, producing structured per-property verdicts and a strict query summary. Errors: unsupported or unknown checks remain visible and prevent aggregate success. [paraphrase]
- **R7:** Independent fixtures and layer-specific mutations demonstrate that model, mapping, and property mistakes fail at the responsible boundary. Errors: observation tests cannot reuse the property implementation as their oracle. [paraphrase]

## Boundaries
<!-- scope: business -->

No live server integration, additional evidence profile, long-term raw-evidence store, distributed
clock reconstruction, or evaluate-all-compatible-traces optimization is included. Properties remain
pure and cannot reference logs, spans, RPC names, storage rows, or evidence profiles.

## Decision Context
<!-- scope: both — conditionally substructured -->

### Motivation
<!-- scope: business -->

Evidence interpretation is independently useful and must not contaminate reusable semantic
properties. The offline fixture proves this boundary before a runtime adapter can create false
confidence.

### Implementation Tradeoffs
<!-- scope: technical -->

A qualified wrapper keeps completeness and provenance outside pure traces. Compiled mapping plans
retain composability without runtime callback ambiguity. Returning unknown for several compatible
traces is deliberately conservative until explicit all-traces semantics are designed.
