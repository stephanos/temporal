---
satisfies: [R19]
---
# fn-93-simplify-the-lean-model.35 One in-Lean comparison per golden and one Switch plan golden (D-goldens)

## Description
Lane D-goldens. `Umpire/Artifact/Tests/Goldens.lean` becomes the only in-Lean byte comparison of a golden: delete the surviving duplicate `include_str` comparisons (`Artifact/Tests/Codecs.lean:61`, `Examples/SwitchTests.lean:28`, and any left after B3/B4) and duplicate checksum pins. DG2: keep `Umpire/Artifact/Tests/Fixtures/SwitchPlanV2.json`, delete the byte-identical `Umpire/Examples/Fixtures/SwitchCompiledArtifact.json`, and move `internal/artifactv2`'s test to the survivor.

**Size:** S
**Files:** `model/Umpire/Artifact/Tests/Goldens.lean`, `model/Umpire/Artifact/Tests/Codecs.lean`, `model/Umpire/Examples/SwitchTests.lean`, `model/Umpire/Examples/Fixtures/SwitchCompiledArtifact.json`, `model/Temporal/Tool/Goldens.lean:44,51`, `tools/umpire/internal/artifactv2/artifact_test.go:15-27`
**Touches:** [model/Umpire/Artifact/Tests/**, model/Umpire/Examples/**, model/Temporal/Tool/Goldens.lean, tools/umpire/internal/artifactv2/**]

### Approach
- Record DG2 (default: keep `SwitchPlanV2.json`) in the Done summary.
- `cmp` the two JSON files first; if they differ at start, stop and report.
- `Goldens.lean` keeps a comparison for every surviving golden.

### Quick commands
```sh
cd model && lake build
make umpire-check-goldens
go test ./tools/umpire/internal/artifactv2/...
```

## Acceptance
- [ ] Each surviving golden has exactly one in-Lean comparison, in `Goldens.lean`
- [ ] One Switch plan golden; Go test reads it and passes
- [ ] `make umpire-check-goldens` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
