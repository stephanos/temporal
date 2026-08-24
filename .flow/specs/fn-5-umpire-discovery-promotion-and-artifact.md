# Umpire discovery promotion and artifact evolution

## Goal & Context
<!-- scope: business -->

Make Umpire's semantic vocabulary and generated results discoverable, reviewable, promotable, and
safe to evolve. Model engineers must be able to explain a term, inspect an artifact, and turn a
selected exploratory witness into an exact regression without translating into another authoring
language.

Success means generated documentation stays synchronized with Lean, promoted regressions retain why
they exist, and compatible readers handle persisted artifacts without silently changing meaning.

## Architecture & Data Models
<!-- scope: technical -->

Lean declaration metadata is authoritative. It deterministically generates a checked-in
`model/GLOSSARY.md` and a machine-readable vocabulary index. Discovery commands project the same
catalog for properties, behaviors, queries, capabilities, mappings, and related vocabulary.

Promotion creates a complete regression unit containing an exact semantic trace, its properties,
target composition, expanded bounds, source query, semantic digests, selection reason, and
provenance. Artifact formats use explicit major/minor compatibility and deterministic named
migrations. Unknown major versions and unrecognized semantic changes fail closed.

All model-generation and staleness checks are wired through the repository's top-level Makefile;
no model-local Makefile is introduced or extended.

## API Contracts
<!-- scope: technical -->

- Every public vocabulary entry has a stable namespaced identity, kind, summary, declaration source,
  required/provided capabilities, references, aliases, deprecations, and semantic digest.
- `glossary list`, `glossary explain`, and corresponding property/behavior/query discovery commands
  read projections of Lean metadata rather than independent documentation.
- Promotion never emits a second Drive authoring language. It creates checked source using
  `traceExactly` plus the original semantic context.
- Artifact readers reject unknown majors, accept only declared compatible minor additions, and apply
  deterministic named migrations when old meaning must be transformed.
- Documentation-only or proof-only edits may preserve semantic identity only when consumed contracts
  are unchanged.

## Edge Cases & Constraints
<!-- scope: technical -->

Duplicate or wrong-kind identities, stale generated Markdown, broken references, alias cycles,
missing replacements, nondeterministic ordering, unsupported format versions, and incomplete
promotion provenance fail verification. Regeneration is byte-for-byte deterministic.

Human readability and stable machine consumption have equal priority. Generated artifacts cannot
acquire semantics not present in Lean declarations.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Public Lean metadata deterministically generates a checked-in `model/GLOSSARY.md` and machine-readable index covering framework and model vocabulary. Errors: duplicate identities, wrong kinds, alias cycles, and broken references fail generation. [paraphrase]
- **R2:** Top-level Makefile generation and check targets fail when the glossary or index is stale, inconsistent, or nondeterministic; no model-local Makefile carries this wiring. [user]
- **R3:** Discovery commands explain vocabulary and list properties, behaviors, and queries from the generated catalog without creating a second semantic authority. Errors: unknown or deprecated terms produce structured guidance. [paraphrase]
- **R4:** Promotion turns a selected witness into a complete `traceExactly` regression retaining properties, target, expanded bounds, semantic digests, source query, selection reason, and provenance. Errors: incomplete context prevents promotion. [paraphrase]
- **R5:** Versioned readers and deterministic named migrations preserve declared compatible artifacts and reject unknown majors or semantic reinterpretation. Errors: best-effort parsing cannot silently ignore meaning-bearing fields. [paraphrase]

## Boundaries
<!-- scope: business -->

No graphical interface, Go authoring wrapper, arbitrary artifact repair, undocumented compatibility
heuristic, or second hand-maintained glossary is included. Advanced minimization may later feed
promotion but is not part of this spec.

## Decision Context
<!-- scope: both — conditionally substructured -->

### Motivation
<!-- scope: business -->

Stable concepts are useful only when authors can discover them and reviewers can understand changes.
Promotion closes the exploratory-to-regression workflow without duplicating semantics.

### Implementation Tradeoffs
<!-- scope: technical -->

Checked-in generated documentation makes semantic change reviewable. Explicit migrations trade some
maintenance for safe persisted artifacts. A complete promoted unit prevents exact action schedules
from losing their properties or provenance.
