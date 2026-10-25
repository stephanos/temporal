I’ll read the task specification first, then review the diff and supporting evidence for contract and consistency issues.
The retained verifier passes: 19 stages, 13 script-tree files, nine xtask source files, and 45 repository-root joins. I’m checking the ownership and migration claims against the current code; this task delivers the design, with implementation assigned to later tasks.
No surviving findings. The design matches task .9’s contract: complete inventories, recorded entry-point/workspace/policy decisions, and explicit migration ownership. The retained verifier and `git diff --check` both pass. Production validation remains assigned to implementation tasks.

## Requirements coverage

Statuses reflect this design task’s scope.

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Historical report belongs to .1/.8. |
| R2 | deferred | Inner-loop implementation belongs to .6. |
| R3 | deferred | TLC optimization belongs to .2. |
| R4 | deferred | Test migration belongs to .3/.4/.7. |
| R5 | deferred | Composition implementation belongs to .5. |
| R6 | deferred | Final measurements belong to .8. |
| R7 | met | .9’s inventory and decisions are documented; runtime implementation/proof belongs to .10/.11/.13. |
| R8 | deferred | Budget implementation belongs to .12. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>