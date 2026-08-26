# Standalone API and config input catalogs

> HTML render lens: local file `.flow/artifacts/fn-15-standalone-api-and-config-input-catalogs/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Overview

Provide deterministic standalone JSON views of the mechanical API and dynamic-config inputs available to the current model. Both domains support canonical `list`, exact `explain`, and invariant-only `check` operations through one reusable pure `Temporal.Tool.Catalog.Core` engine, while retaining separate domain records, authorities, and format versions.

The API side derives a complete logical descriptor catalog from the existing projection and Lean declaration plan, renders it into the existing three-file generated API boundary, and overlays a bounded handwritten current-model selection policy. The config side projects the complete generated settings registry and explains exactly the six semantic uses already authored by Callback and Matching. The CLI consumes compiled checked Lean values and never regenerates, compares, repairs, or persists source artifacts.

## Goal & Context
<!-- scope: business -->

Model authors currently have to read a very large generated Protobuf tree and several configuration modules to answer basic questions: which declarations exist, how each API field maps to Lean, which inputs the current model selects, and which generated settings already have authored meaning. Missing or silently skipped field mappings are particularly dangerous because later evidence adapters could infer treatment from absence.

This slice makes every API logical field accountable and gives both input domains a small inspectable interface without creating another semantic model. The query mechanics are shared with fn-5's semantic catalog; input facts, selection policy, config meaning, JSON shapes, and executable remain domain-owned and separate.

## Architecture & Data Models
<!-- scope: technical -->

```text
merged descriptors
  -> existing projection + Lean plan
  -> complete generated API catalog facts in Temporal.API
  -> handwritten bounded current-model selection --+
                                                      |
DynamicConfig.Settings.all ---------------------------+--> Catalog.Core
  + six checked Callback/Matching use definitions ---+      |
                                                             v
                                                   temporal-input-catalog JSON
