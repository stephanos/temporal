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
RuntimeConfiguration, ExperimentRun, and RawEvidence now carry checked KnownGapSet values; canonical serializers and direct Temporal consumers use read-only projection, while trusted fixtures use checked construction. Focused type and nonempty JSON regressions preserve phase/status/closure behavior, canonical empty fixture bytes, checksums, and generated-view identity.

Baseline: green via handoff (verified at f9556add by fn-48-canonicalize-known-gaps-as-a-checked-set.2). The pre- and post-edit make lint-code runs matched exactly at 1,379 inherited unrelated findings (errcheck 220, exhaustive 5, forbidigo 211, govet 5, revive 798, staticcheck 136, testifylint 4); the unrelated monitor_test.go auto-fix was restored both times. The Go gate required selecting Apple clang and the physical TMPDIR path to correct ambient toolchain/path resolution; the canonical command then passed unchanged.

stage: impl-review - ran [2026-09-03T07:07:12Z..2026-09-03T07:09:44Z] (Codex SHIP; 0 findings)
## Evidence
- Commits: f936174c61eb11ee451f76d1bfacf1cb2e4190ef
- Tests: cd model && mise exec -- lake build Umpire.Planning.Tests.KnownGaps Umpire.Artifact.Tests.Codecs Umpire.Artifact.Tests.Runtime Umpire.Artifact.Tests.Evidence Umpire.Artifact.Tests.Result Temporal.Tool.RunEvaluationTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests, go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/runevaluation, make umpire-check-regression, make lint-model, make lint-code (waived: exact inherited 1,379 findings; auto-fix restored)
- PRs: