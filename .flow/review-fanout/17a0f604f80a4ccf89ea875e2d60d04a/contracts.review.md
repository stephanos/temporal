I’ll read the task spec first, then review the diff and related model contracts against its acceptance criteria.
The diff moves the waiver reasons and preserves their text. Task 1 records lifter refusals for both proposed abstractions, which permits keeping HeldDispatch and the two capability classes. I’m checking that the new references preserve the lifted waiver metadata and have no stale callers.
No findings. The waiver strings and all 21 emitted waiver records remain identical. Independent comparisons confirm all three activity Models differ only in Position metadata. Passing focused Scala and lint receipts match the reviewed source hashes.

The recorded lifter refusals justify retaining HeldDispatch and both capability classes under R5.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Assigned to task .2. |
| R2 | deferred | Assigned to task .3. |
| R3 | deferred | Assigned to task .2. |
| R4 | deferred | Assigned to tasks .2/.3. |
| R5 | met | Reasons moved out of `states`; text preserved; fallbacks and source-symbol moves recorded in task4 `mapping.md`; lifted IR preserved. |
| R6 | deferred | Assigned to task .5. |
| R7 | deferred | This change preserves raw IR and waiver metadata; final regeneration and canonical gates belong to task .6. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>