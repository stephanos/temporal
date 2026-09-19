---
satisfies: [R5, R7]
---
# fn-15-standalone-api-and-config-input-catalogs.5 Generalize the six owner-defined config uses

## Description
Add checked type-erasable owner definitions for the existing four Callback plus two Matching uses without moving their typed decoders, composing the two owner registries, or changing semantic ownership for R5/R7.

**Size:** M
**Files:** `model/Temporal/System/Configuration/Core.lean`, `model/Temporal/System/Callback/Configuration.lean`, `model/Temporal/System/Matching/Configuration.lean`, `model/Temporal/System/Configuration/Tests/Validation.lean`, `model/Temporal/System/Configuration/Tests/Catalog.lean`
**Touches:** [model/Temporal/System/Configuration/Core.lean, model/Temporal/System/Callback/Configuration.lean, model/Temporal/System/Matching/Configuration.lean, model/Temporal/System/Configuration/Tests/Validation.lean, model/Temporal/System/Configuration/Tests/Catalog.lean]

### Approach
- Introduce checked use definitions, closed context policies, erased explain metadata, and typed instantiation in the shared core.
- Refactor the six existing owner constructors to consume owner-defined checked definitions while retaining return types and decoders.
- Expose one exact four-use Callback list and one exact two-use Matching list; leave their cross-owner composition to Task 6's higher-level `Temporal.Tool.Catalog.Config` adapter.
- Validate duplicate IDs/keys, setting identity, schema/default, impacts/digest, context shape, and sampling/change compatibility.
- Prove unrelated unclassified settings remain valid and existing resolutions are unchanged.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Configuration/Core.lean:11-84,108-174,404-468,739-798` — current checked classification/use/resolution contracts.
- `model/Temporal/DynamicConfig/Types.lean:75-122` and `Settings.lean:6859,7546-7548` — structural authority and identity.
- `model/Temporal/System/Callback/Configuration.lean:258-358` — four owned definitions.
- `model/Temporal/System/Matching/Configuration.lean:19-77` — two owned definitions.
- `model/Temporal/System/Configuration/Tests/Catalog.lean:13-50` — existing use tests.

### Quick command
`cd model && lake env lean Temporal/System/Configuration/Tests/Validation.lean && lake env lean Temporal/System/Configuration/Tests/Catalog.lean`

## Acceptance
- [ ] Callback exposes exactly the four named use/key pairs and Matching exposes exactly the two named use/key pairs from the parent spec, without executing a decoder or defining a cross-owner aggregate.
- [ ] Existing typed constructors retain current return types, owner decoders, and successful resolution behavior.
- [ ] Duplicate use/classification identities, unknown/stale setting, schema/default drift, empty impacts/digest, malformed context, and incompatible lifecycle policy fail with existing config errors where applicable.
- [ ] The shared configuration core imports neither Callback nor Matching.
- [ ] Owner validation proves the six keys are distinct and an unrelated unclassified generated setting remains valid without invented meaning; aggregate list/explain behavior belongs to Task 6.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
