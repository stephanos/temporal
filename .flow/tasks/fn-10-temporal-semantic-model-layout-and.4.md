---
satisfies: [R2, R4, R7]
---
# fn-10-temporal-semantic-model-layout-and.4 Move caller-closure scenario and feature tests

## Description
Move the bounded caller-closure target, scenario, real Workflow/Nexus composition coverage, and canonical fixture ownership into `Temporal.Feature.Nexus` (R2, R4, R7). Keep the former scenario module as an import-only bridge until the inspector moves in task 6.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/CallerClosure.lean`, `model/Temporal/Feature/Nexus/CallerClosureTests.lean`, `model/Temporal/Umpire/NexusCallerClosure.lean`, `model/TemporalUmpireTests.lean`, `model/Temporal/Umpire/testdata/nexus-caller-closure-experiment-spec.json`
**Touches:** [model/Temporal/Feature/Nexus/CallerClosure.lean, model/Temporal/Feature/Nexus/CallerClosureTests.lean, model/Temporal/Umpire/NexusCallerClosure.lean, model/TemporalUmpireTests.lean, model/Temporal/Umpire/testdata/nexus-caller-closure-experiment-spec.json]

### Approach
- Update imports and opens to the Feature auto-close namespace from task 3.
- Extract only the real Workflow/Nexus model, property, behavior, planning, and artifact assertions from the mixed aggregate into the colocated Feature test module.
- Preserve scenario/query identity and every semantic digest; update source provenance and the transitional golden content only where the physical move makes it truthful.
- Leave inspector-specific assertions for task 6 and System assertions in their System test modules.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Umpire/NexusCallerClosure.lean:1-716` — complete checked scenario and compiled artifact
- `model/TemporalUmpireTests.lean:1-351` — mixed Feature and inspector coverage to separate
- `model/Temporal/Umpire/testdata/nexus-caller-closure-experiment-spec.json` — current canonical artifact
- `model/Umpire/Examples/Switch.lean:7-613` — domain-neutral scenario using the same public Umpire interfaces

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/AutoClose.lean` — Feature model created by task 3

### Acceptance
- [ ] CallerClosure compiles and plans through public Umpire interfaces under the Feature namespace.
- [ ] Real Workflow/Nexus composition and property coverage exists only in the Temporal Feature tests.
- [ ] Query identity, declaration identities, digests, planner order, validation, and portable artifact fields are unchanged.
- [ ] Golden differences are limited to truthful source provenance.
- [ ] Existing comments remain attached to moved declarations and assertions.

## Acceptance
- [ ] Feature scenario and colocated tests compile under the new namespace.
- [ ] Artifact and proof behavior remain stable except approved source paths.
- [ ] Product composition coverage no longer depends on reusable test fixtures.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
