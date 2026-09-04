# Decompose the Observation evaluator behind its facade

## Overview

Replace the monolithic Observation evaluator implementation with cohesive contract, structural-analysis, raw-evaluation, and accepted-admission modules. Preserve the existing `Umpire.Observation.Evaluation` and `Umpire.Observation` facades, all root declaration names, the opaque accepted-trace handoff, diagnostics, accepted semantic content, fingerprints, and artifact bytes.

## Goal & Context
<!-- scope: business -->

Observation maintainers currently navigate evidence contracts, structural normalization, raw evaluation, provenance admission, and public orchestration in one very large module. These responsibilities should be independently buildable and testable so diagnostic or provenance work stays local without making Observation authors or downstream evaluators learn a wider interface.

Observation authors, Property and Implementation Link maintainers, Semantic Inventory, Run Evaluation, and artifact consumers see no source or behavior change. This is a maintainability refactor over the established Observation semantics.

## Architecture & Data Models
<!-- scope: technical -->

The stable Evaluation facade is backed by four child modules in one acyclic chain:

1. `Evaluation.Types` owns raw Evidence, status and diagnostic, structural support, Evidence Link, and unchecked-carrier data contracts.
2. `Evaluation.Structure` owns the existing internal normalized structural facts, findings, and `analyzeStructure` authority.
3. `Evaluation.Raw` owns bundle validation, expression evaluation, disposition processing, emission ordering, Evidence Link construction, structural-finding translation, and one internal unchecked-trace producer.
4. `Evaluation.Admission` co-locates the opaque accepted trace, its private constructor, Observation results, provenance reconstruction, and accepted-trace validation.

The stable Evaluation facade retains the module documentation, raw-to-admission result mapping, and `evaluateEvidence` orchestration. The public Observation umbrella keeps exporting the same surface, and existing direct Evaluation importers continue to import only the facade.

The opaque accepted type remains with its admission validator because moving it into a data-only module would require exporting a construction bypass and reopen the established hard seam. Shared implementation helpers remain internal to Observation; no generic support framework, public child-module interface, or pass-through common module is added.

## API Contracts
<!-- scope: technical -->

- Every existing Evidence input, support, Evidence Link, status, failure, diagnostic, result, unchecked carrier, accepted trace, validation, evaluation, and digest declaration retains its fully qualified name and observable behavior.
- Public inductive constructor order, structure field order and defaults, derived equality and rendering instances, classifier ordering, status mapping, and dot-notation projections remain exact.
- `EvidenceBackedTrace` remains opaque and constructible only after successful accepted-trace validation. Negative fixtures continue to use the unchecked carrier; no proof-only, unchecked, or compatibility constructor is exposed.
- `Observation.Internal.analyzeStructure` remains the single normalized authority for identity, source mode, sequence, causality, closure expectations, and per-link support.
- `evaluateEvidence` remains the sole public raw-evaluation entry point and returns the same accepted value or first diagnostic for the same checked plan and bundle.
- Existing direct imports of `Umpire.Observation.Evaluation` and the `Umpire.Observation` umbrella remain valid without consumer migration to child modules.

## Edge Cases & Constraints
<!-- scope: technical -->

- Empty, single-source, and multi-source input; duplicates; mixed origins; sequence gaps; missing parents; cycles; reverse edges; missing required kinds; and closure sequence, count, byte, or support inconsistencies retain normalized finding order.
- Bound, source-closure, Known Gap, profile, version, kind, field, binding, digest, disposition, ambiguity, fault-target, and emission-order validation retains its exact operation and first-failure precedence.
- Accepted admission retains checks for canonical plan identity, bounds, coordinates, link metadata, complete identity coverage, order and closure support, record support, field retention, digests, expressions, and trace identity.
- Invalid raw input produces no partial trace. Malformed unchecked wrappers produce no accepted value. Raw, redacted, or rejected material cannot cross the accepted seam.
- Raw and accepted paths may consume the same structural findings but retain their distinct diagnostic vocabularies, statuses, related-identity ordering, and semantic altitude.
- Raw and accepted paths each invoke structural analysis once. At ten times the evidence size, the split adds no second normalization or sort traversal, speculative cache, or copied intermediate pipeline.
- Evaluation remains pure total Lean code with no I/O, credential, runtime, concurrency, or crash-recovery surface. No trust assumption, axiom, or native evaluation path is added.
- Existing comments and docstrings move with their declarations and otherwise remain unchanged. Public architecture documentation changes only if the facade or internal ownership statement becomes false.

## Approach

1. Establish a data-only Evaluation contract module while temporarily leaving accepted construction and results with the facade.
2. Move the existing structural analyzer intact and give its current direct examples a focused test module.
3. Extract raw Evidence evaluation without changing validation order or exporting another evaluation entry point.
4. Co-locate accepted-trace admission and construction, reduce the facade to orchestration, and lock public and direct-import compatibility.
5. Run focused Observation, direct-consumer, aggregate model, artifact regression, trust, import, and lint gates.

## Quick commands

