---
satisfies: [R1]
---
# fn-5-umpire-discovery-promotion-and-artifact.1 Define the closed Nexus discovery inventory

## Description
Create the concrete checked inventory that is the sole input to retained Nexus `list` and `explain`.

**Size:** S
**Files:** `model/Temporal/Tool/NexusDiscovery.lean`, `model/Temporal/Tool/NexusDiscoveryTests.lean`, `model/TemporalExperimentalTests.lean`
**Touches:** [model/Temporal/Tool/NexusDiscovery.lean, model/Temporal/Tool/NexusDiscoveryTests.lean, model/TemporalExperimentalTests.lean]

### Approach

- Define one private-constructor `NexusDiscoveryEntry` from existing checked Property, Behavior,
  Query, source, Behavior Fingerprint, and planned `ExperimentSpec` identities.
- Register exactly the async-start, cancellation, successful-completion, and exact-action
  caller-closure examples in canonical query-identity order.
- Validate the entire inventory before exposing it: exact four-row membership, unique and correctly
  owned identities, expected declaration kinds, nonempty source/fingerprint values, and present
  plans; canonicalize valid input permutations by query identity before constructing the checked value.
- Keep semantic values in their current owning modules; this adapter projects identities only.

### Non-goals

- No generic semantic graph, generated glossary, machine index, source scan, or broad regression set.

### Investigation targets

**Required:**
- `model/Temporal/Feature/Nexus/Operations/AsyncStart.lean` — first ordinary checked example.
- `model/Temporal/Feature/Nexus/Operations/Cancellation.lean` — second ordinary checked example.
- `model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean` — third ordinary checked example.
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean` — retained experimental example.
- `model/Temporal/Tool/Inspect.lean` — current scenario registry and exact diagnostics.
- `model/Temporal/Tool/InspectTests.lean` — current inspector compatibility tests.

### Quick command

`cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests`

## Acceptance
- [ ] The inventory contains exactly the four named Nexus query examples in canonical query-identity order.
- [ ] Every row is constructed from existing checked Property, Behavior, Query, source, fingerprint, and planned ExperimentSpec values without copied semantic prose.
- [ ] Duplicate, missing, wrong-kind, crossed-owner, and missing-plan fixtures fail; reordered valid fixtures produce the same checked inventory and bytes.
- [ ] The module performs no source scan and imports only the concrete Nexus owners it projects.
- [ ] Existing comments in touched files are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
