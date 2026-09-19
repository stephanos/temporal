# Centralize configuration authoring with ConfigUseSpec

## Overview

Replace the repeated classification, interpretation, checked-result, proof-witness, and extraction ceremony in Temporal configuration owner modules with one typed `ConfigUseSpec α` authoring interface. Hard-cut the six Callback and Matching declarations to the new interface while preserving the existing checked configuration semantics and consumer-facing use functions.

## Goal & Context
<!-- scope: business -->

Temporal model authors currently repeat each setting key and identity across multiple records, then repeat an `Except` result, an `isSome` theorem, and a `get` expression to obtain the checked definition. The ceremony obscures the authored meaning and creates opportunities for internally inconsistent metadata. Configuration owners should state each independent catalog expectation once and still receive the same fail-closed checked definition, diagnostics, provenance, and runtime resolution behavior.

## Scope

- Add a shared typed `ConfigUseSpec α` that owns the complete owner-authored meaning for one setting without duplicating its key or identity.
- Project the existing classification, interpretation, and definition forms and delegate checking to the current central validator.
- Expose an explicit proof-taking checked extraction interface so owner declarations compile only after successful validation without named per-setting result/theorem/get scaffolding.
- Hard-cut exactly the four Callback and two Matching settings to the new authoring path.
- Preserve existing contexts, registries, use functions, decoders, callback behavior, resolution, provenance, diagnostics, ordering, and comments.

## Architecture & Data Models
<!-- scope: technical -->

`ConfigUseSpec α` is an unchecked immutable authoring record in shared Temporal System configuration. It holds the Definition ID; one setting key and identity; impacts; expected schema and default; optional opaque-default replacement; behavior fingerprint; decoder; context policy; sampling point; and change effect. Its semantic projections produce the existing low-level records, and its checking path calls the existing `checkConfigUseDefinition` implementation rather than duplicating catalog validation.

The checked product remains `CheckedConfigUseDefinition α`. Concrete uses continue to be created by `CheckedConfigUseDefinition.instantiate`, and resolution continues through the unchanged `ConfigUse` and `ConfigView` interfaces.

```mermaid
flowchart LR
    Owner[Owner-authored literals] --> Spec[ConfigUseSpec α]
    Spec --> Projection[Existing declaration forms]
    Projection --> Validator[Existing central validator]
    Validator --> Checked[CheckedConfigUseDefinition α]
    Checked --> Use[Existing instantiate and resolve path]
```

## API Contracts
<!-- scope: technical -->

- `ConfigUseSpec α` contains one canonical authored key and one canonical authored setting identity; every classification, interpretation, and definition projection uses those exact fields.
- `ConfigUseSpec.classification`, `.interpretation`, and `.definition` expose the existing record meanings for tests and expert consumers without introducing a second validator.
- `ConfigUseSpec.check` returns `Except ConfigError (CheckedConfigUseDefinition α)` through the existing validation path.
- `ConfigUseSpec.checked` accepts an explicit proof that `.check` succeeded and returns the checked definition. It performs no fallback, defaulting, unchecked construction, or hidden native decision.
- Callback and Matching keep their existing public context constructors, checked-definition registries, and typed `*Use` functions. The hard cut removes the individual classification, interpretation, result, and witness names from the public authoring surface; only private spec and checked-definition bindings may remain behind those stable consumer interfaces.

## Edge Cases & Constraints
<!-- scope: technical -->

- Independently authored key, identity, schema, default, context policy, fingerprint, and decoder values must not be inferred from the current generated setting; doing so would make drift checks tautological.
- The abstraction must support every existing `ConfigInterpretation` capability, including all `SettingDefault` variants and optional opaque-default replacement, even though the six migrated settings do not exercise every form.
- Invalid specs retain the exact existing `ConfigError` kind, Definition ID, key, offending value, related identities, and decoder message. Unknown keys, empty impacts, identity/schema/default/policy drift, malformed opaque replacements, and decoder failures remain fail-closed.
- Owner modules retain their exact dependency direction. Shared configuration must not import Callback or Matching, and owner-specific decoders and exact-context construction remain owner-owned.
- Registry order and all exposed metadata remain deterministic and equivalent across the cutover.
- Existing private `native_decide` use is permitted only as the explicit owner proof supplied to the checked extraction interface; the shared helper must not hide or expand the compiler-trust boundary.
- This is a source-level hard cut with no compatibility aliases for the removed authoring ceremony. All repository consumers are updated atomically.

## Approach

1. Add and document the shared authoring record, semantic projections, delegated check, and proof-taking checked extraction; exercise success, projection consistency, drift failures, opaque replacement, and trust boundaries in focused configuration tests.
2. Replace the six owner declarations with specs and inline checked witnesses, remove obsolete split authoring declarations, update cross-owner metadata assertions, and run the focused and full model gates.

