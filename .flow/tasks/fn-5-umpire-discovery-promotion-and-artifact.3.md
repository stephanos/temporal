---
satisfies: [R1]
---
# fn-5-umpire-discovery-promotion-and-artifact.3 Add exact Nexus explain output

## Description
Add one exact `explain` lookup over the same checked Nexus inventory used by `list`.

**Size:** S
**Files:** `model/Temporal/Tool/NexusDiscovery.lean`, `model/Temporal/Tool/NexusDiscoveryTests.lean`, `model/Temporal/Tool/Inspect.lean`, `model/Temporal/Tool/InspectTests.lean`
**Touches:** [model/Temporal/Tool/NexusDiscovery.lean, model/Temporal/Tool/NexusDiscoveryTests.lean, model/Temporal/Tool/Inspect.lean, model/Temporal/Tool/InspectTests.lean]

### Approach

- Resolve exactly one canonical query identity from the checked inventory without case folding,
  prefix matching, aliases, or semantic redirection.
- Emit one canonical `umpire-nexus-explanation/v1` value containing the matching list summary plus
  its checked Property/Behavior/Query and planned ExperimentSpec lineage.
- Reuse the list row encoder and diagnostic shell so shared fields cannot drift across commands.
- Pin all four successful identities and the unknown, case-shifted, empty, and extra-argument paths.

### Non-goals

- No generic reference graph, generated glossary, machine index, search language, or source-text explanation.

### Investigation targets

**Required:**
- `model/Temporal/Tool/NexusDiscovery.lean` — task `.1` inventory and task `.2` canonical row encoder.
- `model/Temporal/Tool/Inspect.lean` — existing CLI dispatch and failure result.
- `model/Temporal/Tool/InspectTests.lean` — exact diagnostics and compatibility expectations.
- `.plans/UMPIRE4_SPEC.md` — CLI-03 inspectability vocabulary.

### Quick command

`cd model && mise exec -- lake build Temporal.Tool.NexusDiscoveryTests Temporal.Tool.InspectTests temporal-model-inspect`

## Acceptance
- [ ] Each of the four canonical query identities returns one deterministic explanation consistent with its list row.
- [ ] Unknown, case-shifted, prefix, empty, ambiguous, and extra selectors fail with empty stdout and one exact structured diagnostic.
- [ ] Explanation lineage comes from checked values and the existing planned ExperimentSpec, never copied prose or source parsing.
- [ ] `list`, existing positional inspection, and their canonical outputs remain unchanged.
- [ ] Existing comments in touched files are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
