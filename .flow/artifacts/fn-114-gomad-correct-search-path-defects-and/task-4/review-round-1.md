I’ll review the frozen task-4 patch against its recorded baseline, trace the affected paths, and check the supplied evidence without rerunning broad tests.

The patch hash and all 19 frozen source hashes match. The new result path keeps divergence out of outcomes and failure signatures; I’m now checking candidate binding, journal validation, and resume behavior for gaps.

The journal binds retained divergence evidence to its candidate and checks the same evidence during resume. I found a possible gap in the validator’s `observation` case: it appears to accept evidence after the forced prefix ends. I’m tracing that case through the executor and commit paths before deciding whether to flag it.

- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `tools/gomad3/runner/internal/exploration/choice/engine.go:378`
- **R-IDs**: [R4]
- **Problem**: `DivergenceObservation` can bypass forced-prefix binding. For a one-decision prefix, evidence with `Ordinal=1`, `TapeRecords=1`, `Expected=nil`, and a valid observed decision at ordinal 1 passes validation. The delegated terminal validator allows ordinals beyond the tape in prefix mode and requires only an observed record for this reason. A custom executor returning that evidence therefore gets a committed divergence with no evidence matching the candidate’s forced decision. Under `budget` or `all`, exploration continues instead of preserving the HostError boundary; resume accepts the same evidence.
- **Suggestion**: Require retained divergence evidence to reference an ordinal within the forced prefix and include its matching expected decision. Add regression cases for observation divergence at and beyond the prefix boundary.

The patch SHA256 and all 19 frozen source hashes match. The supplied final host-test receipt binds those sources and its log matches the recorded hash. No tests were rerun.

## Requirements coverage

Coverage is limited to task 4’s assigned requirement.

| R-ID | Status | Evidence |
|---|---|---|
| R4 | partial | Sibling commits, divergence retention, failure policies, inspection, schema rejection, and SIGKILL/resume are implemented and covered by supplied tests. The observation-evidence binding gap remains. |

Unaddressed R-IDs: [R4]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R4"]}
```

<verdict>NEEDS_WORK</verdict>
