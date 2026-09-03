# Partition the Observation authoring language

## Overview

Separate inert Observation declarations from deterministic checking and canonical plan construction while preserving `Umpire.Observation.Language`, `Umpire.Observation`, and the complete public `Umpire.*` authoring surface. Keep the proof-taking convenience in Language and make Evaluation and portable artifacts depend on the narrowest established child module.

## Goal & Context
<!-- scope: business -->

Observation authors currently rely on a stable public facade, but its implementation combines inert declaration data, field projections, target-meaning resolution, typed errors, checked-plan contracts, canonical identity construction, and the full checker. Maintainers should be able to change declaration shapes or compiler mechanics locally without creating a second authoring path or asking ordinary model authors to learn internal modules.

Temporal model authors retain the same declarations and imports. Evaluation, portable-artifact, and compiler maintainers gain ownership-specific imports and focused verification. There is no end-user, runtime, configuration, artifact-schema, or generated-code change.

## Architecture & Data Models
<!-- scope: technical -->

`Umpire.Observation.Declaration` owns the existing inert authored vocabulary and field-specification projections. It imports only the lowest sufficient reusable substrate and keeps every declaration in namespace `Umpire` with its current fully qualified name, representation, defaults, coercions, instances, and comments.

`Umpire.Observation.Compiler` is the deep module behind `checkObservation`. It owns check context, resolved Target meaning selection, typed errors, checked expression and plan contracts, canonical identity rendering, private ordering helpers, and the full deterministic checking pipeline. Canonical sorting and fingerprint construction remain together so no new interface or duplicate ordering authority is created.

`Umpire.Observation.Language` remains the compatible focused authoring import. It aggregates Declaration and Compiler and retains `checkedObservation`, the existing explicit-proof convenience. `Umpire.Observation` remains the complete facade that also exposes Evaluation, Verdict, and Run Evaluation checking.

Observation Evaluation imports the checked compiler contract rather than the authoring aggregator. Portable Evaluation Contract data imports only Declaration. Neither focused internal import changes what the public facades export.

## API Contracts
<!-- scope: technical -->

- Every authored value type, profile, bound, reference, field specification and projection, expression, disposition, binding, rule, ordering, closure, mapping, context, error, checked expression, checked plan, checker, and proof-taking convenience retains its fully qualified name and observable contract.
- Declaration data remains inert. Invalid combinations remain representable and are rejected only by `checkObservation`; projections perform no validation, registration, normalization, or default selection.
- `checkObservation` remains the sole raw-to-checked interface and returns the same typed error or complete checked plan. Failure returns no partial plan.
- `checkedObservation` continues to require an explicit proof that the same raw checker succeeds. No hidden native decision, unchecked constructor, recovery path, or trust dependency is added.
- Provider meaning resolution, explicit connector reconciliation, canonical ordering, exact error rendering, source fallback, first-failure precedence, and fingerprint inputs remain exact.
- Public callers continue importing Language, Observation, or the Umpire umbrella rather than child modules.

## Edge Cases & Constraints
<!-- scope: technical -->

- Blank, malformed, duplicate, unknown, wrong-type, missing-disposition, invalid ordering, invalid closure, unsupported callback or recursive expression, and invalid bound combinations remain representable as raw declarations and fail at the established checker boundary.
- All existing Observation error kinds retain exact kind, definition ID, source-path fallback, offending value, canonical related-identity order, exact rendered JSON, and first-failure precedence.
- Agreeing providers retain their meaning; conflicting providers require the existing explicit connector reconciliation; missing reconciliation remains unauthorized.
- Reordering semantically equivalent declarations preserves checked-plan identity. Behavior-affecting changes, including the Evidence bound, continue to change the fingerprint.
- The Declaration module does not import Target, DefinitionGraph, Evaluation, or Temporal. Compiler dependency direction remains acyclic and satisfies reusable-module isolation.
- At ten times declaration or Evidence volume, the same sorting and traversal algorithms run with no cache, second normalization pass, or copied alternate representation.
- All work remains pure Lean. No I/O, runtime, credential, concurrency, recovery, third-party dependency, axiom, or compiler-trust path is introduced.
- Existing comments and authored documentation values move intact. Architecture documentation names the internal ownership while public usage guidance remains unchanged.

## Approach

1. Move inert declaration vocabulary and field projections into Declaration and narrow the portable artifact's import.
2. Move checked contracts, canonicalization, target resolution, typed diagnostics, and checker mechanics intact into Compiler.
3. Keep Language as the compatibility and proof-taking authoring seam, narrow Evaluation's internal import, and strengthen facade checks.
4. Update internal architecture navigation and run focused, downstream, aggregate, artifact regression, trust, import, and lint gates.

## Quick commands

