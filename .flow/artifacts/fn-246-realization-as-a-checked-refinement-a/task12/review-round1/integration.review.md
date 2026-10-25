I’ll read the task spec first, then trace the changed code’s integration points and focused verification evidence.
The change generates structural mirrors from semantic claims and filters inferred probes through claim metadata. I’m tracing how that filtering interacts with existing validators, especially for ignored subtrees and conditional queue/retry rules.
The focused Cargo command names a package that no longer exists; using its current name gets as far as the sandbox’s read-only build-cache limit. I’ll continue tracing the code and check the retained focused-test binaries. The task explicitly hands live queue/retry checks to tasks .3/.6, so those remain deferred.
The retained focused binaries pass the mirror, catalog, probe, accounting, and build-input tests. I found a gap they miss: generated claims label secret metadata maps and their values as `Bytes`, so the new Secret-bound filter retains unsupported inferred bounds. I’m also checking whether generated Ignored fields can still be admitted as mapped reads.
1. **Severity:** P2
   **Confidence:** 75
   **Classification:** introduced
   **File:Line:** `model/temporal/annotate/syntax/src/realization.rs:137`
   **R-IDs:** [R3]
   **Problem:** Expanding missing claims into Ignored fields before admission removes the previous exhaustiveness protection without checking those claims against realization reads. For example, removing `ActivityExecutionInfo.run_state` from the claims generates an Ignored field, but the existing `reads(info.run_state)` still passes: `Schema::resolve` checks opacity only when traversing further descendants. The realization can therefore consume a field that its catalog declares unclaimed and skips validating.
   **Suggestion:** Reject mapped reads/fills of unclaimed fields, including leaf fields. Add a negative control that removes a claim while retaining its realization read.

2. **Severity:** P2
   **Confidence:** 100
   **Classification:** introduced
   **File:Line:** `model/umpire/catalog/build/src/claims.rs:28`
   **R-IDs:** [R3, R4]
   **Problem:** `field.kind` comes from the lossy admission representation. It labels `Payload.metadata: Map<Text, Secret>` and `Payload.MetadataEntry.value` as `Bytes`, although their executable catalogs use `Kind::Secret`. Consequently, `Activity::admits_case` misses these secrets. The committed reconciliation retains unpredicted `RespondActivityTaskCompletedRequest.result.payloads.metadata.value:mutation.limit_plus_one(2097152)`, contrary to the required suppression of unsupported inferred Secret bounds.
   **Suggestion:** Preserve semantic kinds and map-value kinds when generating claims. Suppress unsupported secret-map/value bounds while retaining mutations justified by explicit aggregate constraints, and update the reconciliation.

Retained focused binaries passed 31 tests. Fresh Cargo verification was blocked by the read-only build cache.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Reach/frontier work belongs to other tasks. |
| R2 | deferred | Capability migration belongs to other tasks. |
| R3 | partial | Generated mirrors and claims exist; claim omissions and secret-map kinds remain unsound. |
| R4 | partial | Exact reconciliation and conditional controls exist; unsupported secret-map bounds remain. Live checks are handed to .3/.6. |
| R5 | deferred | Protocol/refinement work belongs to other tasks. |
| R6 | deferred | Worker-policy work belongs to other tasks. |
| R7 | deferred | Final measurements belong to .3/.6. |

Unaddressed R-IDs: [R3, R4]

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R3","R4"]}
```

<verdict>NEEDS_WORK</verdict> - Fix claim admission and semantic-kind generation.