## Quick commands

```bash
cd model && mise exec -- lake build Temporal.System.Configuration.Tests
cd model && mise exec -- lake build TemporalModelTests
make umpire-build-model
make lint-model
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Shared configuration exposes a documented typed `ConfigUseSpec α` containing each owner-authored expectation once, including optional opaque-default replacement, and its projections reproduce the existing classification, interpretation, and use-definition meanings from one key and identity. Errors: malformed values are not rejected during record construction; they are reported by R2 through the existing checked path.
- **R2:** `ConfigUseSpec.check` delegates to existing configuration validation, and `ConfigUseSpec.checked` requires an explicit success proof while eliminating named per-setting result/isSome/get ceremony. Errors: unknown or empty keys, empty impacts, identity/schema/default/context-policy drift, invalid opaque replacements, and decoding failures retain their exact existing `ConfigError` diagnostics; no unchecked fallback exists.
- **R3:** The four Callback and two Matching settings use only the new spec authoring path. Obsolete split classification, interpretation, result, and witness declarations disappear from the public surface; owner modules retain at most private spec and checked-definition bindings behind the stable registries and use functions. Definition IDs, keys, identities, impacts, schemas, defaults, opaque replacement values, fingerprints, decoders, policies, sampling points, change effects, registry contents, and registry order are unchanged. Errors: any metadata or ordering difference is a regression failure.
- **R4:** Existing context constructors, public typed use functions, defaults, overrides, legal and illegal context handling, callback address interpretation, resolution sources, immutable views, and provenance remain behaviorally identical after the hard cut. Errors: missing/illegal contexts, schema mismatch, default or identity drift, and decoder failures return the same stable diagnostics and never enter resolution.
- **R5:** The change preserves every existing comment, adds public Lean docstrings at the new abstraction seam, keeps shared-to-owner imports one-directional, and passes focused configuration tests, `TemporalModelTests`, the full model build, and model lint. Errors: new import-graph violations, broader imports, generated catalog edits, warnings, or a wider native-decision trust boundary fail acceptance.

## Early proof point

Task fn-42-centralize-configuration-authoring-with.1 validates the core approach by compiling the typed projections, delegated checking, explicit checked extraction, and representative positive and negative drift cases.
If it fails, re-evaluate whether the repetition can be removed without weakening independent authored expectations before migrating Callback or Matching.

## Boundaries
<!-- scope: business -->

- No custom Lean syntax, command elaborator, macro DSL, or generated authoring code.
- No changes to the generated dynamic-configuration catalog, generator, resolver algorithms, persisted formats, runtime configuration, CI workflows, or drift-check commands.
- No callback address-parser, callback domain-model, file-layout, facade-import, or general module-splitting refactor.
- No compatibility aliases for authoring declarations removed by the hard cut; stable consumer use functions and registries remain.
- No external documentation rewrite when existing module paths and semantic ownership statements remain accurate.

## Decision Context
<!-- scope: both -->

Use an ordinary typed record and methods because the repetition is coupled metadata and checked extraction, not a new language. This creates a deep module with a small Lean-native interface, normal editor support, and direct reuse of the existing validator. Reject a custom DSL as unnecessary syntax and elaboration machinery for six settings. Keep the existing low-level records and central validator as expert/testing seams rather than replacing validated semantics during a readability refactor. Preserve consumer functions and registry shapes while taking the user-authorized hard cut on unused intermediate authoring names.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Single-source typed authoring record and projections | fn-42-centralize-configuration-authoring-with.1 | — |
| R2 | Delegated validation and explicit checked extraction | fn-42-centralize-configuration-authoring-with.1 | — |
| R3 | Hard-cut six Callback/Matching declarations | fn-42-centralize-configuration-authoring-with.2 | — |
| R4 | Preserve contexts, resolution, diagnostics, and provenance | fn-42-centralize-configuration-authoring-with.1, fn-42-centralize-configuration-authoring-with.2 | — |
| R5 | Preserve comments, module rules, trust boundary, and verification | fn-42-centralize-configuration-authoring-with.1, fn-42-centralize-configuration-authoring-with.2 | — |

## References

- Lean Authoring Guidelines sections 2, 4, 5, and 6 — deep interfaces, public docstrings, preserved comments, proof-trust review, and focused/full verification.
- Umpire 4 rules MOD-02, MOD-04, MOD-07, MOD-08, AUT-01, and AUT-03 — System ownership, focused modules, narrow contracts, isolated testing, approachable authoring, and checked declarations.
- Existing central configuration validation and checked-definition lifecycle remain the semantic authority for this refactor.