```bash
cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Temporal.Feature.Nexus.ObservationTests Temporal.System.Nexus.Tests
cd model && mise exec -- lake build UmpireTests TemporalModelTests
make umpire-build-model
make umpire-check-regression
make lint-model
make lint-code
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `Umpire.Observation.Declaration` owns the existing inert Observation vocabulary and field projections without changing any fully qualified name, constructor, field order, default, derived instance, coercion, declaration order, comment, or documented meaning. Errors: blank, malformed, duplicate, unknown, wrong-type, missing-disposition, and otherwise invalid raw combinations remain representable and continue to be rejected only by `checkObservation`; projections add no validation, registration, normalization, or default disposition.
- **R2:** `Umpire.Observation.Compiler` owns check-context construction, resolved Target meaning selection, typed errors, checked expression and plan contracts, canonical identity construction, and the complete deterministic `checkObservation` pipeline behind the same interface. Errors: every existing error kind retains exact kind, definition ID, source fallback, offending value, canonical related-identity order, rendered form, and first-failure precedence; failure returns no partial checked plan.
- **R3:** Provider meaning resolution and canonical identity remain exact: agreeing providers retain their meaning, conflicting providers require existing explicit connector reconciliation, missing reconciliation remains unauthorized, equivalent declaration reordering preserves identity, and behavior-affecting changes still change the fingerprint. Errors: missing or conflicting meaning, unauthorized semantic declarations, noncanonical ordering, and fingerprint drift retain current diagnostics and fail closed.
- **R4:** Language, Observation, and the Umpire umbrella preserve the complete public surface, and `checkedObservation` still requires an explicit proof that the same raw checker succeeds. Errors: proof omission remains an elaboration failure; invalid declarations remain available only through the typed checker result; no hidden native decision, unchecked constructor, recovery path, or new trust dependency is added.
- **R5:** Internal imports become ownership-specific: Observation Evaluation imports the compiler contract and portable contract data imports only Declaration, while public facades retain all transitive names. Errors: an import cycle, lost public name, broadened Temporal dependency, reusable-module isolation violation, or direct public reliance on a child module fails focused builds and import lint; there is no runtime error surface.
- **R6:** Checked examples and compatibility gates prove field-projection equality, typed checked-expression retention, connected-target reconciliation, exact canonical plan and error rendering, source paths, diagnostic precedence, facade visibility, unchanged downstream Observation behavior, and preserved comments. Errors: identity, artifact byte, warning, trust, documentation, regression, or lint drift blocks completion.

## Early proof point

The declaration extraction must reproduce the existing field projections exactly while compiling the portable contract without Compiler. If inert data needs Target or DefinitionGraph, or a projection begins validating or defaulting, reconsider the seam before moving the checker. The compiler extraction must then preserve the complete exact diagnostic matrix before any downstream import is narrowed.

## Boundaries
<!-- scope: business -->

- No new Observation authoring language, builder, macro, coercion-driven DSL, registry, callback, recursive authoring form, or parallel representation.
- No change to declarations, checked plans, Target meaning, error vocabulary, canonical identity, fingerprint, or source-location behavior.
- No additional checked-plan or canonical-helper module that would widen private ordering interfaces or duplicate identity logic.
- No Evaluation, Evidence, accepted-trace, Property, runtime, artifact schema, generated-file, or persisted-byte semantics change.
- No compatibility layer, public child-module contract, new dependency, axiom, cache, or CI workflow.

## Decision Context
<!-- scope: both — conditionally substructured -->

Declaration is inert and low-dependency; Compiler hides the complex target resolution, validation, canonicalization, and diagnostics behind the existing small checker interface. Language remains the stable authoring seam and proof-taking convenience. Splitting checked-plan data from its checker is rejected because canonical plan construction shares private ordering and fingerprint machinery; a separate shallow module would widen internals or duplicate identity logic.

Completed authoring, accepted-trace, coordinate, field-specification, and structural-analysis work defines the semantics preserved here. This spec precedes the evaluator decomposition: establishing final Declaration and Compiler imports first lets each future evaluator child choose a narrow dependency once and avoids concurrent edits to the public Observation facade.

Two new internal modules and import edges add minor structural complexity in exchange for locality and smaller recompilation surfaces. Algorithms, asymptotic performance, scalability, pure crash behavior, information-flow security, and trust remain unchanged.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Inert declaration ownership and exact projections | `.1` | — |
| R2 | Deep deterministic compiler and exact diagnostics | `.2` | — |
| R3 | Provider meaning and canonical identity preservation | `.2` | — |
| R4 | Stable public authoring facades and proof-taking convenience | `.2`, `.3` | — |
| R5 | Ownership-specific internal imports and acyclic graph | `.1`–`.3` | — |
| R6 | Focused, downstream, docs, regression, trust, and lint verification | `.1`–`.3` | — |

## References

- Umpire 4 rules SEM-01 through SEM-04, MOD-01, MOD-06 through MOD-08, AUT-01 through AUT-07, and EVD-03 through EVD-04 define model authority, language separation, module isolation, approachable checked authoring, and fail-closed Evidence use.
- Lean Authoring Guidelines sections 2, 4, 5, and 6 govern declaration interfaces, module documentation, trust, and verification.
- Completed ordinary-authoring, accepted-trace, coordinate, field-specification, and structural-analysis cleanup specs define the public and internal contracts preserved here.
- The evaluator decomposition consumes the final compiler seam after this spec lands.