```bash
cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Umpire.ImplementationLink.Tests
cd model && mise exec -- lake build UmpireTests TemporalModelTests
make umpire-build-model
make umpire-check-regression
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

The accepted pre-edit Go lint baseline is 1,381 findings across the existing branch. Verification
must not exceed that global count, and `make lint-code GOLANGCI_LINT_FIX=false` with
`GOLANGCI_LINT_BASE_REV` set to this task's base commit must report zero diff-scoped findings.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Observation Evaluation is implemented by contract, structure, raw-evaluation, and accepted-admission child modules with an acyclic import graph, while the existing Evaluation and Observation facades remain unchanged for callers. Errors: an import cycle, missing transitive declaration, direct-importer migration, added root-level helper, public child-module dependency, or changed facade import behavior fails focused import and consumer builds; there is no runtime error surface beyond existing Observation results.
- **R2:** Evidence, diagnostic, support, Evidence Link, result, and unchecked-carrier contracts retain exact Lean names, constructors, field order and defaults, derivations, rendering, classifiers, and status mapping. Errors: constructor, field, instance, classifier, status, or projection drift, or exposure of the accepted constructor, fails import and equality checks; no new validation behavior is introduced.
- **R3:** `Observation.Internal.analyzeStructure` remains the single normalized authority for identity, origin mode, sequence, causality, closure expectations, and per-link support, with its current interface and finding order. Errors: empty inputs, duplicate identities, sequences or closures, mixed origins, gaps, missing parents, cycles, reverse edges, absent required kinds, closure sequence, count or byte mismatch, and inconsistent per-link support produce the same normalized findings in the same order.
- **R4:** Raw evaluation returns the same accepted unchecked semantic content or first `ObservationDiagnostic` for every existing bundle. Errors: empty, over-limit, open, or gapped Evidence; Known Gaps; profile, version, kind, field, binding, digest, disposition, ambiguity, alternative, fault-target, closure, and multi-source failures retain exact status, kind, related-identity order, Limit and count fields, precedence, and no-partial-trace behavior.
- **R5:** Accepted-trace admission retains the opaque established handoff and validates the same complete provenance envelope before construction. Errors: noncanonical plan identity, bound overflow, absent, duplicate, extra, zero, or out-of-range coordinates, inconsistent link metadata, incomplete identity coverage, order or closure support drift, malformed record support, forbidden field material, digest or expression mismatch, and trace-identity drift retain exact diagnostics, related identities, precedence, and no accepted value.
- **R6:** Focused structural, raw, and admission regressions; facade import checks; direct-consumer and aggregate builds; artifact regression; and model and code lint prove a behavior-neutral split. Errors: changed accepted traces, Evidence Links, fingerprints, persisted bytes, new trust dependency, warning, lost comment, extra traversal at tenfold evidence size, stale architecture text, a new lint finding, or an increase above the approved inherited exact 1,381-finding Go-lint baseline blocks completion; task-diff-scoped Go lint must report zero findings.

## Early proof point

The contract extraction must preserve all root names and derived instances while leaving opaque accepted construction private. The structural extraction must then reproduce the existing normalized finding matrix independently. If either requires a public construction helper, duplicate structural algorithm, or consumer import migration, reconsider the child-module chain before raw or admission work proceeds.

## Boundaries
<!-- scope: business -->

- No new Observation expression, rule, profile, status, diagnostic, Evidence, support, or accepted-trace semantics.
- No redesign of structural analysis, Model Coordinate semantics, accepted construction, authoring declarations, or diagnostic vocabularies established by prior specs.
- No new public evaluator, validator, helper framework, child-module import contract, compatibility overload, or generic graph library.
- No raw Evidence, artifact, schema, version, checksum, fingerprint, generated-code, runtime, or persisted-byte change.
- No cache, index, second traversal, callback, new dependency, axiom, or trust-policy change.

## Decision Context
<!-- scope: both — conditionally substructured -->

The decomposition follows the existing semantic seams rather than line count: contracts are inert data, structure is one normalized authority, raw evaluation owns Evidence interpretation, admission owns the opaque provenance handoff, and the facade owns orchestration. The accepted type stays with its validator to prevent an exported bypass. A generic common module is rejected because it would be a shallow pass-through and broaden internal details.

Completed Observation cleanup already centralized coordinates, sealed accepted traces, introduced field specifications, and centralized structural analysis. This plan moves those established units intact and must not reinterpret them. It follows the Observation authoring partition so each evaluator child can choose its final narrow language import once; the two specs must not execute concurrently because both adjust the public umbrella.

Additional files increase import-graph and navigation surface, offset by focused compilation and locality. Runtime and asymptotic performance, scalability, pure crash behavior, information-flow security, and trust remain unchanged.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1 | Acyclic child modules behind stable facades | `.1`–`.4` | — |
| R2 | Exact public contract preservation | `.1`, `.4` | — |
| R3 | Single structural authority and finding order | `.2` | — |
| R4 | Raw evaluation and diagnostic compatibility | `.3` | — |
| R5 | Opaque accepted-trace admission compatibility | `.4` | — |
| R6 | Focused, consumer, regression, trust, and lint verification | `.1`–`.4` | — |

## References

- Umpire 4 rules MOD-06 through MOD-08, AUT-01 through AUT-03, AUT-07, and EVD-02 through EVD-09 require small interfaces, explicit responsibilities, isolated testability, fail-closed checked Evidence, independent statuses, and auditable Evidence Links.
- Lean Authoring Guidelines sections 2, 4, 5, and 6 govern interfaces, module documentation, trust, and verification.
- Completed accepted-trace, coordinate, field, and structural-analysis cleanup specs define the established units and diagnostics preserved here.
- The Observation authoring partition is the active prerequisite that establishes the final upstream import seams.
