I’ll read the task specification first, then trace the changed code and focused tests against its acceptance criteria.
The change adds descriptor-generated mirrors, claim metadata, and probe admission based on those claims. I’m checking whether suppression preserves explicit constraints and whether the queue and retry predictions use the actual request values.
The focused test command cannot run in this read-only environment because Cargo needs to write build artifacts. I’m continuing with code tracing and the committed accounting manifest, including the exact disposition of every removed case.
I found a gap in the new suppression policy: map values declared as `Secret` become `Bytes` in claim metadata, so their unsupported length probes remain admitted. The manifest confirms nine such cases without predictions. The cached accounting test passes because it expects those cases to remain unchanged.
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/catalog/build/src/claims.rs:28`
- **R-IDs**: [R3, R4]
- **Problem**: `let kind = field.kind.to_string()` reads the coverage representation, which assigns `Bytes` to synthetic map-entry fields. Consequently, `Payload.metadata: Map<Text, Secret>` generates a `Bytes` claim for `Payload.MetadataEntry.value`. Temporal’s new admission policy only suppresses bounds whose claim kind is `Secret`, so nine applicable length probes without predictions survive, including `StartActivityExecutionRequest.input.payloads.metadata.value:mutation.limit_plus_one(1000)`. This violates the required suppression of unsupported inferred Secret bounds.
- **Suggestion**: Preserve semantic map-value kinds when generating claims. Suppress these unsupported Secret bounds while retaining mutations that trigger explicit ancestor constraints, such as Start’s input-size limit. Update the reconciliation manifest and add map-value coverage.

Fresh Cargo verification was blocked by the read-only filesystem. Cached binaries passed the accounting test, 12 catalog tests, two mirror tests, and the opaque-admission test; those checks currently accept this omission.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Reach/frontier work belongs to other epic tasks. |
| R2 | deferred | Capability migration belongs to other epic tasks. |
| R3 | partial | Descriptor mirrors and claims implemented; semantic map-value kinds are lost. |
| R4 | partial | Per-ID accounting and conditional predictions implemented; unsupported Secret bounds remain. |
| R5 | deferred | Protocol/refinement work belongs to other epic tasks. |
| R6 | deferred | Worker-policy work belongs to other epic tasks. |
| R7 | deferred | Mapping measurements belong to tasks .3/.6. |

Unaddressed R-IDs: [R3, R4]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R3","R4"]}
```

<verdict>NEEDS_WORK</verdict>