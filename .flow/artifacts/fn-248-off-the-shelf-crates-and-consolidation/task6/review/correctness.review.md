I’ll read the task spec, inspect the diff, and trace the decoder behavior and tests against the acceptance criteria.
The derived envelope retains strict nested parsing, rejects explicit null for `params`, and guards against serde accepting a positional array. I’m checking the existing decoder and replay controls, plus the joined validation evidence, for gaps in those behaviors.
No blocking findings. The derived envelope preserves duplicate/unknown-field rejection, required fields, explicit-null refusal, object-only roots, nested validation, and resource budgets.

Committed joined evidence records 28 focused tests passing and a required-mode live vector replay passing. I could not rerun them: Cargo’s build-lock creation failed on the read-only filesystem.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Task .6 scope: derived envelope, strict refusal controls, existing locked serde_derive, host/guest feature inventories. Other R1 replacements belong to tasks .4/.5. |
| R2 | deferred | Owned by task .2. |
| R3 | deferred | Owned by task .3. |
| R4 | deferred | Owned by task .1. |
| R5 | deferred | Full joined gate owned by task .7. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge (no blocking introduced findings, all R-IDs met or deferred)