```

`Temporal.Tool.Catalog.Core` is a pure parameterized deep module. A catalog definition supplies its stable name, format version, catalog identity, closed item kinds and views, exact canonical items, and a domain validator. The engine owns selector parsing, item uniqueness, lexicographic ordering by `(kind, identity)`, canonical response/error envelopes, and list/explain/check dispatch. It knows nothing about Protobuf, configuration, semantic declarations, IO, files, generation, or runtime state.

Selectors are `<kind>:<canonical-identity>`, split on the first colon. Identities are exact, non-empty, case-sensitive, and fully qualified. There are no aliases, fuzzy matches, substrings, pagination, cursors, or silent redirects. A valid empty view succeeds; an unknown explain subject fails.

### Complete API catalog

The existing descriptor projection and completed Lean plan jointly derive `umpire-api-input-catalog/v1`. Every enum, enum value, message, real oneof, logical field, service, and method records canonical Protobuf identity, source descriptor path, generated Lean identity where supported, structural flags, deprecation, and relevant type metadata.

Every logical field of every non-map-entry message appears exactly once:

- maps appear as one logical map field; synthetic map-entry key/value declarations do not become catalog fields;
- proto3 optional fields appear as fields; their synthetic oneof containers do not become real oneofs;
- real oneofs and their members remain explicit;
- nested and imported declarations remain present;
- recursive fields record `MessageRef` lowering; and
- extensions remain present with an explicit unsupported disposition until ordinary Lean field projection supports them.

Projection dispositions are closed: `lean-structure-field`, `lean-oneof-constructor`, or `unsupported-extension`. Supported logical fields must match exactly one planned Lean field or one planned real-oneof constructor. Unsupported ordinary fields, missing planned mappings, duplicate logical fields, or silently absent extensions fail generation before publication.

Catalog structures and values are rendered into the existing generated `Temporal.API` umbrella; output ownership remains exactly `API.lean`, `API/Proto.lean`, and `API/Types.lean`. No fourth generated file or retained JSON artifact is added.

### Bounded current-model API policy

`temporal-current-model/v1` is a handwritten policy over the complete generated catalog. Its exact v1 root manifest is the following nine message identities, in canonical order:

```text
temporal.api.history.v1.NexusOperationCancelRequestCompletedEventAttributes
temporal.api.history.v1.NexusOperationCancelRequestFailedEventAttributes
temporal.api.history.v1.NexusOperationCancelRequestedEventAttributes
temporal.api.history.v1.NexusOperationCanceledEventAttributes
temporal.api.history.v1.NexusOperationCompletedEventAttributes
temporal.api.history.v1.NexusOperationFailedEventAttributes
temporal.api.history.v1.NexusOperationScheduledEventAttributes
temporal.api.history.v1.NexusOperationStartedEventAttributes
temporal.api.history.v1.NexusOperationTimedOutEventAttributes
```

The policy schema accepts one to 32 unique exact message or method roots and fixes the maximum message-reference depth at four. Message roots seed themselves at depth zero. A method root selects the method as a root, selects its containing service as a dependency, and seeds its request and response messages at depth zero; selecting a service never selects its other methods, and no non-root method is selected. A selected message includes all of its logical field items and real-oneof containers. A referenced enum and every value of that enum are terminal dependencies at the owning message's depth. One traversal hop follows a message-valued ordinary field or map value and increments message depth by one; scalars, enum references, map keys, and map values that are not messages terminate.

Traversal is deterministic breadth-first traversal. The canonical candidate key is `(depth, root identity, edge path, message identity)`, where an edge path is the ordered list of fully qualified field identities and method request/response seed labels from that root. Queue order and tie-breaking use this full key, not descriptor order. Each work item carries its ancestor-message path. A message edge that would exceed depth four or re-enter that active path is not expanded and emits a `BoundaryEdge` record containing the owner field, target message, root, path, stopping depth, and `depth|cycle` reason. The target message is `selected-boundary` only when it has no admissible root/dependency path; boundary-edge metadata remains visible even when another admissible path selects that target.

Every item has one final disposition with precedence `selected-root` over `selected-dependency` over `selected-boundary` over `outside-selection`. Among admissible dependency paths the smallest canonical candidate key supplies provenance. A message selected through an admissible path is expanded once using that winning path; a later better path replaces provenance and is processed before any larger pending key. Explicit roots always retain root disposition and empty message path. The `all` view contains the complete generated catalog; `current-model` contains roots, dependencies, boundary-only target messages, and all boundary-edge metadata. The policy annotates rather than prunes the generated API surface.

### Config catalog and six-use overlay

`umpire-config-input-catalog/v1` starts from every entry in `Temporal.DynamicConfig.Settings.all` and its generated catalog identity. Structural facts remain authoritative for keys, descriptions, schemas, codecs, defaults, precedence/scope policy, provenance, fixtures, and setting identities. Opaque defaults remain explicitly opaque.

Configuration gains a checked, type-erasable use-definition layer. Typed owner-module definitions keep their decoders; an erased checked view exposes only explainable metadata. A use definition records namespaced use identity, setting key and identity, schema/default expectations, non-empty classification impacts, non-empty semantic digest, sampling point, change effect, and a closed context policy: `global`, `namespace`, `namespace-destination`, or `task-queue`.

The combined registry contains exactly four Callback and two Matching definitions over six distinct generated setting keys:

- `temporal.callback.enable-chasm` -> `history.enablechasmcallbacks`;
- `temporal.callback.max-per-execution` -> `callback.maxperexecution`;
- `temporal.callback.allowed-addresses` -> `callback.allowedaddresses`;
- `temporal.callback.request-timeout` -> `callback.request.timeout`;
- `temporal.matching.update-ack-interval` -> `matching.updateackinterval`; and
- `temporal.matching.worker-registry-num-buckets` -> `matching.workerregistrynumbuckets`.

`config list all` contains every one of the current 685 generated `setting:<key>` items plus exactly the six `classification:<key>` and six `use:<id>` items above: 697 items in v1. `config list semantic-uses` contains exactly the six setting/classification/use triples above: 18 items. The counts are validation output and golden tests pin both the complete selector sets and canonical order. A future generated-settings change deliberately changes the `all` setting count and catalog identity, while the six-use view changes only through reviewed owner definitions.

Sampling/change compatibility is closed: `live-access`, `request`, or `task` sampling requires `next-read`; `entity-creation` requires `new-entities-only`; `process-startup` requires `restart-required`. Existing configuration errors remain authoritative where applicable. The shared configuration core never imports Callback or Matching; the tool adapter composes their erased definitions. An unclassified generated setting remains valid and explains with no classification and an empty use list.

## API Contracts
<!-- scope: technical -->

The standalone command grammar is:

```text
temporal-input-catalog api list [all|current-model]
temporal-input-catalog api explain <kind>:<canonical-identity>
temporal-input-catalog api check

