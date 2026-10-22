I’ll read the task specification first, then inspect the diff and affected callers for correctness and logic issues.

The derive appears to preserve the existing constructor paths and field order. I’m checking container printing and macro expansion behavior next, including the added-field control and the golden and IR comparisons.

I haven’t found a correctness regression. Cargo’s Quick command could not run because the sandbox blocks writes to its build lock. I’ll run the cached added-field test and independently check the committed golden hashes and compressed IR dump.

No surviving correctness or logic findings. Constructor paths, field order, enum variants, and scalar/container printing preserve the previous behavior.

The cached added-field/variant control passed. Independently verified all 650 manifest hashes and the compressed IR dump checksum. Committed evidence records passing required-TLC, expansion, semantic, slice, and gaps checks. Cargo’s Quick rerun was blocked by the read-only build lock.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Existing `model/design/shared-front-end.md` compares both approaches. |
| R2 | met | Derived IR constructors replace handwritten registrations; added-field/variant control passes. |
| R3 | met | Unchanged goldens and IR dump; retained TLC and replay evidence. |
| R4 | met | `DESIGN.md`, line-delta artifacts, and timing receipts report results and overruns; existing MILESTONES describes the boundary. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge (no blocking introduced findings, all R-IDs met or deferred)