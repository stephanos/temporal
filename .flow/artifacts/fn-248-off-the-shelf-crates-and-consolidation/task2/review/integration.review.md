I’ll read the task specification first, then trace the changed helpers through their callers and check the focused validation evidence.
The helper moves preserve the existing RNG sequence and applicability mapping. `replace_in` also tightens delimiter checks; I’m checking its callers to see whether that affects valid source mutations.
The existing mutation callers use unique end delimiters, and I found no integration defect in the helper moves. Both focused test commands were blocked by the read-only sandbox when Cargo tried to open its build lock. The committed logs record passing runs for the library, wire, generation, probe, and source-edit checks.
No surviving findings. The shared helpers preserve caller behavior, and the stricter source-region delimiter check is compatible with existing callers.

Focused test reruns were blocked by the read-only Cargo build lock. Committed passing validation logs were reviewed.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Owned by tasks .4–.6. |
| R2 | met | Single helper implementations replace former copies; controls cover reproducibility, witness replay, applicability, error ordering, source edits, and hex decoding. |
| R3 | deferred | Owned by task .3. |
| R4 | deferred | Owned by task .1. |
| R5 | deferred | Joined validation belongs to task .7. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>