temporal-input-catalog config list [all|semantic-uses]
temporal-input-catalog config explain <kind>:<canonical-identity>
temporal-input-catalog config check
```

Omitting the list view selects `all`. API kinds are `enum`, `enum-value`, `message`, `oneof`, `field`, `service`, and `method`. Config kinds are `setting`, `classification`, and `use`.

List responses contain format version, catalog name/identity, operation, view, and canonically ordered items. Explain contains the same identity fields plus exact kind, identity, and one domain-owned subject. Check contains the identity fields, `status: valid`, and deterministic validation counts. Compact UTF-8 JSON has fixed object-field order and exactly one final line feed.

Success returns status 0, stdout only, and empty stderr. `invalid-arguments`, `malformed-selector`, `unknown-kind`, `unknown-view`, `unknown-subject`, `duplicate-subject`, `invalid-catalog-item`, and `catalog-invalid` return status 1, empty stdout, and one canonical stderr JSON document.

Root Make exposes list, explain, and combined integrity-check commands. `CATALOG` is required and restricted to `api|config`; `VIEW` defaults to `all`; `SUBJECT` is required for explain. The combined check consumes only compiled values and invokes neither generator.

## Edge Cases & Constraints
<!-- scope: technical -->

- Descriptor file/order changes must not alter the canonical API catalog or identity.
- An independent raw-descriptor logical-field walk must equal the derived catalog field set exactly, including map, optional, oneof, recursive, nested, imported, proto2 required/default/packed, and extension cases.
- The generator still renders and publishes all three API artifacts only after the complete projection and plan validate; a catalog failure cannot publish a partial tree.
- Current-model policy rejects any root manifest other than the exact nine-root v1 manifest, as well as empty, duplicate, wrong-kind, unknown, or more than 32 roots; any depth other than four; missing/non-minimal provenance; conflicting selection dispositions; or inconsistent cycle/depth boundary-edge metadata.
- Config validation rejects duplicate use IDs across owners, duplicate classification keys within one owner, unknown keys, setting-identity mismatch, empty impacts/digests, schema/default drift, malformed context, or incompatible sampling/change-effect pairs.
- `config explain setting:<key>` succeeds for an unclassified setting; `classification:<key>` for that same key is an exact unknown subject.
- List/explain/check read compiled values only. Missing or stale generated sources fail the normal build; the command never repairs, regenerates, or compares them.

## Quick commands

```bash
go test -count=1 -tags test_dep ./tools/umpire/internal/generate/api
cd model && mise exec -- lake build Temporal.Tool.Catalog.QueryTests Temporal.System.API.CatalogTests Temporal.System.Configuration.Tests.Validation Temporal.Tool.InputCatalogTests TemporalModelTests temporal-input-catalog
cd model && lake exe temporal-input-catalog api list current-model
cd model && lake exe temporal-input-catalog config explain use:temporal.callback.max-per-execution
make umpire-list-input-catalog CATALOG=api VIEW=current-model
make umpire-explain-input-catalog CATALOG=config SUBJECT=use:temporal.callback.max-per-execution
make umpire-check-input-catalogs
make umpire-build-model
```

## Acceptance Criteria
<!-- scope: both -->

- **R1:** One pure `Temporal.Tool.Catalog.Core` supplies exact list/explain/check behavior, canonical sorting/envelopes, selector parsing, and structured errors without importing either domain. Errors: malformed selectors, unknown kinds/views/subjects, duplicate subjects, invalid items, and failed domain validation return the exact failure channel/status contract; valid empty views succeed. [paraphrase]
- **R2:** The complete compiled API catalog is available as `umpire-api-input-catalog/v1` through deterministic `all`/`current-model` list, exact explain, and invariant check. Errors: empty/duplicate identities, unknown references, invalid catalog identity, inconsistent generated metadata, or source-order sensitivity fail before a checked catalog exists. [paraphrase]
- **R3:** Every logical descriptor field, including map, proto3 optional, real oneof, recursive, nested/imported, proto2 required/default/packed, and extension cases, appears exactly once with one closed projection disposition and complete planned Lean mapping facts where supported. Errors: duplicate fields, unmatched plans, synthetic-container leakage, missing extension disposition, or unsupported ordinary fields fail before publication. [paraphrase]
- **R4:** `temporal-current-model/v1` fixes the nine named Nexus history-message roots, accepts only the exact reviewed manifest, and applies exact depth-four canonical traversal, minimal deterministic path provenance, explicit stopped-edge metadata, and four closed selection dispositions over the complete catalog without pruning generated API declarations. Errors: invalid roots/depth, missing or non-minimal provenance, bound overflow, or inconsistent cycle/boundary treatment fail API checking. [paraphrase]
- **R5:** The config catalog exposes every generated setting and exactly the existing four Callback plus two Matching checked use definitions, preserving typed owner decoders and existing config errors. Errors: duplicate use/classification identities, unknown/stale settings, schema/default drift, empty impacts/digests, malformed context, or incompatible lifecycle policy fail aggregate checking; unrelated unclassified settings remain valid. [paraphrase]
- **R6:** `temporal-input-catalog` and root Make expose the exact API/config list, explain, and check grammar with canonical JSON and status/stdout/stderr behavior. Errors: missing or invalid `CATALOG`, `VIEW`, `SUBJECT`, operation, arity, or selector fail without success output. [paraphrase]
- **R7:** Focused Go/Lean tests prove complete field coverage, determinism, selector behavior, selection bounds, all six config uses, executable output, and root commands while preserving existing comments and domain ownership. No retained JSON, generator drift comparison, CI/workflow, API pruning, live config lookup, complete classification obligation, runtime, evidence, conformance, or semantic IR is introduced. [user]
- **R8:** Fn-15 publishes and tests `Temporal.Tool.Catalog.Core` as the generic query contract and records fn-5 as its reverse consumer; actual semantic-adapter consumption remains fn-5 implementation work. API/config adapters remain Temporal-owned, and no Temporal vocabulary enters reusable `model/Umpire`. No Umpire3 dependency, inspection, invocation, artifact, schema, reference, compatibility, or implementation reuse is permitted. [user]

## Early proof point

Task `.2` is the proof gate. Using an independent raw-descriptor logical-field walker, the existing integrated fixture, and an inline extension descriptor, it must prove that raw logical-field count and identity set exactly equal the derived catalog; synthetic map/optional containers do not leak; every supported field maps to exactly one planned Lean name/type; extensions remain explicitly unsupported; every listed edge case receives its expected disposition; and reversing descriptor files leaves values and identity unchanged. Failure blocks rendering and selection work and requires revisiting the shared projection/plan derivation rather than adding a second descriptor interpretation.

## Boundaries
<!-- scope: business -->

- No fourth generated API artifact; ownership remains exactly the existing three generated files.
- No generated-output drift verification, temporary regeneration/diff, stale-check target, persisted JSON, or CI/GitHub Actions change.
- No pruning or replacement of the complete generated API surface and no second Protobuf projection authority.
- No live dynamic-config values, YAML/converter interpretation, or requirement to classify every setting.
- No runtime, observation, evidence, conformance, exploration, replay, promotion, qualification, or new semantic IR.
- No human-text mode in v1; canonical JSON is the supported output.
- No Umpire3 dependency, inspection, invocation, artifact, schema, reference, sharing, compatibility, migration, or implementation reuse.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

The catalog must answer both “what did the importer project?” and “what does the current model select?” Complete generated facts make omission explicit; a separate bounded policy keeps current-model scope intentional and reviewable.

### Implementation Tradeoffs
<!-- scope: technical -->

Deriving facts from the existing neutral projection and Lean plan avoids a second descriptor interpreter and lets an independent walker test completeness. Rendering into the existing API umbrella preserves the exact three-artifact ownership contract. Compiled-value checks intentionally do not prove generated-source freshness because drift verification and CI coverage were explicitly declined. Config use definitions become type-erasable only after their typed owner checks, so explainability does not move decoders or product meaning into the generic engine.

## References
<!-- scope: technical -->

- `fn-6-plan-simplify-lean-api-generator-output` — exact generated API module and publication boundary.
- `fn-7-migrate-lean-model-descriptor-generator` — deterministic descriptor acquisition and shared tooling.
- `fn-8-umpire-temporal-dynamic-config` — complete generated settings catalog and typed resolution layer.
- `fn-5-umpire-discovery-promotion-and-artifact` — reverse consumer of the generic query core.
- Declined generated API drift-verification decision — no source comparison or CI gate.

## Requirement coverage
<!-- scope: both -->

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Shared generic catalog querying | `.1`, `.6` | — |
| R2 | Standalone complete API projection | `.2`, `.3`, `.6` | — |
| R3 | Complete mechanical field dispositions | `.2`, `.3` | — |
| R4 | Bounded current-model API selection | `.4`, `.6` | — |
| R5 | Six selected config uses | `.5`, `.6` | — |
| R6 | CLI and root Make usability | `.6`, `.7` | — |
| R7 | Verification and strict boundaries | `.1`-`.7` | — |
| R8 | Shared-core ownership and Umpire boundary | `.1`, `.6`, `.7` | — |
