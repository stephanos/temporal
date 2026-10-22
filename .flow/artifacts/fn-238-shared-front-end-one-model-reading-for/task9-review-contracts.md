I’ll read the task spec first, then review the diff and relevant callers for contract and consistency issues.
The derive covers every existing IR constructor, and the scalar/container printers retain their previous behavior. I’ll check the equivalence evidence and run the specified Quick command next.
No surviving findings. The derived constructors preserve existing paths, fields, variants, and scalar/container behavior.

The Quick rerun was blocked by the read-only filesystem at Cargo’s build lock. Committed evidence records passing Quick, expansion, semantic, required-TLC, slice, and gaps checks. Baseline golden hashes and the compressed IR dump checksum match.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | Existing `model/design/shared-front-end.md` compares both approaches and their costs. |
| R2 | met | IR printing is derived; the field/variant control verifies printing without registration. Existing front-end boundaries remain intact. |
| R3 | met | Constructor equivalence covers 26 Models; goldens remain unchanged; required-TLC evidence passes. |
| R4 | met | `DESIGN.md` records line deltas, timing, and target misses; existing `MILESTONES.md` describes the boundary. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>