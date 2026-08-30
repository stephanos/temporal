---
satisfies: [R4]
---
# fn-48-canonicalize-known-gaps-as-a-checked-set.3 Migrate Runtime and Evidence Known Gaps

## Description
Carry checked collections through Runtime Configuration, Experiment Run, and Raw Evidence plus their direct Temporal consumers (R4). Interpreted Evidence remains with Result in task `.4`.

**Size:** M
**Files:** `model/Umpire/Artifact/Runtime.lean`, `model/Umpire/Artifact/Evidence.lean`, `model/Umpire/Artifact/Tests/Runtime.lean`, `model/Umpire/Artifact/Tests/Evidence.lean`, `model/Temporal/System/Execution/Nexus.lean`, `model/Temporal/NexusExecutionIntegrationTests.lean`
**Touches:** [model/Umpire/Artifact/Runtime.lean, model/Umpire/Artifact/Evidence.lean, model/Umpire/Artifact/Tests/Runtime.lean, model/Umpire/Artifact/Tests/Evidence.lean, model/Temporal/System/Execution/Nexus.lean, model/Temporal/NexusExecutionIntegrationTests.lean]

### Approach
- Replace per-document raw-list validity predicates with checked semantic values and canonical projection.
- Update the System execution renderer/validation and integration fixture to use read-only projection or checked construction.
- Preserve all non-Known-Gap phase validation/status precedence and pin empty/non-empty canonical JSON.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/Runtime.lean:101-109,224-230,440-456,541-551` — runtime fields and duplicate validity gates
- `model/Umpire/Artifact/Evidence.lean:89-126,203-221` — Raw Evidence rendering and validation
- `model/Temporal/System/Execution/Nexus.lean:400-425,545-565` — Runtime Configuration projection/validation consumer
- `model/Temporal/NexusExecutionIntegrationTests.lean:365-390` — nonempty Runtime Configuration fixture
- `model/Umpire/Artifact/Tests/Runtime.lean` — runtime negative-case style
- `model/Umpire/Artifact/Tests/Evidence.lean` — Raw Evidence identity and mutation coverage
## Acceptance
- [ ] Runtime Configuration, Experiment Run, and Raw Evidence semantic values expose only checked Known Gaps.
- [ ] System execution and integration consumers compile through checked construction/projection with exact output.
- [ ] Existing phase status, provenance, closure, Limit precedence, canonical fixture bytes, and checksums remain exact.
- [ ] Runtime, Raw Evidence, System execution, and `TemporalExperimentalTests` pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
