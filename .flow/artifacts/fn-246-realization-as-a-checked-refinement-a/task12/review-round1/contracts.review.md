I’ll read the task contract first, then review the diff and focused tests for agreement between the implementation and its stated guarantees.
The patch adds descriptor-generated mirrors and a Temporal-specific probe admission policy. I’m checking that the policy preserves explicit constraints and that each removed or strengthened probe matches the committed reconciliation.
Cargo verification is blocked by the read-only build cache. The task’s command also names a package that no longer exists; after substituting `umpire-temporal-annotate-syntax`, Cargo reached the filesystem block. I’m continuing with source and artifact checks.
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/catalog/build/src/claims.rs:28`
- **R-IDs**: [R3, R4]
- **Problem**: Generated claims lose map value kinds. `Payload.metadata` declares `Map<Text, Secret>`, but `catalogs()` represents its synthetic entry fields as `Bytes`; copying `field.kind` therefore records `MetadataEntry.value` as `Bytes`. Temporal’s new admission policy checks `claim.kind == "Secret"`, so it retains unsupported Secret length mutations for metadata values while suppressing equivalent mutations for `Payload.data`. The committed reconciliation confirms nine retained metadata-value bound cases with no prediction, contradicting the task’s Secret-bound suppression contract.
- **Suggestion**: Preserve authored map key/value kinds when generating claims. Add coverage showing that unsupported metadata-value bounds are suppressed while the explicit Start input-size constraint remains enforced.

Verification could not run: Cargo requires writes to the read-only build cache. The committed proof-artifact hashes match their recorded values.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Other runtime/frontier tasks. |
| R2 | deferred | Other capability migration tasks. |
| R3 | partial | Descriptor mirrors and claim admission implemented; map value kinds are lost. Remaining lens work belongs to other tasks. |
| R4 | partial | Exact reconciliation and conditional predictions added; unsupported Secret bounds remain for map values. Live controls are explicitly handed to .3/.6. |
| R5 | deferred | Protocol/refinement tasks. |
| R6 | deferred | Worker capability/strategy tasks. |
| R7 | deferred | Measurement/documentation tasks. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>