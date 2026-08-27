---
satisfies: [R2, R3, R4, R5]
---
# fn-38-consolidate-layered-model-helpers.7 Pin facade and complete fixture-value compatibility

## Description
Add value-level compatibility assertions after every production and fixture migration. This task covers the source-compatible facades and every distinct Umpire fixture constructor shape for R2, R3, R4, and R5.

**Size:** M
**Files:** `model/Umpire/Tests/MigrationCompatibility.lean`, `model/Temporal/Feature/Nexus/LifecycleTests.lean`, `model/Temporal/Feature/Nexus/OperationsTests.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean`
**Touches:** [model/Umpire/Tests/MigrationCompatibility.lean, model/Temporal/Feature/Nexus/LifecycleTests.lean, model/Temporal/Feature/Nexus/OperationsTests.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean]

### Approach
- Import the original public facades and all six Umpire fixture modules; never use a shared module as a substitute consumer API.
- Assert complete `SourceLocation` and `DefinitionMetadata` values for every distinct fixture constructor shape, including Target's parameterized path/digest default, each concern source path, Query/Planning documentation, and empty-documentation families.
- Pin existing fully qualified declarations, types, visibility, Definition IDs, metadata-sensitive behavior, and canonical/serialized outputs for Switch and the affected Nexus facades.
- Keep experimental compatibility coverage in the existing experimental test root.
- Preserve command-based checks and comments whose purpose is import, visibility, source, or diagnostic testing.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Tests/MigrationCompatibility.lean:1-16,59-61` — public migration inventory and Switch source pin.
- `model/Umpire/Target/Tests/Fixtures.lean:9-26` — parameterized path and digest/default shape.
- `model/Umpire/Query/Tests/Fixtures.lean:9-37` — query documentation/default shape.
- `model/Umpire/Planning/Tests/Fixtures.lean:9-37` — planning documentation/default shape.
- `model/Umpire/Behavior/Tests/Fixtures.lean:9-25` — derived canonical behavior and empty documentation shape.
- `model/Temporal/Feature/Nexus/LifecycleTests.lean:35-46,135-136` — existing identity/source coverage.

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/OperationsTests.lean:38-56,199-204` — current identity and canonical JSON coverage.

## Acceptance
- [ ] All six Umpire fixture families have value-level coverage for complete source and metadata fields, with every distinct default/documentation shape pinned.
- [ ] Original public facade imports and fully qualified names compile without consumer changes.
- [ ] Switch, Lifecycle, Operations, and Caller Closure preserve exact source, identity, metadata-sensitive, canonical, and experimental behavior.
- [ ] Existing import/visibility/source test comments and command checks remain intact.
- [ ] `cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
